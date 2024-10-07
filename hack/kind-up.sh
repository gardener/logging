#!/usr/bin/env bash

source $(dirname $0)/.includes.sh
name="local"

# Check if local kind cluster is running
if ! $repo_root/tools/kind get clusters | grep -q "^$name$"; then
  printf '\u274c "%s" cluster is not found, setting up\n' $name
  $repo_root/tools/kind create cluster \
      --name $name \
      --config $repo_root/example/kind/kind-config.yaml
else
  printf '\u2714 "%s" cluster is found\n' $name
fi

kubeconfig_path=$repo_root/example/kind/kubeconfig

kind get kubeconfig --name $name | \
    tee $kubeconfig_path > /dev/null

$repo_root/tools/kubectl --kubeconfig $kubeconfig_path \
    apply -f $repo_root/example/kind/cluster-crd.yaml

$repo_root/tools/kubectl --kubeconfig $kubeconfig_path \
    create namespace vali --dry-run=client -o yaml | \
    $repo_root/tools/kubectl --kubeconfig $kubeconfig_path apply -f -

$repo_root/tools/kubectl --kubeconfig $kubeconfig_path \
    create namespace fluent-bit --dry-run=client -o yaml | \
    $repo_root/tools/kubectl --kubeconfig $kubeconfig_path apply -f -

$repo_root/tools/kubectl --kubeconfig $kubeconfig_path \
    apply --namespace vali -f $repo_root/example/kind/vali.yaml