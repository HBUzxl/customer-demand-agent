---
type: product
title: APISEC-硬件部署手册
product: APISEC
aliases: []
tags:
    - 流量安全
    - 快速开始
summary: APISEC 硬件设备部署安装步骤。
status: verified
---

# APISEC 硬件部署手册


## 安装过程
1 解压硬件安装包 zip 文件，将文件内容复制到 优盘 根目录

2 将优盘插入硬件的 USB1 口，重启设备，在如下画面中，根据提示进入 setup（系统 BIOS）

![图片](images/APISEC硬件部署手册/image_p1_2.png)

3 在如下位置选择你的优盘进行启动，具体内容视优盘厂商型号不同而不同

![图片](images/APISEC硬件部署手册/image_p1_1.png)

4 在如下界面按回车进入 APISEC Installer

![图片](images/APISEC硬件部署手册/image_p2_3.png)

5 在如下界面使用 root 用户进入 shell

![图片](images/APISEC硬件部署手册/image_p2_4.png)

6 进入优盘所在目录，此处为 /media/sdc1，具体盘符视情况而定，可以使用 ls 查看目录内容是否与优盘一致来判断是否进入了正确的目录

7 执行 ./install.sh <系统盘盘符> <数据盘盘符> 执行安装，请注意下图中的 sda 不一定一直为 120G 的 SSD 系统盘，请仔细确认

![图片](images/APISEC硬件部署手册/image_p3_7.png)

8 执行过程如下图所示

![图片](images/APISEC硬件部署手册/image_p3_5.png)

9 执行至此时，对于挂载了一块以上的数据盘的设备，可输入 y 回车并继续在此输入盘符，额外的数据盘将用来增加数据计算服务的吞吐。若没有额外的数据盘，则输入 n 回车即可。

![图片](images/APISEC硬件部署手册/image_p3_6.png)

10 安装正常完成后如下图所示

![图片](images/APISEC硬件部署手册/image_p4_8.png)

11 重启设备

12 完成重启后，系统将自动开始安装，并在安装结束后自动重启设备

![图片](images/APISEC硬件部署手册/image_p4_9.png)

13 可以在系统底层shell中执行如下命令查看安装日志 `journalctl -u deploy-apisec.service -f`

![图片](images/APISEC硬件部署手册/image_p5_11.png)

14 若失败，则执行以下命令重试 `systemctl restart apisec-load`

## 安装成功验证
1 安装过程结束后重启设备

2 设备启动后通过 docker ps 查看，容器运行均正常（mario 处于 unhealthy 状态是预期中的），没用容器不断重启的现象

![图片](images/APISEC硬件部署手册/image_p5_10.png)

3 systemctl status sshd 查看 ssh 运行状态正常
