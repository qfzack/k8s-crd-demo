## Sentinel状态检查

### 检查redis节点的主从状态

```shell
# 查看当前的主从节点
redis-cli -h <ip> -p <port> info replciation
```

主节点中的执行结果：

```shell
:/$ redis-cli info replication
# Replication
role:master
connected_slaves:2
slave0:ip=10.1.1.50,port=6379,state=online,offset=145269,lag=1
slave1:ip=10.1.1.53,port=6379,state=online,offset=145269,lag=1
master_failover_state:no-failover
master_replid:e6989bbd44b9e4708ebe5fe4c9654ed568dac3f0
master_replid2:0000000000000000000000000000000000000000
master_repl_offset:145535
second_repl_offset:-1
repl_backlog_active:1
repl_backlog_size:1048576
repl_backlog_first_byte_offset:1
repl_backlog_histlen:145535
```

从节点中的执行结果：

```shell
:/$ redis-cli info replication
# Replication
role:slave
master_host:redis-master-0.redis-master
master_port:6379
master_link_status:up
master_last_io_seconds_ago:0
master_sync_in_progress:0
slave_read_repl_offset:220414
slave_repl_offset:220414
slave_priority:100
slave_read_only:1
replica_announced:1
connected_slaves:0
master_failover_state:no-failover
master_replid:e6989bbd44b9e4708ebe5fe4c9654ed568dac3f0
master_replid2:0000000000000000000000000000000000000000
master_repl_offset:220414
second_repl_offset:-1
repl_backlog_active:1
repl_backlog_size:1048576
repl_backlog_first_byte_offset:113768
repl_backlog_histlen:106647
```

### 检查哨兵节点的状态

```shell
# 查看sentinel集群的信息
redis-cli -h <sentinel_ip> -p <sentinel_port> INFO sentinel

# 查看命令帮助信息
redis-cli -h <sentinel_ip> -p <sentinel_port> SENTINEL help

# 查看sentinel监控的主节点信息
redis-cli -h <sentinel_ip> -p <sentinel_port> SENTINEL masters

# 查看其他监控相同节点的sentinel信息
redis-cli -h <sentinel_ip> -p <sentinel_port> SENTINEL sentinels <master-name>

# 重新调整主从节点
redis-cli -h <sentinel_ip> -p <sentinel_port> SENTINEL failover <master-name>
```

```shell
:/# redis-cli -p 26379 info SENTINEL     
# Sentinel
sentinel_masters:1
sentinel_tilt:0
sentinel_tilt_since_seconds:-1
sentinel_running_scripts:0
sentinel_scripts_queue_length:0
sentinel_simulate_failure_flags:0
master0:name=mymaster,status=ok,address=10.1.1.56:6379,slaves=4,sentinels=3
```