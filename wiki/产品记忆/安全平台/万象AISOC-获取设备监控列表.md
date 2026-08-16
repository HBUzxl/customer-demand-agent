---
type: product
title: 万象AISOC-获取设备监控列表
product: 万象AISOC
aliases: []
tags:
    - 安全平台
    - 产品技术手册
    - 接口文档
    - 设备监控
summary: 介绍获取设备监控列表相关的接口能力、请求方式与使用说明。
status: verified
---

<!-- toc -->

<a id=获取设备监控列表2979> </a>

### 基本信息

**Path：** https://ip_addr/pedestal/rpc

**Method：** POST

**接口描述：**

<h3>请求</h3>
<pre><code>{
    "method": "AgentService.GetDeviceMonitoringList",
    "params": {
        "count": 20,
        "offset": 0,
        "device_filter": {
            "id": null,
            "ip": [
                {
                    "oper": "like",
                    "target": "127.0.0.1"
                }
            ],
            "name": [
                {
                    "oper": "like",
                    "target": "127.0.0.1"
                }
            ],
            "device_type": [
                {
                    "oper": "=",
                    "target": 1001
                }
            ],
            "product_name": null,
            "security_scope_id": null,
            "last_receive_time": null,
            "vendor": null,
            "is_monitoring": null,
            "status": null,
            "owner_id": null,
            "organization_ids": null
        },
        "monitoring_filter": {
            "device_is_up": 2,
            "interface_is_up": 0,
            "cpu_is_overload": 0,
            "memory_is_overload": 0,
            "disk_is_overload": 0,
            "license_is_expired": 0
        }
    },
    "jsonrpc": "2.0",
    "id": "0"
}
</code></pre>
<h3>响应</h3>
<pre><code>{
    "jsonrpc": "2.0",
    "result": {
        "total": 1,
        "data": [
            {
                "id": 2,
                "is_up": false,
                "interfaces": [],
                "engines": [],
                "disks": [],
                "device_info": {
                    "name": "127.0.0.1",
                    "ip": "127.0.0.1",
                    "device_type": 1001,
                    "device_group": 1,
                    "security_scope_id": 5,
                    "security_scope": [
                        {
                            "id": 5,
                            "name": "DMZ域"
                        }
                    ],
                    "owner": {
                        "id": 1,
                        "name": "黑马喽",
                        "username": "admin"
                    },
                    "action": [],
                    "ssh": {
                        "user": "",
                        "password": "",
                        "private_key": "",
                        "port": 0,
                        "login_type": 0
                    },
                    "status": 0,
                    "normalize_rules": [],
                    "organizations": [
                        {
                            "ID": 1,
                            "CreatedAt": "2024-09-18T01:51:05.206776Z",
                            "UpdatedAt": "2024-10-16T08:21:13.146926Z",
                            "DeletedAt": null,
                            "name": "根组织机构",
                            "category": 0,
                            "industry": 0,
                            "principal": [],
                            "tags": [],
                            "location": {
                                "ID": 0,
                                "CreatedAt": "0001-01-01T00:00:00Z",
                                "UpdatedAt": "0001-01-01T00:00:00Z",
                                "DeletedAt": null,
                                "OrganizationID": 0,
                                "ProvinceID": 0,
                                "Province": {
                                    "ID": 0,
                                    "CreatedAt": "0001-01-01T00:00:00Z",
                                    "UpdatedAt": "0001-01-01T00:00:00Z",
                                    "DeletedAt": null,
                                    "Name": "",
                                    "Level": 0,
                                    "ParentID": 0,
                                    "FullName": "",
                                    "CountryLevelID": 0,
                                    "FullID": 0,
                                    "Geo": null
                                },
                                "CityID": 0,
                                "City": {
                                    "ID": 0,
                                    "CreatedAt": "0001-01-01T00:00:00Z",
                                    "UpdatedAt": "0001-01-01T00:00:00Z",
                                    "DeletedAt": null,
                                    "Name": "",
                                    "Level": 0,
                                    "ParentID": 0,
                                    "FullName": "",
                                    "CountryLevelID": 0,
                                    "FullID": 0,
                                    "Geo": null
                                },
                                "CountryID": 0,
                                "Country": {
                                    "ID": 0,
                                    "CreatedAt": "0001-01-01T00:00:00Z",
                                    "UpdatedAt": "0001-01-01T00:00:00Z",
                                    "DeletedAt": null,
                                    "Name": "",
                                    "Level": 0,
                                    "ParentID": 0,
                                    "FullName": "",
                                    "CountryLevelID": 0,
                                    "FullID": 0,
                                    "Geo": null
                                },
                                "StreetID": 0,
                                "Street": {
                                    "ID": 0,
                                    "CreatedAt": "0001-01-01T00:00:00Z",
                                    "UpdatedAt": "0001-01-01T00:00:00Z",
                                    "DeletedAt": null,
                                    "Name": "",
                                    "Level": 0,
                                    "ParentID": 0,
                                    "FullName": "",
                                    "CountryLevelID": 0,
                                    "FullID": 0,
                                    "Geo": null
                                }
                            },
                            "Geographic": null,
                            "superior_organization_id": 0,
                            "superior_organization": null,
                            "last_edit_by_id": 0,
                            "LastEditBy": null,
                            "Users": [],
                            "is_system": true,
                            "is_enabled": true
                        }
                    ],
                    "created_by_id": 0,
                    "decode_type": 0
                },
                "disk_usage_percentage": 0,
                "last_receive_time": null,
                "full_connection": 0,
                "half_connection": 0,
                "current_connection": 0
            }
        ]
    },
    "id": "0"
}
</code></pre>

