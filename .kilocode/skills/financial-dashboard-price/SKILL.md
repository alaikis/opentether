---
name: financial-dashboard-price
description: 财务数据大盘价格指数和价格预警查询技能。提供价格指数趋势、价格异常告警、SKU价格变动、供应商价格波动等价格相关分析。
skill_type: text2sql
category: finance
keywords: 价格,价格指数,价格预警,SKU,供应商,价格波动,price,price_alerts,price_sku
enabled: true
---

# 财务数据大盘价格指标

本技能提供财务数据大盘价格相关指标的查询和分析能力，关注价格变动和供应商价格波动。

## 核心数据范围

- **价格指数趋势**: 12个月价格指数变化趋势
- **价格异常告警**: 30天内价格异常变动（变动≥5%触发告警）
  - critical: 变动≥15%
  - warning: 变动≥8%
  - info: 变动≥5%
- **SKU价格变动**: 当月SKU价格调整记录
- **供应商价格波动**: 供应商价格波动率、风险等级
  - HIGH: 波动率>30%
  - MEDIUM: 波动率>15%
  - LOW: 波动率≤15%

## 数据来源

- `price_advisor_records` 表 - 价格顾问记录
- `price_advisor_subscribe` 表 - 价格订阅
- `price_adjust_history` 表 - 价格调整历史

## 查询示例

```
查询价格指数趋势
查询本月SKU价格变动
查询供应商价格波动
查询价格异常告警
```

## 权限说明

- 员工: 可查看本人相关数据
- 管理层: 可查看全公司数据
- 采购: 可查看详细采购和供应商价格数据
