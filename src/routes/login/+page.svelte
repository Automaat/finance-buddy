<script lang="ts">
	import { enhance } from '$app/forms';
	import { KeyRound, ShieldCheck } from 'lucide-svelte';
	import type { ActionData, PageData } from './$types';

	let { data, form }: { data: PageData; form: ActionData } = $props();
	let submitting = $state(false);

	const oidcErrorMessages: Record<string, string> = {
		oidc_no_user:
			'To konto Authelia nie ma użytkownika w aplikacji — poproś administratora o konto',
		oidc_failed: 'Logowanie przez Authelia nie powiodło się — spróbuj ponownie',
		oidc_unavailable: 'Authelia jest chwilowo niedostępna — zaloguj się hasłem'
	};
	const errorMessage = $derived(
		form?.error ?? (data.oidcError ? oidcErrorMessages[data.oidcError] : undefined)
	);
</script>

<div
	class="flex min-h-screen items-center justify-center bg-surface-50-950 text-surface-950-50 p-4"
>
	<div class="card preset-filled-surface-50-950 w-full max-w-sm shadow-xl p-6 space-y-5">
		<div class="flex items-center gap-2 text-lg font-bold">
			<ShieldCheck class="text-primary-500" size={24} />
			<span>Finansowa Forteca</span>
		</div>

		<form
			method="POST"
			class="space-y-4"
			use:enhance={() => {
				submitting = true;
				return async ({ update }) => {
					await update();
					submitting = false;
				};
			}}
		>
			<label class="label">
				<span class="font-semibold text-sm">Nazwa użytkownika</span>
				<input
					name="username"
					type="text"
					class="input"
					autocomplete="username"
					value={form?.username ?? ''}
					required
				/>
			</label>

			<label class="label">
				<span class="font-semibold text-sm">Hasło</span>
				<input
					name="password"
					type="password"
					class="input"
					autocomplete="current-password"
					required
				/>
			</label>

			<label class="flex items-center gap-2 text-sm">
				<input name="remember_me" type="checkbox" class="checkbox" />
				<span>Zapamiętaj mnie</span>
			</label>

			{#if errorMessage}
				<div class="card preset-filled-error-500 p-3 text-sm" role="alert">{errorMessage}</div>
			{/if}

			<button type="submit" class="btn preset-filled-primary-500 w-full" disabled={submitting}>
				{submitting ? 'Logowanie...' : 'Zaloguj'}
			</button>
		</form>

		{#if data.oidcEnabled}
			<a href="/api/auth/oidc/login" class="btn preset-tonal-primary w-full" data-sveltekit-reload>
				<KeyRound size={18} />
				<span>Zaloguj przez Authelia</span>
			</a>
		{/if}
	</div>
</div>
