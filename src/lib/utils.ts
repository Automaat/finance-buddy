import { type ClassValue, clsx } from 'clsx';
import { twMerge } from 'tailwind-merge';
import type { Component } from 'svelte';

export function cn(...inputs: ClassValue[]) {
	return twMerge(clsx(inputs));
}

export type WithElementRef<
	T extends Record<string, any>,
	Element extends HTMLElement = HTMLElement
> = T & {
	ref?: Element | null;
};
