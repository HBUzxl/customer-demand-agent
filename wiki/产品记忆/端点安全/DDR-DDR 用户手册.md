---
type: product
title: DDR-DDR 用户手册
product: DDR
aliases: []
tags:
    - 端点安全
    - v3.10.1
    - 使用指南
summary: DDR v3.9 完整用户操作指南，涵盖总览、数据安全、桌面管理、网络管控、风险运营等全部功能模块的使用说明。
status: verified
---

# 长亭终端统一管控与安全检测响应平台 DDR 用户手册


**名词解释**

|  |  |
|:--:|:--:|
| **名词** | **解释** |
| 企业管理员 | 有权限操作长亭终端统一管控与安全检测响应平台(DDR) 控制台的使用者 |
| 终端用户 | 已安装DDR Agent设备的员工 |

1. **总览**

总览实时显示包括 CPU
利用率、内存使用率、磁盘占用率、入带宽、出带宽等数据在内的**系统性能统计**，活跃终端、安全运行等**统计数据**，待办事项、数据泄露、资产扫描、桌管态势、网络管理等**核心指标**，无需跳转多个页面即可掌握系统整体运行状况。

提供终端安装、创建扫描、创建策略，资产地图、数据日志、行为洞察共六类常用功能的**快速入口**，可以实现从总览到各高频使用功能的快速跳转。

<img src="../images/image-002.png" style="width:5.75in;height:2.8125in" />

各模块功能说明如下：

1.1 **快速入口**

快速入口中汇集多个高频功能入口，点击按钮可以快速进入对应页面，包括：

- **终端安装**：快速进入「桌面管理-终端管理-下载管理」页面。

<!-- -->

- **创建扫描**：快速进入「数据安全-资产发现」页面，并打开创建资产扫描任务弹窗。

<!-- -->

- **创建策略**：鼠标悬浮时打开选项，从选项中快速进入对应管控策略页面。如联网管控按钮，可以快速进入「网络管理-联网管控-全局联网」页面。

<img src="../images/image-003.png"
style="width:1.95833in;height:2.82292in" />

- **资产地图**：快速进入「数据安全-资产地图」页面。

<!-- -->

- **数据日志**：快速进入「日志中心-数据安全-渠道管控」页面。

<!-- -->

- **行为洞察**：快速进入行为洞察页面。

<img src="../images/image-004.png" style="width:5.75in;height:0.58333in" />

<span id="_Toc1312009624" class="anchor"></span>1.2 **系统性能**

在性能指标中，显示系统性能的重要指标，包括：

- **CPU利用率**：利用率和核心数

<!-- -->

- **内存使用率**：使用率和内存大小

<!-- -->

- **磁盘占用**：已使用磁盘和磁盘总量

<!-- -->

- **入带宽**：入带宽的大小

<!-- -->

- **出带宽**：出带宽的大小

对于超过一定限度的性能数据，将给出“过高”警示。

<img src="../images/image-005.png" style="width:5.75in;height:0.83333in" />

<span id="_Toc583663172" class="anchor"></span>1.3 **活跃终端**

统计近24小时在线状态的终端数量，分别显示不同操作系统活跃终端的数量。

<img src="../images/image-006.png" style="width:5.75in;height:2.125in" />

<span id="_Toc2069115955" class="anchor"></span>1.4 **安全运行**

安全运行监控将显示长亭终端统一管控与安全检测响应平台的安全运行告警信息，点击不同信息将跳转到对应的功能页面，包括：

|                  |                          |                            |
|:-----------------|:-------------------------|:---------------------------|
| **安全运行信息** | **对应功能**             | **说明**                   |
| 失效终端         | 终端管理                 | 统计失效的终端数量         |
| 未绑定终端用户   | 用户管理-未绑定用户      | 统计未绑定用户的终端数量   |
| 系统状态         | 模块管理                 | 统计当前系统状态           |
| 产品生命周期     | 系统管理-授权配置        | 统计当前系统是否过期       |
| 已监控渠道       | 泄露管控-渠道管控-管控源 | 统计当前监控中的渠道数量   |
| 生效联网管控策略 | 网络管理-联网管控        | 统计当前生效的联网管控策略 |
| 生效外设管控策略 | 桌面管理-外设管控        | 统计当前生效的外设管控策略 |
| 生效泄露管控策略 | 数据安全-泄露管控        | 统计当前生效的泄露管控策略 |
| 生效合规基线策略 | 桌面管理-合规基线        | 统计当前生效的合规基线策略 |
| 生效软件管控策略 | 桌面管理-软件管控        | 统计当前生效的软件管控策略 |

|  |  |
|:--:|:--:|
| <img src="../images/image-007.png"
style="width:2.66667in;height:2.48958in" /> | <img src="../images/image-008.png" style="width:2.73958in;height:2.5in" /> |

<span id="_Toc1429159814" class="anchor"></span>1.5 **多模块数据**

多模块数据将显示待办事项，数据泄露，资产扫描、桌管态势、网络管理的统计数据：

- 待办事项：风险处理率（包括高危、中危、低危三个等级风险的处理情况）、任务审批率（包括渠道管控、软件管控、进程管控的审批任务处理情况）

<img src="../images/image-009.png" style="width:5.75in;height:1.61458in" />

- 数据泄露：风险行为趋势（基于时间维度，显示不同风险等级的风险行为事件数量信息）、风险渠道统计
  TOP10（依据风险渠道相关的行为数量排序）、风险员工统计
  TOP10（依据员工产生的风险行为数量排序）

> <img src="../images/image-010.png" style="width:5.75in;height:3.20833in" />

- 资产扫描：资产发现（资产发现任务的完成情况）、命中分类分级
  TOP5（不同分类、分级数据的命中情况统计）

<img src="../images/image-011.png" style="width:5.75in;height:1.8125in" />

- 桌管态势：终端数量（当前注册的终端数量以及环比变化率）、终端信息（统计在线、离线、失效的终端设备数量）、不合规终端TOP10（未满足的合规项数终端终端设备信息）、软件安装TOP10（域内安装数TOP10的软件信息及总数统计）

<img src="../images/image-012.png" style="width:5.75in;height:3.3125in" />

- 网络管理：全局联网趋势（基于时间维度，显示不同风险等级的全局联网事件数量信息。）、全局联网趋势（基于时间维度，显示联网配置下的进程联网事件数量信息。）

<img src="../images/image-013.png" style="width:5.75in;height:3.1875in" />

<span id="_Toc284402203" class="anchor"></span>1.6 **系统模块状态**

将实时显示系统模块的启用状态，点击「查看详情」即可查看和修改系统不同模块的启用/关停状态。同时，此处将展示控制台版本、策略版本、授权到期信息。

<img src="../images/image-014.png" style="width:5.75in;height:6.88542in" />

<span id="_Toc1796711246" class="anchor"></span>2. **审批任务**

<span id="heading_8" class="anchor"></span>2.1 **审批任务**

支持审批文件外发和其他命中策略情况，企业管理员可设置系统审批或IM应用审批两种审批方式。所有的审批任务都将会在「审批任务」中进行管理和呈现。

<table style="width:88%;">
<colgroup>
<col style="width: 88%" />
</colgroup>
<tbody>
<tr>
<td style="text-align: left;"><p><strong>系统审批</strong>：</p>
<p><strong>适配场景</strong>：适用于小微企业，IT
管理员负责整个企业文件外发的审批工作.</p>
<p><strong>IM审批：</strong></p>
<p><strong>适配场景</strong>：适用于使用统一企业IM软件（如钉钉、飞书）的公司，审批流程较为复杂多变，配置完成后，员工的上级领导可通过企业IM进行审批操作。</p>
<p><strong>高级审批：</strong></p>
<p><strong>适配场景</strong>：适用于使用AD进行设备和员工管理的企业，员工可在DDR界面上发起审批请求，审批人可在终端界面/邮件完成审批操作。</p></td>
</tr>
</tbody>
</table>

2.1.1 **审批模板**

审批通常作为风险事件的一种处置响应方式，因此在应对各种风险策略配置页中，大多可以找到审批模板的配置页。如图，以全局联网的审批模板页为例，可以在此处管理和查看所有已经创建的审批模板。注意，此处同样包含了其他类型的处置模板，可以通过模板上方的标签，找到对应的审批模板。

在模板列表中，企业管理员可以对审批模板可以进行编辑，复制，删除和依赖查询操作。

<img src="../images/image-015.png" style="width:5.75in;height:3.6875in" />

包含审批模板配置的页面具体有：

- 桌面管理-外设管控

<!-- -->

- 桌面管理-软件管控

<!-- -->

- 数据安全-泄露管控-外发管控

2.1.2 **创建审批模板**

2.1.2.1 **普通审批流程模板配置**

**注意:
在使用审批功能之前，我们需要先配置审批模板供DLP策略触发时调用。**

Step1. 在「管控配置」-「处置模板」页，点击新增模板，打开新增模板抽屉。

<img src="../images/image-016.png" style="width:5.75in;height:1.4375in" />

Step2.
在新增模板页面，填写模板名称并按需填写模板描述，在处置动作处选择审批。

<img src="../images/image-017.png" style="width:5.75in;height:3.23958in" />

Step3.
填写审批模板的相关信息。包括审批模式、终端提示内容、生效时长、生效次数信息。

<img src="../images/image-018.png" style="width:5.75in;height:5.21875in" />

<table style="width:89%;">
<colgroup>
<col style="width: 11%" />
<col style="width: 30%" />
<col style="width: 46%" />
</colgroup>
<tbody>
<tr>
<td style="text-align: left;"><strong>填写内容</strong></td>
<td style="text-align: left;"><strong>选项</strong></td>
<td style="text-align: left;"><strong>说明</strong></td>
</tr>
<tr>
<td style="text-align: left;">审批模式</td>
<td style="text-align: left;">【必填】普通模式/高级模式</td>
<td style="text-align: left;"><p>选择审批模式。</p>
<table style="width:44%;">
<colgroup>
<col style="width: 43%" />
</colgroup>
<tbody>
<tr>
<td
style="text-align: left;"><p>普通模式：直接在页面即可配置是否通过审批。</p>
<p>高级模式：</p>
<ul>
<li><p>模式一：用户可以在终端GUI上进行审批操作</p></li>
</ul>
<ul>
<li><p>模式二：审批通过请求将直接通过IM应用发送到相关处理人，处理人直接在IM应用通过审批即可。</p></li>
</ul></td>
</tr>
</tbody>
</table></td>
</tr>
<tr>
<td style="text-align: left;">终端提示内容</td>
<td style="text-align: left;">【选填】模板+变量</td>
<td
style="text-align: left;">填写终端提示内容，支持插入变量，提示内容将出现在提交审批时的弹窗上。</td>
</tr>
<tr>
<td style="text-align: left;">生效时长</td>
<td style="text-align: left;">【必填】生效时间长短</td>
<td style="text-align: left;">审批通过后允许使用/外发的生效时长</td>
</tr>
<tr>
<td style="text-align: left;">生效次数</td>
<td style="text-align: left;">【仅外发管控】</td>
<td style="text-align: left;">审批通过后允许外发的生效次数</td>
</tr>
</tbody>
</table>

2.1.2.2 **高级审批流程模板配置**

1.  在「管控配置」-「处置模板」页，点击新增模板，打开新增模板抽屉。

<img src="../images/image-016.png" style="width:5.75in;height:1.4375in" />

2.  在新增模板页面，填写模板名称并按需填写模板描述，在处置动作处选择审批。

<img src="../images/image-017.png" style="width:5.75in;height:3.23958in" />

<img src="../images/image-019.png" style="width:5.75in;height:6.36458in" />

3.  审批模式：选择「高级模式」。高级审批模式可以配置多层级的审批流程。

<!-- -->

4.  设置审批流程

- 一级审批：在一级审批部分，点击「设置审批人」，支持选择：

<!-- -->

- 系统管理员：由系统指定的管理员进行审批。

<!-- -->

- 提交人上级：由提交人的直接上级进行审批。

<!-- -->

- 指定员工：选择特定的员工进行审批。

<!-- -->

- 提交人自选：由提交人自主选择审批人。提交人亦可以选择自己作为审批人。

<!-- -->

- 如需增加多个审批人，点击「添加审批人」按钮。

<!-- -->

- 二级审批：如果需要多级审批，点击「添加二级审批」按钮，并按照同样的方式设置审批人。审批流最大支持二级。

5.  设置审批规则

- 审批人为空时：如果指定的审批人为空，可选择：

<!-- -->

- 自动通过：任务将直接通过审批。

<!-- -->

- 指定审批人：任务将转交指定的审批人。

<!-- -->

- 系统管理员：任务将转交给系统管理员。

<!-- -->

- 超时设置：配置审批超时时间，例如设置1小时内审批人未完成任务，系统会将任务转交给系统管理员。

6.  发件邮箱：配置邮件通知渠道，当审批任务生成时，系统将自动向审批人邮箱发送审批通知邮件。

<!-- -->

7.  点击"保存"。

2.1.3 **审批任务列表**

在审批任务列表中点击「通过审批」后，申请人可在指定生效期（次数）内发送敏感文件，生效信息可在处置模板进行配置。

点击「详情」，查看更具体的审批日志信息。

<img src="../images/image-020.png" style="width:5.75in;height:3.07292in" />

2.1.4 **审批详情**

企业管理员可在此处对所有审批日志进行查看。若账号拥有对应的审批权限，也可在进行审批处理，处理的结果会通过通知的方式同步给对应人员。

<img src="../images/image-021.png" style="width:5.75in;height:3.65625in" />

2.1.5 **终端显示**

当员工命中处置动作为审批的策略时，将会在终端进行弹窗，员工需要输入相应的理由，经由审批人审批通过后方可在指定失效期内进行外发/使用。

<table style="width:89%;">
<colgroup>
<col style="width: 38%" />
<col style="width: 50%" />
</colgroup>
<tbody>
<tr>
<td style="text-align: center;"><p><img src="../images/image-022.png"
style="width:2.30208in;height:2.46875in" /></p>
<p>终端界面-提交审批</p></td>
<td style="text-align: center;"><p><img src="../images/image-023.png"
style="width:3.10417in;height:2.51042in" /></p>
<p>终端界面-审批结果</p></td>
</tr>
</tbody>
</table>

3. **风险调查**

风险调查模块用于还原风险事件发生过程，帮助管理员在第一时间判断风险真实性和影响范围。模块入口为“风险调查-实时监控”，

日常使用中，建议先建立监控任务，再结合截图与标记信息进行连续排查，这样可以更快定位问题终端和责任人。

3.1 **实时监控**

实时监控页面主要承担三件事：管理监控任务、查看事件截图、记录处置过程。进入页面后可以看到任务列表，列表展示任务名称、运行状态、目标终端和创建时间。状态支持“开始”和“暂停”切换，任务不再需要时可以删除。

3.1.1 **新增任务**

创建任务时，需要填写任务名称，选择目标终端，并设置生效周期、生效时段、监管方式和存储周期。保存成功后，任务会立即出现在列表中。建议在创建后先观察一段时间，确认数据持续上报正常，再投入正式调查使用。

<img src="../images/image-024.png" style="width:5.75in;height:2.72917in" />

3.1.2 **任务详情**

点击“详情”会进入任务详情页。详情页支持按日期筛选，并可按时间段快速切换事件。左侧展示截图序列，右侧显示当前事件信息和标记内容。调查过程中可以对关键事件添加标记评论，便于后续复盘和交接。截图支持放大查看，也支持下载留存作为证据材料。

<img src="../images/image-025.png"
style="width:5.57014in;height:2.92708in" />

<span id="_Toc13410918" class="anchor"></span>4. **日志中心**

提供网络、桌管、终端、系统等全功能模块日志，管理员可在此处执行查看日志列表、日志详情、处理日志对应事件等操作。

<span id="_Toc2058999538"
class="anchor"></span>**通用日志操作-日志列表**

1.  在风险日志列表中，可以进行搜索、查看、筛选、删除、导出风险数据信息等操作。

<img src="../images/image-026.png" style="width:5.75in;height:3.32292in" />

图表视角

<img src="../images/image-027.png" style="width:5.75in;height:3.38542in" />

列表视角

2.  管理员可根据需要，选中多条日志后进行批量删除或批量处理。

<img src="../images/image-028.png" style="width:5.75in;height:1.44792in" />

<img src="../images/image-029.png" style="width:5.75in;height:2.48958in" />

3.  在左侧可根据时间选择框点选时间筛选项，选择后下方列表将会显示筛选结果。

<img src="../images/image-030.png" style="width:5.75in;height:4.51042in" />

4.  按条件筛选风险行为

<!-- -->

1.  点击「筛选」按钮，选择筛选信息，点击保存/另存为快速筛选。可按照需求筛选符合条件的风险信息。

<!-- -->

2.  若选择另存为，筛选信息将被保存，后续无需重复配置，可一键快速获取该筛选条件下的筛选结果。

<img src="../images/image-031.png" style="width:5.75in;height:2.77083in" />

5.  导出风险行为

<!-- -->

1.  筛选需要导出的风险行为信息。

<!-- -->

2.  点击相应图标进行数据导出。

<!-- -->

3.  填写相应信息。

<!-- -->

4.  点击完成，即可创建风险行为导出任务。

<!-- -->

5.  导出任务创建成功后，企业管理员可以通过弹窗快速进入导出任务列表查看并下载结果。

> <img src="../images/image-032.png" style="width:5.75in;height:4.58333in" />

<img src="../images/image-033.png" style="width:5.75in;height:2.25in" />

<span id="_Toc1053747408"
class="anchor"></span>**通用日志操作-日志详情**

1.  在风险日志列表右侧“操作”栏，可通过点击“详情”，查看日志的详情信息；或者点击删除，删除本条日志。

<img src="../images/image-034.png" style="width:5.75in;height:9in" />

2.  详情页面内容包含员工信息、终端信息、规则类型、风险等级、命中风险规则、传输方式、风险事件、文件命中策略信息、取证信息、内容摘要、分类详情、流转分析、文件的真实格式、文件后缀信息、终端截屏信息等。不同日志的详情页面内容有所不同。图为外发管控日志详情：

- 点击「待处理」，可以调整该条日志的处理状态

<!-- -->

- 点击「下载文件」或「远程提取」可以获取原文件信息（注：「下载文件」按钮：您需在配置策略时勾选「上传固证」功能）

<img src="../images/image-035.png" style="width:5.75in;height:3.13542in" />

3.  传输信息

- 命中策略：点击「命中策略」，将显示该文件命中的策略列表。

<!-- -->

- 取证信息：点击「取证信息」，将显示文件外发时系统截屏取证或录屏取证的内容。

<img src="../images/image-036.png" style="width:5.75in;height:4.82292in" />

- 内容摘要：点击内容摘要，将显示文件开头和结尾的各100个字节。

<img src="../images/image-037.png" style="width:5.75in;height:1.10417in" />

- 分类详情：点击「分类详情」，将显示该文件命中的分类详情及命中策略。

<img src="../images/image-038.png" style="width:5.75in;height:3.72917in" />

- 流转分析：点击「流转分析」，将显示该文件的流转信息。

<img src="../images/image-039.png" style="width:5.75in;height:2.95833in" />

<img src="../images/image-040.png" style="width:5.75in;height:5.27083in" />

如图示：贾xx（Windows设备）通过飞书平台下载文件并外发给吴xx，随后吴xx（Apple设备）再次通过飞书下载该文件，并使用企业微信进行二次外发。

<span id="heading_21" class="anchor"></span>4.1 **网络管理**

4.1.1 **联网管控**

命中联网管控（进程联网）策略产生的日志。

<img src="../images/image-041.png" style="width:5.75in;height:3.625in" />

4.1.2 **入网认证**

员工入网日志记录列表。

<img src="../images/image-042.png" style="width:5.75in;height:3.26042in" />

4.1.3 **访客登录**

访客登录的日志记录列表。

<img src="../images/image-043.png" style="width:5.75in;height:1.625in" />

<span id="_Toc665536451" class="anchor"></span>4.2 **桌面管理**

4.2.1 **合规基线**

命中合规基线策略产生的日志。

<img src="../images/image-044.png" style="width:5.75in;height:3.08333in" />

点击操作列的「详情」，查看策略详情信息。

<img src="../images/image-045.png" style="width:5.75in;height:4.09375in" />

4.2.2 **外设管控**

命中外设管控策略产生的日志。

<img src="../images/image-046.png" style="width:5.75in;height:3.04167in" />

点击操作列的「快捷加白」，选择将该条日志对应的终端或外设直接加入外设管控白名单（同时会将此外设纳入「外设管理-移动存储」管理界面）。

<img src="../images/image-047.png" style="width:5.75in;height:4.44792in" />

4.2.3 **软件管控**

命中软件运行管控策略产生的日志。

<img src="../images/image-048.png" style="width:5.75in;height:2.71875in" />

点击操作列的「快捷加白」，选择将该条日志对应的终端或软件直接加入软件管控白名单。

<img src="../images/image-049.png" style="width:5.75in;height:3.1875in" />

<span id="_Toc1576298381" class="anchor"></span>4.3 **数据安全**

4.3.1 **外发管控**

命中外发管控策略产生的日志，记录通过软件、应用外发文件的行为。

<img src="../images/image-050.png" style="width:5.75in;height:2.96875in" />

图表视角

<img src="../images/image-051.png" style="width:5.75in;height:3.10417in" />

列表视角

<img src="../images/image-052.png" style="width:5.75in;height:4.625in" />

抽屉信息

点击操作列的「详情」，查看该条日志的详细信息。

<img src="../images/image-053.png" style="width:5.75in;height:3.32292in" />

切换至流转分析界面，您可以在此查看文件的流转轨迹和数据变动情况。

<img src="../images/image-054.png" style="width:5.75in;height:2.95833in" />

若已配置落盘管控策略，并且满足文件追踪功能的启动条件，即可实现对该文件的跨设备流转记录。

|  |
|:---|
| 无论设备是否在相同网络区域，流转过程中的所有细节和节点都将被精确捕捉和记录，确保您对文件流转路径有清晰的掌握。 |

<img src="../images/image-055.png" style="width:5.75in;height:5.28125in" />

如图示：贾xx（Windows设备）通过飞书平台下载文件并外发给吴xx，随后吴xx（Apple设备）再次通过飞书下载该文件，并使用企业微信进行二次外发。

4.3.2 **落盘管控**

命中落盘管控策略产生的日志，记录文件的下载行为。

<img src="../images/image-056.png" style="width:5.75in;height:2.63542in" />

点击操作列中的「详情」查看日志详细信息。若追踪状态显示「成功」，则可启用跨设备流转监控功能以跟踪文件外发。

<img src="../images/image-057.png" style="width:5.75in;height:3.16667in" />

4.3.3 **邮件管控**

