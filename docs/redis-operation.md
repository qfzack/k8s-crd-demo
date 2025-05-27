## 1.Cluster状态检查

```shell
# 查看集群的节点状态
redis-cli -h <ip> -p <port> cluster nodes
# 查看集群状态
redis-cli -h <ip> -p <port> cluster info
# 查看slot分布状态
redis-cli -h <ip> -p <port> cluster slots
```

```shell
:/$ redis-cli cluster nodes
14e03487a5f0f7e6b3bd11d074fb15e34b737f76 10.1.1.107:6379@16379 slave 6fde96bda7819643bc1eaa901c29a570d0d55714 0 1748274527000 2 connected
b0c041bb1a8eb9eebee95630eea9b312aaabc9f5 10.1.1.106:6379@16379 slave 944d64745d8211aae82e03f4a275d978520bf52f 0 1748274526386 1 connected
944d64745d8211aae82e03f4a275d978520bf52f 10.1.1.102:6379@16379 myself,master - 0 1748274525000 1 connected 0-5460
596f9eccd6b115ef129b7f8f178677ad1be01c9a 10.1.1.105:6379@16379 slave 9269294956210624842e51882224d18b276a7552 0 1748274529441 3 connected
9269294956210624842e51882224d18b276a7552 10.1.1.104:6379@16379 master - 0 1748274529000 3 connected 10923-16383
6fde96bda7819643bc1eaa901c29a570d0d55714 10.1.1.103:6379@16379 master - 0 1748274527406 2 connected 5461-10922
```

```shell
:/$ redis-cli cluster info
cluster_state:ok
cluster_slots_assigned:16384
cluster_slots_ok:16384
cluster_slots_pfail:0
cluster_slots_fail:0
cluster_known_nodes:6
cluster_size:3
cluster_current_epoch:6
cluster_my_epoch:1
cluster_stats_messages_ping_sent:197
cluster_stats_messages_pong_sent:188
cluster_stats_messages_sent:385
cluster_stats_messages_ping_received:183
cluster_stats_messages_pong_received:197
cluster_stats_messages_meet_received:5
cluster_stats_messages_received:385
total_cluster_links_buffer_limit_exceeded:0
```

```shell
:/$ redis-cli cluster slots
1) 1) (integer) 0
   2) (integer) 5460
   3) 1) "10.1.1.102"
      2) (integer) 6379
      3) "944d64745d8211aae82e03f4a275d978520bf52f"
      4) (empty array)
   4) 1) "10.1.1.106"
      2) (integer) 6379
      3) "b0c041bb1a8eb9eebee95630eea9b312aaabc9f5"
      4) (empty array)
2) 1) (integer) 5461
   2) (integer) 10922
   3) 1) "10.1.1.103"
      2) (integer) 6379
      3) "6fde96bda7819643bc1eaa901c29a570d0d55714"
      4) (empty array)
   4) 1) "10.1.1.107"
      2) (integer) 6379
      3) "14e03487a5f0f7e6b3bd11d074fb15e34b737f76"
      4) (empty array)
3) 1) (integer) 10923
   2) (integer) 16383
   3) 1) "10.1.1.104"
      2) (integer) 6379
      3) "9269294956210624842e51882224d18b276a7552"
      4) (empty array)
   4) 1) "10.1.1.105"
      2) (integer) 6379
      3) "596f9eccd6b115ef129b7f8f178677ad1be01c9a"
      4) (empty array)
```

## 2.Sentinel状态检查

### 检查redis节点的主从状态

```shell
# 查看当前的主从节点
redis-cli -h <ip> -p <port> info replication
```

主节点中的执行结果：

```shell
:/$ redis-cli info replication
# Replication
role:master
connected_slaves:2
slave0:ip=10.1.1.99,port=6379,state=online,offset=39281,lag=0
slave1:ip=10.1.1.100,port=6379,state=online,offset=39148,lag=0
master_failover_state:no-failover
master_replid:12c3e4c5ff084f78ac820f25717e4c96c3a53885
master_replid2:0000000000000000000000000000000000000000
master_repl_offset:39414
second_repl_offset:-1
repl_backlog_active:1
repl_backlog_size:1048576
repl_backlog_first_byte_offset:1
repl_backlog_histlen:39414
```

从节点中的执行结果：

```shell
:/$ redis-cli info replication
# Replication
role:slave
master_host:redis-0.redis
master_port:6379
master_link_status:up
master_last_io_seconds_ago:1
master_sync_in_progress:0
slave_read_repl_offset:49858
slave_repl_offset:49858
slave_priority:100
slave_read_only:1
replica_announced:1
connected_slaves:0
master_failover_state:no-failover
master_replid:12c3e4c5ff084f78ac820f25717e4c96c3a53885
master_replid2:0000000000000000000000000000000000000000
master_repl_offset:49858
second_repl_offset:-1
repl_backlog_active:1
repl_backlog_size:1048576
repl_backlog_first_byte_offset:822
repl_backlog_histlen:49037
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
master0:name=mymaster,status=ok,address=10.1.1.95:6379,slaves=2,sentinels=3
```
