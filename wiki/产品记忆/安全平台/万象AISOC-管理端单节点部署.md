---
type: product
title: 万象AISOC-管理端单节点部署
product: 万象AISOC
aliases: []
tags:
    - 安全平台
    - 产品技术手册
    - 部署手册
    - 产品安装
    - 软件集群安装步骤
summary: 将管理端安装包传到管理端节点系统下或管理端集群主节点系统下，传输完成后 ssh 登录主节点系统，进入安装包的所在目录，以 root 用户执行下面命令。
status: verified
---

<!-- toc -->

将管理端安装包传到管理端节点系统下或管理端集群主节点系统下，传输完成后 ssh 登录主节点系统，进入安装包的所在目录，以 root 用户执行下面命令。

##### 管理端单节点部署

```sh
#校验 MD5
md5sum ./cosmos-k3s-*.bin
#给安装包执行权限
chmod +x ./cosmos-k3s-*.bin
#管理端单节点安装命令
#执行安装。installerpasswd 替换为安装包的密码；1.1.1.1 替换为本机 IP；sdb 替换为数据盘的盘符
mkdir -p /data/ && ./cosmos-k3s-*.bin -C /data/ -p installerpasswd -e 'INSTALLER_MASTER_IPS=1.1.1.1' -e 'INSTALLER_MOUNT_DISK_DEVICES=sdb' -e 'INSTALLER_BIGDATA=false' -e 'INSTALLER_NO_OUTER_DNS=true'
```

安装成功示意图：

![](图片附件/image15.png)

<!-- endtoc -->
