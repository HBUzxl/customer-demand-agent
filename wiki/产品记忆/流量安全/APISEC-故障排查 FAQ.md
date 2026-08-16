---
type: product
title: APISEC-故障排查 FAQ
product: APISEC
aliases: []
tags:
    - 流量安全
    - 故障排查
summary: APISEC 常见故障排查与问题解答，涵盖硬件网口、API 发现、流量配置等。
status: verified
---

# APISEC 故障排查 FAQ

## 硬件与网口

### 硬件上架以后有部分网口在 network show 中看不到

硬件部署中，APISEC 的流量镜像服务（apisec-ripley）会默认将一些网口配置为流量镜像网口，并使用 dpdk 将接管网口，bypass 内核驱动，所以在 `network show`、`ip addr`、`ip link` 等命令中均不会看到这些网口。

默认网口为 **eth1.1,eth1.2,eth2.1,eth2.2,eth3.1,eth4.1** 其中如果有任何网口不存在（比如对应插槽并没有插入网卡），那么将跳过该网口。

### 硬件上架后，镜像口灯不亮

默认网口为 **eth1.1,eth1.2,eth2.1,eth2.2,eth3.1,eth4.1**，确认下网口接入是否正确。

**目前，硬件灌装后的新增网卡不会被自动识别**，比如灌装后设备的扩展槽四未插卡，此时插入网卡到扩展槽四，eth4.1 不会被识别为镜像口。

**进入底层 shell**，查看 `docker logs --tail 100 f apisec-ripley` 是否有报错信息，正常情况应该类似如图所示。

![架构对比图](images/APISEC故障排查FAQ/embedded_p2_1.png)

**底层 shell 中**，`cat /data/apisec/resources/ripley/ripley.yml`，看你所需要的镜像口是否在文件中。

如果上述步骤都没有异常，基本可以确定软件没有问题，需要看一下硬件：

- 光模块波长是否对应
- nobypass 的万兆光卡应该是能够自适应千兆口的
- 插入非镜像口看看是否亮灯
- 更换网线
- 确认客户那边网口是否 up

## API 发现问题

### API 没有被发现怎么办

这里有三种分类的原因：引流是否完整、是否被判断为 API、查看时是否筛选正确。

#### 引流是否完整

对于 Waf 引流（或 Waf+模式），流量由 Waf 的日志分析服务通过 Golang T1k SDK 将访问日志转化为 T1k 流量传入 APISEC，所以需要保证：

- 测试流量可以在 waf 记录访问日志
  - 透明桥模式与流量镜像模式下，不能使用 https 进行测试
  - 流量对应站点开启了 "记录不存储访问日志" 或 "记录并存储访问日志"
  - 较新版本的 Waf 在 "记录访问日志" 的配置中可以配置 "记录响应 Body"，Waf 默认不开启，引流时该选项需要开启
- 请求与响应均生成访问日志，并能够被合并记录
  - 客户 Waf 当前负载支持的情况下，可以短暂开启 Waf 的 "记录并存储访问日志"，来检查一下是否记录了完整的访问日志
- 流量镜像模式下，可能因为镜像设备的 "feature"，请求与响应对应的数据包上 Vlan 不同（可以抓包确认），此时需要在工作组中开启 "忽略 Vlan"
- 嵌入式模式下需要开启完整的响应检测，piggyback 不足以提供足够的响应数据

#### 请求是否会被判断为 API

有如下一些判断依据可以参考：

- 当有 Host 头时，值如果不包含 `.` 时（如 `Host: uqweghcj`），我们会将其抛弃。可以在站点发现中查看是否有对应 Host 的站点或者对应目标 IP 的站点。
  - 一方面，我们发现一些扫描器会使用这种 Host
  - 另一方面，这种域名一般作为替代 IP 的主机名，而我们常见的域名不会这样
- 一般该域名站点当前绑定了某个应用，无应用的站点不会被进行任何处理
- 初次访问的 URL 需要响应码为 200（以此证明此路径为一个正常的服务）

#### 查看时是否筛选正确

