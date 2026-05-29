import { redirect } from '@sveltejs/kit';
import type { PageLoad } from './$types';

// /investments → default to the holdings tab.
export const load: PageLoad = () => {
	throw redirect(307, '/investments/holdings');
};
