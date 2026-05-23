// Global UI state for keyboard-driven affordances: the shortcut help
// overlay and the command palette. Mounted once via the root layout.

let helpOpen = $state(false);
let paletteOpen = $state(false);

export const shortcutsUI = {
	get helpOpen() {
		return helpOpen;
	},
	get paletteOpen() {
		return paletteOpen;
	},
	openHelp() {
		helpOpen = true;
		paletteOpen = false;
	},
	closeHelp() {
		helpOpen = false;
	},
	toggleHelp() {
		helpOpen = !helpOpen;
		if (helpOpen) paletteOpen = false;
	},
	openPalette() {
		paletteOpen = true;
		helpOpen = false;
	},
	closePalette() {
		paletteOpen = false;
	}
};
