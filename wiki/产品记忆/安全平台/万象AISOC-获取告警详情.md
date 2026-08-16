---
type: product
title: 万象AISOC-获取告警详情
product: 万象AISOC
aliases: []
tags:
    - 安全平台
    - 产品技术手册
    - 接口文档
    - 告警
summary: 介绍获取告警详情相关的接口能力、请求方式与使用说明。
status: verified
---

<!-- toc -->

<a id=告警详情2763> </a>

### 基本信息

**Path：** https://ip_addr/pedestal/rpc

**Method：** POST

**接口描述：**
<h3>请求</h3>
<pre><code>{
    "jsonrpc": "2.0",
    "method": "AlarmService.GetAlarmInfo",
    "params": {
        "id": "01929938-f6e9-72cc-906d-8e5ab6a935ab"
    },
    "id": "0"
}
</code></pre>

<h3>响应</h3>
<pre><code>{
    "jsonrpc": "2.0",
    "result": {
        "id": "01929938-f6e9-72cc-906d-8e5ab6a935ab",
        "alarm_name": "滚动-30分-变量推荐",
        "alarm_level": "4",
        "alarm_major_type": "51000",
        "alarm_minor_type": "51025",
        "comment": "超危=超危\nSQL注入 in 端口扫描、SQL注入\n其它 in 目的执行、痕迹清理\n成功 in 成功、疑似成功\n内到外 in 内到内、外到内\n区域：设备所属安全域\n失陷：基于攻击结果判定",
        "created_at": 1729147500.300807,
        "updated_at": 1729147500.300807,
        "operation_at": -62135596800.000000,
        "etl_time": 1729147446.181000,
        "alarm_area": {
            "id": 5,
            "content": "DMZ域"
        },
        "query_id": "8f2e3359-6986-46f1-8d38-2fa329bf8268",
        "tag": [],
        "attack_ip": [
            "2.1.1.189",
            "2.1.1.178",
            "2.1.1.78",
            "2.1.1.203",
            "2.1.1.0",
            "2.1.1.153",
            "2.1.1.138"
        ],
        "victim_ip": [
            "123.101.10.121",
            "123.101.10.92",
            "123.101.10.53",
            "123.101.10.114",
            "123.101.10.13",
            "123.101.10.67",
            "123.101.10.29"
        ],
        "victim_web_url": [],
        "attack_chain_phase": [
            -1
        ],
        "device_id": [
            2
        ],
        "disposition_advice": "1",
        "judged_state": 0,
        "disposed_state": 0,
        "log_count": 7,
        "operate_event": [],
        "iso_code": null,
        "victim_iso_code": null,
        "origin_log_ids": [
            "1846804360563000429",
            "1846804394142606454",
            "1846804427730584703",
            "1846804461314376840",
            "1846804494889784465",
            "1846804528477766810",
            "1846804562061563043"
        ],
        "attack_result": -1,
        "fall": -1,
        "payload": "",
        "disposed_raw_status": "未发送",
        "attack_method": "",
        "business_ext": "",
        "log_start_at": 1729147446.000000,
        "log_end_at": 1729147494.000000,
        "attack_port": [],
        "victim_port": [],
        "archived_at": -62135596800.000000,
        "hit_intelligence": 2,
        "window_time": "2024-10-17T14:44~2024-10-17T14:45",
        "engine_type": "flink",
        "attack_ip_pic": "",
        "victim_ip_pic": "",
        "attack_direction": "out",
        "extra_field": {
            "eve_new_ip": null
        },
        "is_asset_hit": 0,
        "focused": false,
        "base_focused": false,
        "http_status": "",
        "dns_info": "",
        "account_info": "",
        "attacker_info": "",
        "victim_info": "",
        "suspicious_action": "",
        "vuln_info": "",
        "weak_pwd": "",
        "compliance_baseline": "",
        "file_info": "",
        "file_tags": "",
        "endpoint_info": "",
        "endpoint_protection": "",
        "origin_info": "",
        "protocol_info": "",
        "email_info": "",
        "sensitive_data": "",
        "event_history": []
    },
    "id": "0"
}
</code></pre>


### 请求参数
**Headers**

| 参数名称  | 参数值  |  是否必须 | 示例  | 备注  |
| ------------ | ------------ | ------------ | ------------ | ------------ |
| Content-Type  |  application/json | 是  |   |   |
| authorization | bearer token_value | 是       |   -   |   -   |
| x-menu-name | 26                | 是       |   -   |   -   |
| x-request-path | pedestal        | 是       |   -   |   -   |
**Body**

