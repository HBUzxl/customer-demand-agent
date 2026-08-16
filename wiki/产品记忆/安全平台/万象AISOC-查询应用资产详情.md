---
type: product
title: 万象AISOC-查询应用资产详情
product: 万象AISOC
aliases: []
tags:
    - 安全平台
    - 产品技术手册
    - 接口文档
    - 资产
summary: 介绍查询应用资产详情相关的接口能力、请求方式与使用说明。
status: verified
---

<!-- toc -->
<a id=查询Web资产详情2817> </a>

### 基本信息

**Path：** https://ip_addr/pedestal/rpc

**Method：** POST

**接口描述：**
<h3>请求</h3>
<pre><code>{
    "method": "AssetService.GetWebAsset",
    "params": {
        "id":34 
    },
    "jsonrpc": "2.0",
    "id": "0"
}
</code></pre>
<h3>响应</h3>
<pre><code>{
    "jsonrpc": "2.0",
    "result": {
        "kernel_attribute": {
            "id": 34,
            "name": "zxh_test",
            "find_by": "manual",
            "organization": {
                "id": 1,
                "name": "根组织机构"
            },
            "security_scope": {
                "id": 5,
                "name": "DMZ域",
                "description": "",
                "category": 1,
                "level": 0,
                "ips": [],
                "parent_id": 0,
                "asset_deploy_location_id": 1,
                "created_at": 1726624265.849453,
                "update_at": 1726624266.680677,
                "is_built_in": false,
                "created_by_id": 0,
                "is_internal": true
            },
            "business_scope": {
                "id": 2,
                "name": "业务默认域",
                "description": "",
                "category": 2,
                "level": 0,
                "ips": [],
                "parent_id": 0,
                "asset_deploy_location_id": 1,
                "created_at": 1726624265.849453,
                "update_at": 1726624266.025561,
                "is_built_in": true,
                "created_by_id": 0,
                "is_internal": true
            },
            "tags": [
                {
                    "id": 1,
                    "name": "重点资产",
                    "level": 0,
                    "category": 0
                }
            ],
            "description": "这是一个重要资产",
            "state": null,
            "categories": [
                {
                    "id": 0,
                    "name": "Web资产",
                    "version_name": "",
                    "created_by_id": 0
                }
            ],
            "external_accessible": false,
            "auto_update": false,
            "site_url": "",
            "domain": "url",
            "title": "网站标题",
            "framework": "网站指纹",
            "icp_record": "",
            "net_police_record": "",
            "related_host_assets": [
                {
                    "id": 7,
                    "name": "资产7",
                    "ip": "1.1.1.7"
                }
            ],
            "alarm_count": 0,
            "vuln_count": 0,
            "find_by_source": [
                {
                    "asset_id": 34,
                    "find_by": "manual",
                    "device_id": 0,
                    "device_name": ""
                }
            ]
        },
        "user_attribute": {
            "asset_owner": {
                "id": 1,
                "name": "黑马喽",
                "tel": "17890009999",
                "user_name": "admin"
            },
            "asset_owner_name": "zxh",
            "business_owner": {
                "id": 1,
                "name": "黑马喽",
                "tel": "17890009999",
                "user_name": "admin"
            },
            "business_owner_name": "zxh"
        },
        "time_attribute": {
            "find_at": 1729587125.547126,
            "last_find_at": 1729587125.547377,
            "last_modify_at": 1729587125.547377,
            "last_online": null,
            "last_offline": null,
            "domain_expired_at": null
        },
        "position_attribute": {
            "geo_location": "北京/北京/北京",
            "room_location": "",
            "cabinet_location": ""
        },
        "security_attribute": {
            "importance": 1,
            "availability_lv": "low",
            "confidentiality_lv": "low",
            "integrity_lv": "low",
            "value_lv": ""
        },
        "custom_common_attribute": {},
        "base_info": {
            "web_framework": "",
            "http_request_header": "",
            "http_response_header": ""
        }
    },
    "id": "0"
}
</code></pre>


### 请求参数
**Headers**

| 参数名称  | 参数值              |  是否必须 | 示例 | 备注 |
| ------------ |------------------| ------------ |----|----|
| Content-Type  | application/json | 是  | -  | -  |
| authorization | bearer token_value | 是       | -  | -  |
| x-menu-name | 59               | 是       | -  | -  |
| x-request-path | pedestal         | 是       | -  | -  |
**Body**

<table>
  <thead class="ant-table-thead">
    <tr>
      <th key=name>名称</th><th key=type>类型</th><th key=required>是否必须</th><th key=default>默认值</th><th key=desc>备注</th><th key=sub>其他信息</th>
    </tr>
  </thead><tbody className="ant-table-tbody"><tr key=0-0><td key=0><span style="padding-left: 0px"><span style="color: #8c8a8a"></span> method</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1><td key=0><span style="padding-left: 0px"><span style="color: #8c8a8a"></span> jsonrpc</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2><td key=0><span style="padding-left: 0px"><span style="color: #8c8a8a"></span> id</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">资产id</span></td><td key=5></td></tr><tr key=0-3><td key=0><span style="padding-left: 0px"><span style="color: #8c8a8a"></span> params</span></td><td key=1><span>object</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-3-0><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> id</span></td><td key=1><span>integer</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr>
               </tbody>
              </table>

### 返回数据

