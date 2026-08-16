---
type: product
title: 守元-FAQ
product: 守元
aliases: []
tags:
    - 流量安全
    - AG-S10-26.06.001
    - 常见问题
summary: 汇总守元客户使用过程中的高频问题和简要处理建议，待产品方补充具体问答。
status: verified
---

# FAQ - 守元

# 前期交流

## 是否需要 GPU 硬件？

需要，守元基于模型进行检测，模型运行需要 GPU。

## 需要什么样的GPU？

目前实测情况，仅测试文本检测，如果单卡显存小于 24G，需要两卡；

## 客户 GPU 需要通过 k8s 提供怎么办？

请带具体商机联系产线。

## 守元检测模型是否适配国产化GPU算力？

理论可行，但公司内现在没有国产化GPU用于适配。

## 客户有一套自己的 网关/平台，能接入么？

不同平台有自己的不同逻辑，不保证都可以无痛接入，尤其是第三方连带基础大模型一起提供的。

客户自己有独立大模型基建，或平台支持一些 “插件” 形式导出问答交互的比较有希望。

具体请提供 平台 官网给产线以供调研。

客户自研平台的话，我们可以提供 API 供客户接入。

## 单次检测耗时（如1000字中文文本）平均耗时多久？

Nvidia 4090 下运行检测耗时在 100～200ms。

其他 GPU 耗时无法估计，也没有硬件资源测试。

## 是否支持水平扩展？

支持，卡越多，检测总 qps 越大。

## 检测方式（是否检测上下文 / 是否检测回答 / ...）

每次输入 / 回答，会进行一次检测，每次检测关联上下文。

*   基于 API、插件等非串联网关形式接入时，视接入方式而定（接入时需要能关联上下文）
    

如果响应较长，会被分割成多个分片依次检测，分片长度可在站点配置中配置。

## 有没有计划做防token滥用？是否支持限频？

支持基于访问频率、token 消耗（预估值，和实际使用量有差距）的限频

## 遇到新的绕过手法或注入，守元如何应急？

1.  写正则规则
    
2.  加急训练 Bert 小模型（可能误杀）
    
3.  客户现场通过飞轮训练模型
    
4.  再训练稳定的模型
    

## 出现误报，是否支持加白？

1.  可以通过飞轮，根据误报内容本地迭代模型
    
2.  专用场景或者某些客户的特殊需求可能会重新提供模型（但也需要客户提供相应数据）
    

## 检出结果除安全、不安全外，是否有争议性或待确认标签？

策略实现中，风险判定结果是一个小数，根据策略中配置的阈值范围判定是否安全。可以认为 放行 与 拦截 之间的观察部分是 “有争议” 的

# 部署交付

## Goalkeeper 容器无法启动

### 缺少 nvidia-container-toolkit

