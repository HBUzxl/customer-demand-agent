---
type: product
title: APISEC-软件部署手册
product: APISEC
aliases: []
tags:
    - 流量安全
    - 快速开始
summary: APISEC 软件安装部署操作指南，含单机与集群部署方式。
status: verified
---

# APISEC 软件部署手册


## 软件依赖及版本

| 项目 | 要求 |
|------|------|
| 操作系统 | Ubuntu 16.04、Ubuntu 18.04、Ubuntu 18.10、Ubuntu 20.04、Ubuntu 22.04、CentOS 7.2 及以上 |
| Docker | ＞ 20.10.09（不含） |

---
## 软件单机部署
### 拓扑

![单机部署拓扑](images/APISEC软件部署手册/page2_img60.png)
### 安装部署

**1. 下载软件安装包**

请使用 "APISEC 软件安装包" 进行安装。

**2. 运行安装包**

```bash
INSTALL_DIR=apisec-`date +%F`
mkdir $INSTALL_DIR
cp apisec-*.bin $INSTALL_DIR
cd $INSTALL_DIR
chmod +x *.bin
./*.bin
```

![运行状态确认](images/APISEC软件部署手册/page3_img91.png)
**3. 执行 `aminion setup -t Software`**

a. 需要配置软件流量镜像的网口（Interface）

- 若需要 API 节点接收软件流量镜像，则输入对应网口名称（如 eth1，ens193 等）
- 若不需要则保留空直接回车
- 如果填一个错误的，会导致 ripley 服务无法启动，API 节点将一直报节点异常
  - 可以通过重新执行 `aminion setup -t Software` 进行配置变更，需要手动 `systemctl restart apisec-minion` 与 `docker restart apisec-ripley`

b. 可多磁盘优化吞吐时，配置 Clickhouse Data Directories

- 若存在额外的空闲磁盘可用于 Clickhouse 的存储，并增加 Clickhouse 吞吐，需保证空闲磁盘已挂载在空目录
  - 在 Clickhouse Data Directories 填写对应的目录路径，如 `/sdc,/sdd`，用半角逗号分隔
- 若不存在时，则保留空直接回车即可

c. 可多磁盘优化吞吐时，配置 Zookeeper Data Directory

- 受磁盘 IOPS 限制，尽量给 Zookeeper 配置单独的数据存储磁盘，即不和 Clickhouse 存储磁盘重复
- 如果没有额外数据盘，可以使用系统盘，如填写 `/zookeeper` 即可

**4. 执行 `systemctl restart apisec-minion`**，无报错输出

**5. 执行 `systemctl restart apisec-load`**，无报错输出，此过程需等待 1-2 分钟

- 若有错误可重试一次
### 运行状态确认

执行 `docker ps`，结果如下，其中 mario 显示 unhealthy 为正常现象，上传 license 后将恢复正常。


![运行状态信息](images/APISEC软件部署手册/page3_img92.png)

---
## 软件简单集群
### 拓扑

![简单集群拓扑](images/APISEC软件部署手册/page4_img123.png)
### 安装部署
#### 管理节点

**1. 下载软件安装包**，请使用 "APISEC 软件安装包" 进行安装

**2. 运行安装包**

```bash
INSTALL_DIR=apisec-`date +%F`
mkdir $INSTALL_DIR
cp apisec-*.bin $INSTALL_DIR
cd $INSTALL_DIR
chmod +x *.bin
./*.bin
```

**3. 执行 `aminion setup -t SimpleMaster`**

a. 可多磁盘优化吞吐时，配置 Clickhouse Data Directories

- 若存在额外的空闲磁盘可用于 Clickhouse 的存储，并增加 Clickhouse 吞吐，需保证空闲磁盘已挂载在空目录
  - 在 Clickhouse Data Directories 填写对应的目录路径，如 `/sdc,/sdd`，用半角逗号分隔
- 若不存在时，则保留空直接回车即可

**4. 执行 `systemctl restart apisec-minion`**，无报错输出

**5. 执行 `systemctl restart apisec-load`**，无报错输出，此过程需等待 1-2 分钟

- 若有错误可重试一次
#### 采集与拦截节点——接收流量镜像

**1. 下载软件安装包**，请使用 "APISEC 软件安装包" 进行安装

**2. 运行安装包**

