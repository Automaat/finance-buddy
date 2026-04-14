<script lang="ts">
	import '../app.css';
	import { page } from '$app/stores';
	import {
		LayoutDashboard,
		TrendingUp,
		Sparkles,
		Landmark,
		Building2,
		Wallet,
		ArrowRightLeft,
		Home,
		ClipboardList,
		Camera,
		Banknote,
		Calculator,
		Settings,
		User,
		ShieldCheck
	} from 'lucide-svelte';

	const navItems = [
		{ href: '/', label: 'Dashboard', icon: LayoutDashboard },
		{ href: '/metryki', label: 'Metryki', icon: TrendingUp },
		{ href: '/simulations', label: 'Symulacje', icon: Sparkles },
		{ href: '/simulations/mortgage', label: 'Hipoteka', icon: Landmark },
		{ href: '/simulations/zus', label: 'Emerytura ZUS', icon: Building2 },
		{ href: '/accounts', label: 'Konta', icon: Wallet },
		{ href: '/transactions', label: 'Transakcje', icon: ArrowRightLeft },
		{ href: '/assets', label: 'Majątek', icon: Home },
		{ href: '/debts', label: 'Zobowiązania', icon: ClipboardList },
		{ href: '/snapshots', label: 'Snapshoty', icon: Camera },
		{ href: '/salaries', label: 'Wynagrodzenia', icon: Banknote },
		{ href: '/kalkulator', label: 'Kalkulator', icon: Calculator },
		{ href: '/config', label: 'Konfiguracja', icon: Settings },
		{ href: '/settings', label: 'Ustawienia', icon: User }
	];

	let { children } = $props();
</script>

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
				{@const active = $page.url.pathname === item.href}
				{@const Icon = item.icon}
				<a
					href={item.href}
					class="flex items-center gap-3 px-3 py-2 rounded-container text-sm transition-colors
						{active
						? 'preset-filled-primary-500 font-semibold'
						: 'hover:preset-tonal-primary text-surface-800-200'}"
				>
					<Icon size={18} />
					<span>{item.label}</span>
				</a>
			{/each}
		</nav>
	</aside>

	<div class="flex flex-1 flex-col min-w-0">
		<header
			class="md:hidden sticky top-0 z-20 flex items-center justify-between px-4 py-3 bg-surface-100-900 border-b border-surface-200-800"
		>
			<span class="flex items-center gap-2 text-base font-bold">
				<ShieldCheck class="text-primary-500" size={20} />
				<span>Finansowa Forteca</span>
			</span>
		</header>

		<main class="flex-1 w-full max-w-[1200px] mx-auto p-4 md:p-6 lg:p-8 pb-24 md:pb-8">
			{@render children?.()}
		</main>

		<nav
			class="md:hidden fixed bottom-0 left-0 right-0 z-30 flex overflow-x-auto bg-surface-100-900 border-t border-surface-200-800"
			aria-label="Mobile navigation"
		>
			{#each navItems as item}
				{@const active = $page.url.pathname === item.href}
				{@const Icon = item.icon}
				<a
					href={item.href}
					class="flex flex-col items-center justify-center gap-1 min-w-16 shrink-0 px-3 py-2 text-[10px]
						{active ? 'text-primary-500 font-semibold' : 'text-surface-700-300'}"
				>
					<Icon size={20} />
					<span class="whitespace-nowrap">{item.label}</span>
				</a>
			{/each}
		</nav>
	</div>
</div>
