# Finance Buddy - Implementation Plan

Web app to replace "Finansowa Forteca.xlsx" - beautiful, self-hosted personal finance tracker.

## Tech Stack

- **Framework:** SvelteKit 2.x + TypeScript
- **Database:** PostgreSQL (existing instance) + Drizzle ORM
- **UI:** shadcn-svelte + Tailwind CSS
- **Charts:** Apache ECharts
- **Deployment:** Docker container
- **Users:** Multi-user (Marcin + Ewa, shared login for MVP)

**Why this stack:**

- SvelteKit: 50% less JS than Next.js, better dashboard performance ([source](https://dev.to/paulthedev/sveltekit-vs-nextjs-in-2026-why-the-underdog-is-winning-a-developers-deep-dive-155b))
- ECharts: Superior for financial time-series, handles millions of data points ([source](https://www.metabase.com/blog/best-open-source-chart-library))
- PostgreSQL: Already running, better for multi-user

## Data Structure (from Excel)

**12 sheets identified:**

- `wartosc_netto` - net worth snapshots over time
- `inwestycje` - investments (IKE, IKZE, PPK, stocks, bonds)
- `cele_krotkoterminowe` - short-term goals with progress tracking
- `raty_0` - 0% installment payments
- `ike_ikze_limity` - retirement account contribution limits
- `wplaty_inwestycje`, `wplaty_hipoteka`, `wynagrodzenie` - tracking deposits/payments/salary
- `parametry_podstawowe`, `helper_data`, `helper_data_series`, `Strategia` - configs/calculations

**Assets tracked:** bank accounts, emergency fund, retirement savings, real estate (mieszkanie, działka), car, electronics, collections, sports equipment, hobbies

**12 charts** need to be recreated in ECharts

---

## Phase 1: Project Setup

### 1.1 Initialize SvelteKit Project

```bash
npm create svelte@latest .
# Choose: Skeleton project, TypeScript, ESLint, Prettier
npm install
```

### 1.2 Install Dependencies

```bash
# Database
npm install drizzle-orm postgres dotenv
npm install -D drizzle-kit

# UI Components
npm install -D tailwindcss postcss autoprefixer
npx tailwindcss init -p
npx shadcn-svelte@latest init

# Charts
npm install echarts echarts-for-svelte

# Forms & Validation
npm install bits-ui formsnap sveltekit-superforms zod

# Date handling
npm install @internationalized/date
```

### 1.3 Configure Environment

Create `.env`:

```env
DATABASE_URL=postgresql://user:pass@your-postgres-host:5432/finance
APP_PASSWORD=shared-secret-for-mvp
PUBLIC_APP_URL=http://localhost:3000
```

### 1.4 Setup Tailwind

Update `tailwind.config.js`:

```js
export default {
	content: ['./src/**/*.{html,js,svelte,ts}'],
	theme: { extend: {} },
	plugins: []
};
```

Add to `src/app.css`:

```css
@tailwind base;
@tailwind components;
@tailwind utilities;
```

---

## Phase 2: Database Schema

### 2.1 Create Schema File

**File:** `src/lib/db/schema.ts`

```typescript
import {
	pgTable,
	serial,
	text,
	numeric,
	timestamp,
	date,
	boolean,
	integer
} from 'drizzle-orm/pg-core';

export const accounts = pgTable('accounts', {
	id: serial('id').primaryKey(),
	name: text('name').notNull(),
	type: text('type').notNull(), // 'asset' | 'liability'
	category: text('category').notNull(), // 'bank', 'ike', 'ikze', 'ppk', 'real_estate', etc.
	owner: text('owner'), // 'Marcin' | 'Ewa' | 'Shared'
	currency: text('currency').default('PLN'),
	isActive: boolean('is_active').default(true),
	createdAt: timestamp('created_at').defaultNow()
});

export const snapshots = pgTable('snapshots', {
	id: serial('id').primaryKey(),
	date: date('date').notNull().unique(),
	notes: text('notes'),
	createdAt: timestamp('created_at').defaultNow()
});

export const snapshotValues = pgTable('snapshot_values', {
	id: serial('id').primaryKey(),
	snapshotId: integer('snapshot_id')
		.references(() => snapshots.id, { onDelete: 'cascade' })
		.notNull(),
	accountId: integer('account_id')
		.references(() => accounts.id, { onDelete: 'cascade' })
		.notNull(),
	value: numeric('value', { precision: 15, scale: 2 }).notNull()
});

export const goals = pgTable('goals', {
	id: serial('id').primaryKey(),
	name: text('name').notNull(),
	targetAmount: numeric('target_amount', { precision: 15, scale: 2 }).notNull(),
	targetDate: date('target_date').notNull(),
	currentAmount: numeric('current_amount', { precision: 15, scale: 2 }).default('0'),
	monthlyContribution: numeric('monthly_contribution', { precision: 15, scale: 2 }),
	isCompleted: boolean('is_completed').default(false),
	createdAt: timestamp('created_at').defaultNow()
});
```

### 2.2 Create Database Client

**File:** `src/lib/db/index.ts`

```typescript
import { drizzle } from 'drizzle-orm/postgres-js';
import postgres from 'postgres';
import { DATABASE_URL } from '$env/static/private';
import * as schema from './schema';

const client = postgres(DATABASE_URL);
export const db = drizzle(client, { schema });
```

### 2.3 Configure Drizzle Kit

**File:** `drizzle.config.ts`

```typescript
import type { Config } from 'drizzle-kit';
import * as dotenv from 'dotenv';

dotenv.config();

export default {
	schema: './src/lib/db/schema.ts',
	out: './drizzle',
	driver: 'pg',
	dbCredentials: {
		connectionString: process.env.DATABASE_URL!
	}
} satisfies Config;
```

### 2.4 Generate & Run Migrations

```bash
npx drizzle-kit generate:pg
npx drizzle-kit push:pg
```

---

## Phase 3: Excel Data Migration

### 3.1 Create Migration Script

**File:** `scripts/migrate_excel.py`

```python
import pandas as pd
import psycopg2
from datetime import datetime
import os

# Configuration
EXCEL_FILE = 'Finansowa Forteca.xlsx'
DATABASE_URL = os.getenv('DATABASE_URL', 'postgresql://user:pass@localhost:5432/finance')

def determine_type(account_name):
    """Determine if account is asset or liability"""
    liabilities = ['raty 0', 'hipoteka']
    return 'liability' if any(l in account_name.lower() for l in liabilities) else 'asset'

def determine_category(account_name):
    """Map account name to category"""
    name_lower = account_name.lower()
    if 'ike' in name_lower: return 'ike'
    if 'ikze' in name_lower: return 'ikze'
    if 'ppk' in name_lower: return 'ppk'
    if 'konto' in name_lower or 'oszczednosc' in name_lower: return 'bank'
    if 'mieszkanie' in name_lower or 'działka' in name_lower: return 'real_estate'
    if 'samochod' in name_lower: return 'vehicle'
    if 'obligacje' in name_lower: return 'bonds'
    if 'akcje' in name_lower: return 'stocks'
    if 'hipoteka' in name_lower: return 'mortgage'
    if 'raty' in name_lower: return 'installment'
    return 'other'

def determine_owner(account_name):
    """Determine account owner from name"""
    name_lower = account_name.lower()
    if 'marcin' in name_lower: return 'Marcin'
    if 'ewa' in name_lower: return 'Ewa'
    return 'Shared'

def migrate():
    # Read Excel
    print(f"Reading {EXCEL_FILE}...")
    df_net_worth = pd.read_excel(EXCEL_FILE, sheet_name='wartosc_netto')
    df_goals = pd.read_excel(EXCEL_FILE, sheet_name='cele_krotkoterminowe')

    # Connect to PostgreSQL
    print("Connecting to database...")
    conn = psycopg2.connect(DATABASE_URL)
    cur = conn.cursor()

    # 1. Create accounts
    print("Creating accounts...")
    accounts_map = {}
    skip_columns = ['Data', 'ROR', 'wartość netto']

    for column in df_net_worth.columns:
        if column not in skip_columns and not pd.isna(df_net_worth[column].iloc[0]):
            cur.execute(
                """
                INSERT INTO accounts (name, type, category, owner)
                VALUES (%s, %s, %s, %s)
                RETURNING id
                """,
                (column, determine_type(column), determine_category(column), determine_owner(column))
            )
            accounts_map[column] = cur.fetchone()[0]
            print(f"  Created: {column} (ID: {accounts_map[column]})")

    # 2. Create snapshots with values
    print("Creating snapshots...")
    snapshot_count = 0

    for _, row in df_net_worth.iterrows():
        date = row['Data']
        if pd.isna(date):
            continue

        # Create snapshot
        cur.execute(
            "INSERT INTO snapshots (date) VALUES (%s) RETURNING id",
            (date,)
        )
        snapshot_id = cur.fetchone()[0]

        # Add values for each account
        for account_name, account_id in accounts_map.items():
            value = row[account_name]
            if pd.notna(value) and value != 0:
                cur.execute(
                    """
                    INSERT INTO snapshot_values (snapshot_id, account_id, value)
                    VALUES (%s, %s, %s)
                    """,
                    (snapshot_id, account_id, float(value))
                )

        snapshot_count += 1
        if snapshot_count % 10 == 0:
            print(f"  Processed {snapshot_count} snapshots...")

    print(f"Created {snapshot_count} snapshots")

    # 3. Import goals
    print("Creating goals...")
    goal_count = 0

    for _, row in df_goals.iterrows():
        if pd.notna(row['Cel']):
            cur.execute(
                """
                INSERT INTO goals (name, target_amount, target_date, current_amount, monthly_contribution)
                VALUES (%s, %s, %s, %s, %s)
                """,
                (
                    row['Cel'],
                    float(row['Kwota']) if pd.notna(row['Kwota']) else 0,
                    row['Termin'] if pd.notna(row['Termin']) else None,
                    float(row['Ile już mamy']) if pd.notna(row['Ile już mamy']) else 0,
                    float(row['Ile miesięcznie musimy odkładać']) if pd.notna(row['Ile miesięcznie musimy odkładać']) else None
                )
            )
            goal_count += 1

    print(f"Created {goal_count} goals")

    # Commit and close
    conn.commit()
    cur.close()
    conn.close()

    print("\n✓ Migration completed successfully!")
    print(f"  - {len(accounts_map)} accounts")
    print(f"  - {snapshot_count} snapshots")
    print(f"  - {goal_count} goals")

if __name__ == '__main__':
    migrate()
```

### 3.2 Run Migration

```bash
cd scripts
pip3 install pandas openpyxl psycopg2-binary python-dotenv
python3 migrate_excel.py
```

---

## Phase 4: Core UI Components

### 4.1 Install shadcn-svelte Components

```bash
npx shadcn-svelte@latest add card
npx shadcn-svelte@latest add button
npx shadcn-svelte@latest add input
npx shadcn-svelte@latest add label
npx shadcn-svelte@latest add table
npx shadcn-svelte@latest add badge
npx shadcn-svelte@latest add separator
```

### 4.2 Create Utility Functions

**File:** `src/lib/utils/format.ts`

```typescript
export function formatPLN(value: number | string): string {
	const num = typeof value === 'string' ? parseFloat(value) : value;
	return new Intl.NumberFormat('pl-PL', {
		style: 'currency',
		currency: 'PLN',
		minimumFractionDigits: 2
	}).format(num);
}

export function formatDate(date: string | Date): string {
	const d = typeof date === 'string' ? new Date(date) : date;
	return new Intl.DateTimeFormat('pl-PL').format(d);
}

export function formatPercent(value: number): string {
	return new Intl.NumberFormat('pl-PL', {
		style: 'percent',
		minimumFractionDigits: 2
	}).format(value / 100);
}
```

**File:** `src/lib/utils/calculations.ts`

```typescript
export function calculateNetWorth(assets: number, liabilities: number): number {
	return assets - liabilities;
}

export function calculateChange(
	current: number,
	previous: number
): {
	value: number;
	percent: number;
} {
	const value = current - previous;
	const percent = previous !== 0 ? (value / previous) * 100 : 0;
	return { value, percent };
}

export function calculateGoalProgress(current: number, target: number): number {
	return target !== 0 ? (current / target) * 100 : 0;
}

export function calculateMonthsRemaining(targetDate: Date): number {
	const now = new Date();
	const months =
		(targetDate.getFullYear() - now.getFullYear()) * 12 + (targetDate.getMonth() - now.getMonth());
	return Math.max(0, months);
}
```

### 4.3 Create Layout

**File:** `src/routes/+layout.svelte`

```svelte
<script lang="ts">
	import '../app.css';
	import { page } from '$app/stores';

	const navItems = [
		{ href: '/', label: 'Dashboard', icon: '📊' },
		{ href: '/accounts', label: 'Konta', icon: '💰' },
		{ href: '/goals', label: 'Cele', icon: '🎯' },
		{ href: '/investments', label: 'Inwestycje', icon: '📈' },
		{ href: '/snapshots', label: 'Snapshoty', icon: '📸' }
	];
</script>

<div class="min-h-screen bg-background">
	<nav class="border-b">
		<div class="container mx-auto px-4 py-4">
			<div class="flex items-center justify-between">
				<h1 class="text-2xl font-bold">💪 Finansowa Forteca</h1>

				<div class="flex gap-4">
					{#each navItems as item}
						<a
							href={item.href}
							class="px-4 py-2 rounded-md hover:bg-accent transition-colors"
							class:bg-accent={$page.url.pathname === item.href}
						>
							<span class="mr-2">{item.icon}</span>
							{item.label}
						</a>
					{/each}
				</div>
			</div>
		</div>
	</nav>

	<main class="container mx-auto px-4 py-8">
		<slot />
	</main>
</div>
```

---

## Phase 5: Dashboard Page (Priority 1)

### 5.1 Server-side Data Loading

**File:** `src/routes/+page.server.ts`

```typescript
import { db } from '$lib/db';
import { snapshots, snapshotValues, accounts } from '$lib/db/schema';
import { sql, eq, desc } from 'drizzle-orm';

export async function load() {
	// Get net worth history
	const history = await db
		.select({
			date: snapshots.date,
			netWorth: sql<number>`
        COALESCE(SUM(
          CASE
            WHEN ${accounts.type} = 'asset' THEN ${snapshotValues.value}::numeric
            ELSE -${snapshotValues.value}::numeric
          END
        ), 0)
      `.as('net_worth')
		})
		.from(snapshots)
		.leftJoin(snapshotValues, eq(snapshots.id, snapshotValues.snapshotId))
		.leftJoin(accounts, eq(snapshotValues.accountId, accounts.id))
		.where(eq(accounts.isActive, true))
		.groupBy(snapshots.id, snapshots.date)
		.orderBy(snapshots.date);

	// Get latest snapshot for current values
	const latest = await db
		.select({
			type: accounts.type,
			total: sql<number>`SUM(${snapshotValues.value}::numeric)`.as('total')
		})
		.from(snapshotValues)
		.innerJoin(accounts, eq(snapshotValues.accountId, accounts.id))
		.innerJoin(snapshots, eq(snapshotValues.snapshotId, snapshots.id))
		.where(eq(accounts.isActive, true))
		.groupBy(accounts.type)
		.orderBy(desc(snapshots.date))
		.limit(1);

	const totalAssets = latest.find((l) => l.type === 'asset')?.total || 0;
	const totalLiabilities = latest.find((l) => l.type === 'liability')?.total || 0;
	const currentNetWorth = Number(history[history.length - 1]?.netWorth || 0);
	const lastMonthNetWorth = Number(history[history.length - 2]?.netWorth || 0);

	// Get asset allocation
	const allocation = await db
		.select({
			category: accounts.category,
			owner: accounts.owner,
			total: sql<number>`SUM(${snapshotValues.value}::numeric)`.as('total')
		})
		.from(snapshotValues)
		.innerJoin(accounts, eq(snapshotValues.accountId, accounts.id))
		.innerJoin(
			sql`(SELECT id FROM ${snapshots} ORDER BY date DESC LIMIT 1)`,
			sql`${snapshotValues.snapshotId} = id`
		)
		.where(eq(accounts.type, 'asset'))
		.groupBy(accounts.category, accounts.owner);

	return {
		netWorthHistory: history.map((h) => ({
			date: h.date,
			value: Number(h.netWorth)
		})),
		currentNetWorth,
		changeVsLastMonth: currentNetWorth - lastMonthNetWorth,
		totalAssets: Number(totalAssets),
		totalLiabilities: Number(totalLiabilities),
		allocation: allocation.map((a) => ({
			category: a.category,
			owner: a.owner,
			value: Number(a.total)
		}))
	};
}
```

### 5.2 Dashboard Component

**File:** `src/routes/+page.svelte`

```svelte
<script lang="ts">
	import { onMount } from 'svelte';
	import * as echarts from 'echarts';
	import type { EChartsOption } from 'echarts';
	import { Card, CardContent, CardHeader, CardTitle } from '$lib/components/ui/card';
	import { formatPLN, formatPercent } from '$lib/utils/format';
	import { calculateChange } from '$lib/utils/calculations';

	export let data;

	let chartContainer: HTMLDivElement;
	let pieChartContainer: HTMLDivElement;

	onMount(() => {
		// Net Worth Line Chart
		const lineChart = echarts.init(chartContainer);

		const lineOption: EChartsOption = {
			title: {
				text: 'Wartość Netto w Czasie',
				left: 'center'
			},
			tooltip: {
				trigger: 'axis',
				formatter: (params: any) => {
					const date = new Date(params[0].value[0]).toLocaleDateString('pl-PL');
					const value = formatPLN(params[0].value[1]);
					return `${date}<br/>Wartość: ${value}`;
				}
			},
			xAxis: {
				type: 'time'
			},
			yAxis: {
				type: 'value',
				axisLabel: {
					formatter: (value: number) => formatPLN(value)
				}
			},
			series: [
				{
					data: data.netWorthHistory.map((h: any) => [h.date, h.value]),
					type: 'line',
					smooth: true,
					areaStyle: {
						color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
							{ offset: 0, color: 'rgba(59, 130, 246, 0.5)' },
							{ offset: 1, color: 'rgba(59, 130, 246, 0.1)' }
						])
					},
					lineStyle: {
						color: 'rgb(59, 130, 246)',
						width: 2
					}
				}
			],
			grid: {
				left: '80px',
				right: '40px'
			}
		};

		lineChart.setOption(lineOption);

		// Asset Allocation Pie Chart
		const pieChart = echarts.init(pieChartContainer);

		const pieOption: EChartsOption = {
			title: {
				text: 'Alokacja Aktywów',
				left: 'center'
			},
			tooltip: {
				trigger: 'item',
				formatter: '{b}: {c} PLN ({d}%)'
			},
			series: [
				{
					type: 'pie',
					radius: ['40%', '70%'],
					data: data.allocation.map((a: any) => ({
						name: `${a.category}${a.owner ? ` (${a.owner})` : ''}`,
						value: a.value
					})),
					emphasis: {
						itemStyle: {
							shadowBlur: 10,
							shadowOffsetX: 0,
							shadowColor: 'rgba(0, 0, 0, 0.5)'
						}
					}
				}
			]
		};

		pieChart.setOption(pieOption);

		// Responsive resize
		window.addEventListener('resize', () => {
			lineChart.resize();
			pieChart.resize();
		});
	});

	const change = calculateChange(
		data.currentNetWorth,
		data.currentNetWorth - data.changeVsLastMonth
	);
</script>

<svelte:head>
	<title>Dashboard | Finansowa Forteca</title>
</svelte:head>

<div class="space-y-8">
	<div>
		<h1 class="text-4xl font-bold mb-2">Dashboard</h1>
		<p class="text-muted-foreground">Twoja sytuacja finansowa w jednym miejscu</p>
	</div>

	<!-- KPI Cards -->
	<div class="grid grid-cols-1 md:grid-cols-3 gap-6">
		<Card>
			<CardHeader>
				<CardTitle class="text-sm font-medium">Wartość Netto</CardTitle>
			</CardHeader>
			<CardContent>
				<div class="text-3xl font-bold">{formatPLN(data.currentNetWorth)}</div>
				<p
					class="text-sm mt-2"
					class:text-green-600={data.changeVsLastMonth >= 0}
					class:text-red-600={data.changeVsLastMonth < 0}
				>
					{data.changeVsLastMonth >= 0 ? '↑' : '↓'}
					{formatPLN(Math.abs(data.changeVsLastMonth))}
					({formatPercent(Math.abs(change.percent))})
					<span class="text-muted-foreground">vs poprzedni miesiąc</span>
				</p>
			</CardContent>
		</Card>

		<Card>
			<CardHeader>
				<CardTitle class="text-sm font-medium">Aktywa</CardTitle>
			</CardHeader>
			<CardContent>
				<div class="text-3xl font-bold text-green-600">{formatPLN(data.totalAssets)}</div>
				<p class="text-sm text-muted-foreground mt-2">Suma wszystkich aktywów</p>
			</CardContent>
		</Card>

		<Card>
			<CardHeader>
				<CardTitle class="text-sm font-medium">Zobowiązania</CardTitle>
			</CardHeader>
			<CardContent>
				<div class="text-3xl font-bold text-red-600">{formatPLN(data.totalLiabilities)}</div>
				<p class="text-sm text-muted-foreground mt-2">Suma wszystkich zobowiązań</p>
			</CardContent>
		</Card>
	</div>

	<!-- Charts -->
	<div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
		<Card>
			<CardContent class="p-6">
				<div bind:this={chartContainer} class="w-full h-96"></div>
			</CardContent>
		</Card>

		<Card>
			<CardContent class="p-6">
				<div bind:this={pieChartContainer} class="w-full h-96"></div>
			</CardContent>
		</Card>
	</div>
</div>
```

---

## Phase 6: Account Management

### 6.1 Accounts List Page

**File:** `src/routes/accounts/+page.server.ts`

```typescript
import { db } from '$lib/db';
import { accounts, snapshotValues, snapshots } from '$lib/db/schema';
import { sql, eq, desc } from 'drizzle-orm';

export async function load() {
	// Get latest values for all accounts
	const latestSnapshot = await db
		.select({ id: snapshots.id })
		.from(snapshots)
		.orderBy(desc(snapshots.date))
		.limit(1);

	const accountsWithValues = await db
		.select({
			id: accounts.id,
			name: accounts.name,
			type: accounts.type,
			category: accounts.category,
			owner: accounts.owner,
			value: sql<number>`COALESCE(${snapshotValues.value}::numeric, 0)`.as('value')
		})
		.from(accounts)
		.leftJoin(
			snapshotValues,
			sql`${snapshotValues.accountId} = ${accounts.id} AND ${snapshotValues.snapshotId} = ${latestSnapshot[0].id}`
		)
		.where(eq(accounts.isActive, true))
		.orderBy(accounts.type, accounts.category);

	// Group by type
	const grouped = {
		assets: accountsWithValues.filter((a) => a.type === 'asset'),
		liabilities: accountsWithValues.filter((a) => a.type === 'liability')
	};

	return { accounts: grouped };
}
```

**File:** `src/routes/accounts/+page.svelte`

```svelte
<script lang="ts">
	import { Card, CardContent, CardHeader, CardTitle } from '$lib/components/ui/card';
	import { Badge } from '$lib/components/ui/badge';
	import { Button } from '$lib/components/ui/button';
	import { formatPLN } from '$lib/utils/format';

	export let data;

	const categoryLabels: Record<string, string> = {
		bank: 'Konto bankowe',
		ike: 'IKE',
		ikze: 'IKZE',
		ppk: 'PPK',
		bonds: 'Obligacje',
		stocks: 'Akcje',
		real_estate: 'Nieruchomości',
		vehicle: 'Pojazd',
		mortgage: 'Hipoteka',
		installment: 'Raty',
		other: 'Inne'
	};
</script>

<svelte:head>
	<title>Konta | Finansowa Forteca</title>
</svelte:head>

<div class="space-y-8">
	<div class="flex items-center justify-between">
		<div>
			<h1 class="text-4xl font-bold mb-2">Konta</h1>
			<p class="text-muted-foreground">Zarządzaj swoimi kontami i aktywami</p>
		</div>
		<Button href="/accounts/new">+ Dodaj Konto</Button>
	</div>

	<!-- Assets -->
	<section>
		<h2 class="text-2xl font-semibold mb-4 text-green-600">💰 Aktywa</h2>
		<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
			{#each data.accounts.assets as account}
				<Card class="hover:shadow-md transition-shadow cursor-pointer">
					<a href="/accounts/{account.id}">
						<CardHeader>
							<div class="flex items-start justify-between">
								<CardTitle class="text-lg">{account.name}</CardTitle>
								{#if account.owner}
									<Badge variant="outline">{account.owner}</Badge>
								{/if}
							</div>
							<p class="text-sm text-muted-foreground">{categoryLabels[account.category]}</p>
						</CardHeader>
						<CardContent>
							<p class="text-2xl font-bold">{formatPLN(Number(account.value))}</p>
						</CardContent>
					</a>
				</Card>
			{/each}
		</div>
	</section>

	<!-- Liabilities -->
	<section>
		<h2 class="text-2xl font-semibold mb-4 text-red-600">💸 Zobowiązania</h2>
		<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
			{#each data.accounts.liabilities as account}
				<Card class="hover:shadow-md transition-shadow cursor-pointer">
					<a href="/accounts/{account.id}">
						<CardHeader>
							<div class="flex items-start justify-between">
								<CardTitle class="text-lg">{account.name}</CardTitle>
								{#if account.owner}
									<Badge variant="outline">{account.owner}</Badge>
								{/if}
							</div>
							<p class="text-sm text-muted-foreground">{categoryLabels[account.category]}</p>
						</CardHeader>
						<CardContent>
							<p class="text-2xl font-bold text-red-600">{formatPLN(Number(account.value))}</p>
						</CardContent>
					</a>
				</Card>
			{/each}
		</div>
	</section>
</div>
```

---

## Phase 7: Snapshot Creation

### 7.1 Snapshot API

**File:** `src/routes/api/snapshots/+server.ts`

```typescript
import { json } from '@sveltejs/kit';
import { db } from '$lib/db';
import { snapshots, snapshotValues } from '$lib/db/schema';

export async function POST({ request }) {
	const body = await request.json();
	const { date, values } = body;

	try {
		// Create snapshot
		const [snapshot] = await db.insert(snapshots).values({ date, notes: body.notes }).returning();

		// Insert values
		await db.insert(snapshotValues).values(
			values.map((v: any) => ({
				snapshotId: snapshot.id,
				accountId: v.accountId,
				value: v.value.toString()
			}))
		);

		return json({ success: true, snapshot });
	} catch (error) {
		return json({ success: false, error: error.message }, { status: 500 });
	}
}
```

### 7.2 Snapshot Creation Page

**File:** `src/routes/snapshots/new/+page.server.ts`

```typescript
import { db } from '$lib/db';
import { accounts, snapshots, snapshotValues } from '$lib/db/schema';
import { eq, desc } from 'drizzle-orm';

export async function load() {
	// Get all active accounts
	const allAccounts = await db
		.select()
		.from(accounts)
		.where(eq(accounts.isActive, true))
		.orderBy(accounts.type, accounts.category);

	// Get latest snapshot values for pre-filling
	const latestSnapshot = await db.select().from(snapshots).orderBy(desc(snapshots.date)).limit(1);

	let latestValues = {};
	if (latestSnapshot.length > 0) {
		const values = await db
			.select()
			.from(snapshotValues)
			.where(eq(snapshotValues.snapshotId, latestSnapshot[0].id));

		latestValues = Object.fromEntries(values.map((v) => [v.accountId, v.value]));
	}

	return {
		accounts: allAccounts.map((a) => ({
			...a,
			latestValue: latestValues[a.id] || '0'
		}))
	};
}
```

**File:** `src/routes/snapshots/new/+page.svelte`

```svelte
<script lang="ts">
	import { goto } from '$app/navigation';
	import { Card, CardContent, CardHeader, CardTitle } from '$lib/components/ui/card';
	import { Input } from '$lib/components/ui/input';
	import { Label } from '$lib/components/ui/label';
	import { Button } from '$lib/components/ui/button';
	import { Separator } from '$lib/components/ui/separator';

	export let data;

	let snapshotDate = new Date().toISOString().split('T')[0];
	let values: Record<number, string> = {};
	let notes = '';
	let loading = false;

	// Pre-fill with latest values
	data.accounts.forEach((account) => {
		values[account.id] = account.latestValue;
	});

	async function handleSubmit() {
		loading = true;

		const response = await fetch('/api/snapshots', {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({
				date: snapshotDate,
				notes,
				values: Object.entries(values).map(([accountId, value]) => ({
					accountId: parseInt(accountId),
					value: parseFloat(value) || 0
				}))
			})
		});

		if (response.ok) {
			goto('/');
		} else {
			alert('Błąd podczas zapisywania snapshot');
		}

		loading = false;
	}

	const groupedAccounts = {
		assets: data.accounts.filter((a) => a.type === 'asset'),
		liabilities: data.accounts.filter((a) => a.type === 'liability')
	};
</script>

<svelte:head>
	<title>Nowy Snapshot | Finansowa Forteca</title>
</svelte:head>

<div class="max-w-4xl mx-auto space-y-8">
	<div>
		<h1 class="text-4xl font-bold mb-2">Nowy Snapshot</h1>
		<p class="text-muted-foreground">Zaktualizuj wartości wszystkich kont</p>
	</div>

	<form on:submit|preventDefault={handleSubmit} class="space-y-6">
		<!-- Date -->
		<Card>
			<CardHeader>
				<CardTitle>Data Snapshot</CardTitle>
			</CardHeader>
			<CardContent>
				<Label for="date">Data</Label>
				<Input id="date" type="date" bind:value={snapshotDate} required />

				<Label for="notes" class="mt-4">Notatki (opcjonalne)</Label>
				<Input id="notes" type="text" bind:value={notes} placeholder="Dodaj notatkę..." />
			</CardContent>
		</Card>

		<!-- Assets -->
		<Card>
			<CardHeader>
				<CardTitle class="text-green-600">💰 Aktywa</CardTitle>
			</CardHeader>
			<CardContent class="space-y-4">
				{#each groupedAccounts.assets as account}
					<div>
						<Label for="account-{account.id}">
							{account.name}
							<span class="text-sm text-muted-foreground">({account.category})</span>
						</Label>
						<Input
							id="account-{account.id}"
							type="number"
							step="0.01"
							bind:value={values[account.id]}
							placeholder="0.00"
							required
						/>
					</div>
				{/each}
			</CardContent>
		</Card>

		<!-- Liabilities -->
		<Card>
			<CardHeader>
				<CardTitle class="text-red-600">💸 Zobowiązania</CardTitle>
			</CardHeader>
			<CardContent class="space-y-4">
				{#each groupedAccounts.liabilities as account}
					<div>
						<Label for="account-{account.id}">
							{account.name}
							<span class="text-sm text-muted-foreground">({account.category})</span>
						</Label>
						<Input
							id="account-{account.id}"
							type="number"
							step="0.01"
							bind:value={values[account.id]}
							placeholder="0.00"
							required
						/>
					</div>
				{/each}
			</CardContent>
		</Card>

		<!-- Submit -->
		<div class="flex gap-4">
			<Button type="submit" disabled={loading} class="flex-1">
				{loading ? 'Zapisywanie...' : '💾 Zapisz Snapshot'}
			</Button>
			<Button type="button" variant="outline" on:click={() => goto('/')}>Anuluj</Button>
		</div>
	</form>
</div>
```

---

## Phase 8: Goals Page

### 8.1 Goals Data Loading

**File:** `src/routes/goals/+page.server.ts`

```typescript
import { db } from '$lib/db';
import { goals } from '$lib/db/schema';
import { eq } from 'drizzle-orm';

export async function load() {
	const allGoals = await db
		.select()
		.from(goals)
		.where(eq(goals.isCompleted, false))
		.orderBy(goals.targetDate);

	return { goals: allGoals };
}
```

### 8.2 Goals Page

**File:** `src/routes/goals/+page.svelte`

```svelte
<script lang="ts">
	import { Card, CardContent, CardHeader, CardTitle } from '$lib/components/ui/card';
	import { Progress } from '$lib/components/ui/progress';
	import { Badge } from '$lib/components/ui/badge';
	import { formatPLN, formatDate } from '$lib/utils/format';
	import { calculateGoalProgress, calculateMonthsRemaining } from '$lib/utils/calculations';

	export let data;
</script>

<svelte:head>
	<title>Cele | Finansowa Forteca</title>
</svelte:head>

<div class="space-y-8">
	<div>
		<h1 class="text-4xl font-bold mb-2">🎯 Cele Finansowe</h1>
		<p class="text-muted-foreground">Śledź postęp swoich celów krótkoterminowych</p>
	</div>

	<div class="grid grid-cols-1 md:grid-cols-2 gap-6">
		{#each data.goals as goal}
			{@const progress = calculateGoalProgress(
				Number(goal.currentAmount),
				Number(goal.targetAmount)
			)}
			{@const monthsLeft = calculateMonthsRemaining(new Date(goal.targetDate))}

			<Card>
				<CardHeader>
					<div class="flex items-start justify-between">
						<CardTitle>{goal.name}</CardTitle>
						<Badge variant={monthsLeft < 6 ? 'destructive' : 'default'}>
							{monthsLeft} mies.
						</Badge>
					</div>
				</CardHeader>
				<CardContent class="space-y-4">
					<!-- Progress Bar -->
					<div>
						<div class="flex justify-between text-sm mb-2">
							<span>Postęp</span>
							<span class="font-semibold">{progress.toFixed(1)}%</span>
						</div>
						<Progress value={progress} />
					</div>

					<!-- Amounts -->
					<div class="grid grid-cols-2 gap-4 text-sm">
						<div>
							<p class="text-muted-foreground">Aktualna kwota</p>
							<p class="font-semibold">{formatPLN(Number(goal.currentAmount))}</p>
						</div>
						<div>
							<p class="text-muted-foreground">Cel</p>
							<p class="font-semibold">{formatPLN(Number(goal.targetAmount))}</p>
						</div>
					</div>

					<!-- Remaining -->
					<div class="grid grid-cols-2 gap-4 text-sm">
						<div>
							<p class="text-muted-foreground">Pozostało</p>
							<p class="font-semibold text-orange-600">
								{formatPLN(Number(goal.targetAmount) - Number(goal.currentAmount))}
							</p>
						</div>
						<div>
							<p class="text-muted-foreground">Miesięczna wpłata</p>
							<p class="font-semibold text-blue-600">
								{formatPLN(Number(goal.monthlyContribution))}
							</p>
						</div>
					</div>

					<!-- Target Date -->
					<div class="text-sm">
						<p class="text-muted-foreground">Termin</p>
						<p class="font-semibold">{formatDate(goal.targetDate)}</p>
					</div>
				</CardContent>
			</Card>
		{/each}
	</div>
</div>
```

---

## Phase 9: Docker Deployment

### 9.1 Create Dockerfile

**File:** `Dockerfile`

```dockerfile
# Build stage
FROM node:20-alpine AS builder

WORKDIR /app

# Copy package files
COPY package*.json ./

# Install dependencies
RUN npm ci

# Copy source code
COPY . .

# Build app
RUN npm run build

# Production stage
FROM node:20-alpine

WORKDIR /app

# Copy built app and dependencies
COPY --from=builder /app/build ./build
COPY --from=builder /app/node_modules ./node_modules
COPY --from=builder /app/package.json ./

# Expose port
EXPOSE 3000

# Set environment
ENV NODE_ENV=production

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD node -e "require('http').get('http://localhost:3000/', (r) => process.exit(r.statusCode === 200 ? 0 : 1))"

# Start app
CMD ["node", "build"]
```

### 9.2 Create docker-compose.yml

**File:** `docker-compose.yml`

```yaml
version: '3.8'

services:
  finance-app:
    build: .
    container_name: finance-buddy
    ports:
      - '3000:3000'
    environment:
      - DATABASE_URL=postgresql://finance:password@your-postgres-host:5432/finance
      - ORIGIN=http://localhost:3000
      - APP_PASSWORD=shared-secret-change-me
    restart: unless-stopped
    depends_on:
      - postgres # Remove if using external PostgreSQL


  # Optional: Include PostgreSQL if not using external instance
  # postgres:
  #   image: postgres:16-alpine
  #   container_name: finance-postgres
  #   environment:
  #     - POSTGRES_USER=finance
  #     - POSTGRES_PASSWORD=password
  #     - POSTGRES_DB=finance
  #   volumes:
  #     - postgres-data:/var/lib/postgresql/data
  #   restart: unless-stopped
# volumes:
#   postgres-data:
```

### 9.3 Create .dockerignore

**File:** `.dockerignore`

```
node_modules
.svelte-kit
build
.env
.env.*
.DS_Store
*.log
npm-debug.log*
.vscode
.idea
*.xlsx
scripts/*.py
drizzle
```

### 9.4 Build & Run

```bash
# Build image
docker build -t finance-buddy .

# Run container
docker run -d \
  --name finance-buddy \
  -p 3000:3000 \
  -e DATABASE_URL="postgresql://user:pass@host:5432/finance" \
  -e ORIGIN="https://finance.yourdomain.com" \
  -e APP_PASSWORD="your-secure-password" \
  --restart unless-stopped \
  finance-buddy

# Or with docker-compose
docker-compose up -d
```

---

## Phase 10: Final Steps

### 10.1 Add Progress Component

```bash
npx shadcn-svelte@latest add progress
```

### 10.2 Authentication (Simple MVP)

**File:** `src/hooks.server.ts`

```typescript
import { redirect } from '@sveltejs/kit';
import type { Handle } from '@sveltejs/kit';

const APP_PASSWORD = process.env.APP_PASSWORD || 'changeme';

export const handle: Handle = async ({ event, resolve }) => {
	const session = event.cookies.get('finance_session');
	const isLoginPage = event.url.pathname === '/login';

	// Check authentication
	if (!session && !isLoginPage) {
		throw redirect(302, '/login');
	}

	return resolve(event);
};
```

**File:** `src/routes/login/+page.svelte`

```svelte
<script lang="ts">
	import { goto } from '$app/navigation';
	import { Card, CardContent, CardHeader, CardTitle } from '$lib/components/ui/card';
	import { Input } from '$lib/components/ui/input';
	import { Label } from '$lib/components/ui/label';
	import { Button } from '$lib/components/ui/button';

	let password = '';
	let error = '';

	async function handleLogin() {
		const response = await fetch('/api/login', {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ password })
		});

		if (response.ok) {
			goto('/');
		} else {
			error = 'Nieprawidłowe hasło';
		}
	}
</script>

<div class="min-h-screen flex items-center justify-center bg-background">
	<Card class="w-full max-w-md">
		<CardHeader>
			<CardTitle class="text-2xl text-center">💪 Finansowa Forteca</CardTitle>
			<p class="text-center text-muted-foreground">Zaloguj się aby kontynuować</p>
		</CardHeader>
		<CardContent>
			<form on:submit|preventDefault={handleLogin} class="space-y-4">
				<div>
					<Label for="password">Hasło</Label>
					<Input id="password" type="password" bind:value={password} required />
				</div>

				{#if error}
					<p class="text-sm text-red-600">{error}</p>
				{/if}

				<Button type="submit" class="w-full">Zaloguj</Button>
			</form>
		</CardContent>
	</Card>
</div>
```

**File:** `src/routes/api/login/+server.ts`

```typescript
import { json } from '@sveltejs/kit';

const APP_PASSWORD = process.env.APP_PASSWORD || 'changeme';

export async function POST({ request, cookies }) {
	const { password } = await request.json();

	if (password === APP_PASSWORD) {
		cookies.set('finance_session', 'authenticated', {
			path: '/',
			httpOnly: true,
			secure: process.env.NODE_ENV === 'production',
			maxAge: 60 * 60 * 24 * 30 // 30 days
		});

		return json({ success: true });
	}

	return json({ success: false }, { status: 401 });
}
```

### 10.3 Update svelte.config.js

**File:** `svelte.config.js`

```javascript
import adapter from '@sveltejs/adapter-node';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';

export default {
	preprocess: vitePreprocess(),
	kit: {
		adapter: adapter(),
		csrf: {
			checkOrigin: process.env.NODE_ENV === 'production'
		}
	}
};
```

### 10.4 Testing Checklist

- [ ] Run `npm run dev` - app starts without errors
- [ ] Login page works with APP_PASSWORD
- [ ] Dashboard displays net worth chart
- [ ] Dashboard shows correct KPIs (net worth, assets, liabilities)
- [ ] Asset allocation pie chart renders
- [ ] Accounts page lists all accounts
- [ ] Create snapshot form pre-fills with latest values
- [ ] Creating snapshot updates dashboard immediately
- [ ] Goals page shows all goals with progress
- [ ] Responsive on mobile (test with dev tools)
- [ ] Build succeeds: `npm run build`
- [ ] Docker build succeeds: `docker build -t finance-buddy .`
- [ ] Docker container runs and connects to PostgreSQL
- [ ] Migration script imports Excel data successfully

---

## Summary

**Total phases:** 10
**Estimated development time:** 5-7 days (focused work)

**Tech Stack:**

- SvelteKit + TypeScript
- PostgreSQL + Drizzle ORM
- shadcn-svelte + Tailwind
- Apache ECharts
- Docker

**MVP Features:**
✅ Dashboard with net worth chart
✅ Asset allocation visualization
✅ Account management
✅ Snapshot creation (data entry)
✅ Goal tracking
✅ Simple authentication
✅ Docker deployment

**Next Steps After MVP:**

- Investment detail pages (IKE/IKZE/PPK)
- Transaction history
- Installment tracking
- Mortgage payment schedule
- Advanced analytics & forecasting
- Multi-user with separate accounts (Lucia Auth)
- Mobile app (Progressive Web App)
- CSV/PDF export
- Automated bank imports

**References:**

- [SvelteKit Performance](https://dev.to/paulthedev/sveltekit-vs-nextjs-in-2026-why-the-underdog-is-winning-a-developers-deep-dive-155b)
- [ECharts Documentation](https://echarts.apache.org/en/feature.html)
- [Drizzle ORM Docs](https://orm.drizzle.team)
- [shadcn-svelte](https://www.shadcn-svelte.com)
