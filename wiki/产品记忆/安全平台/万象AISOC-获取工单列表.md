---
type: product
title: 万象AISOC-获取工单列表
product: 万象AISOC
aliases: []
tags:
    - 安全平台
    - 产品技术手册
    - 接口文档
    - 工单
summary: 介绍获取工单列表相关的接口能力、请求方式与使用说明。
status: verified
---

<!-- toc -->

<a id=根据条件获取工单列表3578> </a>

### 基本信息

**Path：** https://ip_addr/workflow/activiti/list

**Method：** POST

**接口描述：**
<h3>请求</h3>
<p>请求示例 (根据当前步骤筛选)</p>
<pre><code>curl -k --request POST \
  --url 'https://k3s.alpha2.umb.staging.dev.in.chaitin.net/workflow/activiti/list?current=1&amp;size=20&amp;type=my&amp;user_id=1&amp;now_node_name=1111111' \
  --header 'Connection: close' \
  --header 'Content-Length: 0' \
  --header 'Host: workflow.default.svc.cluster.local:8080' \
  --header 'accept: application/json' \
  --header 'accept-encoding: gzip, deflate, br' \
  --header 'cookie: sessionid=6cd4ecb9036846d4b0869ebd2f5fe2ab;jwt=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3MTk5OTE0MTAsImlhdCI6MTcxOTM5MTE3MCwiaXNzIjoicGVkZXN0YWwiLCJ1c2VyX2lkIjoxLCJ1c2VybmFtZSI6IiJ9._kZrofny5SObFNxSuD4MmYlaO7O0AyaF5FJg5hNXvh4' \
  --header 'traceparent: 00-fcb7096810f760ee27918d163d919f48-f82d51935f3d07a8-01' \
  --header 'user-agent: got (https://github.com/sindresorhus/got)' \
  --header 'x-request-path: pedestal'
</code></pre>
<p>注: k3s.alpha2.umb.staging.dev.in.chaitin.net 换成客户环境管理端 IP</p>
<h3>响应</h3>
<pre><code>{
  "code": 200,
  "msg": "success",
  "data": {
    "total": 1,
    "list": [
      {
        "taskId": 8,
        "taskName": "工单名称",
        "nodeName": "1111111",
        "nodeUser": 1,
        "nodeId": "29b67f49-339e-11ef-9b01-0e08850fc4aa",
        "activitiId": "Activity_02f3whv",
        "beforeNode": [
          {
            "date": "2024-06-26 17:26:26",
            "desc": "开始",
            "action": "START",
            "assignee": 1,
            "taskName": "开始",
            "activitiId": "Event_1ifn5hf",
            "taskResult": "创建工单"
          }
        ],
        "nowNode": [
          {
            "taskId": "29b67f49-339e-11ef-9b01-0e08850fc4aa",
            "assignee": 0,
            "taskName": "1111111",
            "activitiId": "Activity_02f3whv",
            "candidateUser": [
              "1",
              "3",
              "4"
            ]
          }
        ],
        "nextNode": [],
        "priority": "MEDIUM",
        "taskResult": "创建工单",
        "startWay": "人工触发",
        "startUser": 1,
        "processDefinitionId": "f44b47ff-3396-11ef-9b01-0e08850fc4aa",
        "processDefinitionKey": "csadasda",
        "relationDataType": "NO_LINK",
        "relationData": [],
        "doneNode": [
          {
            "date": "2024-06-26 17:26:26",
            "desc": "开始",
            "action": "START",
            "assignee": 1,
            "taskName": "开始",
            "activitiId": "Event_1ifn5hf",
            "taskResult": "创建工单"
          }
        ],
        "labels": "",
        "taskDesc": "",
        "ctime": "2024-06-26T09:26:26.993+00:00",
        "doneUser": [
          1
        ],
        "platDate": "1719998784777",
        "userId": 1,
        "execStatus": "RUNNING",
        "remark": {
          "notice_note": "您好：\n请您关注工单：{工单名称}，当前待完成：{节点名称}",
          "notice_status": "false"
        },
        "executeInfo": [
          {
            "date": "2024-06-26 17:26:26",
            "desc": "开始",
            "action": "START",
            "assignee": 1,
            "taskName": "开始",
            "activitiId": "Event_1ifn5hf",
            "taskResult": "创建工单"
          }
        ],
        "taskContent": {
          "files": [],
          "remark": "备注",
          "tidy_action": "整理操作"
        },
        "userFollows": [],
        "timeOutStatus": false,
        "tagNames": []
      }
    ]
  }
}
</code></pre>


### 请求参数
**Headers**

| 参数名称  | 参数值              |  是否必须 | 示例  | 备注 |
| ------------ |------------------| ------------ | ------------ |----|
| Content-Type  | application/json | 是  | -  | -  |
| authorization | bearer token_value | 是       | -  | -  |
| x-menu-name | 46               | 是       | -  | -  |
| x-request-path | pedestal         | 是       | -  | -  |

