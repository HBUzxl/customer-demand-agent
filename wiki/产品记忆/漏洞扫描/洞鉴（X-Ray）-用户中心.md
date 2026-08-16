---
type: product
title: 洞鉴（X-Ray）-用户中心
product: 洞鉴（X-Ray）
aliases: []
tags:
    - 漏洞扫描
    - support
    - 功能使用
    - 界面功能
summary: 个人中心信息管理和 Open API Token 管理。
status: verified
---

# 个人中心与 Open API

在系统右上角悬浮，可点击进入「个人中心」、「Open API」页面。

## 个人中心

个人中心展示当前登录账号的基本信息。

![](../images/upload_8b1a9898af3caf2375410d09f0f8d115.png)

**常用操作：**

* **登录工作区**：表示登录后默认切换的组织单位
* **修改登录密码**：输入 “旧密码”、“新密码”，和“确认新密码” 即可修改当前密码。修改成功后，会自动登出系统，需要重新登录
![](../images/upload_7dbc60f7d9cc9d0f85f3ed99af47195d.png)
* **配置动态身份验证**：输入登录密码后可以开启配置，启用后每次登录都需要输入动态认证密钥。
    * 认证方法：扫描系统生成的 “动态身份验证” 二维码（例如，手机 APP 应用商城搜索“Auth”或者“动态身份验证”，如 Authy、OTP Auth、Google Authenticator 等），添加私钥到认证应用，然后在洞鉴管理界面输入应用内生成的 6 位动态密码，完成配置
![](../images/upload_af9d13685f58789b384152aad6c4ec0c.png)

## Open API

OpenAPI 可以用于对接联动设备、自动化脚本，灵活适配多种使用需要。

OpenAPI 管理页面可以 **添加、删除、修改、查看** API Token

![](../images/upload_da1fa90eefdf6d3d795cceb66235bc3c.png)


### 添加 Token

添加 Token 步骤：

1. 点击“添加一条 Open API”
![](../images/upload_7f03107d7eb14a04abc4318855c4d813.png)
2. 配置基本信息。包含“过期时间”、“备注”。“过期时间” 若设置空，表示不限制使用期限
3. 点击“下一步”，选择是否配置 “IP 白名单”。配置后，仅名单中的 IP 可使用该 Token
![](../images/upload_8fc88615b93b39d7928843c04d5561de.png)
4. 点击“下一步”，确认信息。
![](../images/upload_c71f0b44b2d6fe6f7b3ed39b3a25ddc9.png)
5. 点击 “完成”，复制 Token。弹窗页面关闭后，将不再显示 Token，请务必对 Token 进行**复制**操作，并妥善保管。
![](../images/upload_6fdb939bc5da2aaafd2e725533b1bf18.png)

### 查看 Open API 文档

点击“查看 Open API 文档”按钮，即可访问「Open API 手册」页面。

![](../images/upload_c70e5f79fe89a9a739c080e581ef9c84.png)
