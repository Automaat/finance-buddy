<script lang="ts">
	import { tick } from 'svelte';
	import { goto } from '$app/navigation';
	import { shortcutsUI } from '$lib/stores/shortcuts.svelte';

	interface Command {
		label: string;
		hint?: string;
		run: () => void;
	}

	const commands: Command[] = [
		{ label: 'Dashboard', hint: '/', run: () => goto('/') },
		{ label: 'Metryki', hint: '/metryki', run: () => goto('/metryki') },
		{ label: 'Symulacje', hint: '/simulations', run: () => goto('/simulations') },
		{ label: 'Emerytura (Monte Carlo)', hint: '/retirement', run: () => goto('/retirement') },
		{ label: 'Konta', hint: '/accounts', run: () => goto('/accounts') },
		{ label: 'Transakcje', hint: '/transactions', run: () => goto('/transactions') },
		{ label: 'Majątek', hint: '/assets', run: () => goto('/assets') },
		{ label: 'Zobowiązania', hint: '/debts', run: () => goto('/debts') },
		{ label: 'Cele', hint: '/goals', run: () => goto('/goals') },
		{ label: 'Snapshoty', hint: '/snapshots', run: () => goto('/snapshots') },
		{ label: 'Nowy snapshot', hint: '/snapshots/new', run: () => goto('/snapshots/new') },
		{ label: 'Wynagrodzenia', hint: '/salaries', run: () => goto('/salaries') },
		{ label: 'Ustawienia', hint: '/settings', run: () => goto('/settings') }
	];

	let query = $state('');
	let activeIndex = $state(0);
	let inputEl: HTMLInputElement | null = $state(null);

	const filtered = $derived.by(() => {
		const q = query.trim().toLowerCase();
		if (!q) return commands;
		return commands.filter(
			(c) => c.label.toLowerCase().includes(q) || c.hint?.toLowerCase().includes(q)
		);
	});

	$effect(() => {
		if (shortcutsUI.paletteOpen) {
			query = '';
			activeIndex = 0;
			tick().then(() => inputEl?.focus());
		}
	});

	$effect(() => {
		if (activeIndex >= filtered.length) activeIndex = 0;
	});

	function close() {
		shortcutsUI.closePalette();
	}

	function runCommand(cmd: Command) {
		close();
		cmd.run();
	}

	function handleKeydown(event: KeyboardEvent) {
		if (event.key === 'Escape') {
			event.preventDefault();
			close();
			return;
		}
		if (event.key === 'ArrowDown') {
			event.preventDefault();
			if (filtered.length === 0) return;
			activeIndex = (activeIndex + 1) % filtered.length;
			return;
		}
		if (event.key === 'ArrowUp') {
			event.preventDefault();
			if (filtered.length === 0) return;
			activeIndex = (activeIndex - 1 + filtered.length) % filtered.length;
			return;
		}
		if (event.key === 'Enter') {
			event.preventDefault();
			const cmd = filtered[activeIndex];
			if (cmd) runCommand(cmd);
		}
	}

	function handleBackdrop(event: MouseEvent) {
		if (event.target === event.currentTarget) close();
	}
</script>

{#if shortcutsUI.paletteOpen}
	<div
		class="fixed inset-0 z-50 flex items-start justify-center bg-surface-950/60 backdrop-blur-sm p-4 pt-[10vh]"
		role="presentation"
		onclick={handleBackdrop}
	>
		<div
			class="card preset-filled-surface-50-950 w-full max-w-xl shadow-xl flex flex-col"
			role="dialog"
			aria-modal="true"
			aria-label="Paleta komend"
		>
			<div class="p-3 border-b border-surface-200-800">
				<input
					bind:this={inputEl}
					type="text"
					class="input w-full"
					placeholder="Szukaj komendy lub strony…"
					bind:value={query}
					onkeydown={handleKeydown}
					aria-label="Szukaj komendy"
				/>
			</div>
			<ul class="max-h-[50vh] overflow-y-auto py-1" role="listbox">
				{#if filtered.length === 0}
					<li class="px-4 py-3 text-sm text-surface-700-300">Brak wyników</li>
				{:else}
					{#each filtered as cmd, i (cmd.hint ?? cmd.label)}
						<li role="option" aria-selected={i === activeIndex}>
							<button
								type="button"
								class="w-full flex items-center justify-between gap-4 px-4 py-2 text-left text-sm transition-colors
									{i === activeIndex
									? 'preset-filled-primary-500'
									: 'hover:preset-tonal-primary text-surface-800-200'}"
								onclick={() => runCommand(cmd)}
								onmouseenter={() => (activeIndex = i)}
							>
								<span>{cmd.label}</span>
								{#if cmd.hint}
									<span
										class="font-mono text-xs opacity-70 {i === activeIndex
											? ''
											: 'text-surface-700-300'}"
									>
										{cmd.hint}
									</span>
								{/if}
							</button>
						</li>
					{/each}
				{/if}
			</ul>
		</div>
	</div>
{/if}
