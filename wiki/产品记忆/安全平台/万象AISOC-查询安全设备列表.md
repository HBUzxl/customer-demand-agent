---
type: product
title: 万象AISOC-查询安全设备列表
product: 万象AISOC
aliases: []
tags:
    - 安全平台
    - 产品技术手册
    - 接口文档
    - 安全设备
summary: 介绍查询安全设备列表相关的接口能力、请求方式与使用说明。
status: verified
---

<!-- toc -->

### 基本信息

**Path：** https://ip_addr/pedestal/rpc

**Method：** POST

**接口描述：**

<h3>请求</h3>
<pre><code>{
    "method": "AgentService.SearchDeviceList",
    "params": {
        "count": 20,
        "offset": 0,
        "id": [],
        "ip": [
            {
                "oper": "=",
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
        "product_name": [
            {
                "oper": "like",
                "target": "主机产品"
            }
        ],
        "security_scope_id": [
            {
                "oper": "=",
                "target": 5
            }
        ],
        "last_receive_time": [
            {
                "oper": "in",
                "target": "1729008000-1729180800"
            }
        ],
        "is_monitoring": [
            {
                "oper": "=",
                "target": false
            }
        ],
        "status": [
            {
                "oper": "=",
                "target": 1
            }
        ],
        "owner_id": null,
        "organization_ids": null
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
                "name": "127.0.0.1",
                "ip": "127.0.0.1",
                "device_type": 1001,
                "vendor": "127.0.0.1",
                "product_name": "主机产品",
                "organization_id": 0,
                "last_receive_time": 1729148151.786000,
                "device_group": 1,
                "update_err": null,
                "security_scope_id": 5,
                "license_start_time": null,
                "license_end_time": null,
                "is_monitoring": false,
                "owner_id": 1,
                "security_scope": [
                    {
                        "id": 5,
                        "name": "DMZ域",
                        "is_built_in": false
                    }
                ],
                "owner": {
                    "id": 1,
                    "name": "黑马喽",
                    "username": "admin",
                    "read_mode": 2
                },
                "action": [
                    {
                        "action_id": 1,
                        "name": "重启",
                        "password_verify": true,
                        "allowed": false
                    },
                    {
                        "action_id": 2,
                        "name": "关机",
                        "password_verify": true,
                        "allowed": false
                    }
                ],
                "ssh": {
                    "user": "",
                    "password": "",
                    "private_key": "",
                    "port": 0,
                    "login_type": 0
                },
                "status": 1,
                "normalize_rules": [
                    {
                        "id": 101,
                        "name": "oem",
                        "display_name": "oem",
                        "read_mode": 2
                    },
                    {
                        "id": 102,
                        "name": "shujushangbao",
                        "display_name": "数据上报shujushangbao",
                        "read_mode": 2
                    },
                    {
                        "id": 188,
                        "name": "safelineAttackLogJSON",
                        "display_name": "长亭-雷池-攻击检测日志-JSON-22.01.001",
                        "read_mode": 2
                    },
                    {
                        "id": 213,
                        "name": "qimingxingchen",
                        "display_name": "启明星辰",
                        "read_mode": 2
                    },
                    {
                        "id": 179,
                        "name": "qingtengrongqi_common_kv",
                        "display_name": "青藤-蜂巢-容器安全-KEYVALUE-NULL",
                        "read_mode": 2
                    }
                ],
                "organizations": [
                    {
                        "id": 1,
                        "name": "根组织机构",
                        "superior_organization_id": 0,
                        "read_mode": 2
                    }
                ],
                "organization_names": [
                    "根组织机构"
                ],
                "today_parse_count": 6713,
                "today_non_standard_count": 0,
                "created_by_id": 1,
                "alarm_notify": {
                    "is_notify": false,
                    "email_receivers": null,
                    "sms_receivers": null,
                    "webhook_receivers": [],
                    "title": "",
                    "template": ""
                },
                "decode_type": 0
            }
        ]
    },
    "id": "0"
}
</code></pre>

### 请求参数

**Headers**

| 参数名称     | 参数值             | 是否必须 | 示例 | 备注 |
| ------------ |-----------------| -------- | ---- | ---- |
| Content-Type | application/json | 是       |   -   |   -   |
| authorization | bearer token_value | 是       |   -   |   -   |
| x-menu-name | 79                | 是       |   -   |   -   |
| x-request-path | pedestal        | 是       |   -   |   -   |

