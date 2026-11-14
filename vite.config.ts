import adapter from '@sveltejs/adapter-node';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';
import tailwindcss from '@tailwindcss/vite';
import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vitest/config';

export default defineConfig({
	plugins: [
		tailwindcss(),
		sveltekit({
			preprocess: vitePreprocess(),
			adapter: adapter(),
			csrf: { trustedOrigins: ['http://localhost:5173'] }
		})
	],
	resolve: {
		conditions: ['browser']
	},
	test: {
		globals: true,
		environment: 'jsdom',
		include: ['src/**/*.{test,spec}.{js,ts}'],
		coverage: {
			provider: 'v8',
			reporter: ['text', 'json', 'html', 'lcov'],
			exclude: ['node_modules/**', '.svelte-kit/**', 'build/**', '**/*.config.*', '**/.*rc.*'],
			// Thresholds are a ratchet against the current baseline, not a
			// target — raise them as coverage improves so regressions can't
			// slip past CI while the codebase already covers more.
			thresholds: {
				statements: 79,
				branches: 60,
				functions: 82,
				lines: 81
			}
		}
	}
});
