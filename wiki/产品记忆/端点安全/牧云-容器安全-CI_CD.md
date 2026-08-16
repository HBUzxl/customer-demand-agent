---
type: product
title: 牧云-容器安全-CI/CD
product: 牧云-容器安全
aliases: []
tags:
    - 端点安全
    - 产品功能
    - 安全左移
summary: 介绍CICD的主要内容、操作方法和注意事项。
status: verified
---

<!-- toc -->

# CI/CD

在项目开发过程中，我们频繁在生产环境使用镜像完成基础构建，但在镜像创建、构建、传输的过程中，我们该如何保障镜像的安全性与完整性？

牧云·云原生安全平台提供 CI/CD 功能，兼容多种主流 CI/CD，如 Jenkins 和 Gitlab。支持一键生成检测策略，并查看具体检测结果，直观高效，精准定位！

下文以对接 Jenkins 为例给出使用说明，其他 CI/CD 自动化工具对接方式类似，不再进行重复赘述。

## Jenkins 对接

点击 Jenkins 卡片进入 Jenkins 对接和策略管理页面。

![](../../../images/product-features/upload_599a271cf5dd2cb4fc4d66c50b3171fc.png)

点击页面右上角 “对接指引” 按钮，打开 Jenkins 对接指引弹窗。按照弹窗中的步骤下载插件文件到本地，并前往 Jenkins 面板完成插件安装和初始化配置。

![](../../../images/product-features/upload_f15b2f5af4e81bcf1da5f304bb2772f6.png)

![](../../../images/product-features/upload_5210bc43d22d877883fa642bcd84e8b4.png)

## 创建策略

完成 Jenkins 对接后需要在容器安全管理平台创建检测策略，接着在 Jenkins 项目中添加策略对应的扫描任务，添加成功后构建流程将执行此扫描任务。

系统内置默认策略，对所有使用插件的 CI/CD 进行安全检测扫描并上报结果到平台，默认策略不会进行镜像阻断操作。如针对不同的 Pipline 有不同的检测需求，支持用户根据检测需求自定义策略。
点击右上角 “创建策略” 按钮，在弹窗中定义策略名称、扫描镜像范围、启用插件、阻断开启情况，成功创建检测策略。

![](../../../images/product-features/upload_fe8837507cfc08e9655db1a3d558afc0.png)

策略创建成功后将自动跳转到策略详情页面，可根据详情页面左侧指引，在Jenkins 项目中添加策略对应的扫描任务。

![](../../../images/product-features/upload_e2b82461afe545e950d721fd1435e09a.png)

## 策略管理

系统内置默认策略仅支持查看，自定义策略支持编辑和删除操作。

![](../../../images/product-features/upload_5d847924ca486aa4d84e305de5fa0249.png)

## 查看检测结果

列表页左侧选中某一个策略对象，右侧将会展示使用此策略进行扫描上报的所有事件数据，点击 “Job” 字段可以查看每条事件详细内容。支持通过筛选条件进行事件筛选，也支持对上报上来的事件进行删除操作。

![](../../../images/product-features/upload_64af82a9e7f5b21f1b9309383732a32c.png)

点击 “Job” 字段进入 Pipline 事件详情页面，Pipline 事件详情包括CI/CD 信息、扫描镜像信息、策略详情、检测详情。提供 CI/CD 具体字段、本次扫描的镜像对象、本次扫描使用的策略内容和扫描的结果，多类型数据统一展示，便于快速排查风险并进行下一步修复。

![](../../../images/product-features/upload_af56faf6c917d7795b0b0a247ec1eef3.png)

<!-- endtoc -->
