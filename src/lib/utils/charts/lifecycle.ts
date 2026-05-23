import * as echarts from 'echarts';
import type { ECharts } from 'echarts';

export interface ChartHandle {
	chart: ECharts;
	dispose: () => void;
}

// createChart wires echarts.init + a ResizeObserver on the container and
// returns a ChartHandle. Use inside onMount and call handle.dispose() from
// the cleanup callback so navigation away can't leak canvases or observers.
//
// ResizeObserver is preferred over a global window resize listener: it
// fires only when the container actually changes size (e.g. layout shifts,
// sidebar toggles), and it's bound to the element so multiple charts on a
// page don't share one global handler.
export function createChart(container: HTMLElement): ChartHandle {
	const chart = echarts.init(container);
	let disposed = false;
	let ro: ResizeObserver | undefined;
	if (typeof ResizeObserver !== 'undefined') {
		ro = new ResizeObserver(() => {
			// Skip when teardown has started — ResizeObserver entries can flush
			// on the next animation frame, after dispose() ran, and ECharts
			// logs a warning on resize() of a disposed instance.
			if (!disposed) chart.resize();
		});
		ro.observe(container);
	}
	return {
		chart,
		dispose: () => {
			disposed = true;
			ro?.disconnect();
			chart.dispose();
		}
	};
}
