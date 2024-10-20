KUBECONFIG=$(HOME)/.kube/dev
image=paskalmaksim/wkhtmltopdf:dev

build:
	git tag -d `git tag -l "wkhtmltopdf-*"`
	go run github.com/goreleaser/goreleaser@latest build --clean --skip=validate --snapshot
	mv ./dist/wkhtmltopdf_linux_amd64_v1/wkhtmltopdf wkhtmltopdf
	docker build --pull --push . -t $(image)

run:
	go run ./cmd -graceful-shutdown=0s

test-chart:
	ct lint --all
	helm template ./charts/wkhtmltopdf | kubectl apply --dry-run=server -f -

test:
	./scripts/validate-license.sh
	go mod tidy
	go run github.com/golangci/golangci-lint/cmd/golangci-lint@latest run -v

deploy:
	helm upgrade --install wkhtmltopdf \
	--namespace wkhtmltopdf \
	--create-namespace \
	--set image=$(image) \
	--set imagePullPolicy=Always \
	./charts/wkhtmltopdf
	kubectl -n wkhtmltopdf delete pods --all