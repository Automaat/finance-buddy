import type { LayoutServerLoad } from './$types';

// Expose the authenticated user to the layout (and, via parent(), to pages).
export const load: LayoutServerLoad = ({ locals }) => {
	return { user: locals.user };
};
