import { type ClassValue, clsx } from 'clsx';
import { twMerge } from 'tailwind-merge';

export function cn(...inputs: ClassValue[]) {
	return twMerge(clsx(inputs));
}

export type WithElementRef<
	T extends Record<string, any>,
	Element extends HTMLElement = HTMLElement
> = T & {
	ref?: Element | null;
};
