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
package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/maksim-paskal/wkhtmltopdf/internal"
)

var (
	debug            = flag.Bool("debug", false, "Debug mode")
	gracefulShutdown = flag.Duration("graceful-shutdown", 10*time.Second, "Enable graceful shutdown") //nolint:mnd
)

func main() {
	flag.Parse()

	logLevel := slog.LevelInfo

	if *debug {
		logLevel = slog.LevelDebug
	}

	if _, ok := os.LookupEnv("KUBERNETES_SERVICE_HOST"); ok {
		slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: logLevel,
		})))
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	signalChanInterrupt := make(chan os.Signal, 1)
	signal.Notify(signalChanInterrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-signalChanInterrupt
		cancel()
		<-signalChanInterrupt
		os.Exit(1)
	}()

	application := internal.NewApplication()

	application.Start(ctx)

	<-ctx.Done()

	slog.Warn("Graceful shutdown", "timeout", *gracefulShutdown)
	time.Sleep(*gracefulShutdown)
}
