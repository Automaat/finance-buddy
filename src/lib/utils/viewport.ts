import { readable, type Readable } from 'svelte/store';
import { browser } from '$app/environment';

function mediaQueryStore(query: string): Readable<boolean> {
	return readable(false, (set) => {
		if (!browser) return;
		const mql = window.matchMedia(query);
		set(mql.matches);
		const onChange = (e: MediaQueryListEvent) => set(e.matches);
		mql.addEventListener('change', onChange);
		return () => mql.removeEventListener('change', onChange);
	});
}

export const isMobile = mediaQueryStore('(max-width: 767px)');
export const isTablet = mediaQueryStore('(min-width: 768px) and (max-width: 1023px)');
export const isDesktop = mediaQueryStore('(min-width: 1024px)');
