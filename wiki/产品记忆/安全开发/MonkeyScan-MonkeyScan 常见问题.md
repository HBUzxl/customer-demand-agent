---
type: product
title: MonkeyScan-MonkeyScan 常见问题
product: MonkeyScan
aliases: []
tags:
    - 安全开发
    - FAQ
summary: 介绍MonkeyCode可能遇到的问题
status: verified
---

## 💡 是否支持私有化部署？

MonkeyScan 定位是 SaaS 产品，没有支持私有化部署计划。

## 💡 我的代码上传后是否安全？

MonkeyScan 会采取必要的安全措施保护数据，每次代码审计任务在独立沙箱中，审计完毕自动删除，只保留部分有缺陷的代码片段，用于用户查看漏洞细节。

## 💡 MonkeyScan 当前支持哪些源码接入方式？

当前文档建议按以下两种主要方式理解：

- GitHub 仓库接入
- ZIP 源码包上传

## 💡 为什么使用前需要实名认证？

当前扫描服务要求用户先完成百智云实名认证后再使用，实名认证使用支付宝提供的服务，MonkeyScan 不额外保留实名认证信息。

这属于平台的准入和合规要求。完成认证后，后续即可正常发起扫描任务。

认证入口：

- [百智云个人中心](https://baizhi.cloud/user)
