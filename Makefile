.DEFAULT_GOAL:=help
GOLANGCI_LINT_VER = "1.30.0"
# Image URL to use all building/pushing image targets
IMG ?= danielxlee/jdseckill:latest

##@ Code management

vet: ## Run go vet for this project. More info: https://golang.org/cmd/vet/
	@echo go vet
	go vet $$(go list ./... )

fmt: ## Run go fmt for this project
	@echo go fmt
	go fmt $$(go list ./... )

tidy: ## Run go mod tidy to update dependencies
	@echo go mod tidy
	go mod tidy -v

lint: ## Install and run golangci-lint checks
	@echo run golangci-lint checks
ifneq (${GOLANGCI_LINT_VER}, "$(shell ./bin/golangci-lint --version 2>/dev/null | cut -b 27-32)")
	@echo "golangci-lint missing or not version '${GOLANGCI_LINT_VER}', downloading..."
	curl -sSfL "https://raw.githubusercontent.com/golangci/golangci-lint/v${GOLANGCI_LINT_VER}/install.sh" | sh -s -- -b ./bin "v${GOLANGCI_LINT_VER}"
endif
	./bin/golangci-lint --timeout 5m run

check: ## Run all dev code manager
	- make fmt
	- make vet
	- make tidy
	- make lint

##@ Build and Run

build: ## Build seckill binary
	go build -o bin/seckill main.go

run: check ## Run main programe
	go run ./main.go

docker-build: ## Build the docker image
	docker build . -t ${IMG}

docker-push: ## Push the docker image
	docker push ${IMG}

docker-run: ## Run docker container
	@docker run -d --rm \
	--name jd-seckill ${IMG}

show-qrcode: ## Show qrcode image from docker container
	@docker cp jd-seckill:/qr_code.png .
	@open qr_code.png

show-logs: ## show container logs
	@docker logs jd-seckill -f

start: ## start seckill
	- make docker-run
	- sleep 5
	- make show-qrcode
	- make show-logs

stop: ## stop docker container
	@docker stop jd-seckill

##@ Help
help: ## Display this help
	@echo "Usage:\n  make \033[36m<target>\033[0m"
	@awk 'BEGIN {FS = ":.*##"}; \
		/^[a-zA-Z0-9_-]+:.*?##/ { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 } \
		/^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
