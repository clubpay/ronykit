wd=$(pwd)

cd "$wd"/contrib || exit
go mod tidy

cd "$wd"/std/clusters/rediscluster || exit
go mod tidy

cd "$wd"/std/gateways/fasthttp || exit
go mod tidy

cd "$wd"/std/gateways/fastws || exit
go mod tidy

cd "$wd"/std/gateways/silverhttp || exit
go mod tidy
