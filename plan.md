# Finance Buddy - Implementation Plan

Web app to replace "Finansowa Forteca.xlsx" - beautiful, self-hosted personal finance tracker.

## Tech Stack

**Backend:**
- **Framework:** FastAPI (Python 3.12+)
- **Data Processing:** pandas (Excel import, financial calculations, aggregations)
- **Database:** PostgreSQL + SQLAlchemy ORM
- **Validation:** Pydantic v2
- **Package Manager:** uv
- **Linter:** ruff

**Frontend:**
- **Framework:** SvelteKit 2.x + TypeScript
- **UI:** shadcn-svelte + Tailwind CSS
- **Charts:** Apache ECharts
- **API Client:** native fetch

**Deployment:**
- Docker Compose (FastAPI + SvelteKit)
- PostgreSQL (existing instance)
- Multi-user: shared login for MVP

**Why this stack:**

- FastAPI: Modern async Python, auto OpenAPI docs, great DX
- pandas: Built for financial data - Excel I/O, time series, aggregations, pivot tables
- SvelteKit: 50% less JS than Next.js, better dashboard performance ([source](https://dev.to/paulthedev/sveltekit-vs-nextjs-in-2026-why-the-underdog-is-winning-a-developers-deep-dive-155b))
- ECharts: Superior for financial time-series, handles millions of data points ([source](https://www.metabase.com/blog/best-open-source-chart-library))
- uv: 10-100x faster than pip, built-in venv management
- ruff: 10-100x faster than pylint/flake8, auto-fix

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

### 1.1 Initialize Python Backend

```bash
# Create backend directory
mkdir backend
cd backend

# Initialize uv project
uv init --name finance-buddy-api --python 3.12

# Install dependencies
uv add fastapi uvicorn[standard] sqlalchemy psycopg2-binary pandas openpyxl pydantic-settings python-dotenv

# Install dev dependencies
uv add --dev ruff pytest httpx

# Create ruff config
cat > ruff.toml <<EOF
line-length = 100
target-version = "py312"

[lint]
select = ["E", "F", "I", "N", "W", "UP", "B", "A", "C4", "DTZ", "T20", "RET", "SIM", "ARG"]
ignore = []

[lint.isort]
known-first-party = ["app"]
EOF

# Create project structure
mkdir -p app/{api,core,models,schemas,services}
touch app/__init__.py
touch app/{api,core,models,schemas,services}/__init__.py
touch app/main.py
```

### 1.2 Initialize SvelteKit Frontend

```bash
cd ..
mkdir frontend
cd frontend

# Initialize SvelteKit
npm create svelte@latest .
# Choose: Skeleton project, TypeScript, ESLint, Prettier
npm install

# UI Components
npm install -D tailwindcss postcss autoprefixer
npx tailwindcss init -p
npx shadcn-svelte@latest init

# Charts
npm install echarts

# Forms & Validation
npm install bits-ui formsnap sveltekit-superforms zod

# Date handling
npm install @internationalized/date
```

### 1.3 Configure Environment

**Backend `.env`** (`backend/.env`):

```env
DATABASE_URL=postgresql://user:pass@your-postgres-host:5432/finance
APP_PASSWORD=shared-secret-for-mvp
CORS_ORIGINS=http://localhost:5173,http://localhost:3000
```

**Frontend `.env`** (`frontend/.env`):

```env
PUBLIC_API_URL=http://localhost:8000
```

### 1.4 Setup Tailwind (Frontend)

Update `frontend/tailwind.config.js`:

```js
export default {
	content: ['./src/**/*.{html,js,svelte,ts}'],
	theme: { extend: {} },
	plugins: []
};
```

Add to `frontend/src/app.css`:

```css
@tailwind base;
@tailwind components;
@tailwind utilities;
```

### 1.5 Create FastAPI App Structure

**File:** `backend/app/core/config.py`

```python
from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    database_url: str
    app_password: str
    cors_origins: str = "http://localhost:5173"

    model_config = SettingsConfigDict(env_file=".env")


settings = Settings()
```

**File:** `backend/app/main.py`

```python
from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware
from app.core.config import settings

app = FastAPI(title="Finance Buddy API", version="1.0.0")

# CORS
app.add_middleware(
    CORSMiddleware,
    allow_origins=settings.cors_origins.split(","),
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)


@app.get("/health")
async def health():
    return {"status": "ok"}
```

**Test it:**

```bash
cd backend
uv run uvicorn app.main:app --reload
# Visit http://localhost:8000/docs
```

---

## Phase 2: Database Schema

### 2.1 Create SQLAlchemy Models

**File:** `backend/app/models/account.py`

```python
from sqlalchemy import Boolean, Column, Integer, String, Numeric, DateTime
from sqlalchemy.sql import func
from app.core.database import Base


class Account(Base):
    __tablename__ = "accounts"

    id = Column(Integer, primary_key=True, index=True)
    name = Column(String, nullable=False)
    type = Column(String, nullable=False)  # 'asset' | 'liability'
    category = Column(String, nullable=False)  # 'bank', 'ike', 'ikze', etc.
    owner = Column(String)  # 'Marcin' | 'Ewa' | 'Shared'
    currency = Column(String, default="PLN")
    is_active = Column(Boolean, default=True)
    created_at = Column(DateTime(timezone=True), server_default=func.now())
```

**File:** `backend/app/models/snapshot.py`

```python
from sqlalchemy import Column, Integer, String, Date, DateTime, Numeric, ForeignKey
from sqlalchemy.sql import func
from app.core.database import Base


class Snapshot(Base):
    __tablename__ = "snapshots"

    id = Column(Integer, primary_key=True, index=True)
    date = Column(Date, nullable=False, unique=True, index=True)
    notes = Column(String)
    created_at = Column(DateTime(timezone=True), server_default=func.now())


class SnapshotValue(Base):
    __tablename__ = "snapshot_values"

    id = Column(Integer, primary_key=True, index=True)
    snapshot_id = Column(Integer, ForeignKey("snapshots.id", ondelete="CASCADE"), nullable=False)
    account_id = Column(Integer, ForeignKey("accounts.id", ondelete="CASCADE"), nullable=False)
    value = Column(Numeric(15, 2), nullable=False)
```

**File:** `backend/app/models/goal.py`

```python
from sqlalchemy import Boolean, Column, Integer, String, Numeric, Date, DateTime
from sqlalchemy.sql import func
from app.core.database import Base


class Goal(Base):
    __tablename__ = "goals"

    id = Column(Integer, primary_key=True, index=True)
    name = Column(String, nullable=False)
    target_amount = Column(Numeric(15, 2), nullable=False)
    target_date = Column(Date, nullable=False)
    current_amount = Column(Numeric(15, 2), default=0)
    monthly_contribution = Column(Numeric(15, 2))
    is_completed = Column(Boolean, default=False)
    created_at = Column(DateTime(timezone=True), server_default=func.now())
```

**File:** `backend/app/models/__init__.py`

```python
from app.models.account import Account
from app.models.snapshot import Snapshot, SnapshotValue
from app.models.goal import Goal

__all__ = ["Account", "Snapshot", "SnapshotValue", "Goal"]
```

### 2.2 Create Database Connection

**File:** `backend/app/core/database.py`

```python
from sqlalchemy import create_engine
from sqlalchemy.ext.declarative import declarative_base
from sqlalchemy.orm import sessionmaker
from app.core.config import settings

engine = create_engine(settings.database_url)
SessionLocal = sessionmaker(autocommit=False, autoflush=False, bind=engine)

Base = declarative_base()


def get_db():
    db = SessionLocal()
    try:
        yield db
    finally:
        db.close()
```

### 2.3 Create Migration Script

**File:** `backend/app/core/init_db.py`

```python
from app.core.database import engine, Base
from app.models import Account, Snapshot, SnapshotValue, Goal


def init_db():
    """Create all tables"""
    Base.metadata.create_all(bind=engine)
    print("✓ Database tables created")


if __name__ == "__main__":
    init_db()
```

### 2.4 Run Migrations

```bash
cd backend
uv run python -m app.core.init_db
```

---

## Phase 3: Excel Data Migration (Learn pandas!)

### 3.1 Create Migration Script with pandas

**File:** `backend/scripts/migrate_excel.py`

```python
"""
Excel to PostgreSQL migration using pandas.
Great for learning pandas operations:
- read_excel(): Load Excel sheets
- DataFrame.columns: Access column names
- DataFrame.iterrows(): Iterate rows
- pd.isna(), pd.notna(): Handle missing values
- Boolean indexing: df[condition]
- String operations: .str.lower(), .str.contains()
"""
import os
from pathlib import Path

import pandas as pd
from dotenv import load_dotenv
from sqlalchemy import create_engine
from sqlalchemy.orm import Session

# Add parent directory to path to import models
import sys
sys.path.insert(0, str(Path(__file__).parent.parent))

from app.models import Account, Snapshot, SnapshotValue, Goal
from app.core.database import engine

load_dotenv()

EXCEL_FILE = "Finansowa Forteca.xlsx"


def determine_type(account_name: str) -> str:
    """Determine if account is asset or liability using pandas-style logic"""
    liabilities = ["raty 0", "hipoteka"]
    return "liability" if any(l in account_name.lower() for l in liabilities) else "asset"


def determine_category(account_name: str) -> str:
    """Map account name to category - could use pandas.Series.map() for bulk operations"""
    name_lower = account_name.lower()
    category_map = {
        "ike": "ike",
        "ikze": "ikze",
        "ppk": "ppk",
        "konto": "bank",
        "oszczednosc": "bank",
        "mieszkanie": "real_estate",
        "działka": "real_estate",
        "samochod": "vehicle",
        "obligacje": "bonds",
        "akcje": "stocks",
        "hipoteka": "mortgage",
        "raty": "installment",
    }

    for key, value in category_map.items():
        if key in name_lower:
            return value
    return "other"


def determine_owner(account_name: str) -> str:
    """Determine owner - demonstrates pattern matching in pandas workflows"""
    name_lower = account_name.lower()
    if "marcin" in name_lower:
        return "Marcin"
    if "ewa" in name_lower:
        return "Ewa"
    return "Shared"


def migrate():
    """Main migration - demonstrates pandas Excel I/O and data transformation"""

    # pandas: read_excel() - Load multiple sheets
    print(f"📖 Reading {EXCEL_FILE}...")
    df_net_worth = pd.read_excel(EXCEL_FILE, sheet_name="wartosc_netto")
    df_goals = pd.read_excel(EXCEL_FILE, sheet_name="cele_krotkoterminowe")

    print(f"  Net worth sheet: {df_net_worth.shape[0]} rows, {df_net_worth.shape[1]} columns")
    print(f"  Goals sheet: {df_goals.shape[0]} rows, {df_goals.shape[1]} columns")

    # SQLAlchemy session
    db = Session(engine)

    # 1. Create accounts from DataFrame columns
    print("\n💰 Creating accounts...")
    accounts_map = {}
    skip_columns = ["Data", "ROR", "wartość netto"]

    # pandas: .columns returns Index of column names
    account_columns = [col for col in df_net_worth.columns if col not in skip_columns]

    for column in account_columns:
        # pandas: .iloc[0] - first row value check
        if pd.notna(df_net_worth[column].iloc[0]):
            account = Account(
                name=column,
                type=determine_type(column),
                category=determine_category(column),
                owner=determine_owner(column),
            )
            db.add(account)
            db.flush()  # Get ID before commit
            accounts_map[column] = account.id
            print(f"  ✓ {column} → {account.category} ({account.owner})")

    db.commit()
    print(f"  Created {len(accounts_map)} accounts")

    # 2. Create snapshots from DataFrame rows
    print("\n📸 Creating snapshots...")
    snapshot_count = 0

    # pandas: .iterrows() - iterate over DataFrame rows
    for idx, row in df_net_worth.iterrows():
        date = row["Data"]

        # pandas: pd.isna() - check for NaN/None values
        if pd.isna(date):
            continue

        # Create snapshot
        snapshot = Snapshot(date=date)
        db.add(snapshot)
        db.flush()

        # pandas: Access row values by column name
        for account_name, account_id in accounts_map.items():
            value = row[account_name]

            # pandas: pd.notna() - check value exists
            if pd.notna(value) and value != 0:
                snapshot_value = SnapshotValue(
                    snapshot_id=snapshot.id, account_id=account_id, value=float(value)
                )
                db.add(snapshot_value)

        snapshot_count += 1
        if snapshot_count % 10 == 0:
            print(f"  Processed {snapshot_count} snapshots...")
            db.commit()  # Commit in batches

    db.commit()
    print(f"  ✓ Created {snapshot_count} snapshots")

    # 3. Import goals
    print("\n🎯 Creating goals...")
    goal_count = 0

    # pandas: Could use .dropna(subset=['Cel']) to filter rows first
    for _, row in df_goals.iterrows():
        if pd.notna(row["Cel"]):
            goal = Goal(
                name=row["Cel"],
                target_amount=float(row["Kwota"]) if pd.notna(row["Kwota"]) else 0,
                target_date=row["Termin"] if pd.notna(row["Termin"]) else None,
                current_amount=float(row["Ile już mamy"]) if pd.notna(row["Ile już mamy"]) else 0,
                monthly_contribution=float(row["Ile miesięcznie musimy odkładać"])
                if pd.notna(row["Ile miesięcznie musimy odkładać"])
                else None,
            )
            db.add(goal)
            goal_count += 1

    db.commit()
    db.close()

    print(f"  ✓ Created {goal_count} goals")
    print("\n✅ Migration completed successfully!")
    print(f"  📊 {len(accounts_map)} accounts")
    print(f"  📸 {snapshot_count} snapshots")
    print(f"  🎯 {goal_count} goals")


if __name__ == "__main__":
    migrate()
```

### 3.2 Run Migration

```bash
cd backend

# Place Excel file in backend directory
# cp ~/path/to/"Finansowa Forteca.xlsx" .

# Run migration
uv run python scripts/migrate_excel.py
```

**pandas Learning Notes:**

Key operations demonstrated:
- `pd.read_excel()` - Load Excel sheets
- `df.shape` - Get (rows, columns)
- `df.columns` - Access column names
- `df.iterrows()` - Iterate rows
- `df.iloc[0]` - Position-based indexing
- `pd.isna()` / `pd.notna()` - Handle missing values
- `row['column']` - Access values

**Next pandas features to explore in Phase 5+:**
- `df.groupby()` - Aggregate data (total by category/owner)
- `df.pivot_table()` - Create summary tables
- `df.resample()` - Time series aggregation (monthly/yearly)
- `df.merge()` - Join DataFrames
- `df.query()` - SQL-like filtering

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

## Phase 5: Dashboard API (Learn pandas aggregations!)

### 5.1 Create Pydantic Schemas

**File:** `backend/app/schemas/dashboard.py`

```python
from datetime import date
from decimal import Decimal

from pydantic import BaseModel


class NetWorthPoint(BaseModel):
    date: date
    value: float


class AllocationItem(BaseModel):
    category: str
    owner: str | None
    value: float


class DashboardResponse(BaseModel):
    net_worth_history: list[NetWorthPoint]
    current_net_worth: float
    change_vs_last_month: float
    total_assets: float
    total_liabilities: float
    allocation: list[AllocationItem]
```

### 5.2 Create Dashboard Service with pandas

**File:** `backend/app/services/dashboard.py`

```python
"""
Dashboard service using pandas for financial calculations.
Demonstrates: groupby, pivot, merge, aggregations, time series
"""
from datetime import date
from decimal import Decimal

import pandas as pd
from sqlalchemy import select
from sqlalchemy.orm import Session

from app.models import Account, Snapshot, SnapshotValue
from app.schemas.dashboard import AllocationItem, DashboardResponse, NetWorthPoint


def get_dashboard_data(db: Session) -> DashboardResponse:
    """
    Calculate dashboard metrics using pandas.

    pandas features used:
    - pd.DataFrame(): Create from query results
    - df.merge(): Join DataFrames (like SQL JOIN)
    - df.groupby(): Aggregate data
    - df.pivot_table(): Reshape data
    - df.sort_values(): Order data
    """

    # Fetch all data needed
    accounts_query = select(Account).where(Account.is_active == True)  # noqa: E712
    accounts_df = pd.read_sql(accounts_query, db.connection())

    snapshots_query = select(Snapshot).order_by(Snapshot.date)
    snapshots_df = pd.read_sql(snapshots_query, db.connection())

    values_query = select(SnapshotValue)
    values_df = pd.read_sql(values_query, db.connection())

    # pandas: merge() - Join snapshot values with accounts
    # Similar to SQL: SELECT * FROM snapshot_values JOIN accounts
    df = values_df.merge(accounts_df, left_on="account_id", right_on="id", suffixes=("", "_account"))
    df = df.merge(snapshots_df, left_on="snapshot_id", right_on="id", suffixes=("", "_snapshot"))

    # Calculate net worth per snapshot
    # pandas: groupby() + apply() - Custom aggregation per group
    def calculate_net_worth(group):
        assets = group[group["type"] == "asset"]["value"].sum()
        liabilities = group[group["type"] == "liability"]["value"].sum()
        return float(assets - liabilities)

    # pandas: groupby() groups rows, then we aggregate
    net_worth_by_date = (
        df.groupby("date").apply(calculate_net_worth, include_groups=False).reset_index()
    )
    net_worth_by_date.columns = ["date", "net_worth"]

    # pandas: sort_values() - Order by date
    net_worth_by_date = net_worth_by_date.sort_values("date")

    # Convert to response format
    net_worth_history = [
        NetWorthPoint(date=row["date"], value=row["net_worth"])
        for _, row in net_worth_by_date.iterrows()
    ]

    # Current metrics (latest snapshot)
    if len(net_worth_by_date) > 0:
        current_net_worth = float(net_worth_by_date.iloc[-1]["net_worth"])
        last_month_net_worth = (
            float(net_worth_by_date.iloc[-2]["net_worth"]) if len(net_worth_by_date) > 1 else 0
        )
    else:
        current_net_worth = 0
        last_month_net_worth = 0

    # Latest snapshot data for current totals
    latest_snapshot = snapshots_df.iloc[-1] if len(snapshots_df) > 0 else None

    if latest_snapshot is not None:
        # pandas: Boolean indexing - Filter rows
        latest_df = df[df["snapshot_id"] == latest_snapshot["id"]]

        # pandas: groupby() + sum() - Aggregate by type
        totals_by_type = latest_df.groupby("type")["value"].sum()

        total_assets = float(totals_by_type.get("asset", 0))
        total_liabilities = float(totals_by_type.get("liability", 0))

        # Asset allocation
        # pandas: Query filter + groupby multiple columns
        assets_df = latest_df[latest_df["type"] == "asset"]

        # pandas: groupby() with multiple columns
        allocation_df = assets_df.groupby(["category", "owner"])["value"].sum().reset_index()

        allocation = [
            AllocationItem(
                category=row["category"], owner=row["owner"], value=float(row["value"])
            )
            for _, row in allocation_df.iterrows()
        ]
    else:
        total_assets = 0
        total_liabilities = 0
        allocation = []

    return DashboardResponse(
        net_worth_history=net_worth_history,
        current_net_worth=current_net_worth,
        change_vs_last_month=current_net_worth - last_month_net_worth,
        total_assets=total_assets,
        total_liabilities=total_liabilities,
        allocation=allocation,
    )
```

### 5.3 Create Dashboard API Endpoint

**File:** `backend/app/api/dashboard.py`

```python
from fastapi import APIRouter, Depends
from sqlalchemy.orm import Session

from app.core.database import get_db
from app.schemas.dashboard import DashboardResponse
from app.services.dashboard import get_dashboard_data

router = APIRouter(prefix="/api/dashboard", tags=["dashboard"])


@router.get("", response_model=DashboardResponse)
def get_dashboard(db: Session = Depends(get_db)):
    """Get dashboard data with net worth history and allocation"""
    return get_dashboard_data(db)
```

**Update:** `backend/app/main.py`

```python
from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware
from app.core.config import settings
from app.api import dashboard  # Add this

app = FastAPI(title="Finance Buddy API", version="1.0.0")

# CORS
app.add_middleware(
    CORSMiddleware,
    allow_origins=settings.cors_origins.split(","),
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# Include routers
app.include_router(dashboard.router)  # Add this


@app.get("/health")
async def health():
    return {"status": "ok"}
```

**Create:** `backend/app/api/__init__.py` (empty file)

**Create:** `backend/app/schemas/__init__.py` (empty file)

**Create:** `backend/app/services/__init__.py` (empty file)

### 5.4 Update Frontend to Fetch from API

**File:** `frontend/src/routes/+page.ts`

```typescript
import { env } from '$env/dynamic/public';

export async function load({ fetch }) {
	const response = await fetch(`${env.PUBLIC_API_URL}/api/dashboard`);
	const data = await response.json();
	return data;
}
```

### 5.5 Dashboard Component

**File:** `frontend/src/routes/+page.svelte`

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

**Backend:**
- [ ] FastAPI starts: `cd backend && uv run uvicorn app.main:app --reload`
- [ ] Swagger docs available at http://localhost:8000/docs
- [ ] ruff linter passes: `cd backend && uv run ruff check .`
- [ ] ruff formatter passes: `cd backend && uv run ruff format --check .`
- [ ] Tests pass: `cd backend && uv run pytest`
- [ ] Coverage ≥80%: Check coverage report
- [ ] Migration imports Excel data successfully
- [ ] Dashboard API returns correct data

**Frontend:**
- [ ] Run `cd frontend && npm run dev` - app starts
- [ ] Connects to backend API successfully
- [ ] Login page works with APP_PASSWORD
- [ ] Dashboard displays net worth chart
- [ ] Dashboard shows correct KPIs
- [ ] Asset allocation pie chart renders
- [ ] Accounts page lists all accounts
- [ ] Create snapshot form pre-fills with latest values
- [ ] Creating snapshot updates dashboard immediately
- [ ] Goals page shows all goals with progress
- [ ] Responsive on mobile (test with dev tools)
- [ ] Build succeeds: `npm run build`

**Integration:**
- [ ] Docker build succeeds: `docker build -t finance-buddy .`
- [ ] Docker containers run and connect to PostgreSQL
- [ ] Both frontend and backend work together

---

## Summary

**Total phases:** 10

**Tech Stack:**

**Backend:**
- FastAPI + Python 3.12
- pandas (data processing & financial calculations)
- PostgreSQL + SQLAlchemy
- Pydantic v2
- uv (package manager)
- ruff (linter/formatter)
- pytest + coverage

**Frontend:**
- SvelteKit + TypeScript
- shadcn-svelte + Tailwind CSS
- Apache ECharts
- Vitest + @testing-library/svelte

**DevOps:**
- Docker Compose
- GitHub Actions CI
- codecov (80% coverage target)

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
