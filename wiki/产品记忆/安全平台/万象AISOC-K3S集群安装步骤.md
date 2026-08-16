---
type: product
title: 万象AISOC-K3S集群安装步骤
product: 万象AISOC
aliases: []
tags:
    - 安全平台
    - 产品技术手册
    - 部署手册
    - 产品安装
summary: K3S集群部署，master 节点必须大于等于 3，worker 节点数量不要求。
status: verified
---

<!-- toc -->

#### K3S集群安装步骤


K3S集群部署，master 节点必须大于等于 3，worker 节点数量不要求。

>与软件集群管理端集群部署方式几乎一致，仅安装命令和安装后状态校验不同。
>安装命令相较软件集群管理端集群需要将 -e 'INSTALLER_BIGDATA=false' 删除或false改为true  即安装K3S的大数据

将管理端 SSH_PUBLIC_KEY 通过 Base64 解码后独占一行加入到各个节点的 authorized_keys 文件中

```sh
vi /root/.ssh/authorized_keys
```
添加示意图：

![](图片附件/image17.png)



###### 执行部署

```sh
#给安装包执行权限
chmod +x ./cosmos-k3s-*.bin
#管理端单节点安装命令：
#installerpasswd替换为安装包的密码
#1.1.1.1,1.1.1.2,1.1.1.3替换为所有节点IP，多IP之间英文逗号分割。（参数名称与之前版本不同了注意区分！）
#sdb替换为数据盘的盘符
#1.1.1.6替换集群VIP的IP地址；若不设置VIP IP需要删除【-e 'INSTALLER_KUBEVIP_ENDPOINT=1.1.1.6' -e 'INSTALLER_KUBEVIP_IFACE=eth1'】
#eth1替换为绑定VIP IP的网卡名称，与节点IP的网卡名称一致，同时各个节点网卡名称必须一致。
#VIP IP的设置可以保证通过VIP IP访问时任何节点异常不影响页面访问和使用
 mkdir -p /data/ && ./cosmos-k3s-*.bin -C /data/ -p installerpasswd -e 'INSTALLER_NODE_IPS=1.1.1.1,1.1.1.2,1.1.1.3' -e  'INSTALLER_MOUNT_DISK_DEVICES=sdb' -e 'INSTALLER_KUBEVIP_ENDPOINT=1.1.1.6' -e  'INSTALLER_KUBEVIP_IFACE=eth1' -e 'INSTALLER_NO_OUTER_DNS=true'
 
 ```
安装成功示意图：

![](图片附件/image15.png)



<!-- endtoc -->
