#!/bin/zsh

# Check for errors and warnings
go vet ./... || exit

# Generate version and git commit
go generate ./... || exit

# Format your code
go fmt ./... || exit

# Format the code
dirs=$(go list -f {{.Dir}} ./...)
for d in $dirs; do goimports -w $d/*.go; done

golangci-lint run
