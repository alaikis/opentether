import re

with open("admin-ui/src/routes/admin/skills/+page.svelte", "r", encoding="utf-8") as f:
    content = f.read()

old_template = """<!-- Modal -->
{#if showModal}
    <div
        class="fixed inset-0 z-50 flex items-center justify-center bg-black/40"
        on:click|self={closeModal}
        on:keydown={(e) => e.key === "Escape" && closeModal()}
    >
        <div
            class="bg-white rounded-2xl shadow-xl border border-slate-100 w-full {formType ===
            'text2sql'
                ? 'max-w-5xl'
                : 'max-w-lg'} max-h-[88vh] overflow-y-auto p-6"
        >
            <h3 class="text-lg font-bold text-slate-800 mb-4">
                {editingSkill ? "编辑 Skill" : "创建 Skill"}
            </h3>

            <div class="space-y-4">"""

new_template = """<!-- Modal -->
{#if showModal}
    <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/40" on:click|self={closeModal} on:keydown={(e) => e.key === "Escape" && closeModal()}>
        <div class="bg-white rounded-2xl shadow-xl border border-slate-100 w-full {formType === 'text2sql' ? 'max-w-4xl' : 'max-w-lg'} max-h-[88vh] overflow-y-auto p-6">
            <h3 class="text-lg font-bold text-slate-800 mb-4">
                {editingSkill ? "编辑 Skill" : "创建 Skill"}
            </h3>

            <div class="space-y-4">"""

content = content.replace(old_template, new_template)

with open("admin-ui/src/routes/admin/skills/+page.svelte", "w", encoding="utf-8") as f:
    f.write(content)
print("Template updated")
