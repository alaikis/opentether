<script lang="ts">
    import { auth } from "$lib/stores/auth";
    import { goto } from "$app/navigation";
    import Button from "$lib/components/Button.svelte";
    import Input from "$lib/components/Input.svelte";
    import { toast } from "$lib/stores/toast";

    let username = "";
    let password = "";
    let loading = false;
    let error = "";

    async function handleLogin() {
        error = "";
        if (!username || !password) {
            error = "请输入用户名和密码";
            return;
        }
        loading = true;
        try {
            await auth.login(username, password);
            toast.success("登录成功");
            goto("/admin");
        } catch (e: any) {
            error = e.message || "登录失败";
            toast.error(error);
        } finally {
            loading = false;
        }
    }

    function handleKeydown(e: KeyboardEvent) {
        if (e.key === "Enter") handleLogin();
    }
</script>

<svelte:head><title>登录 - OpenTether</title></svelte:head>

<div
    class="min-h-screen flex items-center justify-center bg-gradient-to-br from-slate-100 to-primary-50 p-4"
>
    <div class="w-full max-w-md">
        <!-- Logo -->
        <div class="text-center mb-8">
            <div
                class="w-14 h-14 rounded-2xl bg-primary-600 flex items-center justify-center text-white text-xl font-bold mx-auto mb-4 shadow-lg shadow-primary-200"
            >
                OT
            </div>
            <h1 class="text-2xl font-bold text-slate-800">OpenTether</h1>
            <p class="text-slate-500 text-sm mt-1">企业级智能体管理平台</p>
        </div>

        <!-- Card -->
        <div class="bg-white rounded-2xl shadow-xl border border-slate-100 p-8">
            <h2 class="text-lg font-semibold text-slate-800 mb-6">账号登录</h2>

            {#if error}
                <div
                    class="mb-4 p-3 rounded-lg bg-red-50 border border-red-200 text-red-700 text-sm"
                >
                    {error}
                </div>
            {/if}

            <form on:submit|preventDefault={handleLogin} class="space-y-4">
                <Input
                    label="用户名"
                    placeholder="请输入用户名"
                    bind:value={username}
                    required
                    on:keydown={handleKeydown}
                />
                <Input
                    label="密码"
                    type="password"
                    placeholder="请输入密码"
                    bind:value={password}
                    required
                    on:keydown={handleKeydown}
                />

                <Button
                    type="submit"
                    variant="primary"
                    size="lg"
                    {loading}
                    class="w-full mt-2"
                >
                    登 录
                </Button>
            </form>
        </div>

        <p class="text-center text-xs text-slate-400 mt-6">
            OpenTether Enterprise v1.0.0
        </p>
    </div>
</div>
