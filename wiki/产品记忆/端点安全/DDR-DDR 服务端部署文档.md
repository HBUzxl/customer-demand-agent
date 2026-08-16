---
type: product
title: DDR-DDR 服务端部署文档
product: DDR
aliases: []
tags:
    - 端点安全
    - v3.10.1
    - 快速开始
summary: 介绍 DDR v3.9 服务端的系统要求、安装部署、版本升级、授权和异常排查方法。
status: verified
---

# DDR 服务端部署文档

## 一、系统及软件要求

- 服务器 CPU 主频 2\.5GHz 以上，文件系统 XFS

- 根据平均值估算，网络规划应按每 1,000 个终端预留约 80 Mbps 的带宽

- 生产环境建议使用 SAS 10K 以上规格的硬盘

- 服务端不支持 IPV6

|**终端数量**|**CPU**|**内存**|**网络**|**本地存储**|**数据存储时长**|**机器数量**|
|---|---|---|---|---|---|---|
|1000|8|32|千兆网卡|1T|1年|1台|
|2000|16|64|千兆网卡|2T|1年|1台|
|4000|32|128|千兆网卡|4T|1年|1台|

<table>
  <tr>
    <th>操作系统支持版本(选择一种)</th>
    <th>备注</th>
  </tr>
  <tr>
    <td>Ubuntu 24.04.2 LTS x86_64</td>
    <td rowspan="14">部署后端服务，人员需要机器的ROOT权限或SUDO权限</td>
  </tr>
  <tr><td>Rocky Linux 10 x86_64</td></tr>
  <tr><td>Rocky Linux 9 x86_64</td></tr>
  <tr><td>Rocky Linux 8 x86_64</td></tr>
  <tr><td>Red Hat Enterprise Linux 10 x86_64</td></tr>
  <tr><td>Red Hat Enterprise Linux 9 x86_64</td></tr>
  <tr><td>Red Hat Enterprise Linux 8 x86_64</td></tr>
  <tr><td>Centos 10 (stream) x86_64</td></tr>
  <tr><td>Centos 9 (stream) x86_64</td></tr>
  <tr><td>Centos 8 (stream) x86_64</td></tr>
  <tr><td>Amazon Linux 2023 x86_64</td></tr>
  <tr><td>龙蜥操作系统 Anolis OS 8 x86_64</td></tr>
  <tr><td>龙蜥操作系统 Anolis OS 7.9 x86_64</td></tr>
  <tr><td>银河麒麟高级服务器操作系统（AMD64版）V10</td></tr>
</table>

- 即使只测试 1-2 台终端，也会有基本的 CPU 和内存要求，⽽不是可以⼀直降下去，⽐如
  4C8G ⽆法满⾜服务的启动
--- 

## 二、需要开放的端口

|**开放端口类型**|**开放端口**|**开放范围**|**端口用途**|
|---|---|---|---|
|TCP|8441|全网放开|终端接口|
||8442|全网放开|终端webview|
||8443|管理员IP|web控制台|
||8445\(可选\)|全网放开|浏览器插件管控|
||13125|全网放开|用于终端消息通信|

## 三、域名及证书申请（可选）

|**域名**|**SSL证书**|**备注**|
|---|---|---|
|ddr\-xxxx\.xxx\.com|可选|作为web控制台登陆主域名|

## 四、安装部署

  注意:

- 将提供的安装包放置到您需要部署运⾏的位置，推荐放置到 /home
  ⽬录下，后续沟通均默认以此⽬录为基准,请保证该⽬录磁盘可⽤空间满⾜资源标准

<!-- -->

- 所有命令请以 root 权限或者 sudo 超级权限执行

### 1、首次部署

  执行命令即可一键部署完成：

```bash
./qzh-installer-3-9-*.bin -p "$(< qzh-installer-3-9-*.bin.password)"
```

  如果需要部署在非默认安装目录，需要添加参数（示例）：-e
ddr_home=/data，则会部署到 /data 目录下：

```bash
./qzh-installer-3-9-0-release-20260209-*.bin -p "$(< qzh-installer-3-9-0-release-20260209-*.bin.password)"
```

  默认部署英文版，如需部署中文版需要添加参数：-e
language=zh-hans，如需部署到 /data 目录下，语言为中文：

```bash
./qzh-installer-3-9-*.bin -p "$(< qzh-installer-3-9-*.bin.password)" -e ddr_home=/data -e language=zh-hans
```

  一键卸载**：**

