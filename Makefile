SHELL := /bin/bash
.ONESHELL:

ENV_FILE := code/product/.env
SERVER_DIR := code/apps/webapp
export WEBAPP_ADDR ?= :9765
export DATA_SOURCE ?=

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