为兼顾用户一般查看 API 资产列表时的清爽，以及特殊需求下 API 资产列表的完整性，我们在 API 资产列表默认增加了几个筛选条件，其影响分别如下：

- **"接口参数：不为空"**：指经过解析后，除去常见的 header（如 content-type），请求响应中仍有用户自定义数据传输。
  - 注：当简单的使用 `curl "http://10.2.2.2/api"` 这样的方式测试时，如果响应也只是一个 200 空 body，那么极有可能没有用户参数可以被识别
  - 注：上述测试中，如果更换请求方法，响应仍为空 body，则结果相同，依旧会被认为没有用户参数可以被识别
- **"业务接口：等于 是"**：一般情况下，传输 HTML、jpg、js 等内容的路径被认为是 "静态文件" 的路径，而传输 json 业务数据的路径被认为是 API 路径，并更加关注。当一个接口（根据 URL 与 content-type 判断）传输过非常见静态资源时，会被标记为 "业务接口"
  - 注："业务接口" 与 "静态资源" 并不互斥，一种业务情况是：数据获取成功时返回 json 数据，数据查询失败时返回 HTML 页面

## 流量排查

### 没有流量怎么排查

#### 步骤零：对端抓包

最简单的办法是去机房把镜像口设备那端网线拔下来插某个电脑上，然后对这个网口抓包。

虽然也有可能是网口网线接触不良，不过可以先抓包看看，然后换网线看看。

#### 步骤一：web 端确认服务运行状态

- 在 **系统设置 / 设备运行状态 / 负载状态** 中，确认 **流量镜像服务** 与 **流量检测服务** 服务状态正常。
  - 保证相关服务是正在运行的，没有因某些问题而运行异常，异常的流量镜像服务或流量检测服务可能导致看上去没有在处理流量
- 在 **系统设置 / 设备运行状态 / 网络状态** 中，确认网口吞吐，尤其是镜像口吞吐
  - 镜像服务正常情况下，这里会反应包含镜像口流量在内的吞吐情况，这里是 TCP 层流量，远低于预期时可能根本没有流量（无论是否 http，能否检测）
- 在 **系统设置 / 设备运行状态 / 检测状态** 中，确认检测 QPS
  - 流量检测服务运行正常情况下，每秒检测请求数反应在镜像流量中每秒处理的 http 请求数，它可以反应是否有极低的流量通过
  - 流量小可能是镜像配置的问题，或本身流量不大
  - 流量大检测小说明流量中非 http 流量较多
  - 这里反馈的是实际能处理的 http 请求数

#### 步骤二：底层 shell 确认是否差错网线

使用 `ifconfig | grep -A 4 -e '^eth'` 命令查看各网口的收包，目的是确认是否是客户接错了镜像口。

由于使用 dpdk 进行镜像流量获取，所以 ifconfig 不会有配置了的镜像口，那么 ifconfig 反应的就是接错的非镜像口流量。

![ripley 配置截图2](images/APISEC故障排查FAQ/embedded_p6_3.png)

ifconfig 会中反应一些设备内部网络流量，其中：

- apisec 为 docker 的网桥，用于设备内容器流量交互
- veth 开头为各个容器的虚拟网络设备，反应不同容器的网络流量收发情况

#### 步骤三：底层确认流量镜像服务正常

由于某些情况下，流量镜像服务可能对外表示正常，但内部 IO worker 等运行有问题。

执行 `docker logs --tail 20 apisec-ripley`，确认尾部输出与下图类似，则流量镜像服务运行正常。

![ripley 配置截图](images/APISEC故障排查FAQ/embedded_p6_2.png)

#### 步骤四：确认流量镜像服务没有数据

防止流量镜像的统计数据在处理过程中有异常，导致镜像服务实际处理量没有得到确认。

```bash
cd /data/apisec/resources/ripley/
curl --unix-socket ripley_server.sock http://r/api/v1/stats | grep total_num
curl --unix-socket ripley_server.sock http://r/api/v1/stats | grep tcp_total
```

