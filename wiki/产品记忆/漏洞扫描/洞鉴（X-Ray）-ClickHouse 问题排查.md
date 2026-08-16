---
type: product
title: 洞鉴（X-Ray）-ClickHouse 问题排查
product: 洞鉴（X-Ray）
aliases: []
tags:
    - 漏洞扫描
    - support
    - 故障排查
summary: ClickHouse 数据仓库问题的诊断和修复。
status: verified
---

<!-- toc -->
# ClickHouse相关问题

## 异常断电或重启导致ClickHouse无法启动


1. 10-24.06.001之前版本执行

```shell

rm  -rf  /data/x-ray/container/data_warehouse   # 或者  mv 到某个backup 目录吧
 
docker restart xray-data-warehouse
 
docker exec -it xray-web python manage.py shell
 
from server.data_warehouse import get_clickhouse_orm_database
 
from vulnerability.models_clickhouse import VulnHistoricalModel
 
from safety_asset.models_clickhouse import IPAddressHistoricalModel, WebsiteHistoricalModel
 
database = get_clickhouse_orm_database()
 
database.create_table(VulnHistoricalModel)
 
database.create_table(IPAddressHistoricalModel)
 
database.create_table(WebsiteHistoricalModel)
 
exit()
 
docker restart xray-data-warehouse
 
 
 
```

2. 24.06.001 之后执行脚本
 
```shell
# rm 或者 mv到backup目录备份下原数据
mv /data/x-ray/container/data_warehouse /data/x-ray/container/data_warehouse.bak
 
docker restart xray-data-warehouse
 
docker exec -it xray-web python manage.py shell -c "from server.management.commands.init_migrate import init_migrate_ck;init_migrate_ck()"

```

<!-- endtoc -->
