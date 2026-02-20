SHELL := /bin/bash
.ONESHELL:


ENV_FILE := code/product/.env
SERVER_DIR := code/apps/webapp
export WEBAPP_ADDR ?= :9765
export DATA_SOURCE ?=
ROOT_DIR := $(CURDIR)
TEST_REPORT_DIR := code/product/test-report
ALLURE_RESULTS_DIR := $(TEST_REPORT_DIR)/allure-results
ALLURE_REPORT_DIR := $(TEST_REPORT_DIR)/allure-report
TEST_RUN_ID ?= default
TEST_REPORT_DIR_EFFECTIVE := $(if $(filter default,$(TEST_RUN_ID)),$(ROOT_DIR)/$(TEST_REPORT_DIR),$(ROOT_DIR)/$(TEST_REPORT_DIR)-$(TEST_RUN_ID))
COMPOSE_PROJECT_NAME ?= $(if $(filter default,$(TEST_RUN_ID)),test-ci,test-ci-$(TEST_RUN_ID))

.PHONY: run
run:
	set -a
	source $(ENV_FILE)
	set +a
	cd code/components/webui
	npm install
	npm run build
	cd ../business
	go build -tags plugin -buildmode=plugin -o ../../apps/webapp/build/business.so ./cmd/plugin
	cd ../data
	go build -tags plugin -buildmode=plugin -o ../../apps/webapp/build/data.so ./cmd/plugin
	cd ../../apps/webapp
	# Do not use broad `pkill -f webapp`: with `.ONESHELL` it can match and kill this shell (because the whole recipe is in argv).
	pkill -f '^go run ./cmd/webapp' || true
	pkill -x webapp || true
	trap 'exit 0' SIGINT SIGTERM
	POSTGRES_DSN="$$POSTGRES_DSN" DATA_PLUGIN_PATH=./build/data.so BUSINESS_PLUGIN_PATH=./build/business.so \
		OPENAI_MODEL="$$OPENAI_MODEL" OPENAI_API_KEY="$$OPENAI_API_KEY" OPENAI_BASE_URL="$$OPENAI_BASE_URL" \
		go run ./cmd/webapp

.PHONY: run-local
run-local:
	set -a
	source $(ENV_FILE)
	set +a
	cd code/components/webui
	npm install
	npm run build
	cd ../business
	go build -tags plugin -buildmode=plugin -o ../../apps/webapp/build/business.so ./cmd/plugin
	cd ../data
	go build -tags plugin -buildmode=plugin -o ../../apps/webapp/build/data.so ./cmd/plugin
	cd ../../apps/webapp
	mkdir -p runtime/data
	# Do not use broad `pkill -f webapp`: with `.ONESHELL` it can match and kill this shell (because the whole recipe is in argv).
	pkill -f '^go run ./cmd/webapp' || true
	pkill -x webapp || true
	trap 'exit 0' SIGINT SIGTERM
	DATA_SOURCE=./config/data_local.json DATA_PLUGIN_PATH=./build/data.so BUSINESS_PLUGIN_PATH=./build/business.so \
		OPENAI_MODEL="$$OPENAI_MODEL" OPENAI_API_KEY="$$OPENAI_API_KEY" OPENAI_BASE_URL="$$OPENAI_BASE_URL" \
		go run ./cmd/webapp

.PHONY: test-business
test-business:
	cd code/components/business
	go test ./... -count=1

.PHONY: test-data
test-data:
	cd code/components/data
	go test ./... -count=1

.PHONY: test-sdk
test-sdk:
	cd code/sdk
	go test ./... -count=1

.PHONY: test-unit
test-unit: test-business test-data test-sdk

.PHONY: test-unit-shuffle
test-unit-shuffle:
	cd code/components/business
	go test ./... -count=1 -shuffle=on
	cd ../data
	go test ./... -count=1 -shuffle=on
	cd ../../sdk
	go test ./... -count=1 -shuffle=on

.PHONY: test-unit-offline
test-unit-offline:
	cd code/components/business
	GONOSUMDB='*' GOPROXY=off go test ./... -count=1
	cd ../data
	GONOSUMDB='*' GOPROXY=off go test ./... -count=1
	cd ../../sdk
	GONOSUMDB='*' GOPROXY=off go test ./... -count=1

.PHONY: test-unit-serial
test-unit-serial:
	cd code/components/business
	go test ./... -count=1 -p=1
	cd ../data
	go test ./... -count=1 -p=1
	cd ../../sdk
	go test ./... -count=1 -p=1

