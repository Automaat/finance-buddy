// Keyboard shortcut helpers. Detects whether the user is currently typing
// (so global shortcuts can stay silent) and exposes the binding metadata
// used by both the layout dispatcher and the help overlay.

export interface ShortcutBinding {
	keys: string;
	description: string;
}

export interface GotoRoute {
	key: string;
	href: string;
	label: string;
}

export const GOTO_ROUTES: GotoRoute[] = [
	{ key: 'h', href: '/', label: 'Pulpit' },
	{ key: 'a', href: '/accounts', label: 'Konta' },
	{ key: 's', href: '/snapshots', label: 'Migawki' },
	{ key: 't', href: '/transactions', label: 'Transakcje' }
];

export const SHORTCUT_BINDINGS: ShortcutBinding[] = [
	{ keys: 'n', description: 'Nowa migawka' },
	{ keys: 'g h', description: 'Pulpit' },
	{ keys: 'g a', description: 'Konta' },
	{ keys: 'g s', description: 'Migawki' },
	{ keys: 'g t', description: 'Transakcje' },
	{ keys: '?', description: 'Pokaż skróty' },
	{ keys: 'Cmd/Ctrl + K', description: 'Paleta komend' }
];

export function isTypingTarget(target: EventTarget | null): boolean {
	if (!(target instanceof HTMLElement)) return false;
	if (target.isContentEditable) return true;
	const editable = target.getAttribute('contenteditable');
	if (editable === '' || editable === 'true' || editable === 'plaintext-only') return true;
	const tag = target.tagName;
	if (tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT') return true;
	return false;
}

export function isModifierKeyPressed(event: KeyboardEvent): boolean {
	return event.ctrlKey || event.metaKey || event.altKey;
}
