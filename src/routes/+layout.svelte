<script lang="ts">
	import '../app.css';
	import { page } from '$app/stores';

	const navItems = [
		{ href: '/', label: 'Dashboard', icon: '📊' },
		{ href: '/metryki', label: 'Metryki', icon: '📈' },
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
	<nav class="navbar">
		<div class="container nav-container">
			<h1 class="brand">💪 Finansowa Forteca</h1>

			<div class="nav-links">
				{#each navItems as item}
					<a href={item.href} class="nav-link" class:active={$page.url.pathname === item.href}>
						<span class="icon">{item.icon}</span>
						{item.label}
					</a>
				{/each}
			</div>
		</div>
	</nav>

	<main class="container main">
		<slot />
	</main>
</div>

<style>
	.app {
		min-height: 100vh;
	}

	.navbar {
		background: var(--color-bg-card);
		border-bottom: 1px solid var(--color-border);
		margin-bottom: var(--size-6);
	}

	.nav-container {
		display: flex;
		justify-content: space-between;
		align-items: center;
		padding-block: var(--size-4);
	}

	.brand {
		font-size: var(--font-size-4);
		font-weight: var(--font-weight-7);
		color: var(--color-primary);
		margin: 0;
	}

	.nav-links {
		display: flex;
		gap: var(--size-2);
	}

	.nav-link {
		display: flex;
		align-items: center;
		gap: var(--size-2);
		padding: var(--size-2) var(--size-4);
		border-radius: var(--radius-2);
		color: var(--color-text);
		text-decoration: none;
		font-weight: var(--font-weight-5);
		transition: all 0.2s;
	}

	.nav-link:hover {
		background: var(--color-accent);
		color: var(--nord6);
	}

	.nav-link.active {
		background: var(--color-primary);
		color: var(--nord6);
	}

	.icon {
		font-size: var(--font-size-3);
	}

	.main {
		padding-block: var(--size-6);
	}
</style>
