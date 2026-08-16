---
type: product
title: 洞鉴（X-Ray）-PostgreSQL 问题排查
product: 洞鉴（X-Ray）
aliases: []
tags:
    - 漏洞扫描
    - support
    - 故障排查
summary: PostgreSQL 数据库常见问题的诊断和修复步骤。
status: verified
---

<!-- toc -->

# Postgresql相关问题


## 突然断电xray-db损坏


```shell

1、停止Postgresql数据库容器
docker stop xray-db
 
2、启动修复容器(新旧洞鉴镜像不同)
 
# 旧版本洞鉴，数据库在data_13文件夹
docker run --rm -it -v /data/x-ray/container/data_13:/var/lib/postgresql/data  portus.in.chaitin.net/x-ray-public/postgres:latest bash
 
# 新版本洞鉴，数据库在data_15文件夹
docker run --rm -it -v /data/x-ray/container/data_15:/var/lib/postgresql/data  portus.in.chaitin.net/x-ray-public/postgres-mgmt:latest bash
 
3、修复容器内执行切换用户
su postgres
 
4、修复容器内执行修复命令
pg_resetwal -f /var/lib/postgresql/data
 
5、退出修复容器
exit
 
6、重新启动数据库
docker start xray-db
```

---

## Postgresql 13升级Postgresql 15,出现database user "balisong" is not the installer user


```shell
# 启动 pg 容器
docker run --rm --name pg -v /data/x-ray/container/data_13:/var/lib/postgresql/data portus.in.chaitin.net/x-ray-public/postgres:13
 
# 新开一shell执行下面的命令
# 进入数据库
docker exec -it pg psql -U postgres
 
# 修改用户名称
update pg_authid set rolname ='balisong2' where rolname = 'balisong';
update pg_authid set rolname ='balisong' where rolname = 'postgres';
 
# 重设密码
set password_encryption='md5';
\password
（此时会要求输入新的密码，输入两次 balisong 即可）
 
# 推出数据库
\q
 
# 停止 pg 容器
docker stop pg
 
# 删除升级失败残留的data_15文件
cd /data/x-ray/container/
rm -rf data_15
 
# 重新升级
./xxxxxx.bin
```
<!-- endtoc -->
