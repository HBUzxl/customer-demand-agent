---
type: product
title: 万象AISOC-获取告警列表
product: 万象AISOC
aliases: []
tags:
    - 安全平台
    - 产品技术手册
    - 接口文档
    - 告警
summary: 介绍获取告警列表相关的接口能力、请求方式与使用说明。
status: verified
---

<!-- toc -->
<a id=告警列表2763> </a>

### 基本信息

**Path：** https://ip_addr/pedestal/rpc

**Method：** POST

**接口描述：**
<p>alarm_level_statistics: 是否对告警等级做统计<br>
attack_chain_total: 是否对攻击链做统计</p>
<h3>请求</h3>
<pre><code>{
    "jsonrpc": "2.0",
    "method": "AlarmService.GetAlarmList",
    "params": {
        "count": 100,
        "offset": 0,
        "attack_chain_statistics": false,
        "alarm_level_statistics": true,
        "ql": null,
        "alarm_name": [
            {
                "oper": "like",
                "target": "测试名称"
            }
        ],
        "alarm_level": [
            {
                "oper": "=",
                "target": "0"
            },
            {
                "oper": "=",
                "target": "1"
            },
            {
                "oper": "=",
                "target": "2"
            }
        ],
        "alarm_major_type": [
            {
                "oper": "=",
                "target": "50000"
            }
        ],
        "alarm_minor_type": [
            {
                "oper": "=",
                "target": "50001"
            }
        ],
        "alarm_area_id": [
            {
                "oper": "=",
                "target": 5
            }
        ],
        "attack_ip_string": [
            {
                "oper": "like",
                "target": "127.0.0.1"
            }
        ],
        "victim_ip_string": [
            {
                "oper": "like",
                "target": "127.0.0.1"
            }
        ],
        "attack_ip_asset": [
            {
                "oper": "=",
                "target": "219.232.88.128"
            }
        ],
        "victim_ip_asset": [
            {
                "oper": "=",
                "target": "219.232.88.128"
            }
        ],
        "device_id_int": [
            {
                "oper": "=",
                "target": 32
            }
        ],
        "attack_ip": null,
        "victim_ip": null,
        "created_at": [
            {
                "oper": "eq_in",
                "target": "1727884800000-1727971200000"
            }
        ],
        "updated_at": [
            {
                "oper": "eq_in",
                "target": "1729612800000-1729699199999"
            }
        ],
        "operation_at": [
            {
                "oper": "eq_in",
                "target": "1727798400000-1727884800000"
            }
        ],
        "log_start_at": [
            {
                "oper": "eq_in",
                "target": "1727712000000-1727798400000"
            }
        ],
        "log_end_at": [
            {
                "oper": "eq_in",
                "target": "1727884800000-1727971200000"
            }
        ],
        "victim_web_url": [
            {
                "oper": "like",
                "target": "https://www.baidu.com"
            }
        ],
        "tag": [
            {
                "oper": "array_in",
                "target": "019270b3-1"
            }
        ],
        "attack_chain_phase": [
            {
                "oper": "array_in",
                "target": 1
            }
        ],
        "query_id": [
            {
                "oper": "=",
                "target": "37f35cae-0ebe-43c1-98e7-795d72d69ec9"
            }
        ],
        "id": [
            {
                "oper": "=",
                "target": "0192b80b-8952-7840-8999-e904fbdbd014"
            }
        ],
        "custom_data": null,
        "device_id": null,
        "origin_log_id": null,
        "comment": null,
        "location_id": null,
        "judged_state": [
            {
                "oper": "=",
                "target": 0
            }
        ],
        "disposed_state": [
            {
                "oper": "=",
                "target": 1
            }
        ],
        "victim_web_asset_id": null,
        "attack_result": [
            {
                "oper": "=",
                "target": 2
            }
        ],
        "fall": null,
        "operate_event": [
            {
                "oper": "array_in",
                "target": 1
            }
        ],
        "notice_status": [
            {
                "oper": "=",
                "target": 1
            }
        ],
        "disposed_raw_status": null,
        "attack_port": null,
        "victim_port": null,
        "attack_method": [
            {
                "oper": "=",
                "target": "攻击方式"
            }
        ],
        "business_ext": null,
        "advanced_query": null,
        "condition_query": null,
        "hit_intelligence": [
            {
                "oper": "=",
                "target": 1
            }
        ],
        "window_time": null,
        "organization_id": [
            {
                "oper": "=",
                "target": 10
            }
        ],
        "ip": null,
        "is_blocked": true,
        "is_hi_blocked": true,
        "scenarios": 0,
        "workflow_id": 0,
        "attack_direction": [
            {
                "oper": "array_in",
                "target": "lateral"
            }
        ],
        "filter": null,
        "is_asset_hit": [
            {
                "oper": "=",
                "target": 1
            }
        ],
        "workflow_status": [
            {
                "oper": "=",
                "target": 1
            }
        ],
        "attack_ip_scope": [
            {
                "oper": "=",
                "target": 5
            }
        ],
        "victim_ip_scope": [
            {
                "oper": "=",
                "target": 5
            }
        ],
        "host_asset_by_web": [
            {
                "oper": "=",
                "target": 2317
            }
        ],
        "focused": [
            {
                "oper": "=",
                "target": false
            }
        ]
    },
    "id": "0"
}
</code></pre>
<h3>响应</h3>
<pre><code>{
    "jsonrpc": "2.0",
    "result": {
        "total": 49,
        "data": [
            {
                "id": "4c9b1f9f-e2a3-4bc2-a91b-a47db200ee2d",
                "alarm_name": "qqqqqq",
                "alarm_level": "-1",
                "alarm_major_type": "50000",
                "alarm_minor_type": "50004",
                "comment": "qqqqqq",
                "created_at": 1710299183.296589,
                "updated_at": 1710299183.296589,
                "operation_at": 1710299183.000000,
                "etl_time": 1710090759.854000,
                "alarm_area": {
                    "id": 5,
                    "content": "DMZ域"
                },
                "query_id": "",
                "custom_data": "",
                "tag": [],
                "attack_ip": [
                    "2.1.93.158"
                ],
                "victim_ip": [
                    "10.13.20.5"
                ],
                "victim_web_url": [],
                "attack_chain_phase": [
                    123,
                    45,
                    49,
                    125
                ],
                "device_id": [],
                "disposition_advice": "qqqqqq",
                "judged_state": 1,
                "disposed_state": 0,
                "log_count": 1,
                "operate_event": [],
                "iso_code": "FR",
                "victim_iso_code": null,
                "origin_log_ids": [
                    "1766874824874929145"
                ],
                "attack_result": 1,
                "fall": 1,
                "payload": "",
                "disposed_raw_status": "未发送",
                "attack_method": "",
                "business_ext": "",
                "log_start_at": 1710090759.854000,
                "log_end_at": 1710090759.854000,
                "attack_port": [],
                "victim_port": [],
                "archived_at": -62135596800.000000,
                "hit_intelligence": 2,
                "window_time": "",
                "engine_type": "",
                "attack_ip_pic": "",
                "victim_ip_pic": "",
                "attack_direction": "lateral",
                "extra_field": {
                    "operate_event": "ezEwMDUsMX0="
                }
            }
        ],
        "attack_chain_total": {
            "recon": 0,
            "exploit": 0,
            "delivery": 0,
            "command_control": 0,
            "lateral": 0,
            "actions_on_objective": 0,
            "cleanup": 0
        },
        "alarm_level_total": {
            "urgent": 0,
            "high": 0,
            "middle": 0,
            "low": 0
        }
    },
    "id": 0
}
</code></pre>