**Query**

| 参数名称  |  是否必须 | 示例  | 备注  |
| ------------ | ------------ | ------------ | ------------ |
| create_end | 否  |   |  工单创建时间段，结束时间 |
| create_start | 否  |   |  工单创建时间段，开始时间 |
| current | 否  |   |  当前页数,默认首页 |
| data_info | 否  |   |  关联数据,  NO_LINK = 未关联, SECURITY_ALARM = 安全告警,VULNERABILITY_EVENT = 漏洞事件, CONFIRMED_ASSET = 确认资产, UNKNOWN_ASSET = 未知资产 |
| exec_user_id | 否  |   |  执行用户 ID |
| now_node_name | 否  |   |  当前步骤名称 |
| priority | 否  |   |  优先级 |
| size | 否  |   |  每页数量，默认10 |
| tag_names | 否  |   |  标签名称列表 |
| task_id | 否  |   |  工单id |
| task_name | 否  |   |  task_name |
| task_status | 否  |   |  工单状态: CLOSE = 已关闭, RUNNING = 运行中, END = 已完成, NOT_START = 未开始, PROCESSING = 处理中, TIMEOUT = 超时 (仅在筛选入参中使用) |
| time_out | 否  |   |  是否超时 |
| type | 否  |   |  my=我的,follow=关注,history=历史,create=我创建的 |
| user_id | 否  |   |  用户id |
**Body**

<table>
  <thead class="ant-table-thead">
    <tr>
      <th key=name>名称</th><th key=type>类型</th><th key=required>是否必须</th><th key=default>默认值</th><th key=desc>备注</th><th key=sub>其他信息</th>
    </tr>
  </thead><tbody className="ant-table-tbody">
               </tbody>
              </table>

### 返回数据

