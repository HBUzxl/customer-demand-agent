---
type: product
title: 全悉-全悉流量采集 Agent 部署
product: 全悉
aliases: []
tags:
    - 流量安全
    - 部署手册
summary: 介绍全悉流量采集 Agent 的环境准备、安装部署和卸载流程。
status: verified
---

# 全悉流量采集 Agent 部署

当前只支持 Windows 和 Linux 64 位操作系统，此安装手册适用于未安装过 Agent 的设备。

## 目录

- [1.1 环境准备](#11-环境准备)
- [1.2 部署](#12-部署)
- [1.3 卸载](#13-卸载)

---

## 1.1 环境准备

![流量采集Agent环境准备](images/3-003-image-2025-3-4_14-12-23.png)

1. Agent 和全悉网络可通，Agent→全悉 **tcp:443** 和 **udp:4789** 端口可达
2. 从全悉探针获取 **API Token** 用于对 Agent 的管理
3. 全悉通信口的 IP 地址，用于发送数据和通信管理
4. 数据卡1和数据卡2可以共享单张网卡，此情况下数据卡1最大流量不能超过网卡吞吐的 40%

## 1.2 部署

1. 准备当前系统对应的安装包，上传到服务器任意目录：
   - Windows：`compass-agent-windows.zip`
   - Linux：`compass-agent-linux.tar.gz`

2. 将压缩文件解压到要安装的目录，安装目录就是 Agent 的运行目录

3. 如果为 Windows 版本则需要安装驱动 winpcap-4.13.exe 或者 npcap-1.80.exe，二者安装其一，推荐安装 npcap，安装包内提供相应的驱动

   - **winpcap-4.13.exe** 支持静默安装和手动安装，静默安装则需要使用管理员运行 PowerShell，运行如下命令：

   ```powershell
   Start-Process -Wait -NoNewWindow -FilePath ".\winpcap-4.13.exe" -ArgumentList "/S"
   ```

   - **npcap-1.80** 按照默认选项进行安装即可，如图所示：

   ![npcap安装界面](images/3-002-image-2025-2-14_15-55-36.png)

4. 注册并安装服务，Windows 需以管理员权限运行 PowerShell，Linux 需以 root 运行，执行 `compass_setup` 进行部署安装：

```bash
./compass_setup --help
Usage: compass_setup -k <TOKEN> -a <ANSWER_IP> -i <INTERFACE>

Options:
  -k, --key <TOKEN>            API token for authentication
  -a, --answer-ip <ANSWER_IP>  全悉的管理地址, 用于上报数据包
  -i, --interface <INTERFACE>  Network interface
  -p, --prefix <PREFIX>        采集口名称的前缀
  -m, --mgmt-ip <MGMT_IP>      当前设备采集口的管理IP范围,多个用逗号分隔
  -h, --help                   Print help
```

**参数说明**：

| 参数 | 说明 | 是否必填 |
|------|------|---------|
| `-k, --key` | 从全悉引擎探针申请的 API Token | 必填 |
| `-a, --answer-ip` | 全悉管理地址，用于上报数据包和管理功能 | 必填 |
| `-i, --interface` | 指定流量采集 Agent 采集口网口名称，支持自动选择 auto 或者直接指定网口名称，Linux 通过 ifconfig 或者 ip a 等命令获取，Windows 通过执行 show_interface.exe 列出所有网口，选择的网口格式为 `\\Device\\NPF_{xx-xx-xx-xx-xx}` | 必填 |
| `-p, --prefix` | auto 模式下限定网口范围，建议多网口使用 | 可选 |
| `-m, --mgmt-ip` | auto 模式下限定采集口范围，支持 IP 掩码/范围/单 IP | 可选 |

Linux 举例，API TOKEN 为 uuHnwM6DntdJSr9oIBY1，全悉的管理地址为 10.2.82.4，使用自动模式，并且已知流量采集口（客户业务服务器业务地址）的 IP 地址池为 10.0.0.0/16，程序自动选择的网口为 ens160，如下图所示：

```bash
./compass_setup -k uuHnwM6DntdJSr9oIBY1 -a 10.2.82.4 -i auto -m 10.0.0.0/16
```

![compass_setup Linux举例](images/3-004-image-2025-3-4_14-45-38.png)

5. 同时需要在全悉**【系统管理→设置→采集口配置】**把 `-a` 对应 IP 的网口配置成采集口。

## 1.3 卸载

**Linux**：进入 compass agent 工作目录，停止服务，并删除目录即可。

```bash
cd <path/of/compass>
./CompassService stop
rm -rf <path/of/compass>
```

---