```bash
INSTALL_DIR=apisec-`date +%F`
mkdir $INSTALL_DIR
cp apisec-*.bin $INSTALL_DIR
cd $INSTALL_DIR
chmod +x *.bin
./*.bin
```

**3. 执行 `aminion setup -t MirrorAgent`**

a. 需要配置软件流量镜像的网口（Interface）

- 若需要 API 节点接收软件流量镜像，则输入对应网口名称（如 eth1，ens193 等）
- 若不需要则保留空直接回车
- 如果填一个错误的，会导致 ripley 服务无法启动，API 节点将一直报节点异常
  - 可以通过重新执行 `aminion setup -t Software` 进行配置变更，需要手动 `systemctl restart apisec-minion` 与 `docker restart apisec-ripley`

**4. 执行 `systemctl restart apisec-minion`**，无报错输出
#### 采集与拦截节点——接收 T1k 报文

**1. 下载软件安装包**，请使用 "APISEC 软件安装包" 进行安装

**2. 运行安装包**

```bash
INSTALL_DIR=apisec-`date +%F`
mkdir $INSTALL_DIR
cp apisec-*.bin $INSTALL_DIR
cd $INSTALL_DIR
chmod +x *.bin
./*.bin
```

**3. 执行 `aminion setup -t T1kAgent`**

**4. 执行 `systemctl restart apisec-minion`**，无报错输出

---
## 软件复杂集群
### 拓扑

![复杂集群拓扑](images/APISEC软件部署手册/page6_img184.png)
### 节点说明

#### APISEC 采集与拦截节点

此类节点主要功能是采集网络流量，经过初步处理后发送给分析服务。

- 此类节点包含多种部署方式，主要应对不同的采集方式，如流量镜像、Waf 引流、Nginx 嵌入式等
- 一个节点仅支持一种相同原理的部署方式，一套 APISEC 集群可以有多种不同采集原理的采集与拦截节点
- 在以 Nginx 嵌入式方式接入时，能够通过 APISEC 内的自定义规则实现流量拦截

#### APISEC 分析节点

此类节点进行 API 资产的识别、流量解析、数据识别，以及异步统计分析。

- 由于需要后置的，对一定时间内的流量进行异步分析，同时留下相关溯源证据，所以需要进行全流量的保存与写入，对磁盘容量与性能有较高要求
- APISEC 分析节点可以根据需要支持的分片与副本数量进行配置，每一个分片的一个副本为一个设备。分片提高吞吐，副本保证高可用
- 一个分片的完全下线会导致一定数量的数据丢失，但是产品整体仍可运行

#### APISEC 管理节点

Web 后端，分析任务调度、资产分析结果存储。
### 网络策略

| 源 IP | 源端口 | 目的 IP | 目的端口 | 协议 | 用途 |
|-------|--------|---------|---------|------|------|
| 管理员网络 | any | 管理节点 | 9443 | TCP | 管理界面 |
| 管理节点 | any | 分析节点 | 9000 | TCP | clickhouse 服务 |
| 分析节点 | any | 管理节点 | 4150 | TCP | 心跳与配置 |
| 分析节点 | any | 管理节点 | 3335 | TCP | mario 服务 |
| 分析节点 | any | 分析节点 | 9009 | TCP | clickhouse 同步、zookeeper 同步 |
| 分析节点 | any | 分析节点 | 2181 | TCP | clickhouse 同步、zookeeper 同步 |
| 分析节点 | any | 分析节点 | 2888 | TCP | clickhouse 同步、zookeeper 同步 |
| 分析节点 | any | 分析节点 | 3888 | TCP | clickhouse 同步、zookeeper 同步 |
| 采集与拦截节点 | any | 管理节点 | 4150 | TCP | 心跳与配置 |
| 采集与拦截节点 | any | 分析节点 | 3335 | TCP | mario 服务 |
| nginx 服务器（嵌入式接入） | any | 采集与拦截节点 | 8000 | TCP | 转发数据 |
| nginx 服务器（嵌入式接入） | any | 采集与拦截节点 | 8001 | TCP | 健康检查 |
| 管理节点（需要联动 Waf 处置） | any | 目标 Waf | 9443 / 443 | TCP |  |
### 安装部署
#### 管理节点

**1. 下载软件安装包**，请使用 "APISEC 软件安装包" 进行安装

**2. 运行安装包**

