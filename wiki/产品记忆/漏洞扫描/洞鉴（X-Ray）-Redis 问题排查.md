---
type: product
title: 洞鉴（X-Ray）-Redis 问题排查
product: 洞鉴（X-Ray）
aliases: []
tags:
    - 漏洞扫描
    - support
    - 故障排查
summary: Redis 缓存和消息队列问题的诊断和修复。
status: verified
---

<!-- toc -->

# Redis相关问题


## Redis因AOF文件损坏导致无法启动或一直Restarting



```shell

docker run --rm -v /data/x-ray/container/redis:/data -v/data/x-ray/container/redis.conf:/etc/redis.conf  portus.in.chaitin.net/x-ray-public/redis:latest redis-check-aof --fix /data/appendonly.aof

```

---

## Redis因AOF文件过大导致启动缓慢或启动超时

```shell

docker exec -it xray-redis redis-cli -a {redis-password}
 
bgrewriteaof
save
```
<!-- endtoc -->
