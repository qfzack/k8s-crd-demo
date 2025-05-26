#!/bin/bash

set -euo pipefail

echo "Waiting for all Redis nodes to be ready..."
WAIT_TIMEOUT=60
START_TIME=$(date +%s)
for i in $(seq 0 $((REPLICAS-1))); do
    node="${HOSTNAME%%-*}-$i.${HOSTNAME%%-*}"
    while true; do
        CURRENT_TIME=$(date +%s)
        ELAPSED_TIME=$((CURRENT_TIME - START_TIME))
        
        if [ $ELAPSED_TIME -gt $WAIT_TIMEOUT ]; then
            echo "Error: Timeout waiting for Redis nodes after ${WAIT_TIMEOUT} seconds"
            echo "Failed node: $node"
            exit 1
        fi

        if redis-cli -h "$node" ping > /dev/null 2>&1; then
            echo "Node $node is ready"
            break
        else
            echo "Waiting for $node... (${ELAPSED_TIME}s elapsed)"
            sleep 2
        fi
    done
done

# initialize redis master-slave
echo "Creating Redis master-slave..."
# TODO set first pod as master
MASTER_POD="redis-0.redis"
echo "Using $MASTER_POD as master"

# set replicas
for i in $(seq 1 $((REPLICAS-1))); do
    REPLICA_POD="redis-$i.redis"
    echo "Configuring $REPLICA_POD as replica of $MASTER_POD"
    redis-cli -h "$REPLICA_POD" replicaof "$MASTER_POD" "$REDIS_PORT"
done

echo "Redis master-replica configuration completed successfully"
