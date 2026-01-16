<script lang="ts">
	import '../app.css';
	import { page } from '$app/stores';

	const navItems = [
		{ href: '/', label: 'Dashboard', icon: '📊' },
		{ href: '/accounts', label: 'Konta', icon: '💰' },
		{ href: '/goals', label: 'Cele', icon: '🎯' },
		{ href: '/investments', label: 'Inwestycje', icon: '📈' },
		{ href: '/snapshots', label: 'Snapshoty', icon: '📸' }
	];
</script>

<div class="min-h-screen bg-background">
	<nav class="border-b bg-card">
		<div class="container mx-auto px-4 py-4">
			<div class="flex items-center justify-between">
				<h1 class="text-2xl font-bold text-primary">💪 Finansowa Forteca</h1>

				<div class="flex gap-2">
					{#each navItems as item}
						<a
							href={item.href}
							class="px-4 py-2 rounded-md hover:bg-accent hover:text-accent-foreground transition-colors font-medium"
							class:bg-primary={$page.url.pathname === item.href}
							class:text-primary-foreground={$page.url.pathname === item.href}
							class:text-foreground={$page.url.pathname !== item.href}
						>
							<span class="mr-2">{item.icon}</span>
							{item.label}
						</a>
					{/each}
				</div>
			</div>
		</div>
	</nav>

	<main class="container mx-auto px-4 py-8">
		<slot />
	</main>
</div>