1.  前往 [Release v1.18.1 · NVIDIA/nvidia-container-toolkit](https://github.com/NVIDIA/nvidia-container-toolkit/releases/tag/v1.18.1)下载 deb 或则 rpm 包集合
    
2.  上传到环境后 tar -xzvf nvidia-container-toolkit\_1.18.1\*.tar.gz
    
3.  进入解压后的内部路径，执行安装
    
    1.  deb 包：dpkg -i ./\*
        
    2.  rpm 包：yum install ./\*
        
4.  nvidia-ctk runtime configure --runtime=docker
    
5.  systemctl reload docker && systemctl restart docker
    

### libseccomp2 is needed by libnvidia-container-libseccomp2-1.17.8-1.x86\_64

1.  删掉 libnvidia-container-libseccomp2 包再安装 [Why install nvidia-container-toolkit with manual failed in centos? · Issue #1254 · NVIDIA/nvidia-container-toolkit](https://github.com/NVIDIA/nvidia-container-toolkit/issues/1254)
    

### initialization error: time out

docker: Error response from daemon: failed to create shim task: OCI runtime create failed: runc create failed: unable to start container process: error during container init: error running hook #0: error running hook: exit status 1, stdout: , stderr: Auto-detected mode as 'legacy'

nvidia-container-cli: initialization error: driver rpc error: timed out: unknown.

1.  执行 nvidia-smi -pm 1 将 GPU mode 切入 persistence mode [Increase the timeout of nvidia-container-toolkit · Issue #202 · NVIDIA/nvidia-container-toolkit](https://github.com/NVIDIA/nvidia-container-toolkit/issues/202)
    

## 检测策略中的阈值含义说明

我们的检测引擎是多分类任务，输出的阈值概率的含义是：模型判断当前输入属于该类别的置信度为 xx%。 多分类任务中，BERT 的输出层会接softmax 函数，输出一个概率分布向量，向量中每个元素对应一个类别的概率，且所有元素之和为 1。

举例：

比如 6 分类任务中，概率分布为 \[0.1, 0.1, 0.1, 0.1, 0.1,0.5\] → 0.1 对应的类别属于低置信度候选类，模型明确倾向于概率 0.5 的类别。

更极端的如 \[0.2, 0.2, 0.2, 0.2, 0.1, 0.1\] 是完全无把握，这种低置信度的正确预测，往往是 “运气成分” 居多，说明模型对该样本的特征学习不充分，泛化能力存疑。

对于阈值配置的建议：如果某一类型属于业务场景非常关注的风险类型，可以把阈值设置较低，尽量多的捕获风险场景。如果风险类型对业务影响较小，可以把阈值设置较高，减少误报。

## 检测能力（特征库）怎么更新？离线还是在线？

1.  离线更新
    
2.  可以当作更新了模型，模型文件可能 100M ～ 10G
    

## 如何选择模型运行的 GPU

编辑配置文件 /data/aiguard/resources/goalkeeper/config/goalkeeper.yml

其中 CUDA\_VISIBLE\_DEVICES 表示对应模型运行的显卡编号。

修改配置后保存，然后 docker restart ag-goalkeeper。一般需要数分钟的加载时间。

ai-guard-model-installer-T覆盖模型：

*   service-bert-guard
    
*   service-embedding
    
*   service-qwen3-guard-6class
    
*   service-model-attack-guard
    

## 是否有直观方式查看模型是否加载正常

暂无。

可以先执行 docker logs --tail 20 -f ag-goalkeeper，然后进行文件检测，观察是否同步有异常日志打印来判断。


## 配置要求

正在优化部署，完成优化后配置要求将更新在产品规格说明书中，目前配置要求可参考：

首先根据需求来分，主要看是否有多模态和智能体需求：
1. 纯文本测试：
* 基础平台（用于部署围栏软件）：8核16线程CPU 1颗，64G内存，4T块硬盘
* 算力服务器：（用于部署围栏检测模型）：16核32线程CPU 1颗，64G内存，4Tx2块硬盘，3张昇腾或支持 CUDA的 NVIDIA 显卡（如果不用数据飞轮功能可以2张），单张显存容量16GB。
2. 涉及多模态测试：
* 基础平台（用于部署围栏软件）：8核16线程CPU 1颗，64G内存，4T块硬盘
* 算力服务器：（用于部署围栏检测模型）：16核32线程CPU 1颗，64G内存，4Tx2块硬盘，4张昇腾或支持 CUDA的 NVIDIA 显卡（如果不用数据飞轮功能可以3张），单张显存容量24GB。
3. 涉及智能体测试：
* 基础平台（用于部署围栏软件）：8核16线程CPU 1颗，64G内存，4T块硬盘
* 算力服务器：（用于部署围栏检测模型）：16核32线程CPU 1颗，64G内存，4Tx2块硬盘，4张昇腾或支持 CUDA的 NVIDIA 显卡（如果不用数据飞轮功能可以3张），单张显存容量24GB。
4. 涉及多模态+智能体测试（卡的数量影响吞吐量）：
* 基础平台（用于部署围栏软件）：8核16线程CPU 1颗，64G内存，4T块硬盘
* 算力服务器：（用于部署围栏检测模型）：16核32线程CPU 1颗，64G内存，4Tx2块硬盘，5-8张昇腾或支持 CUDA的 NVIDIA 显卡（如果不用数据飞轮功能可以少1张），单张显存容量24GB。

## 能支撑多少 QPS 

吞吐量跟卡的个数以及模型部署方式有关：

纯文本一共有三个模型 A B C；两张卡是 A B 在同一张部署，C 在另一个张，因为 A 显存少，速度快，就还行能 100qps

如果有三张卡，ABC各一张，那能到 300QPS

如果带上 agent 防护 X、多模态 Y 中任意一个，qps 会降到个位数。多模态当然要检测图片才会，不影响文本

另外不同的卡QPS不同，Nvidia 单4090D 是 100QPS，昇腾的QPS可能要比NVIDIA低