**Body**

<table>
  <thead class="ant-table-thead">
    <tr>
      <th key=name>名称</th><th key=type>类型</th><th key=required>是否必须</th><th key=default>默认值</th><th key=desc>备注</th><th key=sub>其他信息</th>
    </tr>
  </thead><tbody className="ant-table-tbody"><tr key=0-0><td key=0><span style="padding-left: 0px"><span style="color: #8c8a8a"></span> method</span></td><td key=1><span>string</span></td><td key=2>必须</td><td key=3>AgentService.SearchDeviceList</td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5><p key=2><span style="font-weight: '700'">枚举: </span><span>AgentService.CreateDevice</span></p></td></tr><tr key=0-1><td key=0><span style="padding-left: 0px"><span style="color: #8c8a8a"></span> params</span></td><td key=1><span>object</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-0><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> count</span></td><td key=1><span>integer</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">数量</span></td><td key=5></td></tr><tr key=0-1-1><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> offset</span></td><td key=1><span>integer</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">页码</span></td><td key=5></td></tr><tr key=0-1-2><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> id</span></td><td key=1><span>object []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">设备id搜索</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-1-2-0><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> oper</span></td><td key=1><span>string</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-2-1><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> target</span></td><td key=1><span>integer</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-3><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> ip</span></td><td key=1><span>object []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">设备ip搜索</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-1-3-0><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> oper</span></td><td key=1><span>string</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-3-1><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> target</span></td><td key=1><span>string</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-4><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> name</span></td><td key=1><span>object []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">设备名称搜索</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-1-4-0><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> oper</span></td><td key=1><span>string</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-4-1><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> target</span></td><td key=1><span>string</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-5><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> device_type</span></td><td key=1><span>object []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">设备类型搜索</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-1-5-0><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> oper</span></td><td key=1><span>string</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-5-1><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> target</span></td><td key=1><span>integer</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-6><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> product_name</span></td><td key=1><span>object []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">产品名称搜索</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-1-6-0><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> oper</span></td><td key=1><span>string</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-6-1><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> target</span></td><td key=1><span>string</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-7><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> organization_ids</span></td><td key=1><span>integer []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">关联单位ids搜索</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>integer</span></p></td></tr><tr key=array-37><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> </span></td><td key=1><span></span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">组织id</span></td><td key=5></td></tr><tr key=0-1-8><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> security_scope_id</span></td><td key=1><span>object []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">安全域id搜索</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-1-8-0><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> oper</span></td><td key=1><span>string</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-8-1><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> target</span></td><td key=1><span>integer</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-9><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> agent_id</span></td><td key=1><span>object []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">探针id搜索</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-1-9-0><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> oper</span></td><td key=1><span>string</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-9-1><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> target</span></td><td key=1><span>integer</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-10><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> last_receive_time</span></td><td key=1><span>object []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">最后接收时间搜索</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-1-10-0><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> oper</span></td><td key=1><span>string</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-10-1><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> target</span></td><td key=1><span>string</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-11><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> is_monitoring</span></td><td key=1><span>object []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">是否开启监控搜索</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-1-11-0><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> oper</span></td><td key=1><span>string</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-11-1><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> target</span></td><td key=1><span>boolean</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-12><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> status</span></td><td key=1><span>object []</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">是否开启</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-1-12-0><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> oper</span></td><td key=1><span>string</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-12-1><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> target</span></td><td key=1><span>integer</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2><td key=0><span style="padding-left: 0px"><span style="color: #8c8a8a"></span> jsonrpc</span></td><td key=1><span>string</span></td><td key=2>必须</td><td key=3>2.0</td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5><p key=2><span style="font-weight: '700'">枚举: </span><span>2.0</span></p></td></tr><tr key=0-3><td key=0><span style="padding-left: 0px"><span style="color: #8c8a8a"></span> id</span></td><td key=1><span>string</span></td><td key=2>必须</td><td key=3>0</td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5><p key=2><span style="font-weight: '700'">枚举: </span><span>0</span></p></td></tr>
               </tbody>
              </table>

### 返回数据