```bash
INSTALL_DIR=apisec-`date +%F`
mkdir $INSTALL_DIR
cp apisec-*.bin $INSTALL_DIR
cd $INSTALL_DIR
chmod +x *.bin
./*.bin
```

**3. 执行 `aminion setup -t ComplexMaster`**

	a. 配置分析节点 IP Analysis Address（Address of Analysis node）：因为管理节点需要从分析节点上的数据库获取数据，所以需要知道分析节点的 IP，无论分片还是副本，多个 IP 时以半角逗号隔开

	b. 配置 Postgres 对外暴露 IP：即分析节点可达的管理节点 IP，因为分析节点需要结合 Postgres 上的资产数据进行分析判定，所以需要知道如何连通管理节点上 Postgres
	![复杂集群网络拓扑](images/APISEC软件部署手册/final/page7_img215.png)

**4. 复制出来 DB Password 和 Minion Token**，后续节点安装将会用到

**5. 执行 `systemctl restart apisec-minion`**，分析节点部署完成后，无报错输出
#### 分析节点

**1. 多个分析节点时，提前配置设备的 hostname，保证 hostname 不同**

**2. 下载软件安装包**，请使用 "APISEC 软件安装包" 进行安装

**3. 运行安装包**

```bash
INSTALL_DIR=apisec-`date +%F`
mkdir $INSTALL_DIR
cp apisec-*.bin $INSTALL_DIR
cd $INSTALL_DIR
chmod +x *.bin
./*.bin
```

**4. 执行 `aminion setup -t AnalysisNode`**

	a. 管理节点 IP Host Address：即分析节点可达的管理节点 IP，用于心跳上报、配置获取等

	b. 填入 DB Password 与 Minion Token

	c. 配置 Clickhouse Data Directories，可达到多磁盘优化吞吐

		- 若存在额外的空闲磁盘可用于 Clickhouse 的存储，并增加 Clickhouse 吞吐，需保证空闲磁盘已挂载在空目录
  			- 在 Clickhouse Data Directories 填写对应的目录路径，如 `/sdc,/sdd`，用半角逗号分隔
		- 若不存在时，则保留空直接回车即可

	d. 配置 Clickhouse 分片 Clickhouse Shard Servers 与副本 Clickhouse Replica Servers，多个分片设备可增加吞吐性能，多个副本设备可增加高可用性，最终分析节点数量：副本数 × 分片数

		- 分片与副本的配置在每一个分析节点是相同的（扩容需要都修改配置）
		- 配置时需要每台设备的 hostname，且均不相同
		- 副本节点可以不存在，不存在时，Clickhouse Replica Servers 留空直接回车
		- 配置时，每个设备以 `hostname:IP` 的方式书写，多个设备中间以半角逗号分隔

	e. 配置管理节点 Mario 可达地址
![复杂集群拓扑](images/APISEC软件部署手册/page8_img246.png)

**5. 执行 `systemctl restart apisec-minion`**，分析节点部署完成后，无报错输出
#### 采集与拦截节点

##### 可拦截的 t1k 采集

此类配置节点用于 nginx t1k 嵌入式、SDK、Waf 引流等方式的流量采集。

**1. 下载软件安装包**，请使用 "APISEC 软件安装包" 进行安装

**2. 运行安装包**

```bash
INSTALL_DIR=apisec-`date +%F`
mkdir $INSTALL_DIR
cp apisec-*.bin $INSTALL_DIR
cd $INSTALL_DIR
chmod +x *.bin
./*.bin
```

**3. 执行 `aminion setup -t ComplexT1kAgent`**

	a. 管理节点 IP Host Address：即分析节点可达的管理节点 IP，用于心跳上报、配置获取等

	b. 填入 Minion Token

	c. 配置 Mario Aggregator 地址，即采集与拦截节点可达的分析节点 IP，多个时使用半角逗号分隔
![复杂集群拓扑](images/APISEC软件部署手册/page8_img247.png)

**4. 执行 `systemctl restart apisec-minion`**，分析节点部署完成后，无报错输出

##### 低阻塞的 t1k 采集

相较于 "可拦截的 t1k 采集" 节点，此类节点对 t1k 报文会快速响应放行结果，并异步处理 t1k 报文的检测过程，以此达到对转发服务的低延迟影响。但同时也将失去阻断能力。

**1. 下载软件安装包**，请使用 "APISEC 软件安装包" 进行安装

**2. 运行安装包**

