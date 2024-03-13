
generate-fixtures:
	@echo "Generating fixtures..."
	@go-bindata -o builder/fixturesfs/fixtures.go -ignore __pycache__ -pkg fixturesfs templates/...

build-upload-python-sdk:
	@echo "Building and uploading Python SDK..."
	@rm -rf sdk/python/dist
	@cd sdk/python && python3 -m build
	@cd sdk/python && python3 -m twine upload dist/*
