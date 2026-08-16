---
type: product
title: 万象AISOC-获取日志聚合
product: 万象AISOC
aliases: []
tags:
    - 安全平台
    - 产品技术手册
    - 接口文档
    - 日志
summary: 介绍获取日志聚合相关的接口能力、请求方式与使用说明。
status: verified
---

<!-- toc -->

<a id=日志聚合3966> </a>

### 基本信息

**Path：** https://ip_addr/pedestal/rpc

**Method：** POST

**接口描述：**

<h3>请求</h3>
<pre><code>{
    "method": "LogService.SearchAggregationStatistics",
    "params": {
        "keyword": [
            "2.1.1.72"
        ],
        "time_range_start": 1729154011,
        "time_range_end": 1729154311,
        "attack_chain_phase": null,
        "fall": null,
        "advanced_query": null,
        "condition_query": {
            "logical_op": "AND",
            "expressions": [
                {
                    "column": "src_ip",
                    "op": "equal",
                    "value": "2.1.1.72"
                }
            ]
        },
        "organization": [
            {
                "oper": "=",
                "target": 1
            },
            {
                "oper": "=",
                "target": 34
            }
        ],
        "filter": {
            "origin_event_name": null,
            "src_ip": null,
            "dest_ip": null,
            "src_country": null,
            "src_port": null,
            "dest_port": null,
            "attack_result": null
        },
        "key": [
            "src_ip",
            "dest_ip",
            "event_type"
        ],
        "count": 10,
        "asc": false
    },
    "jsonrpc": "2.0",
    "id": "0"
}
</code></pre>
<h3>响应</h3>
<pre><code>{
    "jsonrpc": "2.0",
    "result": {
        "data": [
            {
                "result": {
                    "dest_ip": "123.101.10.110",
                    "event_type": 52001,
                    "src_ip": "2.1.1.72"
                },
                "data": [
                    {
                        "start_time": 1729154011,
                        "count": 0
                    },
                    {
                        "start_time": 1729154026,
                        "count": 0
                    },
                    {
                        "start_time": 1729154041,
                        "count": 0
                    },
                    {
                        "start_time": 1729154056,
                        "count": 1
                    },
                    {
                        "start_time": 1729154071,
                        "count": 0
                    },
                    {
                        "start_time": 1729154086,
                        "count": 0
                    },
                    {
                        "start_time": 1729154101,
                        "count": 0
                    },
                    {
                        "start_time": 1729154116,
                        "count": 0
                    },
                    {
                        "start_time": 1729154131,
                        "count": 0
                    },
                    {
                        "start_time": 1729154146,
                        "count": 0
                    },
                    {
                        "start_time": 1729154161,
                        "count": 0
                    },
                    {
                        "start_time": 1729154176,
                        "count": 0
                    },
                    {
                        "start_time": 1729154191,
                        "count": 0
                    },
                    {
                        "start_time": 1729154206,
                        "count": 0
                    },
                    {
                        "start_time": 1729154221,
                        "count": 0
                    },
                    {
                        "start_time": 1729154236,
                        "count": 0
                    },
                    {
                        "start_time": 1729154251,
                        "count": 0
                    },
                    {
                        "start_time": 1729154266,
                        "count": 0
                    },
                    {
                        "start_time": 1729154281,
                        "count": 0
                    },
                    {
                        "start_time": 1729154296,
                        "count": 0
                    }
                ],
                "count": 1
            }
        ],
        "total": 1
    },
    "id": "0"
}
</code></pre>

### 请求参数

**Headers**

| 参数名称     | 参数值           | 是否必须 | 示例 | 备注 |
| ------------ | ---------------- | -------- | ---- | ---- |
| Content-Type | application/json | 是       |   -   |   -   |
| authorization | bearer token_value | 是       | -  | -  |
| x-menu-name | 31                | 是       | -  | -  |
| x-request-path | pedestal        | 是       | -  | -  |

**Body**

<table>
  <thead class="ant-table-thead">
    <tr>
      <th key=name>名称</th><th key=type>类型</th><th key=required>是否必须</th><th key=default>默认值</th><th key=desc>备注</th><th key=sub>其他信息</th>
    </tr>
  </thead><tbody className="ant-table-tbody"><tr key=0-0><td key=0><span style="padding-left: 0px"><span style="color: #8c8a8a"></span> method</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1><td key=0><span style="padding-left: 0px"><span style="color: #8c8a8a"></span> params</span></td><td key=1><span>object</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-0><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> keyword</span></td><td key=1><span>string []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">模糊关键字匹配</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>string</span></p></td></tr><tr key=array-186><tr key=0-1-1><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> time_range_start</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">时间范围开始</span></td><td key=5></td></tr><tr key=0-1-2><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> time_range_end</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">时间范围结束</span></td><td key=5></td></tr><tr key=0-1-3><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> condition_query</span></td><td key=1><span>object</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">条件筛选</span></td><td key=5></td></tr><tr key=0-1-3-0><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> logical_op</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-3-1><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> expressions</span></td><td key=1><span>object []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-1-3-1-0><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> column</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-3-1-1><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> op</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-3-1-2><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> value</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-4><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> organization</span></td><td key=1><span>object []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">组织机构筛选</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-1-4-0><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> oper</span></td><td key=1><span>string</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-4-1><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> target</span></td><td key=1><span>number</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-5><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> key</span></td><td key=1><span>string []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">聚合字段配置</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>string</span></p></td></tr><tr key=array-187><tr key=0-1-6><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> count</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">聚合数量</span></td><td key=5></td></tr><tr key=0-1-7><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> asc</span></td><td key=1><span>boolean</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">排序方式 false：降序 true：升序</span></td><td key=5></td></tr><tr key=0-2><td key=0><span style="padding-left: 0px"><span style="color: #8c8a8a"></span> jsonrpc</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-3><td key=0><span style="padding-left: 0px"><span style="color: #8c8a8a"></span> id</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr>
               </tbody>
              </table>

### 返回数据

<table>
  <thead class="ant-table-thead">
    <tr>
      <th key=name>名称</th><th key=type>类型</th><th key=required>是否必须</th><th key=default>默认值</th><th key=desc>备注</th><th key=sub>其他信息</th>
    </tr>
  </thead><tbody className="ant-table-tbody"><tr key=0-0><td key=0><span style="padding-left: 0px"><span style="color: #8c8a8a"></span> jsonrpc</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1><td key=0><span style="padding-left: 0px"><span style="color: #8c8a8a"></span> result</span></td><td key=1><span>object</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-0><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> data</span></td><td key=1><span>object []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-1-0-0><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> result</span></td><td key=1><span>object</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-0-0-0><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> dest_ip</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">目的ip</span></td><td key=5></td></tr><tr key=0-1-0-0-1><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> event_type</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">事件类型</span></td><td key=5></td></tr><tr key=0-1-0-0-2><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> src_ip</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">源ip</span></td><td key=5></td></tr><tr key=0-1-0-1><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> data</span></td><td key=1><span>object []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-1-0-1-0><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> start_time</span></td><td key=1><span>number</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">开始时间</span></td><td key=5></td></tr><tr key=0-1-0-1-1><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> count</span></td><td key=1><span>number</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">数量</span></td><td key=5></td></tr><tr key=0-1-0-2><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> count</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-1><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> total</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">总数</span></td><td key=5></td></tr><tr key=0-2><td key=0><span style="padding-left: 0px"><span style="color: #8c8a8a"></span> id</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr>
               </tbody>
              </table>
<!-- endtoc -->