命中邮件管控策略产生的日志，记录通过邮件客户端、Web邮件应用外发邮件的行为（说明：此功能仅支持Windows操作系统）。

<img src="../images/image-058.png" style="width:5.75in;height:2.94792in" />

图表视角

<img src="../images/image-059.png" style="width:5.75in;height:2.8125in" />

列表视角

点击操作列的「详情」，查看该条日志的详细信息。

<img src="../images/image-060.png" style="width:4.9375in;height:7.0625in" />

4.3.4 **代码管控**

命中代码管控策略产生的日志，记录了代码拉取和提交的行为。

<img src="../images/image-061.png" style="width:5.75in;height:2.78125in" />

图表视角

<img src="../images/image-062.png" style="width:5.75in;height:2.95833in" />

列表视角

点击操作列的「详情」，查看该条日志的详细信息。

<img src="../images/image-063.png" style="width:5.75in;height:5.1875in" />

4.3.5 **剪贴板管控**

命中剪贴板管控策略产生的日志。记录剪贴板操作行为。

<img src="../images/image-064.png" style="width:5.75in;height:2.72917in" />

图表视角

<img src="../images/image-065.png" style="width:5.75in;height:2.59375in" />

列表视角

点击操作列中的「详情」，查看该条日志的详细信息。

<img src="../images/image-066.png" style="width:5.75in;height:3.47917in" />

4.3.6 **剪贴板日志**

开启剪贴板管控的「全量日志上报」功能，除白名单内的实体，所有剪贴板行为都会记录。

<img src="../images/image-067.png" style="width:5.75in;height:2.82292in" />

图表视角

<img src="../images/image-068.png" style="width:5.75in;height:2.61458in" />

列表视角

点击操作列中的「详情」，查看该条日志的详细信息。

<img src="../images/image-069.png" style="width:5.75in;height:3.4375in" />

4.3.7 **数据行为**

显示单机数据行为日志，并以统计图显示数据行为趋势。

<img src="../images/image-070.png" style="width:5.75in;height:3.40625in" />

图表视角

<img src="../images/image-071.png" style="width:5.75in;height:3.375in" />

列表视角

点击操作列的「详情」，查看该条日志的详细信息。

<img src="../images/image-072.png" style="width:5.75in;height:7.04167in" />

4.3.8 **AI分析**

展示AI分析文档任务的状态与结果。

<img src="../images/image-073.png" style="width:5.75in;height:2.58333in" />

<span id="heading_39" class="anchor"></span>4.4 **安全防护**

4.4.1 **钓鱼防护**

命中钓鱼防护策略产生的日志，记录员工通过敏感进程下载的风险文件行为。

<img src="../images/image-074.png" style="width:5.75in;height:3.04167in" />

4.4.2 **插件运行**

用于查看终端浏览器插件的运行与命中记录。

<img src="../images/image-075.png" style="width:5.75in;height:2.64583in" />

<span id="_Toc1053112975" class="anchor"></span>4.5 **终端日志**

4.5.1 **终端启动**

显示终端启动的日志。

<img src="../images/image-076.png" style="width:5.75in;height:3.35417in" />

4.5.2 **终端离线**

显示终端离线的日志。

<img src="../images/image-077.png" style="width:5.75in;height:3.375in" />

4.5.3 **终端绑定**

显示终端绑定和解除绑定的日志。

<img src="../images/image-078.png" style="width:5.75in;height:2.19792in" />

4.5.4 **终端升级**

显示终端升级的日志。

<img src="../images/image-079.png" style="width:5.75in;height:2.58333in" />

4.5.5 **终端卸载**

显示终端卸载的日志。

<img src="../images/image-080.png" style="width:5.75in;height:3.27083in" />

4.5.6 **终端健康**

显示终端健康的日志。

<img src="../images/image-081.png" style="width:5.75in;height:1.9375in" />

<table style="width:89%;">
<colgroup>
<col style="width: 16%" />
<col style="width: 31%" />
<col style="width: 8%" />
<col style="width: 31%" />
</colgroup>
<tbody>
<tr>
<td style="text-align: left;">操作系统</td>
<td style="text-align: left;">健康异常信息</td>
<td style="text-align: left;">错误代码</td>
<td style="text-align: left;">异常原因</td>
</tr>
<tr>
<td rowspan="8" style="text-align: center;">Windows</td>
<td style="text-align: left;">终端磁盘空间不足，已降级 Agent 功能</td>
<td style="text-align: left;">256</td>
<td style="text-align: left;">终端电脑磁盘空间不足</td>
</tr>
<tr>
<td style="text-align: left;">.net环境异常，影响部分终端GUI界面</td>
<td style="text-align: left;">32</td>
<td style="text-align: left;">.net环境不匹配</td>
</tr>
<tr>
<td
style="text-align: left;">系统版本过低，影响终端安装、运行与管控功能</td>
<td style="text-align: left;">64</td>
<td style="text-align: left;">终端为win7及以下版本，或32位版本</td>
</tr>
<tr>
<td
style="text-align: left;">与{驱动名}驱动冲突，潜在导致网络访问异常，已降级网络管控功能</td>
<td style="text-align: left;">128</td>
<td
style="text-align: left;">Windows客户端和联想电脑管家的网络驱动不兼容问题</td>
</tr>
<tr>
<td
style="text-align: left;">注册表被删除或篡改，可能将影响客户端全部功能</td>
<td style="text-align: left;">16384</td>
<td style="text-align: left;">注册表被删除或篡改</td>
</tr>
<tr>
<td
style="text-align: left;">终端文件被删除或篡改，可能将影响客户端全部功能</td>
<td style="text-align: left;">8192</td>
<td style="text-align: left;">终端文件完整性被破坏</td>
</tr>
<tr>
<td style="text-align: left;">终端开机启动项被禁用</td>
<td style="text-align: left;">4096</td>
<td style="text-align: left;">终端开机启动项被禁用</td>
</tr>
<tr>
<td style="text-align: left;">终端检测到被调试，存在对抗行为</td>
<td style="text-align: left;">1024</td>
<td style="text-align: left;">终端被调试</td>
</tr>
<tr>
<td rowspan="4" style="text-align: center;">macOS</td>
<td style="text-align: left;">磁盘未授权，将影响客户端全部功能</td>
<td style="text-align: left;">1</td>
<td style="text-align: left;">完全磁盘访问权限未授权</td>
</tr>
<tr>
<td style="text-align: left;">录屏未授权，无法取证外发操作</td>
<td style="text-align: left;">2</td>
<td style="text-align: left;">录屏未授权</td>
</tr>
<tr>
<td style="text-align: left;">网络未授权，将影响联网管控功能</td>
<td style="text-align: left;">4</td>
<td style="text-align: left;">网络未授权</td>
</tr>
<tr>
<td style="text-align: left;">macOS版本在10.15以下版本</td>
<td style="text-align: left;">8</td>
<td style="text-align: left;">操作系统版本过低</td>
</tr>
<tr>
<td style="text-align: center;">Linux/UOS/Kylin</td>
<td style="text-align: left;">内核版本匹配异常，部分功能不可用</td>
<td style="text-align: left;">16</td>
<td
style="text-align: left;">Linux/UOS/Kylin内核版本处于客户端不支持的版本</td>
</tr>
</tbody>
</table>

4.5.7 **应用身份**

记录终端用户登录Microsoft Word、PowerPoint、Excel的账号信息。

<img src="../images/image-082.png" style="width:5.75in;height:2.84375in" />

<span id="_Toc109552251" class="anchor"></span>4.6 **系统日志**

4.6.1 **系统登录**

记录系统账户登录时的日志。

<img src="../images/image-083.png" style="width:5.75in;height:3.25in" />

4.6.2 **系统操作**

记录系统账户在系统内操的日志，用于行为审计。

<img src="../images/image-084.png" style="width:5.75in;height:3.03125in" />

<span id="_Toc851197078" class="anchor"></span>4.7 **远程操作**

4.7.1 **远程删除**

显示远程文件删除操作的日志，对于执行失败的任务，点击操作列的「触发」可再次下发任务。

<img src="../images/image-085.png" style="width:5.75in;height:1.61458in" />

4.7.2 **远程提取**

显示远程文件提取操作的日志。对于执行失败的任务，点击操作列的「触发」可再次下发任务。对于执行成功的任务，可以点击「下载」按钮，下载提取成功的文件。

<img src="../images/image-086.png" style="width:5.75in;height:3.39583in" />

<span id="_Toc1680717279" class="anchor"></span>5. **事件洞察**

根据多个行业实践，DDR内置了功能强大的日志分析框架。企业管理员可以根据企业内部安全管理要求进行差异化管理，通过事件洞察功能构建定制化的事件生成规则，降低误报日志带来的运营成本。

<span id="heading_57" class="anchor"></span>5.1 **事件分析**

企业管理员可在此页面查看哪些用户或设备触发了事件策略。该页面详细列出了所有命中策略的日志，帮助您快速识别潜在的安全威胁或违规行为。通过事件分析页面，您还可以深入了解每个事件的具体细节，包括触发条件、时间戳和相关数据源，方便进一步的调查和处理。

<img src="../images/image-087.png" style="width:5.75in;height:3.52083in" />

**添加关注用户**

在系统中，您可以通过左侧边栏的“全部用户”选项来选择您需要关注的用户。例如，如果某位用户即将离职，您可以将其添加到关注列表中。这样，系统将优先监控该用户的活动，帮助您及时发现并应对潜在的安全风险。

<img src="../images/image-088.png" style="width:5.75in;height:4.84375in" />

<span id="heading_58" class="anchor"></span>5.2 **事件配置**

您可以在事件配置中添加事件捕获规则，以聚合和统计的方式分析员工的异常操作行为。针对常见场景，DDR提供了丰富的预设事件策略，您可以轻松应用各种预设规则模板，快速构建适合自己的规则组。

**新增事件配置**

1.  在「事件洞察-事件配置」页面，点击"事件配置"

<img src="../images/image-089.png" style="width:5.75in;height:2.46875in" />

2.  在"事件配置"页面，依次完成以下配置

<img src="../images/image-090.png" style="width:5.75in;height:4.4375in" />

<table style="width:89%;">
<colgroup>
<col style="width: 9%" />
<col style="width: 79%" />
</colgroup>
<tbody>
<tr>
<td style="text-align: left;">基本信息</td>
<td style="text-align: left;">说明</td>
</tr>
<tr>
<td style="text-align: left;">事件名称</td>
<td
style="text-align: left;">为新事件输入一个具有描述性的名称，便于识别和管理。</td>
</tr>
<tr>
<td style="text-align: left;">事件类型</td>
<td
style="text-align: left;">选择事件的类型。例如：外发行为、对抗行为、下载行为</td>
</tr>
<tr>
<td style="text-align: left;">威胁分值</td>
<td
style="text-align: left;">设置该事件的分值，用于评估其事件严重性。</td>
</tr>
<tr>
<td style="text-align: left;">事件描述</td>
<td
style="text-align: left;">提供对事件的详细描述，帮助理解事件的背景和潜在影响。</td>
</tr>
<tr>
<td style="text-align: left;">数据来源</td>
<td style="text-align: left;">选择该事件使用的日志数据来源</td>
</tr>
<tr>
<td style="text-align: left;">命中条件</td>
<td
style="text-align: left;">定义触发该事件的必要条件，支持设置一个或多个参数，按需增加判断条件。</td>
</tr>
<tr>
<td style="text-align: left;">检测时间窗</td>
<td
style="text-align: left;">设定用于检测该事件的时间窗口。例如，若在您指定的时间窗口内，日志命中次数达到Y条，系统将自动触发警报。</td>
</tr>
<tr>
<td style="text-align: left;">命中日志</td>
<td style="text-align: left;"><p>可选固定值或者智能基线值。</p>
<p>•
<strong>固定值</strong>：设定一个具体的数值作为阈值。例如，若在指定的X小时内，日志命中次数达到您设定的固定值，系统将触发告警。</p>
<p>•
<strong>智能基线值</strong>：阈值是通过机器学习算法根据您所在部门的平均日志活动量动态计算的。这种方法能够根据日志活动的变化自动调整，从而更精准地识别异常行为。</p></td>
</tr>
<tr>
<td style="text-align: left;">通知告警</td>
<td
style="text-align: left;">配置在事件触发时的通知和告警机制，如邮件通知或IM机器人消息通知。</td>
</tr>
</tbody>
</table>

3.  使用“当前条件预览”功能对命中条件进行数据预览。系统将根据您设定的条件筛选日志，并显示命中的日志数量和基线。您可以根据数据预览判断设定的阈值是否合理。

<img src="../images/image-091.png"
style="width:5.4375in;height:4.21875in" />

4.  配置完成后，保存设置。在事件配置页面启用您刚配置的策略。策略将在大约15分钟后开始对日志进行分析和计算。

<img src="../images/image-092.png" style="width:5.75in;height:1.03125in" />

<span id="_Toc252446480" class="anchor"></span>6. **统计报告**

<span id="_Toc1587786535" class="anchor"></span>6.1 **报告视图**

企业管理员可根据需要选择相应的日志信息配置图表，建立可视化数据安全指标，以不断提升数据安全水平。

<img src="../images/image-093.png" style="width:5.75in;height:2.96875in" />

<img src="../images/image-094.png" style="width:5.75in;height:3.54167in" />

1.  新建视图及图表

所有视图以卡片形式展现，若要添加新视图，点击页面右上角「新建视图」，完成相应信息填写后进入新建图表页面。根据页面提示进入创建图表页面。

<img src="../images/image-095.png" style="width:5.75in;height:2.625in" />

<img src="../images/image-096.png" style="width:5.75in;height:2.96875in" />

- 内置图表

长亭终端统一管控与安全检测响应平台 DDR
根据客户需求内置一批图表。企业管理员可按需选择，开箱即用。点击右上角「新建图表」，选择相应图表创建即可。

<img src="../images/image-097.png" style="width:5.75in;height:3.98958in" />

- 自定义图表

企业管理员可根据企业实际需要构建自定义图表。点击右上角「新建图表」，选择「自定义图表」，进入页面后按照信息配置内容保存后完成自定义图表配置。

界面左侧为数据配置及数据分析区，右侧为图表显示区。企业管理员在左侧选择需要的数据集字段列表，配置相应的图表分析后，右侧将根据左侧的筛选条件，自动提取数据并生成图表。

<img src="../images/image-098.png" style="width:5.75in;height:3.42708in" />

名词解释

- 维度

维度划分数据类别与观察角度，是没有聚合状态的字段，如发生时间、员工姓名等。

- 指标

指标是以不同聚合方式度量数据，大部分情况下是数字，比如风险等级总条数等。

如：查看同一语文成绩的学生共有几名，则可将“语文成绩”配为维度，将"学生（计数）"（即学生计数）配为指标。

2.  视图管理

企业管理员可对产品中的所有视图执行删除、查看详情操作。同时可直接对视图进行新建订阅和订阅管理操作。

3.  图表管理

视图中的每个图表都支持编辑和删除。企业管理员可编辑内置图表的名称等信息；自定义图表除了以上功能外，还支持进一步对结果数据筛选、排序，以及前往对应的日志源查看源数据三种功能。

<img src="../images/image-099.png" style="width:5.75in;height:3.13542in" />

<img src="../images/image-100.png" style="width:5.75in;height:3.125in" />

<img src="../images/image-101.png" style="width:5.75in;height:3.72917in" />

4.  示例：查看近七天风险传输渠道 TOP 10

第一步：新建视图，取名“风险视图”

第二步：在“风险视图”中，新建名为“近七天风险传输渠道 TOP
10”图表，数据源选择“渠道管控日志”

第三步：数据配置中选择时间“过去七天”，维度选择“渠道名称”、“风险等级”；指标选择“风险等级（计数）”

<img src="../images/image-102.png" style="width:5.75in;height:3.32292in" />

第四步：根据查询的数据明细，选择合适的图形进行绘制。如选择柱状图，横轴选择“渠道名称”，纵轴选择“风险等级（计数）”，分组轴选择“渠道名称”

<img src="../images/image-103.png"
style="width:5.70833in;height:7.04167in" />

第五步：根据绘图结果可简单调整数据展现方式。切换至「图表分析」界面，选择计算类型“截取N条维度项”，维度选择“渠道名称”，数量选择“10”，排序依据“风险等级（计数）”，降序。（在数据结果中按照渠道名称维度截取风险等级数量为前10的结果）

<img src="../images/image-104.png" style="width:5.75in;height:2.5625in" />

第六步：保存图表

<img src="../images/image-105.png" style="width:5.75in;height:3.92708in" />

<span id="heading_61" class="anchor"></span>6.2 **报告订阅**

企业管理员可通过订阅报告视图，定时查收长亭终端统一管控与安全检测响应平台
DDR
推送的视图信息，密切跟进数据近况。同时产品提供的订阅管理功能，可以协助企业管理员管理自己的订阅记录，并支持对报告视图的订阅信息进行修改。

1.  新建订阅

点击页面右上角选项，点击「新建订阅」填写相应信息即可创建一条新订阅

<img src="../images/image-106.png" style="width:5.75in;height:1.95833in" />

完成后，点击「发送测试」。如果数据填写无误，将会接收到一条订阅的测试（比如此处设置的是「风险事件视图-日报」定时推送机器人）

<img src="../images/image-107.png" style="width:5.75in;height:8.05208in" />

<img src="../images/image-108.png" style="width:5.75in;height:4.34375in" />

确认测试内容无误后，可点击「确定」保存设置

2.  订阅管理

已保存的订阅可以在「统计报告」-「报告订阅」中进行修改和批量管理，或从具体的报告视图点击「订阅管理」查看与本视图相关订阅信息，进行编辑管理。

<img src="../images/image-109.png" style="width:5.75in;height:2.78125in" />

3.  订阅配置

- 订阅周期：支持按日、周、月周期性进行报告推送。推送时会将当时的视图信息生成
  PDF 一同发送。

<!-- -->

- 发送方式：当前支持通过邮件进行报告推送。选择邮件通知渠道，输入邮件发送主题；企业管理员可根据实际需要开启或关闭主题后是否附加发送时间的选项。

<img src="../images/image-110.png" style="width:5.75in;height:4.77083in" />

- 发送内容：开关开启后，将在邮件正文中包含相应信息。产品默认内置正文内容，企业管理员也可按需自定义创建信息。

<img src="../images/image-111.png" style="width:5.75in;height:3.625in" />

4.  订阅详情

每一条订阅信息在执行完成一次推送后，均会生成记录。点击「详情」，可查看每次订阅的推送信息。

<img src="../images/image-112.png" style="width:5.75in;height:6.5in" />

<span id="_Toc1841256799" class="anchor"></span>7. **网络管理**

<span id="_Toc763667523" class="anchor"></span>7.1 **联网管控**

联网管控可对企业员工上网行为进行细粒度管控，规避法律法规风险。

<img src="../images/image-113.png" style="width:5.75in;height:3.5in" />

- **实体白名单**

产品支持为特权人群配置“实体白名单”，实体白名单内的人群将具备在终端上任意使用软件的特权。不会受到管控策略的影响。

点击左侧目录栏「网络管理」-
「联网管控」，进入联网管控页面，点击界面「实体白名单」。企业管理员可以根据需要将终端、用户和部门的信息纳入实体白名单中，以实现个性化的管理需求。

<img src="../images/image-114.png" style="width:5.75in;height:0.84375in" />

- 管控白名单

产品支持在软件策略配置中添加联网管控的白名单，并限定只有在白名单中的软件才能够进行联网操作，或被纳入进程白名单中的进程将不受管控策略影响。通过设置白名单，可以精确控制企业内部允许访问互联网的软件，从而加强对联网行为的管控和安全性保障。有助于维护网络安全、降低潜在风险，并确保员工在联网过程中的合规性。需要注意的是联网管控白名单中的HTTP/HTTPS数据将与「泄漏管控」-「外发管控」白名单数据联动。

点击左侧目录栏「网络管理」-
「联网管控」，点击界面「管控白名单」，填写相应信息。

<img src="../images/image-115.png" style="width:5.75in;height:0.86458in" />

|  |  |
|:--:|:--:|
| <img src="../images/image-116.png"
style="width:2.32292in;height:1.55208in" /> | <img src="../images/image-117.png"
style="width:3.08333in;height:1.57292in" /> |

- **管控配置**

<img src="../images/image-118.png" style="width:5.75in;height:1.78125in" />

1.  域名管控

对于 Windows
系统，当使用域名作为管控条件时，建议开启域名管控开关，以防出现策略不生效问题。

2.  弹窗间隔

企业管理员可在此处配置弹窗间隔时间。

3.  处置模板

企业管理员可根据管控策略，在处置模板界面定制终端弹窗提醒模板，设定弹窗显示时长，并根据不同操作系统的语言环境，配置相应的提示信息。

<img src="../images/image-119.png" style="width:5.75in;height:7.52083in" />

7.1.1 **策略配置**

企业管理员可以通过配置策略，进而有效监控和控制相关软件联网行为，从而保护数据的安全。

- **基本信息**

包含策略名称、描述。

- **生效范围**

包含策略的下发范围：策略是对系统内全部人员及设备生效，或对部分被选中的人员生效。

- **管控处置**

生效时段及周期：限制策略在某时间段内是可用的。

管控模式：在检测渠道中至少需要选择一个进程渠道，可选择全局进程或指定进程。

处置动作：根据需要配置进程联网的不同模式（优先级：允许联网 \> 限定禁网
\> 限定联网 \> 禁止联网）

处置模板：根据需要选择对应的处置响应模板。命中策略后将根据配置进行弹窗通知。

1.  创建策略

点击界面右上角「创建策略」，完成相应信息填写后保存。

<img src="../images/image-120.png" style="width:5.75in;height:4.0625in" />

<img src="../images/image-121.png" style="width:5.75in;height:5.61458in" />

- 管控模式

企业管理员可在此处配置需要管控的进程。为了方便配置，产品支持通过不同方式来添加管控进程：可以手动新增软件信息，批量上传、或者从已获取的软件信息中选择需要进行管控的软件。通过这些方式，企业管理员可以轻松管理和配置管控源，确保软件行为符合要求。

<img src="../images/image-122.png" style="width:5.75in;height:3.66667in" />

<img src="../images/image-123.png" style="width:5.75in;height:2.0625in" />

- 处置动作

“处置动作”是指对软件的联网行为进行管理和控制的功能。产品提供了多种方式来实现联网管控，以满足不同的需求。

第一种方式是允许联网，企业管理员可以选择将某些软件添加到管控源中，使其与互联网进行通信的同时，记录相关的行为日志。

第二种种方式是禁止联网，企业管理员可以选择将某些软件添加到管控源中，使其无法与互联网进行通信。进而提高系统的安全性和隐私保护。

