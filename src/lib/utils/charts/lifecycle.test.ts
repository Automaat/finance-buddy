import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { createChart } from './lifecycle';
import * as echarts from 'echarts';

vi.mock('echarts', () => ({
	init: vi.fn()
}));

describe('createChart', () => {
	const disposeSpy = vi.fn();
	const resizeSpy = vi.fn();
	const observe = vi.fn();
	const disconnect = vi.fn();

	beforeEach(() => {
		vi.mocked(echarts.init).mockReturnValue({
			dispose: disposeSpy,
			resize: resizeSpy
		} as unknown as ReturnType<typeof echarts.init>);
		// Stub ResizeObserver so this test runs without browser support.
		// vitest's jsdom env doesn't ship one.
		(globalThis as { ResizeObserver?: unknown }).ResizeObserver = class {
			constructor(public cb: ResizeObserverCallback) {}
			observe = observe;
			disconnect = disconnect;
		};
	});

	afterEach(() => {
		vi.clearAllMocks();
		delete (globalThis as { ResizeObserver?: unknown }).ResizeObserver;
	});

	it('initializes echarts on the container and observes it', () => {
		const el = document.createElement('div');
		createChart(el);
		expect(echarts.init).toHaveBeenCalledWith(el);
		expect(observe).toHaveBeenCalledWith(el);
	});

	it('dispose tears down both the observer and the chart', () => {
		const el = document.createElement('div');
		const handle = createChart(el);
		handle.dispose();
		expect(disconnect).toHaveBeenCalledTimes(1);
		expect(disposeSpy).toHaveBeenCalledTimes(1);
	});

	it('still works when ResizeObserver is unavailable', () => {
		delete (globalThis as { ResizeObserver?: unknown }).ResizeObserver;
		const el = document.createElement('div');
		const handle = createChart(el);
		// Dispose must not throw and must still tear down the chart.
		handle.dispose();
		expect(disposeSpy).toHaveBeenCalledTimes(1);
	});

	it('skips resize after dispose', () => {
		let observerCb: ResizeObserverCallback | undefined;
		(globalThis as { ResizeObserver?: unknown }).ResizeObserver = class {
			constructor(cb: ResizeObserverCallback) {
				observerCb = cb;
			}
			observe = observe;
			disconnect = disconnect;
		};
		const el = document.createElement('div');
		const handle = createChart(el);
		handle.dispose();
		// Simulate a late callback flushing after disconnect.
		observerCb?.([], {} as ResizeObserver);
		expect(resizeSpy).not.toHaveBeenCalled();
	});
});
