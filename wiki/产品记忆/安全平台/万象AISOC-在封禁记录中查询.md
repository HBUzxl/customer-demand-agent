---
type: product
title: 万象AISOC-在封禁记录中查询
product: 万象AISOC
aliases: []
tags:
    - 安全平台
    - 产品技术手册
    - 接口文档
    - IP封禁
summary: 介绍在封禁记录中查询相关的接口能力、请求方式与使用说明。
status: verified
---

<!-- toc -->
<a id=查找已封禁列表1827> </a>

### 基本信息

**Path：** https://ip_addr/pedestal/rpc

**Method：** POST

**接口描述：**

<p><strong>请求</strong></p>
<pre><code>{
  "jsonrpc": "2.0",
  "id": "0",
  "method": "IpBlockService.SearchBlockedList",
  "params": {
    "filter": {
      "ip": [{"oper": "like", "target": "1.1.1.1"}],
      "dev_id": [{"oper": "=", "target": "a1177a69-d198-4d21-8c56-e9c2a55e4e8c"}],
      "block_time": [{"oper": "in", "target": "1686710379000-1686720379000"}],
      "block_mode": [{"oper": "in", "target": "1686710379000-1686720379000"}],
      "organization_id": [1]
    },
    "count": 10
  }
}
</code></pre>
<p><strong>响应</strong></p>
<pre><code>{
  "jsonrpc": "2.0",
  "result": {
    "data": [
      {
        "ip": "1.1.1.2",
        "dev_id": "a1177a69-d198-4d21-8c56-e9c2a55e4e8c",
        "dev_name": "device3",
        "remark": "这是二个测试",
        "create_user_id": 1,
        "create_user": "admin",
        "create_time": 1686722427.038666,
        "block_time": 1686722427.038666,
        "unblock_time": 1686722427.038666,
        "block_mode": "自动化",
        "unblock_mode": "自动化",
        "unblock_remark": ""
      },
      {
        "ip": "1.1.1.4",
        "dev_id": "a1177a69-d198-4d21-8c56-e9c2a55e4e8c",
        "dev_name": "device3",
        "remark": "这是一个测试",
        "create_user_id": 1,
        "create_user": "admin",
        "create_time": 1686722427.037082,
        "block_time": 1686722427.037082,
        "unblock_time": 1686722427.037082,
        "block_mode": "人工",
        "unblock_mode": "人工",
        "unblock_remark": ""
      },
      {
        "ip": "1.1.1.3",
        "dev_id": "a1177a69-d198-4d21-8c56-e9c2a55e4e8c",
        "dev_name": "device3",
        "remark": "这是一个测试",
        "create_user_id": 1,
        "create_user": "admin",
        "create_time": 1686722427.035522,
        "block_time": 1686722427.035522,
        "unblock_time": 1686722427.035522,
        "block_mode": "人工",
        "unblock_mode": "人工",
        "unblock_remark": ""
      },
      {
        "ip": "1.1.1.1",
        "dev_id": "a1177a69-d198-4d21-8c56-e9c2a55e4e8c",
        "dev_name": "device3",
        "remark": "这是一个测试",
        "create_user_id": 1,
        "create_user": "admin",
        "create_time": 1686720215.346006,
        "block_time": 1686720215.346006,
        "unblock_time": 1686720215.346006,
        "block_mode": "人工",
        "unblock_mode": "人工",
        "unblock_remark": ""
      },
      {
        "ip": "1.1.1.21",
        "dev_id": "a1177a69-d198-4d21-8c56-e9c2a55e4e8c",
        "dev_name": "device3",
        "remark": "这是二个测试",
        "create_user_id": 1,
        "create_user": "admin",
        "create_time": 1686716349.367927,
        "block_time": 1686716349.367927,
        "unblock_time": 1686716349.367927,
        "block_mode": "自动化",
        "unblock_mode": "自动化",
        "unblock_remark": ""
      },
      {
        "ip": "1.1.1.41",
        "dev_id": "a1177a69-d198-4d21-8c56-e9c2a55e4e8c",
        "dev_name": "device3",
        "remark": "这是一个测试",
        "create_user_id": 1,
        "create_user": "admin",
        "create_time": 1686716349.366759,
        "block_time": 1686716349.366759,
        "unblock_time": 1686716349.366759,
        "block_mode": "人工",
        "unblock_mode": "人工",
        "unblock_remark": ""
      },
      {
        "ip": "1.1.1.31",
        "dev_id": "a1177a69-d198-4d21-8c56-e9c2a55e4e8c",
        "dev_name": "device3",
        "remark": "这是一个测试",
        "create_user_id": 1,
        "create_user": "admin",
        "create_time": 1686716349.365172,
        "block_time": 1686716349.365172,
        "unblock_time": 1686716349.365172,
        "block_mode": "人工",
        "unblock_mode": "人工",
        "unblock_remark": ""
      },
      {
        "ip": "1.1.1.11",
        "dev_id": "a1177a69-d198-4d21-8c56-e9c2a55e4e8c",
        "dev_name": "device3",
        "remark": "这是一个测试",
        "create_user_id": 1,
        "create_user": "admin",
        "create_time": 1686716349.361347,
        "block_time": 1686716349.361347,
        "unblock_time": 1686716349.361347,
        "block_mode": "人工",
        "unblock_mode": "人工",
        "unblock_remark": ""
      }
    ],
    "total": 8
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
| x-menu-name | 42               | 是       | -  | -  |
| x-request-path | pedestal         | 是       | -  | -  |

**Body**

<table>
  <thead class="ant-table-thead">
    <tr>
      <th key=name>名称</th><th key=type>类型</th><th key=required>是否必须</th><th key=default>默认值</th><th key=desc>备注</th><th key=sub>其他信息</th>
    </tr>
  </thead><tbody className="ant-table-tbody"><tr key=0-0><td key=0><span style="padding-left: 0px"><span style="color: #8c8a8a"></span> jsonrpc</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1><td key=0><span style="padding-left: 0px"><span style="color: #8c8a8a"></span> id</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2><td key=0><span style="padding-left: 0px"><span style="color: #8c8a8a"></span> method</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-3><td key=0><span style="padding-left: 0px"><span style="color: #8c8a8a"></span> params</span></td><td key=1><span>object</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-3-0><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> filter</span></td><td key=1><span>object</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-3-0-0><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> dev_id</span></td><td key=1><span>object []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">设备id筛选</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-3-0-0-0><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> oper</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">操作符 =</span></td><td key=5></td></tr><tr key=0-3-0-0-1><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> target</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">操作值</span></td><td key=5></td></tr><tr key=0-3-0-1><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> ip</span></td><td key=1><span>object []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">ip筛选</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-3-0-1-0><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> oper</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">操作符 like</span></td><td key=5></td></tr><tr key=0-3-0-1-1><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> target</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">操作值</span></td><td key=5></td></tr><tr key=0-3-0-2><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> block_mode</span></td><td key=1><span>object []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">封禁类型筛选</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-3-0-2-0><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> oper</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">操作符 =</span></td><td key=5></td></tr><tr key=0-3-0-2-1><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> target</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">操作值</span></td><td key=5></td></tr><tr key=0-3-0-3><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> block_time</span></td><td key=1><span>object []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">封禁时间筛选</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-3-0-3-0><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> oper</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">操作符 in</span></td><td key=5></td></tr><tr key=0-3-0-3-1><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> target</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">操作值 时间戳范围</span></td><td key=5></td></tr><tr key=0-3-0-4><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> organization_id</span></td><td key=1><span>number []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">组织机构id</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>number</span></p></td></tr><tr key=array-18><tr key=0-3-1><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> count</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr>
               </tbody>
              </table>

### 返回数据

<table>
  <thead class="ant-table-thead">
    <tr>
      <th key=name>名称</th><th key=type>类型</th><th key=required>是否必须</th><th key=default>默认值</th><th key=desc>备注</th><th key=sub>其他信息</th>
    </tr>
  </thead><tbody className="ant-table-tbody"><tr key=0-0><td key=0><span style="padding-left: 0px"><span style="color: #8c8a8a"></span> jsonrpc</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1><td key=0><span style="padding-left: 0px"><span style="color: #8c8a8a"></span> result</span></td><td key=1><span>object</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-0><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> data</span></td><td key=1><span>object []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-1-0-0><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> ip</span></td><td key=1><span>string</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">ip</span></td><td key=5></td></tr><tr key=0-1-0-1><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> dev_id</span></td><td key=1><span>string</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">设备id</span></td><td key=5></td></tr><tr key=0-1-0-2><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> dev_name</span></td><td key=1><span>string</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">设备名</span></td><td key=5></td></tr><tr key=0-1-0-3><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> remark</span></td><td key=1><span>string</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">备注</span></td><td key=5></td></tr><tr key=0-1-0-4><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> create_user_id</span></td><td key=1><span>number</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">创建用户id</span></td><td key=5></td></tr><tr key=0-1-0-5><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> create_user</span></td><td key=1><span>string</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">创建用户名</span></td><td key=5></td></tr><tr key=0-1-0-6><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> create_time</span></td><td key=1><span>number</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">记录创建时间</span></td><td key=5></td></tr><tr key=0-1-0-7><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> block_time</span></td><td key=1><span>number</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">封禁时间</span></td><td key=5></td></tr><tr key=0-1-0-8><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> unblock_time</span></td><td key=1><span>number</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">解禁时间</span></td><td key=5></td></tr><tr key=0-1-0-9><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> block_mode</span></td><td key=1><span>string</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">封禁类型</span></td><td key=5></td></tr><tr key=0-1-0-10><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> unblock_mode</span></td><td key=1><span>string</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">解禁类型</span></td><td key=5></td></tr><tr key=0-1-0-11><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> unblock_remark</span></td><td key=1><span>string</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">解禁备注</span></td><td key=5></td></tr><tr key=0-1-1><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> total</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">总数</span></td><td key=5></td></tr><tr key=0-2><td key=0><span style="padding-left: 0px"><span style="color: #8c8a8a"></span> id</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr>
               </tbody>
              </table>
<!-- endtoc -->
