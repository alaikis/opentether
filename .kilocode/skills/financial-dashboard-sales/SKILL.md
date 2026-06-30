---
name: financial-dashboard-sales
description: 财务数据大盘销售分析技能。提供按员工、市场、SKU、分类等多维度的销售数据分析，包括环比变化和排名。
skill_type: text2sql
category: finance
keywords: 销售,员工,市场,SKU,分类,销售渠道,销售分析,sales,staff,market
enabled: true
---

# 财务数据大盘销售分析

本技能提供财务数据大盘销售维度的多维度分析能力。

## 核心数据范围

- **按员工**: 本月/上月各员工销售额排名（默认Top10）
- **按市场**: 本月/上月各市场（渠道）销售额排名（默认Top10）
- **按SKU**: 本月各SKU销售额排名（默认Top10）
- **按分类**: 本月各商品分类销售额汇总（按 catalog_name 合并）
- **销售渠道**: 各店铺/渠道订单数、收入、占比、环比变化

## 数据来源

- `order` 表 - 订单数据
- `order_item` / `order_sales_profit` 表 - 订单明细和利润
- `channel_shop` 相关表 - 渠道店铺

## 查询示例

```
查询本月销售Top10员工
查询各市场销售额
查询SKU销售额排名
查询商品分类销售额
查询销售渠道占比
```

## 权限说明

- 员工: 可查看本人相关数据
- 管理层: 可查看全公司数据
- 销售: 可查看详细销售数据