### 请求参数

**Headers**

| 参数名称     | 参数值              | 是否必须 | 示例 | 备注 |
| ------------ |------------------| -------- | ---- | ---- |
| Content-Type | application/json | 是       |   -   |   -   |
| authorization | bearer token_value | 是       | -  | -  |
| x-menu-name | 4802             | 是       | -  | -  |
| x-request-path | pedestal         | 是       | -  | -  |

**Body**

<table>
  <thead class="ant-table-thead">
    <tr>
      <th key=name>名称</th><th key=type>类型</th><th key=required>是否必须</th><th key=default>默认值</th><th key=desc>备注</th><th key=sub>其他信息</th>
    </tr>
  </thead><tbody className="ant-table-tbody"><tr key=0-0><td key=0><span style="padding-left: 0px"><span style="color: #8c8a8a"></span> method</span></td><td key=1><span>string</span></td><td key=2>必须</td><td key=3>AgentService.SearchDeviceList</td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5><p key=2><span style="font-weight: '700'">枚举: </span><span>AgentService.CreateDevice</span></p></td></tr><tr key=0-1><td key=0><span style="padding-left: 0px"><span style="color: #8c8a8a"></span> params</span></td><td key=1><span>object</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-0><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> count</span></td><td key=1><span>integer</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-1><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> offset</span></td><td key=1><span>integer</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-2><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> device_filter</span></td><td key=1><span>object</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">设备筛选</span></td><td key=5></td></tr><tr key=0-1-2-0><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> id</span></td><td key=1><span>object []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">设备id搜索</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-1-2-0-0><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> oper</span></td><td key=1><span>string</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-2-0-1><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> target</span></td><td key=1><span>integer</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-2-1><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> ip</span></td><td key=1><span>object []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">设备ip搜索</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-1-2-1-0><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> oper</span></td><td key=1><span>string</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-2-1-1><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> target</span></td><td key=1><span>string</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-2-2><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> name</span></td><td key=1><span>object []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">设备名称搜索</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-1-2-2-0><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> oper</span></td><td key=1><span>string</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-2-2-1><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> target</span></td><td key=1><span>string</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-2-3><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> device_type</span></td><td key=1><span>object []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">设备类型搜索</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-1-2-3-0><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> oper</span></td><td key=1><span>string</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-2-3-1><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> target</span></td><td key=1><span>integer</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-2-4><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> product_name</span></td><td key=1><span>object []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">产品名称搜索</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-1-2-4-0><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> oper</span></td><td key=1><span>string</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-2-4-1><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> target</span></td><td key=1><span>string</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-2-5><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> organization_ids</span></td><td key=1><span>integer []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">关联单位ids搜索</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>integer</span></p></td></tr><tr key=array-70><tr key=0-1-2-6><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> security_scope_id</span></td><td key=1><span>object []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">安全域id搜索</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-1-2-6-0><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> oper</span></td><td key=1><span>string</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-2-6-1><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> target</span></td><td key=1><span>integer</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-2-7><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> last_receive_time</span></td><td key=1><span>object []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">最后接收时间搜索</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-1-2-7-0><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> oper</span></td><td key=1><span>string</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-2-7-1><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> target</span></td><td key=1><span>string</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-2-8><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> is_monitoring</span></td><td key=1><span>object []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">是否开启监控搜索</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-1-2-8-0><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> oper</span></td><td key=1><span>string</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-2-8-1><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> target</span></td><td key=1><span>boolean</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-2-9><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> owner_id</span></td><td key=1><span>object []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">责任人id搜索</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-1-2-9-0><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> oper</span></td><td key=1><span>string</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-2-9-1><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> target</span></td><td key=1><span>integer</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-2-10><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> vendor</span></td><td key=1><span>object []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">厂商名搜索</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-1-2-10-0><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> oper</span></td><td key=1><span>string</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-2-10-1><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> target</span></td><td key=1><span>string</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-3><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> monitoring_filter</span></td><td key=1><span>object</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">监控筛选</span></td><td key=5></td></tr><tr key=0-1-3-0><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> device_is_up</span></td><td key=1><span>integer</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-3-1><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> interface_is_up</span></td><td key=1><span>integer</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-3-2><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> cpu_is_overload</span></td><td key=1><span>integer</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-3-3><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> memory_is_overload</span></td><td key=1><span>integer</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-3-4><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> disk_is_overload</span></td><td key=1><span>integer</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-3-5><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> license_is_expired</span></td><td key=1><span>integer</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2><td key=0><span style="padding-left: 0px"><span style="color: #8c8a8a"></span> jsonrpc</span></td><td key=1><span>string</span></td><td key=2>必须</td><td key=3>2.0</td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5><p key=2><span style="font-weight: '700'">枚举: </span><span>2.0</span></p></td></tr><tr key=0-3><td key=0><span style="padding-left: 0px"><span style="color: #8c8a8a"></span> id</span></td><td key=1><span>string</span></td><td key=2>必须</td><td key=3>0</td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5><p key=2><span style="font-weight: '700'">枚举: </span><span>0</span></p></td></tr>
               </tbody>
              </table>