```bash
INSTALL_DIR=apisec-`date +%F`
mkdir $INSTALL_DIR
cp apisec-*.bin $INSTALL_DIR
cd $INSTALL_DIR
chmod +x *.bin
./*.bin
```

**3. 执行 `aminion setup -t ComplexBufferAgent`**

	a. 管理节点 IP Host Address：即分析节点可达的管理节点 IP，用于心跳上报、配置获取等

	b. 填入 Minion Token

	c. 配置 Mario Aggregator 地址，即采集与拦截节点可达的分析节点 IP，多个时使用半角逗号分隔
![复杂集群拓扑](images/APISEC软件部署手册/page9_img278.png)

**4. 执行 `systemctl restart apisec-minion`**，分析节点部署完成后，无报错输出

##### 软件流量镜像采集

此类节点使用 libpcap 进行流量抓取，设备负载压力不大时，也可作为宿主机流量探针进行 POC。

**1. 下载软件安装包**，请使用 "APISEC 软件安装包" 进行安装

**2. 运行安装包**

```bash
INSTALL_DIR=apisec-`date +%F`
mkdir $INSTALL_DIR
cp apisec-*.bin $INSTALL_DIR
cd $INSTALL_DIR
chmod +x *.bin
./*.bin
```

**3. 执行 `aminion setup -t ComplexMirrorAgent`**

	a. 管理节点 IP Host Address：即分析节点可达的管理节点 IP，用于心跳上报、配置获取等

	b. 填入 Minion Token

	c. 配置 Mario Aggregator 地址，即采集与拦截节点可达的分析节点 IP，多个时使用半角逗号分隔

	d. 配置镜像口，多个镜像口时半角逗号分隔
![复杂集群拓扑](images/APISEC软件部署手册/page9_img279.png)

**4. 执行 `systemctl restart apisec-minion`**，分析节点部署完成后，无报错输出

---
## Waf 引流部署

![功能启停控制](images/APISEC软件部署手册/page10_img310.png)

Waf 引流部署是通过 Waf mario (server) 中使用 Go t1k SDK 向 APISEC 发送重新拼装的 t1k 报文，APISEC 的 apisec-detector-srv 在 Software 等模式下直接暴露 8000 端口接收此 t1k 报文。

Waf 21.07.005 的 p13（SafeLine-20-2023-57），以及 23.01.003 中 mario 已经自带 apiup 模块，即向 APISEC 转发流量的模块，默认情况下不会开启，需要在 shell 中手动修改 `mario.yml` 来启动。

由于 Waf 的 mario 配置在 21.07 与 23.01 之间进行过较大改动，下面分别进行配置修改说明。

> **注 1：** 基于上述设计，天然的 APISEC 无法跟随 Waf 检测节点的冗余情况进行冗余，因为流量是从日志分析服务 mario 来的，Waf detector 的存在情况 APISEC 没有感知。

> **注 2：** Waf 引流与 Waf+ 的引流实现是一致的，主要区别在于 "Waf+ 模式下，可以在 Waf 界面管理 API，可以在 Waf 配置 API 级别的自定义规则与检测策略"
### 雷池20 24.07.001 及后续版本

24.07.001 及后续版本可以直接在 **系统设置 / 通用设置 / 功能启停控制** 页面配置引流 IP 等。

> 注：需要 license 中包含 "API 安全" 功能后才会显示。

![引流配置](images/APISEC软件部署手册/page10_img311.png)
其中 APISEC 流量采集地址为 APISEC 的采集节点的 8000 端口，或单机模式采集流量的 IP 的 8000 端口。

![引流配置](images/APISEC软件部署手册/page12_img373.png)

配置 APISEC 的管理地址与 API Token 可以在 Waf 页面免登录跳转至 APISEC 页面中。

![引流配置](images/APISEC软件部署手册/page12_img374.png)
### 雷池20 23.01.010 及后续版本

**1. 修改 `/data/safeline/resources/mario/mario.yml`**，在 `server.handler.apiup` 配置内增加如下配置项：

```yaml
addrs:
- "10.9.32.201:8000"   # 目标 API 产品检测服务地址
r: 100                  # 采样率，整型，0-100，小于等于0不会发送，默认为0 
```

编辑后结果类似：