第三种方式是限定联网，企业管理员可以根据具体需求，设定联网条件来允许被选定的软件或进程进行联网操作。通过配置合适的条件，产品可以灵活地控制软件的联网权限，确保在满足条件的情况下正常进行联网操作。

第四种方式是限定禁网，企业管理员可以根据具体需求，设定联网条件来禁止被选定的软件或进程在一定范围内进行联网操作。

以上四种模式的优先级如下：允许联网 \> 限定禁网 \> 限定联网 \>
禁止联网。通过联网管控功能，产品能够有效地管理和控制软件的联网行为，保护系统的安全性并提供灵活的网络连接配置。

<img src="../images/image-124.png" style="width:5.75in;height:4.02083in" />

2.  查看策略

点击左侧目录栏「网络管理」-
「联网管控」，可在界面查看所有已配置的策略，企业管理员可通过点击启用状态的开关开启或关闭策略。点击策略详情，即可浏览和编辑策略的详细信息。

<img src="../images/image-125.png" style="width:5.75in;height:2.125in" />

<span id="_Toc1597784589" class="anchor"></span>8. **桌面管理**

8.1 **终端管理**

企业管理员可对入网的终端进行安全管理。

<img src="../images/image-126.png" style="width:5.75in;height:3.48958in" />

8.1.1 **终端配置**

为保障终端的统一可用，且不影响员工正常工作。长亭终端统一管控与安全检测响应平台
DDR 为企业管理员提供了丰富的终端配置项：终端运行资源、云端通信、macOS
授权、高级配置。

- 运行资源

企业管理员可在本页面配置 DDR Agent 终端运行时的CPU、存储占用信息等。

<img src="../images/image-127.png" style="width:5.75in;height:4.41667in" />

- 云端通信

网络配置：若企业管理员期望保障终端与控制台正常通信，可在保证网络联通的情况下在「云端通信」-「新增配置」页面中添加多条网络配置信息。终端将会根据页面信息尝试网络配置连接，直至联通成功为止。

<img src="../images/image-128.png" style="width:5.75in;height:2.55208in" />

通信连接：企业管理员可在此页面进行终端、控制台之间的通信时间配置，包括心跳、策略的同步时间以及终端状态信息判定时间。

<img src="../images/image-129.png" style="width:5.75in;height:4.48958in" />

- macOS 授权

因 macOS 系统的特殊性，企业管理员若想在 macOS
使用长亭终端统一管控与安全检测响应平台 DDR
的全功能，需要员工在终端设备开启授权。针对未开启授权的终端用户进行提醒拒不授权的事件进行通知告警。

- 完全磁盘访问：开关关闭后，不再检查完全磁盘访问权限（已授权不影响），授权弹窗不再显示相关引导。未授权用户仅保留数据行为能力；

<!-- -->

- 屏幕录制检查：允许企业管理员根据需要设置生效范围；开关关闭后，不再检查屏幕录制授权（已授权不影响），授权弹窗不再显示相关引导。未授权终端用户将无法使用截屏/录屏取证功能；部分开启后，将对macOS
  14以下版本系统开启屏幕录制功能；

<!-- -->

- 网络拓展检查：允许企业管理员根据需要设置生效范围；开关关闭后，不再检查网络扩展（已授权不影响），授权弹窗不再显示相关引导。未授权用户无法使用联网管控功能进行管控。开启后，员工一旦根据引导开启授权，则无法手动撤销该权限；

<!-- -->

- 提醒频率：若终端用户不愿为长亭终端统一管控与安全检测响应平台DDR
  授权，长亭终端统一管控与安全检测响应平台 DDR
  将按照配置频率弹窗提醒员工进行授权；

<!-- -->

- 提醒告警：当弹窗累计多次而员工仍不授权时，系统将会通知有权限的企业管理员协调处理；

<!-- -->

- 通知告警设置：可通过邮件、IM机器人等多种方式对拒不授权的事件进行通知告警。

<img src="../images/image-130.png" style="width:5.75in;height:3.14583in" />

- 高级配置

企业管理员可在此处开启或关闭终端OCR、隐藏进程等开关。

- 终端OCR：启用后，终端将对图像文件执行文字提取与识别操作

<!-- -->

- 压缩文件：启用后，系统将依据配置，对压缩文件执行多层解压和内容识别

<!-- -->

- 内嵌图片解析：启用后，系统将提取文件中内嵌的图像信息。该功能性能开销较高，建议默认关闭

<!-- -->

- 二进制解析：启用后，系统将对非常见格式的文件进行二进制流扫描。该功能性能开销较高，建议默认关闭

<!-- -->

- 内容摘要配置：支持自定义上报内容摘要的长度，用于信息提取与上报

<!-- -->

- 安全模式管控：启用后，将禁止终端在 Windows 安全模式下进行外设挂载操作

<!-- -->

- 隐藏进程：启用后，将隐藏客户端进程在 Windows
  任务管理器中的显示信息，增强客户端隐蔽性

<!-- -->

- Agent退出码：可通过图形界面输入退出码，以终止客户端进程

<!-- -->

- JS- SDK配置：内部web应用接入使用

<img src="../images/image-131.png" style="width:5.75in;height:3.76042in" />

<table style="width:88%;">
<colgroup>
<col style="width: 88%" />
</colgroup>
<tbody>
<tr>
<td style="text-align: left;"><p><strong>Agent 退出</strong></p>
<p><strong>自动获取退出码</strong></p>
<p><strong>获取Agent退出码并在客户端Agent中输入，即可关闭客户端的功能。</strong></p>
<ol type="1">
<li><p>登录产品后台界面，选择导航栏「桌面管理」，点击右上角「终端管理」-「终端配置」-「高级配置」，找到「Agent退出码」，点击「配置」。</p></li>
</ol>
<p><img src="../images/image-132.png"
style="width:5.58333in;height:1.60417in" /></p>
<ol start="2" type="1">
<li><p>复制退出码。</p></li>
</ol>
<p>退出码有效期为24小时。</p>
<p><img src="../images/image-133.png"
style="width:5.58333in;height:3.19792in" /></p>
<ol start="3" type="1">
<li><p>在需要退出的设备中，进入GUI界面点击「设置」-「退出Agent」，输入退出码。即可关闭客户端功能。</p></li>
</ol>
<p><img src="../images/image-134.png"
style="width:5.58333in;height:3.52083in" /></p>
<p>终端页面：macOS输入退出码</p>
<p><img src="../images/image-135.jpeg"
style="width:5.58333in;height:3.58333in" /></p>
<p>终端页面：Windows输入退出码</p>
<p><img src="../images/image-136.jpeg"
style="width:5.58333in;height:4.03125in" /></p>
<p>终端页面：Linux/Kylin/UOS输入退出码</p>
<p><strong>手动配置退出码</strong></p>
<p><strong>获取Agent退出码并在客户端Agent中输入，即可关闭客户端的功能。</strong></p>
<p>请注意，手动输入的退出码可能会导致离线设备退出失败。</p>
<ol type="1">
<li><p>登录产品后台界面，选择导航栏「桌面管理」，点击右上角「终端管理」-「终端配置」-「高级配置」，找到「Agent退出码」，点击「配置」。</p></li>
</ol>
<p><img src="../images/image-132.png"
style="width:5.58333in;height:1.60417in" /></p>
<ol start="2" type="1">
<li><p>开启「手动输入」开关，可随机生成或自定义退出码</p></li>
</ol>
<p><img src="../images/image-137.png"
style="width:5.58333in;height:3.6875in" /></p>
<ol start="3" type="1">
<li><p>记录退出码并保存</p></li>
</ol>
<ol start="4" type="1">
<li><p>在需要退出的设备中，进入GUI界面点击「设置」-「退出Agent」，输入退出码。即可关闭客户端功能。</p></li>
</ol>
<p><img src="../images/image-134.png"
style="width:5.58333in;height:3.52083in" /></p>
<p>终端页面：macOS输入退出码</p>
<p><img src="../images/image-135.jpeg"
style="width:5.58333in;height:3.58333in" /></p>
<p>终端页面：Windows输入退出码</p>
<p><img src="../images/image-138.png"
style="width:5.58333in;height:3.6875in" /></p>
<p>终端页面：Linux/Kylin/UOS输入退出码</p></td>
</tr>
</tbody>
</table>

- 数据行为

企业管理员可以在此处对如下的数据行为日志信息进行配置。包括：

- 关注数据下载行为的来源，即关注从哪些地方下载了数据；

<!-- -->

- 关注哪些文件进行创建、复制、移动等操作；

<!-- -->

- 配置是否监控文件的打开行为日志记录等。

通过配置这些信息，企业管理员可以更好地监控和管理企业的数据行为，以保障数据的安全和合规性。

<img src="../images/image-139.png" style="width:5.75in;height:4.66667in" />

8.1.2 **终端列表**

长亭终端统一管控与安全检测响应平台 DDR
提供终端管理能力，企业管理员可在本页面查看终端信息，包括不同版本的数据统计、异常终端统计等。同时页面以卡片形式展示当前已安装
Agent 的终端信息，点击详情可查看具体信息。

1.  **版本分布**

依据各操作系统下最新的5个 Agent 版本信息。帮助企业管理员快速掌握 Agent
版本升级/安装情况。

2.  **信息统计**

根据既定的信息，产品内置统计异常状态的终端信息。帮助企业管理员快速了解
Agent 异常统计情况。

3.  **终端信息**

以卡片形式展示终端信息。包括但不限于终端安装 Agent
版本信息、启用停用状态、终端操作系统、绑定信息、健康及合规情况、上报IP及时间信息等；若要查看更详细的信息，点击查看详情可查看具体终端的详细信息。

<img src="../images/image-140.png" style="width:5.75in;height:3.13542in" />

- **终端状态说明**

1.  终端依据存活状态可分为：在线、离线、失效（长期处于离线状态）三种状态；依据生命周期可分为：已安装、卸载中、已卸载；依据健康状态可分为：运行正常、运行异常；依据基线状态，可分为：合规、不合规

<!-- -->

2.  终端设备卸载Agent后，列表中的终端状态将变为已卸载。已卸载状态的设备不会在列表上显示，可通过筛选「已卸载终端」查看全部终端信息

<!-- -->

3.  默认当终端超过10天无心跳时，终端列表将显示“失效”（时间可至「终端配置」-「通信连接」-「失效时间」进行配置）。此时企业管理员需要线下排查终端的情况，查看失效原因

<!-- -->

4.  当终端卸载时，页面将显示“卸载中”状态。若通过在线卸载进行卸载，卸载完成后状态将变为已卸载；若通过离线卸载进行卸载，管理员需手动点击“完成卸载”按键

<!-- -->

5.  若 macOS
    的终端未完全授权设备权限、Linux/UOS/Kylin的终端内核版本适配异常，会提示运行异常

- **Agent启停说明**

当企业管理员需要关停某台设备上 Agent
运行时，可以通过设备的Agent启停来控制。点击启用/停用开关状态可进行Agent
的启停。当终端设备上的Agent出现故障时，建议使用此功能。

6.  **终端详情**

- 基础信息

显示终端硬件信息、上报IP、终端标签等内容。

<img src="../images/image-141.png" style="width:5.75in;height:3.59375in" />

- 一键诊断

如遇终端出现问题，可点击「一键诊断」获取诊断日志，并将日志传给长亭科技工程师，辅助分析。

<img src="../images/image-142.png" style="width:5.75in;height:2.29167in" />

- 配置信息

显示属于本终端的状态信息、硬件信息、软件信息、数据资产、执行策略、合规基线、外泄风险信息。

- 状态信息：展示终端设备在一段时间内的CPU、内存、磁盘及运行进程信息。

<!-- -->

- 硬件信息：展示终端设备的硬件信息（包括系统型号、内存、CPU等内容），终端网卡信息。

<!-- -->

- 软件信息：展示特定时间点所采集的终端设备上已安装软件的详细信息，包括常规软件及盗版软件（若启用）检测信息。

<!-- -->

- 数据资产：展示当前终端上的数据资产以及数据分类及分类分布结果。默认展示数据采集情况，点击「详情」可跳转至资产扫描或文件治理界面；点击右上角「分类分级」，切换至数据分类分级信息，点击「详情」可跳转至数据资产地图页面，查看本终端拥有的数据文件信息。

<!-- -->

- 执行策略：展示该终端关联策略信息

<!-- -->

- 合规基线：展示已配置、未配置的基线信息，以及终端不合规基线的详细信息

<!-- -->

- 外泄风险：以日历盘的形式展示终端命中渠道策略的信息，点击不同日期将打开对应日期、对应终端的风险日志信息

<img src="../images/image-143.png" style="width:5.75in;height:3.29167in" />

- Agent 信息

显示 Agent
的存活状态、版本名称、策略版本、注册时间、上报时间、卸载时间信息。

若要卸载终端，点击「在线卸载/离线卸载」。若终端能联通产品，会直接执行卸载流程；若无法联通，则需要通过输入卸载码，手动卸载设备。

<img src="../images/image-144.png" style="width:5.75in;height:4.57292in" />

- 所属信息

展示设备绑定员工信息（包含设备的历史绑定信息），关联方式，上级信息，部门信息等

<img src="../images/image-145.png" style="width:5.75in;height:5.63542in" />

7.  **终端标签**

产品支持为不同终端配置终端标签信息。点击「终端标签」，可以对终端所有的标签信息进行管理。

|  |  |
|:--:|:--:|
| <img src="../images/image-146.png"
style="width:3.09375in;height:1.69792in" /> | <img src="../images/image-147.png"
style="width:2.3125in;height:1.67708in" /> |

<img src="../images/image-148.png" style="width:5.75in;height:2.4375in" />

8.1.3 **终端升级**

1.  登录管理后台，进入「桌面管理」-「终端管理」-「终端升级」页面，点击页面右上角「导入安装包」。

<img src="../images/image-149.png" style="width:5.75in;height:1.65625in" />

- 将厂商提供的安装包上传，根据需要选择是否立即发布，而后点击「确定」完成操作。

<img src="../images/image-150.png" style="width:5.75in;height:4.96875in" />

2.  根据实际需求，结合已上传的安装包版本，对符合条件的办公域内终端进行批量升级。

- 升级模板：登录管理后台，点击左侧目录栏「桌面管理」-「终端管理」，切换至「终端升级」页面，点击右上角「升级配置」-「升级模块」。企业管理员可以手动设置批次升级模板。通过使用预设的升级模板，简化升级流程，避免每次升级时重复配置相同部门或员工的设备，提高工作效率。

<img src="../images/image-151.png" style="width:5.75in;height:6.3125in" />

DDR管理后台：配置升级模板

- **待更新：**等待发布的安装包。

<img src="../images/image-152.png" style="width:5.75in;height:3.48958in" />

DDR管理后台：等待发布的安装包

- **已发布：**当前发布的终端包信息。

<img src="../images/image-153.png" style="width:5.75in;height:1.41667in" />

当前升级中的批次不支持添加新设备。如需添加更多设备，请点击下一批次并编辑设备信息以扩展发布范围。或者，您也可以选择等待当前任务执行至最后一批，系统将自动对办公域内所有符合新版本条件的设备进行升级。

<img src="../images/image-154.png" style="width:5.75in;height:2.8125in" />

DDR管理后台：添加新设备升级

- **已归档：** 所有已使用过的终端安装包均会保存于此，以供记录和参考。

<img src="../images/image-155.png" style="width:5.75in;height:3.125in" />

DDR管理后台：已归档的安装包

- **回滚版本：**点击发布记录中的「回滚」，可将终端回滚至某一版本

<img src="../images/image-156.png" style="width:5.75in;height:2.32292in" />

- **个体升级（建议先咨询厂商）：**
  允许选择单个或部分设备进行升级，不推荐客户自行操作，使用前请咨询厂商以确保正确执行。

<img src="../images/image-157.png" style="width:5.75in;height:2.92708in" />

DDR管理后台：个体发布状态

若要添加更多设备进行个体发布，请访问发布详情页面并点击右侧「编辑终端」，以增加发布设备。

<img src="../images/image-158.png" style="width:5.75in;height:2.44792in" />

8.1.4 **下载管理**

注：由于 Windows 7 操作系统暂不支持 SHA-2
代码签名并且缺少.net依赖包，因此终端安装Agent前需要打以下补丁：

- SHA-2
  代码签名补丁链接：<https://www.microsoft.com/zh-cn/download/details.aspx?id=46148>

<!-- -->

- .net依赖补丁：<https://go.microsoft.com/fwlink/?LinkId=2088631>

1.  下载安装包

登录管理后台，点击左侧目录栏「桌面管理」-「终端管理」，进入页面选择「下载管理」，在页面中选择对于系统及架构、选定通信地址及校验方式，随后即可下载所需安装包。下载完成后，可将安装包分发给员工，以便他们在个人设备上安装客户端。

|  |
|:---|
| 通信地址校验：启用校验功能可在终端安装过程中，自动检查设备与服务器之间的通信状态。跳过校验步骤安装虽快，但可能引起后续通信管理问题。**推荐使用校验功能。** |

2.  配置下载链接

为方便员工下载安装包，企业管理员仅需在管理后台简单配置，便可通过下载页面直接下载。

登录管理后台，点击左侧目录栏「桌面管理」-「终端管理」，进入页面选择「下载管理」，点击页面右上角「下载链接配置」。填写相应信息后保存（仅已发布的安装包支持通过此形式下载），复制「安装包下载页面地址」发放给员工，即可通过访问页面下载不同操作系统终端安装包。

<table style="width:88%;">
<colgroup>
<col style="width: 88%" />
</colgroup>
<tbody>
<tr>
<td style="text-align: left;"><p>通信地址：终端Agent
与云端通信地址。推荐您配置外网通信地址，确保即使员工离开办公内网也能保证终端Agent的正常管控。</p>
<p>通信地址校验：启用校验功能可在终端安装过程中，自动检查设备与服务器之间的通信状态。跳过校验步骤安装虽快，但可能引起后续通信管理问题。<strong>推荐使用校验功能。</strong></p></td>
</tr>
</tbody>
</table>

<img src="../images/image-159.png" style="width:5.75in;height:4.95833in" />

<img src="../images/image-160.png" style="width:5.75in;height:1.69583in" />

员工下载安装包页面

8.1.5 **终端卸载**

针对异常终端或长期处于失效状态终端，企业管理员可根据需要执行终端卸载任务。

1.  **新建卸载任务**

点击「终端卸载」页面右上角「新建任务」界面，填写内容并保存，即可创建批量卸载任务。

<img src="../images/image-161.png" style="width:5.75in;height:2.96875in" />

2.  **查看卸载任务详情**

在列表中找到卸载任务信息，点击「详情」，可查看卸载任务执行详情。若执行任务中的终端处于卸载异常或已离线状态，推荐直接点击「离线卸载」按钮，在相应终端输入卸载码卸载。对于需要手动更正设备卸载状态的设备，企业管理员可点击「批量完成卸载」，一键完成整个批量选择完成卸载过程，省去了手动勾选的繁琐步骤。

<img src="../images/image-162.png" style="width:5.75in;height:4.46875in" />

8.1.6 **性能监控**

为保证终端Agent的可用性，产品为企业管理员提供终端性能监控界面。支持查看
Windows、macOS、Linux 等操作系统上 Agent 分别占用的
CPU、内存不同时段的平均值。

<img src="../images/image-163.png" style="width:5.75in;height:1.79167in" />

<span id="_Toc1830065235" class="anchor"></span>8.2 **合规基线**

企业管理员可配置终端合规基线，约束入网终端的安全基线。终端会根据企业管理员执行的策略配置项进行安全合规检查。

<img src="../images/image-164.png" style="width:5.75in;height:3.45833in" />

当前可进行检查的基线项包括：杀毒软件检查、禁止运行进程检查、必须运行进程检查、系统屏保检查、防火墙检查、AD域控检查、系统启用账户检查等。

当前合规基线的处置动作为：提醒和审计。

提醒：当检查结束，将会在长亭终端统一管控与安全检测响应平台客户端界面上提醒员工基线不合规，需修复。并会将检测结果的日志上传至服务端。

审计：将终端不合规的基线信息打包成日志上报服务端。

1.  **创建策略（以常规基线检测为例）**

第一步：点击界面右上角「新增策略」，即可进入策略配置界面

第二步：按照提示填写基本信息后，选择需要检测的终端操作系统，配置不同的检测项。企业管理员可为不同的操作项配置不同的处置模板。

<img src="../images/image-165.png" style="width:5.75in;height:3.5in" />

<img src="../images/image-166.png" style="width:5.75in;height:3.54167in" />

不同检测项可单独配置处置模板

<img src="../images/image-167.png" style="width:5.75in;height:3.96875in" />

配置合规策略

第三步：确认无误后，点击「保存」，完成配置。

2.  **查看策略**

点击左侧目录栏「桌面管理」-「合规基线」，进入合规基线页面。可查看所有已配置的策略，企业管理员可通过点击启用状态的开关开启或关闭策略。点击策略详情，即可浏览和编辑策略的详细信息。

<img src="../images/image-168.png" style="width:5.75in;height:2.47917in" />

**注意事项**

1.  创建策略时，需区分操作系统进行创建，不同的操作系统可进行的合规基线项并不一样。

<!-- -->

2.  创建策略时，需要配置所需的合规基线项目，并为其设置相应的处置动作和修复URL。

<!-- -->

3.  常规检查将会周期式（默认每间隔24h执行一次）执行，企业管理员可在「模块配置」-「辅助配置」界面进行手动触发，终端用户也可在GUI图形化终端界面执行「重新检查」。

- 实体白名单

如果存在一些终端设备不需要进行合规基线检查，可以将这些终端设备添加到白名单中。检查时将自动跳过对这些设备的合规检查。

点击左侧目录栏「桌面管理」-「合规基线」，进入合规基线页面，点击界面右上角「实体白名单」。企业管理员可以根据需要将终端、用户、部门、员工标签、终端标签的信息纳入实体白名单中，以实现个性化的管理需求。

<img src="../images/image-169.png" style="width:5.75in;height:1.14583in" />

- 模块配置

<img src="../images/image-170.png" style="width:5.75in;height:2.1875in" />

- 辅助配置

1.  常规检查默认每间隔24h执行一次（企业管理员可自定义配置检查周期），企业管理员可在「模块配置」-「辅助配置」界面进行手动触发

<!-- -->

2.  企业管理员可自定义设置终端GUI的展示选项，开启或关闭合规基线模块的界面显示。当开关开启时，合规基线模块将不再在终端GUI中展示。

- 处置模板

针对未通过合规检查的项，企业管理员可以通过配置终端 GUI
界面，向终端用户显示未通过检测的内容以及相应的修复指南，以提供详细的信息和操作指引。这样，终端用户可以更加方便地了解问题所在并采取正确的修复措施。

模板内容说明：

