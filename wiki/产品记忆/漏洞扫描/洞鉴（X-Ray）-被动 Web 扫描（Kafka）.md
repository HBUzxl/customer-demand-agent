---
type: product
title: 洞鉴（X-Ray）-被动 Web 扫描（Kafka）
product: 洞鉴（X-Ray）
aliases: []
tags:
    - 漏洞扫描
    - support
    - 功能使用
    - 扫描策略
status: verified
---

# 被动 Web 扫描（Kafka）

## 新建扫描任务

新建被动 Web 扫描（Kafka）任务，并配置用于接收 Kafka 消息的引擎地址。

![被动 Web 扫描 Kafka 任务配置](../images/web_kafka_-1-680aa25584d2.png)

关键配置说明：

- **接收地址**：填写接收 Kafka 消息的引擎 IP。单机部署时填写 `169.254.1.1`；分布式部署时填写对应引擎节点 IP。
- **监听端口**：根据实际需求填写，需确保端口未被占用且网络可达。

## 使用 Kafka 工具

### 生成配置文件

首次执行 Kafka 工具时，如果缺少配置文件，会报错并生成对应的 `yml` 配置文件。

![首次执行生成 yml 配置文件](../images/web_kafka_-2-a0781a9310c2.png)

### 编辑配置文件

根据任务中的接收地址、监听端口等信息，编辑生成的 `yml` 配置文件。

![编辑 Kafka 工具 yml 配置](../images/web_kafka_-3-eff2bb9e32db.png)

### 启动 Kafka 工具

配置完成后，执行 Kafka 工具文件。

![启动 Kafka 工具](../images/web_kafka_-4-8b5ab2a998d3.png)

出现连接成功提示后，即表示 Kafka 工具已正常连接。

## 确认任务连接状态

Kafka 工具连接成功后，可在任务视角中查看连接状态。

![Kafka 工具连接成功状态](../images/web_kafka_-5-bd6b07132a9b.png)

## 测试生产数据

可在 Kafka 的 `test` topic 中测试生产数据，并在任务中心查看是否检测到漏洞。

Kafka 生产消息示例：

```json
{"url":"http://196.134.58.226:8088/vulnerabilities/brute/","method":"POST","headers":{"Content-Length":239,"User-Agent":"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_13_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/84.0.4147.105 Safari/537.36"},"body":"ZGF0YT0lM0MlM0Z4bWwrdmVyc2lvbiUzRCUyMjEuMCUyMiUzRiUzRSUwRCUwQSUzQyUyMURPQ1RZUEUrQU5ZKyU1QiUwRCUwQSUwOSUzQyUyMUVOVElUWStjb250ZW50K1NZU1RFTSslMjJmaWxlJTNBJTJGJTJGJTJGZXRjJTJGcGFzc3dkJTIyJTNFJTBEJTBBJTVEJTNFJTBEJTBBJTNDbm90ZSUzRSUwRCUwQSUwOSUzQ25hbWUlM0UlMjZjb250ZW50JTNCJTNDJTJGbmFtZSUzRSUwRCUwQSUzQyUyRm5vdGUlM0UlMDklMDk="}
{"url":"http://196.134.58.226:8088/vulnerabilities/brute/","method":"POST","body":"ZGF0YT0lM0MlM0Z4bWwrdmVyc2lvbiUzRCUyMjEuMCUyMiUzRiUzRSUwRCUwQSUzQyUyMURPQ1RZUEUrQU5ZKyU1QiUwRCUwQSUwOSUzQyUyMUVOVElUWStjb250ZW50K1NZU1RFTSslMjJmaWxlJTNBJTJGJTJGJTJGZXRjJTJGcGFzc3dkJTIyJTNFJTBEJTBBJTVEJTNFJTBEJTBBJTNDbm90ZSUzRSUwRCUwQSUwOSUzQ25hbWUlM0UlMjZjb250ZW50JTNCJTNDJTJGbmFtZSUzRSUwRCUwQSUzQyUyRm5vdGUlM0UlMDklMDk="}
{"url":"http://196.134.58.226:8088/vulnerabilities/brute/","method":"POST","body":"ZGF0YT0lM0MlM0Z4bWwrdmVyc2lvbiUzRCUyMjEuMCUyMiUzRiUzRSUwRCUwQSUzQyUyMURPQ1RZUEUrQU5ZKyU1QiUwRCUwQSUwOSUzQyUyMUVOVElUWStjb250ZW50K1NZU1RFTSslMjJmaWxlJTNBJTJGJTJGJTJGZXRjJTJGcGFzc3dkJTIyJTNFJTBEJTBBJTVEJTNFJTBEJTBBJTNDbm90ZSUzRSUwRCUwQSUwOSUzQ25hbWUlM0UlMjZjb250ZW50JTNCJTNDJTJGbmFtZSUzRSUwRCUwQSUzQyUyRm5vdGUlM0UlMDklMDk="}
{"url":"http://196.134.58.226:8088/vulnerabilities/brute/","method":"POST","body":"ZGF0YT0lM0MlM0Z4bWwrdmVyc2lvbiUzRCUyMjEuMCUyMiUzRiUzRSUwRCUwQSUzQyUyMURPQ1RZUEUrQU5ZKyU1QiUwRCUwQSUwOSUzQyUyMUVOVElUWStjb250ZW50K1NZU1RFTSslMjJmaWxlJTNBJTJGJTJGJTJGZXRjJTJGcGFzc3dkJTIyJTNFJTBEJTBBJTVEJTNFJTBEJTBBJTNDbm90ZSUzRSUwRCUwQSUwOSUzQ25hbWUlM0UlMjZjb250ZW50JTNCJTNDJTJGbmFtZSUzRSUwRCUwQSUzQyUyRm5vdGUlM0UlMDklMDk="}
```