### 返回数据

<table>
  <thead class="ant-table-thead">
    <tr>
      <th key=name>名称</th><th key=type>类型</th><th key=required>是否必须</th><th key=default>默认值</th><th key=desc>备注</th><th key=sub>其他信息</th>
    </tr>
  </thead><tbody className="ant-table-tbody"><tr key=0-0><td key=0><span style="padding-left: 0px"><span style="color: #8c8a8a"></span> jsonrpc</span></td><td key=1><span>string</span></td><td key=2>必须</td><td key=3>2.0</td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5><p key=2><span style="font-weight: '700'">枚举: </span><span>2.0</span></p></td></tr><tr key=0-1><td key=0><span style="padding-left: 0px"><span style="color: #8c8a8a"></span> result</span></td><td key=1><span>object</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-0><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> total</span></td><td key=1><span>number</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-1><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> data</span></td><td key=1><span>object []</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-1-1-0><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> id</span></td><td key=1><span>integer</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-1-1><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> is_up</span></td><td key=1><span>boolean</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">在线状态</span></td><td key=5></td></tr><tr key=0-1-1-2><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> cpu_usage_percentage</span></td><td key=1><span>integer</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">cpu使用百分比</span></td><td key=5></td></tr><tr key=0-1-1-3><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> memory_usage_percentage</span></td><td key=1><span>integer</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">内存使用百分比</span></td><td key=5></td></tr><tr key=0-1-1-4><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> memory_total</span></td><td key=1><span>integer</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">内存总量</span></td><td key=5></td></tr><tr key=0-1-1-5><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> memory_usage</span></td><td key=1><span>integer</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">内存使用</span></td><td key=5></td></tr><tr key=0-1-1-6><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> interfaces</span></td><td key=1><span>object []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">接口信息</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-1-1-6-0><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> id</span></td><td key=1><span>integer</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-1-6-1><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> name</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-1-6-2><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> nickname</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-1-6-3><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> mac</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-1-6-4><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> is_up</span></td><td key=1><span>boolean</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-1-6-5><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> rated_speed</span></td><td key=1><span>integer</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-1-6-6><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> traffic_speed</span></td><td key=1><span>integer</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-1-6-7><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> io_receive</span></td><td key=1><span>integer</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-1-6-8><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> io_transmit</span></td><td key=1><span>integer</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-1-6-9><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> is_show</span></td><td key=1><span>boolean</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-1-7><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> engines</span></td><td key=1><span>object []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">引擎信息</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-1-1-7-0><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> id</span></td><td key=1><span>integer</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-1-7-1><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> name</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-1-7-2><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> is_up</span></td><td key=1><span>boolean</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-1-7-3><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> speed</span></td><td key=1><span>integer</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-1-7-4><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> load_status</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">float</span></td><td key=5></td></tr><tr key=0-1-1-8><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> disks</span></td><td key=1><span>object []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">磁盘信息</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-1-1-8-0><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> id</span></td><td key=1><span>integer</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-1-8-1><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> name</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-1-8-2><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> total</span></td><td key=1><span>integer</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-1-8-3><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> usage</span></td><td key=1><span>integer</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-1-8-4><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> io_receive</span></td><td key=1><span>integer</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-1-8-5><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> io_transmit</span></td><td key=1><span>integer</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-1-8-6><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> usage_percentage</span></td><td key=1><span>integer</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-1-9><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> device_info</span></td><td key=1><span>object</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">设备信息</span></td><td key=5></td></tr><tr key=0-1-1-9-0><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> id</span></td><td key=1><span>number</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">设备ID</span></td><td key=5></td></tr><tr key=0-1-1-9-1><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> name</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">设备名称</span></td><td key=5></td></tr><tr key=0-1-1-9-2><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> ip</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">设备IP</span></td><td key=5></td></tr><tr key=0-1-1-9-3><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> device_type</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">设备类型</span></td><td key=5></td></tr><tr key=0-1-1-9-4><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> device_group</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">设备分组</span></td><td key=5></td></tr><tr key=0-1-1-9-5><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> security_scope_id</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">安全域ID</span></td><td key=5></td></tr><tr key=0-1-1-9-6><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> license_start_time</span></td><td key=1><span>integer</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">license开始时间</span></td><td key=5></td></tr><tr key=0-1-1-9-7><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> license_end_time</span></td><td key=1><span>integer</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">license结束时间</span></td><td key=5></td></tr><tr key=0-1-1-9-8><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> security_scope</span></td><td key=1><span>object []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">相关安全域</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-1-1-9-8-0><td key=0><span style="padding-left: 80px"><span style="color: #8c8a8a">├─</span> id</span></td><td key=1><span>integer</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-1-9-8-1><td key=0><span style="padding-left: 80px"><span style="color: #8c8a8a">├─</span> name</span></td><td key=1><span>string</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-1-9-8-2><td key=0><span style="padding-left: 80px"><span style="color: #8c8a8a">├─</span> superior_security_scope_id</span></td><td key=1><span>integer</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-1-9-8-3><td key=0><span style="padding-left: 80px"><span style="color: #8c8a8a">├─</span> is_built_in</span></td><td key=1><span>boolean</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-1-9-9><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> organizations</span></td><td key=1><span>object []</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">相关单位</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-1-1-9-9-0><td key=0><span style="padding-left: 80px"><span style="color: #8c8a8a">├─</span> id</span></td><td key=1><span>number</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-1-9-9-1><td key=0><span style="padding-left: 80px"><span style="color: #8c8a8a">├─</span> name</span></td><td key=1><span>string</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-1-9-9-2><td key=0><span style="padding-left: 80px"><span style="color: #8c8a8a">├─</span> superior_organization_id</span></td><td key=1><span>integer</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-1-9-9-3><td key=0><span style="padding-left: 80px"><span style="color: #8c8a8a">├─</span> read_mode</span></td><td key=1><span>number</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">1:可读；2: 可编辑</span></td><td key=5></td></tr><tr key=0-1-1-9-10><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> owner</span></td><td key=1><span>object</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">责任人</span></td><td key=5></td></tr><tr key=0-1-1-9-10-0><td key=0><span style="padding-left: 80px"><span style="color: #8c8a8a">├─</span> id</span></td><td key=1><span>number</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-1-9-10-1><td key=0><span style="padding-left: 80px"><span style="color: #8c8a8a">├─</span> name</span></td><td key=1><span>string</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-1-9-10-2><td key=0><span style="padding-left: 80px"><span style="color: #8c8a8a">├─</span> username</span></td><td key=1><span>string</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-1-9-10-3><td key=0><span style="padding-left: 80px"><span style="color: #8c8a8a">├─</span> read_mode</span></td><td key=1><span>number</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">1:可编辑；2: 不可编辑</span></td><td key=5></td></tr><tr key=0-1-1-10><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> disk_usage_percentage</span></td><td key=1><span>integer</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">磁盘使用百分比</span></td><td key=5></td></tr><tr key=0-1-1-11><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> last_receive_time</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">最后一次接收时间</span></td><td key=5></td></tr><tr key=0-1-1-12><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> alarm_threshold</span></td><td key=1><span>object</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-1-12-0><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> cpu</span></td><td key=1><span>integer</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-1-12-1><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> memory</span></td><td key=1><span>integer</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-1-12-2><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> disk</span></td><td key=1><span>integer</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-1-13><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> today_parse_count</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">今日解析日志条数</span></td><td key=5></td></tr><tr key=0-1-1-14><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> today_non_standard_count</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">今日非标日志条数</span></td><td key=5></td></tr><tr key=0-2><td key=0><span style="padding-left: 0px"><span style="color: #8c8a8a"></span> id</span></td><td key=1><span>string</span></td><td key=2>必须</td><td key=3>0</td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5><p key=2><span style="font-weight: '700'">枚举: </span><span>0</span></p></td></tr>
               </tbody>
              </table>

<!-- endtoc -->
