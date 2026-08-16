---
type: product
title: 牧云-容器安全-WebShell-产品功能
product: 牧云-容器安全
aliases: []
tags:
    - 端点安全
    - 产品功能
    - 运行时检测
summary: 介绍WebShell的主要内容、操作方法和注意事项。
status: verified
---

<!-- toc -->

# WebShell

牧云·云原生安全平台通过分析容器配置自动定位 Web 目录，对 Web 目录内的文件进行全面扫描，并持续监控 Web 目录内的文件变动，从而实现对于 WebShell 实时感知能力。使用动态沙箱模拟执行、静态语义分析、静态特征匹配等多种检测算法对 Web 脚本深度扫描，提供详细的文件信息与检测依据。

## 操作方法

列表提供资产名称、风险等级、状态等字段进行筛选，点击【事件名称】可以前往事件详情页面。

点击操作列的【处理】可选择对事件的处理方式，包括暂停容器（主机）、移除容器（主机）、隔离容器所属 Pod 资源（集群）、加白事件（主机 / 集群）。加白事件将自动生成加白规则并对事件进行加白。

![](../../../../images/product-features/upload_a5c414d4547e7b35a6003abc6fae6017.png)

点击页面右上角 “规则配置” 按钮，可自定义开启和关闭自动隔离规则。

![](../../../../images/product-features/upload_dc900419a747b8945aad1cdccda0cf04.png)

如存在极少数 Web 目录无法被探针识别，导致无法检测这些目录下的入侵事件和风险，可点击 “自定义 Web 路径” 进入内层页面添加这类 Web 目录，添加后即可进行检测。

![](../../../../images/product-features/upload_1e0d2f3abcf15e732c9eee72bd052b49.png)

点击右上角 “添加 Web 路径”按钮，在表单中定义 Web 路径和路径对应的作用范围，成功添加 Web 路径信息。支持对添加的 Web 路径进行编辑、删除、筛选操作。

![](../../../../images/product-features/upload_18551416ffb7769cb825280294c8d44d.png)

事件详情页面包括以下模块：文件信息、检测详情、资产信息、事件信息、处理记录。
1. 文件信息提供文件名称等 10 个字段信息，点击右上角![](../../../../images/product-features/upload_644a830e974245cbd5ae3fb1a81d4877.png)可下载 WebShell 样本，便于排查；
2. 检测详情提供检测依据和解决方案信息；
3. 资产信息展示存在此事件的资产对象，并支持关联跳转查看资产详情，包含容器、Pod、主机、集群；
4. 事件信息包括 WebShell 类型、风险等级、发现时间等信息；
5. 处理记录模块记录此事件状态更改情况。

![](../../../../images/product-features/upload_eedeb2a3cef088ecb98a544b490d0bb5.png)

<!-- endtoc -->
