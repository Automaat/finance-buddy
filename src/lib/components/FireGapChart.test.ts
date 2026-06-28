import { describe, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/svelte';
import FireGapChart from './FireGapChart.svelte';

vi.mock('echarts', () => ({
	init: vi.fn(() => ({ setOption: vi.fn(), resize: vi.fn(), dispose: vi.fn() }))
}));

describe('FireGapChart', () => {
	it('keeps return assumptions owned by the parent page', () => {
		render(FireGapChart, { props: { expectedReturnPct: 7, expectedReturnDisabled: true } });

		expect(screen.queryByLabelText(/Oczekiwana stopa zwrotu/)).toBeNull();
	});
});
