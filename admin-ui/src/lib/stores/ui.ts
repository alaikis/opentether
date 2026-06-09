import { writable } from "svelte/store";
import { browser } from "$app/environment";

function createSidebarStore() {
	const stored = browser ? localStorage.getItem("sidebar_collapsed") : null;
	const { subscribe, set, update } = writable<boolean>(stored === "true");

	return {
		subscribe,
		toggle() {
			update((v) => {
				const next = !v;
				if (browser) localStorage.setItem("sidebar_collapsed", String(next));
				return next;
			});
		},
		set(val: boolean) {
			set(val);
			if (browser) localStorage.setItem("sidebar_collapsed", String(val));
		},
	};
}

export const sidebarCollapsed = createSidebarStore();
