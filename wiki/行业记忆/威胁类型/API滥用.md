---
type: threat
title: API滥用
aliases: ["接口滥用", "API攻击", "BOLA"]
tags: ["API安全", "Web安全"]
summary: 针对业务 API 的恶意调用，含批量爬取、撞库、未授权对象访问
typical_signs: ["单一 API 高频调用", "接口返回超预期数据", "异常接口遍历（递增 ID）"]
related_products: ["雷池"]
status: verified
---

API 滥用针对业务接口，含批量数据爬取、撞库、未授权对象访问（BOLA，越权读取其他对象）。随 API 化趋势已成高频威胁，需 API 网关 + WAF 协同。
