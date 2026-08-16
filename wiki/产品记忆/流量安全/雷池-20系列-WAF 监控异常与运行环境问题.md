---
type: product
title: 雷池-20系列-WAF 监控异常与运行环境问题
product: 雷池-20系列
aliases: []
tags:
    - 流量安全
    - 故障排查
summary: CPU/内存占用过高排查与处置
status: verified
---

<!-- toc -->
# WAF 监控异常与运行环境问题
## CPU 负载过高
CPU 负载问题通常比较复杂，仅列举几个预期的 CPU 占用现象，若不属于以下几种场景的，请联系二线技术支持：

1. HA 模式下正常检查，属于预期的现象
	1. 通常会占用一个 CPU 核心
	2. `ps aux --sort=-%cpu` 查看显示为 `python3 -W ignore manage.py logical_replication --action=get_stats` 的进程占用
2. WAF 启动/加载程序开销，耗时较短，属于预期现象
3. ` curl http://169.254.0.5:8001/stat` 发现 req 和 rsp 的 average_length 较大，表明客户报文通常较大，大报文本身对检测和日志记录会存在 CPU 影响
	1. 可以尝试减小检测 body 限制

## 内存占用过高
WAF 利用 docker 部署服务，同时也会操作宿主机的一些文件资源或调度 WAF 所处环境的硬件资源，因而内存问题需要分开看，此处分为容器内存问题以及宿主机内存两种
### 容器内存问题
**在检查各个容器内部情况之前，需要确认环境是否有过扩容的情况，如果进行了扩容，需要重新执行 minion setup 创建一个新的部署方案，以此计算适用于当前内存大小的容器内存限制参数。**
#### mgt-api 内存过高
一般 mgt-api 内存占用高于 75% 以上认为占用较高，通常存在以下几种原因：

1. 安全资产过多，通常会伴随规则生效慢、站点生效慢等现象，主要包括几种资产：站点、自定义规则、ACL 规则、Hook 规则、IP 组、白名单
	1. 处置手段:
		1. 临时处理可以先停止新增安全资产，同时重启 mgt-api 容器
		2. 可以手动减少安全资产，例如站点使用\*等通配符
		3. 条件允许也可以适当扩容内存，前提是资产数量不能无限制扩增
2. mgt-api  内部多个进程重启竞争内存导致 OOM 
	1. 观察 mgt-api 内部进程内存状态变化，`watch -n 1 'docker exec mgt-api ps aux --sort=-rss'`
	2. 观察是否存在一个或多个进程(如 fvm-manager)的内存逐渐爬升最后断崖式下降（重启）
	3. 处置手段
		1. 重启 mgt-api
		2. 条件允许也可以适当扩容内存
		3. 调节 gunicorn 和 dramatiq 进程数

临时调节 gunicorn 和 dramtiq 进程数的方法：

进入 mgt-api 容器并编辑配置文件：

- `docker exec -it mgt-api bash`
- `vim /app/deploy/supervisor/services.conf`

具体修改如下：
将 `%(ENV_MAX_WORKER_NUM)s` 修改为预期进程数量，如 10
![](images/2.WAF监控异常与运行环境问题_images/2.WAF-监控异常与运行环境问题-2024-09-19-02-46-32.png)
![](images/2.WAF监控异常与运行环境问题_images/2.WAF-监控异常与运行环境问题-2024-09-19-02-48-59.png)
dramatiq 同理
![](images/2.WAF监控异常与运行环境问题_images/2.WAF-监控异常与运行环境问题-2024-09-19-02-48-28.png)
配置文件修改完成后，执行以下命令重启进程：

- `supervisorctl reread`
- `supervisorctl update`

最后检查改动是否生效

`ps -ef | grep -E "gunicorn|dramatiq"`

![](images/2.WAF监控异常与运行环境问题_images/2.WAF-监控异常与运行环境问题-2024-09-19-02-52-57.png)

#### mario 内存过高
1. 如果是 mario-collector 和 mario-aggregator 内存过高
若 CPU 还有富余可以尝试提高并发度 cocurrent_num 参数可以适当调大，默认是 100
如果允许一定程度的日志丢失，可以调小 max_queue_size 队列大小，默认是 1000

按需调整 collector 和 aggregator 配置文件：

- collector：`vim /data/safeline/resources/mario_collector/mario.yml`
- aggregator：`vim /data/safeline/resources/mario_aggregator/mario.yml`

![](images/2.WAF监控异常与运行环境问题_images/2.WAF-监控异常与运行环境问题-2024-09-19-03-14-00.png)

2. 若 mario 内存较高
尝试减小检测 body 的大小
![](images/2.WAF监控异常与运行环境问题_images/2.WAF-监控异常与运行环境问题-2024-09-19-03-20-09.png)

#### es 内存占用过高
1. 可以适当调大 es 容器内存大小，以及 es 的堆内存限制
Xms 和 Xmx 通常设置为容器限制参数的一半

编辑服务配置文件：`vim /data/safeline/resources/minion/service_profile.yml`

![](images/2.WAF监控异常与运行环境问题_images/2.WAF-监控异常与运行环境问题-2024-09-19-03-29-39.png)

2. 调小日志归档日期

### 宿主机内存
#### 内核空间占用过大、宿主机内存碎片化

通过以下命令计算容器内存占用，并判断内核占用大小：

- `docker stats`
- `cat /proc/meminfo`

建议优先联系二线技术支持，情况紧急时可先重启机器

## 心跳异常
### 集群模式节点过多导致心跳处理过慢
通常需要分割管理集群，如将一个集群分为两个集群等，分别管理
### 心跳时间过旧，不更新
确认节点数符合预期后，若心跳仍不更新，检查 mgt-api 状态

通过以下命令检查 mgt-api 状态：

- `docker exec mgt-api supervisorctl status`
- `top`

可以尝试重启 node 服务或 mgt-api 容器：

- 重启 node：`docker exec mgt-api supervisorctl restart node`
- 重启 mgt-api 容器：`docker restart mgt-api`

若仍未解决需要联系技术支持
