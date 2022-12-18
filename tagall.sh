#!/bin/zsh

git tag -a std/gateways/fasthttp/"$1" -m "$1"
git tag -a std/gateways/fastws/"$1" -m "$1"
git tag -a std/gateways/silverhttp/"$1" -m "$1"
git tag -a std/clusters/rediscluster/"$1" -m "$1"
