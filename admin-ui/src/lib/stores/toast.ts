import { writable } from 'svelte/store';

interface Toast { id: number; message: string; type: 'success' | 'error' | 'info' }

const toasts = writable<Toast[]>([]);
let id = 0;

function add(message: string, type: Toast['type'] = 'info', duration = 4000) {
	const tid = ++id;
	toasts.update(t => [...t, { id: tid, message, type }]);
	setTimeout(() => toasts.update(t => t.filter(x => x.id !== tid)), duration);
}

export const toast = {
	success(msg: string) { add(msg, 'success'); },
	error(msg: string) { add(msg, 'error', 6000); },
	info(msg: string) { add(msg, 'info'); },
};

export { toasts };
