import { redirect } from '@sveltejs/kit';
import type { PageLoad } from './$types';

// Legacy path preserved for stable bookmarks/links; the page moved under
// /investments so the bonds + holdings views could share a tab strip.
export const load: PageLoad = () => {
	throw redirect(308, '/investments/bonds');
};