- 提示内容：终端用户在GUI界面中可以清晰地查看与要求不符的具体文案信息。

<!-- -->

- 修复建议URL：填写URL。终端用户可点击界面「修复指南」跳转至对应URL进行查看。

<img src="../images/image-171.png" style="width:5.75in;height:5.03125in" />

3.  **终端界面查看合规基线**

员工可在终端页面进行合规基线的查看。打开终端图形化（GUI）界面，点击「安全」-「合规基线」，若有不合规基线，将展示在页面中。如果员工终端为macOS操作系统，安全基线会检测
macOS 基础权限是否开启。

<img src="../images/image-172.png" style="width:5.75in;height:3.85417in" />

<img src="../images/image-173.png" style="width:5.75in;height:3.82292in" />

点击「前往查看」，可以查看具体信息。

<img src="../images/image-174.png" style="width:5.75in;height:3.67708in" />

8.3 **外设管控**

8.3.1 **❤️策略管控**

外设管控可协助企业管理员对企业内使用的外置设备（如：U盘、硬盘、光驱、蓝牙、存储卡、手机等）与终端设备的连接进行管控。支持高级模式和极速模式两种配置模式。

<img src="../images/image-175.png" style="width:5.75in;height:3.46875in" />

1.  **极速模式**

点击左侧目录栏「桌面管理」-
「外设管控」，进入外设管控界面。界面默认为极速模式，方便企业管理员快速查看和配置各外设的管控信息。

<img src="../images/image-176.png" style="width:5.75in;height:2.01042in" />

点击「编辑」，可配置各类外设的管控策略和管控动作。

<img src="../images/image-177.png" style="width:5.75in;height:5.54167in" />

- 实体白名单

产品支持为特权人群配置“实体白名单”，实体白名单内的人群将具备在终端上任意使用外设的特权。不会受到外设策略的影响。

点击左侧目录栏「桌面管理」-
「外设管控」，进入外设管控页面，点击界面「实体白名单」。企业管理员可以根据需要将终端、用户、部门、员工标签、终端标签的信息纳入实体白名单中，以实现个性化的管理需求。

<img src="../images/image-178.png" style="width:5.75in;height:1.86458in" />

- 实体黑名单

产品支持将可疑员工或风险较大的员工添加至“实体黑名单”。黑名单实体的终端将禁止使用任何外设。

点击左侧目录栏「桌面管理」-
「外设管控」，进入外设管控页面，点击界面「实体黑名单」。企业管理员可以根据需要将终端、用户、部门、员工标签、终端标签的信息纳入实体黑名单中，以实现个性化的管理需求。

<img src="../images/image-179.png" style="width:5.75in;height:2.52083in" />

- 外设白名单

产品同时支持对外设配置白名单。外设白名单将会允许在“黑名单实体”外的终端上使用外设，优先级高于管控策略。（常用于财务U盾、企业安全U盘等场景使用）。

点击左侧目录栏「桌面管理」-
「外设管控」，进入外设管控页面，点击界面「外设白名单」。企业管理员可以根据需要将外设纳入名单中。

- 处置模板

根据不同的外设管控策略，企业管理员可以在处置模板界面配置不同的终端弹窗提醒模板，以便进行相应的操作提醒。请注意，极速模式仅限弹窗提醒模板，而高级模式支持对移动存储设备配置审批动作模板。企业管理员还可自定义弹窗显示时长和中英文提示语，终端将根据系统语言自动展示相应提示。

<img src="../images/image-180.png" style="width:5.75in;height:6.84375in" />

2.  **高级模式**

基于极速模式，高级模式下企业管理员可根据企业实际办公场景进行灵活配置，以支持多种策略并按需使用。

点击左侧目录栏「桌面管理」-
「外设管控」，进入外设管控页面，点击切换为高级模式。

<img src="../images/image-181.png" style="width:5.75in;height:1.47917in" />

企业管理员可在此页面管理现有策略或创建新策略。点击「新增策略」，选择相应外设并填写必要信息后保存即可。

请注意：高级模式下，仅移动存储设备支持配置审批、弹窗提醒、审计动作模板，其余外设仅支持配置弹窗提醒、审计动作模板。

<img src="../images/image-182.png" style="width:5.75in;height:0.90625in" />

|  |  |
|:--:|:--:|
| <img src="../images/image-183.png"
style="width:3.05208in;height:2.11458in" /> | <img src="../images/image-184.png"
style="width:2.35417in;height:2.08333in" /> |

8.3.2 **移动存储**

企业管理员可通过在本页面新增移动存储信息，实现集中化管理。此外，已在渠道管控或外设管控中列入白名单的移动存储设备也会在此展示。

<img src="../images/image-185.png" style="width:5.75in;height:1.40625in" />

8.4 **软件管控**

8.4.1 **软件总览**

长亭终端统一管控与安全检测响应平台 DDR
提供了软件总览功能，可以收集各个客户端设备的软件安装信息，并将其统一展示在列表中。在软件统计列表中，可以查看各类软件的基本信息，包括名称、版本、操作系统、软件类型、Bundle
ID/厂商等。同时，还提供统计数据，如已安装终端数、采集用户数和安装率等。

企业管理员可以通过软件统计列表了解企业员工设备上高频使用的办公软件情况，从而根据实际需求进行设备软件管理。例如，对于商业付费软件，可以根据已安装终端数和采集用户数的统计信息，合理规划软件采购方案。进而可以更好地管理和控制企业的软件资源，提高工作效率和软件管理的质量。

<img src="../images/image-186.png" style="width:5.75in;height:3.76042in" />

1.  点击左侧目录栏「桌面管理」-
    「软件管控」，选择「软件总览」。在「软件总览」页面上部分，可看到软件分布信息和软件排行信息的显示。启用盗版软件检测后，可通过切换标签页（Tab），分别浏览常规软件与盗版软件的统计数据。

<img src="../images/image-187.png" style="width:5.75in;height:1.85417in" />

2.  在页面下方，用户可以查看各类软件的详细信息，涵盖软件名称、版本、适用操作系统、类型、Bundle
    ID/厂商信息以及软件的安装总数和用户数。此外，用户还可以删除特定软件信息。

<img src="../images/image-188.png" style="width:5.75in;height:1.44792in" />

3.  点击「管控」，产品将会自动将携带软件信息内容跳转至管控策略配置页面。在该页面，企业管理员可以对特定的软件进行管控策略的配置。通过设置策略，可以限制用户访问软件的行为等，从而确保数据的安全性。

<!-- -->

4.  点击「详情」，可查看软件详情信息，并支持导出数据结果。

<img src="../images/image-189.png" style="width:5.75in;height:2.9375in" />

8.4.2 **❤️运行管控**

通过配置软件管控策略，企业管理员可以有效监控和控制相关软件的运行行为，从而保护数据的安全。该策略的设置可以帮助您掌握组织内部的风险软件使用情况，并采取必要的措施来降低潜在的安全风险。

有两种方式可以创建新的软件管控策略，可以根据需要选择合适的方式创建策略。创建策略和策略模板的步骤相似，用户手册中以创建策略为例。

<img src="../images/image-190.png" style="width:5.75in;height:2.26042in" />

1.  创建策略

点击界面右上角「新增策略」，填写相应信息后点击保存即可完成创建。

<img src="../images/image-191.png" style="width:5.75in;height:3.23958in" />

<img src="../images/image-192.png" style="width:5.75in;height:4.4375in" />

- 管控源

需要进行管控的软件称为"管控源"。为了方便配置，产品支持通过不同方式来添加管控源：可以手动新增软件信息，批量上传、或者从已获取的软件信息中选择需要进行管控的软件。通过这些方式，企业管理员可以轻松管理和配置管控源，确保软件行为符合要求。

<img src="../images/image-193.png" style="width:5.75in;height:3.39583in" />

- 运行管控

"运行管控"是指对正在运行的软件进行管理和控制的功能。产品提供如阻断（阻止某些软件的运行）、弹窗提醒（在特定条件下向终端用户提供相关信息或警示）等方式来实现运行管控。灵活地配置和管理软件的运行行为，可以提高系统的安全性和管理效率。

请注意，处置动作执行优先级依次为：允许运行\>阻断运行\>审批\>弹窗提醒\>审计。若配置的处置动作设为「阻断运行」时，企业管理员可选择是否激活「结束已运行进程」功能。一旦此开关被开启，符合条件且正在运行的进程将被立即终止。

<img src="../images/image-194.png" style="width:5.75in;height:2.59375in" />

2.  查看策略

点击左侧目录栏「桌面管理」-
「软件管控」，选择「策略管控」，可在界面查看所有已配置的策略，企业管理员可通过点击启用状态的开关开启或关闭策略。点击策略详情，即可浏览和编辑策略的详细信息。

- 实体白名单

产品支持为特权人群配置“实体白名单”，实体白名单内的人群将具备在终端上任意使用软件的特权。不会受到软件管控策略的影响。

点击左侧目录栏「桌面管理」-
「软件管控」，进入软件管控页面，选择「策略管控」，点击界面「实体白名单」。企业管理员可以根据需要将终端、用户、部门、员工标签、终端标签的信息纳入实体白名单中，以实现个性化的管理需求。

<img src="../images/image-195.png" style="width:5.75in;height:1.80208in" />

- 管控白名单

产品支持在软件策略管控中配置进程白名单，从而能够更精确地控制允许运行的进程，更好地管理和控制产品上的进程行为。

点击左侧目录栏「桌面管理」-
「软件管控」，进入软件管控页面，选择「策略管控」，点击界面「管控白名单」，填写信息即可。

<img src="../images/image-196.png" style="width:5.75in;height:2.02083in" />

8.4.3 **软件分发**

不同企业常常有着各自独特的必备软件清单，为了确保每个员工终端电脑都能正确安装这些软件，通常会有以下几种方案：

1.  批量采购终端并预装必备软件：该方案旨在尽可能确保员工在获得终端后能够迅速开始工作。然而，这种方式会给管理员带来较大的工作量，特别是当员工数量众多时，该方案的执行较为困难，并且无法避免员工后续自行卸载软件的情况。

<!-- -->

2.  在线服务器上传必备软件：将必备软件上传至在线服务器，并通过邮件、公告等方式通知员工，并引导他们下载软件安装包进行安装。但是，该方案无法确保终端软件的安装率。

<!-- -->

3.  使用软件分发系统（DDR）：将必备软件链接上传至软件分发系统（DDR），通过其中的软件分发功能，高效地将企业所需的软件分发到各个终端。这种方式减少了操作步骤，提高了软件分发的效率。

综上，通过长亭终端统一管控与安全检测响应平台 DDR
来进行软件分发是一种较为好用的方案。它能够确保企业管理员能够快速将必备软件分发给员工，并提高分发的效率，同时还能够对软件安装情况进行有效管理和监控。

目前长亭终端统一管控与安全检测响应平台 DDR 支持上传 Windows 或 Mac
端的软件包链接，分发软件需要以下操作步骤：a. 添加软件 b.分发软件

1.  添加软件

点击左侧目录栏「桌面管理」-
「软件管控」，进入软件管控页面，选择「软件分发」，点击界面「新增软件包」，填写信息。

<img src="../images/image-197.png" style="width:5.75in;height:4.66667in" />

配置项说明

<table style="width:89%;">
<colgroup>
<col style="width: 18%" />
<col style="width: 69%" />
</colgroup>
<tbody>
<tr>
<td style="text-align: left;">配置项</td>
<td style="text-align: left;">说明</td>
</tr>
<tr>
<td style="text-align: left;">运行环境</td>
<td style="text-align: left;">软件包适配的具体操作系统及架构环境</td>
</tr>
<tr>
<td style="text-align: left;">上传方式</td>
<td
style="text-align: left;"><p>现支持链接下载模式，协议类型支持HTTP及HTTPS</p>
<p>注意：企业管理员需要输入能直接点击下载的安装包链接，终端将直接请求链接地址下载软件包</p></td>
</tr>
<tr>
<td style="text-align: left;">检查安装</td>
<td
style="text-align: left;"><p>开启后，在分发软件包时可检测软件包是否已安装至员工电脑，若已安装，则不会继续进行后续的安装流程。若未开启开关，将会直接安装，有概率覆盖安装员工原有软件，请谨慎关闭此开关。注意开启后需要配置以下参数信息：</p>
<ul>
<li><p>Windows 软件包：可以配置应用名称（推荐）以及进程名称，如 ToDesk
或者 ToDesk.exe，应用名称可在 Windows
系统安装的应用列表中查看。</p></li>
</ul>
<p><img src="../images/image-198.jpeg"
style="width:4.36458in;height:4.59375in" /></p>
<ul>
<li><p>Mac 软件包：配置 Bundle
ID，可以根据页面提示信息手动获取参数值。</p></li>
</ul></td>
</tr>
</tbody>
</table>

2.  分发软件

点击左侧目录栏「桌面管理」-
「软件管控」，进入软件管控页面，选择「软件分发」，在界面找到指定软件包，并在操作列，点击「详情」，在新页面中点击「新建分发」，完成配置即可创建分发任务。

<img src="../images/image-199.png" style="width:5.75in;height:3.55208in" />

<img src="../images/image-200.png" style="width:5.75in;height:5.66667in" />

配置项说明：

<table style="width:89%;">
<colgroup>
<col style="width: 20%" />
<col style="width: 68%" />
</colgroup>
<tbody>
<tr>
<td style="text-align: center;">配置项</td>
<td style="text-align: center;">说明</td>
</tr>
<tr>
<td style="text-align: center;">实体选择</td>
<td style="text-align: left;">软件包在企业内的分发范围</td>
</tr>
<tr>
<td style="text-align: center;">分发时间</td>
<td style="text-align: center;"><ul>
<li><p>立即执行：添加分发任务完成后，软件包立即分发至指定范围的员工设备内。</p></li>
</ul>
<ul>
<li><p>自定义：手动设置固定时间进行软件分发。</p></li>
</ul></td>
</tr>
<tr>
<td style="text-align: center;">分发有效期</td>
<td
style="text-align: left;">指定当前分发任务的有效时长。仅在有效期内，分发任务会向员工设备分发软件包。如果超过有效期，则分发任务自动过期，不再分发软件包。</td>
</tr>
<tr>
<td style="text-align: center;">失败重试</td>
<td
style="text-align: left;">终端获取软件包失败或安装失败后重试次数配置</td>
</tr>
<tr>
<td style="text-align: center;">运行参数</td>
<td
style="text-align: left;"><p>软件分发时，软件包采取静默安装（即安装时用户无需操作，通过默认方式直接安装）的方式完成安装。如果软件包静默安装时需要执行运行参数才能完成，则请设置运行参数。</p>
<table style="width:66%;">
<colgroup>
<col style="width: 65%" />
</colgroup>
<tbody>
<tr>
<td
style="text-align: left;"><p>软件静默安装是一种在不显示任何用户界面的情况下自动完成安装过程的方法。这对于自动化部署和批量安装软件非常有用。不同的软件安装程序可能支持不同的静默安装参数。以下是一些常见的静默安装参数及其用途：</p>
<ol type="1">
<li><p><strong>Microsoft Windows Installer</strong>：如果软件是用
Windows Installer 打包的，通常可以看到 .msi 文件。可以使用
/QB（显示基本安装进程窗口）或
/QN（不显示任何窗口，后台自动安装）参数进行自动安装。例如，使用 /QB
参数的命令可能是 msiexec /i dtools.msi /qb。</p></li>
</ol>
<ol start="2" type="1">
<li><p><strong>Windows 补丁包</strong>：对于IE增量补丁包等，可以通过添加
/q:a /r:n 参数实现静默安装。对于其他Windows补丁，可以使用 /U /N /Z 或
/passive /norestart 参数。</p></li>
</ol>
<ol start="3" type="1">
<li><p><strong>InstallShield</strong>：使用 InstallShield
技术打包的程序可以通过创建 setup.iss 文件并使用 -R
参数运行安装程序来获取静默安装参数。然后使用 setup.exe -s [-sms]
命令进行静默安装。</p></li>
</ol>
<ol start="4" type="1">
<li><p><strong>Inno Setup</strong>：Inno Setup 制作的安装文件可以使用
/SILENT 或 /VERYSILENT 参数进行静默安装。/SILENT 会在出错时提示，而
/VERYSILENT 则完全不会提示。</p></li>
</ol>
<ol start="5" type="1">
<li><p><strong>NullSoft Installation System (NSIS)</strong>：使用 NSIS
制作的安装文件，可以使用 /S 参数进行静默安装。例如，Setup.exe
/S。</p></li>
</ol>
<ol start="6" type="1">
<li><p><strong>WISE Installer</strong>：WISE 技术打包的软件可以使用 /s
参数进行自动安装。</p></li>
</ol>
<ol start="7" type="1">
<li><p><strong>腾讯系软件</strong>：腾讯系软件如QQ、微信等通常使用 /S
参数进行静默安装，而腾讯会议使用 /SilentInstall=0 参数。</p></li>
</ol>
<ol start="8" type="1">
<li><p><strong>其他特定软件</strong>：例如，7-Zip 使用 -y /q /r:n
参数，迅雷使用 /S 参数进行静默安装。</p></li>
</ol>
<p><img src="../images/image-201.png"
style="width:4.09375in;height:4.14583in" /></p>
<p>请注意，这些参数可能会因软件版本或特定安装程序的不同而有所变化。在实施静默安装之前，最好查阅软件的官方文档或使用
/? 参数查询可用的静默安装选项。</p></td>
</tr>
</tbody>
</table></td>
</tr>
</tbody>
</table>

返回任务信息页面，可查看不同任务的分发进度、周期、创建人/时间等信息。点击右侧「详情」，可查看具体分发情况。

<img src="../images/image-202.png" style="width:5.75in;height:2.40625in" />

3.  管理软件清单

点击左侧目录栏「桌面管理」-
「软件管控」，进入软件管控页面，选择「软件分发」，页面上为软件清单信息。企业管理员可在本页面对已配置的软件包进行查看和管理。

<img src="../images/image-203.png" style="width:5.75in;height:1.63542in" />

8.4.4 **软件仓库**

企业管理员可以将企业内常用软件添加至长亭终端统一管控与安全检测响应平台
DDR
管理后台软件库，软件库中的软件最终会同步至长亭终端统一管控与安全检测响应平台
DDR
客户端的软件库中，员工可通过客户端自行下载企业软件。软件库目前支持添加
Windows 系统或者 Mac 系统的软件。

1.  添加软件至客户端

点击左侧目录栏「桌面管理」-
「软件管控」，进入软件管控页面，选择「软件仓库」。在本页面单击「新增软件」，填写信息。

<img src="../images/image-204.png" style="width:5.75in;height:4.51042in" />

配置项说明：

<table style="width:89%;">
<colgroup>
<col style="width: 22%" />
<col style="width: 65%" />
</colgroup>
<tbody>
<tr>
<td style="text-align: center;">配置项</td>
<td style="text-align: center;">说明</td>
</tr>
<tr>
<td style="text-align: center;">软件名称</td>
<td style="text-align: left;">填写软件实际的中英文名称</td>
</tr>
<tr>
<td style="text-align: center;">软件类型</td>
<td style="text-align: left;">选择软件对应的类型</td>
</tr>
<tr>
<td style="text-align: center;">软件版本</td>
<td style="text-align: left;">填写软件的版本信息</td>
</tr>
<tr>
<td style="text-align: center;">发布者</td>
<td
style="text-align: left;">终端获取软件包失败或安装失败后重试次数配置</td>
</tr>
<tr>
<td style="text-align: center;">软件描述</td>
<td style="text-align: left;">填写软件的中英文描述</td>
</tr>
<tr>
<td style="text-align: center;">软件图标</td>
<td
style="text-align: left;">可选上传软件图标，图标会对应展示在长亭终端统一管控与安全检测响应平台
DDR 客户端内，如果不上传图标则使用系统默认图标。</td>
</tr>
<tr>
<td style="text-align: center;">操作系统</td>
<td style="text-align: left;">选择软件对应的操作系统，可选 Windows
系统、Mac 系统，操作系统选择 Mac 系统后，如果添加的软件需要区分 Mac 的
Apple 与 Intel 芯片，则请选中 Mac 软件包区分 Apple 与 Intel 芯片。</td>
</tr>
<tr>
<td style="text-align: center;">上传方式</td>
<td
style="text-align: left;"><p>现只支持链接下载。选择协议类型，包括HTTP及HTTPS</p>
<p>注意：企业管理员需要输入能直接点击下载的安装包链接，终端将直接请求链接地址下载软件包</p></td>
</tr>
<tr>
<td style="text-align: center;">安装方式</td>
<td
style="text-align: left;">企业管理员可根据需要选择安装方式。如果为Windows系统，可指定手动安装或者指定安装目录。指定目录：指定安装目录参数。例如
installdir。</td>
</tr>
<tr>
<td style="text-align: center;">上架市场</td>
<td
style="text-align: left;">开启后将上架至长亭终端统一管控与安全检测响应平台
DDR 客户端，员工可在软件库中选择软件进行安装</td>
</tr>
</tbody>
</table>

2.  管理软件仓库

点击左侧目录栏「桌面管理」-
「软件管控」，进入软件管控页面，选择「软件仓库」。页面上为相关信息。企业管理员可在本页面对已配置的软件信息进行查看和管理。如查看软件是否上架，下载次数等内容。已上架的软件不支持删除，软件一旦被删除不可恢复，请谨慎操作。

<img src="../images/image-205.png" style="width:5.75in;height:2.08333in" />

点击「详情」，可查看具体的软件信息。若软件处于未上架的状态，企业管理员可点击「编辑」按钮，修改软件信息。

<img src="../images/image-206.png" style="width:5.75in;height:4.34375in" />

长亭终端统一管控与安全检测响应平台DDR
客户端界面如图，员工可根据需要下载软件。

<img src="../images/image-207.png" style="width:5.75in;height:3.75in" />

8.4.5 **️软件卸载**

通过配置软件卸载任务，企业管理员可以对终端中的软件进行集中管理，及时卸载未经授权、存在安全隐患或不再使用的应用程序，从而保障终端环境的规范性和数据安全。

1.  创建任务

点击界面右上角「新增任务」，填写相应信息后点击保存即可完成创建。

<img src="../images/image-208.png" style="width:5.75in;height:4.21875in" />

<img src="../images/image-209.png" style="width:5.75in;height:2.70833in" />

- 卸载源

需要进行卸载的软件称为"卸载源"。为了方便配置，产品支持通过不同方式来添加卸载源：可以手动新增软件信息，批量上传、或者从已获取的软件信息中选择需要进行卸载的软件。

|  |
|:---|
| 注：请避免卸载系统组件、驱动程序、IE等关键软件，以免造成不必要的系统风险。 |

<img src="../images/image-210.png" style="width:5.75in;height:3.77083in" />

