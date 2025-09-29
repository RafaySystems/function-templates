TS := $(shell /bin/date "+%Y%m%d%H%M%S")
DEV_USER ?= dev
GO_DEV_TAG := registry.dev.rafay-edge.net/${DEV_USER}/go-func:${TS}
PY_DEV_TAG := registry.dev.rafay-edge.net/${DEV_USER}/python-func:${TS}

GO := go

generate-fixtures:
	@echo "Generating fixtures..."
	@go-bindata -o builder/fixturesfs/fixtures.go -ignore __pycache__ -pkg fixturesfs templates/...

build-upload-python-sdk:
	@echo "Building and uploading Python SDK..."
	@rm -rf sdk/python/dist
	@cd sdk/python && python3 -m build
	@cd sdk/python && python3 -m twine upload dist/*

.PHONY: build-go-func
build-go-func:
	docker buildx build --platform=linux/amd64,linux/arm64 --push -f templates/go/Dockerfile -t ${GO_DEV_TAG} templates/go

.PHONY: build-python-func
build-python-func:
	docker buildx build --platform=linux/amd64,linux/arm64 --push -f templates/python/Dockerfile -t ${PY_DEV_TAG} templates/python