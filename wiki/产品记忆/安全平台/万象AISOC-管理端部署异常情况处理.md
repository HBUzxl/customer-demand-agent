---
type: product
title: 万象AISOC-管理端部署异常情况处理
product: 万象AISOC
aliases: []
tags:
    - 安全平台
    - 产品技术手册
    - 部署手册
    - 产品安装
    - 软件集群安装步骤
summary: 出现异常情况，处理完成后需要重新执行，执行命令： 安装完成后的数据清理：将各个节点的 authorized_keys 文件中因部署添加的 SSH_PUBLIC_KEY
status: verified
---

<!-- toc -->

##### 管理端部署异常情况处理

![](图片附件/image18.png)

出现异常情况，**处理完成后**需要重新执行，执行命令：

```sh
bash /data/scripts/install-use-bmc.sh
```

安装完成后的数据清理：将各个节点的 authorized_keys 文件中因部署添加的 SSH_PUBLIC_KEY 进行删除。

<!-- endtoc -->
