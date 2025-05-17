#!/bin/bash

set -euo pipefail

# verify enviroment variables
required_vars=("REDIS_PORT" "CLUSTER_REPLICAS" "TOTAL_REPLICAS")
for var in "${required_vars[@]}"; do
    if [ -z "${!var:-}" ]; then
        echo "Error: Required environment variable $var is not set"
        exit 1
    fi
done

# compute redis cluster nodes
MASTER_COUNT=$((TOTAL_REPLICAS / (CLUSTER_REPLICAS + 1)))
echo "Cluster configuration:"
echo "- Total nodes: $TOTAL_REPLICAS"
echo "- Master nodes: $MASTER_COUNT"
echo "- Replicas per master: $CLUSTER_REPLICAS"

echo "Waiting for all Redis nodes to be ready..."
for i in $(seq 0 $((TOTAL_REPLICAS-1))); do
    until redis-cli -h "${HOSTNAME%%-*}-$i.${HOSTNAME%%-*}" ping > /dev/null 2>&1; do
        echo "Waiting for ${HOSTNAME%%-*}-$i.${HOSTNAME%%-*}..."
        sleep 2
    done
    echo "Node ${HOSTNAME%%-*}-$i is ready"
done

check_cluster_status() {
    local node="$1"
    local result
    result=$(redis-cli -h "$node" cluster info 2>/dev/null) || return 1
    
    echo "Cluster status for $node:"
    echo "$result"
    
    if echo "$result" | grep -q "cluster_state:ok"; then
        return 0
    else
        return 1
    fi
}
if check_cluster_status "${HOSTNAME%%-*}-0.${HOSTNAME%%-*}"; then
    echo "Redis cluster is already initialized and healthy"
    exit 0
fi

# initialize redis cluster
echo "Creating Redis cluster..."
# get redis nodes list
nodes=""
for i in $(seq 0 $((TOTAL_REPLICAS-1))); do
    nodes="$nodes ${HOSTNAME%%-*}-$i.${HOSTNAME%%-*}:$REDIS_PORT"
done

# create redis cluster with redis-cli
if redis-cli --cluster create $nodes \
    --cluster-replicas "$CLUSTER_REPLICAS" \
    --cluster-yes; then
    echo "Cluster created successfully"
    
    # verify the redis cluster status
    echo "Verifying cluster status..."
    if redis-cli --cluster check "${HOSTNAME}.${HOSTNAME%%-*}:${REDIS_PORT}"; then
        echo "Cluster verification completed successfully"
    else
        echo "Cluster verification failed"
        exit 1
    fi
else
    echo "Failed to create cluster"
    exit 1
fi