#### 步骤五：修改回内核驱动，然后设备上抓包

```bash
# 停掉流量镜像服务
docker stop apisec-ripley

# 查看驱动应用情况
dpdk-devbind.py -s

# 修改驱动
dpdk-devbind.py -b <驱动> <pci 地址>
# 如：dpdk-devbind.py -b igb 00:02.0

# 此时使用 ip link 即可看到相应的 ethx.y 的网口设备了，并可以使用 tcpdump 进行抓包了

# 恢复只需
docker restart apisec-ripley
```

## 旁路阻断排查

### 旁路阻断没有阻断成功怎么排查

旁路阻断失败可能有多种原因，必须一一抓包排查。

**1. 客户端抓包。** 旁路阻断的原理中，rst 包需要发往客户端与服务端，触发两端对 tcp 链接的 reset 行为，从而终止后续 tcp 链接上包交换，故首先需要确认客户端是否能够收到 reset 包。

- 如果收到了 reset 包，需要判断 reset 包的接收时间是否晚于带有数据的 ack,psh 包，rst 包必须早于数据包才可能阻断成功，否则必定失败。当 rst 包到达较晚时，一般因为 rst 发出的链路慢于数据包返回的链路，比如 nginx 直接返回的静态资源，其响应一般都早于旁路阻断包
- 如果没有收到 reset 包，说明 reset 包网络链路有问题，需要继续下面的排查步骤

**2. 设备端阻断口抓包。** 关键的自证动作，用来证明设备发出了对应的 rst 包。

- 如果设备没有发出对应的 rst 包，请再进行镜像口抓包，首先确认镜像数据中包含需拦截数据包，然后确认设备内配置规则是否正确。
- 如果设备阻断口有对应的 rst 包，那么说明问题出在 "rst 发往客户端的链路是否可达" 上，需要继续下面的排查步骤

**3. 流量镜像是否有 vlan，几层 vlan，同一 tcp 连接上 vlan 是否相同。** 这里需要抓取 "设备的镜像口" 流量进行确认，客户提供的其他 "镜像流量" 不能保证和设备收到的一模一样，而 rst 包只根据设备收到的镜像流量进行组装。

- 镜像流量不包含 vlan，rst 包一定不会包含 vlan，请确认阻断口对端交换机不需要 vlan 并可达客户端与服务端
- 镜像流量包含相同的 vlan，修改配置文件中的 `vlan_ignore` 为 false，则 rst 包一定包含 vlan，请确认阻断口对端交换机允许该 vlan 并可达客户端与服务端
- 镜像流量同 TCP 连接上包含不同 vlan，一些 tap 流量汇聚分流设备会使同一个 tcp 连接上的数据包在不同方向上 vlan 层数不同，需要保证配置文件中的 `vlan_ignore` 为 true，rst 包一定不带 vlan（因为没法确定 vlan 该如何配置）

**4. 上述检查后未成功，** 需要对链路中每个可抓包节点进行抓包确认，查看是否能够抓到设备发出的 rst 包，检查设备发出的 rst 包在哪个节点被丢失了，并检查上一节点是否对 rst 包进行了预期外的修改（比如以前见过会将 rst 包的 seq 修改为固定的 1111111111111，从而导致后续节点无法将该 rst 归入 tcp 连接最后丢弃）。

## 数据运维

### 设备下架，如何清除所有数据

进 shell 执行如下脚本，包括 license、用户配置、流量数据等等将会被清除，网络配置、ssh 密钥、shell 用户密码等操作系统配置不会变化。

数据清理脚本：

```bash
if [ -d "/data/apisec" ]; then
        systemctl stop apisec-minion
        docker rm -f $(docker ps -a -q)
        rm -rf /data/apisec/resources/postgres/data
        rm -rf /data/apisec/resources/clickhouse/data
        rm -rf /data/apisec/resources/zookeeper/data
        rm -rf /data/apisec/resources/zookeeper/datalog
        rm /data/apisec/resources/redis/dump.rdb
        systemctl restart apisec-minion
        systemctl restart apisec-load
fi
```

