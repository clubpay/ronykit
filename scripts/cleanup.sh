#!/bin/zsh

wd=$(pwd)


array1=(
	boxship contrib kit rony stub ronyup
	std/gateways/fasthttp
	std/gateways/fastws
	std/gateways/silverhttp
	std/clusters/rediscluster
	std/clusters/p2pcluster
#	example/ex-01-rpc
#	example/ex-02-rest
#	example/ex-03-cluster
#	example/ex-04-stubgen
#	example/ex-05-counter
#	example/ex-06-counter-stream
#	example/ex-08-echo
#	example/ex-09-mw
)

for i in "${array1[@]}"
do
	echo "Cleaning up [$i]..."
	cd "$wd"/"$i" || exit
	go mod tidy
  go fmt ./...
  go vet ./...
  GOWORK=off golangci-lint run --path-prefix "$i" --fix
done


