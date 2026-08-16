---
type: product
title: 谛听-Chaitin-CLI 使用说明
product: 谛听
aliases: []
tags:
    - 端点安全
    - 25.12.001_r0
    - 配置参考
summary: 介绍 chaitin-cli 命令行工具的安装、配置及谛听产品的使用方法。
status: verified
---

# Chaitin-CLI 使用说明

Chaitin-CLI 是长亭科技的统一命令行工具，可以在一个二进制中管理 SafeLine、X-Ray、Dsensor（谛听）、CloudWalker 等多款产品，便于运维人员和自动化脚本统一调用。

项目地址：[https://github.com/chaitin/chaitin-cli](https://github.com/chaitin/chaitin-cli)

本文以 Linux 系统为例，介绍 chaitin-cli 连接谛听产品的完整配置和使用流程。

## 安装

执行以下命令一键安装：

```bash
curl -fsSL https://raw.githubusercontent.com/chaitin/chaitin-cli/main/skills/chaitin-cli/scripts/install-chaitin-cli.sh | bash
```

![安装 chaitin-cli](images/chaitin-cli-install.png)

<center>图8.1-1 安装 chaitin-cli</center>

安装完成后，`chaitin-cli` 将放置在 `/usr/local/bin/` 目录下，可直接在终端使用。

> **提示**：Windows 用户请从 [GitHub Releases](https://github.com/chaitin/chaitin-cli/releases) 下载对应版本。

## 获取 API Token

chaitin-cli 通过 API Token 认证访问谛听平台。获取方式：

1. 登录谛听管理界面
2. 点击右上角 **用户 → 个人中心 → OpenAPI**
3. 点击"生成"创建 API Token（请妥善保管，部分版本仅展示一次）

## 创建配置文件

chaitin-cli 默认从**当前工作目录**下的 `config.yaml` 读取配置。在 `/root/` 目录下创建配置文件：

```bash
cat > /root/config.yaml << 'EOF'
dsensor:
  url: https://你的谛听地址
  api_key: 你的API-Token
EOF
```

![创建 config.yaml](images/chaitin-cli-config.png)

<center>图8.1-2 创建配置文件</center>

也可以通过环境变量配置，优先级为 **命令行参数 > 环境变量 > config.yaml**：

```bash
export DSENSOR_URL=https://你的谛听地址
export DSENSOR_API_KEY=你的API-Token
```

> **注意**：使用 chaitin-cli 时需要在 `config.yaml` 所在目录下执行，或通过 `-c` 参数指定配置文件路径：
> ```bash
> chaitin-cli -c /root/config.yaml dsensor --help
> ```

## 常用参数

| 参数 | 说明 |
|------|------|
| `--insecure` | 跳过 TLS 证书验证（自签名证书时需要） |
| `-o json` | 输出 JSON 格式（默认为 table） |
| `-o table` | 输出表格格式 |
| `-v` | 打印请求路径和 body（调试用） |
| `-c config.yaml` | 指定配置文件路径（默认当前目录 `config.yaml`） |
| `--dry-run` | 只打印请求内容，不实际发送 |

## 查看可用命令

### 查看顶层模块

```bash
chaitin-cli dsensor --help
```

![查看顶层模块](images/chaitin-cli-help.png)

<center>图8.1-3 查看顶层模块</center>

谛听支持以下模块：

| 模块 | 说明 |
|------|------|
| `account` | 用户信息 |
| `agent` | 探针管理 |
| `alarm` | 告警配置 |
| `archive` | 日志归档管理 |
| `audit` | 用户操作日志 |
| `captain` | 集中管理 |
| `event` | 攻击者画像与威胁日志 |
| `honeypot` | 蜜罐管理 |
| `intellectual` | 智学习 |
| `license` | 许可证信息 |
| `report` | 报告管理 |
| `syslog` | Syslog 管理与系统日志 |
| `system` | 系统配置与系统信息 |

### 查看模块下的具体命令

```bash
chaitin-cli dsensor agent --help      # 探针管理子命令
chaitin-cli dsensor event --help      # 攻击事件子命令
chaitin-cli dsensor honeypot --help   # 蜜罐管理子命令
```

![查看子命令](images/chaitin-cli-subcommand.png)

<center>图8.1-4 查看模块下的子命令</center>

## 命令示例

### 获取蜜网蜜罐信息

```bash
cd /root
chaitin-cli dsensor --insecure honeypot all_honey -o json
```

![蜜罐信息查询](images/chaitin-cli-example.png)

<center>图8.1-5 获取蜜网蜜罐信息</center>

### 获取探针列表

```bash
chaitin-cli dsensor --insecure agent agent -o json
```

### 查询系统 CPU/内存状态

```bash
chaitin-cli dsensor --insecure system host_cpumem_stat -o json
```

### 查看攻击者画像

```bash
chaitin-cli dsensor --insecure event --help
```

### 调试模式（查看请求内容）

```bash
chaitin-cli dsensor --insecure -v system host_cpumem_stat
```

## 常见问题

### 执行命令提示 `no API runner configured (missing --url?)`

`config.yaml` 不在当前目录。解决方式：

```bash
cd /root && chaitin-cli dsensor --help
# 或
chaitin-cli -c /root/config.yaml dsensor --help
```

### 执行命令提示 401 `login_required`

API Token 认证失败。请确认：

1. API Token 是否正确（在管理界面 个人中心 → OpenAPI 处重新生成）
2. `config.yaml` 中的 `url` 和 `api_key` 是否填写正确
3. 确认使用的是修复后的 chaitin-cli 版本（发送 `API-Token` Header 而非 `X-API-Key`）

### 自签名证书连接失败

使用 `--insecure` 参数跳过 TLS 证书验证：

```bash
chaitin-cli dsensor --insecure honeypot all_honey
```
