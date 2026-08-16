---
type: product
title: 洞鉴（X-Ray）-被动 Web 扫描（日志）
product: 洞鉴（X-Ray）
aliases: []
tags:
    - 漏洞扫描
    - support
    - 功能使用
    - 扫描策略
status: verified
---

# 被动 Web 扫描（日志）

## 扫描任务配置

新建被动 Web 扫描（日志）任务，并配置日志数据来源。

![被动 Web 扫描日志任务配置](../images/web_-1-7591fae02396.png)

### Syslog 服务器连接配置

- **Syslog 服务器 IP**：点击“全局配置”，前往全局配置页面进行设置。
- **Syslog 服务器端口**：填写通信端口。填写后可点击“验证端口可用性”，确认端口可正常使用。

### 日志参数匹配规则

日志参数匹配规则用于解析 Web 访问日志。目前支持多种自定义日志匹配规则，以下以 `COMMONNGINXLOG` 和 `COMMONAPACHELOG` 为例。

#### COMMONNGINXLOG

`COMMONNGINXLOG` 对应默认 Nginx 日志格式，示例如下：

```shell
log_format main '$remote_addr - $remote_user [$time_local] "$request" '
                '$status $body_bytes_sent "$http_referer" '
                '"$http_user_agent" "$http_x_forwarded_for"';

# 日志样例
# 192.168.1.1 - - [19/Mar/2022:16:03:53 +0800] "GET / HTTP/1.1" 304 100 "-" "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/97.0.4692.99 Safari/537.36" "-"
```

#### COMMONAPACHELOG

`COMMONAPACHELOG` 对应默认 Apache 日志格式，示例如下：

```shell
LogFormat "%h %l %u %t \"%r\" %>s %b \"%{Referer}i\" \"%{User-Agent}i\"" combined

# 日志样例
# 192.168.1.2 - - [02/Feb/2016:17:44:13 +0800] "GET /favicon.ico HTTP/1.1" 404 209 "http://localhost/x1.html" "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_11_3) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/48.0.2564.97 Safari/537.36"
```

### 验证日志规则

在“验证日志规则有效性”中输入日志样例，确认当前规则可以正常解析。

![验证日志规则有效性](../images/web_-2-49636f197699.png)

![日志规则验证结果](../images/web_-3-b160bc2051c8.png)

### Host 设置

当日志文件中没有 Host 信息时，可以在此处配置 Host。

- **选择协议**：根据目标站点选择 HTTP 或 HTTPS。
- **服务器 IP 和端口**：填写目标服务器 IP 与端口。
- **日志内 Host 优先选项**：如果日志中配置了 `$host`，优先使用日志中的 Host；如果日志中没有配置 `$host`，则使用此处配置的 IP 或域名。

## 创建扫描任务

确认日志来源、解析规则与 Host 设置后，创建扫描任务。

![创建被动 Web 扫描日志任务](../images/web_-4-bc4ed1019a10.png)

## 配置日志传输客户端

下载“日志传输客户端”辅助工具后，在发送日志的主机上进行配置。如需获取工具，请联系技术支持人员。

客户端适用系统如下：

| 工具名称 | 适用系统 |
| --- | --- |
| `pioneer_darwin_amd64` | macOS 64 位 |
| `pioneer_darwin_386` | macOS 32 位 |
| `pioneer_linux_amd64` | Linux 64 位 |
| `pioneer_linux_386` | Linux 32 位 |
| `pioneer_windows_386` | Windows 32 位 |
| `pioneer_windows_amd64` | Windows 64 位 |

操作命令格式如下：

```shell
[工具名称] send-log-file \
  --log-server-host [使用的引擎 IP] \
  --log-server-port [传输日志的端口] \
  --logfile [日志文件名，多个文件用英文逗号分隔] \
  --log-per-second [发送速率，建议 10]
```

示例：

```shell
./pioneer_linux_amd64 send-log-file \
  --log-server-host 172.22.230.74 \
  --log-server-port 3399 \
  --logfile access.log \
  --log-per-second 10
```

## 查看扫描结果

日志传输客户端开始发送日志后，可在洞鉴 Web 页面查看扫描任务结果。

![被动 Web 扫描日志任务结果](../images/web_-5-c4f2d3fac89e.png)

![被动 Web 扫描日志漏洞结果](../images/web_-6-a4c75a67fc37.png)
