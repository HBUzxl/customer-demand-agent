---
type: product
title: 万象AISOC-ip漏洞列表查看
product: 万象AISOC
aliases: []
tags:
    - 安全平台
    - 产品技术手册
    - 接口文档
    - 漏洞
summary: 介绍ip漏洞列表查看相关的接口能力、请求方式与使用说明。
status: verified
---

<!-- toc -->


### 基本信息

**Path：** https://ip_addr/pedestal/rpc

**Method：** POST

**接口描述：**
<h3>请求参数</h3>
<pre><code>{
    "method": "ScanVulnIpService.SearchScanVulnIpList",
    "jsonrpc": "2.0",
    "id": "0",
    "params": {
        "count": 20,
        "offset": 0,
        "id": [
            {
                "oper": "=",
                "target": 1
            }
        ],
        "organization_id": [
            {
                "oper": "=",
                "target": 1
            }
        ],
        "name": [
            {
                "oper": "like",
                "target": "d"
            }
        ],
        "port": [
            {
                "oper": "=",
                "target": 80
            }
        ],
        "service": [
            {
                "oper": "=",
                "target": "dd"
            }
        ],
        "vuln_category": [
            {
                "oper": "=",
                "target": "Configuration error"
            }
        ],
        "vuln_level": [
            {
                "oper": "=",
                "target": 4
            },
            {
                "oper": "=",
                "target": 3
            }
        ],
        "vuln_ip": [
            {
                "oper": "like",
                "target": "dd"
            }
        ],
        "merge_num": [
            {
                "oper": "\u003e",
                "target": 1
            }
        ],
        "updated_at": [
            {
                "oper": "in",
                "target": "1741104000-1744300800"
            }
        ],
        "created_at": [
            {
                "oper": "in",
                "target": "1741881600-1745337600"
            }
        ],
        "vuln_update_time": [
            {
                "oper": "in",
                "target": "1741190400-1744128000"
            }
        ],
        "vuln_dispose_type": null,
        "vuln_status": [
            {
                "oper": "=",
                "target": 1
            }
        ],
        "is_match_vuln": [
            {
                "oper": "=",
                "target": true
            }
        ],
        "is_match_asset": [
            {
                "oper": "=",
                "target": true
            }
        ],
        "find_by": null,
        "user": [
            {
                "oper": "like",
                "target": "kkk"
            }
        ],
        "password": [
            {
                "oper": "like",
                "target": "pass"
            }
        ],
        "vuln_data_type": [
            {
                "oper": "=",
                "target": 1
            }
        ],
        "condition_query": null
    }
}
</code></pre>
<h3>响应</h3>
<pre><code>{
    "jsonrpc": "2.0",
    "result": {
        "total": 1,
        "data": [
            {
                "id": 6,
                "organization_id": 1,
                "name": "弱口令测试001",
                "vuln_major_category": "weak password",
                "vuln_category": "weak_password_weak_password",
                "vuln_level": 4,
                "vuln_ip": "1.1.1.1",
                "vuln_ip_count": 0,
                "port": 0,
                "service": "",
                "find_by": [
                    "手动提交"
                ],
                "merge_num": 1,
                "created_at": 1736840029.318785,
                "updated_at": 1736840029.326120,
                "vuln_tag": [],
                "cve_id": "",
                "cnnvd": "",
                "cvss": 0,
                "release_by": [],
                "vuln_status": 1,
                "vuln_dispose_type": 0,
                "dispose_by": [],
                "is_match_vuln": false,
                "vulndb_ids": [],
                "vuln_update_time": 1736840029.319105,
                "had_send_email": false,
                "vuln_belong_os": "",
                "vuln_oa_send": "",
                "vlun_asset": "",
                "vuln_belong_office": "",
                "vuln_belong_ownership": "",
                "user": "",
                "password": "",
                "vuln_data_type": 1,
                "extra_field": {}
            }
        ]
    },
    "id": "0"
}
</code></pre>


### 请求参数
**Headers**

| 参数名称  | 参数值  |  是否必须 | 示例 | 备注 |
| ------------ | ------------ | ------------ |----|----|
| Content-Type  |  application/json | 是  | -  | -  |
| authorization | bearer token_value | 是       | -  | -  |
| x-menu-name | 6201               | 是       | -  | -  |
| x-request-path | pedestal       | 是       | -  | -  |
**Body**

