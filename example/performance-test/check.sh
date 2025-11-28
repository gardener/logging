#!/usr/bin/env bash
# Copyright 2025 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
# SPDX-License-Identifier: Apache-2.0


set -eo pipefail
dir=$(dirname "$0")
QUERY_WAIT=${QUERY_WAIT:-1}       # seconds to wait before first query
QUERY_RETRIES=${QUERY_RETRIES:-10}  # number of query attempts
QUERY_INTERVAL=${QUERY_INTERVAL:-5} # seconds between query attempts

RED="\033[31m"
GREEN="\033[32m"
NOCOLOR="\033[0m"

# check to see if vali-shoot is port-forwarded
if ! curl --silent --max-time 2 http://localhost:3100/metrics >/dev/null; then
    echo "Please port-forward vali-shoot to localhost:3100"
    echo "kubectl port-forward -n fluent-bit pod/logging-vali-shoot-0 3100:3100"
    exit 1
fi


run_log_query() {
  echo "Waiting ${QUERY_WAIT}s before querying Vali..."
  sleep "${QUERY_WAIT}"

  local q='sum(count_over_time({container_name="logger"}[24h]))'
  local attempt=0
  while (( attempt < QUERY_RETRIES )); do
    if out=$(${dir}/bin/logcli query "$q" --quiet --output=jsonl 2>/dev/null); then
      # Extract last pair [timestamp, value]
      if pair=$(printf '%s' "$out" | jq -r '.[0].values | last?'); then
        if [[ -n "$pair" && "$pair" != "null" ]]; then
          ts=$(printf '%s' "$pair" | jq '.[0]')
          val=$(printf '%s' "$pair" | jq -r '.[1]')
          int=${ts%.*}
          frac=${ts#*.}; ms=$(printf '%03d' "$frac")
          # macOS date
          human=$(date -d "@$int" '+%Y-%m-%d %H:%M:%S')
          echo "Query: $q"
          echo "Last sample: $human.${ms}Z (raw=${ts}) value=${val}"
          return 0
        fi
      fi
    fi
    attempt=$((attempt+1))
    echo "No data yet (attempt ${attempt}/${QUERY_RETRIES}), retrying in ${QUERY_INTERVAL}s..."
    sleep "${QUERY_INTERVAL}"
  done
  echo "Failed to obtain data for query after ${QUERY_RETRIES} attempts" >&2
  return 1
}

run_log_query