2.  查看任务

点击左侧目录栏「桌面管理」-
「软件管控」，选择「软件卸载」，可在界面查看所有任务，企业管理员可通过点击任务详情，查看任务执行信息。

<img src="../images/image-211.png" style="width:5.75in;height:2.90625in" />

<table style="width:89%;">
<colgroup>
<col style="width: 26%" />
<col style="width: 31%" />
<col style="width: 30%" />
</colgroup>
<tbody>
<tr>
<td style="text-align: center;"><p><img src="../images/image-212.png"
style="width:1.57292in;height:0.77083in" /></p>
<p>mac设备卸载软件</p></td>
<td style="text-align: center;"><p><img src="../images/image-213.png"
style="width:1.85417in;height:0.78125in" /></p>
<p>Windows设备卸载软件</p></td>
<td style="text-align: center;"><p><img src="../images/image-214.png"
style="width:1.8125in;height:0.78125in" /></p>
<p>Windows 卸载软件达到重试上限</p></td>
</tr>
</tbody>
</table>

- 实体白名单

产品支持为特权人群配置“实体白名单”，实体白名单内的人群将不会受到软件卸载任务的影响。

点击左侧目录栏「桌面管理」-
「软件管控」，进入软件管控页面，选择「软件卸载」，点击界面「实体白名单」。企业管理员可以根据需要将终端、用户、部门、员工标签、终端标签的信息纳入实体白名单中，以实现个性化的管理需求。

<img src="../images/image-215.png" style="width:5.75in;height:1.94792in" />

- 卸载配置

> 点击左侧目录栏「桌面管理」-
> 「软件管控」，进入软件管控页面，选择「软件卸载」，点击界面「卸载配置」。

- 终端软件卸载提示

> 产品默认以静默方式执行软件卸载，但在 Windows
> 平台中，部分软件由于为破解版本或绿色免安装版本，可能无法直接完成静默卸载。系统将在可能的情况下尝试调用卸载流程。启用此功能后，系统将在终端设备上弹窗提示用户，引导其手动完成软件卸载操作。

- 卸载重试

> 在 Windows
> 平台中，对于无法以静默方式卸载的软件，系统将尝试多次触发卸载流程，并提示用户手动完成卸载操作。为避免频繁提醒，支持在此配置卸载重试次数（默认为3次）。

<img src="../images/image-216.png" style="width:5.75in;height:1.13542in" />

8.4.6 **模块配置**

长亭终端统一管控与安全检测响应平台 DDR
为企业管理员提供了模块配置项：处置模板及软件扫描。

- 处置模板

针对不同的策略，企业管理员可以在处置模板界面配置不同的终端弹窗提醒模板，以便进行相应的操作提醒。企业管理员还可自定义弹窗显示时长和中英文提示语，终端将根据系统语言自动展示相应提示。

<img src="../images/image-217.png" style="width:5.75in;height:4.19792in" />

- 软件扫描

周期扫描：周期扫描功能可以定期对所有终端设备执行软件扫描操作，并将扫描到的软件信息上报到控制台。打开周期扫描开关后，长亭终端统一管控与安全检测响应平台
DDR 会按预设的时间间隔自动触发扫描操作，确保及时更新软件信息。

手动扫描：可以立即对所有终端设备执行一次软件扫描，将扫描结果信息即时上报到控制台。

弹窗间隔：设定软件运行管控策略中，终端触发策略后的弹窗提醒时间间隔。

盗版检测：可设置是否启用盗版检测，并可选择性地针对特定软件开启或关闭该检测功能。

盗版配置：针对Windows系统可手动配置KMS激活服务器黑名单、序列号黑名单信息。

<img src="../images/image-218.png" style="width:5.75in;height:2.60417in" />

<span id="heading_84" class="anchor"></span>9. **数据安全**

<span id="_Toc364221264" class="anchor"></span>9.1 **数据定义**

9.1.1 **分类分级**

数据分类分级为企业核心的数据安全运营手段之一。长亭终端统一管控与安全检测响应平台
DDR
内置通用的分类分级方案库，并支持通过数据分类、数据分级两种视角，切换查看分类分级情况。

<img src="../images/image-219.png" style="width:5.75in;height:2.57292in" />

9.1.1.1 **初始化**

初次进入数据分类分级页面，可选择「手动添加」「方案库选择」「导入方案包」三种模式。

- 手动添加：企业管理员可通过手动的方式添加分类集、分类等信息。

<!-- -->

- 方案库选择：企业管理员可从方案库中选择相应的方案进行复制，并添加为分类分级。

<!-- -->

- 导入方案包：企业管理员可以将数据分类分级方案包/长亭科技工程师提供的方案包导入至方案库中。

<img src="../images/image-220.png" style="width:5.75in;height:3.40625in" />

9.1.1.2 **方案库**

方案库中内置了各行各业的分类分级方案，可供企业管理员使用。通过快速复制操作，企业管理员可以高效地进行分类分级工作。同时，系统也支持企业管理员自定义方案的导入，以进一步丰富方案库的内容。

- 复制分类

点击「内置分类」，选择需要的分类，点击克隆。

<img src="../images/image-221.png" style="width:5.75in;height:3.38542in" />

<img src="../images/image-222.png" style="width:5.75in;height:2.72917in" />

- 导入方案

点击「内置分类」-「导入方案」，导入数据方案包，完成相关信息配置即可新增一种数据分类分级方案。

<img src="../images/image-223.png" style="width:5.75in;height:3.41667in" />

<img src="../images/image-224.png" style="width:5.75in;height:1.97917in" />

9.1.1.3 **数据分类**

企业管理员可在本页面进行新增分类、编辑分类、删除分类、添加分类子集、添加数据分类识别规则等操作以满足企对数据分类体系的动态变化和不断优化的需求。

- 新增分类集

在分类分级页面，单击「新增分类集」，可直接添加分类集，或从方案库中选择分类集。产品还支持创建多层级的分类集，以满足不同的数据分类分级需求

<img src="../images/image-225.png" style="width:5.75in;height:3.53125in" />

- 数据分类

1.  点击「新增分类」，进入新增分类页面。企业管理员可选择自行添加或直接从方案库中进行添加。

<img src="../images/image-226.png" style="width:5.75in;height:4.89583in" />

2.  在策略配置中可以选择策略表达或智能学习模式，这两种模式分别对应不同数据识别策略。

- **策略表达**

> **1）进行扫描位置**的选择，可添加多个扫描位置（包括：文件名、首行文字、页眉页脚、文件类型、文件内容）；
>
> **2）填写匹配操作符**，包括至少命中N次、每项都至少命中N次、至少能够命中M项、至少一项命中N次等多种操作符。

**语句解析**

所有项共命中至少N次：外发内容所有项至少命中关键词条N次；

每项都命中至少N次：即该敏感词条包含的全部敏感词每项都命中至少N次；

命中至少M项：即该敏感词条在不重复的情况下，命中至少M项；

至少其中一项命中N次及以上：即该敏感词条中，至少有一个关键词可以命中N次，才为命中。

**举例：**

词条A内的敏感词为：（a,b,c,）

所有项共命中至少3次：命中情况如：1)abc ;2)aaa;3)aab;

每项都命中至少3次：命中情况如：aaabbbccc；

命中至少3项：命中情况如：abc；

至少其中一项命中3次及以上：命中情况如：aaabc；

**3）填写右变量：**可在右变量中选择数据识别名称或数据识别的详情信息，并支持通过高级搜索的方式，快速新增识别规则/关键词。

**4）策略配置逻辑关系：**产品默认使用结构公式，提供了快速形成逻辑组合的能力，例如：A
and B / A or
B；可以通过这种方式快速组合多个条件以形成所需的逻辑关系。同时，产品还支持添加任意条件组，以创建更复杂的逻辑结构，例如：A
and (B or (C and
D))。这样的设计使得逻辑配置更加灵活和自由，能够满足不同场景下的复杂逻辑需求。

另外，产品也提供了切换为条件公式的功能，可以手动输入 "&&"（AND）或
"\|\|"（OR）来进行逻辑配置。这种方式适用于需要自定义逻辑操作符的情况，便于企业管理员能够更加灵活地进行逻辑设置。

完成上述步骤后，即成功配置数据分类。配置完成后，可以直接在风险策略和资产发现中引用这些数据分类。

- **智能学习**

> 点击「智能学习」，选择文件特征，并对文件相似度（0～100%）进行调节，点击完成即可创建成功。
>
> 请注意：智能学习属于异步识别策略，无法配合风险防控策略中的阻断、警告弹窗等处置动作。

- **文件指纹**

> 点击「文件指纹」，选择文件指纹，并配置宽松/严格的相似度，点击完成即可创建成功。
>
> 请注意：文件指纹属于异步识别策略，无法配合风险防控策略中的阻断、警告弹窗等处置动作。

3.  查看分类：
    在数据分类的列表页中，点击详情即可查看分类的信息，包括此分类相关的策略配置、血缘信息、资产分布等。

<img src="../images/image-227.png" style="width:5.75in;height:3.26042in" />

点击右上角「编辑」，可在页面编辑数据分类信息。

<img src="../images/image-228.png" style="width:5.75in;height:5.26042in" />

9.1.1.4 **数据分级**

1.  点击数据分类分级页面右上角「分级视角」，即可切换至分级视角页面。

<img src="../images/image-229.png" style="width:5.75in;height:1.19792in" />

<img src="../images/image-230.png" style="width:5.75in;height:2.26042in" />

2.  点击「配置分级」，可以进行分级信息的修改、删除和增加操作。产品默认内置了4个分级，可以根据需要对其进行相应的处理。需要注意的是，只有未被任何分类引用的分级可以被删除，并且删除操作必须按照从高至低的顺序进行。

<img src="../images/image-231.png" style="width:5.75in;height:2.10417in" />

<img src="../images/image-232.png" style="width:5.75in;height:2.4375in" />

9.1.2 **数据对象**

长亭终端统一管控与安全检测响应平台 DDR
提供配置数据对象功能。企业管理员可以在产品后台页面进行数据对象配置。一旦配置完成，便可以在分类配置页面进行引用。同时，若员工命中包含数据来源的策略信息将被上报至相应的页面，方便企业管理员查看和继续操作。

点击左侧目录栏「数据安全」-「数据定义」界面，选择「数据对象」，页面布局如图所示。

1.  点击左侧目录栏「数据安全」-「数据定义」界面，选择「数据对象」，点击页面右上角「+
    新建对象」，配置对象名称、选择来源类型（包括终端应用、Web应用、代码仓库）。而后根据页面提示完成信息配置。

<img src="../images/image-233.png" style="width:5.75in;height:3.4375in" />

2.  完成配置后，页面将会生成一条新记录。

<img src="../images/image-234.png" style="width:5.75in;height:1.29167in" />

3.  企业管理员可至「数据安全」-「数据定义」-「分类分级」-「新增分类」处进行引用。

9.1.3 **数据识别**

数据识别包括敏感词库、格式识别、智能识别、数据聚类。识别条件将作为数据识别的最小原子化能力，企业管理员可以根据此能力进一步配置数据的分类分级和管控策略，最终将配置结果下发至终端执行。便于实现对数据的精确分类和细致管控。

- 敏感词库

<img src="../images/image-235.png" style="width:5.75in;height:2.9375in" />

内容识别内置多条常用敏感词条。除此之外，企业管理员可以自行添加内容识别规则，并在本页面进行内容识别管理。若要新建一条内容识别，点击右上角「内容识别」，填写相应信息保存即可完成创建。

规则类型说明：

关键词组：企业管理员可输入关键词语，实现对敏感词的规则识别

正则匹配：支持输入正则信息，实现对敏感词的规则识别

关键词对：支持输入关键词及间隔信息。如：关键词1:中华 关键词2:国
间隔字符：6（一个中文字符占据3个字符）
，则形如“中华XX国”的敏感词将会命中规则

- 格式识别

长亭终端统一管控与安全检测响应平台 DDR
具备出色的数据识别能力，能够准确地对数据进行真实文件格式的识别。产品内置引擎支持数百种常见文件格式，并支持管理员自定义添加其他文件格式。企业管理员可以更全面地进行数据识别和管理，满足各种文件格式的需求。

<img src="../images/image-236.png" style="width:5.75in;height:3.33333in" />

点击左侧目录栏「数据安全」-「数据定义」，在界面中选择「数据识别」-「格式识别」，点击右上角「格式识别」，填写相应信息保存完成创建。

<img src="../images/image-237.png" style="width:5.75in;height:3.625in" />

- 智能识别

长亭终端统一管控与安全检测响应平台 DDR
提供了一种通过企业管理员上传文件样本的方式进行机器学习的功能。它能够计算文件特征，并根据这些特征帮助企业管理员提取样本信息。基于这些样本信息，长亭终端统一管控与安全检测响应平台
DDR
还能够进行训练，并生成新的机器学习规则。智能识别功能使得数据处理更加智能化和个性化，提高了识别和分类的准确度，满足了企业管理员对数据安全管理的需求。

<img src="../images/image-238.png" style="width:5.75in;height:3.0625in" />

- 文件指纹

长亭终端统一管控与安全检测响应平台 DDR
提供了一种通过企业管理员上传文件样本的方式进行识别文件指纹的功能。

<img src="../images/image-239.png" style="width:5.75in;height:1.59375in" />

- 数据聚类

在面对一批分类和特征未知的文件时，企业管理员可以利用长亭终端统一管控与安全检测响应平台
DDR
提供的数据聚类功能。通过数据聚类，企业管理员可以将这些文件进行有效的分组，形成相似性较高的数据簇。随后，可以利用这些聚类结果进行数据归类，并生成相应的识别规则。这种数据聚类的方法使得企业管理员可以更加高效地处理未知文件，提高数据管理的效率，确保数据被正确归类和处理。

1.  首先，上传一批样本文件，并根据预期来调节文件相似度的参数。一旦完成了设置操作，只需点击开始聚类即可启动聚类过程。这样，就可以方便地对文件进行聚类分析。

<img src="../images/image-240.png" style="width:5.75in;height:5.26042in" />

2.  在聚类分析完成后，可以在左侧栏看到聚类结果的大类信息。这个功能将根据文件的相似度将它们分成若干个大类别。同时，右侧会展示具体分类下的文件列表。企业管理员可以通过切换词云或表格按钮的方式，更好地了解文件中的关键词信息，从而更好地掌握文件内容。

<img src="../images/image-241.png" style="width:5.75in;height:2.82292in" />

3.  最后，点击右侧的「数据归类」。在基于文件聚类结果的基础上，企业管理员可以按需勾选关键词信息，并将文件分类到「新建机器学习」或「归为已有学习」的选项中。

<img src="../images/image-242.png" style="width:5.75in;height:4.80208in" />

<span id="_Toc1138390098" class="anchor"></span>9.2 **资产发现**

通过对企业数据资产进行全面扫描，发现敏感文件的存储位置。支持对待离职人员的敏感数据进行梳理，有效降低企业数据泄露风险。

<img src="../images/image-243.png" style="width:5.75in;height:2.89583in" />

9.2.1 **任务管理**

1.  资产发现任务管理页面，显示已创建任务，包含任务基本信息和任务状态。

<img src="../images/image-244.png" style="width:5.75in;height:3.42708in" />

2.  点击右上角「新增任务」，进入添加终端资产扫描任务界面。企业管理员可以直接从模板添加任务，也可以新建任务。

<img src="../images/image-245.png" style="width:5.75in;height:3.15625in" />

3.  进入空白任务界面，在页面完成相应信息填写后保存，即可完成任务创建。

> <img src="../images/image-246.png" style="width:5.75in;height:3.23958in" />
>
> <img src="../images/image-247.png" style="width:5.75in;height:4.4375in" />

<table style="width:84%;">
<colgroup>
<col style="width: 10%" />
<col style="width: 72%" />
</colgroup>
<tbody>
<tr>
<td style="text-align: left;">配置项</td>
<td style="text-align: left;">说明</td>
</tr>
<tr>
<td style="text-align: left;">任务名称</td>
<td
style="text-align: left;">输入自定义的任务名称，用于标识和引用特定的扫描任务。</td>
</tr>
<tr>
<td style="text-align: left;">任务描述</td>
<td
style="text-align: left;">提供任务的简要描述，包括其主要目标和预期的扫描结果。</td>
</tr>
<tr>
<td style="text-align: left;">生效范围</td>
<td
style="text-align: left;">确定扫描任务的下发范围：全局下发（对所有终端生效），部分下发（选择特定的终端或用户组）。</td>
</tr>
<tr>
<td style="text-align: left;">生效类型</td>
<td
style="text-align: left;">设置任务的执行频率：立即执行（立即下发任务），定时执行（指定时间下发任务），周期（设定周期性下发任务时间）。</td>
</tr>
<tr>
<td style="text-align: left;">生效时段</td>
<td style="text-align: left;">扫描任务在终端正式执行的时间段</td>
</tr>
<tr>
<td style="text-align: left;">扫描路径</td>
<td
style="text-align: left;">指定需要扫描的文件路径。可以是具体的目录或是多个目录。</td>
</tr>
<tr>
<td style="text-align: left;">分类限制</td>
<td
style="text-align: left;">设置文件类型的扫描限制：无限制（扫描所有文件类型），指定分类（只扫描特定类型的文件）。</td>
</tr>
<tr>
<td style="text-align: left;">文件大小</td>
<td
style="text-align: left;">定义扫描文件的大小限制，可设置最大和最小文件大小阈值。</td>
</tr>
<tr>
<td style="text-align: left;">高级选项</td>
<td
style="text-align: left;"><p><strong>启动方式：</strong>择扫描任务的启动方式：静默启动（无用户交互），点击启动（用户/员工需手动触发）。</p>
<p><strong>完成时限：</strong>设定任务完成的时间限制：智能（系统自动计算），自定义（手动设置时间限制）。超过此时间后，服务器将不再接收此任务的扫描结果。</p>
<p><strong>文件修改时间:</strong>
设置扫描范围为在特定日期或时间之后修改过的文件，以减少扫描的数据量和提高效率。</p>
<p><strong>加白路径:</strong>
定义一组路径，这些路径中的文件将被扫描任务忽略，不进行扫描，用于排除对系统稳定性和性能影响较大的目录。</p>
<p><strong>文件格式:</strong>设置扫描任务只关注特定类型的文件。包含格式：扫描任务将只考虑这些格式的文件；排除格式：这些格式的文件将不被扫描。</p>
<p><strong>未分类数据处理:</strong>
确定如何处理扫描过程中发现的未分类数据。不上传：未分类数据将被忽略；上传分析：未分类数据将被上传到服务器进行进一步聚类分析。</p></td>
</tr>
</tbody>
</table>

4.  对于已创建完成的任务，企业管理员可直接在资产发现列表中查看。

<!-- -->

5.  对于已完成的任务，企业管理员可以在资产发现列表页面，选择具体任务，点击「详情」查看该任务信息。

<img src="../images/image-248.png" style="width:5.75in;height:3.41667in" />

6.  点击页面右上角「列表视图」，可切换至列表展示。

<img src="../images/image-249.png" style="width:5.75in;height:3.17708in" />

7.  执行成功的任务，可以点击「复制」，再次进行任务下发；任务执行失败时，点击「重扫」，可选择重新扫描任务或仅针对异常设备进行专项扫描；执行中的任务，可以点击「取消」，终止本次扫描任务。

<!-- -->

8.  企业管理员可在列表页面，点击操作栏中的「删除」，删除所选任务。

9.2.2 **任务模板**

1.  点击创建任务，打开数据资产发现模板页面，页面布局如图。

<img src="../images/image-250.png" style="width:5.75in;height:5.59375in" />

2.  企业管理员可以查看内置模板，或者点击页面右上角「+
    新增模板」，新增资产发现模板。

<!-- -->

3.  在弹出的内容页填写相关信息保存即可完成创建。

<!-- -->

4.  鼠标移动至模板，可对模板进行“转任务”“编辑”“删除”操作。企业管理员可以根据需要编辑或者删除模板；也可以点击“转任务”直接将模板转为资产发现任务。

<span id="_Toc990565963" class="anchor"></span>9.3 **泄露管控**

配置不同的管控策略，快速识别和阻止敏感数据在内部和外部渠道的意外或恶意泄露，确保数据安全和合规性。

<img src="../images/image-251.png" style="width:5.75in;height:3.73958in" />

9.3.1 **外发管控**

9.3.1.1 **白名单和配置**

<img src="../images/image-252.png" style="width:5.75in;height:0.46875in" />

- 实体白名单

可以对实体（终端、用户、部门、员工标签、终端标签）进行加白，名单内的实体（终端、用户、部门、员工标签、终端标签）将不会执行管控策略。

- 管控白名单

加白内容将被管控策略过滤。您可在此处配置HTTP/HTTPS、SCP、打印机、SMB共享、远程桌面、进程、移动存储的管控白名单。

- 管控源

查看内置的管控渠道，对管控渠道进行分组，支持手动添加管控渠道。点击「新增应用」按钮，填写应用名称并补充该应用在各系统下的相应信息，即可完成手动添加。

- 管控配置

<!-- -->

- 辅助配置

> 此处配置分为管控类及固证类。管控类包括大文件解析配置（即当外发文件超过一定阈值后则不执行内容解析）、应用后台外传开发、浏览器深度管控开关；固证类包括截屏取证开关，录屏取证开关，企业管理员可手动配置截图张数、录屏时长等内容。

- 处置模板

> 配置和管理处置模板。在配置策略时，将使用此处创建的处置模板。企业管理员可自定义弹窗显示时长和中英文提示语，终端将根据系统语言自动展示相应提示。

9.3.1.2 **手动添加渠道**

点击「管控源」，打开管控渠道列表。点击列表右上方的「新增应用」，进入手动添加渠道页。

<img src="../images/image-253.png" style="width:5.75in;height:3.14583in" />

在打开的页面中，完成以下内容的填写，即可完成渠道的手动添加：

1.  填写应用名称。后续渠道列表将显示填写的渠道名称。

<!-- -->

2.  选择分组。新增的渠道将会被添加到所选择的分组中。

<!-- -->

3.  终端配置。为不同的操作系统配置应用进程信息。

<!-- -->

4.  信任文件路径。忽略并过滤指定路径下的上报内容。

9.3.1.3 **外发管控策略**

1.  创建策略

<img src="../images/image-254.png" style="width:5.75in;height:3.73958in" />

有两种方式可以创建新的渠道管控策略，可以根据需要选择合适的方式创建策略。

创建策略和策略模板的步骤相似，用户手册中以创建策略为例。

<img src="../images/image-255.png" style="width:5.75in;height:2.26042in" />

创建策略包含极速模式和高级模式。

极速模式：填写基本信息及快速配置内容

<img src="../images/image-256.png" style="width:5.75in;height:4.375in" />

