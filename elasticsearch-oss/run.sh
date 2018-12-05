#!/bin/sh

# Copyright (c) 2018 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -ex

export PATH_DATA=${PATH_DATA:-/data}
# Set environment variables defaults
export ES_JAVA_OPTS=${ES_JAVA_OPTS:-"-Xms512m -Xmx512m"}
export CLUSTER_NAME=${CLUSTER_NAME:-kubernetes-logging}
export NODE_NAME=${NODE_NAME:-${HOSTNAME}}
export NODE_MASTER=${NODE_MASTER:-true}
export NODE_DATA=${NODE_DATA:-true}
export NODE_INGEST=${NODE_INGEST:-true}
export HTTP_ENABLE=${HTTP_ENABLE:-true}
export HTTP_PORT=${HTTP_PORT:-9200}
export TRANSPORT_PORT=${TRANSPORT_PORT:-9300}
export HTTP_CORS_ENABLE=${HTTP_CORS_ENABLE:-false}
export HTTP_CORS_ALLOW_ORIGIN=${HTTP_CORS_ENABLE:-"*"}
export NETWORK_HOST=${NETWORK_HOST:-"0.0.0.0"}
export NUMBER_OF_MASTERS=${NUMBER_OF_MASTERS:-1}
export MAX_LOCAL_STORAGE_NODES=${MAX_LOCAL_STORAGE_NODES:-1}
export DISCOVERY_SERVICE=${DISCOVERY_SERVICE:-"elasticsearch-logging"}
#single node optimization

#10% of the total heap allocated to a node will be used as 
#the indexing buffer size shared across all shards.
export INDEX_BUFFER_SIZE=${INDEX_BUFFER_SIZE:-"10%"}
#adjust the index query size
export INDEX_QUEUE_SIZE=${INDEX_QUEUE_SIZE:-200}
#adjust the bulk query size
export INDEX_QUEUE_SIZE=${BULK_QUEUE_SIZE:-200}
#enaable the disk allocation decider.
export ALLOW_DISK_ALLOCATION=${ALLOW_DISK_ALLOCATION:-true}
# Elasticsearch will attempt to relocate shards away from a node whose disk usage is above X%
export DISK_WATERMARK_HIGHT=${DISK_WATERMARK_HIGHT:-"90%"}
# disk usage point beyond which ES won’t allocate new shards to that node
export DISK_WATERMARK_HIGHT=${DISK_WATERMARK_LOW:-"85%"}

 #threshold for read only lock
export DISK_WATERMARK_FLOOD_STAGE=${DISK_WATERMARK_FLOOD_STAGE:-"95%"}

export SHARD_REBALANCING_FOR=${SHARD_REBALANCING_FOR:-"all"}

chown -R elasticsearch:elasticsearch ${PATH_DATA}

exec su elasticsearch -c /usr/local/bin/docker-entrypoint.sh