```yaml
cpu_profile: false
mem_profile: false
mutex_profile: false
time_zone: Asia/Shanghai
log:
  output: /logs/mario/mario.log
  level: info
  print_func: true
server:
  listen_addr: 0.0.0.0:3335
  downstream: http
  plugin_dir: ./plugins
  detect_log_db: ""
  mgt_server_url: unix:///resources/minion/socks/gateway.sock
  etcd_addrs: []
  advertise_addr: ""
  redis_urls: []
  role: server
  subscriber:
  	envelop:
      topic: /log/envelop
      buffer: 10000
      worker: 32
      entry:
        focus: true
        mode: 2
        buffer: 10000
        worker: 32
        transformer: forward
        child_points:
        - focus: true
          mode: 2
          buffer: 10000
          worker: 32
          transformer: envelop
          child_points:
          - focus: true
            mode: 3
            transformer: stat
            child_points:
            - mode: 3
              handler: plumber
          - focus: true
            mode: 3
            transformer: detect
            child_points:
            - mode: 3
              handler: persistence
            - mode: 2
              worker: 8
              handler: syslog
            - mode: 3
              handler: plumber
            - mode: 3
              handler: plugin
          - focus: true
            mode: 3
            transformer: access
            child_points:
            - mode: 3
              transformer: model
              child_points:
              - mode: 3
                handler: plugin
            - mode: 3
              handler: persistence
            - mode: 2
              worker: 8
              handler: syslog
            - mode: 3
              handler: plumber
            - mode: 3
              handler: plugin
          - focus: true
            mode: 3
            transformer: bot
            child_points:
            - mode: 3
              handler: persistence
            - mode: 2
              worker: 8
              handler: syslog
            - mode: 3
              handler: plumber
          - focus: true
            mode: 3
            transformer: dashboard
            child_points:
            - mode: 3
              handler: persistence
          - mode: 3
            handler: portrait
          - mode: 3
            handler: apiup
  handler:
    persistence:
      urls:
      - http://169.254.0.9:9200
      username: elastic
      password: mUSg1CnFcS8zpSCRVMdmmDFI7SZty9Df
      no_sniff: false
    syslog: {}
    plugin:
      detect_log_db: postgres://safeline:password@169.254.0.2/safeline?sslmode=disable
      redis_urls:
      - redis://:password@169.254.0.3:6379/5
    plumber: {}
    apiup:
      r: 100
      addrs:
      - "10.9.32.202:8000"
      - "10.9.32.203:8000"
    portrait:
      grpc_addr: 169.254.0.11:3336
      enable: true
    transformer:
      forward:
        decoder: bin_envelop
        cache:
          size: 100000
          timeout: 60
      access: {}
    detect: {}
    stat: {}
    bot: {}
    model:
      worker_num: 32
    dashboard: {}
    envelop:
      ip_geo_db: ./GeoLite2-City.mmdb
      etcd_username: ""
      etcd_password: ""
      max_procs: 1
```

**2. 重启 mario**

```bash
docker restart mario
```

> 注：为保证升级过程的兼容性，依旧支持 `uri` 参数项（但 `addrs` 参数项优先级更高，`addrs` 不为空时将使用 `addrs`），升级 Waf 可不改动该配置。
### 雷池20 23.01.003 及后续版本

**1. 修改 `/data/safeline/resources/mario/mario.yml`**，在 `server.handler.apiup` 配置内增加如下配置项：

```yaml
uri: "10.9.32.201:8000"   # 目标 API 产品 host
r: 100                     # 采样率，整型，0-100，小于等于0不会发送，默认为0
```

编辑后结果类似：

