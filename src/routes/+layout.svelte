<script lang="ts">
	import '@mskalski/home-ui/styles';
	import { Navbar } from '@mskalski/home-ui';
	import { page } from '$app/stores';

	const navItems = [
		{ href: '/', label: 'Dashboard', icon: '📊' },
		{ href: '/metryki', label: 'Metryki', icon: '📈' },
		{ href: '/simulations', label: 'Symulacje', icon: '🔮' },
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
		padding: var(--size-6);
		max-width: 1200px;
		margin: 0 auto;
		width: 100%;
	}
</style>
