GOLANGCI_LINT_CACHE?=/tmp/praktikum-golangci-lint-cache
METRICSTEST_BIN=./metricstest
SERVER_BIN=./build/server
AGENT_BIN=./build/agent
COMPOSE_FILE=docker-compose.yml
ITERATION=10

.PHONY: build
build:
	docker-compose -f $(COMPOSE_FILE) build

.PHONY: up
up:
	docker-compose -f $(COMPOSE_FILE) up

.PHONY: down
down:
	docker-compose -f $(COMPOSE_FILE) down

.PHONY: test
test:
	go build -o $(SERVER_BIN) ./cmd/server
	go build -o $(AGENT_BIN) ./cmd/agent

	chmod 644 ./store

	for i in $(shell seq 1 $(ITERATION)); do \
		$(METRICSTEST_BIN) -test.v -test.run=^TestIteration$$i$$ \
			-binary-path=$(SERVER_BIN) \
			-agent-binary-path=$(AGENT_BIN) \
			-source-path=. \
			-server-port=8080 \
			-file-storage-path=./store \
			-database-dsn=postgres://postgres:postgres@postgres:5432/praktikum?sslmode=disable; \
	done

.PHONY: lint
lint: _golangci-lint-rm-unformatted-report

.PHONY: _golangci-lint-reports-mkdir
_golangci-lint-reports-mkdir:
	mkdir -p ./golangci-lint

.PHONY: _golangci-lint-run
_golangci-lint-run: _golangci-lint-reports-mkdir
	-docker run --rm \
	-v $(shell pwd):/app \
	-v $(GOLANGCI_LINT_CACHE):/root/.cache \
	-w /app \
	golangci/golangci-lint:v1.62.2 \
		golangci-lint run \
			-c .golangci.yml \
	> ./golangci-lint/report-unformatted.json

.PHONY: _golangci-lint-format-report
_golangci-lint-format-report: _golangci-lint-run
	cat ./golangci-lint/report-unformatted.json | jq > ./golangci-lint/report.json

.PHONY: _golangci-lint-rm-unformatted-report
_golangci-lint-rm-unformatted-report: _golangci-lint-format-report
	rm ./golangci-lint/report-unformatted.json

.PHONY: golangci-lint-clean
golangci-lint-clean:
	sudo rm -rf ./golangci-lint