---
type: product
title: 洞鉴（X-Ray）-Docker 相关问题
product: 洞鉴（X-Ray）
aliases: []
tags:
    - 漏洞扫描
    - support
    - 故障排查
summary: 安装部署过程中常见 Docker 问题的排查和解决方法。
status: verified
---

<!-- toc -->

# Docker相关问题


## 安装过程中报错，出现 no such image 字样

---

大部分情况是由于硬盘空间不足导致的,运行脚本查看磁盘剩余空间。
```shell
 df -h
```

> 查看硬盘状态，docker 默认镜像存储路径为 /var/lib/docker/,  需要查看这个路径挂载的硬盘还有没有剩余空间。

解决方式：

1. 加硬盘
2. 修改 docker 默认存储路径（如果客户之前有使用 docker 的话，请慎重操作。）


如何修改docker默认存储路径？

> 以从“/var/lib/docker”迁移到“/foo/bar/docker”为例。


```shell


停止 minion及docker 服务：
/data/x-ray/minion stop

systemctl stop docker

同步数据至新的数据目录：
rsync -avzP /var/lib/docker /foo/bar/

备份原数据目录：
mv /var/lib/docker /var/lib/docker.bak

应用新的数据目录，有两种方法：

1.修改默认路径
编辑 docker 服务配置文件（通常位于 /etc/systemd/system/docker.service 或 /usr/lib/systemd/system/docker.service），在“ExecStart”一行的启动命令中添加“--graph=/foo/bar/docker”，然后执行 

systemctl daemon-reload


2.软链接
ln -s /foo/bar/docker /var/lib

启动minion及docker 服务：
/data/x-ray/minion start 
systemctl start docker

查看结果：观察镜像、容器是否正常
docker images
docker ps

移除备份数据：
rm -rf /var/lib/docker.bak
```


---

# 客户网段是172.17.0.0/16与洞鉴docker网络冲突了怎么办？

---

进入洞鉴的shell，硬件版需要获取passcode。

```shell
cd /data/x-ray
./minion stop
vi /etc/docker/daemon.json

## 增加如下配置,  其中10.0.0.1为和客户不冲突的网段。  
"bip":"10.0.0.1/24"


systemctl restart docker
./minion start

<!-- endtoc -->