<img src="../images/image-257.png" style="width:5.75in;height:2.875in" />

高级模式：需配置基本信息、生效范围、策略配置、处置响应共四个内容

- **基本信息**

需要填写策略的名称、策略组、策略描述、策略级别，策略级别将对应相应的风险级别；

- **生效范围**

可以选择该策略的下发范围，确定策略下发的人员及设备范围。

- **策略配置**

可选择策略的生效周期、生效时段，限制策略在某时间段内可用；

可以设置策略为在线/离线策略，确定策略的使用场景是终端在线或离线场景（支持同时选择）；

管控逻辑通过表达式的形式撰写，左变量包括传输渠道、文件大小、文件加密、文件压缩等，其中传输渠道为必选项；

设置的逻辑变量包括：大于、小于、大于等于、小于等于、等于、不等于。根据不同的左变量，可以选择不同的逻辑变量和右变量；

可根据需要添加多个检测条件。通过设置不同变量的组合，可以清晰构建需要检测防控的事件。

- **处置响应**

根据需要选择对应的处置响应模板，并可配置策略命中后的告警方式。

固证选项：支持将外发文件上传至产品后台，并对外发行为进行截屏和录屏，以便进行后续审计和追踪。

根据实际需求选择通知渠道类型（包括企微、钉钉、飞书、邮件、Slack），并配置相应的通知机器人、邮箱地址及通知模板。系统将依据告警情况生成并发送相应的告警信息。

<img src="../images/image-258.png" style="width:5.75in;height:5.375in" />

2.  管理策略

- 策略分组：通过策略分组，对所有策略进行便捷管理；

<!-- -->

- 搜索策略：通过表头上方的搜索功能和筛选功能，快速定位到感兴趣的策略；

<!-- -->

- 删除策略：将策略从列表中删除。

9.3.2 **落盘管控**

<img src="../images/image-259.png" style="width:5.75in;height:1.64583in" />

配置不同的落盘管控策略，快速识别敏感数据在内部和外部渠道的下载行为，并追踪文件跨设备流转信息。

9.3.2.1 **白名单和配置**

- 实体白名单

可以对实体（终端、用户、部门、员工标签、终端标签）进行加白，名单内的实体（终端、用户、部门、员工标签、终端标签）将不会执行管控策略。

- 管控源

查看内置的管控渠道。包含IM通讯、浏览器等应用。

9.3.2.2 **落盘管控策略**

1.  创建策略

- **基本信息**

需要填写策略的名称和策略描述。

- **生效范围**

可以选择该策略的下发范围，确定策略下发的人员及设备范围。

- **策略配置**

管控逻辑处，支持配置管控的传输渠道、数据分类。

- **处置响应**

> 处置动作处，根据需要选择对应的处置动作。

- 审计：记录命中策略的文件落盘日志信息。

<!-- -->

- 文件追踪：对于命中策略的行为，将对文件进行追踪，若追踪状态为「成功」，系统将提供文件外发的跨设备流转监控功能，确保数据流转的安全性。

<!-- -->

- 固证选项：支持对文件落盘行为进行截屏和录屏，以便进行后续审计和追踪。

<img src="../images/image-260.png" style="width:5.75in;height:4.21875in" />

2.  管理策略

- 搜索策略：通过表头上方的搜索功能和筛选功能，快速定位到感兴趣的策略；

<!-- -->

- 删除策略：从列表中将策略删除。

9.3.3 **邮件管控**

9.3.3.1 **白名单和配置**

- 实体白名单

可以对实体（终端、用户、部门、员工标签、终端标签）进行加白，名单内的实体（终端、用户、部门、员工标签、终端标签）将不会执行管控策略。

- 管控白名单

企业管理员可配置企业内部邮箱域（即配置安全域，在此处配置的邮箱域都将不受产品监控），如可配置企业的邮箱域名。

- 管控源

查看内置的邮箱管控渠道。企业管理员可根据需要对管控信息进行配置。

<img src="../images/image-261.png" style="width:5.75in;height:1.96875in" />

9.3.3.2 **邮件管控策略**

1.  创建策略

- **基本信息**

需要填写策略的名称和策略描述；

- **生效范围**

可以选择该策略的下发范围，确定策略下发的人员及设备范围。

- **策略配置**

管控逻辑处，支持配置管控的传输渠道。

- **处置响应**

> 处置动作处，根据需要选择对应的处置动作。

- 审计：记录命中策略的日志信息。

<img src="../images/image-262.png" style="width:5.75in;height:3.88542in" />

2.  管理策略

- 搜索策略：通过表头上方的搜索功能和筛选功能，快速定位到感兴趣的策略；

<!-- -->

- 删除策略：从列表中将策略删除。

9.3.4 **代码管控**

<img src="../images/image-263.png" style="width:5.75in;height:1.92708in" />

9.3.4.1 **白名单和配置**

- 实体白名单

可以对实体（终端、用户、部门、员工标签、终端标签）进行加白，名单内的实体（终端、用户、部门、员工标签、终端标签）将不会执行管控策略。

- 管控白名单

白名单内的仓库地址将不会执行管控策略，上报相关日志。

- 管控配置

<!-- -->

- 辅助配置

> 查看支持的代码托管平台

- 处置模板

> 配置和管理处置模板。在配置策略时，将使用创建的处置模板。企业管理员可自定义弹窗显示时长和中英文提示语，终端将根据系统语言自动展示相应提示。

9.3.4.2 **代码管控策略**

1.  **创建策略**

- **基本信息**

需要填写策略的名称和策略描述；

- **生效范围**

可以选择该策略的下发范围，确定策略下发的人员及设备范围。

- **策略配置**

可选择策略的生效时间，限制策略在某时间段内可用；

管控逻辑处，配置策略管控的动作，可以选择「代码上传」和「代码下载」其一。

管控逻辑处，支持配置Git仓库地址、仓库组名、仓库域名，SVN仓库地址。

处置动作处，根据需要选择对应的处置动作。

- 审计：记录命中策略的日志信息。

<!-- -->

- 阻断：对于命中策略的行为，将按照配置的处置模板进行弹窗告知，并阻断本次行为（现仅代码上传功能支持配置阻断动作）

<!-- -->

- 固证选项：支持对代码下载或上传行为进行截屏和录屏，以便进行后续审计和追踪。

<img src="../images/image-264.png" style="width:5.75in;height:4.16667in" />

2.  **管理策略**

- 搜索策略：通过表头上方的搜索功能和筛选功能，快速定位到感兴趣的策略；

<!-- -->

- 删除策略：从列表中将策略删除。

9.3.5 **剪贴板管控**

<img src="../images/image-265.png" style="width:5.75in;height:2.88542in" />

9.3.5.1 **白名单和配置**

- 实体白名单

可以对实体（终端、用户、部门、员工标签、终端标签）进行加白，名单内的实体（终端、用户、部门、员工标签、终端标签）将不会执行管控策略。

- 进程白名单

名单内的加白进程将不会执行管控策略，上报日志。

- 管控配置

<!-- -->

- 辅助配置

> **配置剪贴板管控的内容大小上限（最大值为10KB）**
>
> 剪贴板内容大小大于配置的上限时，将仅保留配置大小内容。（如：配置上限为10KB时，大小超过10KB的内容将仅保留其前10K字节的文字内容）
>
> **全量日志上报开关**
>
> 开关开启后，将开启剪贴板日志全量上报。将会上报除白名单实体和白名单进程外，所有的拷贝/粘贴日志信息。
>
> **图片检测**
>
> 开启开启后，将管控剪贴板中的截图信息。
>
> <img src="../images/image-266.png" style="width:5.75in;height:2.27083in" />

- 处置模板

> 配置和管理处置模板。在配置策略时，将使用此处创建的处置模板。

9.3.5.2 **剪贴板管控策略**

1.  创建策略

- **基本信息**

需要填写策略的名称和策略描述；

- **生效范围**

可以选择该策略的下发范围，确定策略下发的人员及设备范围；

- **策略配置**

可选择策略的生效时间，限制策略在某时间段内可用；

管控逻辑处，首先选择下发策略的系统，如果系统类型选择和下发实体的系统类型不匹配，策略将不会生效。

配置进程和内容规则，设置不同变量组合，以清晰构建需要检测防控的剪贴板事件。

处置动作处，根据需要选择对应的处置动作。

- 审计：记录命中策略的日志信息。

<!-- -->

- 阻断：对于命中策略的行为，配置并选择处置模板，根据处置模板中配置的方式进行阻断。

<img src="../images/image-267.png" style="width:5.75in;height:4.45833in" />

2.  管理策略

- 搜索策略：通过表头上方的搜索功能和筛选功能，快速定位到感兴趣的策略；

<!-- -->

- 删除策略：从列表中将策略删除。

9.3.6 **网页管控**

可配置需要监控的 URL 地址，拦截或审计通过网页表单（POST
请求）传输的敏感字段和 Body
内容，实现对网页键入内容（如文本框、富文本）的实时检测

<img src="../images/image-268.png" style="width:5.75in;height:2.65625in" />

9.3.6.1 **白名单和配置**

- 实体白名单

支持对终端、用户、部门、员工标签、终端标签进行加白。名单内实体默认不执行网页管控策略。

- 管控白名单

支持将指定网站地址加入白名单。命中白名单的网站不执行网页管控动作，仅保留访问日志。

- 管控配置

<!-- -->

- 网站分类

> 查看当前支持的网页分类，用于策略配置时快速选择目标网站范围。

- 处置模板

> 配置和管理网页管控弹窗模板。可设置提示语、展示时长和中英文内容，终端按系统语言展示。

9.3.6.2 **网页管控策略**

1.  创建策略

- 基本信息

填写策略名称和策略描述，便于后续检索与区分。

- 生效范围

选择策略生效的人员和终端范围。

- 策略配置

可设置策略生效时间，限定策略执行时段；

管控逻辑支持按网站地址、网站域名、网站分类配置；

可按业务需要选择网页上传、网页下载、网页访问等行为进行管控。

- 处置动作

<!-- -->

- 审计：记录命中策略的访问与操作日志。

<!-- -->

- 阻断：命中策略时弹窗提示并阻断本次行为。

<!-- -->

- 固证选项：支持对命中行为进行截屏或录屏，便于后续追溯。

2.  管理策略

- 搜索策略：通过名称、状态、生效范围等条件筛选策略；

<!-- -->

- 启停策略：可按需启用或停用策略；

<!-- -->

- 删除策略：不再使用的策略可从列表中删除。

<img src="../images/image-269.png" style="width:5.75in;height:3.625in" />

9.3.7 **IM管控**

<img src="../images/image-270.png" style="width:5.75in;height:2.91667in" />

9.3.7.1 **白名单和配置**

- 实体白名单

支持对终端、用户、部门、员工标签、终端标签进行加白。名单内实体不执行IM管控策略。

- 管控白名单

支持对IM应用或会话对象加白。命中白名单的IM行为不触发拦截动作，仅记录审计日志。

- 管控配置

<!-- -->

- IM应用列表

> 查看当前支持的IM应用范围，用于策略配置时选择目标应用。

- 处置模板

> 配置和管理IM管控弹窗模板，可分别维护中英文提示信息和弹窗显示时长。

9.3.7.2 **IM管控策略**

1.  创建策略

- 基本信息

填写策略名称和策略描述。

- 生效范围

选择策略下发范围，明确生效人员与终端。

- 策略配置

可设置策略生效时间；

管控逻辑支持按IM应用、会话对象、消息方向（发送/接收）配置；

可对消息发送、文件发送、文件接收等行为进行管控。

- 处置动作

<!-- -->

- 审计：记录命中策略的IM行为日志。

<!-- -->

- 阻断：命中策略时执行弹窗提示并阻断当前行为。

<!-- -->

- 固证选项：支持截屏、录屏等固证方式，用于审计取证。

2.  管理策略

- 搜索策略：支持按策略名称、状态、应用类型等条件进行筛选；

<!-- -->

- 启停策略：支持快速启用或停用；

<!-- -->

- 删除策略：支持删除失效或废弃策略。

<img src="../images/image-271.png" style="width:5.75in;height:3.08333in" />

<span id="heading_120" class="anchor"></span>9.4 **数字水印**

<img src="../images/image-272.png" style="width:5.75in;height:2.69792in" />

9.4.1 **水印模板**

点击右上方「水印模板」，查看和管理创建的所有水印模板。

鼠标悬浮在水印模板上，可选择“引用查询”“ 编辑”“
删除”等不同的操作。被引用的模板无法删除。

<img src="../images/image-273.png" style="width:5.75in;height:0.57292in" />

<img src="../images/image-274.png" style="width:5.75in;height:6.71875in" />

点击右上方「新增模板」，创建新的水印模板。支持屏幕水印、打印水印、应用水印（即在某个应用中嵌入水印）三种模板类型；屏幕水印支持明水印、暗水印、点阵水印和二维码水印四种水印类型。不同类型屏幕水印的详情如下：

<table style="width:89%;">
<colgroup>
<col style="width: 10%" />
<col style="width: 28%" />
<col style="width: 49%" />
</colgroup>
<tbody>
<tr>
<td style="text-align: left;"><strong>水印类型</strong></td>
<td style="text-align: left;"><strong>水印特点</strong></td>
<td style="text-align: left;"><strong>水印内容</strong></td>
</tr>
<tr>
<td style="text-align: left;">明水印</td>
<td style="text-align: left;">一种可见的、添加在屏幕上的标记</td>
<td style="text-align: left;"><ol type="1">
<li><p>当前系统账户</p></li>
</ol>
<ol start="2" type="1">
<li><p>当前员工姓名</p></li>
</ol>
<ol start="3" type="1">
<li><p>当前终端ID</p></li>
</ol>
<ol start="4" type="1">
<li><p>当前IP地址</p></li>
</ol>
<ol start="5" type="1">
<li><p>当前系统时间</p></li>
</ol>
<ol start="6" type="1">
<li><p>自定义内容</p></li>
</ol></td>
</tr>
<tr>
<td style="text-align: left;">暗水印</td>
<td style="text-align: left;">一种隐蔽的、嵌入屏幕内容的标记方法</td>
<td style="text-align: left;"><ol type="1">
<li><p>当前系统账户</p></li>
</ol>
<ol start="2" type="1">
<li><p>当前员工姓名</p></li>
</ol>
<ol start="3" type="1">
<li><p>当前终端ID</p></li>
</ol>
<ol start="4" type="1">
<li><p>当前IP地址</p></li>
</ol>
<ol start="5" type="1">
<li><p>当前系统时间</p></li>
</ol>
<ol start="6" type="1">
<li><p>自定义内容</p></li>
</ol></td>
</tr>
<tr>
<td style="text-align: left;">点阵水印</td>
<td style="text-align: left;">以点阵形式传达信息的水印</td>
<td style="text-align: left;">固定内容：当前终端ID</td>
</tr>
<tr>
<td style="text-align: left;">二维码水印</td>
<td style="text-align: left;">以二维码形式传达信息的水印</td>
<td style="text-align: left;">固定内容</td>
</tr>
</tbody>
</table>

除此之外，明水印支持对字体进行配置；暗水印除支持对字体、水印鲁棒性进行配置；点阵水印、二维码水印支持对水印透明度进行配置。

<img src="../images/image-275.png" style="width:5.75in;height:2.61458in" />

9.4.2 **水印策略**

通过设置水印策略可以下发相应的水印策略，通过部署开关控制水印的生效情况，同时还支持用户对水印策略进行编辑和删除。

<img src="../images/image-276.png" style="width:5.75in;height:2.07292in" />

点击页面右上角「+
创建策略」，选择水印类型或水印模板，即可进入水印下发页面。填写相应信息后保存即可生成新的水印策略。在列表页面，开启部署开关，完成水印下发。

<img src="../images/image-277.png" style="width:5.75in;height:3.19792in" />

<img src="../images/image-278.png" style="width:5.75in;height:2.72917in" />

9.4.3 **水印分析**

对于通过屏幕暗水印、点阵水印、二维码水印获取的图片可上传至水印分析模块进行分析溯源。

点击左侧文件上传，可以从本地上传图片进行水印分析。下方列表显示了水印分析历史，可以查看、重新上传和删除分析历史中的图片。

<img src="../images/image-279.png" style="width:5.75in;height:3.66667in" />

<span id="_Toc1681319602" class="anchor"></span>9.5 **资产地图**

数据资产地图为企业管理员展示办公网内的数据资产信息。同时，企业管理员可通过搜索文件的MD5值/分类名/分类等级/设备ID/员工姓名等内容，或通过上传文件，或根据文件相似度，找寻相对应的数据资产。

企业管理员可根据当前设计根据文件分级、分类、归属部门、归属定位文件具体信息，可视化发现不合规文件。

<img src="../images/image-280.png" style="width:5.75in;height:2.70833in" />

<img src="../images/image-281.png" style="width:5.75in;height:4.77083in" />

1.  通过配置筛选条件查询某一员工设备上的某一类型文件信息，同时企业管理员可点击界面右上角切换按钮，可进入数据流展示界面。

<img src="../images/image-282.png" style="width:5.75in;height:3.15625in" />

<img src="../images/image-283.png" style="width:5.75in;height:3.125in" />

2.  通过文件搜索，查询办公域内文件分布情况。

<img src="../images/image-284.png" style="width:5.75in;height:2.44792in" />

<img src="../images/image-285.png" style="width:5.75in;height:2.60417in" />

3.  使用文件追溯功能，系统将分析已上传文件中包含的追踪信息，进而追溯文件的持有者和跨设备流转的详细情况。

<img src="../images/image-286.png" style="width:5.75in;height:2.09375in" />

<img src="../images/image-287.png" style="width:5.75in;height:3.54167in" />

4.  根据查询结果，点击「详情」。可以查看文件具体的分布信息及基础信息。

- 数据分布界面将依据文件的发现方式进行分类展示，帮助企业管理员高效定位不同发现条件下文件曾出现过的终端设备信息。点击设备详情后，可查看该文件的详细追溯信息，并支持跳转至对应的行为日志记录。如涉及的部门或员工存在高风险行为，系统将进行显著标识，以便管理员重点关注。

<img src="../images/image-288.png" style="width:5.75in;height:2.76042in" />

<img src="../images/image-289.png" style="width:5.75in;height:2.95833in" />

- 基础信息界面将展示文件的基本属性、内容摘要及命中详情，便于企业管理员快速定位文件内容，并辅助判断其是否属于敏感信息。

<img src="../images/image-290.png" style="width:5.75in;height:5.57292in" />

<span id="_Toc1348723588" class="anchor"></span>9.6 **文件治理**

长亭终端统一管控与安全检测响应平台DDR为企业管理员提供专项治理能力，允许企业管理员直接在产品控制台启动治理任务，专注于特定资产风险的目标专项，并确保风险数据的处置流程形成完整的闭环管理。

<img src="../images/image-291.png" style="width:5.75in;height:3.625in" />

1.  **新增专项**

点击左侧目录栏「数据安全」-「文件治理」界面，进入界面后点击右上方「新增专项」，填写必要信息即可创建新的专项治理任务。

<img src="../images/image-292.png" style="width:5.75in;height:3.25in" />

|  |  |
|:---|:---|
| 配置项 | 说明 |
| 专项名称 | 填写此文件治理专项的名称。 |
| 处置最后期限 | 确定一个截止日期，标志着文件治理工作应完成的最后期限。请注意，即使超出此日期，治理工作依然会进行。 |
| 专项背景 | 提供一个简要说明，阐述启动此文件治理专项的背景和目的，帮助员工/用户理解这一任务的重要性。 |
| 允许用户加白名单 | 启用该功能后，员工可在图形用户界面（GUI）中对不合规文件执行加白操作。已加白的文件将在后续扫描中被自动忽略，不再进行合规性检测。 |
| 加白名单提示 | 终端用户选择加白文件时终端界面展示的提示文案。 |

2.  **添加专项扫描**

点击「添加扫描」并输入配置信息，以创建新的扫描任务。

<img src="../images/image-293.png" style="width:5.75in;height:3.26042in" />

<img src="../images/image-294.png" style="width:5.75in;height:3.19792in" />

<img src="../images/image-295.png" style="width:5.75in;height:4.47917in" />

<table style="width:89%;">
<colgroup>
<col style="width: 11%" />
<col style="width: 77%" />
</colgroup>
<tbody>
<tr>
<td style="text-align: left;">配置项</td>
<td style="text-align: left;">说明</td>
</tr>
<tr>
<td style="text-align: left;">任务名称</td>
<td
style="text-align: left;">输入自定义的任务名称，用于标识和引用特定的扫描任务。</td>
</tr>
<tr>
<td style="text-align: left;">任务描述</td>
<td
style="text-align: left;">提供任务的简要描述，包括其主要目标和预期的扫描结果。</td>
</tr>
<tr>
<td style="text-align: left;">生效范围</td>
<td
style="text-align: left;">确定扫描任务的下发范围：全局下发（对所有终端生效），部分下发（选择特定的终端或用户组）。</td>
</tr>
<tr>
<td style="text-align: left;">生效类型</td>
<td
style="text-align: left;">设置任务的执行频率：立即执行（立即下发任务），定时执行（指定时间下发任务），周期（设定周期性下发任务时间）。</td>
</tr>
<tr>
<td style="text-align: left;">生效时段</td>
<td style="text-align: left;">扫描任务在终端正式执行的时间段</td>
</tr>
<tr>
<td style="text-align: left;">扫描路径</td>
<td
style="text-align: left;">指定需要扫描的文件路径。可以是具体的目录或是多个目录。</td>
</tr>
<tr>
<td style="text-align: left;">指定分类</td>
<td style="text-align: left;">设置扫描特定类型的文件</td>
</tr>
<tr>
<td style="text-align: left;">自动推送</td>
<td
style="text-align: left;"><p>扫描任务完成后，系统将自动识别符合预设分类的文件，并依据自动推送配置中的系统通知消息和各类别的处置建议，将相关信息分别推送展示至终端页面。</p>
<p><img src="../images/image-296.png"
style="width:4.85417in;height:3.53125in" /></p></td>
</tr>
<tr>
<td style="text-align: left;">文件大小</td>
<td
style="text-align: left;">定义扫描文件的大小限制，可设置最大和最小文件大小阈值。</td>
</tr>
<tr>
<td style="text-align: left;">资源使用</td>
<td
style="text-align: left;">选择扫描任务的资源使用策略：低打扰（时间长）（低资源占用，减少对终端用户的干扰），平衡（终端Agent将基于当前终端状态智能调配扫描速率），无限制（扫描速率最高，会显著影响终端负载和电量）。</td>
</tr>
<tr>
<td style="text-align: left;">高级选项</td>
<td
style="text-align: left;"><p><strong>启动方式：</strong>择扫描任务的启动方式：静默启动（无用户交互），点击启动（用户/员工需手动触发）。</p>
<p><strong>完成时限：</strong>设定任务完成的时间限制：智能（系统自动计算），自定义（手动设置时间限制）。超过此时间后，服务器将不再接收此任务的扫描结果。</p>
<p><strong>文件修改时间:</strong>
设置扫描范围为在特定日期或时间之后修改过的文件，以减少扫描的数据量和提高效率。</p>
<p><strong>加白路径:</strong>
定义一组路径，这些路径中的文件将被扫描任务忽略，不进行扫描，用于排除对系统稳定性和性能影响较大的目录。</p>
<p><strong>文件格式:</strong>设置扫描任务只关注特定类型的文件。包含格式：扫描任务将只考虑这些格式的文件；排除格式：这些格式的文件将不被扫描。</p></td>
</tr>
</tbody>
</table>