<table>
  <thead class="ant-table-thead">
    <tr>
      <th key=name>名称</th><th key=type>类型</th><th key=required>是否必须</th><th key=default>默认值</th><th key=desc>备注</th><th key=sub>其他信息</th>
    </tr>
  </thead><tbody className="ant-table-tbody"><tr key=0-0><td key=0><span style="padding-left: 0px"><span style="color: #8c8a8a"></span> jsonrpc</span></td><td key=1><span>string</span></td><td key=2>必须</td><td key=3>2.0</td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5><p key=2><span style="font-weight: '700'">枚举: </span><span>2.0</span></p></td></tr><tr key=0-1><td key=0><span style="padding-left: 0px"><span style="color: #8c8a8a"></span> result</span></td><td key=1><span>object</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-0><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> total</span></td><td key=1><span>number</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-1><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> data</span></td><td key=1><span>object []</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-1-1-0><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> id</span></td><td key=1><span>number</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">设备ID</span></td><td key=5></td></tr><tr key=0-1-1-1><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> name</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">设备名称</span></td><td key=5></td></tr><tr key=0-1-1-2><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> ip</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">设备IP</span></td><td key=5></td></tr><tr key=0-1-1-3><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> device_type</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">设备类型</span></td><td key=5></td></tr><tr key=0-1-1-4><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> vendor</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">设备厂商</span></td><td key=5></td></tr><tr key=0-1-1-5><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> status</span></td><td key=1><span>integer</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">1:启用；2:禁用；其他非法</span></td><td key=5></td></tr><tr key=0-1-1-6><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> product_name</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">产品名称</span></td><td key=5></td></tr><tr key=0-1-1-7><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> agent</span></td><td key=1><span>object</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">探针信息</span></td><td key=5></td></tr><tr key=0-1-1-7-0><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> id</span></td><td key=1><span>integer</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-1-7-1><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> name</span></td><td key=1><span>string</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-1-8><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> last_receive_time</span></td><td key=1><span>integer</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">上次事件接收时间</span></td><td key=5></td></tr><tr key=0-1-1-9><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> device_group</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">设备分组</span></td><td key=5></td></tr><tr key=0-1-1-10><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> update_err</span></td><td key=1><span>object</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">更新错误信息</span></td><td key=5></td></tr><tr key=0-1-1-11><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> control_agent</span></td><td key=1><span>object</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">控制探针信息</span></td><td key=5></td></tr><tr key=0-1-1-11-0><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> id</span></td><td key=1><span>number</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-1-11-1><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> name</span></td><td key=1><span>string</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-1-12><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> security_scope_id</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">安全域ID</span></td><td key=5></td></tr><tr key=0-1-1-13><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> license_start_time</span></td><td key=1><span>integer</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">license开始时间</span></td><td key=5></td></tr><tr key=0-1-1-14><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> license_end_time</span></td><td key=1><span>integer</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">license结束时间</span></td><td key=5></td></tr><tr key=0-1-1-15><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> is_monitoring</span></td><td key=1><span>boolean</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">是否开启监控</span></td><td key=5></td></tr><tr key=0-1-1-16><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> owner_id</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">责任人ID</span></td><td key=5></td></tr><tr key=0-1-1-17><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> security_scope</span></td><td key=1><span>object []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">相关安全域</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-1-1-17-0><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> id</span></td><td key=1><span>integer</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-1-17-1><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> name</span></td><td key=1><span>string</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-1-17-2><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> superior_security_scope_id</span></td><td key=1><span>integer</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-1-17-3><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> is_built_in</span></td><td key=1><span>boolean</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-1-18><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> organizations</span></td><td key=1><span>object []</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">相关单位</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-1-1-18-0><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> id</span></td><td key=1><span>number</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-1-18-1><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> name</span></td><td key=1><span>string</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-1-18-2><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> superior_organization_id</span></td><td key=1><span>integer</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-1-18-3><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> read_mode</span></td><td key=1><span>number</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">1:可读；2: 可编辑</span></td><td key=5></td></tr><tr key=0-1-1-19><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> organization_names</span></td><td key=1><span>string []</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">组织机构树名称</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>string</span></p></td></tr><tr key=array-38><tr key=0-1-1-20><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> owner</span></td><td key=1><span>object</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">责任人</span></td><td key=5></td></tr><tr key=0-1-1-20-0><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> id</span></td><td key=1><span>number</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-1-20-1><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> name</span></td><td key=1><span>string</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-1-20-2><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> username</span></td><td key=1><span>string</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-1-20-3><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> read_mode</span></td><td key=1><span>number</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">1:可编辑；2: 不可编辑</span></td><td key=5></td></tr><tr key=0-1-1-21><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> action</span></td><td key=1><span>object []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">设备操作列表</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-1-1-21-0><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> action_id</span></td><td key=1><span>number</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">操作id</span></td><td key=5></td></tr><tr key=0-1-1-21-1><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> name</span></td><td key=1><span>string</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">操作名称</span></td><td key=5></td></tr><tr key=0-1-1-21-2><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> password_verify</span></td><td key=1><span>boolean</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">是否需要密码验证</span></td><td key=5></td></tr><tr key=0-1-1-21-3><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> allowed</span></td><td key=1><span>boolean</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">是否允许操作</span></td><td key=5></td></tr><tr key=0-1-1-22><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> ssh</span></td><td key=1><span>object</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">远程配置</span></td><td key=5></td></tr><tr key=0-1-1-22-0><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> user</span></td><td key=1><span>string</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">SSH用户</span></td><td key=5></td></tr><tr key=0-1-1-22-1><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> password</span></td><td key=1><span>string</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">SSH密码</span></td><td key=5></td></tr><tr key=0-1-1-22-2><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> private_key</span></td><td key=1><span>string</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">SSH私钥</span></td><td key=5></td></tr><tr key=0-1-1-22-3><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> port</span></td><td key=1><span>number</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">SSH端口</span></td><td key=5></td></tr><tr key=0-1-1-22-4><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> login_type</span></td><td key=1><span>number</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">远程连接方式</span></td><td key=5></td></tr><tr key=0-1-1-23><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> normalize_rules</span></td><td key=1><span>object []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-1-1-23-0><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> id</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">规则id</span></td><td key=5></td></tr><tr key=0-1-1-23-1><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> name</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">规则名称</span></td><td key=5></td></tr><tr key=0-1-1-23-2><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> display_name</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">中文名称</span></td><td key=5></td></tr><tr key=0-1-1-23-3><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> read_mode</span></td><td key=1><span>number</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">1:可编辑；2: 不可编辑</span></td><td key=5></td></tr><tr key=0-1-1-24><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> today_parse_count</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">今日命中解析数</span></td><td key=5></td></tr><tr key=0-1-1-25><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> created_by_id</span></td><td key=1><span>number</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">创建人</span></td><td key=5></td></tr><tr key=0-1-1-26><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> alarm_notify</span></td><td key=1><span>object</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">设备告警通知配置</span></td><td key=5></td></tr><tr key=0-1-1-26-0><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> is_notify</span></td><td key=1><span>boolean</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-1-26-1><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> email_receivers</span></td><td key=1><span>object</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-1-26-1-0><td key=0><span style="padding-left: 80px"><span style="color: #8c8a8a">├─</span> notify_owner</span></td><td key=1><span>boolean</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-1-26-1-1><td key=0><span style="padding-left: 80px"><span style="color: #8c8a8a">├─</span> email</span></td><td key=1><span>string []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>string</span></p></td></tr>