<table>
  <thead class="ant-table-thead">
    <tr>
      <th key=name>名称</th><th key=type>类型</th><th key=required>是否必须</th><th key=default>默认值</th><th key=desc>备注</th><th key=sub>其他信息</th>
    </tr>
  </thead><tbody className="ant-table-tbody"><tr key=0-0><td key=0><span style="padding-left: 0px"><span style="color: #8c8a8a"></span> jsonrpc</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1><td key=0><span style="padding-left: 0px"><span style="color: #8c8a8a"></span> result</span></td><td key=1><span>object</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-0><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> kernel_attribute</span></td><td key=1><span>object</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">基础属性</span></td><td key=5></td></tr><tr key=0-1-0-0><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> id</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">资产id</span></td><td key=5></td></tr><tr key=0-1-0-1><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> name</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">资产名称</span></td><td key=5></td></tr><tr key=0-1-0-2><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> organization</span></td><td key=1><span>object</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">组织机构</span></td><td key=5></td></tr><tr key=0-1-0-2-0><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> id</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-0-2-1><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> name</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-0-3><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> security_scope</span></td><td key=1><span>object</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">安全域</span></td><td key=5></td></tr><tr key=0-1-0-3-0><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> id</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-0-3-1><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> name</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-0-3-2><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> category</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-0-3-3><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> level</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-0-3-4><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> ip_type</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-0-3-5><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> ip</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-0-3-6><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> parent_id</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-0-3-7><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> asset_deploy_location_id</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-0-3-8><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> created_at</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-0-3-9><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> update_at</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-0-3-10><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> is_built_in</span></td><td key=1><span>boolean</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-0-3-11><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> created_by_id</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-0-4><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> business_scope</span></td><td key=1><span>object</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">业务域</span></td><td key=5></td></tr><tr key=0-1-0-4-0><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> id</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-0-4-1><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> name</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-0-4-2><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> category</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-0-4-3><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> level</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-0-4-4><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> ip_type</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-0-4-5><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> ip</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-0-4-6><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> parent_id</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-0-4-7><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> asset_deploy_location_id</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-0-4-8><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> created_at</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-0-4-9><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> update_at</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-0-4-10><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> is_built_in</span></td><td key=1><span>boolean</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-0-4-11><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> created_by_id</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-0-5><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> tags</span></td><td key=1><span>string []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">标签</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>string</span></p></td></tr><tr key=array-821><tr key=0-1-0-6><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> description</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">资产描述</span></td><td key=5></td></tr><tr key=0-1-0-7><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> state</span></td><td key=1><span>null</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">资产状态</span></td><td key=5></td></tr><tr key=0-1-0-8><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> categories</span></td><td key=1><span>object []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">资产类别</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-1-0-8-0><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> id</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-0-8-1><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> name</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-0-8-2><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> version_name</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-0-8-3><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> created_by_id</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-0-9><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> external_accessible</span></td><td key=1><span>boolean</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">是否从互联网访问</span></td><td key=5></td></tr><tr key=0-1-0-10><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> auto_update</span></td><td key=1><span>boolean</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">自动更新</span></td><td key=5></td></tr><tr key=0-1-0-11><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> site_url</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">应用URL</span></td><td key=5></td></tr><tr key=0-1-0-12><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> domain</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">网站域名</span></td><td key=5></td></tr><tr key=0-1-0-13><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> title</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">网站标题</span></td><td key=5></td></tr><tr key=0-1-0-14><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> framework</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">网站指纹</span></td><td key=5></td></tr><tr key=0-1-0-15><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> icp_record</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">icp记录</span></td><td key=5></td></tr><tr key=0-1-0-16><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> net_police_record</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">网上警务记录</span></td><td key=5></td></tr><tr key=0-1-0-17><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> related_host_assets</span></td><td key=1><span>string []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">关联IP资产</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>string</span></p></td></tr><tr key=array-822><tr key=0-1-0-18><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> find_by_source</span></td><td key=1><span>object []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">源</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-1-0-18-0><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> asset_id</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-0-18-1><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> find_by</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-0-18-2><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> device_id</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-0-19><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> alarm_count</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">告警数</span></td><td key=5></td></tr><tr key=0-1-0-20><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> vuln_count</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">漏洞数</span></td><td key=5></td></tr><tr key=0-1-1><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> user_attribute</span></td><td key=1><span>object</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">责任人属性</span></td><td key=5></td></tr><tr key=0-1-1-0><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> asset_owner</span></td><td key=1><span>object</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">资产责任人</span></td><td key=5></td></tr><tr key=0-1-1-0-0><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> id</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-1-0-1><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> name</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-1-0-2><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> tel</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-1-1><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> business_owner</span></td><td key=1><span>object</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">业务责任人</span></td><td key=5></td></tr><tr key=0-1-1-1-0><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> id</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-1-1-1><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> name</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-1-1-2><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> tel</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-2><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> time_attribute</span></td><td key=1><span>object</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">时间属性</span></td><td key=5></td></tr><tr key=0-1-2-0><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> find_at</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">最早发现时间</span></td><td key=5></td></tr><tr key=0-1-2-1><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> last_find_at</span></td><td key=1><span>null</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">最近发现时间</span></td><td key=5></td></tr><tr key=0-1-2-2><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> last_modify_at</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">最近更新时间</span></td><td key=5></td></tr><tr key=0-1-2-3><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> last_online</span></td><td key=1><span>null</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">最近上线时间</span></td><td key=5></td></tr><tr key=0-1-2-4><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> last_offline</span></td><td key=1><span>null</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">最近下线时间</span></td><td key=5></td></tr><tr key=0-1-3><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> security_attribute</span></td><td key=1><span>object</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-3-0><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> importance</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-3-1><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> availability_lv</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-3-2><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> confidentiality_lv</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-3-3><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> integrity_lv</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-3-4><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> value_lv</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-4><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> custom_common_attribute</span></td><td key=1><span>object</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">通用扩展属性</span></td><td key=5></td></tr><tr key=0-2><td key=0><span style="padding-left: 0px"><span style="color: #8c8a8a"></span> id</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr>
               </tbody>
              </table>

<!-- endtoc -->
