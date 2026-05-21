<script lang="ts">
	interface Props {
		class?: string;
		width?: string;
		height?: string;
		rounded?: 'sm' | 'md' | 'lg' | 'full';
		'aria-label'?: string;
	}

	let {
		class: className = '',
		width,
		height,
		rounded = 'md',
		'aria-label': ariaLabel = 'Ładowanie'
	}: Props = $props();

	const radiusClass = $derived(
		rounded === 'full'
			? 'rounded-full'
			: rounded === 'lg'
				? 'rounded-lg'
				: rounded === 'sm'
					? 'rounded-sm'
					: 'rounded-md'
	);
</script>

<div
	class="skeleton {radiusClass} {className}"
	style:width
	style:height
	role="status"
	aria-busy="true"
	aria-live="polite"
	aria-label={ariaLabel}
></div>

<style>
	.skeleton {
		display: block;
		background: linear-gradient(
			90deg,
			rgba(127, 127, 127, 0.12) 0%,
			rgba(127, 127, 127, 0.22) 50%,
			rgba(127, 127, 127, 0.12) 100%
		);
		background-size: 200% 100%;
		opacity: 0;
		animation:
			skeleton-shimmer 1.4s ease-in-out infinite,
			skeleton-reveal 0s linear 150ms forwards;
	}

	:global(.dark) .skeleton {
		background: linear-gradient(
			90deg,
			rgba(255, 255, 255, 0.06) 0%,
			rgba(255, 255, 255, 0.14) 50%,
			rgba(255, 255, 255, 0.06) 100%
		);
		background-size: 200% 100%;
	}

	@keyframes skeleton-shimmer {
		0% {
			background-position: 200% 0;
		}
		100% {
			background-position: -200% 0;
		}
	}

	@keyframes skeleton-reveal {
		to {
			opacity: 1;
		}
	}

	@media (prefers-reduced-motion: reduce) {
		.skeleton {
			animation: skeleton-reveal 0s linear 150ms forwards;
			background: rgba(127, 127, 127, 0.18);
		}
	}
</style>