```yaml
cpu_profile: false
mem_profile: false
mutex_profile: false
time_zone: Asia/Shanghai
log:
  output: /logs/mario/mario.log
  level: info
  print_func: true
server:
  listen_addr: 0.0.0.0:3335
  downstream: http
  plugin_dir: ./plugins
  detect_log_db: ""
  mgt_server_url: unix:///resources/minion/socks/gateway.sock
  etcd_addrs: []
  advertise_addr: ""
  redis_urls: []
  role: server
  subscriber:
    envelop:
      topic: /log/envelop
      buffer: 10000
      worker: 32
      entry:
        focus: true
        mode: 2
        buffer: 10000
        worker: 32
        transformer: forward
        child_points:
        - focus: true
          mode: 2
          buffer: 10000
          worker: 32
          transformer: envelop
          child_points:
          - focus: true
            mode: 3
            transformer: stat
            child_points:
            - mode: 3
              handler: plumber
          - focus: true
            mode: 3
            transformer: detect
            child_points:
            - mode: 3
              handler: persistence
            - mode: 2
              worker: 8
              handler: syslog
            - mode: 3
              handler: plumber
            - mode: 3
              handler: plugin
          - focus: true
            mode: 3
            transformer: access
            child_points:
            - mode: 3
              transformer: model
              child_points:
              - mode: 3
                handler: plugin
            - mode: 3
              handler: persistence
            - mode: 2
              worker: 8
              handler: syslog
            - mode: 3
              handler: plumber
            - mode: 3
              handler: plugin
          - focus: true
            mode: 3
            transformer: bot
            child_points:
            - mode: 3
              handler: persistence
            - mode: 2
              worker: 8
              handler: syslog
            - mode: 3
              handler: plumber
          - focus: true
            mode: 3
            transformer: dashboard
            child_points:
            - mode: 3
              handler: persistence
          - mode: 3
            handler: portrait
          - mode: 3
            handler: apiup
  handler:
    persistence:
      urls:
      - http://169.254.0.9:9200
      username: elastic
      password: mUSg1CnFcS8zpSCRVMdmmDFI7SZty9Df
      no_sniff: false
    syslog: {}
    plugin:
      detect_log_db: postgres://safeline:password@169.254.0.2/safeline?sslmode=disable
      redis_urls:
      - redis://:password@169.254.0.3:6379/5
    plumber: {}
    apiup:
      r: 100
      uri: "10.9.32.201:8000"
    portrait:
      grpc_addr: 169.254.0.11:3336
      enable: true
  transformer:
    forward:
      decoder: bin_envelop
      cache:
        size: 100000
        timeout: 60
    access: {}
    detect: {}
    stat: {}
    bot: {}
    model:
      worker_num: 32
    dashboard: {}
    envelop:
      ip_geo_db: ./GeoLite2-City.mmdb
  etcd_username: ""
  etcd_password: ""
max_procs: 1
```

**2. 重启 mario**

```bash
docker restart mario
```
### 雷池20 21.07.005 p13

**1. 修改 `/data/safeline/resources/mario/mario.yml`**，在 `server` 配置内增加如下配置项：

```yaml
apiup_enable: true       # 开启引流
apiup_addrs:             # 目标 API 产品检测服务地址，优先级高于 apiup_url
- "10.9.32.202:8000"
- "10.9.32.203:8000"
apiup_uri: "10.9.32.201:8000"   # 目标 API 产品检测服务地址
apiup_r: 100                     # 采样率，整型，0-100，小于等于0不会发送
```

> **注：**
> 1. `apiup_addrs` 中的地址会相对平均的收到 mario 发送的流量
> 2. `apiup_addrs` 中的地址如果持续发送失败将被标记为断连，待连接恢复后再向其发送流量
>    - `apiup_addrs` 中的地址如果三次未能发送成功将被标记为断线
>    - 每 10s 一个周期处理 `apiup_addrs` 中地址是否从发送目标中移除，或添加已连接目标
> 3. 为兼容旧版本配置，`apiup_uri` 仍可使用，在 `apiup_addrs` 为空时使用 `apiup_uri`

编辑后结果类似：

```yaml
cpu_profile: false
mem_profile: false
mutex_profile: false
time_zone: Asia/Shanghai
log:
  output: /logs/mario/mario.log
  level: info
  print_func: true
server:
  listen_addr: 0.0.0.0:3335
  access_uri: /log/access
  detect_uri: /log/detect
  plugin_dir: ./plugins
  ip_geo_db: ./GeoLite2-City.mmdb
  detect_log_db: postgres://safeline:password@169.254.0.2/safeline?sslmode=disable
  mgt_server_url: unix:///resources/minion/socks/gateway.sock
  etcd_addrs: []
  advertise_addr: ""
  redis_urls: []
  assemble_detect_log: true
  envelop_uri: /log/envelop
  es:
    urls:
    - http://169.254.0.9:9200
    username: elastic
    password: password
  portrait:
    grpc_addr: 169.254.0.11:3336
    enable: true
  apiup_uri: "10.9.32.201:8900"
  apiup_r: 100
  apiup_enable: true
  apiup_addrs:
  - "10.9.32.202:8000"
  - "10.9.32.203:8000"
  max_procs: 1
```

**2. 重启 mario**

```bash
docker restart mario
```
