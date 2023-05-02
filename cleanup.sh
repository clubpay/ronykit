#!/bin/zsh

wd=$(pwd)

echo "Cleaning up [contrib]..."
cd "$wd"/contrib || exit
go mod tidy
go fmt ./...
go vet ./...
GOWORK=off golangci-lint run

echo "Cleaning up [kit]..."
cd "$wd"/kit || exit
go mod tidy
go fmt ./...
go vet ./...
GOWORK=off golangci-lint run

echo "Cleaning up [redisCluster]..."
cd "$wd"/std/clusters/rediscluster || exit
go mod tidy
go fmt ./...
go vet ./...
GOWORK=off golangci-lint run

echo "Cleaning up [fasthttp]..."
cd "$wd"/std/gateways/fasthttp || exit
go mod tidy
go fmt ./...
go vet ./...
GOWORK=off golangci-lint run

echo "Cleaning up [fastws]..."
cd "$wd"/std/gateways/fastws || exit
go mod tidy
go fmt ./...
go vet ./...
GOWORK=off golangci-lint run

echo "Cleaning up [silverhttp]..."
cd "$wd"/std/gateways/silverhttp || exit
go mod tidy
go fmt ./...
go vet ./...
GOWORK=off golangci-lint run
