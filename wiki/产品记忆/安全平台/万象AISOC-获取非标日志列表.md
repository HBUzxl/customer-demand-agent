---
type: product
title: 万象AISOC-获取非标日志列表
product: 万象AISOC
aliases: []
tags:
    - 安全平台
    - 产品技术手册
    - 接口文档
    - 非标日志
summary: 介绍获取非标日志列表相关的接口能力、请求方式与使用说明。
status: verified
---

<!-- toc -->

<a id=非标日志列表2781> </a>

### 基本信息

**Path：** https://ip_addr/pedestal/rpc

**Method：** POST

**接口描述：**

<h3>请求</h3>
<pre><code>{
    "method": "NonstandardLogApiService.SearchNonstandardLog",
    "params": {
        "offset": 0,
        "count": 20,
        "key_words": [
            "\u003c165\u003e1"
        ],
        "start_time": 1729064590079,
        "end_time": 1729150990079,
        "dev_type": [
            1001
        ],
        "log_id": [
            {
                "oper": "=",
                "target": "1846819046251566033"
            }
        ],
        "device_name": [
            {
                "oper": "like",
                "target": "master-1"
            }
        ],
        "agent_name": [
            {
                "oper": "like",
                "target": "产品内置-k3s-master-2"
            }
        ],
        "condition_query": null
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
                "log_id": "1846819046251566033",
                "dev_id": 6,
                "device_name": "master-1",
                "dev_type": 1001,
                "agent_id": 3,
                "agent_name": "产品内置-k3s-master-2",
                "received_time": 1729150947.000000,
                "origin_log": "&lt;165&gt;1 2019-07-11T22:14:15.000003-07:00 10.2.33.114 test 8888 cloudwalker-webshell - {\"action\":\"allow\",\"attack_type\":\"command_injection\",\"body\":\"--------------\",\"cookie\":\"\",\"country\":\"\",\"decode_path\":\"\",\"dest_ip\":\"123.101.10.18\",\"dest_port\":80,\"detector_ip_source\":\"Socket\",\"event_id\":\"6fe9ed4bdd0f4ecd81febae272eded8a\",\"host\":\"\",\"location\":\"\",\"method\":\"POST\",\"module\":\"m_file_upload\",\"node\":\"chaitin-safeline\",\"payload\":\"\",\"protocol\":\"http\",\"province\":\"\",\"proxy_name\":\"872e16e4dd6e\",\"query_string\":\"\",\"reason\":\"......... ............ ......\",\"referer\":\"\",\"req_action\":\"allow\",\"req_attack_type\":\"file_upload\",\"req_decode_path\":\"\",\"req_detector_name\":\"chaitin-safeline\",\"req_end_time\":1617264570749,\"req_header_raw\":\"POST /xoda-0.4.3/?upload HTTP/1.0\\r\\n\\tContent-Type: multipart/form-data\\t; boundary=---------------------------496965075811\\t83027245\\t4563555\\r\\nContent-length: 320\\r\\n\\r\\n\",\"req_location\":\"\",\"req_payload\":\"\",\"req_policy_group_id\":\"default-skynet-config\",\"req_proxy_name\":\"872e16e4dd6e\",\"req_risk_level\":\"medium\",\"req_rule_id\":\"m_file_upload\",\"req_rule_module\":\"m_file_upload\",\"req_start_time\":1617264570749,\"resp_body\":\"+++++++++\",\"resp_reason_phrase\":\"\",\"risk_level\":\"high\",\"rsp_end_time\":null,\"rsp_start_time\":null,\"rule_id\":\"m_file_upload\",\"scheme\":\"http\",\"session\":\"\",\"site_url\":\"http://10.240.19.181\",\"site_uuid\":\"default-site-config\",\"socket_ip\":\"10.240.67.9\",\"src_ip\":\"1.1.2.26\",\"src_port\":55482,\"timestamp\":1617264570,\"timestamp_human\":\"2021-04-01 16:09:30\",\"urlpath\":\"/xoda-0.4.3/?upload\",\"user_agent\":\"\"}",
                "err_code": 101,
                "err_code_str": "未命中规则",
                "err": "产生日志设备: 名称 master-1, 编号 6; 非标原因: 此设备未绑定任何规则\n"
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
| authorization | bearer token_value | 是       |   -   |   -   |
| x-menu-name | 84               | 是       |   -   |   -   |
| x-request-path | pedestal         | 是       |   -   |   -   |

**Body**

<table>
  <thead class="ant-table-thead">
    <tr>
      <th key=name>名称</th><th key=type>类型</th><th key=required>是否必须</th><th key=default>默认值</th><th key=desc>备注</th><th key=sub>其他信息</th>
    </tr>
  </thead><tbody className="ant-table-tbody"><tr key=0-0><td key=0><span style="padding-left: 0px"><span style="color: #8c8a8a"></span> method</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1><td key=0><span style="padding-left: 0px"><span style="color: #8c8a8a"></span> params</span></td><td key=1><span>object</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-0><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> offset</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">页</span></td><td key=5></td></tr><tr key=0-1-1><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> count</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">每页条数</span></td><td key=5></td></tr><tr key=0-1-2><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> key_words</span></td><td key=1><span>string []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">日志关键字筛选</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>string</span></p></td></tr><tr key=array-180><tr key=0-1-3><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> start_time</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">日志接收时间 开始</span></td><td key=5></td></tr><tr key=0-1-4><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> end_time</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">日志接收时间 结束</span></td><td key=5></td></tr><tr key=0-1-5><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> dev_type</span></td><td key=1><span>number []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">设备类型筛选</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>number</span></p></td></tr><tr key=array-181><tr key=0-1-6><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> log_id</span></td><td key=1><span>object []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">日志id筛选</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-1-6-0><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> oper</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-6-1><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> target</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-7><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> device_name</span></td><td key=1><span>object []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">设备名筛选</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-1-7-0><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> oper</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-7-1><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> target</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-8><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> agent_name</span></td><td key=1><span>object []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">探针名筛选</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-1-8-0><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> oper</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-8-1><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> target</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2><td key=0><span style="padding-left: 0px"><span style="color: #8c8a8a"></span> jsonrpc</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-3><td key=0><span style="padding-left: 0px"><span style="color: #8c8a8a"></span> id</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr>
               </tbody>
              </table>

### 返回数据

<table>
  <thead class="ant-table-thead">
    <tr>
      <th key=name>名称</th><th key=type>类型</th><th key=required>是否必须</th><th key=default>默认值</th><th key=desc>备注</th><th key=sub>其他信息</th>
    </tr>
  </thead><tbody className="ant-table-tbody"><tr key=0-0><td key=0><span style="padding-left: 0px"><span style="color: #8c8a8a"></span> jsonrpc</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1><td key=0><span style="padding-left: 0px"><span style="color: #8c8a8a"></span> result</span></td><td key=1><span>object</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-0><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> total</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">总数</span></td><td key=5></td></tr><tr key=0-1-1><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> data</span></td><td key=1><span>object []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-1-1-0><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> log_id</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">日志id</span></td><td key=5></td></tr><tr key=0-1-1-1><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> dev_id</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">设备id</span></td><td key=5></td></tr><tr key=0-1-1-2><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> device_name</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">设备名</span></td><td key=5></td></tr><tr key=0-1-1-3><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> dev_type</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">设备类型</span></td><td key=5></td></tr><tr key=0-1-1-4><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> agent_id</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">探针id</span></td><td key=5></td></tr><tr key=0-1-1-5><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> agent_name</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">探针名</span></td><td key=5></td></tr><tr key=0-1-1-6><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> received_time</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">日志接收时间</span></td><td key=5></td></tr><tr key=0-1-1-7><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> origin_log</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">原始日志</span></td><td key=5></td></tr><tr key=0-1-1-8><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> err_code</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">日志进入非标原因错误码</span></td><td key=5></td></tr><tr key=0-1-1-9><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> err_code_str</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">日志进入非标原因</span></td><td key=5></td></tr><tr key=0-1-1-10><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> err</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">日志进入非标错误详情</span></td><td key=5></td></tr><tr key=0-2><td key=0><span style="padding-left: 0px"><span style="color: #8c8a8a"></span> id</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr>
               </tbody>
              </table>
<!-- endtoc -->