.PHONY: test-junit
test-junit:
	mkdir -p $(ALLURE_RESULTS_DIR)
	cd code/components/business
	go run gotest.tools/gotestsum@latest --format standard-quiet --junitfile $(ROOT_DIR)/$(ALLURE_RESULTS_DIR)/unit-business-junit.xml -- -count=1 ./...
	cd ../data
	go run gotest.tools/gotestsum@latest --format standard-quiet --junitfile $(ROOT_DIR)/$(ALLURE_RESULTS_DIR)/unit-data-junit.xml -- -count=1 ./...
	cd ../../sdk
	go run gotest.tools/gotestsum@latest --format standard-quiet --junitfile $(ROOT_DIR)/$(ALLURE_RESULTS_DIR)/unit-sdk-junit.xml -- -count=1 ./...

.PHONY: test-integration-junit
test-integration-junit:
	mkdir -p $(ALLURE_RESULTS_DIR)
	cd code/components/business
	go run gotest.tools/gotestsum@latest --format standard-quiet --junitfile $(ROOT_DIR)/$(ALLURE_RESULTS_DIR)/integration-business-junit.xml -- -count=1 -tags integration ./...
	cd ../data
	go run gotest.tools/gotestsum@latest --format standard-quiet --junitfile $(ROOT_DIR)/$(ALLURE_RESULTS_DIR)/integration-data-junit.xml -- -count=1 -tags integration ./...

.PHONY: test-e2e-junit
test-e2e-junit:
	mkdir -p $(ALLURE_RESULTS_DIR)
	cd code/tests/e2e
	go run gotest.tools/gotestsum@latest --format standard-quiet --junitfile $(ROOT_DIR)/$(ALLURE_RESULTS_DIR)/e2e-webapp-junit.xml -- -count=1 -tags e2e ./...

.PHONY: allure-generate
allure-generate:
	rm -rf $(ALLURE_RESULTS_DIR)/*
	rm -rf $(ALLURE_REPORT_DIR)
	$(MAKE) test-junit
	$(MAKE) test-integration-junit
	$(MAKE) test-e2e-junit
	if [ -d "$(TEST_REPORT_DIR)/allure-history" ]; then \
		cp -R "$(TEST_REPORT_DIR)/allure-history" "$(ALLURE_RESULTS_DIR)/history"; \
	fi
	command -v allure >/dev/null || { echo "Install Allure CLI (https://github.com/allure-framework/allure2)."; exit 1; }
	allure generate $(ALLURE_RESULTS_DIR) -o $(ALLURE_REPORT_DIR) --clean
	rm -rf "$(TEST_REPORT_DIR)/allure-history"
	if [ -d "$(ALLURE_REPORT_DIR)/history" ]; then \
		cp -R "$(ALLURE_REPORT_DIR)/history" "$(TEST_REPORT_DIR)/allure-history"; \
	fi

.PHONY: allure-open
allure-open:
	cd $(ALLURE_REPORT_DIR)
	python3 -m http.server 8080

.PHONY: test-ci
test-ci:
	./code/product/run_ci_tests.sh

.PHONY: test-ci-docker
test-ci-docker:
	if docker compose version >/dev/null 2>&1; then \
		TEST_REPORT_DIR="$(TEST_REPORT_DIR_EFFECTIVE)" COMPOSE_PROJECT_NAME="$(COMPOSE_PROJECT_NAME)" \
			docker compose -p "$(COMPOSE_PROJECT_NAME)" -f docker/docker-compose.test.yml up; \
	else \
		TEST_REPORT_DIR="$(TEST_REPORT_DIR_EFFECTIVE)" COMPOSE_PROJECT_NAME="$(COMPOSE_PROJECT_NAME)" \
			docker-compose -p "$(COMPOSE_PROJECT_NAME)" -f docker/docker-compose.test.yml up; \
	fi

.PHONY: test-ci-docker-down
test-ci-docker-down:
	if docker compose version >/dev/null 2>&1; then \
		TEST_REPORT_DIR="$(TEST_REPORT_DIR_EFFECTIVE)" COMPOSE_PROJECT_NAME="$(COMPOSE_PROJECT_NAME)" \
			docker compose -p "$(COMPOSE_PROJECT_NAME)" -f docker/docker-compose.test.yml down -v; \
	else \
		TEST_REPORT_DIR="$(TEST_REPORT_DIR_EFFECTIVE)" COMPOSE_PROJECT_NAME="$(COMPOSE_PROJECT_NAME)" \
			docker-compose -p "$(COMPOSE_PROJECT_NAME)" -f docker/docker-compose.test.yml down -v; \
	fi
