#!/usr/bin/env bash
# Copyright 2025 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
# SPDX-License-Identifier: Apache-2.0


set -e

dir=$(dirname "$0")

namespace="fluent-bit"
nameOverride="logging"

helm uninstall logging \
    --namespace $namespace \
    --wait \
    --timeout 300s \
    --ignore-not-found || true

kubectl delete pvc \
    --selector "app.kubernetes.io/name=$nameOverride-vali" \
    --namespace $namespace \
    --ignore-not-found=true || true

kubectl delete pvc \
    --selector "app.kubernetes.io/name=$nameOverride-prometheus" \
    --namespace $namespace \
    --ignore-not-found=true || true