成功添加后，企业管理员可在专项治理模块中查看具体的扫描任务详情。

<img src="../images/image-297.png" style="width:5.75in;height:0.96875in" />

3.  **查看扫描详情**

点击扫描任务旁的「详情」按钮，可查看扫描详情的执行结果。扫描结束后，可在风险总表模块中查看不合规文件。

企业管理员可以对不合规的文件进行处理，例如将其推送到终端设备以供处理或直接删除。

注意事项：

1.  加白：一旦文件被终端设备标记为加白文件，在下次扫描时将直接跳过该文件。

<!-- -->

2.  推送：长亭终端统一管控与安全检测响应平台 DDR
    将会把选中文件推送到终端用户设备上，需要员工自行在设备中对不合规文件按照终端GUI提示进行处理。

<img src="../images/image-298.png" style="width:5.75in;height:3.36458in" />

3.  删除：直接删除此条日志记录。

<img src="../images/image-299.png" style="width:5.75in;height:3.08333in" />

4.  **终端查看及处置**

终端用户可以打开图形用户界面(GUI)，点击左侧目录栏的「安全」-「资产扫描」，界面将显示需要处理的资产扫描任务和文件治理项目信息。终端用户可以按照提示选择不同方式来进行处理不合规文件的操作。

<img src="../images/image-300.jpeg"
style="width:5.75in;height:1.98958in" />

终端用户在接受到资产扫描任务后，可选择推迟扫描任务的执行时间，并填入推迟原因。

<img src="../images/image-301.jpeg"
style="width:5.75in;height:5.36458in" />

进入数据治理项目后，可查看和处置风险文件。

<img src="../images/image-302.jpeg" style="width:5.75in;height:2.75in" />

2.  员工拥有自主选择处理文件的权力。可采取以下几种方式：

- **一键删除：**
  此功能将选定文件移动至回收站。需注意，文件的彻底删除需由员工手动清空回收站完成。

<!-- -->

- **文件隔离：**
  选择此功能，文件将被转移至隔离区。隔离区仅限长亭终端统一管控与安全检测响应平台DDR客户端访问，以防止可能出现的潜在风险。

<!-- -->

- **加白名单：**
  若确认文件安全无需处置，员工可使用此功能将其加入白名单，避免后续扫描中的误报。

<img src="../images/image-303.jpeg"
style="width:5.75in;height:4.79167in" />

<span id="heading_126" class="anchor"></span>10. **安全防护**

<span id="_Toc1249910713" class="anchor"></span>10.1 **隐私保护**

隐私保护策略的主要功能是防止恶意软件读取
QQ、微信、浏览器等应用的隐私信息。通过限制进程的访问权限，仅允许被信任的白名单进程访问这些敏感目录，从而有效保障用户的隐私安全。

**操作步骤**

1.  **进入隐私保护页面.**选择需要配置隐私保护的应用程序（如
    QQ、微信、浏览器等）图标右上方的“齿轮”进入策略编辑页面。

> <img src="../images/image-304.png" style="width:5.75in;height:2.9375in" />

2.  **选择策略生效范围：**您可以选择策略全局生效，或者仅在部分设备上生效。还可以通过配置排除特定设备，不对其应用隐私保护策略。

> <img src="../images/image-305.png"
> style="width:4.97917in;height:7.9375in" />

3.  **模式选择：**
    选择“观察”模式或“阻断”模式**.建议首次配置时，开启“观察”模式运行30天，以监控应用的敏感数据读取情况。**

- **观察模式**：系统将收集所有试图读取该应用敏感数据的进程，并将这些进程上报至服务器。

<!-- -->

- **阻断模式**：系统直接阻止非白名单进程读取应用程序的隐私数据，确保数据安全。

4.  **配置可信进程：**在配置页面的“更多选项”中，您可以查看当前有哪些进程正在读取该应用的敏感数据，并了解这些进程在企业内部的分布情况。此信息可以帮助您更好地管理和审查进程的访问权限。

- 如果有必要信任些进程读取敏感数据，您可以将其从观察结果中加入右侧的“信任进程”列表，确保开启阻断模式后某些进程可正常访问敏感数据。

5.  返回隐私保护主页面，将策略开关进行开启。

<span id="heading_128" class="anchor"></span>10.2 **钓鱼防护**

钓鱼防护功能旨在防止指定应用下载特定后缀的文件，从而有效抵御可能存在的钓鱼攻击。管理员可以根据企业需求设置文件下载的限制规则，保护企业数据安全并避免员工误下载恶意文件。

<img src="../images/image-306.png" style="width:5.75in;height:2.1875in" />

**策略配置项**

<img src="../images/image-307.png"
style="width:4.55208in;height:5.125in" />

1\. **生效范围**

您可以灵活选择钓鱼防护策略的生效范围：

- **全局生效**：该策略将对企业内所有设备统一生效。

<!-- -->

- **部分设备生效**：您可以针对特定设备或设备组应用钓鱼防护策略，也可以排除某些设备不适用该策略。

2\. **受保护的软件**

管理员可以选择希望应用钓鱼防护策略的软件。选择了受保护的软件后，系统将对这些应用进行监控，防止它们下载指定的文件类型。

3\. **处置方式**

管理员可以为钓鱼防护策略配置不同的处置方式，以灵活应对不同级别的风险：

- **审计**：系统将记录并监控应用下载被禁文件类型的行为，但不会阻止下载，供管理员审查分析。

<!-- -->

- **弹窗提示**：当用户尝试下载被禁文件时，系统会弹出警告提示，提醒用户潜在风险，但不阻止下载操作。

<!-- -->

- **阻断**：当用户尝试下载被禁文件时，系统将直接阻止文件下载，确保其无法被下载到设备上。

4\. **黑名单后缀**

在策略配置中，管理员可以设置禁止下载的文件类型，通过添加文件后缀到黑名单中实现。这些黑名单后缀可以根据企业的需求进行自定义，避免用户下载具有安全风险的文件类型。

<span id="heading_129" class="anchor"></span>10.3 **浏览器管控**

浏览器管控用于识别终端上的浏览器与插件安装情况，并对指定插件执行安全策略。你可以通过这个功能完成三件事：

1\. 统一查看浏览器与插件的安装分布。

2\. 对高风险或不合规插件下发处置策略。

3\. 在日志中心追踪策略命中与处置结果。

<img src="../images/image-308.png" style="width:5.75in;height:2.87986in" />

10.3.1 **首次扫描**

如果第一次使用，或刚接入新终端，建议先做扫描再配策略。

操作路径：右上角模块块配置。

你可以开启“周期扫描”，也可以点击“执行”做一次“手动扫描”。完成后返回列表查看是否已出现目标插件。

<img src="../images/image-309.png" style="width:5.75in;height:2.76111in" />

10.3.2 **创建插件管控策略**

进入：浏览器管控 -\> 插件管理 -\> 创建策略。

- 按页面顺序配置即可，推荐这样填写：

策略名称 建议包含插件名和用途，例如“Chrome-高风险插件拦截-办公终端”。

策略描述 建议写明目标和影响范围，便于后续审计。

- 在“生效范围”里选择部门、人员、终端或标签。

建议先选小范围灰度（例如测试部门或少量终端），验证无误后再扩大。

- **生效时间**

在“策略配置”中设置：

1.  生效周期（如长期生效、按周期生效）。

<!-- -->

2.  生效时段（例如工作日 09:00-18:00）。

如果业务对时段敏感，先按时段启用，避免一次性全时段生效。

- **管控对象**

在“管控插件”中选择目标插件。建议优先按插件 ID 精确匹配，减少同名误命中。

如果你是在“浏览器及插件”列表点“管控”进入，插件会自动带入。

- **处置动作**

在“处置动作”中选择对应动作和模板。

推荐顺序是“先告警/提示，再限制”，确认影响面后再使用更严格动作。

- **保存与启用**

保存后回到“插件管理”列表，把策略状态切换为“启用”。

<img src="../images/image-310.png" style="width:5.75in;height:2.69792in" />

<img src="../images/image-311.png" style="width:5.75in;height:2.9125in" />

10.3.3 **配置白名单**

白名单入口在 插件管理 顶部卡片。

- **实体白名单**

用于对特定人员、部门或设备做例外放行。适合临时业务豁免。

- **插件白名单**

用于对指定插件统一放行。适合明确可信、长期需要的插件。

建议按最小范围配置白名单，不建议一开始大面积加白。

10.3.4 **如何从资产页快速发起管控**

进入 浏览器及插件 -\> 插件列表，定位目标插件后点击“管控”。

系统会跳到策略创建页并自动带入插件信息。你只需要补齐范围、时间、动作并保存启用。

这个方式适合日常临时处置，效率更高。

<img src="../images/image-312.png" style="width:5.75in;height:2.91458in" />

<span id="heading_134" class="anchor"></span>11. **用户管理**

产品坚持以人为中心，管控、运营、分析都应围绕人来进行，故长亭终端统一管控与安全检测响应平台具备齐全的员工管理功能，包括与企业现有的组织架构进行对接、根据企业现有部门及部门特性设置安全策略、以人的维度进行安全分析。

长亭终端统一管控与安全检测响应平台支持选择一个第三方系统进行用户同步，且第三方同步的信息不支持管理员进行编辑。也支持直接通过手动方式，在系统中创建用户，在系统内创建的用户有“手动”的标记，支持在系统内编辑和修改信息。

<span id="_Toc880430843" class="anchor"></span>11.1 **员工列表**

点击侧边栏「用户管理」，进入员工用户管理页面。页面左侧为企业组织架构信息，右侧为员工列表。员工管理页面可以显示员工所持设备，为员工赋予横向部门标签。员工管理页面包含系统创建和第三方系统导入的用户。

<img src="../images/image-313.png" style="width:5.75in;height:3.30208in" />

<span id="_Toc1238850471" class="anchor"></span>11.2 **员工详情**

选定员工并点击「详情」，即可访问其基本属性、文件下载记录、外发记录以及用户事件洞察中配置的事件信息。

<img src="../images/image-314.png" style="width:5.75in;height:3.65625in" />

<span id="heading_137" class="anchor"></span>11.3 **手工创建员工**

点击列表页右上方的「新增员工」，在侧边栏选择批量导入多个用户/新增单个添加用户。

- **批量导入**

切记批量导入需先导出模版，并要根据模板填写相关信息。在导入过程中，管理员需决定是否覆盖系统中已存在的用户信息。选择覆盖将替换系统中相同用户的数据。

<img src="../images/image-315.png" style="width:5.75in;height:3.92708in" />

- **单个添加**

当无需批量添加用户时，可以选择单个添加用户进行快速添加。输入相应的用户信息后完成创建。

<img src="../images/image-316.png"
style="width:5.41667in;height:7.20833in" />

<span id="_Toc1708236729" class="anchor"></span>11.4 **员工标签**

企业管理员可在此处新增、编辑、删除员工标签，并用于用户管理界面。

<img src="../images/image-317.png" style="width:5.75in;height:1.38542in" />

<img src="../images/image-318.png" style="width:5.75in;height:4.82292in" />

<span id="heading_139" class="anchor"></span>11.5
**自动同步企业组织架构**

如需自动同步您的企业组织架构，请按照以下步骤操作：

1.  在「用户管理」页面，找到并点击「账号体系」，进入账号体系页面，如下所示：

<img src="../images/image-319.png" style="width:5.75in;height:2.21875in" />

2.  在账号体系页面，选择「架构同步」选项，然后点击「添加配置」，以选择您企业的组织架构同步数据源，如下图所示：

<img src="../images/image-320.png" style="width:5.75in;height:2.98958in" />

<img src="../images/image-321.png" style="width:5.75in;height:2.77083in" />

<table style="width:88%;">
<colgroup>
<col style="width: 88%" />
</colgroup>
<tbody>
<tr>
<td
style="text-align: left;"><p><strong>如果您选择飞书组织架构同步，请参考第10.6.1章节中的“飞书同步组织架构”方法。</strong></p>
<p><strong>如果您选择钉钉组织架构同步，请参考第10.6.2章节中的“钉钉同步组织架构”方法。</strong></p>
<p><strong>如果您选择企业微信组织架构同步，请参考第10.6.3章节中的“钉钉同步组织架构”方法。</strong></p>
<p><strong>如果您选择LDAP或AD组织架构同步，请参考第10.6.4章节中的“LDAP/AD同步组织架构”方法。</strong></p></td>
</tr>
</tbody>
</table>

<span id="_Toc2056381561" class="anchor"></span>11.6 **同步详情**

支持查看用户同步的日志，以飞书同步为例，点击对应卡片即可进入详情页面。

<img src="../images/image-322.png" style="width:5.75in;height:2.96875in" />

企业管理员可在本页面查看所有同步信息。

<img src="../images/image-323.png" style="width:5.75in;height:3.08333in" />

点击日志列表中右侧的「详情」，查看该条日志的详细信息。

<img src="../images/image-324.png" style="width:5.75in;height:4.38542in" />

11.6.1 **飞书同步组织架构**

配置飞书同步，长亭终端统一管控与安全检测响应平台DDR产品需要获取以下参数（所有的参数信息需在飞书开放后台中创建一个内部应用）

- App ID：飞书应用的 App ID

<!-- -->

- App Secret：飞书应用的 Secret

11.6.1.1 **步骤一：登录飞书开放平台，创建应用并获取相应参数信息**

1.  登录飞书后台地址： https://open.feishu.cn/app
    ，使用具有飞书企业管理员权限的账户进行登录。

<!-- -->

2.  进入飞书开放平台「首页」，点击「企业自建应用」。

<img src="../images/image-325.png" style="width:5.75in;height:2.96875in" />

3.  点击创建应用，填写相应的信息。（选择企业自建应用）

<img src="../images/image-326.png" style="width:5.75in;height:6.77083in" />

4.  进入应用列表页面，选择对应的企业自建应用。点击进入，并在右侧边栏选择「凭证与基础信息」，即可查询App
    ID 和 App Secret；

<img src="../images/image-327.png" style="width:5.75in;height:2.25in" />

5.  点击「权限管理-数据权限」-通讯录权限，配置权限范围：全部成员。

<img src="../images/image-328.png" style="width:5.75in;height:2.83333in" />

6.  点击「权限管理- API权限」，为 长亭终端统一管控与安全检测响应平台DDR
    开通“获取部门基础信息，获取通讯录部门组织架构信息，获取用户基本信息，获取用户组织架构信息，获取用户邮箱信息，获取用户受雇信息，获取用户User
    ID，获取用户性别，更新通讯录，获取通讯录基本信息，更新用户
    ID，获取用户手机号，获取部门组织架构信息、查询用户席位信息、获取成员所在部门路径、查看成员的虚线上级
    ID、查看成员工号、查询用户职级、查询用户所属的工作序列、获取角色权限”权限

获取职务列表

<img src="../images/image-329.png" style="width:5.75in;height:2.75in" />

7.  选择侧边栏「版本管理与发布」，点击右上角「创建版本」，输入内容信息，点击「保存」。在保存完成后的返回页面，点击右上角「申请线上发布」，联系飞书管理员（若登录账户为管理员权限可使用当前账号审核通过）通过权限及应用发布审核。

<img src="../images/image-330.png" style="width:5.75in;height:4.80208in" />

8.  至此，应用创建成功！

11.6.1.2 **步骤二：登录产品后台，进行同步配置**

1.  选择 「账号体系-架构同步 - 飞书同步」 或
    「账号体系-架构同步-添加配置-飞书」

<!-- -->

2.  填写配置信息

- App ID ： 输入步骤一获取的应用参数App ID

<!-- -->

- APP Secret ：输入步骤一获取的应用参数Secret

<img src="../images/image-331.png" style="width:5.75in;height:2.28125in" />

- 点击下一步之后将进行连通性检测

<!-- -->

- 填写导入部门，选择导入字段等信息

<img src="../images/image-332.png" style="width:5.75in;height:3.55208in" />

- 当上游数据发生变更时，提供两种同步方式可供选择，包括手动导入和自动导入

> 注：不支持反向同步(在DDR侧变更再同步至上游数据源)

- 选择手工导入，将由人工手动触发全量同步；选择自动导入，并配置导入时间，系统将在指定时间自动导入

<img src="../images/image-333.png" style="width:5.75in;height:3.03125in" />

11.6.2 **钉钉同步组织架构**

11.6.2.1 **步骤一：登录钉钉开放平台，创建应用并获取相应参数信息**

1.  登录钉钉后台地址：https://open.dingtalk.com/，使用具有钉钉企业管理员权限的账户进行登录。

<!-- -->

2.  进入「开发者后台」，点击「应用开发」。

<!-- -->

3.  点击创建应用，填写相应的信息。

<img src="../images/image-334.png" style="width:5.75in;height:2.84375in" />

<img src="../images/image-335.png" style="width:5.75in;height:6.34375in" />

4.  进入应用详情页面，在右侧边栏选择「基础信息」-「凭证与基础信息」，即可查询AppKey和AppSecret；

<img src="../images/image-336.jpeg"
style="width:5.75in;height:2.32292in" />

旧版

<img src="../images/image-337.png" style="width:5.75in;height:1.73958in" />

新版

5.  点击权限管理 。

- 配置权限范围：全部成员

<!-- -->

- 为 长亭终端统一管控与安全检测响应平台DDR
  开通“个人权限-通讯录个人信息读权限、通讯录管理-通讯录部门信息读权限、通讯录管理-成员信息读权限、通讯录管理-通讯录部门成员读权限、通讯录管理-企业员工手机号信息、通讯录管理-邮箱等个人信息”权限。

<!-- -->

- 若后续需要使用钉钉审批，需开通“企业调用接口执行审批操作的权限、审批流数据管理权限、工作流实例写权限、工作流模板写权限、工作流模板读权限、工作流实例读权限”

<img src="../images/image-338.jpeg" style="width:5.75in;height:3.3125in" />

旧版

<img src="../images/image-339.png" style="width:5.75in;height:2.96875in" />

新版

- 点击右上角「批量申请」

6.  选择侧边栏「部署与发布-版本管理与发布」，点击「确认发布」。

<img src="../images/image-340.jpeg"
style="width:5.75in;height:3.29167in" />

7.  配置使用范围：全部员工

> <img src="../images/image-341.jpeg"
> style="width:5.75in;height:3.22917in" />

8.  至此，应用创建成功！

11.6.2.2 **步骤二：登录产品后台，进行同步配置**

1.  选择 「账号体系-架构同步 - 钉钉同步」 或
    「架构同步-添加配置-钉钉」填写配置信息

- App Key：输入步骤一获取的应用参数AppKey/Client ID

<!-- -->

- APP Secret ：输入步骤一获取的应用参数AppSecret/Client Secret

<img src="../images/image-342.png" style="width:5.75in;height:2.23958in" />

- 点击下一步后进行连通性检测

<!-- -->

- 填写导入部门，选择导入字段等信息

<img src="../images/image-343.png" style="width:5.75in;height:3.63542in" />

- 当上游数据发生变更时，提供两种同步方式可供选择，包括手动导入和自动导入

> 注：不支持反向同步(在DDR侧变更再同步至上游数据源)

- 选择手工导入，将由人工手动触发全量同步；选择自动导入，并配置导入时间，系统将在指定时间自动导入

<img src="../images/image-333.png" style="width:5.75in;height:3.03125in" />

11.6.3 **企业微信同步组织架构**

11.6.3.1 **步骤一：登录企业微信开放平台，创建应用并获取相应参数信息**

1.  登录企业微信后台地址：https://work.weixin.qq.com/，使用具有企业微信企业管理员权限的账户进行登录。

<!-- -->

2.  进入「我的企业」，点击「企业信息」，获取企业ID，此为CORP ID信息。

<img src="../images/image-344.png" style="width:5.75in;height:4.5in" />

3.  进入「安全与管理」，点击「管理工具」-「数据同步」，记录Secret，此为CORP
    Secret信息。

<img src="../images/image-345.png" style="width:5.75in;height:1.875in" />

<img src="../images/image-346.png" style="width:5.75in;height:2.625in" />

11.6.3.2 **步骤二：登录产品后台，进行同步配置**

1.  选择 「账号体系-架构同步 - 钉钉同步」 或
    「架构同步-添加配置-钉钉」填写配置信息

- App Key：输入步骤一获取的应用参数Corp ID

<!-- -->

- CORP Secret ：输入步骤一获取的应用参数Corp Secret

<img src="../images/image-347.png" style="width:5.75in;height:2.51042in" />

- 点击下一步后进行连通性检测

<!-- -->

- 填写导入部门，选择导入字段等信息

<img src="../images/image-348.png" style="width:5.75in;height:3.53125in" />

- 当上游数据发生变更时，提供两种同步方式可供选择，包括手动导入和自动导入

> 注：不支持反向同步(在DDR侧变更再同步至上游数据源)

- 选择手工导入，将由人工手动触发全量同步；选择自动导入，并配置导入时间，系统将在指定时间自动导入

<img src="../images/image-333.png" style="width:5.75in;height:3.03125in" />

11.6.4 **LDAP/AD同步组织架构**

产品为企业管理员提供了LDAP和AD同步用户及组织架构的功能，企业管理员可通过选择LDAP和AD同步，填写相应的信息后快速同步LDAP和AD上的企业员工信息及组织架构信息。

1.  点击账号体系，选择LDAP或AD域同步。LDAP和AD域均支持多组织分别挂载，故可以设置多个LDAP/AD的同步设置。（因LDAP和AD的同步类似，此处将以LDAP举例，不再单独举例AD）。

<!-- -->

2.  选择LDAP并填写相应的同步内容。

- 填写完连接信息后，可进行连接测试；

<!-- -->

- 企业管理员可根据需要进行用户、组织架构的过滤；

<!-- -->

- 用户OU及根域名节点切记勿填错，其他用户字段可根据实际情况调整。

3.  创建完同步设置后，每次同步后，将显示同步历史，管理员可即时查看每次的同步情况。包括新增员工、更新员工、注销员工及是否有同步失败的情况，同步的组织部门、更新的组织部门、注销的组织部门是否有同步失败的情况。