<table>
  <thead class="ant-table-thead">
    <tr>
      <th key=name>名称</th><th key=type>类型</th><th key=required>是否必须</th><th key=default>默认值</th><th key=desc>备注</th><th key=sub>其他信息</th>
    </tr>
  </thead><tbody className="ant-table-tbody"><tr key=0-0><td key=0><span style="padding-left: 0px"><span style="color: #8c8a8a"></span> method</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1><td key=0><span style="padding-left: 0px"><span style="color: #8c8a8a"></span> jsonrpc</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2><td key=0><span style="padding-left: 0px"><span style="color: #8c8a8a"></span> id</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-3><td key=0><span style="padding-left: 0px"><span style="color: #8c8a8a"></span> params</span></td><td key=1><span>object</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-3-0><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> count</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">当前页条数</span></td><td key=5></td></tr><tr key=0-3-1><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> offset</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">页码</span></td><td key=5></td></tr><tr key=0-3-2><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> id</span></td><td key=1><span>object []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">漏洞id筛选</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-3-2-0><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> oper</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-3-2-1><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> target</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-3-3><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> organization_id</span></td><td key=1><span>object []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">组织机构id筛选</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-3-3-0><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> oper</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-3-3-1><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> target</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-3-4><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> name</span></td><td key=1><span>object []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">漏洞名称筛选</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-3-4-0><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> oper</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-3-4-1><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> target</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-3-5><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> port</span></td><td key=1><span>object []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">漏洞端口筛选</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-3-5-0><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> oper</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-3-5-1><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> target</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-3-6><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> service</span></td><td key=1><span>object []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">服务筛选</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-3-6-0><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> oper</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-3-6-1><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> target</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-3-7><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> vuln_category</span></td><td key=1><span>object []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">漏洞类型筛选</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-3-7-0><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> oper</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-3-7-1><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> target</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-3-8><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> vuln_level</span></td><td key=1><span>object []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">漏洞等级筛选</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-3-8-0><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> oper</span></td><td key=1><span>string</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-3-8-1><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> target</span></td><td key=1><span>number</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-3-9><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> vuln_ip</span></td><td key=1><span>object []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">影响资产(漏洞ip)筛选</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-3-9-0><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> oper</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-3-9-1><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> target</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-3-10><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> merge_num</span></td><td key=1><span>object []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">漏洞合并筛选</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-3-10-0><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> oper</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-3-10-1><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> target</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-3-11><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> updated_at</span></td><td key=1><span>object []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">最近更新时间筛选</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-3-11-0><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> oper</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-3-11-1><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> target</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-3-12><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> created_at</span></td><td key=1><span>object []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">首次发现时间筛选</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-3-12-0><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> oper</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-3-12-1><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> target</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-3-13><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> vuln_update_time</span></td><td key=1><span>object []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">最近发现时间筛选</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-3-13-0><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> oper</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-3-13-1><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> target</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-3-14><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> vuln_dispose_type</span></td><td key=1><span>object []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">漏洞处置方式</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-3-14-0><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> oper</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-3-14-1><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> target</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-3-15><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> vuln_status</span></td><td key=1><span>object []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">漏洞状态筛选</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-3-15-0><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> oper</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-3-15-1><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> target</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-3-16><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> is_match_vuln</span></td><td key=1><span>object []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">是否命中情报筛选</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-3-16-0><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> oper</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-3-16-1><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> target</span></td><td key=1><span>boolean</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-3-17><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> is_match_asset</span></td><td key=1><span>object []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">是否匹配资产筛选</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-3-17-0><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> oper</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-3-17-1><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> target</span></td><td key=1><span>boolean</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-3-18><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> find_by</span></td><td key=1><span>object []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">漏洞来源筛选</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-3-18-0><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> oper</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-3-18-1><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> target</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-3-19><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> user</span></td><td key=1><span>object []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">用户筛选</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-3-19-0><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> oper</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-3-19-1><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> target</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-3-20><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> password</span></td><td key=1><span>object []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">密码筛选</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-3-20-0><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> oper</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-3-20-1><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> target</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-3-21><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> vuln_data_type</span></td><td key=1><span>object []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">漏洞数据分类筛选</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-3-21-0><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> oper</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-3-21-1><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> target</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-3-22><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> condition_query</span></td><td key=1><span>null</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr>
               </tbody>
              </table>

### 返回数据

