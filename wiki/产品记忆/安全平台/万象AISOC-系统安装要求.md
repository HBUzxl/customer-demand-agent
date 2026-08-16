---
type: product
title: 万象AISOC-系统安装要求
product: 万象AISOC
aliases: []
tags:
    - 安全平台
    - 产品技术手册
    - 部署手册
    - 环境准备
    - 操作系统与安装要求
summary: 介绍系统安装要求相关的部署要求、安装步骤或初始化方法。
status: verified
---

<!-- toc -->

#### 系统安装要求

##### 单机版或集群版管理端

| 项                               | 要求                                                                                                                                      |
| :--------------------------------: | ----------------------------------------------------------------------------------------------------------------------------------------- |
| **语言（LANGUAGE SUPPORT）**         | 英文                                                                                                                                      |
| **时间（DATE & TIME）**          | 时区：Asia/Shanghai <br/> 时间：与北京时间一致 <br/> 若客户有 ntp 服务器可配置，保持各节点时间一致。                                      |
| **部署模式（SOFTWARE SELECTION）**   | 详见《操作系统》部分各操作系统的部署模式                                                                                              |
| **磁盘（INSTALLATION DESTINATION）** | 对系统盘进行操作：<br/> - 删除/swap、/home、/backup <br/> - 保留 /boot、/，部分系统可能有/boot/efi <br/> - 空闲磁盘空间全部给/                  |
| **网络（NETWORK & HOST NAME）**      | - 网口按现场要求配置网络 <br/> - 配置网卡自启：勾选了：Automatically to this network when it is available <br/> - 禁止填写 Search domains |
| **Root 密码（Root Passowrd）**       | 设置 Root 密码                                                                                                                            |

##### 集群版大数据

| 项                               | 要求                                                                                                                                              |
| :--------------------------------: | ------------------------------------------------------------------------------------------------------------------------------------------------- |
| **语言（LANGUAGE SUPPORT）**         | 英文                                                                                                                                              |
| **时间（DATE & TIME）**              | 时区：Asia/Shanghai <br/> 时间：与北京时间一致 <br/> 若客户有 ntp 服务器可配置，保持各节点时间一致。                                              |
| **部署模式（SOFTWARE SELECTION）**   | 详见《操作系统》部分各操作系统的部署模式                               |
| **磁盘（INSTALLATION DESTINATION）** | 对系统盘进行操作：<br/> - 删除/home、/backup <br/> - 保留/swap、/boot、/，部分系统可能有/boot/efi <br/> - 空闲磁盘空间全部给/                     |
| **网络（NETWORK & HOST NAME）**      | - 网口按现场要求配置网络 <br/> - 配置网卡自启：勾选了：Automatically connect to this network when it is available <br/> - 禁止填写 Search domains |
| **Root 密码（Root Passowrd）**       | 设置 Root 密码                                                                                                                                    |

<!-- endtoc -->
