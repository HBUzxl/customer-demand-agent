---
type: product
title: 万象AISOC-单机版安装步骤
product: 万象AISOC
aliases: []
tags:
    - 安全平台
    - 产品技术手册
    - 部署手册
    - 产品安装
summary: 将全量安装包传到将要部署单机版系统下，ssh 登录单机版系统，传输完成后进入安装包的所在目录，以 root 用户执行下面命令。
status: verified
---

<!-- toc -->

#### 单机版安装步骤

将全量安装包传到将要部署单机版系统下，ssh 登录单机版系统，传输完成后进入安装包的所在目录，以 root 用户执行下面命令。

```sh
#校验 MD5
md5sum ./cosmos-k3s-*.bin
```
MD5计算与对比示意图：

![](图片附件/image14.png)


```sh
#给安装包执行权限 
chmod +x ./cosmos-k3s-*.bin 
#执行安装。installerpasswd 替换为安装包的密码；1.1.1.1 替换为本机 IP；sdb 替换为数据盘的盘符
mkdir -p /data/ && ./cosmos-k3s-*.bin -C /data/ -p installerpasswd -e 'INSTALLER_MASTER_IPS=1.1.1.1' -e  'INSTALLER_MOUNT_DISK_DEVICES=sdb' 
```
安装成功示意图：

![](图片附件/image15.png)

```sh
#查看 pod 状态 
kubectl get pod
```

![](图片附件/image16.png)
<!-- endtoc -->