<table>
  <thead class="ant-table-thead">
    <tr>
      <th key=name>名称</th><th key=type>类型</th><th key=required>是否必须</th><th key=default>默认值</th><th key=desc>备注</th><th key=sub>其他信息</th>
    </tr>
  </thead><tbody className="ant-table-tbody"><tr key=0-0><td key=0><span style="padding-left: 0px"><span style="color: #8c8a8a"></span> jsonrpc</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1><td key=0><span style="padding-left: 0px"><span style="color: #8c8a8a"></span> result</span></td><td key=1><span>object</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-0><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> total</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-1><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> data</span></td><td key=1><span>object []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-1-1-0><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> id</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">漏洞id</span></td><td key=5></td></tr><tr key=0-1-1-1><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> organization_id</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">组织机构id</span></td><td key=5></td></tr><tr key=0-1-1-2><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> name</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">漏洞名称</span></td><td key=5></td></tr><tr key=0-1-1-3><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> vuln_major_category</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">漏洞类型大类</span></td><td key=5></td></tr><tr key=0-1-1-4><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> vuln_category</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">漏洞类型</span></td><td key=5></td></tr><tr key=0-1-1-5><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> vuln_level</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">漏洞等级</span></td><td key=5></td></tr><tr key=0-1-1-6><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> vuln_ip</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">影响资产</span></td><td key=5></td></tr><tr key=0-1-1-7><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> vuln_ip_count</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">影响资产个数</span></td><td key=5></td></tr><tr key=0-1-1-8><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> port</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">端口</span></td><td key=5></td></tr><tr key=0-1-1-9><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> service</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">服务</span></td><td key=5></td></tr><tr key=0-1-1-10><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> find_by</span></td><td key=1><span>string []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">漏洞来源</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>string</span></p></td></tr><tr key=array-38><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> </span></td><td key=1><span></span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-1-11><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> merge_num</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">合并漏洞数量</span></td><td key=5></td></tr><tr key=0-1-1-12><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> created_at</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">漏洞首次发现时间</span></td><td key=5></td></tr><tr key=0-1-1-13><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> updated_at</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">漏洞最近更新时间</span></td><td key=5></td></tr><tr key=0-1-1-14><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> vuln_tag</span></td><td key=1><span>string []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">标签</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>string</span></p></td></tr><tr key=array-39><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> </span></td><td key=1><span></span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-1-15><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> cve_id</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">漏洞编号信息</span></td><td key=5></td></tr><tr key=0-1-1-16><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> cnnvd</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">cnnvd</span></td><td key=5></td></tr><tr key=0-1-1-17><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> cvss</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">cvss评分</span></td><td key=5></td></tr><tr key=0-1-1-18><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> release_by</span></td><td key=1><span>string []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">下发人</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>string</span></p></td></tr><tr key=array-40><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> </span></td><td key=1><span></span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-1-19><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> vuln_status</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">漏洞状态</span></td><td key=5></td></tr><tr key=0-1-1-20><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> vuln_dispose_type</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">漏洞处置方式</span></td><td key=5></td></tr><tr key=0-1-1-21><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> dispose_by</span></td><td key=1><span>string []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">处置人</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>string</span></p></td></tr><tr key=array-41><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> </span></td><td key=1><span></span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-1-22><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> is_match_vuln</span></td><td key=1><span>boolean</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">是否命中情报</span></td><td key=5></td></tr><tr key=0-1-1-23><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> vulndb_ids</span></td><td key=1><span>string []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">命中的漏洞库 id</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>string</span></p></td></tr><tr key=array-42><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> </span></td><td key=1><span></span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-1-24><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> vuln_update_time</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">漏洞最近发现时间</span></td><td key=5></td></tr><tr key=0-1-1-25><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> had_send_email</span></td><td key=1><span>boolean</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">邮件通知</span></td><td key=5></td></tr><tr key=0-1-1-26><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> vuln_belong_os</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">漏洞影响系统</span></td><td key=5></td></tr><tr key=0-1-1-27><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> vuln_oa_send</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">OA通知</span></td><td key=5></td></tr><tr key=0-1-1-28><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> vlun_asset</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">资产情况</span></td><td key=5></td></tr><tr key=0-1-1-29><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> vuln_belong_office</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">所属组织机构</span></td><td key=5></td></tr><tr key=0-1-1-30><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> vuln_belong_ownership</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">所属父级组织机构</span></td><td key=5></td></tr><tr key=0-1-1-31><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> user</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">用户</span></td><td key=5></td></tr><tr key=0-1-1-32><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> password</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">密码</span></td><td key=5></td></tr><tr key=0-1-1-33><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> vuln_data_type</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">漏洞数据类型</span></td><td key=5></td></tr><tr key=0-1-1-34><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> extra_field</span></td><td key=1><span>object</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">自定义字段</span></td><td key=5></td></tr><tr key=0-2><td key=0><span style="padding-left: 0px"><span style="color: #8c8a8a"></span> id</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr>
               </tbody>
              </table>

<!-- endtoc -->
