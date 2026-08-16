---
type: product
title: 万象AISOC-管理端部署其他参数
product: 万象AISOC
aliases: []
tags:
    - 安全平台
    - 产品技术手册
    - 部署手册
    - 产品安装
    - 软件集群安装步骤
summary: 注意：参数和参数之间一个空格分割，在表格中用橙色背景色标记出来。
status: verified
---

<!-- toc -->

##### 管理端部署其他参数

注意：参数和参数之间一个空格分割，在表格中用橙色背景色标记出来

| 参数                       | 项       | 描述                                                                                                                                                                     |
| -------------------------- | -------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| INSTALLER_BIGDATA          | 描述     | 是否安装 k3s 大数据，参数值 true 或 false。不使用参数默认为 true，即安装 k3s 大数据                                                                                      |
|                            | 使用方法 | -e 'INSTALLER_BIGDATA=false'                                                                                                                                           |
|                            | 备注     | k3s 的大数据仅适用于单机版。                                                                                                                                             |
| INSTALER_CHANGE_HOSTNAME   | 描述     | 是否自动修改操作系统的 hostname，参数值 true 或 false。不使用参数默认为 true，即自动更改 hostname                                                                        |
|                            | 使用方法 | -e 'INSTALLER_CHANGE_HOSTNAME=false'                                                                                                                                   |
|                            | 备注     | false 时需要手动对管理端各节点的 hostname 进行不重名处理，否则使用操作系统缺省 hostname 安装过程会报错。                                                                 |
| INSTALLER_KUBEVIP_ENDPOINT | 描述     | **管理端集群模式**下，集群 Leader 是第一个节点，若当该节点通信异常，整个集群异常。该参数设置一个 VIP IP，Leader 异常时会在 master 节点选举新的 Leader 保持集群通信正常。 |
|                            | 使用方法 | -e 'INSTALLER_KUBEVIP_ENDPOINT=1.1.1.6'                                                                                                                                |
|                            | 备注     | 该 IP 与管理端其他节点 IP 同网段，但不允许重复。该参数**必须**与 INSTALLER_KUBEVIP_IFACE 同时使用                                                                        |
| INSTALLER_KUBEVIP_IFACE    | 描述     | 集群模式下，设置 VIP IP 绑定的网卡名称                                                                                                                                   |
|                            | 使用方法 | -e 'INSTALLER_KUBEVIP_IFACE=eth1'                                                                                                                                      |
|                            | 备注     | 管理端所有 master 节点网卡名称必须一致用方法该参用方法数**必须**与 INSTALLER_KUBEVIP_ENDPOINT 同时使用                                                                   |
| INSTALLER_NO_OUTER_DNS     | 描述     | k3s 默认要求有一个 dns server ，没有 DNS server 场景下安装会失败。此参数会启动一个 mock dns server 忽略 dns server 检查。                                                |
|                            | 使用方法 | -e 'INSTALLER_NO_OUTER_DNS=true'                                                                                                                                       |
|                            | 备注     | 开启此选项，默认无法访问机器外部 dns 做域名解析。如需使用外部 dns server，建议不要使用此参数                                                                             |

<!-- endtoc -->
