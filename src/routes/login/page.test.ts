import { describe, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/svelte';
import Page from './+page.svelte';

vi.mock('$app/forms', () => ({ enhance: () => ({ destroy: () => {} }) }));

type LoginData = { oidcEnabled: boolean; oidcError: string | null };
type LoginForm = { error: string; username: string } | null;

function renderLogin(data: LoginData, form: LoginForm = null) {
	return render(Page, { props: { data: { user: null, ...data }, form } });
}

describe('Login page — Authelia button', () => {
	it('shows the Authelia link when OIDC is enabled', () => {
		renderLogin({ oidcEnabled: true, oidcError: null });
		const link = screen.getByRole('link', { name: 'Zaloguj przez Authelia' });
		expect(link.getAttribute('href')).toBe('/api/auth/oidc/login');
		expect(link.hasAttribute('data-sveltekit-reload')).toBe(true);
	});

	it('hides the Authelia link when OIDC is disabled', () => {
		renderLogin({ oidcEnabled: false, oidcError: null });
		expect(screen.queryByRole('link', { name: 'Zaloguj przez Authelia' })).toBeNull();
		expect(screen.getByRole('button', { name: 'Zaloguj' })).toBeTruthy();
	});
});

describe('Login page — OIDC error messages', () => {
	it.each([
		['oidc_no_user', 'To konto Authelia nie ma użytkownika w aplikacji'],
		['oidc_failed', 'Logowanie przez Authelia nie powiodło się'],
		['oidc_unavailable', 'Authelia jest chwilowo niedostępna']
	])('renders a message for %s', (code, expected) => {
		renderLogin({ oidcEnabled: true, oidcError: code });
		expect(screen.getByRole('alert').textContent).toContain(expected);
	});

	it('shows no alert without an error code', () => {
		renderLogin({ oidcEnabled: true, oidcError: null });
		expect(screen.queryByRole('alert')).toBeNull();
	});

	it('ignores unknown error codes', () => {
		renderLogin({ oidcEnabled: true, oidcError: 'something_else' });
		expect(screen.queryByRole('alert')).toBeNull();
	});

	it('prefers the password form error over the OIDC error', () => {
		renderLogin(
			{ oidcEnabled: true, oidcError: 'oidc_failed' },
			{ error: 'Nieprawidłowa nazwa użytkownika lub hasło', username: 'x' }
		);
		expect(screen.getByRole('alert').textContent).toBe('Nieprawidłowa nazwa użytkownika lub hasło');
	});
});
