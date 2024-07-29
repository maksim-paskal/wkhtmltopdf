/*
Copyright paskal.maksim@gmail.com
Licensed under the Apache License, Version 2.0 (the "License")
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package internal

import (
	"bytes"
	"context"
	"flag"
	"io"
	"log"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"time"

	"github.com/pkg/errors"
)

var (
	wkHTMLToPdf       = flag.String("wkhtmltopdf", "wkhtmltopdf", "Path to wkhtmltopdf binary")
	wkHTMLToImage     = flag.String("wkhtmltoimage", "wkhtmltoimage", "Path to wkhtmltoimage binary")
	webAddress        = flag.String("web.address", ":8080", "Address to listen on")
	webRequestTimeout = flag.Duration("web.timeout", 10*time.Second, "Request timeout")    //nolint:mnd
	webReadTimeout    = flag.Duration("web.readTimeout", 5*time.Second, "Read timeout")    //nolint:mnd
	webWriteTimeout   = flag.Duration("web.writeTimeout", 10*time.Second, "Write timeout") //nolint:mnd
)

func NewApplication() *Application {
	return &Application{
		WkHTMLToPdf:    *wkHTMLToPdf,
		WkHTMLToImage:  *wkHTMLToImage,
		Address:        *webAddress,
		RequestTimeout: *webRequestTimeout,
		ReadTimeout:    *webReadTimeout,
		WriteTimeout:   *webWriteTimeout,
	}
}

type Application struct {
	Address        string
	RequestTimeout time.Duration
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
	WkHTMLToPdf    string
	WkHTMLToImage  string
}

func (a *Application) handler() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	mux.HandleFunc("/version", func(w http.ResponseWriter, r *http.Request) {
		output, err := a.exec(r.Context(), a.WkHTMLToPdf, []string{"--version"})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		} else {
			_, _ = w.Write(output)
		}
	})

	mux.HandleFunc("/jpg", func(w http.ResponseWriter, r *http.Request) {
		output, err := a.processRequest(r.Context(), &ProcessInput{
			Request:     r,
			TempPattern: "*.jpg",
			Binnary:     a.WkHTMLToImage,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		} else {
			w.Header().Set("Content-Type", "image/jpeg")
			_, _ = w.Write(output)
		}
	})

	mux.HandleFunc("/pdf", func(w http.ResponseWriter, r *http.Request) {
		output, err := a.processRequest(r.Context(), &ProcessInput{
			Request:     r,
			TempPattern: "*.pdf",
			Binnary:     a.WkHTMLToPdf,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		} else {
			w.Header().Set("Content-Type", "application/pdf")
			_, _ = w.Write(output)
		}
	})

	return mux
}

func (a *Application) argsFromValues(form url.Values) []string {
	result := make([]string, 0)

	optionsRe2 := regexp.MustCompile(`options\[(.+)\]`)

	for key, value := range form {
		if optionsRe2.MatchString(key) {
			result = append(result, "--"+optionsRe2.FindStringSubmatch(key)[1])
			if len(value[0]) > 0 {
				result = append(result, value[0])
			}
		}
	}

	return result
}

type ProcessInput struct {
	Request     *http.Request
	TempPattern string
	Binnary     string
}

func (a *Application) processRequest(ctx context.Context, processInput *ProcessInput) ([]byte, error) {
	slog.Debug("Processing request", "input", processInput)

	if err := processInput.Request.ParseForm(); err != nil {
		return nil, errors.Wrap(err, "failed to parse form")
	}

	input := processInput.Request.Form.Get("url")
	args := a.argsFromValues(processInput.Request.Form)

	if html := processInput.Request.Form.Get("html"); html != "" {
		outputHTML, err := os.CreateTemp("", "*.html")
		if err != nil {
			return nil, errors.Wrap(err, "failed to create html temp file")
		}
		defer os.Remove(outputHTML.Name())

		if _, err = outputHTML.WriteString(html); err != nil {
			return nil, errors.Wrap(err, "failed to write html temp file")
		}

		input = outputHTML.Name()
	}

	if input == "" {
		return nil, errors.New("url or html is required")
	}

	outputFile, err := os.CreateTemp("", processInput.TempPattern)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create temp file")
	}
	defer os.Remove(outputFile.Name())

	args = append(args, input)
	args = append(args, outputFile.Name())

	if _, err = a.exec(ctx, processInput.Binnary, args); err != nil {
		return nil, errors.Wrap(err, "failed to execute")
	}

	output, err := os.ReadFile(outputFile.Name())
	if err != nil {
		return nil, errors.Wrap(err, "failed to read temp file")
	}

	return output, nil
}

type logExecutor struct{}

func (l *logExecutor) Write(p []byte) (int, error) {
	slog.Debug("Command output", "output", string(p))

	return len(p), nil
}

func (a *Application) exec(ctx context.Context, command string, args []string) ([]byte, error) {
	slog := slog.With(
		"command", command,
		"args", args,
	)
	cmd := exec.CommandContext(ctx, command, args...)

	slog.Debug("Running command")

	var stdout, stderr bytes.Buffer

	cmd.Stdout = io.MultiWriter(&stdout, &logExecutor{})
	cmd.Stderr = io.MultiWriter(&stderr, &logExecutor{})

	if err := cmd.Run(); err != nil {
		slog.Error("Failed to execute", "error", err, "stderr", stderr.String())

		return nil, errors.Wrapf(err, "failed to execute %s", stderr.String())
	}

	return stdout.Bytes(), nil
}

func (a *Application) logRequest(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/healthz" {
			slog.Info("Request",
				"remoteAddr", r.RemoteAddr,
				"method", r.Method,
				"url", r.URL.String(),
			)
		}

		handler.ServeHTTP(w, r)
	})
}

func (a *Application) Run(ctx context.Context) error {
	server := &http.Server{
		Addr:         a.Address,
		Handler:      a.logRequest(http.TimeoutHandler(a.handler(), a.RequestTimeout, "timeout")),
		ReadTimeout:  a.ReadTimeout,
		WriteTimeout: a.WriteTimeout,
	}

	slog.Info("Starting server", "address", server.Addr, "timeout", a.RequestTimeout)

	go func() {
		<-ctx.Done()

		_ = server.Shutdown(context.Background()) //nolint:contextcheck
	}()

	if err := server.ListenAndServe(); err != nil && ctx.Err() == nil {
		return errors.Wrap(err, "error starting server")
	}

	return nil
}

func (a *Application) Start(ctx context.Context) {
	if err := a.Run(ctx); err != nil {
		log.Fatal(err)
	}
}