```bash
cd ${ddr_home}/data/bin && bash tool.sh uninstall
```

### 2、需要开放的端口

<table style="width:80%;">
<colgroup>
<col style="width: 20%" />
<col style="width: 20%" />
<col style="width: 20%" />
<col style="width: 20%" />
</colgroup>
<tbody>
<tr>
<td style="text-align: center;"><strong>开放端口类型</strong></td>
<td style="text-align: center;"><strong>开放端口</strong></td>
<td style="text-align: center;"><strong>开放范围</strong></td>
<td style="text-align: center;"><strong>端口用途</strong></td>
</tr>
<tr>
<td rowspan="5" style="text-align: center;">TCP</td>
<td style="text-align: center;">8441</td>
<td style="text-align: center;">全网放开</td>
<td style="text-align: center;">终端接口</td>
</tr>
<tr>
<td style="text-align: center;">8442</td>
<td style="text-align: center;">全网放开</td>
<td style="text-align: center;">终端 webview</td>
</tr>
<tr>
<td style="text-align: center;">8443</td>
<td style="text-align: center;">管理员 IP</td>
<td style="text-align: center;">web 控制台</td>
</tr>
<tr>
<td style="text-align: center;">8445(可选)</td>
<td style="text-align: center;">全网放开</td>
<td style="text-align: center;">浏览器插件管控</td>
</tr>
<tr>
<td style="text-align: center;">13125</td>
<td style="text-align: center;">全网放开</td>
<td style="text-align: center;">用于终端消息通信</td>
</tr>
</tbody>
</table>

### 3、版本升级

- 注意：升级前推荐先一键熔断或选择低峰期操作，减少升级过程中终端上报的数据错乱

<!-- -->

- 如有开启 UEBA 模块，升级完需要重新开启：【扩展模块】UEBA 部署

<!-- -->

- 如有安装 OpenAPI 模块，升级完需要重新开启，账号密码不变

  执行升级会自动备份核心配置文件到/tmp/backup_XXXX 目录下，日期为执行备份操作的日期：

![](../images/server-deploy/image3.png)

  执行命令开始升级：

```bash
./qzh-installer-3-9-**.bin -p "$(< qzh-installer-3-9-*.bin.password)"
```

### 4、安装机器学习高级模块(可选)

<table style="width:88%;">
<colgroup>
<col style="width: 88%" />
</colgroup>
<tbody>
<tr>
<td style="text-align: left;"><p>模型包：nlp_20231127.tar.gz</p>
<p>文件大小：390 MB</p></td>
</tr>
</tbody>
</table>

  执行命令即可：

```bash
tar xvf nlp_20231127.tar.gz -C ${DDR_HOME}/data/conf/nlp/
```

### 5、常用运维操作

  停止所有服务

```bash
/home/data/bin/tool.sh start_system stop
```

  开启所有服务

```bash
/home/data/bin/tool.sh start_system start
```

  重启所有服务

```bash
/home/data/bin/tool.sh start_system restart
```

## 五、License 授权

  重要：

- 提交授权码前不允许重启服务器

<!-- -->

- License 会限制运行实例和终端数控制

<!-- -->

- 系统大版本升级需要重新授权

  打开谷歌浏览器，访问部署所在服务器的 ip：

  初始账号：companyadmin

  初始密码：请联系工作人员获取

  看到如下页面，下载授权码文件，联系商务同学进行授权，将商务同学发过来的 License 上传即可。

![](../images/server-deploy/image4.png)

## 六、异常排查

- 请查看硬件配置和系统版本是否达到标准，⽐如内存，硬盘，CPU，是否 CentOS7.9+等，查看局域⽹络是否异常，⽐如存在 ip 冲突，dns 配置不当，路由不当等

<!-- -->

- 检查部署包的 md5 值是否一致，如不一致则为传输过程中损坏，需要重新下载

<!-- -->

- 查看各个 docker 服务是否都启动正常：

```bash
docker ps
docker logs 容器 ID
```

<!-- -->

- 查看对应服务进程是否存在：

```bash
ps -ef | grep 进程关键字
```

<!-- -->

- 查看对应服务端口是否存在，并确保端口未被其他应用占用：

```bash
telnet 127.0.0.1 对应端口
sudo netstat -pant | grep 端口
```