<table>
  <thead class="ant-table-thead">
    <tr>
      <th key=name>名称</th><th key=type>类型</th><th key=required>是否必须</th><th key=default>默认值</th><th key=desc>备注</th><th key=sub>其他信息</th>
    </tr>
  </thead><tbody className="ant-table-tbody"><tr key=0-0><td key=0><span style="padding-left: 0px"><span style="color: #8c8a8a"></span> jsonrpc</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1><td key=0><span style="padding-left: 0px"><span style="color: #8c8a8a"></span> method</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2><td key=0><span style="padding-left: 0px"><span style="color: #8c8a8a"></span> params</span></td><td key=1><span>object</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2-0><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> id</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">告警ID</span></td><td key=5></td></tr><tr key=0-3><td key=0><span style="padding-left: 0px"><span style="color: #8c8a8a"></span> id</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr>
               </tbody>
              </table>

### 返回数据

<table>
  <thead class="ant-table-thead">
    <tr>
      <th key=name>名称</th><th key=type>类型</th><th key=required>是否必须</th><th key=default>默认值</th><th key=desc>备注</th><th key=sub>其他信息</th>
    </tr>
  </thead><tbody className="ant-table-tbody">
<tr key=0-0><td key=0><span style="padding-left: 0px"><span style="color: #8c8a8a"></span> jsonrpc</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1><td key=0><span style="padding-left: 0px"><span style="color: #8c8a8a"></span> result</span></td><td key=1><span>object</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-0><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> id</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">告警ID</span></td><td key=5></td></tr><tr key=0-1-1><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> alarm_name</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">告警名称</span></td><td key=5></td></tr><tr key=0-1-2><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> alarm_level</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">告警等级</span></td><td key=5></td></tr><tr key=0-1-3><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> alarm_major_type</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">告警一级分类</span></td><td key=5></td></tr><tr key=0-1-4><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> alarm_minor_type</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">告警二级分类</span></td><td key=5></td></tr><tr key=0-1-5><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> comment</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">告警原因</span></td><td key=5></td></tr><tr key=0-1-6><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> created_at</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">告警时间</span></td><td key=5></td></tr><tr key=0-1-7><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> updated_at</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">告警更新时间</span></td><td key=5></td></tr><tr key=0-1-8><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> operation_at</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">告警操作时间</span></td><td key=5></td></tr><tr key=0-1-9><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> etl_time</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">日志处理时间</span></td><td key=5></td></tr><tr key=0-1-10><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> alarm_area</span></td><td key=1><span>object</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">告警区域</span></td><td key=5></td></tr><tr key=0-1-10-0><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> id</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">告警区域ID</span></td><td key=5></td></tr><tr key=0-1-10-1><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> content</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">告警区域名称</span></td><td key=5></td></tr><tr key=0-1-11><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> query_id</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">关联分析规则ID</span></td><td key=5></td></tr><tr key=0-1-12><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> tag</span></td><td key=1><span>string []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">标签</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>string</span></p></td></tr>

