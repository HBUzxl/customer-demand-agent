---
type: product
title: 洞鉴（X-Ray）-Minion 相关问题
product: 洞鉴（X-Ray）
aliases: []
tags:
    - 漏洞扫描
    - support
    - 故障排查
summary: Minion 引擎安装和运行问题的排查方法。
status: verified
---

<!-- toc -->

# Minion相关问题

## 安装时一直显示 Wait for XXXXXX

1、检查是否被主机安全相关软件拦截


2、检查环境内核模块是否缺失

```shell
lsmod |grep ip_tables
```

3、检查Minion日志，查看具体报错原因

```shell
journalctl -u minion -g error

或

journalctl -u minion  |grep error
```


4、性能较低的机器，请关闭容器状态检查功能

```shell
cd /data/x-ray/host/minion/

vim mgmt.service_profile.yml

把文档中的所有下面2个配置都改下（见图1）
status_check: false
allow_failure: true

```

图1
![](../images/upload_3a0a2797002cae904ca4502447be6142.png)

<!-- endtoc -->
