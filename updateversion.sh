#!/bin/zsh


ss="github.com/clubpay/ronykit $1"
rs="github.com/ronykit/ronykit $2"

array=( contrib std/gateway/fasthttp std/gateways/fastws std/gateways/silverhttp std/clusters/rediscluster )
for i in "${array[@]}"
do
	filename="$i/go.mod"
	echo "update go.mod for [$filename]"
	sed -i "s/$ss/$rs/" "$filename"
done

# for contrib we need to update fasthttp too
ss="github.com/clubpay/ronykit/std/gateways/fasthttp $1"
rs="github.com/clubpay/ronykit/std/gateways/fasthttp $2"
filename="contrib/go.mod"
echo "update go.mod for [$filename]"
sed -i "s/$ss/$rs/" "$filename"

source cleanup.sh

#git add .
#git commit -m "bump version to $2"
#for i in "${array[@]}"
#do
#	git tag -a "$i/$2" -m "$2"
#done

