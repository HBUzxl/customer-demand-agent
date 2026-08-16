---
type: product
title: 万象AISOC-获取soar列表
product: 万象AISOC
aliases: []
tags:
    - 安全平台
    - 产品技术手册
    - 接口文档
    - soar
summary: 介绍获取soar列表相关的接口能力、请求方式与使用说明。
status: verified
---

<!-- toc -->

### 基本信息

<a id=获取soar列表3686> </a>

**Path：** https://ip_addr/pedestal/rpc

**Method：** POST

**接口描述：**
<h3>请求参数：</h3>
<pre><code>{
    "method": "AnalysisService.GetSOARPlaybookList",
    "params": {
    },
    "jsonrpc": "2.0",
    "id": "0"
}
</code></pre>
<h3>响应参数：</h3>
<pre><code>{
    "jsonrpc": "2.0",
    "result": {
        "playbooks": [
            {
                "uuid": "87dcc932fcad4899a8e7ba6eaa961c2c",
                "displayName": "测试",
                "params": []
            },
            {
                "uuid": "f5d741b4d18642b28e95dee5dfcb532d",
                "displayName": "文件情报自动化通报",
                "params": [
                    {
                        "name": "alarm_id",
                        "dataType": "string",
                        "description": "告警ID",
                        "required": true
                    },
                    {
                        "name": "mode",
                        "dataType": "string",
                        "description": "1:手动通报;2:规则Push;3:自动化通知;4:汤臣倍健特殊通知(企业微信+邮件)(默认为1)",
                        "required": false
                    },
                    {
                        "name": "action_descs",
                        "dataType": "string",
                        "description": "告警处置建议（默认大模型填写）",
                        "required": false
                    },
                    {
                        "name": "action_result",
                        "dataType": "string",
                        "description": "攻击结果：默认未成功，其他可选“成功/疑似成功待排查”",
                        "required": false
                    }
                ]
            },
            {
                "uuid": "edf2e83ef8bf4b5598679d859dab08f1",
                "displayName": "【AI】弱口令AI研判流",
                "params": [
                    {
                        "name": "alarm_id",
                        "dataType": "string",
                        "description": "告警ID",
                        "required": true
                    }
                ]
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
| x-menu-name | 71               | 是       | -  | -  |
| x-request-path | pedestal       | 是       | -  | -  |
**Body**

<table>
  <thead class="ant-table-thead">
    <tr>
      <th key=name>名称</th><th key=type>类型</th><th key=required>是否必须</th><th key=default>默认值</th><th key=desc>备注</th><th key=sub>其他信息</th>
    </tr>
  </thead><tbody className="ant-table-tbody"><tr key=0-0><td key=0><span style="padding-left: 0px"><span style="color: #8c8a8a"></span> method</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1><td key=0><span style="padding-left: 0px"><span style="color: #8c8a8a"></span> params</span></td><td key=1><span>object</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2><td key=0><span style="padding-left: 0px"><span style="color: #8c8a8a"></span> jsonrpc</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-3><td key=0><span style="padding-left: 0px"><span style="color: #8c8a8a"></span> id</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr>
               </tbody>
              </table>

### 返回数据

<table>
  <thead class="ant-table-thead">
    <tr>
      <th key=name>名称</th><th key=type>类型</th><th key=required>是否必须</th><th key=default>默认值</th><th key=desc>备注</th><th key=sub>其他信息</th>
    </tr>
  </thead><tbody className="ant-table-tbody"><tr key=0-0><td key=0><span style="padding-left: 0px"><span style="color: #8c8a8a"></span> jsonrpc</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1><td key=0><span style="padding-left: 0px"><span style="color: #8c8a8a"></span> result</span></td><td key=1><span>object</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-1-0><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> playbooks</span></td><td key=1><span>object []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-1-0-0><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> uuid</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">soar唯一Id</span></td><td key=5></td></tr><tr key=0-1-0-1><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> displayName</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">soar中文名</span></td><td key=5></td></tr><tr key=0-1-0-2><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> params</span></td><td key=1><span>object []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">参数描述</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-1-0-2-0><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> name</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">参数名</span></td><td key=5></td></tr><tr key=0-1-0-2-1><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> dataType</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">参数类型</span></td><td key=5></td></tr><tr key=0-1-0-2-2><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> description</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">参数中文描述</span></td><td key=5></td></tr><tr key=0-1-0-2-3><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> required</span></td><td key=1><span>boolean</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">是否必须</span></td><td key=5></td></tr><tr key=0-2><td key=0><span style="padding-left: 0px"><span style="color: #8c8a8a"></span> id</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr>
               </tbody>
              </table>

<!-- endtoc -->
