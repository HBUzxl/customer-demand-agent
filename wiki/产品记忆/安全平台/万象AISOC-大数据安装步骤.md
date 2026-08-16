---
type: product
title: 万象AISOC-大数据安装步骤
product: 万象AISOC
aliases: []
tags:
    - 安全平台
    - 产品技术手册
    - 部署手册
    - 产品安装
    - 软件集群安装步骤
summary: 将大数据安装包传到大数据集群主节点系统下，传输完成后 ssh 进入主节点系统，进入安装包的所在目录，以 root 用户执行下面命令。
status: verified
---

<!-- toc -->

#### 大数据安装步骤

将大数据安装包传到大数据集群**主节点**系统下，传输完成后 ssh 进入主节点系统，进入安装包的所在目录，以 root 用户执行下面命令。

注意：安装包只需要传到主节点即可，不需要传到每个节点。

大数据主节点选取方法：如果客户给定了 hostname，在 hostname 升序排序在第一个的节点，如 bigdata1、bigdata2 选择 bigdata1。

```sh
#校验 MD5
md5sum ./BD-SW-installer-*.bin
#给安装包执行权限
chmod +x ./BD-SW-installer-*.bin
#大数据安装命令
#installerpasswd 替换为安装包密码
./BD-SW-installer-*.bin -p installerpasswd
```
大数据集群部署开始解压示意图：

![](图片附件/image19.png)

大数据集群部署部署包解压完成示意图：

![](图片附件/image20.png)


- 在访问、控制端打开浏览器访问大数据安装页面：http://解压大数据安装包节点IP:5000

![](图片附件/image21.png)

- 将所有大数据节点的信息按照下图的方式全部填写进去

![](图片附件/image22.png)

![](图片附件/image23.png)

- 等待页面所有任务执行，部署完成后 ssh 进入大数据主节点，执行以下命令查看状态是否正常

```sh
tail -f /var/log/bigdata_monitor.log -n 500
```

![](图片附件/image24.png)

注意：图中标注的 9 个服务如果有异常，等待 10\~20 分钟程序自动处理，如果状态仍异常联系产线进行查看。

<!-- endtoc -->
