import type { PageLoad } from './$types';
import { loadConfigDefaults } from '$lib/utils/configDefaults';

export const load: PageLoad = async ({ fetch }) => {
	const defaults = await loadConfigDefaults(fetch);
	return { defaults };
};
