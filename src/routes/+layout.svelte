<script lang="ts">
	import '../app.css';
	import { page } from '$app/stores';
	import Toast from '$lib/components/Toast.svelte';
	import Confirm from '$lib/components/Confirm.svelte';
	import KeyboardShortcuts from '$lib/components/KeyboardShortcuts.svelte';
	import BottomSheet from '$lib/components/BottomSheet.svelte';
	import { navPrefs } from '$lib/stores/navPrefs.svelte';
	import {
		LayoutDashboard,
		TrendingUp,
		Sparkles,
		Wallet,
		ArrowRightLeft,
		Home,
		ClipboardList,
		Camera,
		Banknote,
		Coins,
		Dices,
		Settings,
		Target,
		User,
		ShieldCheck,
		LogOut,
		MoreHorizontal
	} from 'lucide-svelte';
	import type { LayoutData } from './$types';

	let { children, data }: { children: import('svelte').Snippet; data: LayoutData } = $props();

	const navItems = [
		{ href: '/', label: 'Dashboard', icon: LayoutDashboard },
		{ href: '/metryki', label: 'Metryki', icon: TrendingUp },
		{ href: '/simulations', label: 'Symulacje', icon: Sparkles },
		{ href: '/retirement', label: 'Emerytura', icon: Dices },
		{ href: '/accounts', label: 'Konta', icon: Wallet },
		{ href: '/transactions', label: 'Transakcje', icon: ArrowRightLeft },
		{ href: '/assets', label: 'Majątek', icon: Home },
		{ href: '/bonds', label: 'Obligacje', icon: Coins },
		{ href: '/debts', label: 'Zobowiązania', icon: ClipboardList },
		{ href: '/goals', label: 'Cele', icon: Target },
		{ href: '/snapshots', label: 'Snapshoty', icon: Camera },
		{ href: '/salaries', label: 'Wynagrodzenia', icon: Banknote },
		{ href: '/settings', label: 'Ustawienia', icon: Settings }
	];

	const navByHref = new Map(navItems.map((i) => [i.href, i]));

	const pinnedItems = $derived(
		navPrefs.pinned.map((href) => navByHref.get(href)).filter((i) => i !== undefined)
	);
	const overflowItems = $derived(navItems.filter((i) => !navPrefs.pinned.includes(i.href)));

	const isLoginPage = $derived($page.url.pathname === '/login');

	let moreOpen = $state(false);

	function isActive(href: string): boolean {
		if (href === '/simulations') return $page.url.pathname.startsWith('/simulations');
		if (href === '/settings') return $page.url.pathname.startsWith('/settings');
		return $page.url.pathname === href;
	}

	const overflowActive = $derived(overflowItems.some((i) => isActive(i.href)));

	function closeMore() {
		moreOpen = false;
	}
</script>

<Toast />
<Confirm />

{#if !isLoginPage}
	<KeyboardShortcuts />
{/if}

{#if isLoginPage}
	{@render children?.()}
{:else}
	<div class="flex min-h-screen bg-surface-50-950 text-surface-950-50">
		<aside
			class="hidden md:flex md:flex-col md:w-60 md:shrink-0 md:border-r md:border-surface-200-800 md:bg-surface-100-900"
		>
			<div class="p-4 flex items-center gap-2 text-lg font-bold">
				<ShieldCheck class="text-primary-500" size={24} />
				<span>Finansowa Forteca</span>
			</div>
			<nav class="flex-1 overflow-y-auto p-2 space-y-1">
				{#each navItems as item}
					<a
						href={item.href}
						class="flex items-center gap-3 px-3 py-2 rounded-container text-sm transition-colors
							{isActive(item.href)
							? 'preset-filled-primary-500 font-semibold'
							: 'hover:preset-tonal-primary text-surface-800-200'}"
					>
						<item.icon size={18} />
						<span>{item.label}</span>
					</a>
				{/each}
			</nav>
			{#if data.user}
				<div class="p-3 border-t border-surface-200-800 space-y-2">
					<div class="flex items-center gap-2 px-1 text-sm text-surface-700-300">
						<User size={16} />
						<span class="truncate">{data.user.name || data.user.username}</span>
					</div>
					<form method="POST" action="/logout">
						<button
							type="submit"
							class="flex w-full items-center gap-3 px-3 py-2 rounded-container text-sm hover:preset-tonal-primary text-surface-800-200"
						>
							<LogOut size={18} />
							<span>Wyloguj</span>
						</button>
					</form>
				</div>
			{/if}
		</aside>

		<div class="flex flex-1 flex-col min-w-0">
			<header
				class="md:hidden sticky top-0 z-20 flex items-center justify-between px-4 py-3 bg-surface-100-900 border-b border-surface-200-800"
			>
				<span class="flex items-center gap-2 text-base font-bold">
					<ShieldCheck class="text-primary-500" size={20} />
					<span>Finansowa Forteca</span>
				</span>
				{#if data.user}
					<form method="POST" action="/logout">
						<button type="submit" class="btn-icon btn-icon-sm" aria-label="Wyloguj">
							<LogOut size={20} />
						</button>
					</form>
				{/if}
			</header>

			<main class="flex-1 w-full max-w-[1200px] mx-auto p-4 md:p-6 lg:p-8 pb-24 md:pb-8">
				{@render children?.()}
			</main>

			<nav
				class="md:hidden fixed bottom-0 left-0 right-0 z-30 grid bg-surface-100-900 border-t border-surface-200-800 pb-[env(safe-area-inset-bottom)]"
				style:grid-template-columns="repeat({pinnedItems.length + 1}, minmax(0, 1fr))"
				aria-label="Mobile navigation"
			>
				{#each pinnedItems as item}
					<a
						href={item.href}
						class="flex flex-col items-center justify-center gap-1 px-1 py-2 text-[10px]
							{isActive(item.href) ? 'text-primary-500 font-semibold' : 'text-surface-700-300'}"
					>
						<item.icon size={20} />
						<span class="whitespace-nowrap truncate max-w-full">{item.label}</span>
					</a>
				{/each}
				<button
					type="button"
					class="flex flex-col items-center justify-center gap-1 px-1 py-2 text-[10px]
						{overflowActive || moreOpen ? 'text-primary-500 font-semibold' : 'text-surface-700-300'}"
					aria-label="Więcej opcji nawigacji"
					aria-expanded={moreOpen}
					onclick={() => (moreOpen = true)}
				>
					<MoreHorizontal size={20} />
					<span class="whitespace-nowrap">Więcej</span>
				</button>
			</nav>

			<BottomSheet open={moreOpen} title="Więcej" onClose={closeMore}>
				<ul class="grid grid-cols-4 gap-2">
					{#each overflowItems as item}
						<li>
							<a
								href={item.href}
								onclick={closeMore}
								class="flex flex-col items-center justify-center gap-1 p-3 rounded-container text-[11px]
									{isActive(item.href)
									? 'preset-filled-primary-500 font-semibold'
									: 'hover:preset-tonal-primary text-surface-800-200'}"
							>
								<item.icon size={22} />
								<span class="text-center leading-tight">{item.label}</span>
							</a>
						</li>
					{/each}
				</ul>
			</BottomSheet>
		</div>
	</div>
{/if}
