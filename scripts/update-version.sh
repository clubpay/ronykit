#!/bin/zsh

### Increments the part of the string
## $1: version itself
## $2: number of part: 0 – major, 1 – minor, 2 – patch
increment_version() {
  local version=$1
  local array=( ${version//./ } )
  array[$2]=$((array[$2]+1))
  if [ "$2" -lt 2 ]; then array[2]=0; fi
  if [ "$2" -lt 1 ]; then array[1]=0; fi

	echo "${array[0]}.${array[1]}.${array[2]}"
}

### Updates go.mod files and push the new tags to the git remote
## $1: old version
## $2: new version
update_version() {
	ss="github.com/clubpay/ronykit/kit $1"
  rs="github.com/clubpay/ronykit/kit $2"

  wd=$(pwd)
  git tag -a kit/"$2" -m "$2"
  git push --tags

  array1=(
  	std/gateways/fasthttp std/gateways/fastws std/gateways/silverhttp
  	std/clusters/rediscluster std/clusters/p2pcluster util
  )
  for i in "${array1[@]}"
  do
  	filename="$i"/go.mod
  	echo "update go.mod for [$filename]: $ss -> $rs"
  	sed -i'' -e 's#'"$ss"'#'"$rs"'#g' "$filename"
  	rm "$i"/go.mod-e
  	cd "$i" || exit
  	go mod tidy
  	cd "$wd" || exit
  done


  # fasthttp
  fss="github.com/clubpay/ronykit/std/gateways/fasthttp $1"
  frs="github.com/clubpay/ronykit/std/gateways/fasthttp $2"

  # contrib
  css="github.com/clubpay/ronykit/contrib $1"
  crs="github.com/clubpay/ronykit/contrib $2"

  # stub
  sss="github.com/clubpay/ronykit/stub $1"
  srs="github.com/clubpay/ronykit/stub $2"

  array2=(contrib rony stub flow)
  for i in "${array2[@]}"
  do
  	filename="$i"/go.mod
  	echo "update go.mod for [$filename]: $ss -> $rs"
  	sed -i'' -e 's#'"$ss"'#'"$rs"'#g' "$filename"
  	echo "update go.mod for [$filename]: $fss -> $frs"
  	sed -i'' -e 's#'"$fss"'#'"$frs"'#g' "$filename"
  	echo "update go.mod for [$filename]: $css -> $crs"
    sed -i'' -e 's#'"$css"'#'"$crs"'#g' "$filename"
    echo "update go.mod for [$filename]: $sss -> $srs"
    sed -i'' -e 's#'"$sss"'#'"$srs"'#g' "$filename"
  	rm "$i"/go.mod-e
  	cd "$i" || exit
  	go mod tidy
  	cd "$wd" || exit
  done

  git add .
  git commit -m "bump version to $2"
  for i in "${array1[@]}"
  do
  	git tag -a "$i/$2" -m "$2"
  done

  for i in "${array2[@]}"
  do
  	git tag -a "$i/$2" -m "$2"
  done

  git push
  git push --tags


  sh cleanup.sh
  git add .
  git commit -m "cleanup go.sum"
  git push
}


prefix=$1
versionPart=$2	 # Possible values  0 – major, 1 – minor, 2 – patch
prevVer=$(git describe --tags --abbrev=0 --match "${prefix}/*")
prevVer=${prevVer#"${prefix}/v"}
currVer=$(increment_version "$prevVer" "$versionPart")

echo "Updating from v${prevVer} --> v${currVer}, do you want to continue? (y/n)"
read -r response

# Check the response
if [[ "$response" == "y" || "$response" == "Y" ]]; then
		update_version "v$prevVer" "v$currVer"
else
    echo "Exiting..."
    exit 1
fi
