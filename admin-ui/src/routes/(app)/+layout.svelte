<script lang="ts">
	// (app) 路由组 - 认证守卫 + 默认 Admin 侧边栏
	import { auth } from '$lib/stores/auth';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';
	import Sidebar from '$lib/components/Sidebar.svelte';
	import Header from '$lib/components/Header.svelte';
	import { LayoutDashboard, Users, UserCog, Cpu, Database, Zap, Calendar, MessageSquare, ScrollText } from 'lucide-svelte';

	const adminNav = [
		{ href: '/admin', label: '仪表盘', icon: LayoutDashboard, exact: true },
		{ href: '/admin/users', label: '用户管理', icon: Users },
		{ href: '/admin/groups', label: '用户组', icon: UserCog },
		{ href: '/admin/providers', label: 'LLM 提供商', icon: Cpu },
		{ href: '/admin/datasources', label: '数据源', icon: Database },
		{ href: '/admin/skills', label: 'Skills', icon: Zap },
		{ href: '/admin/tasks', label: '定时任务', icon: Calendar },
		{ href: '/admin/im', label: 'IM 配置', icon: MessageSquare },
		{ href: '/admin/logs', label: '系统日志', icon: ScrollText },
	];

	$: pageTitle = adminNav.find(i => i.exact ? $page.url.pathname === i.href : $page.url.pathname.startsWith(i.href))?.label || '管理后台';

	onMount(() => {
		if (!$auth.isAuthenticated) {
			goto('/admin/login');
		}
	});
</script>

<!-- 仅当有嵌套 layout 覆盖时不渲染此默认 sidebar -->
{#if !$page.route.id?.startsWith('/(app)/user') && !$page.route.id?.startsWith('/(app)/docs')}
	<div class="flex min-h-screen">
		<Sidebar items={adminNav} />
		<div class="flex-1 flex flex-col ml-64">
			<Header title={pageTitle} />
			<main class="flex-1 p-6">
				<slot />
			</main>
		</div>
	</div>
{:else}
	<slot />
{/if}