<tr key=array-39>
<tr key=0-1-1-26-2><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> sms_receivers</span></td><td key=1><span>object</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-1-26-2-0><td key=0><span style="padding-left: 80px"><span style="color: #8c8a8a">├─</span> notify_owner</span></td><td key=1><span>boolean</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-1-26-2-1><td key=0><span style="padding-left: 80px"><span style="color: #8c8a8a">├─</span> sms</span></td><td key=1><span>string []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>string</span></p></td></tr><tr key=array-40><tr key=0-1-1-26-3><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> webhook_receivers</span></td><td key=1><span>integer []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>integer</span></p></td></tr><tr key=array-41><tr key=0-1-1-26-4><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> title</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-1-26-5><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> template</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-1-27><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> device_data_monitor</span></td><td key=1><span>object</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">设备接入数据监控</span></td><td key=5></td></tr><tr key=0-1-1-27-0><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> enable</span></td><td key=1><span>string</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">开关</span></td><td key=5></td></tr><tr key=0-1-1-27-1><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> interval</span></td><td key=1><span>string</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">周期 默认秒</span></td><td key=5></td></tr><tr key=0-1-1-27-2><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> template</span></td><td key=1><span>string</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">展示模板</span></td><td key=5></td></tr><tr key=0-2><td key=0><span style="padding-left: 0px"><span style="color: #8c8a8a"></span> id</span></td><td key=1><span>string</span></td><td key=2>必须</td><td key=3>0</td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5><p key=2><span style="font-weight: '700'">枚举: </span><span>0</span></p></td></tr>
               </tbody>
              </table>

<!-- endtoc -->
