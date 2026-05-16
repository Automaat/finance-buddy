import { redirect } from '@sveltejs/kit';
import type { PageLoad } from './$types';

// The salary calculator now lives under /simulations/kalkulator.
// This route stays as a permanent redirect for existing bookmarks.
export const load: PageLoad = () => {
	redirect(308, '/simulations/kalkulator');
};
