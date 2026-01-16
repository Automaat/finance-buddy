import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vitest/config';

export default defineConfig({
	plugins: [sveltekit()],
	resolve: {
		conditions: ['browser']
	},
	test: {
		globals: true,
		environment: 'jsdom',
		include: ['frontend/**/*.{test,spec}.{js,ts}'],
		coverage: {
			provider: 'v8',
			reporter: ['text', 'json', 'html', 'lcov'],
			exclude: ['node_modules/**', '.svelte-kit/**', 'build/**', '**/*.config.*', '**/.*rc.*']
		}
	}
});