[//]: # (<tr key=array-863>)
<tr key=0-1-13><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> attack_ip</span></td><td key=1><span>string []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">攻击IP</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>string</span></p></td></tr>

[//]: # (<tr key=array-864>)
<tr key=0-1-14><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> victim_ip</span></td><td key=1><span>string []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">受害IP</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>string</span></p></td></tr>

[//]: # (<tr key=array-865>)
<tr key=0-1-15><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> victim_web_url</span></td><td key=1><span>string []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">受害应用</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>string</span></p></td></tr>
<tr key=array-866>
<tr key=0-1-16><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> attack_chain_phase</span></td><td key=1><span>number []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">攻击链阶段</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>number</span></p></td></tr><tr key=array-867><tr key=0-1-17><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> device_id</span></td><td key=1><span>number []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">关联设备ID</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>number</span></p></td></tr><tr key=array-868><tr key=0-1-18><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> disposition_advice</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">处置建议</span></td><td key=5></td></tr><tr key=0-1-19><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> judged_state</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">研判状态</span></td><td key=5></td></tr><tr key=0-1-20><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> disposed_state</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">处置状态</span></td><td key=5></td></tr><tr key=0-1-21><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> log_count</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">关联日志数</span></td><td key=5></td></tr><tr key=0-1-22><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> operate_event</span></td><td key=1><span>string []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">操作事件</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>string</span></p></td></tr><tr key=array-869><tr key=0-1-23><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> iso_code</span></td><td key=1><span>null</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">攻击IP国家编码</span></td><td key=5></td></tr><tr key=0-1-24><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> victim_iso_code</span></td><td key=1><span>null</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">受害IP国家编码</span></td><td key=5></td></tr><tr key=0-1-25><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> origin_log_ids</span></td><td key=1><span>string []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">关联的安全告警日志id</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>string</span></p></td></tr><tr key=array-870><tr key=0-1-26><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> attack_result</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">攻击结果</span></td><td key=5></td></tr><tr key=0-1-27><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> fall</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">失陷状态</span></td><td key=5></td></tr><tr key=0-1-28><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> payload</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">payload</span></td><td key=5></td></tr><tr key=0-1-29><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> attack_method</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">攻击方式</span></td><td key=5></td></tr><tr key=0-1-30><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> business_ext</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">业务信息</span></td><td key=5></td></tr><tr key=0-1-31><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> log_start_at</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">日志时间-开始</span></td><td key=5></td></tr><tr key=0-1-32><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> log_end_at</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">日志时间-结束</span></td><td key=5></td></tr><tr key=0-1-33><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> attack_port</span></td><td key=1><span>string []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">攻击者端口</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>string</span></p></td></tr><tr key=array-871><tr key=0-1-34><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> victim_port</span></td><td key=1><span>string []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">受害者端口</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>string</span></p></td></tr><tr key=array-872><tr key=0-1-35><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> hit_intelligence</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">是否命中情报</span></td><td key=5></td></tr><tr key=0-1-36><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> window_time</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">产生告警的窗口</span></td><td key=5></td></tr><tr key=0-1-37><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> engine_type</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">产生告警的引擎</span></td><td key=5></td></tr><tr key=0-1-38><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> attack_ip_pic</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">攻击ip责任人</span></td><td key=5></td></tr><tr key=0-1-39><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> victim_ip_pic</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">受害ip责任人</span></td><td key=5></td></tr><tr key=0-1-40><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> attack_direction</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">攻击方向</span></td><td key=5></td></tr><tr key=0-1-41><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> extra_field</span></td><td key=1><span>object</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">用户自定义字段信息</span></td><td key=5></td></tr><tr key=0-1-41-0><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> eve_new_ip</span></td><td key=1><span>null</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-42><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> is_asset_hit</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">是否命中资产</span></td><td key=5></td></tr><tr key=0-1-43><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> focused</span></td><td key=1><span>boolean</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">重点告警</span></td><td key=5></td></tr><tr key=0-1-44><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> base_focused</span></td><td key=1><span>boolean</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">产生告警时是否为重点告警</span></td><td key=5></td></tr><tr key=0-1-45><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> http_status</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">网络攻击-HTTP 状态</span></td><td key=5></td></tr><tr key=0-1-46><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> dns_info</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">网络攻击-DNS 信息</span></td><td key=5></td></tr><tr key=0-1-47><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> account_info</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">用户异常行为 - 账户信息</span></td><td key=5></td></tr><tr key=0-1-48><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> attacker_info</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">用户异常行为 - 攻击者信息</span></td><td key=5></td></tr><tr key=0-1-49><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> victim_info</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">用户异常行为 - 受害者信息</span></td><td key=5></td></tr><tr key=0-1-50><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> suspicious_action</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">用户异常行为 - 可疑操作</span></td><td key=5></td></tr><tr key=0-1-51><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> vuln_info</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">脆弱性 - 漏洞信息</span></td><td key=5></td></tr><tr key=0-1-52><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> weak_pwd</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">脆弱性 - 弱口令</span></td><td key=5></td></tr><tr key=0-1-53><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> compliance_baseline</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">脆弱性 - 合规基线</span></td><td key=5></td></tr><tr key=0-1-54><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> file_info</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">意文件 - 文件信息</span></td><td key=5></td></tr><tr key=0-1-55><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> file_tags</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">恶意文件 - 恶意标识</span></td><td key=5></td></tr><tr key=0-1-56><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> endpoint_info</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">主机安全 - 终端信息</span></td><td key=5></td></tr><tr key=0-1-57><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> endpoint_protection</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">主机安全 - 终端防护</span></td><td key=5></td></tr><tr key=0-1-58><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> origin_info</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">更多信息 - 原始信息</span></td><td key=5></td></tr><tr key=0-1-59><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> protocol_info</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">更多信息 - 协议信息</span></td><td key=5></td></tr><tr key=0-1-60><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> email_info</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">更多信息 - 邮件信息</span></td><td key=5></td></tr><tr key=0-1-61><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> sensitive_data</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">更多信息 - 敏感数据</span></td><td key=5></td></tr><tr key=0-1-62><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> event_history</span></td><td key=1><span>string []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">告警事件历史</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>string</span></p></td></tr><tr key=array-873><tr key=0-2><td key=0><span style="padding-left: 0px"><span style="color: #8c8a8a"></span> id</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr>
               </tbody>
              </table>

<!-- endtoc -->
