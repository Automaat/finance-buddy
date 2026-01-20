<script lang="ts">
	import '../app.css';
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

	let collapsed = false;

	function toggleCollapse() {
		collapsed = !collapsed;
	}
</script>

<div class="app">
	<nav class="navbar" class:collapsed>
		<div class="nav-container">
			{#if !collapsed}
				<h1 class="brand">💪 Finansowa Forteca</h1>
			{/if}

			<div class="nav-links">
				{#each navItems as item}
					<a
						href={item.href}
						class="nav-link"
						class:active={$page.url.pathname === item.href}
						title={item.label}
					>
						<span class="icon">{item.icon}</span>
						{#if !collapsed}
							<span class="label">{item.label}</span>
						{/if}
					</a>
				{/each}
			</div>

			<button class="toggle-btn" on:click={toggleCollapse} aria-label="Toggle navigation">
				{collapsed ? '→' : '←'}
			</button>
		</div>
	</nav>

	<main class="main">
		<slot />
	</main>
</div>

<style>
	.app {
		display: flex;
		min-height: 100vh;
	}

	.navbar {
		width: 250px;
		background: var(--color-bg-card);
		border-right: 1px solid var(--color-border);
		position: sticky;
		top: 0;
		height: 100vh;
		overflow-y: auto;
		transition: width 0.3s ease;
	}

	.navbar.collapsed {
		width: 80px;
	}

	.nav-container {
		display: flex;
		flex-direction: column;
		gap: var(--size-6);
		padding: var(--size-6) var(--size-4);
	}

	.toggle-btn {
		background: var(--color-bg);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-2);
		padding: var(--size-2);
		cursor: pointer;
		font-size: var(--font-size-3);
		color: var(--color-text);
		transition: all 0.2s;
		margin-top: auto;
	}

	.toggle-btn:hover {
		background: var(--color-accent);
		color: var(--nord6);
	}

	.brand {
		font-size: var(--font-size-4);
		font-weight: var(--font-weight-7);
		color: var(--color-primary);
		margin: 0;
		text-align: center;
	}

	.nav-links {
		display: flex;
		flex-direction: column;
		gap: var(--size-2);
	}

	.nav-link {
		display: flex;
		align-items: center;
		gap: var(--size-2);
		padding: var(--size-3) var(--size-4);
		border-radius: var(--radius-2);
		color: var(--color-text);
		text-decoration: none;
		font-weight: var(--font-weight-5);
		transition: all 0.2s;
		white-space: nowrap;
	}

	.navbar.collapsed .nav-link {
		justify-content: center;
		padding: var(--size-3);
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

	.label {
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.main {
		flex: 1;
		padding: var(--size-6);
	}
</style>
