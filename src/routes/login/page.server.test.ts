// @vitest-environment node
import { describe, expect, it, vi } from 'vitest';
import { load } from './+page.server';

vi.mock('$env/dynamic/private', () => ({ env: {} }));

type LoadEvent = Parameters<typeof load>[0];

function runLoad(fetchImpl: () => Promise<Response>, search = '') {
	const event = {
		fetch: vi.fn(fetchImpl),
		url: new URL(`https://finance.example.test/login${search}`)
	} as unknown as LoadEvent;
	return load(event);
}

describe('login load — OIDC status', () => {
	it.each([
		['enabled', async () => Response.json({ enabled: true }), true],
		['disabled', async () => Response.json({ enabled: false }), false],
		['backend error', async () => new Response('boom', { status: 500 }), false],
		[
			'backend unreachable',
			async () => {
				throw new Error('connection refused');
			},
			false
		]
	])('%s', async (_name, fetchImpl, expected) => {
		vi.spyOn(console, 'error').mockImplementation(() => {});
		expect(await runLoad(fetchImpl)).toEqual({ oidcEnabled: expected, oidcError: null });
	});

	it('passes the error code from the query string through', async () => {
		const data = await runLoad(async () => Response.json({ enabled: true }), '?error=oidc_no_user');
		expect(data).toEqual({ oidcEnabled: true, oidcError: 'oidc_no_user' });
	});
});
