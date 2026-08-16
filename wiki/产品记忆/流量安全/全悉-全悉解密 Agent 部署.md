---
type: product
title: 全悉-全悉解密 Agent 部署
product: 全悉
aliases: []
tags:
    - 流量安全
    - 部署手册
summary: 介绍全悉解密 Agent 的部署条件、典型使用方式、启动方式和全悉侧配置。
status: verified
---

# 全悉解密 Agent 部署

## 目录

- [1.1 注意事项](#11-注意事项)
- [1.2 典型使用方式](#12-典型使用方式)
- [1.3 部署和使用](#13-部署和使用)
  - [下载解密 Agent 安装包](#下载解密-agent-安装包)
  - [启动解密 Agent](#启动解密-agent)
- [1.4 全悉配置](#14-全悉配置)

---

## 1.1 注意事项

1. 解密 Agent 只支持 **Linux 系统**，且要求内核版本 ≥ 4.19，不支持 Windows 系统
2. 解密 Agent 应当部署在 **服务端设备**上，而非客户端设备上。（例如，Client(加密) → Server(加密)，解密 Agent 应部署在 Server 端设备上）
3. 解密 Agent 会将 Key 传输给全悉，因此需要网络端口可达。全悉默认使用 **50051 端口**来接收解密 Agent 传输过来的 Key，也可以自定义端口，在全悉配置章节中有介绍

## 1.2 典型使用方式

![典型使用方式](images/4-典型使用方式.png)

## 1.3 部署和使用

### 下载解密 Agent 安装包

- 如果客户侧，部署了牧云，可以直接通过牧云进行安装解密 Agent，详情咨询牧云的师傅
- 如果未部署牧云，则可以使用单独的安装包

  **【安装包下载】** easycapture.tar.gz，包含2个文件：`easycapture` 可执行文件 和 `capture_ca.crt` 证书文件

  **【上传】** 需要将 `easycapture.tar.gz` 文件上传到目标服务器

### 启动解密 Agent

【启动】前台启动运行或放置后台运行，2 选 1 即可。

**前台启动运行**（便于查看日志）：

```bash
tar zxvf easycapture.tar.gz
cd easycapture
./easycapture run --mode grpc --target https://x.x.x.x:50051/
-- 其中 https://x.x.x.x:50051/ 中的 x.x.x.x 是全悉 IP 地址，
-- 50051是全悉的监听口，可在全悉配置文件中自定义，见全悉配置章节
-- 【停止】直接 Ctrl + c 即可
```

**放置后台运行**（推荐，长时间运行）：

```bash
tar zxvf easycapture.tar.gz
cd easycapture
nohup ./easycapture run --mode grpc --target https://x.x.x.x:50051/ > /dev/null 2>&1 &
-- 其中 https://x.x.x.x:50051/ 中的 x.x.x.x 是全悉 IP 地址
-- 50051是全悉的监听口，可在全悉配置文件中自定义，见全悉配置章节
-- 【停止】执行命令 killall easycapture 即可
```

## 1.4 全悉配置

默认情况下，全悉为了性能考虑，默认关闭 TLS 解密功能。

**【开启解密功能】**：

```bash
# 步骤1：修改配置文件
vi /data/jupiter/config/ganymede/config-reload.yaml
# 将 sslkeylog 下的 enabled 字段值，从 no 改成 yes

# 步骤2（可选）：接收解密 Agent 传输 Key 的监听口默认 50051
# 如果想自定义监听口，可以手动修改 sslkeylog 下的 listen-port 字段值

# 步骤3：重启千手
docker restart ganymede
```
