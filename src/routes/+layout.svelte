<script lang="ts">
	import '@mskalski/home-ui/styles';
	import { Navbar } from '@mskalski/home-ui';
	import { page } from '$app/stores';

	const navItems = [
		{ href: '/', label: 'Dashboard', icon: '📊' },
		{ href: '/metryki', label: 'Metryki', icon: '📈' },
		{ href: '/simulations', label: 'Symulacje', icon: '🔮' },
		{ href: '/simulations/mortgage', label: 'Hipoteka', icon: '🏦' },
		{ href: '/accounts', label: 'Konta', icon: '💰' },
		{ href: '/transactions', label: 'Transakcje', icon: '💸' },
		{ href: '/assets', label: 'Majątek', icon: '🏠' },
		{ href: '/debts', label: 'Zobowiązania', icon: '📋' },
		{ href: '/snapshots', label: 'Snapshoty', icon: '📸' },
		{ href: '/salaries', label: 'Wynagrodzenia', icon: '💵' },
		{ href: '/config', label: 'Konfiguracja', icon: '⚙️' }
	];
</script>

<div class="app">
	<Navbar items={navItems} brand="💪 Finansowa Forteca" currentPath={$page.url.pathname} />

	<main class="main">
		<slot />
	</main>
</div>

<style>
	.app {
		display: flex;
		min-height: 100vh;
	}

	.main {
		flex: 1;
		padding: var(--spacing-mobile);
		max-width: 1200px;
		margin: 0 auto;
		width: 100%;
	}

	@media (min-width: 768px) {
		.main {
			padding: var(--spacing-tablet);
		}
	}

	@media (min-width: 1024px) {
		.main {
			padding: var(--spacing-desktop);
		}
	}

	@media (max-width: 767px) {
		.main {
			margin-top: 60px;
		}
	}

	/* Force light mode regardless of OS preference */
	:global(:root) {
		--surface-1: hsl(210 20% 98%);
		--surface-2: hsl(210 16% 93%);
		--surface-3: hsl(210 14% 89%);
		--surface-4: hsl(210 14% 85%);
		--text-1: hsl(210 10% 23%);
		--text-2: hsl(210 7% 40%);
	}
</style>
