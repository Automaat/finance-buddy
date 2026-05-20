export type ToastKind = 'error' | 'success' | 'info';

export interface Toast {
	id: number;
	kind: ToastKind;
	message: string;
}

const DEFAULT_DURATION_MS = 4000;

let nextId = 0;
const items = $state<Toast[]>([]);
const timers = new Map<number, ReturnType<typeof setTimeout>>();

function dismiss(id: number): void {
	const index = items.findIndex((t) => t.id === id);
	if (index !== -1) items.splice(index, 1);
	const timer = timers.get(id);
	if (timer) {
		clearTimeout(timer);
		timers.delete(id);
	}
}

function push(kind: ToastKind, message: string, durationMs = DEFAULT_DURATION_MS): number {
	const id = ++nextId;
	items.push({ id, kind, message });
	if (durationMs > 0) {
		timers.set(
			id,
			setTimeout(() => dismiss(id), durationMs)
		);
	}
	return id;
}

export const toast = {
	get items(): ReadonlyArray<Toast> {
		return items;
	},
	error(message: string, durationMs?: number): number {
		return push('error', message, durationMs);
	},
	success(message: string, durationMs?: number): number {
		return push('success', message, durationMs);
	},
	info(message: string, durationMs?: number): number {
		return push('info', message, durationMs);
	},
	dismiss
};
