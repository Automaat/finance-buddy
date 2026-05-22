import { expect, type Page } from '@playwright/test';

export function uniqueName(prefix: string): string {
	const stamp = Date.now().toString(36) + Math.random().toString(36).slice(2, 6);
	return `${prefix}-${stamp}`;
}

// openDialog clicks a trigger button and waits for its modal, retrying the
// click to absorb the brief window after SSR before SvelteKit finishes
// hydrating — where the button exists but its handler isn't attached yet.
export async function openDialog(page: Page, triggerName: RegExp | string): Promise<void> {
	const dialog = page.getByRole('dialog');
	await expect(async () => {
		if (!(await dialog.isVisible())) {
			await page.getByRole('button', { name: triggerName }).click();
		}
		await expect(dialog).toBeVisible({ timeout: 1000 });
	}).toPass({ timeout: 15_000 });
}
