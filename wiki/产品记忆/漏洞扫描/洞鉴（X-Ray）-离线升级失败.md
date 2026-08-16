---
type: product
title: 洞鉴（X-Ray）-离线升级失败
product: 洞鉴（X-Ray）
aliases: []
tags:
    - 漏洞扫描
    - support
    - 故障排查
summary: 离线升级失败的排查和解决方法。
status: verified
---

<!-- toc -->

# 离线升级失败

---

```shell

1、release下载升级包
2、重命名
mv <package> 1.zip
 
3、复制到升级容器内
docker cp 1.zip xray-patcher-engine:/tmp
 
4、执行升级脚本
docker exec xray-patcher-engine sh -c "export TEMP=$(mktemp -d); unzip /tmp/1.zip -d \${TEMP}; cd \${TEMP}; sh ./run.sh"
 
5、重启引擎容器
./minion stop && ./minion start
```

<!-- endtoc -->