<table>
  <thead class="ant-table-thead">
    <tr>
      <th key=name>名称</th><th key=type>类型</th><th key=required>是否必须</th><th key=default>默认值</th><th key=desc>备注</th><th key=sub>其他信息</th>
    </tr>
  </thead><tbody className="ant-table-tbody"><tr key=0-0><td key=0><span style="padding-left: 0px"><span style="color: #8c8a8a"></span> code</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">状态码</span></td><td key=5></td></tr><tr key=0-1><td key=0><span style="padding-left: 0px"><span style="color: #8c8a8a"></span> msg</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">状态信息</span></td><td key=5></td></tr><tr key=0-2><td key=0><span style="padding-left: 0px"><span style="color: #8c8a8a"></span> data</span></td><td key=1><span>object</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">响应数据</span></td><td key=5></td></tr><tr key=0-2-0><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> total</span></td><td key=1><span>number</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">总数</span></td><td key=5></td></tr><tr key=0-2-1><td key=0><span style="padding-left: 20px"><span style="color: #8c8a8a">├─</span> list</span></td><td key=1><span>object []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">列表数据</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-2-1-0><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> taskId</span></td><td key=1><span>number</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">工单id</span></td><td key=5></td></tr><tr key=0-2-1-1><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> taskName</span></td><td key=1><span>string</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">工单名称</span></td><td key=5></td></tr><tr key=0-2-1-2><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> nodeName</span></td><td key=1><span>string</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">节点名称</span></td><td key=5></td></tr><tr key=0-2-1-3><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> nodeUser</span></td><td key=1><span>number</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">节点用户</span></td><td key=5></td></tr><tr key=0-2-1-4><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> nodeId</span></td><td key=1><span>string</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">节点id</span></td><td key=5></td></tr><tr key=0-2-1-5><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> activitiId</span></td><td key=1><span>string</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">工单activiti Id</span></td><td key=5></td></tr><tr key=0-2-1-6><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> beforeNode</span></td><td key=1><span>object []</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">上一个节点</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-2-1-6-0><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> date</span></td><td key=1><span>string</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2-1-6-1><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> desc</span></td><td key=1><span>string</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2-1-6-2><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> action</span></td><td key=1><span>string</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2-1-6-3><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> assignee</span></td><td key=1><span>number</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2-1-6-4><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> taskName</span></td><td key=1><span>string</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2-1-6-5><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> activitiId</span></td><td key=1><span>string</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2-1-6-6><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> taskResult</span></td><td key=1><span>string</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2-1-7><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> nowNode</span></td><td key=1><span>object []</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">当前节点</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-2-1-7-0><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> taskId</span></td><td key=1><span>string</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2-1-7-1><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> assignee</span></td><td key=1><span>number</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2-1-7-2><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> taskName</span></td><td key=1><span>string</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2-1-7-3><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> activitiId</span></td><td key=1><span>string</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2-1-7-4><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> candidateUser</span></td><td key=1><span>string []</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>string</span></p></td></tr><tr key=array-809><tr key=0-2-1-8><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> nextNode</span></td><td key=1><span>string []</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">下一个节点</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>string</span></p></td></tr><tr key=array-810><tr key=0-2-1-9><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> priority</span></td><td key=1><span>string</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">优先级</span></td><td key=5></td></tr><tr key=0-2-1-10><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> taskResult</span></td><td key=1><span>string</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">工单结果</span></td><td key=5></td></tr><tr key=0-2-1-11><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> startWay</span></td><td key=1><span>string</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">触发方式</span></td><td key=5></td></tr><tr key=0-2-1-12><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> startUser</span></td><td key=1><span>number</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">触发用户</span></td><td key=5></td></tr><tr key=0-2-1-13><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> processDefinitionId</span></td><td key=1><span>string</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">模板 id</span></td><td key=5></td></tr><tr key=0-2-1-14><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> processDefinitionKey</span></td><td key=1><span>string</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">模板名</span></td><td key=5></td></tr><tr key=0-2-1-15><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> relationDataType</span></td><td key=1><span>string</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">流程关联数据类型</span></td><td key=5></td></tr><tr key=0-2-1-16><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> relationData</span></td><td key=1><span>object []</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">流程关联数据</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-2-1-16-0><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> ids</span></td><td key=1><span>string []</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>string</span></p></td></tr><tr key=array-811><tr key=0-2-1-16-1><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> type</span></td><td key=1><span>string</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2-1-17><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> doneNode</span></td><td key=1><span>object []</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">已执行节点</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-2-1-17-0><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> date</span></td><td key=1><span>string</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2-1-17-1><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> desc</span></td><td key=1><span>string</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2-1-17-2><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> action</span></td><td key=1><span>string</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2-1-17-3><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> assignee</span></td><td key=1><span>number</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2-1-17-4><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> taskName</span></td><td key=1><span>string</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2-1-17-5><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> activitiId</span></td><td key=1><span>string</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2-1-17-6><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> taskResult</span></td><td key=1><span>string</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2-1-18><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> labels</span></td><td key=1><span>string</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">标签</span></td><td key=5></td></tr><tr key=0-2-1-19><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> taskDesc</span></td><td key=1><span>string</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">工单描述</span></td><td key=5></td></tr><tr key=0-2-1-20><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> ctime</span></td><td key=1><span>string</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">创建时间</span></td><td key=5></td></tr><tr key=0-2-1-21><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> doneUser</span></td><td key=1><span>number []</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">已执行用户</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>number</span></p></td></tr><tr key=array-812><tr key=0-2-1-22><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> platDate</span></td><td key=1><span>string</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">计划完成时间</span></td><td key=5></td></tr><tr key=0-2-1-23><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> userId</span></td><td key=1><span>number</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">用户 id</span></td><td key=5></td></tr><tr key=0-2-1-24><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> execStatus</span></td><td key=1><span>string</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">工单状态</span></td><td key=5></td></tr><tr key=0-2-1-25><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> remark</span></td><td key=1><span>object</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">备注</span></td><td key=5></td></tr><tr key=0-2-1-25-0><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> notice_note</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2-1-25-1><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> notice_status</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2-1-26><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> executeInfo</span></td><td key=1><span>object []</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">已执行信息</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>object</span></p></td></tr><tr key=0-2-1-26-0><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> date</span></td><td key=1><span>string</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2-1-26-1><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> desc</span></td><td key=1><span>string</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2-1-26-2><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> action</span></td><td key=1><span>string</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2-1-26-3><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> assignee</span></td><td key=1><span>number</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2-1-26-4><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> taskName</span></td><td key=1><span>string</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2-1-26-5><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> activitiId</span></td><td key=1><span>string</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2-1-26-6><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> taskResult</span></td><td key=1><span>string</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2-1-27><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> taskContent</span></td><td key=1><span>object</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">工单内容</span></td><td key=5></td></tr><tr key=0-2-1-27-0><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> files</span></td><td key=1><span>string []</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>string</span></p></td></tr><tr key=array-813><tr key=0-2-1-27-1><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> remark</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2-1-27-2><td key=0><span style="padding-left: 60px"><span style="color: #8c8a8a">├─</span> analysis_explain</span></td><td key=1><span>string</span></td><td key=2>非必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap"></span></td><td key=5></td></tr><tr key=0-2-1-28><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> userFollows</span></td><td key=1><span>string []</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">关注用户</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>string</span></p></td></tr><tr key=array-814><tr key=0-2-1-29><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> timeOutStatus</span></td><td key=1><span>boolean</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">是否超时</span></td><td key=5></td></tr><tr key=0-2-1-30><td key=0><span style="padding-left: 40px"><span style="color: #8c8a8a">├─</span> tagNames</span></td><td key=1><span>string []</span></td><td key=2>必须</td><td key=3></td><td key=4><span style="white-space: pre-wrap">标签名称列表</span></td><td key=5><p key=3><span style="font-weight: '700'">item 类型: </span><span>string</span></p></td></tr><tr key=array-815>
               </tbody>
              </table>


<!-- endtoc -->
