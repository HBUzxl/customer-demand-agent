---
type: product
title: 万象AISOC-工单闭环
product: 万象AISOC
aliases: []
tags:
    - 安全平台
    - 产品技术手册
    - 接口文档
    - 工单
summary: 介绍工单闭环相关的接口能力、请求方式与使用说明。
status: verified
---

<!-- toc -->

<a id=完成相关执行动作657> </a>

### 基本信息

**Path：** https://ip_addr/workflow/activiti/task/done

**Method：** POST

**接口描述：**
<h3>请求</h3>
<p>请求示例</p>
<pre><code>curl -k --request POST \
  --url https://k3s.alpha2.umb.staging.dev.in.chaitin.net/workflow/activiti/task/done \
  --header 'Connection: close' \
  --header 'Host: workflow.default.svc.cluster.local:8080' \
  --header 'accept: application/json' \
  --header 'accept-encoding: gzip, deflate, br' \
  --header 'content-length: 251' \
  --header 'content-type: application/json' \
  --header 'cookie: sessionid=6cd4ecb9036846d4b0869ebd2f5fe2ab;jwt=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3MTk5OTE0MTAsImlhdCI6MTcxOTM5MTE3MCwiaXNzIjoicGVkZXN0YWwiLCJ1c2VyX2lkIjoxLCJ1c2VybmFtZSI6IiJ9._kZrofny5SObFNxSuD4MmYlaO7O0AyaF5FJg5hNXvh4' \
  --header 'traceparent: 00-e6b9184a5d66b57d444d6567fe70a7e6-29b0bdbf1a498a65-01' \
  --header 'x-request-path: pedestal' \
  --data '{"task_data":{"files":[],"remark":"备注","tidy_action":"","exec_explain":"执行说明"},"node_id":"d2e33ea9-339b-11ef-9b01-0e08850fc4aa","remark":"节点备注","task_id":6,"user_id":1,"user_follows":[],"exec_user":{},"task_result":"完成1111111"}'
</code></pre>
<p>注: 此接口请求的 node_id 和获取工单列表接口的 nodeId 是关联的</p>
<h3>响应</h3>
<pre><code>响应示例
{
  "code": 200,
  "msg": "success",
  "data": {
    "status": true
  }
}
</code></pre>


### 请求参数
**Headers**

| 参数名称  | 参数值              |  是否必须 | 示例  | 备注  |
| ------------ |------------------| ------------ | ------------ | ------------ |
| Content-Type  | application/json | 是  | -  |  -  |
| authorization | bearer token_value | 是       | -  | -  |
| x-menu-name | 46               | 是       | -  | -  |
| x-request-path | pedestal         | 是       | -  | -  |
**Body**

<table>
  <thead class="ant-table-thead">
    <tr>
      <th key=name>名称</th><th key=type>类型</th><th key=required>是否必须</th><th key=default>默认值</th><th key=desc>备注</th><th key=sub>其他信息</th>
    </tr>
  </thead><tbody className="ant-table-tbody"><tr key=0-0><td key=0><span style="padding-left: 0px"><span style="color: #8c8a8a"></span> data</span></td><td key=1><span>object</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">业务数据-工单数据-工单关联的业务数据</span></td><td key=5></td></tr><tr key=0-1><td key=0><span style="padding-left: 0px"><span style="color: #8c8a8a"></span> exec_user</span></td><td key=1><span>object</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">指定节点的执行用户</span></td><td key=5></td></tr><tr key=0-2><td key=0><span style="padding-left: 0px"><span style="color: #8c8a8a"></span> node_id</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">执行任务 id</span></td><td key=5></td></tr><tr key=0-3><td key=0><span style="padding-left: 0px"><span style="color: #8c8a8a"></span> notice_content</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">通知内容</span></td><td key=5></td></tr><tr key=0-4><td key=0><span style="padding-left: 0px"><span style="color: #8c8a8a"></span> remark</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">remark</span></td><td key=5></td></tr><tr key=0-5><td key=0><span style="padding-left: 0px"><span style="color: #8c8a8a"></span> task_data</span></td><td key=1><span>object</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">工单内容，流程数据,流程中的必填参数</span></td><td key=5></td></tr><tr key=0-6><td key=0><span style="padding-left: 0px"><span style="color: #8c8a8a"></span> task_id</span></td><td key=1><span>integer</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">工单id</span></td><td key=5><p key=2><span style="font-weight: '700'">format: </span><span>int64</span></p></td></tr><tr key=0-7><td key=0><span style="padding-left: 0px"><span style="color: #8c8a8a"></span> task_result</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">工单结果</span></td><td key=5></td></tr><tr key=0-8><td key=0><span style="padding-left: 0px"><span style="color: #8c8a8a"></span> user_follows</span></td><td key=1><span>integer []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">关注用户</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>integer</span></p></td></tr><tr key=array-800><tr key=0-9><td key=0><span style="padding-left: 0px"><span style="color: #8c8a8a"></span> user_id</span></td><td key=1><span>integer</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">用户 id</span></td><td key=5><p key=2><span style="font-weight: '700'">format: </span><span>int64</span></p></td></tr>
               </tbody>
              </table>

### 返回数据

<table>
  <thead class="ant-table-thead">
    <tr>
      <th key=name>名称</th><th key=type>类型</th><th key=required>是否必须</th><th key=default>默认值</th><th key=desc>备注</th><th key=sub>其他信息</th>
    </tr>
  </thead><tbody className="ant-table-tbody"><tr key=0-0><td key=0><span style="padding-left: 0px"><span style="color: #8c8a8a"></span> code</span></td><td key=1><span>integer</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">状态码,成功为200,其他为异常</span></td><td key=5><p key=2><span style="font-weight: '700'">format: </span><span>int32</span></p></td></tr><tr key=0-1><td key=0><span style="padding-left: 0px"><span style="color: #8c8a8a"></span> data</span></td><td key=1><span>object</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">接口响应数据</span></td><td key=5></td></tr><tr key=0-2><td key=0><span style="padding-left: 0px"><span style="color: #8c8a8a"></span> msg</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">状态信息，成功为success,其他情况显示异常信息</span></td><td key=5></td></tr>
               </tbody>
              </table>

<!-- endtoc -->
