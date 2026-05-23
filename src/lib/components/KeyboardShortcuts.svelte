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
		// Cmd+K toggles the palette and works even while typing — universal
		// escape hatch users expect everywhere.
		if ((event.metaKey || event.ctrlKey) && !event.shiftKey && !event.altKey) {
			if (event.key === 'k' || event.key === 'K') {
				event.preventDefault();
				clearGotoPrefix();
				if (shortcutsUI.paletteOpen) shortcutsUI.closePalette();
				else shortcutsUI.openPalette();
				return;
			}
		}

		if (isTypingTarget(event.target)) {
			clearGotoPrefix();
			return;
		}

		// Don't intercept browser shortcuts (Ctrl+Tab, Alt+Left, etc.) once
		// we're past the Cmd+K case.
		if (isModifierKeyPressed(event)) return;

		// Palette traps its own input; if it's open without focus on the
		// input the user pressed Tab away — still skip global shortcuts.
		if (shortcutsUI.paletteOpen) return;

		if (awaitingGoto) {
			const match = GOTO_ROUTES.find((r) => r.key === event.key);
			clearGotoPrefix();
			if (match) {
				event.preventDefault();
				goto(match.href);
				return;
			}
			// Unmatched g-prefix key falls through so `?` etc. still fire.
		}

		// `?` is a toggle — it must keep working while the help overlay is
		// open so users can close it the same way they opened it.
		if (event.key === '?') {
			event.preventDefault();
			shortcutsUI.toggleHelp();
			return;
		}

		if (shortcutsUI.helpOpen) return;

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
