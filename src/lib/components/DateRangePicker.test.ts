import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/svelte';
import { goto } from '$app/navigation';
import DateRangePicker from './DateRangePicker.svelte';

const page = vi.hoisted(() => ({ url: new URL('http://localhost/') }));

vi.mock('$app/navigation', () => ({
	goto: vi.fn()
}));

vi.mock('$app/state', () => ({ page }));

function setUrl(url: string) {
	page.url = new URL(url);
}

describe('DateRangePicker', () => {
	beforeEach(async () => {
		vi.clearAllMocks();
		setUrl('http://localhost/metryki');
	});

	it('renders all preset chips plus custom', () => {
		render(DateRangePicker, { props: {} });
		expect(screen.getByRole('button', { name: '1M' })).toBeTruthy();
		expect(screen.getByRole('button', { name: '3M' })).toBeTruthy();
		expect(screen.getByRole('button', { name: '6M' })).toBeTruthy();
		expect(screen.getByRole('button', { name: '1R' })).toBeTruthy();
		expect(screen.getByRole('button', { name: '3L' })).toBeTruthy();
		expect(screen.getByRole('button', { name: '5L' })).toBeTruthy();
		expect(screen.getByRole('button', { name: 'Wszystko' })).toBeTruthy();
		expect(screen.getByRole('button', { name: 'Własny' })).toBeTruthy();
	});

	it('marks "Wszystko" as active when no range param is present', () => {
		render(DateRangePicker, { props: {} });
		expect(screen.getByRole('button', { name: 'Wszystko' }).getAttribute('aria-pressed')).toBe(
			'true'
		);
	});

	it('marks the URL preset as active', async () => {
		setUrl('http://localhost/metryki?range=3m');
		render(DateRangePicker, { props: {} });
		expect(screen.getByRole('button', { name: '3M' }).getAttribute('aria-pressed')).toBe('true');
	});

	it('navigates with range= when a preset chip is clicked', async () => {
		render(DateRangePicker, { props: {} });
		await fireEvent.click(screen.getByRole('button', { name: '1M' }));
		expect(goto).toHaveBeenCalledWith('/metryki?range=1m', { reset: false });
	});

	it('navigates without range= when "Wszystko" is clicked from another preset', async () => {
		setUrl('http://localhost/metryki?range=1m');
		render(DateRangePicker, { props: {} });
		await fireEvent.click(screen.getByRole('button', { name: 'Wszystko' }));
		expect(goto).toHaveBeenCalledWith('/metryki', { reset: false });
	});

	it('does nothing when clicking the already-active preset', async () => {
		setUrl('http://localhost/metryki?range=1m');
		render(DateRangePicker, { props: {} });
		await fireEvent.click(screen.getByRole('button', { name: '1M' }));
		expect(goto).not.toHaveBeenCalled();
	});

	it('clicking "Własny" with no bounds sets range=custom', async () => {
		render(DateRangePicker, { props: {} });
		await fireEvent.click(screen.getByRole('button', { name: 'Własny' }));
		expect(goto).toHaveBeenCalledWith('/metryki?range=custom', { reset: false });
	});

	it('shows the custom date form when active range is custom', async () => {
		setUrl('http://localhost/metryki?range=custom&date_from=2024-01-01&date_to=2024-12-31');
		render(DateRangePicker, { props: {} });
		expect(screen.getByLabelText('Data od')).toBeTruthy();
		expect(screen.getByLabelText('Data do')).toBeTruthy();
		expect(screen.getByRole('button', { name: 'Zastosuj' })).toBeTruthy();
	});

	it('infers custom mode when only date_from/date_to are present', async () => {
		setUrl('http://localhost/metryki?date_from=2024-01-01');
		render(DateRangePicker, { props: {} });
		expect(screen.getByRole('button', { name: 'Własny' }).getAttribute('aria-pressed')).toBe(
			'true'
		);
	});

	it('Apply submits range=custom plus the entered bounds', async () => {
		setUrl('http://localhost/metryki?range=custom');
		render(DateRangePicker, { props: {} });
		const from = screen.getByLabelText('Data od') as HTMLInputElement;
		const to = screen.getByLabelText('Data do') as HTMLInputElement;
		await fireEvent.input(from, { target: { value: '2024-01-01' } });
		await fireEvent.input(to, { target: { value: '2024-12-31' } });
		await fireEvent.click(screen.getByRole('button', { name: 'Zastosuj' }));
		expect(goto).toHaveBeenCalled();
		const lastCall = (goto as ReturnType<typeof vi.fn>).mock.calls.at(-1);
		const target = String(lastCall?.[0] ?? '');
		expect(target).toContain('range=custom');
		expect(target).toContain('date_from=2024-01-01');
		expect(target).toContain('date_to=2024-12-31');
	});

	it('Apply with empty bounds drops the date params from the URL', async () => {
		setUrl('http://localhost/metryki?range=custom&date_from=2024-01-01&date_to=2024-12-31');
		render(DateRangePicker, { props: {} });
		const from = screen.getByLabelText('Data od') as HTMLInputElement;
		const to = screen.getByLabelText('Data do') as HTMLInputElement;
		await fireEvent.input(from, { target: { value: '' } });
		await fireEvent.input(to, { target: { value: '' } });
		await fireEvent.click(screen.getByRole('button', { name: 'Zastosuj' }));
		const lastCall = (goto as ReturnType<typeof vi.fn>).mock.calls.at(-1);
		const target = String(lastCall?.[0] ?? '');
		expect(target).toContain('range=custom');
		expect(target).not.toContain('date_from=');
		expect(target).not.toContain('date_to=');
	});

	it('uses the explicit path prop instead of $page.url.pathname', async () => {
		setUrl('http://localhost/something-else');
		render(DateRangePicker, { props: { path: '/metryki' } });
		await fireEvent.click(screen.getByRole('button', { name: '1M' }));
		expect(goto).toHaveBeenCalledWith('/metryki?range=1m', { reset: false });
	});
});
