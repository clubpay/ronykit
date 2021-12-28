#!/bin/zsh

# Generate codes
go generate ./... || exit

# Make sure the code guide lines are met
go vet ./... || exit

# Format the code
dirs=$(go list -f {{.Dir}} ./...)
for d in $dirs; do goimports -w $d/*.go; done


# Check linter
golangci-lint run