### 请求参数
**Headers**

| 参数名称  | 参数值  |  是否必须 | 示例 | 备注 |
| ------------ | ------------ | ------------ |----|----|
| Content-Type  |  application/json | 是  | -  | -  |
| authorization | bearer token_value | 是       | -  | -  |
| x-menu-name | 26                | 是       | -  | -  |
| x-request-path | pedestal        | 是       | -  | -  |
**Body**

<table>
  <thead class="ant-table-thead">
    <tr>
      <th key=name>名称</th><th key=type>类型</th><th key=required>是否必须</th><th key=default>默认值</th><th key=desc>备注</th><th key=sub>其他信息</th>
    </tr>
  </thead><tbody className="ant-table-tbody"><tr key=0-0><td key=0><span style="padding-left: 0px"><span style="color: #8c8a8a"></span> jsonrpc</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1><td key=0><span style="padding-left: 0px"><span style="color: #8c8a8a"></span> method</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2><td key=0><span style="padding-left: 0px"><span style="color: #8c8a8a"></span> params</span></td><td key=1><span>object</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2-0><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> count</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">查询数量</span></td><td key=5></td></tr><tr key=0-2-1><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> offset</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">偏移量</span></td><td key=5></td></tr><tr key=0-2-2><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> attack_chain_statistics</span></td><td key=1><span>boolean</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">是否统计攻击链</span></td><td key=5></td></tr><tr key=0-2-3><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> alarm_level_statistics</span></td><td key=1><span>boolean</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">是否统计告警等级</span></td><td key=5></td></tr><tr key=0-2-4><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> alarm_name</span></td><td key=1><span>object []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">告警名称</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-2-4-0><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> oper</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2-4-1><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> target</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2-5><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> alarm_level</span></td><td key=1><span>object []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">告警等级</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-2-5-0><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> oper</span></td><td key=1><span>string</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2-5-1><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> target</span></td><td key=1><span>string</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2-6><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> alarm_major_type</span></td><td key=1><span>object []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">告警一级分类</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-2-6-0><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> oper</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2-6-1><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> target</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2-7><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> alarm_minor_type</span></td><td key=1><span>object []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">告警二级分类</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-2-7-0><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> oper</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2-7-1><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> target</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2-8><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> alarm_area_id</span></td><td key=1><span>object []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">告警区域</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-2-8-0><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> oper</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2-8-1><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> target</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2-9><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> attack_ip_string</span></td><td key=1><span>object []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">攻击IP</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-2-9-0><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> oper</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2-9-1><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> target</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2-10><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> victim_ip_string</span></td><td key=1><span>object []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">受害IP</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-2-10-0><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> oper</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2-10-1><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> target</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2-11><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> attack_ip_asset</span></td><td key=1><span>object []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">攻击IP资产</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-2-11-0><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> oper</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2-11-1><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> target</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2-12><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> victim_ip_asset</span></td><td key=1><span>object []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">受害IP资产</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-2-12-0><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> oper</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2-12-1><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> target</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2-13><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> device_id_int</span></td><td key=1><span>object []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">关联安全设备</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-2-13-0><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> oper</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2-13-1><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> target</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2-14><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> created_at</span></td><td key=1><span>object []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">告警时间</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-2-14-0><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> oper</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2-14-1><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> target</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2-15><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> updated_at</span></td><td key=1><span>object []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">告警更新时间</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-2-15-0><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> oper</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2-15-1><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> target</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2-16><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> operation_at</span></td><td key=1><span>object []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">告警操作时间</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-2-16-0><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> oper</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2-16-1><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> target</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2-17><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> log_start_at</span></td><td key=1><span>object []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">日志开始时间</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-2-17-0><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> oper</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2-17-1><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> target</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2-18><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> log_end_at</span></td><td key=1><span>object []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">日志结束时间</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-2-18-0><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> oper</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2-18-1><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> target</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2-19><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> victim_web_url</span></td><td key=1><span>object []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">受害应用</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-2-19-0><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> oper</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2-19-1><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> target</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2-20><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> tag</span></td><td key=1><span>object []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">标签</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-2-20-0><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> oper</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2-20-1><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> target</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2-21><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> attack_chain_phase</span></td><td key=1><span>object []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">攻击链</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-2-21-0><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> oper</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2-21-1><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> target</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2-22><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> query_id</span></td><td key=1><span>object []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">分析规则</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-2-22-0><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> oper</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2-22-1><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> target</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2-23><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> id</span></td><td key=1><span>object []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">告警ID</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-2-23-0><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> oper</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2-23-1><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> target</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2-24><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> judged_state</span></td><td key=1><span>object []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">研判状态</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-2-24-0><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> oper</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2-24-1><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> target</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2-25><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> disposed_state</span></td><td key=1><span>object []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">处置状态</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-2-25-0><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> oper</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2-25-1><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> target</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2-26><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> attack_result</span></td><td key=1><span>object []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">攻击结果</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-2-26-0><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> oper</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2-26-1><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> target</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2-27><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> fall</span></td><td key=1><span>null</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">失陷状态</span></td><td key=5></td></tr><tr key=0-2-28><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> operate_event</span></td><td key=1><span>object []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">研判处置状态</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-2-28-0><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> oper</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2-28-1><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> target</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2-29><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> notice_status</span></td><td key=1><span>object []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">通知状态</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-2-29-0><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> oper</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2-29-1><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> target</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2-30><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> attack_method</span></td><td key=1><span>object []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">攻击方式</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-2-30-0><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> oper</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2-30-1><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> target</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2-31><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> condition_query</span></td><td key=1><span>null</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">条件筛选</span></td><td key=5></td></tr><tr key=0-2-32><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> hit_intelligence</span></td><td key=1><span>object []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">命中情报</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-2-32-0><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> oper</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2-32-1><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> target</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2-33><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> organization_id</span></td><td key=1><span>object []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">组织机构</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-2-33-0><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> oper</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2-33-1><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> target</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2-34><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> is_blocked</span></td><td key=1><span>boolean</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">隔离状态</span></td><td key=5></td></tr><tr key=0-2-35><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> is_hi_blocked</span></td><td key=1><span>boolean</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">封禁状态</span></td><td key=5></td></tr><tr key=0-2-36><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> workflow_id</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">工单筛选</span></td><td key=5></td></tr><tr key=0-2-37><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> attack_direction</span></td><td key=1><span>object []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">攻击方向</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-2-37-0><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> oper</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2-37-1><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> target</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2-38><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> is_asset_hit</span></td><td key=1><span>object []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">命中资产</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-2-38-0><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> oper</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2-38-1><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> target</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2-39><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> attack_ip_scope</span></td><td key=1><span>object []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">攻击IP所属安全域</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-2-39-0><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> oper</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2-39-1><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> target</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2-40><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> victim_ip_scope</span></td><td key=1><span>object []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">受害IP所属安全域</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-2-40-0><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> oper</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2-40-1><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> target</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2-41><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> host_asset_by_web</span></td><td key=1><span>object []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">应用关联IP资产</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-2-41-0><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> oper</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2-41-1><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> target</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2-42><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> focused</span></td><td key=1><span>object []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">重点告警</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-2-42-0><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> oper</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2-42-1><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> target</span></td><td key=1><span>boolean</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-3><td key=0><span style="padding-left: 0px"><span style="color: #8c8a8a"></span> id</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr>
               </tbody>
              </table>

