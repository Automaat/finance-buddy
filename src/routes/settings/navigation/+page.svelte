<script lang="ts">
	import { Pin, PinOff, RotateCcw, ChevronUp, ChevronDown } from 'lucide-svelte';
	import { navPrefs, MAX_PINNED, DEFAULT_PINNED } from '$lib/stores/navPrefs.svelte';
	import { NAV_ROUTES } from '$lib/nav/routes';
	import { toast } from '$lib/stores/toast.svelte';

	const pinned = $derived(navPrefs.pinned);

	function isPinned(href: string): boolean {
		return pinned.includes(href);
	}

	function toggle(href: string): void {
		if (isPinned(href)) {
			if (pinned.length <= 1) {
				toast.error('Musi pozostać co najmniej jedna przypięta pozycja.');
				return;
			}
			navPrefs.set(pinned.filter((h) => h !== href));
			return;
		}
		if (pinned.length >= MAX_PINNED) {
			toast.error(`Maksymalnie ${MAX_PINNED} przypiętych pozycji. Odepnij inną najpierw.`);
			return;
		}
		navPrefs.set([...pinned, href]);
	}

	function reset(): void {
		navPrefs.reset();
		toast.success('Przywrócono domyślne przypięte pozycje');
	}

	function move(href: string, delta: -1 | 1): void {
		const idx = pinned.indexOf(href);
		if (idx === -1) return;
		const next = [...pinned];
		const swap = idx + delta;
		if (swap < 0 || swap >= next.length) return;
		[next[idx], next[swap]] = [next[swap], next[idx]];
		navPrefs.set(next);
	}

	const isDefault = $derived(
		pinned.length === DEFAULT_PINNED.length && pinned.every((h, i) => h === DEFAULT_PINNED[i])
	);
</script>

<svelte:head>
	<title>Nawigacja | Ustawienia | Finansowa Forteca</title>
</svelte:head>

<div class="mb-6 space-y-1">
	<h1 class="h2">Nawigacja mobilna</h1>
	<p class="text-surface-700-300 text-sm">
		Przypnij do {MAX_PINNED} pozycji w dolnej nawigacji. Pozostałe trafią do panelu „Więcej”.
	</p>
</div>

<div class="space-y-6">
	<section class="card preset-filled-surface-100-900 p-4 space-y-3">
		<header class="flex items-center justify-between">
			<h3 class="h4 font-semibold">Przypięte ({pinned.length}/{MAX_PINNED})</h3>
			<button
				type="button"
				class="btn btn-sm preset-tonal-surface"
				onclick={reset}
				disabled={isDefault}
			>
				<RotateCcw size={16} />
				<span>Domyślne</span>
			</button>
		</header>
		{#if pinned.length === 0}
			<p class="text-sm text-surface-700-300">Brak przypiętych pozycji.</p>
		{:else}
			<ol class="space-y-2">
				{#each pinned as href, i (href)}
					{@const item = NAV_ROUTES.find((r) => r.href === href)}
					{#if item}
						<li class="flex items-center gap-3 px-3 py-2 rounded-container bg-surface-50-950">
							<item.icon size={18} class="text-primary-500" />
							<span class="flex-1 text-sm font-medium">{item.label}</span>
							<button
								type="button"
								class="btn-icon btn-icon-sm"
								aria-label="W górę"
								disabled={i === 0}
								onclick={() => move(href, -1)}
							>
								<ChevronUp size={16} />
							</button>
							<button
								type="button"
								class="btn-icon btn-icon-sm"
								aria-label="W dół"
								disabled={i === pinned.length - 1}
								onclick={() => move(href, 1)}
							>
								<ChevronDown size={16} />
							</button>
							<button
								type="button"
								class="btn-icon btn-icon-sm"
								aria-label="Odepnij {item.label}"
								onclick={() => toggle(href)}
							>
								<PinOff size={16} />
							</button>
						</li>
					{/if}
				{/each}
			</ol>
		{/if}
	</section>

	<section class="card preset-filled-surface-100-900 p-4 space-y-3">
		<h3 class="h4 font-semibold">Dostępne</h3>
		<ul class="grid grid-cols-1 sm:grid-cols-2 gap-2">
			{#each NAV_ROUTES.filter((r) => !isPinned(r.href)) as item (item.href)}
				<li>
					<button
						type="button"
						class="flex w-full items-center gap-3 px-3 py-2 rounded-container bg-surface-50-950 hover:preset-tonal-primary text-left"
						onclick={() => toggle(item.href)}
						disabled={pinned.length >= MAX_PINNED}
					>
						<item.icon size={18} />
						<span class="flex-1 text-sm">{item.label}</span>
						<Pin size={14} class="text-surface-600-400" />
					</button>
				</li>
			{/each}
		</ul>
	</section>
</div>