目前没有页面一键傻瓜式清除，如果客户提出相关需求，大家可以自行和客户沟通，如无好用的话术，推荐：api 用了很多大数据组件，页面清数据效率太低，如果不考虑二次使用，我们都是建议从底层直接格式化数据目录。

### QPS 并不是很高（~1000），查询请求日志的时候为什么会很慢或超时

这个现象一般出现在软件部署模式，是因为在部署时没有给 zookeeper 额外配置存储目录，使得 zookeeper 和 clickhouse 使用同一块盘存储数据。

这种情况下 zookeeper 会把 iops 跑满（HDD 大约在 200~300），clickhouse 读取时就会超时。相关 issue 可查看 [disk utilization high usage](https://issues.apache.org/jira/browse/CALCITE-5169)。

解决办法是将 zookeeper 和 clickhouse 的数据磁盘分开，可以把 zookeeper 数据盘切换为系统盘（一般为 SSD）或是另一块数据盘，操作方法可见下文《如何迁移 Zookeeper 数据存储目录？》。

### 如何给 Clickhouse 增加数据存储目录

针对 24.04.00x、24.08.00x 版本。

假设之前有系统盘和一块数据盘（/dev/sda），现在新增了一块数据盘（/dev/sdb）。

安装时 clickhouse 的存储目录默认为 `/data/apisec/resources/clickhouse/`。

先初始化并挂载好 /dev/sdb，假设挂载到了 /sdb 目录。

**1、修改 clickhouse 容器的挂载卷**

```bash
EDITOR=vim aminion db edit /minion/v1/services/profile/clickhouse
```

```yaml
volumes:
- ${{host_logs_dir}}/clickhouse:/var/log/clickhouse-server
- ${{host_resources_dir}}/clickhouse/data:/var/lib/clickhouse
```

改为：

```yaml
volumes:
- ${{host_logs_dir}}/clickhouse:/var/log/clickhouse-server
- ${{host_resources_dir}}/clickhouse/data:/var/lib/clickhouse
- /sdb/clickhouse/data:/var/lib/clickhouse/disk_0
```

**2、编辑存储配置文件**

`vim /data/apisec/resources/clickhouse/config.d/storage_configuration.xml`

```xml
<clickhouse>
  <storage_configuration>
    <disks>
      <default>
        <keep_free_space_bytes>53687091200</keep_free_space_bytes>
      </default>
    </disks>
  </storage_configuration>
</clickhouse>
```

改为：

```xml
<clickhouse>
  <storage_configuration>
    <disks>
      <default>
        <keep_free_space_bytes>53687091200</keep_free_space_bytes>
      </default>
      <disk_0>
        <path>/var/lib/clickhouse/disk_0/</path>
        <keep_free_space_bytes>53687091200</keep_free_space_bytes>
      </disk_0>
    </disks>
    <policies>
      <default>
        <volumes>
          <disks>
            <disk>default</disk>
            <disk>disk_0</disk>
          </disks>
        </volumes>
      </default>
    </policies>
  </storage_configuration>
</clickhouse>
```

**3、检查 clickhouse 是否正常**

```bash
systemctl restart apisec-minion
tail -f /data/apisec/logs/clickhouse/clickhouse-server.log
tail -f /data/apisec/logs/clickhouse/clickhouse-server.err.log
tail -f /data/apisec/logs/mario/mario.log
```

以上日志均没有报错。

### Clickhouse 集群如何扩容

**1、** 先在新节点上完成部署、初始化，相关配置参数可以查看已有节点的配置项。

**2、** 在每个分析节点输入：

```bash
EDITOR=vim aminion db edit /minion/v1/services/profile/clickhouse
```

新增节点有两种情况：

- 新增 shard 节点
- 新增 replica 节点

修改 `shard_servers`（新增 shard 节点）或 `replica_servers`（新增 replica 节点）数据。

注意：

- 每个 shard 下最多有两个副本，最少有一个副本（shard 节点本身也是一个副本节点），所以 `replica_servers` 最多为每个 shard 配置一个副本节点
- `replica_servers` 与 `shard_servers` 顺序一一对应

示例：

```
shard_servers: apisec-1:192.168.0.1,apisec-2:192.168.0.2
replica_servers: ""
→ Clickhouse 2个 shard，每个 shard 无 replica

shard_servers: apisec-1:192.168.0.1,apisec-2:192.168.0.2
replica_servers: apisec-3:192.168.0.3
→ Clickhouse 2个 shard，shard=1 的 replica 为 apisec-1 和 apisec-3，shard=2 的 replica 为 apisec-2

shard_servers: apisec-1:192.168.0.1,apisec-2:192.168.0.2
replica_servers: apisec-3:192.168.0.3,apisec-4:192.168.0.4
→ Clickhouse 2个 shard，shard=1 的 replica 为 apisec-1 和 apisec-3，shard=2 的 replica 为 apisec-2 和 apisec-4

shard_servers: apisec-1:192.168.0.1,apisec-2:192.168.0.2
replica_servers: apisec-3:192.168.0.1,apisec-4:192.168.0.3
→ Clickhouse 2个 shard，shard=1 的 replica 为 apisec-1，shard=2 的 replica 为 apisec-2 和 apisec-3
```

**3、** 如果想让数据优先写入某个 shard/replica 则需要手动修改配置文件 `/data/apisec/resources/clickhouse/config.d/remote_servers.xml`

```xml
<clickhouse>
  <remote_servers>
    <apisec_cluster>
      <shard>
        <weight>1</weight>
        <internal_replication>true</internal_replication>
        <replica>
          <priority>1</priority>
          <host>apisec-1</host>
          <port>9000</port>
          <user>apisec</user>
          <password>xxx</password>
        </replica>
        <replica>
          <priority>2</priority>
          <host>apisec-3</host>
          <port>9000</port>
          <user>apisec</user>
          <password>xxx</password>
        </replica>
      </shard>
      <shard>
        <weight>2</weight>
        <internal_replication>true</internal_replication>
        <replica>
          <host>apisec-55</host>
          <port>9000</port>
          <user>apisec</user>
          <password>xxx</password>
        </replica>
      </shard>
    </apisec_cluster>
  </remote_servers>
</clickhouse>
```

- shard：`<weight>1</weight>`，值越大优先级越大
- replica：`<priority>2</priority>`，值越小优先级越大

**4、** 在各分析节点输入：

```bash
systemctl restart apisec-minion
```

**5、** 在管理节点输入：

```bash
EDITOR=vim aminion db edit /minion/v1/services/profile/management
```

修改 `CLICKHOUSE_ANALYSIS_ADDR` 数据：

```yaml
environment:
- CLICKHOUSE_ANALYSIS_ADDR=192.168.0.1,192.168.0.2,192.168.0.3
```

这里填写 `shard_servers`、`replica_servers` 所有节点的 IP。

**6、** 在管理节点输入：

```bash
systemctl restart apisec-minion
```

### 日志分析服务为什么总是显示异常（mario 总重启）

#### 什么原因导致的

1. 大概率是因为 Schema、敏感数据项、站点、API 等数据太多，容器 OOM 了，可以查看 syslog 日志，`cat /var/log/syslog | grep -C 1 -m 1 'CONSTRAINT_MEMCG'` 验证一下
2. 也有小概率是程序 bug，可以查看 mario 容器日志，找到错误原因 `docker logs -n 100 -f apisec-mario` 和 `tail -n 100 -f /data/apisec/logs/mario/mario.log`

#### 如果是 OOM

则可以删除一些无效数据，如 API、敏感数据、Schema 等。如何定义无效数据？比如先筛选出静态文件 API、不再使用的 API，将这些 API 关联的 Schema、敏感数据进行删除。

```sql
-- 查看敏感数据标签数量
docker exec -it apisec-mgt-postgres psql -U apisec

select t1.id,t1.name, t2.count from label_valuecategory as t1
join (select tag_id,count(id) from api_assets_datatags group by tag_id having count(id)>0) as t2
on t1.id=t2.tag_id order by t2.count desc;

-- 静态文件 API 数量
select count(uuid) from api_assets_interface where static=true;
-- 不再使用的 API 数量
select count(uuid) from api_assets_interface where is_in_use=false;
-- 已删除的 Schema 数量
select count(id) from api_assets_schema where is_deleted=true;
-- 静态文件 API 关联的 Schema 数量
select count(id) from api_assets_schema where api_id in (select uuid from api_assets_interface where static=true);
-- 不再使用 API 关联的 Schema 数量
select count(id) from api_assets_schema where api_id in (select uuid from api_assets_interface where is_in_use=false);
```

删除顺序（API → Schema → Schema example → 敏感数据标签）：

```
api_assets_datatags → api_assets_schema_example → api_assets_schema → api_assets_interface
```

#### 如果是程序 bug

则可以联系研发师傅解决。

### 如何解决 Zookeeper 数据存储占用空间过大

在 24.08.x 及以前的版本，zookeeper 使用的是默认配置，没有限制快照数量，会导致 zookeeper 数据存储占用空间过大（比如大几百 G）。

需要修改快照数配置项 **`ZOO_AUTOPURGE_PURGEINTERVAL=3`**，3 表示保留最近 3 份快照文件。

```bash
EDITOR=vim aminion db edit /minion/v1/services/profile/zookeeper
```

```yaml
environment:
- ZOO_MY_ID=1
- ZOO_SERVERS=server.1=OPS-2818:2888:3888;2181
- ZOO_AUTOPURGE_PURGEINTERVAL=3
```

重启 minion：

```bash
systemctl restart apisec-minion
```

### 如何迁移 Zookeeper 数据存储目录

#### 1、查询当前存储目录

```bash
aminion db get /minion/v1/services/profile/zookeeper
```

查找 `volumes` 下的数据，如：

```yaml
volumes:
- ${{host_resources_dir}}/zookeeper/data:/data
- ${{host_resources_dir}}/zookeeper/datalog:/datalog
- ${{host_logs_dir}}/zookeeper:/logs
```

上面为软件部署模式默认配置，需要修改的是：

```yaml
- ${{host_resources_dir}}/zookeeper/data:/data
- ${{host_resources_dir}}/zookeeper/datalog:/datalog
```

#### 2、确认迁移目录

可以额外加块数据盘，也可以移到根目录下（需要保证和数据盘不是一块盘）；以移到根目录为例，替换配置项，保存：

```bash
EDITOR=vim aminion db edit /minion/v1/services/profile/zookeeper
```

```yaml
volumes:
- /zookeeper/data:/data
- /zookeeper/datalog:/datalog
- ${{host_logs_dir}}/zookeeper:/logs
```

#### 3、手动迁移存储数据

停止相关容器：

```bash
docker stop apisec-mario apisec-clickhouse apisec-zookeeper
```

迁移目录，先进入存储目录，查看存储空间大小，正常不会超过 100M，如果太大则需要先清除快照，操作参考《如何解决 Zookeeper 数据存储占用空间过大？》。

```bash
root@OPS-2818:/# cd /data/apisec/resources/zookeeper
root@OPS-2818:/data/apisec/resources/zookeeper# du -h ./
8.0K        ./data/version-2
16K         ./data
22M         ./datalog/version-2
22M         ./datalog
22M         ./
root@OPS-2818:/data/apisec/resources/zookeeper# cp -r /data/apisec/resources/zookeeper /
```

重启相关容器：

```bash
docker restart apisec-mario apisec-clickhouse apisec-zookeeper
```

重启 minion：

```bash
systemctl restart apisec-minion
```

#### 4、检查 zookeeper 是否正常

```bash
tail -f /data/apisec/logs/clickhouse/clickhouse-server.log
tail -f /data/apisec/logs/clickhouse/clickhouse-server.err.log
tail -f /data/apisec/logs/mario/mario.log
```

以上日志均没有报错。
