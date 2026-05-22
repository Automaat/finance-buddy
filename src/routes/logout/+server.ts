import { redirect } from '@sveltejs/kit';
import type { RequestHandler } from './$types';

export const POST: RequestHandler = ({ cookies }) => {
	cookies.delete('fb_token', { path: '/' });
	redirect(303, '/login');
};
