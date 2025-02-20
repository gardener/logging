#!/usr/bin/env bash
# SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0


dir="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
repo_root=${dir}/..
cluster_name="local"
kubeconfig_path=$repo_root/example/kind/kubeconfig
alias kind='go tool kind'
alias kubectl="$repo_root/tools/kubectl"

# Check if local kind cluster is running
if ! kind get clusters | grep -q "^$cluster_name$"; then
  printf '\u274c "%s" cluster is not found, setting up\n' $cluster_name
  kind create cluster \
      --name $cluster_name \
      --config $repo_root/example/kind/kind-config.yaml
fi

printf '\u2714 "%s" cluster is present\n' $cluster_name
printf '\u27a1 Setting up kubeconfig\n'
kind get kubeconfig --name $cluster_name | \
    tee $kubeconfig_path > /dev/null

printf '\u27a1 Applying Cluster CRD\n'
kubectl --kubeconfig $kubeconfig_path \
    apply -f $repo_root/example/kind/cluster-crd.yaml 2>&1 > /dev/null

printf '\u27a1 Create vali namespace\n'
kubectl --kubeconfig $kubeconfig_path \
    create namespace vali --dry-run=client -o yaml | \
        kubectl --kubeconfig $kubeconfig_path apply -f -  2>&1 > /dev/null

printf '\u27a1 Create fluent-bit namespace\n'
kubectl --kubeconfig $kubeconfig_path \
    create namespace fluent-bit --dry-run=client -o yaml | \
        kubectl --kubeconfig $kubeconfig_path apply -f -  2>&1 > /dev/null

printf '\u27a1 Create vali\n'
kubectl --kubeconfig $kubeconfig_path \
    apply --namespace vali -f $repo_root/example/kind/vali.yaml  2>&1 > /dev/null

printf 'Run make skaffold-run to run the fluent-bit'