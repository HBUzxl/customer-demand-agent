---
type: product
title: 洞鉴（X-Ray）-被动 Web 扫描（Burp）
product: 洞鉴（X-Ray）
aliases: []
tags:
    - 漏洞扫描
    - support
    - 功能使用
    - 扫描策略
status: verified
---

# 被动 Web 扫描（Burp）

## 策略介绍

被动 Web 扫描（Burp）支持将 Burp Suite 抓取到的 HTTP 历史记录导出为 XML 文件，并上传到洞鉴进行扫描。

## 操作步骤

**步骤一：**配置 Burp 代理，并通过浏览器访问需要扫描的站点。

![Burp 代理访问目标站点](../images/web_burp_-1-d35f06369eb3.png)

**步骤二：**在 Burp Suite 中进入 `Proxy` -> `HTTP history`，查看生成的浏览记录。

![Burp HTTP history 浏览记录](../images/web_burp_-2-0c7145609983.png)

**步骤三：**全选需要导入洞鉴的浏览记录，右键选择 `Save items`，并保存为 XML 格式。

**步骤四：**在洞鉴中上传 XML 文件，创建扫描任务。

![洞鉴上传 Burp XML 文件](../images/web_burp_-4-b5c604c02074.png)
