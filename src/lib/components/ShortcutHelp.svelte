<script lang="ts">
	import Modal from './Modal.svelte';
	import { shortcutsUI } from '$lib/stores/shortcuts.svelte';
	import { SHORTCUT_BINDINGS } from '$lib/utils/shortcuts';
</script>

<Modal
	open={shortcutsUI.helpOpen}
	title="Skróty klawiaturowe"
	hideFooter
	onCancel={() => shortcutsUI.closeHelp()}
>
	<ul class="divide-y divide-surface-200-800">
		{#each SHORTCUT_BINDINGS as binding (binding.keys)}
			<li class="flex items-center justify-between py-2 gap-4">
				<span class="text-sm text-surface-800-200">{binding.description}</span>
				<kbd class="kbd font-mono text-xs whitespace-nowrap">{binding.keys}</kbd>
			</li>
		{/each}
	</ul>
</Modal>
