<svelte:head><title>仪表盘 - OpenTether</title></svelte:head>
<div class="mb-8"><h2 class="text-2xl font-bold text-slate-800">欢迎回来 👋</h2><p class="text-slate-500 mt-1">OpenTether 企业级智能体管理平台</p></div>
<div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-5">
	{#each [{href:'/admin/users',icon:'👥',color:'bg-blue-50 text-blue-600',title:'用户管理',desc:'管理系统用户、权限和角色',badge:'核心'},{href:'/admin/groups',icon:'👨‍👩‍👧‍👦',color:'bg-emerald-50 text-emerald-600',title:'用户组管理',desc:'管理用户组和数据权限',badge:'安全'},{href:'/admin/providers',icon:'🤖',color:'bg-amber-50 text-amber-600',title:'LLM 提供商',desc:'配置 OpenAI、Azure、Anthropic',badge:'AI'},{href:'/admin/datasources',icon:'🗄️',color:'bg-indigo-50 text-indigo-600',title:'数据源管理',desc:'配置数据库、API 等数据源',badge:'数据'},{href:'/admin/skills',icon:'⚡',color:'bg-pink-50 text-pink-600',title:'Skills 配置',desc:'管理 AI 技能和执行器',badge:'智能'},{href:'/admin/tasks',icon:'📅',color:'bg-cyan-50 text-cyan-600',title:'定时任务',desc:'配置和管理定时执行任务',badge:'自动化'},{href:'/admin/im',icon:'💬',color:'bg-purple-50 text-purple-600',title:'IM 配置',desc:'企业微信、飞书、钉钉集成',badge:'通讯'},{href:'/admin/logs',icon:'📋',color:'bg-slate-100 text-slate-600',title:'系统日志',desc:'审计日志和请求日志',badge:'监控'}] as card}
		<a href={card.href} class="card group">
			<div class="w-11 h-11 rounded-xl {card.color} flex items-center justify-center text-xl mb-4">{card.icon}</div>
			<div class="flex items-center gap-2 mb-2"><h3 class="font-semibold text-slate-800">{card.title}</h3><span class="text-[11px] px-2 py-0.5 rounded-full bg-slate-100 text-slate-500">{card.badge}</span></div>
			<p class="text-sm text-slate-500">{card.desc}</p>
		</a>
	{/each}
</div>
