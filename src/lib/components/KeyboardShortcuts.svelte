<script lang="ts">
	import { goto } from '$app/navigation';
	import { shortcutsUI } from '$lib/stores/shortcuts.svelte';
	import { GOTO_ROUTES, isModifierKeyPressed, isTypingTarget } from '$lib/utils/shortcuts';
	import ShortcutHelp from './ShortcutHelp.svelte';
	import CommandPalette from './CommandPalette.svelte';

	const GOTO_TIMEOUT_MS = 1500;

	let awaitingGoto = $state(false);
	let gotoTimer: ReturnType<typeof setTimeout> | null = null;

	function clearGotoPrefix() {
		awaitingGoto = false;
		if (gotoTimer) {
			clearTimeout(gotoTimer);
			gotoTimer = null;
		}
	}

	function armGotoPrefix() {
		awaitingGoto = true;
		if (gotoTimer) clearTimeout(gotoTimer);
		gotoTimer = setTimeout(clearGotoPrefix, GOTO_TIMEOUT_MS);
	}

	function handleKeydown(event: KeyboardEvent) {
		// Command palette: works even when typing — it's the universal escape
		// hatch users expect from Cmd+K everywhere.
		if ((event.metaKey || event.ctrlKey) && !event.shiftKey && !event.altKey) {
			if (event.key === 'k' || event.key === 'K') {
				event.preventDefault();
				clearGotoPrefix();
				shortcutsUI.openPalette();
				return;
			}
		}

		if (isTypingTarget(event.target)) return;

		// Don't intercept Tab / arrows / etc. that combine with modifiers
		// (browser shortcuts) once we're past the Cmd+K case.
		if (isModifierKeyPressed(event)) return;

		if (awaitingGoto) {
			const match = GOTO_ROUTES.find((r) => r.key === event.key);
			clearGotoPrefix();
			if (match) {
				event.preventDefault();
				goto(match.href);
			}
			return;
		}

		if (event.key === '?') {
			event.preventDefault();
			shortcutsUI.toggleHelp();
			return;
		}

		if (event.key === 'n') {
			event.preventDefault();
			goto('/snapshots/new');
			return;
		}

		if (event.key === 'g') {
			event.preventDefault();
			armGotoPrefix();
		}
	}
</script>

<svelte:window onkeydown={handleKeydown} />

<ShortcutHelp />
<CommandPalette />
