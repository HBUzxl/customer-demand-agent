---
type: product
title: 洞鉴（X-Ray）-字典 Web 页面报错
product: 洞鉴（X-Ray）
aliases: []
tags:
    - 漏洞扫描
    - support
    - 故障排查
summary: 字典 Web 页面提示找不到路径的修复方法。
status: verified
---

<!-- toc -->

# 字典web页面报错，提示找不到路径

---

```shell
docker exec -i xray-web python manage.py fix_dict_path
```

```
docker exec -it xray-web python manage.py shell
from task_control.models import ScannerDict
sds = ScannerDict.objects.all()
sdf = ScannerDict.objects.first()
for sd in sds:
    try:
        print(sd.dict_file.row_num)
    except:
        sd.dict_file.path = sdf.dict_file.path
        sd.dict_file.save()
```
<!-- endtoc -->
