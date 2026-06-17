<script lang="ts">
    let jobs = [
        { name: "员工月度销售", table: "sales_monthly_employee_stats", status: "planned" },
        { name: "产品月度利润", table: "product_monthly_profit_stats", status: "planned" },
        { name: "国家销售分布", table: "country_monthly_sales_stats", status: "planned" },
    ];
</script>
<div class="space-y-6">
    <div><h1 class="text-2xl font-bold">预计算指标表</h1><p class="text-sm text-slate-500">高频管理指标通过物化表秒出，适合驾驶舱和固定报表。</p></div>
    <div class="card space-y-2">{#each jobs as j}<div class="flex justify-between border-b py-2"><span>{j.name}</span><span class="font-mono text-xs">{j.table}</span><span class="text-xs text-slate-500">{j.status}</span></div>{/each}</div>
</div>