11.6.4.1 **步骤一：获取LDAP配置信息**

1.  获取LDAP配置信息

咨询 LDAP 的管理员，获取 LDAP
服务器地址、端口号、加密方式、管理员账号及密码，并登录 LDAP 服务器平台。

<img src="../images/image-349.png" style="width:5.75in;height:2.07292in" />

2.  查看BaseDN (根域名/根目录）节点。

咨询 LDAP
的管理员，获取BaseDN信息。BaseDN信息代表长亭终端统一管控与安全检测响应平台DDR系统将从输入的BaseDN目录开始搜索，如：dc=test,dc=com

<img src="../images/image-350.png"
style="width:5.76528in;height:1.75069in" />

3.  点击部门，记录部门标识的属性名称。

说明：

部门标识：objectClass=organizationalUnit；类型为所有organizationalUnit的对象。在长亭终端统一管控与安全检测响应平台DDR用户过滤中将查询结果中objectClass
= organizationalUnit 的对象作为LDAP 中的部门信息。

4.  点击具体员工，记录用户标识的属性名称。

<img src="../images/image-351.png" style="width:5.75in;height:3.39583in" />

说明：

用户标识：objectClass=top；类型为所有 top
的对象。在长亭终端统一管控与安全检测响应平台DDR用户过滤中将查询结果中
objectClass = top 的对象作为LDAP 中的用户信息。

11.6.4.2 **步骤二：登录产品后台，进行同步配置**

1.  选择 「账号体系-架构同步-
    LDAP同步/AD同步」或「点击添加配置，选择LDAP/AD」

<!-- -->

2.  填写配置信息

- 数据源名称：本次同步配置的名称

<!-- -->

- 链接设置：输入在步骤一中获取
  LDAP服务器地址、端口号、加密方式、管理员账号及密码、BaseDN(根域名/根目录节点)信息。

<img src="../images/image-352.png" style="width:5.75in;height:4.58333in" />

- 用户OU：LDAP信息导入至产品后台的组织架构位置（推荐选择根部门）。

<!-- -->

- 用户过滤：输入在步骤一中获取 LDAP中的部门及用户属性标识。

<img src="../images/image-353.png" style="width:5.75in;height:2.08333in" />

组织单元过滤

<img src="../images/image-354.png" style="width:5.75in;height:2.57292in" />

用户过滤

- 字段匹配：默认即可。

<!-- -->

- 选择手工导入，后续将在同步详情中，由人工手动触发全量同步；选择自动导入，并配置导入时间，系统将在指定时间自动导入

<img src="../images/image-333.png" style="width:5.75in;height:3.03125in" />

<span id="_Toc3080909" class="anchor"></span>11.7 **身份认证**

管理员可在本界面配置客户端员工通过扫码登录的认证信息。

<table style="width:88%;">
<colgroup>
<col style="width: 88%" />
</colgroup>
<tbody>
<tr>
<td
style="text-align: left;"><p>身份认证基础信息：回调域名信息，格式如：https://[host地址]:8442/access-login，配置前请确保服务器8442端口开放，或根据实际配置调整</p>
<p>身份认证GUI：启用后，可在GUI进行扫码登录</p>
<p>提醒用户登录：企业管理员可在此处设定弹出提醒的时间间隔。系统将自动检测未绑定身份的员工，并在必要时自动显示扫码登录界面</p>
<p>安全配置：员工扫码登录后的有效期</p></td>
</tr>
</tbody>
</table>

<img src="../images/image-355.png" style="width:5.75in;height:3.375in" />

<img src="../images/image-356.png" style="width:5.75in;height:3.64583in" />

<span id="heading_154" class="anchor"></span>11.8 **身份绑定**

此页面可配置客户端用户的身份与设备的绑定方式。

身份绑定方式有以下三种

<table style="width:89%;">
<colgroup>
<col style="width: 17%" />
<col style="width: 71%" />
</colgroup>
<tbody>
<tr>
<td style="text-align: left;"><strong>绑定方式</strong></td>
<td style="text-align: left;"><strong>描述</strong></td>
</tr>
<tr>
<td style="text-align: left;">手动绑定</td>
<td style="text-align: left;">企业管理员可在管理后台上传公司的 IT
资产表，通过 IP 或者 MAC绑定用户身份</td>
</tr>
<tr>
<td style="text-align: left;">登录时绑定（推荐）</td>
<td
style="text-align: left;">使用此项绑定方式需要在「用户管理」-「账号体系」-「身份认证」页面开启“身份认证
GUI”开关，管理员开启登录绑定功能后，用户在客户端登录后会将身份绑定至设备。</td>
</tr>
<tr>
<td style="text-align: left;">自动获取身份</td>
<td
style="text-align: left;"><p>若服务端与企业的「AD/LDAP/钉钉/飞书/企业微信」组织架构已完成同步，客户端会自动获取上述服务中的登录身份（最多只支持一种）。</p>
<p>注：如果同时开启了“登录时绑定”和“自动获取身份”功能，以“登录时绑定”的身份为准。</p></td>
</tr>
</tbody>
</table>

<img src="../images/image-357.png" style="width:5.75in;height:5.8125in" />

1.  登录时的绑定开关：

- **允许**：用户在客户端成功登录后，其身份认证信息将自动与当前设备绑定。这意味着设备将识别并关联该用户的身份。

<!-- -->

- **不允许**：即使用户在客户端登录，设备也不会绑定用户的身份认证信息。这样，设备不会与任何用户身份关联。

2.  切换绑定身份的提示机制：

- 当设备已与某个用户身份绑定，且另一用户尝试登录时，此功能决定是否提示用户更换绑定的身份。

<!-- -->

- 启用提示：如果新用户登录，系统将提醒当前用户，他可以选择是否更换设备上的绑定身份。

<!-- -->

- 禁用提示：新用户登录时，系统将不提供任何提示，并自动将设备上的绑定身份更换为新用户。

以下是身份绑定的流程图

<img src="../images/image-358.png" style="width:5.75in;height:3.0625in" />

<span id="_Toc2051556356" class="anchor"></span>12. **系统管理**

管理系统各项配置，显示系统运行监控信息。

<img src="../images/image-359.png" style="width:5.75in;height:3.75in" />

<span id="heading_156" class="anchor"></span>12.1 **系统主题**

个性化配置系统后台主题和客户端主题效果。

12.1.1 **系统后台**

配置系统后台的名称、系统图标和登录页图片。

<img src="../images/image-360.png" style="width:5.75in;height:3.30208in" />

12.1.2 **客户端**

配置客户端GUI的首页文案和首页图片。首页图片建议以1:1比例上传透明底的PNG格式图片。

<img src="../images/image-361.png" style="width:5.75in;height:1.95833in" />

<span id="_Toc685678949" class="anchor"></span>12.2 **安全配置**

为保障系统安全，企业管理员可配置多项安全策略，包括密码复杂性要求、重试次数限制、无操作自动登出时间、全局网页水印（启用后，控制台界面将显示当前登录账号的明水印）、远程文件提取审批开关（启用后，远程提取文件操作需经审批通过后方可执行）、MFA验证选项、钉钉扫码登录设置。

|                                                                |
|:---------------------------------------------------------------|
| 注：控制台水印配置、登录MFA认证功能 仅支持超级管理员帐号配置。 |

<img src="../images/image-362.png" style="width:5.75in;height:3.26042in" />

<img src="../images/image-363.png" style="width:5.75in;height:3.375in" />

**如何开启钉钉 SSO 登录**

1.  使用超级管理员帐号长亭终端统一管控与安全检测响应平台DDR后台，点击「系统管理」-\>「安全配置」

<!-- -->

2.  启用钉钉扫码登录功能

|                                                                |
|:---------------------------------------------------------------|
| 如何配置App Key 及 App Secret 请查看本文中「组织架构同步」部分 |

3.  访问长亭终端统一管控与安全检测响应平台 DDR
    控制台首页，选择钉钉扫码登录

|  |  |
|:--:|:--:|
| <img src="../images/image-364.png"
style="width:2.69792in;height:1.53125in" /> | <img src="../images/image-365.png"
style="width:2.70833in;height:1.53125in" /> |

4.  扫码完成后，若为首次登录，需重新输入账户名和密码进行钉钉账号绑定。

<img src="../images/image-366.png" style="width:5.75in;height:2.9375in" />

5.  绑定完成后，即可进入页面。

<!-- -->

6.  点击界面右上角，进入个人中心，也可查看第三方登录信息。

<img src="../images/image-367.png" style="width:5.75in;height:1.0625in" />

<span id="_Toc808846041" class="anchor"></span>12.3 **系统账户**

系统管理员用于管理所有账号，可以通过列表查看所有账号信息，包括用户名、角色、创建时间、最近登录时间。

12.3.1 **账户管理**

- **添加账户**

<!-- -->

- 账户新增指南：如需新增账户，请在页面右上角点击「+
  新增账户」并填写必要信息。可从现有角色和系统内置角色中进行选择。

<!-- -->

- 系统审批与通知设置：若需使用系统审批功能，并希望在待审批任务时收到GUI界面提示，请为账户指定通知终端。这样，待处理审批任务到达时，系统将在指定终端进行弹窗提醒。

<img src="../images/image-368.png"
style="width:4.10417in;height:5.72917in" />

- **修改、禁用、删除账号**

如需对账号进行修改或删除，可在相应账号操作列单击进行调整。也可点击开关按钮，对账户启用/禁用状态进行设置。

<img src="../images/image-369.png" style="width:5.75in;height:1.69792in" />

1\. **管理员**

系统的管理员包括超级管理员和管理员，超级管理员为系统最高权限的管理者，具备所有权限，但超级管理员账户唯一。管理员也具备系统的所有管理权限，可以有多个管理员的账户，但其不可创建同级别的账户，仅可通过超级管理员创建。

2\. **审计员**

审计员为系统内审计风险事件的角色，可以有多个。

3\. **操作员**

操作员为系统的前置运营人员，可以同步用户、为不同的员工下发策略、统计企业内的数据资产、下发任务、建立数据的分类分级等。

4\. **自创建角色**

在角色管理中可以创建角色，自由分配其权限。

12.3.2 **角色管理**

12.3.2.1 **角色列表**

角色列表显示已创建的角色，以及角色分别被赋予的账户数。

<img src="../images/image-370.png" style="width:5.75in;height:2.78125in" />

12.3.2.2 **新增与编辑角色**

新建角色需填写角色的基本信息，包括角色名称和角色备注。

需要为角色分配功能权限，包括对于不同菜单的读、写权限。

<img src="../images/image-371.png" style="width:5.75in;height:4.02083in" />

<span id="_Toc703925577" class="anchor"></span>12.4 **存储配置**

配置系统日志留存时间以及固证文件的存储位置，支持云上存储和私有化存储

12.4.1 **日志配置**

配置系统日志的留存时间。

<img src="../images/image-372.png" style="width:5.75in;height:2.11458in" />

12.4.2 **固证配置**

配置固证文件的存储位置，支持私有化存储和云上存储。

<img src="../images/image-373.png" style="width:5.75in;height:3.26042in" />

选择后台存储，固证文件（录屏、截图、文件）将存储在本地。

选择其它存储方式，固证文件（录屏、截图、文件）将存储在您配置的云上。企业管理员可根据页面提供的数据存储配置指南，填写对应的配置信息完成配置。

<span id="heading_172" class="anchor"></span>12.5 **AI配置**

<img src="../images/image-374.png" style="width:5.75in;height:1.03125in" />

12.5.1 **分析配置**

12.5.1.1 **机器学习分类**

此功能分为「性能优先」及「效果优先」两种模式。主要影响机器学习功能中的自动分类功能。推荐使用「性能优先」模式。

性能优先：分析速度快，但是识别准确率会降低

效果优先：识别准确率高，但是分析速度会降低

12.5.1.2 **AI 大模型分析**

配置并开启此功能后，将自动对外发文件进行AI归类。现支持常见的Office格式。

<img src="../images/image-375.png" style="width:5.75in;height:5.58333in" />

<table style="width:88%;">
<colgroup>
<col style="width: 88%" />
</colgroup>
<tbody>
<tr>
<td style="text-align: left;"><p>功能说明</p>
<ol type="1">
<li><p>可识别分类选择：企业管理员可以多选若干分类，外发文件经由AI分析后将从已选择的分类中匹配最合适的分类</p></li>
</ol>
<ol start="2" type="1">
<li><p>本公司文档判别：开启后，将由AI判别外发文件是否为本公司文档。企业管理员需按照提示，配置公司名称、核心产品名等信息</p></li>
</ol>
<ol start="3" type="1">
<li><p>高密内容提取：开启后，如果发现文档中包含公司机密信息，将由AI智能提取文件内容并展示在风险日志详情中。</p></li>
</ol></td>
</tr>
</tbody>
</table>

12.5.2 **️模型配置**

12.5.2.1 **文本模型**

企业管理员可自由选择本地或其他文本大模型进行文档分析，系统自动将文档内容发送至所选模型。在使用前，请检查服务器与模型间的网络连通性，并根据页面上的模型接入配置指南填写相关信息以完成配置。

<img src="../images/image-376.png" style="width:5.75in;height:3.60417in" />

12.5.2.2 **多模态模型**

企业管理员可自由选择本地或其他文本大模型进行图片分析，系统自动将图片内容发送至所选模型。在使用前，请检查服务器与模型间的网络连通性，并根据页面上的模型接入配置指南填写相关信息以完成配置。

<img src="../images/image-377.png" style="width:5.75in;height:3.55208in" />

<span id="_Toc893314662" class="anchor"></span>12.6 **终端GUI**

个性化配置终端GUI功能模块。

12.6.1 **全局启停**

企业管理员可通过此模块定制终端GUI展示，设定员工是否可查看Agent界面，并选择性展示身份认证、合规基线、以及资产扫描的特定业务模块界面。

<img src="../images/image-378.png" style="width:5.75in;height:2.90625in" />

12.6.2 **文案配置**

企业管理员可自定义macOS授权和卸载提示，以便员工在安装或卸载遇到问题时能迅速联系到负责人解决。

<img src="../images/image-379.png" style="width:5.75in;height:3.77083in" />

<span id="heading_182" class="anchor"></span>12.7 **通知渠道**

通知渠道帮助企业管理员快速获取系统内的安全告警。此处为通用设置，企业管理员可根据需要添加多个邮箱、IM机器人。

12.7.1 **邮件通知**

1.  点击「新增渠道」

<!-- -->

2.  输入名称，选择类型为「邮件通知」

<!-- -->

3.  填写信息

- 发送邮箱：发送通知的邮箱信息

<!-- -->

- 用户名：发送通知的邮箱用户名（一般情况下发送邮箱和用户名信息可以保持一致）

<!-- -->

- 密码：发送通知的邮箱密码

<!-- -->

- 发送服务器：

<!-- -->

- 如果是内部邮件服务器，请联系内部人员获取配置信息

<!-- -->

- 飞书：smtp.feishu.cn 端口：465

<!-- -->

- QQ：smtp.qq.com 端口：465

<!-- -->

- 企业微信：smtp.exmail.qq.com 端口：465
  参考文档：https://work.weixin.qq.com/help?doc_id=431&helpType=exmail

<!-- -->

- 钉钉企业邮箱：http://mailhelp.mxhichina.com/smartmail/detail.vm?knoId=5871700

<!-- -->

- 端口：填入端口

<!-- -->

- 加密：进行加密方式选择

4.  发送测试：输入一个接受邮件的地址测试配置的正确性

<!-- -->

5.  保存

<img src="../images/image-380.png" style="width:5.75in;height:6.9375in" />

12.7.2 **飞书通知**

1.  点击「新增渠道」

<!-- -->

2.  输入名称，选择类型为「飞书机器人」

<!-- -->

3.  填写信息

- 参考文档：https://open.feishu.cn/document/ukTMukTMukTM/ucTM5YjL3ETO24yNxkjN

<!-- -->

- 获取方式：确认消息推送群聊，在群聊中选择新增自定义机器人，配置完成后，复制webhook地址即可。

<!-- -->

- webhook：填入webhook信息

<!-- -->

- 加签：填入加签信息

4.  保存

<img src="../images/image-381.png" style="width:5.75in;height:5.14583in" />

12.7.3 **钉钉通知**

1.  点击「新增渠道」

<!-- -->

2.  输入名称，选择类型为「钉钉通知」

<!-- -->

3.  填写信息

- 参考文档：https://open.dingtalk.com/document/group/custom-robot-access

<!-- -->

- 获取方式：确认消息推送群聊，在群聊中选择智能群助手，下拉选择新增自定义机器人，勾选加签，配置完成后，复制webhook地址即可。

<!-- -->

- webhook：填入webhook信息

<!-- -->

- 加签：填入加签信息

4.  保存

<img src="../images/image-382.png" style="width:5.75in;height:4.41667in" />

12.7.4 **企业微信通知**

1.  点击「新增渠道」

<!-- -->

2.  输入名称，选择类型为「企业微信通知」

<!-- -->

3.  填写信息

- 参考文档：https://developer.work.weixin.qq.com/document/path/91770

<!-- -->

- 获取方式：确认消息推送群聊（内部群），点击右上角图标进入到群聊设置 —
  进入群机器人页面，添加群机器人，设置群机器人昵称并点击添加，机器人添加完成后，复制webhook地址即可。

<!-- -->

- webhook：填入webhook信息

4.  保存

<img src="../images/image-383.png" style="width:5.75in;height:4.54167in" />

12.7.5 **Slack通知**

1.  点击「新增渠道」

<!-- -->

2.  输入名称，选择类型为「Slack通知」

<!-- -->

3.  Webhook获取方式

- 进入 [Slack 应用管理页](https://api.slack.com/apps)。

<!-- -->

- 单击右上角的 **Create New App**，并选择 From scratch 方式创建。

<!-- -->

- 在配置页面填写应用名称，并选择对应的 Slack Workspace 创建一个 Slack
  APP。

<!-- -->

- 在应用管理页面左侧菜单栏中，选择 **Incoming Webhooks**
  并单击右上角的**开启**。

<!-- -->

- 滑动到子窗口底部，单击 **Add New Webhook to Workspace**。

> <img src="../images/image-384.png" style="width:5.75in;height:5.76042in" />

- 在配置页面中选择对应的应用，并单击 **allow**。

<!-- -->

- 在跳转框中复制 Webhook 地址。

4.  填写信息并保存

<img src="../images/image-385.png" style="width:5.75in;height:5.11458in" />

<span id="_Toc1622080640" class="anchor"></span>12.8 **通知模板**

管理员可在本页面对通知内容进行模版配置。通知模板将会根据告警内容的不同，进行变量更换，实现精准告警。

<img src="../images/image-386.png" style="width:5.75in;height:1.32292in" />

**新增模板**

管理员填写模板名称，选择该模板对应的模块。发送内容将会根据模块选择出现不同提示，管理员可以根据需要进行自定义。切记在自定义时，需要插入“变量”，否则告警信息将无法根据严重程度进行针对性提示。

<img src="../images/image-387.png" style="width:5.75in;height:5.1875in" />

<span id="heading_189" class="anchor"></span>12.9 **日志投递**

配置、管理日志投递任务，满足安全与合规要求。

点击页面右方的「添加任务」，打开添加任务页面。

1.  填写任务名称、任务描述等基本信息；

<!-- -->

2.  选择投递日志类型；

<!-- -->

3.  选择投递方式并填写对应的配置信息。

<img src="../images/image-388.png" style="width:5.75in;height:4.03125in" />

<span id="_Toc1235772707" class="anchor"></span>12.10 **导出任务**

管理创建的导出任务，在此页面可以下载导出数据。

管理员可进入风险行为日志/行为日志分析/资产发现结果等页面，根据需要将数据进行配置导出，支持管理员自定义导出内容。

<img src="../images/image-389.png" style="width:5.75in;height:2.9375in" />

<span id="_Toc1317536412" class="anchor"></span>12.11 **️策略管理**

长亭科技提供策略数据备份功能。企业管理员可在此处导入、导出数据。

企业管理员可以导入长亭科技提供的内置数据，也可以导入自行制作的相关模板、策略数据等。在特殊情况下，可以通过导出数据的方式进行策略数据备份。

<img src="../images/image-390.png" style="width:5.75in;height:3.375in" />

<span id="_Toc1130592267" class="anchor"></span>12.12 **模块管理**

当遇到特殊极端情况时，企业管理员可通过稳定性开关一键启停Agent，也可以针对部分模块功能进行启停。

<img src="../images/image-391.png" style="width:5.75in;height:4.13542in" />

企业管理员可点击「操作通知配置」，配置开启/停用通知信息，实时跟踪模块启停情况。

<img src="../images/image-392.png" style="width:5.75in;height:1.97917in" />

<span id="heading_193" class="anchor"></span>12.13 **运行监控**

为防止服务端运行资源过载，系统为管理员提供了系统运行监控界面，支持其自定义告警通知。当系统运行过载时，系统将第一时间通知管理员进行排查。

<img src="../images/image-393.png" style="width:5.75in;height:3.625in" />

<span id="heading_194" class="anchor"></span>12.14 **授权配置**

授权信息显示已上传的产品License
信息，包含客户名称、产品版本、授权时间、许可类型、授权点数、付费拓展模块、授权码等内容。提供二维码授权码、复制授权码、下载授权码等操作。

当 License
过期或者需要续约时，可点击重新上传以更新License。企业管理员需要将授权码文件提供给售后服务人员，而后从售后人员处获取最新的
License 文件。最后将新的License文件该页面重新上传，即可更新授权信息。

<img src="../images/image-394.png"
style="width:4.83333in;height:4.60139in" />

点击「查看授权码」可查看产品授权码，企业管理员可点击不同按钮进行复制和下载操作。

点击扫码按钮后打开二维码，可使用移动设备扫描该二维码获取授权码。

<img src="../images/image-395.png" style="width:5.75in;height:3.21875in" />

同时支持用户配置消息通知，点击「通知配置」，即可配置告警通知机器人。机器人将在产品到期前三天、前七天以及过期时进行消息推送。

<span id="heading_195" class="anchor"></span>12.15 **通知中心**

显示系统相关的通知信息，包括运行告警、授权提醒、组织同步异常三个模块的系统通知。

- 运行告警

系统监控出现异常时，进行日志记录，在本页面聚合后显示通知。

CPU、内存、磁盘超过80%负载时，将在此处显示告警信息。

<img src="../images/image-396.png" style="width:5.75in;height:1.79167in" />

- 授权提醒

产品授权到期前七天开始，将在此处显示授权到期提醒信息。

- 组织同步异常

组织同步异常时，进行日志记录，在本页面聚合后显示通知。

<img src="../images/image-397.png" style="width:5.75in;height:1.96875in" />
