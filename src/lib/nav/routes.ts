import {
	LayoutDashboard,
	TrendingUp,
	Sparkles,
	Wallet,
	ArrowRightLeft,
	Home,
	ClipboardList,
	Camera,
	Banknote,
	Dices,
	Settings,
	Target,
	Briefcase,
	Compass,
	Globe
} from 'lucide-svelte';

export const NAV_ROUTES = [
	{ href: '/', label: 'Pulpit', icon: LayoutDashboard },
	{ href: '/metryki', label: 'Metryki', icon: TrendingUp },
	{ href: '/simulations', label: 'Symulacje', icon: Sparkles },
	{ href: '/optimizer', label: 'Optymalizator', icon: Compass },
	{ href: '/retirement', label: 'Emerytura', icon: Dices },
	{ href: '/accounts', label: 'Konta', icon: Wallet },
	{ href: '/transactions', label: 'Transakcje', icon: ArrowRightLeft },
	{ href: '/assets', label: 'Majątek', icon: Home },
	{ href: '/investments/holdings', label: 'Inwestycje', icon: Briefcase },
	{ href: '/ekspozycja', label: 'Ekspozycja', icon: Globe },
	{ href: '/debts', label: 'Zobowiązania', icon: ClipboardList },
	{ href: '/goals', label: 'Cele', icon: Target },
	{ href: '/snapshots', label: 'Snapshoty', icon: Camera },
	{ href: '/salaries', label: 'Wynagrodzenia', icon: Banknote },
	{ href: '/settings', label: 'Ustawienia', icon: Settings }
] as const;

export type NavRoute = (typeof NAV_ROUTES)[number];

export const NAV_HREFS: ReadonlySet<string> = new Set(NAV_ROUTES.map((r) => r.href));
