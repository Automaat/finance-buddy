import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { GOTO_ROUTES, SHORTCUT_BINDINGS, isModifierKeyPressed, isTypingTarget } from './shortcuts';

describe('isTypingTarget', () => {
	let container: HTMLElement;

	beforeEach(() => {
		container = document.createElement('div');
		document.body.appendChild(container);
	});

	afterEach(() => {
		container.remove();
	});

	it('returns false for null', () => {
		expect(isTypingTarget(null)).toBe(false);
	});

	it('returns false for non-input elements', () => {
		const div = document.createElement('div');
		expect(isTypingTarget(div)).toBe(false);
	});

	it('returns true for input', () => {
		const el = document.createElement('input');
		expect(isTypingTarget(el)).toBe(true);
	});

	it('returns true for textarea', () => {
		const el = document.createElement('textarea');
		expect(isTypingTarget(el)).toBe(true);
	});

	it('returns true for select', () => {
		const el = document.createElement('select');
		expect(isTypingTarget(el)).toBe(true);
	});

	it('returns true for contenteditable', () => {
		const el = document.createElement('div');
		el.setAttribute('contenteditable', 'true');
		container.appendChild(el);
		expect(isTypingTarget(el)).toBe(true);
	});
});

describe('isModifierKeyPressed', () => {
	function evt(init: Partial<KeyboardEventInit>): KeyboardEvent {
		return new KeyboardEvent('keydown', init);
	}

	it('returns true for ctrl', () => {
		expect(isModifierKeyPressed(evt({ ctrlKey: true }))).toBe(true);
	});

	it('returns true for meta', () => {
		expect(isModifierKeyPressed(evt({ metaKey: true }))).toBe(true);
	});

	it('returns true for alt', () => {
		expect(isModifierKeyPressed(evt({ altKey: true }))).toBe(true);
	});

	it('returns false when no modifier is pressed', () => {
		expect(isModifierKeyPressed(evt({}))).toBe(false);
	});

	it('returns false for shift alone (shift is not considered a modifier here)', () => {
		expect(isModifierKeyPressed(evt({ shiftKey: true }))).toBe(false);
	});
});

describe('GOTO_ROUTES', () => {
	it('covers h, a, s, t', () => {
		const keys = GOTO_ROUTES.map((r) => r.key).sort();
		expect(keys).toEqual(['a', 'h', 's', 't']);
	});

	it('all routes start with /', () => {
		for (const r of GOTO_ROUTES) {
			expect(r.href.startsWith('/')).toBe(true);
		}
	});
});

describe('SHORTCUT_BINDINGS', () => {
	it('includes the acceptance-criteria bindings', () => {
		const keys = SHORTCUT_BINDINGS.map((b) => b.keys);
		expect(keys).toContain('n');
		expect(keys).toContain('g h');
		expect(keys).toContain('g a');
		expect(keys).toContain('g s');
		expect(keys).toContain('g t');
		expect(keys).toContain('?');
		expect(keys).toContain('Cmd/Ctrl + K');
	});
});
