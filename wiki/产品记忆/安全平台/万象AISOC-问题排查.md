---
type: product
title: 万象AISOC-问题排查
product: 万象AISOC
aliases: []
tags:
    - 安全平台
    - 产品技术手册
summary: 部署 Q：部署完成后能改IP么？
status: verified
---

<!-- toc -->

## FAQ
**部署**

Q：部署完成后能改IP么？  
A：部分场景支持。安装版本（不是升级后的版本）为CM-S20-24.11.006的手动安装【单机版】或自动化罐装系统罐装的【单机版】允许改IP，其余场景不允许。不允许场景更改IP可能会导致万象整体崩溃无法使用，慎重执行。
   
   

Q：部署完能够更改时间么？  
A：可以更改为未来时间；禁止将时间改回过去时间会造成服务瘫痪。（以万象各节点系统时间为准判断过去还是未来。）

 

Q：是否支持双网卡？  
A：支持，需遵循IPv4协议要求，在网卡配置上一个网卡配置路由另一个网卡不配路由。在万象部署过程涉及到填写IP时，填写配置路由的IP。


Q：双网卡部署场景下，一个光口一个电口，配置路由的光口未连接光纤可以部署么？  
A：可以，所有网卡的信息配置好后，可通过电口ssh进入操作系统执行部署命令。（ps：部署期间需要有gateway，连通性不强制）

<br/>

**采集探针**

Q：Syslog发送设备存在规范RFC3164、RFC5424；协议TCP、UDP的选项，应如何进行配置？  
A：建议配置为 TCP+RFC5424：RFC5424 规范相较 RFC3164 更新，支持的消息长度更大；TCP 相较 UDP 更稳定。
  
Q：Syslog 加密传输如何配置？  
A:万象缺省配置为非加密端口 514，加密端口 515；在采集探针页面，编辑探针采集方式时可查看当前配置的端口情况。在使用加密端口时需要确认发送 Syslog 的设备对加密的支持情况。

<br/>
 
**其他**

Q：为什么设备监测页面有设备但是没有检测信息？  
A：设备监测是定制化的功能，需要联系产线进行处置：基于 SNMP 收集到的信息通过 SOAR 进行适配处理，最终在万象设备监测页面展示。

Q：什么是日志，什么是告警？  
A：在万象中，由采集探针通过主动（FTP、Kafka 等）或被动（Syslog）收集到的数据称为"原始日志"；原始日志需要通过数据接入规则处理成万象平台可用的"安全日志"（也可简称为日志）；使用者对安全日志的筛选逻辑固化为万象的关联分析规则或自然语境规则，可生成"安全告警"（也可简称为告警）供使用者进行分析和处置。

Q：部署完成后 hghac-see-bigdata-init 开头的 pod 状态异常？  
A:hghac 开头的 pod 是国产化数据库的容器，需要授权激活，目前与通用数据库共存，若非项目要求使用国产化数据库，该容器状态可忽略。

Q：万象如何卸载？  
A：暂时没有卸载方法，如果为达到清除数据的目的，可通过重装系统或格式化磁盘等方式进行处理。

Q：用户自定义数据导入注意事项？  
A：下载模板后，根据不同字段类型根据每列第一行数据的格式进行填写，目前不允许一行数据中部分单元格存在空的情况。

<br/>

**运行维护**

Q：异常断电后单机版、管理端pedestal、redis节点STATUS不是Running如何处理？  
A：执行以下命令
```sh
# 所有管理端节点或K3S集群节点上执行下方命令
rm -rf /var/lib/rancher/k3s/storage/pvc-*_default_redis-data-shared-redis-node-*/{appendonlydir,*.rdb} 

# 任意管理端节点或K3S集群节点上执行下方命令
kubectl rollout restart sts shared-redis-node
kubectl rollout restart deploy pedestal

# 查看pod状态
kubectl get pod
```
<br/>

Q：物理服务器异常重启磁盘无法挂载，进入操作系统安全模式（单机版或管理端节点），发现数据盘没有挂载如何处理？  
A：按下面方法尝试进行处置  
1.清除磁盘异常状态
```sh
##若为软件版集群-管理端节点，执行
xfs_repair -L /dev/mapper/cosmos_k3s-var_lib_rancher_k3s

##若为单机版或K3S集群，执行
xfs_repair -L /dev/sdb1
```
![](图片附件/管理端进入安全模式异常状态清除.png)

2.挂载磁盘
```sh
#挂载数据盘
mount -a

#查看挂载情况
lsblk
```
![](图片附件/管理端进入安全模式挂载成功.png)

3.重启系统，查看容器运行状态
```sh
#重启系统
reboot

#ssh进入系统后查看服务恢复状态
kubectl get pod
```

<br/>

Q：安装单机版或管理端报错“Disk usage is less than 1T for mount point /var/lib/rancher/k3s”如何处理

A：安装命令后面追加【-e "INSTALLER_CHECK_K3S_DISK_SIZE=false"】，然后重新执行安装命令



<!-- endtoc -->
