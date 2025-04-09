
cleanup:
	@go fmt ./...
	@go vet ./...
	@golangci-lint run --fix ./...

tag: cleanup
	@@git tag -a ${tag} -m "${tag}"
	@go generate -v ./...
	@go fmt ./...

install:
	@cd ./cmd/boxship && go install

release: tag
	tag
	@git add .
	@git commit -m "bump version to ${tag}"
	@git push
	@git push --tags
	@gh release create ${tag} --generate-notes