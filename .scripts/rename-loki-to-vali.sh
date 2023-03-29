#!/bin/bash

dir=$(dirname $0)

# checking dependent binaries
execs=(git sed xargs rename)
for e in ${execs[@]}; do
    if ! command -v $e &> /dev/null ; then
      echo "$e is required. Usually brew will do the job."
      exit
    fi
done

# generate an unique id
id=$(uuidgen | tr "[:upper:]" "[:lower:]" | cut -f 1 -d '-')

# checkout a new branch containing the renaming commits
git checkout -b "test-renaming-branch-$id"

# search, replace and commit
searches=("loki" "github.com/grafana" "Grafana/Loki" "Grafana Loki" "GrafanaLoki" "grafana/loki project" "blob/v1.6.0" "Loki" "LOKI" "promtail")
replaces=("vali" "github.com/credativ" "Credativ/Vali" "Credativ Vali" "CredativVali" "credativ/vali project" "blob/v2.2.4" "Vali" "VALI" "valitail")

length=${#searches[@]}
for (( i=0; i<${length}; i++ )); do
  echo "replace: ${searches[$i]}"
  git grep -z -l "${searches[$i]}" :^vendor :^.scripts :^go.mod :^go.sum | xargs -r -0 sed -i "s|${searches[$i]}|${replaces[$i]}|g"
  git add . && git commit -m "renaming from ${searches[$i]} to ${replaces[$i]}"
done


# switch in go.mod from grafana/vali to creadtive/vali
sed -i "s|github.com/grafana/loki v1.6.2-0.20210406003638-babea82ef558|github.com/credativ/vali v0.0.0-20230322125549-22fdbf30c62a|g" go.mod
git add go.mod && git commit -m "switch in go.mod from grafana/loki to creadtive/vali"

# rename folders
find $dir/.. -name "*loki*" -not -path "$dir/../vendor/*" -not -path "$dir/../.git/*" -not -path "$dir/../.scripts/*"  -exec rename 's/loki/vali/' {} \;
git add . && git commit -m "renaming directories containing loki"

# rename files
find $dir/.. -name "*loki*" -not -path "$dir/../vendor/*" -not -path "$dir/../.git/*" -not -path "$dir/../.scripts/*"  -execdir rename 's/loki/vali/' {} \;
git add . && git commit -m "renaming files containing loki"

# update and vendor module dependencies
go mod tidy && go mod vendor
git add . && git commit -m "update and vendor module dependencies"

# format module files
go fmt ./...
git add . && git commit -m "format module files"
