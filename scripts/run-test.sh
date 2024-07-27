

wd=$(pwd)


array1=(
	contrib kit rony stub ronyup
	std/gateways/fasthttp
	std/gateways/fastws
	std/gateways/silverhttp
	std/clusters/rediscluster
	std/clusters/p2pcluster
	example/ex-01-rpc
	example/ex-02-rest
	example/ex-03-cluster
	example/ex-04-stubgen
	example/ex-05-counter
	example/ex-06-counter-stream
	example/ex-08-echo
	example/ex-09-mw
)
for i in "${array1[@]}"
do
	echo "Running tests for [$i]..."
	cd "$wd"/"$i" || exit
  gotestsum --hide-summary=output --format pkgname-and-test-fails \
  					--format-hide-empty-pkg --max-fails 1 \
  					-- -covermode=atomic -coverprofile=coverage.out ./...
  go tool cover -func=coverage.out -o=coverage.out
done


