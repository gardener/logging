#!/usr/bin/env bash
# Copyright 2025 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
# SPDX-License-Identifier: Apache-2.0

set -eo pipefail
dir=$(dirname "$0")

RED="\033[31m"
GREEN="\033[32m"
NOCOLOR="\033[0m"

# VictoriaLogs endpoint
VLOGS_ADDR="${VLOGS_ADDR:-"http://localhost:9428/select/logsql/query"}"

function fetch_logs {
      local i=${1:-1}

      # VictoriaLogs LogsQL query with count() pipe - efficient counting without fetching all logs
      query="_time:24h k8s.namespace.name:shoot--logging--dev-${i} | extract_regexp \".+id.: .(?P<id>([a-z]+|[0-9]+|-)+)\" from _msg | count_uniq(id)"
      echo "Querying logs for cluster dev-${i}..."

      # Query VictoriaLogs using curl with count() stats
      # count() returns: {"_time":"<timestamp>","count":"<number>"}
      result=$(curl $VLOGS_ADDR --data-urlencode "query=$query" 2>/dev/null || echo "")
      if [[ -n "$result" ]]; then
          # Extract count from the stats result
          count=$(printf '%s' "$result" | jq -r '."count_uniq(id)"' | head -1)
      else
          count=0
      fi

      # Convert to number safely
      if ! [[ "$count" =~ ^[0-9]+$ ]]; then
          count=0
      fi

      if (( count < (JOBS * LOGS) )); then
          printf "got %b%d%b logs for cluster dev-%d\n" "${RED}" "$count" "${NOCOLOR}" "$i" >&2
          return
      fi
      printf "got %b%d%b logs for cluster dev-%d\n" "${GREEN}" "$count" "${NOCOLOR}" "$i"
}

function fetch_all_logs {
     for ((i=1; i<=CLUSTERS; i++)); do
        fetch_logs "$i"
     done
}

if [[ $# -eq 0 ]]; then
    fetch_all_logs
else
    fetch_logs "$1"
fi
