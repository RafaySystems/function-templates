
generate-fixtures:
	@echo "Generating fixtures..."
	@go-bindata -o builder/fixturesfs/fixtures.go -pkg fixturesfs templates/...