### 返回数据

<table>
  <thead class="ant-table-thead">
    <tr>
      <th key=name>名称</th><th key=type>类型</th><th key=required>是否必须</th><th key=default>默认值</th><th key=desc>备注</th><th key=sub>其他信息</th>
    </tr>
  </thead><tbody className="ant-table-tbody"><tr key=0-0><td key=0><span style="padding-left: 0px"><span style="color: #8c8a8a"></span> jsonrpc</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1><td key=0><span style="padding-left: 0px"><span style="color: #8c8a8a"></span> result</span></td><td key=1><span>object</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-0><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> total</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-1><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> data</span></td><td key=1><span>object []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-1-1-0><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> id</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">告警ID</span></td><td key=5></td></tr><tr key=0-1-1-1><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> alarm_name</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">告警名称</span></td><td key=5></td></tr><tr key=0-1-1-2><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> alarm_level</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">告警等级</span></td><td key=5></td></tr><tr key=0-1-1-3><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> alarm_major_type</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">告警一级分类</span></td><td key=5></td></tr><tr key=0-1-1-4><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> alarm_minor_type</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">告警二级分类</span></td><td key=5></td></tr><tr key=0-1-1-5><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> comment</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">告警原因</span></td><td key=5></td></tr><tr key=0-1-1-6><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> created_at</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">告警时间</span></td><td key=5></td></tr><tr key=0-1-1-7><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> updated_at</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">告警更新时间</span></td><td key=5></td></tr><tr key=0-1-1-8><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> operation_at</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">告警操作时间</span></td><td key=5></td></tr><tr key=0-1-1-9><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> etl_time</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">日志处理时间</span></td><td key=5></td></tr><tr key=0-1-1-10><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> alarm_area</span></td><td key=1><span>object</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">告警区域</span></td><td key=5></td></tr><tr key=0-1-1-10-0><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> id</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-1-10-1><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> content</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-1-11><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> query_id</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">关联分析规则ID</span></td><td key=5></td></tr><tr key=0-1-1-12><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> tag</span></td><td key=1><span>string []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">标签</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>string</span></p></td></tr><tr key=array-853><tr key=0-1-1-13><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> attack_ip</span></td><td key=1><span>string []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">攻击IP</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>string</span></p></td></tr><tr key=array-854><tr key=0-1-1-14><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> victim_ip</span></td><td key=1><span>string []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">受害IP</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>string</span></p></td></tr><tr key=array-855><tr key=0-1-1-15><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> victim_web_url</span></td><td key=1><span>string []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">受害应用</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>string</span></p></td></tr><tr key=array-856><tr key=0-1-1-16><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> attack_chain_phase</span></td><td key=1><span>number []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">攻击链阶段</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>number</span></p></td></tr><tr key=array-857><tr key=0-1-1-17><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> device_id</span></td><td key=1><span>string []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">关联设备ID</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>string</span></p></td></tr><tr key=array-858><tr key=0-1-1-18><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> disposition_advice</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">处置建议</span></td><td key=5></td></tr><tr key=0-1-1-19><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> judged_state</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">研判状态</span></td><td key=5></td></tr><tr key=0-1-1-20><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> disposed_state</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">处置状态</span></td><td key=5></td></tr><tr key=0-1-1-21><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> log_count</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">关联日志数</span></td><td key=5></td></tr><tr key=0-1-1-22><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> operate_event</span></td><td key=1><span>string []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">操作事件</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>string</span></p></td></tr><tr key=array-859><tr key=0-1-1-23><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> iso_code</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">攻击IP国家编码</span></td><td key=5></td></tr><tr key=0-1-1-24><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> victim_iso_code</span></td><td key=1><span>null</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">受害IP国家编码</span></td><td key=5></td></tr><tr key=0-1-1-25><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> origin_log_ids</span></td><td key=1><span>string []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">关联的安全告警日志id</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>string</span></p></td></tr><tr key=array-860><tr key=0-1-1-26><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> attack_result</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">攻击结果</span></td><td key=5></td></tr><tr key=0-1-1-27><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> fall</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">失陷状态</span></td><td key=5></td></tr><tr key=0-1-1-28><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> payload</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">payload</span></td><td key=5></td></tr><tr key=0-1-1-29><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> attack_method</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">攻击方式</span></td><td key=5></td></tr><tr key=0-1-1-30><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> business_ext</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">业务信息</span></td><td key=5></td></tr><tr key=0-1-1-31><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> log_start_at</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">日志时间-开始</span></td><td key=5></td></tr><tr key=0-1-1-32><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> log_end_at</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">日志时间-结束</span></td><td key=5></td></tr><tr key=0-1-1-33><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> attack_port</span></td><td key=1><span>string []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">攻击者端口</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>string</span></p></td></tr><tr key=array-861><tr key=0-1-1-34><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> victim_port</span></td><td key=1><span>string []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">受害者端口</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>string</span></p></td></tr><tr key=array-862><tr key=0-1-1-35><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> hit_intelligence</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">是否命中情报</span></td><td key=5></td></tr><tr key=0-1-1-36><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> window_time</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">产生告警的窗口</span></td><td key=5></td></tr><tr key=0-1-1-37><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> engine_type</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">产生告警的引擎</span></td><td key=5></td></tr><tr key=0-1-1-38><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> attack_ip_pic</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">攻击ip责任人</span></td><td key=5></td></tr><tr key=0-1-1-39><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> victim_ip_pic</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">受害ip责任人</span></td><td key=5></td></tr><tr key=0-1-1-40><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> attack_direction</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">攻击方向</span></td><td key=5></td></tr><tr key=0-1-1-41><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> extra_field</span></td><td key=1><span>object</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">用户自定义字段</span></td><td key=5></td></tr><tr key=0-1-1-41-0><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> operate_event</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-2><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> attack_chain_total</span></td><td key=1><span>object</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-2-0><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> recon</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">侦查</span></td><td key=5></td></tr><tr key=0-1-2-1><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> exploit</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">入侵</span></td><td key=5></td></tr><tr key=0-1-2-2><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> delivery</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">载荷投递</span></td><td key=5></td></tr><tr key=0-1-2-3><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> command_control</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">命令与控制</span></td><td key=5></td></tr><tr key=0-1-2-4><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> lateral</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">内网横向</span></td><td key=5></td></tr><tr key=0-1-2-5><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> actions_on_objective</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">达成目标</span></td><td key=5></td></tr><tr key=0-1-2-6><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> cleanup</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">痕迹清理</span></td><td key=5></td></tr><tr key=0-1-3><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> alarm_level_total</span></td><td key=1><span>object</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-3-0><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> urgent</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">超危</span></td><td key=5></td></tr><tr key=0-1-3-1><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> high</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">高危</span></td><td key=5></td></tr><tr key=0-1-3-2><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> middle</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">中危</span></td><td key=5></td></tr><tr key=0-1-3-3><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> low</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">低危</span></td><td key=5></td></tr><tr key=0-2><td key=0><span style="padding-left: 0px"><span style="color: #8c8a8a"></span> id</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr>
               </tbody>
              </table>


<!-- endtoc -->
