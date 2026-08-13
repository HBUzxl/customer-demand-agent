---
type: threat
title: CC攻击
aliases: ["CC", "HTTP Flood", "应用层DDoS"]
tags: ["DDoS", "应用层", "Web安全"]
summary: 应用层分布式拒绝服务，通过大量 HTTP 请求耗尽服务器资源
typical_signs: ["网站突然变慢或无响应", "单一接口被大量异常调用", "服务器 CPU 飙升但带宽正常", "大量相似来源请求"]
related_products: ["雷池"]
status: verified
---

CC 攻击（Challenge Collapsar）是应用层 DDoS，攻击者通过大量看似正常的 HTTP 请求耗尽 Web 服务器资源。与网络层 DDoS 不同，CC 流量小但请求密集，需在应用层（WAF）做频率与行为识别。
