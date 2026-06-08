<script lang="ts">
	// (app)/user - 用户系统侧边栏布局
	import Sidebar from '$lib/components/Sidebar.svelte';
	import Header from '$lib/components/Header.svelte';
	import { MessageSquare, History, Settings } from 'lucide-svelte';

	const userNav = [
		{ href: '/admin/user', label: 'AI 对话', icon: MessageSquare, exact: true },
		{ href: '/admin/user/chat', label: '新对话', icon: MessageSquare },
		{ href: '/admin/user/conversations', label: '历史记录', icon: History },
	];
</script>

<div class="flex min-h-screen">
	<Sidebar items={userNav} />
	<div class="flex-1 flex flex-col ml-64">
		<Header title="用户中心" />
		<main class="flex-1 p-6">
			<slot />
		</main>
	</div>
</div>
