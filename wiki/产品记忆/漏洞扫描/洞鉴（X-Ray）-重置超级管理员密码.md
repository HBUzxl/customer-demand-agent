---
type: product
title: 洞鉴（X-Ray）-重置超级管理员密码
product: 洞鉴（X-Ray）
aliases: []
tags:
    - 漏洞扫描
    - support
    - 故障排查
summary: 忘记超级管理员密码后的密码重置方法。
status: verified
---

<!-- toc -->

# 超级管理员密码忘了，修改账户密码

---

```shell

docker exec -i xray-web python manage.py changepassword admin

```

<!-- endtoc -->
