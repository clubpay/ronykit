#!/bin/zsh


ss="github.com/clubpay/ronykit/kit $1"
rs="github.com/clubpay/ronykit/kit $2"

array=( std/gateways/fasthttp std/gateways/fastws std/gateways/silverhttp std/clusters/rediscluster )
for i in "${array[@]}"
do
	filename="$i"/go.mod
	echo "update go.mod for [$filename]: $ss -> $rs"
	sed -i'' -e 's#'"$ss"'#'"$rs"'#g' "$filename"
	rm "$i"/go.mod-e
done

git add .
git commit -m "bump version to $2"
for i in "${array[@]}"
do
	git tag -a "$i/$2" -m "$2"
done

git push
git push --tags

## for contrib we need to update fasthttp too
filename="contrib/go.mod"
ss="github.com/clubpay/ronykit/kit $1"
rs="github.com/clubpay/ronykit/kit $2"
echo "update go.mod for [$filename]: $ss -> $rs"
sed -i'' -e 's#'"$ss"'#'"$rs"'#g' "$filename"
ss="github.com/clubpay/ronykit/std/gateways/fasthttp $1"
rs="github.com/clubpay/ronykit/std/gateways/fasthttp $2"
echo "update go.mod for [$filename]: $ss -> $rs"
sed -i'' -e 's#'"$ss"'#'"$rs"'#g' "$filename"
rm contrib/go.mod-e

git add .
git commit -m "[contrib] bump version to $2"
git tag -a "contrib/$2" -m "$2"
git push
git push --tags
