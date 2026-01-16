# Finance-Buddy

Self-hosted personal finance dashboard for tracking net worth, goals, and investments. Replaces "Finansowa Forteca.xlsx" spreadsheet with automated tracking and visualizations.

**Tech Stack:** SvelteKit 2.49, TypeScript 5.9, FastAPI, Python 3.14, SQLAlchemy 2.0, pandas 2.3, PostgreSQL 18

**Purpose:** Track Polish personal finance (IKE/IKZE/PPK retirement accounts), monitor net worth trends, manage financial goals, analyze asset allocation.

---

## Project Structure

### Directory Layout

```
/backend/          # FastAPI + SQLAlchemy + pandas
  /app/
    /api/          # Routers (dashboard, accounts, snapshots)
    /models/       # SQLAlchemy ORM (Account, Snapshot, Goal)
    /schemas/      # Pydantic DTOs
    /services/     # Business logic + pandas calculations
    /core/         # Config, database, init_db
  /tests/          # Pytest tests (test_*.py)
/src/              # SvelteKit frontend
  /routes/         # File-based routing (+page.svelte, +page.ts)
  /lib/
    /components/   # UI components
    /utils/        # Calculations, formatting
/static/           # Assets
/build/            # Frontend build output
```

### Key Modules

- **Dashboard API** (backend/app/api/dashboard.py) - Net worth endpoint
- **Dashboard Service** (backend/app/services/dashboard.py) - pandas aggregations for time-series net worth, asset allocation
- **Snapshot Creation** (backend/app/services/snapshots.py) - Monthly net worth snapshots with account values
- **Frontend Dashboard** (src/routes/+page.svelte) - ECharts visualizations, net worth trends
- **Financial Calculations** (src/lib/utils/calculations.ts) - Net worth, goal progress, month-over-month changes

---

## Development Workflow

### Before Coding

1. **ASK clarifying questions** until 95% confident about requirements
2. **Research existing patterns** in codebase (especially backend/app/services/* for pandas, src/lib/* for frontend utils)
3. **Create plan** and get approval before implementing
4. **Work incrementally** - one focused task at a time

### Recommended Workflow

**Explore → Plan → Code → Commit**

- Use **Plan Mode** (Shift+Tab twice) for complex tasks
- Search codebase for similar implementations before proposing new patterns
- Propose plan with alternatives when multiple approaches exist
- **Do NOT code until plan is confirmed**

---

## TypeScript Conventions (Frontend)

### Code Style

- **Formatter:** prettier 3.8.0
- **Linter:** oxlint 1.39.0
- **Naming:** camelCase (variables/functions), PascalCase (components), kebab-case (CSS classes)
- **Type safety:** Strict TypeScript, no `any` type
- **Indentation:** Tabs (Svelte convention)

### Linter Errors

**ALWAYS:**
- Attempt to fix linter errors properly
- Research solutions online if unclear how to fix
- Fix root cause, not symptoms

**NEVER:**
- Use skip/disable directives (e.g., `// eslint-disable`, `// @ts-ignore`)
- Ignore linter warnings
- Work around linter errors

**If stuck:**
- Try fixing the error
- Research online for proper solution
- If still unclear after research, ASK what to do (don't skip/disable)

### Frontend Patterns

- Prefer `const` over `let`, never `var`
- Type all function parameters and return values
- SvelteKit load() functions for data fetching (see src/routes/+page.ts)
- Component-local state (no global state management)
- OpenProps CSS variables: `var(--size-4)`, `var(--color-text-1)`
- ECharts for visualizations (see src/routes/+page.svelte)

### Error Handling

```typescript
try {
  const response = await fetch(`/api/endpoint`);
  if (!response.ok) {
    throw new Error(`API call failed: ${response.statusText}`);
  }
  const data = await response.json();
  return data;
} catch (err) {
  console.error('Failed to fetch data:', err);
  if (err instanceof Error) {
    error = err.message;
  }
}
```

### Testing (Vitest)

- **Framework:** Vitest 4.0
- **Style:** Jasmine-style (describe/it blocks)
- **Location:** Co-located with source (calculations.ts → calculations.test.ts)
- **Coverage threshold:** 80%

**Test Pattern:**

```typescript
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { calculateNetWorth } from './calculations';

describe('calculateNetWorth', () => {
  it('calculates positive net worth', () => {
    expect(calculateNetWorth(100000, 30000)).toBe(70000);
  });

  it('handles zero values', () => {
    expect(calculateNetWorth(0, 0)).toBe(0);
  });
});

// For date tests, use fake timers
describe('calculateMonthsRemaining', () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('calculates future months', () => {
    vi.setSystemTime(new Date('2024-01-15'));
    const target = new Date('2024-06-15');
    expect(calculateMonthsRemaining(target)).toBe(5);
  });
});
```

---

## Python Conventions (Backend)

### Code Style

- **Formatter:** ruff format
- **Linter:** ruff check (line-length: 100, target py3.12+)
- **Type checker:** pyrefly 0.48.1
- **Naming:** snake_case (functions/variables), PascalCase (classes)
- **Type hints:** Required for all functions
- **Indentation:** 4 spaces

### Linter Errors

Same rules as TypeScript - **NEVER** use `# noqa` or `# type: ignore` comments. Fix issues properly or ASK.

### Backend Patterns

- **Layered architecture:** models → schemas → api → services
- **Dependency injection:** FastAPI `Depends()` for database sessions (see backend/app/api/*)
- **SQLAlchemy 2.0:** Mapped types, declarative base (see backend/app/models/snapshot.py)
- **Pydantic v2:** Field validators for validation (see backend/app/schemas/snapshots.py)
- **pandas:** Educational comments for complex operations (see backend/app/services/dashboard.py)

### Error Handling

```python
from sqlalchemy.exc import IntegrityError
from fastapi import HTTPException

try:
    db.add(snapshot)
    db.commit()
    db.refresh(snapshot)
except IntegrityError as e:
    db.rollback()
    raise HTTPException(
        status_code=400,
        detail=f"Snapshot for date {data.date} already exists"
    ) from e
```

### Testing (pytest)

- **Framework:** pytest 9.0
- **Style:** Function-based tests with fixtures
- **Location:** backend/tests/test_*.py
- **Fixtures:** conftest.py (test_db_session)
- **Integration tests:** testcontainers[postgres] for PostgreSQL tests
- **Coverage threshold:** 80%

**Test Pattern:**

```python
import pytest
from fastapi import HTTPException
from app.models import Account
from app.services.snapshots import create_snapshot
from app.schemas.snapshots import SnapshotCreate, SnapshotValueInput

def test_create_snapshot_success(test_db_session):
    """Test creating a snapshot with account values"""
    # Create test accounts
    account = Account(
        name="Test Bank", type="asset", category="bank",
        owner="Test", currency="PLN"
    )
    test_db_session.add(account)
    test_db_session.commit()

    # Create snapshot
    data = SnapshotCreate(
        date=date(2024, 1, 31),
        notes="Test snapshot",
        values=[SnapshotValueInput(account_id=account.id, value=10000.50)]
    )

    result = create_snapshot(test_db_session, data)

    assert result.date == date(2024, 1, 31)
    assert len(result.values) == 1

def test_create_snapshot_invalid_account(test_db_session):
    """Test creating snapshot with invalid account ID fails"""
    data = SnapshotCreate(
        date=date(2024, 1, 31),
        notes="Test",
        values=[SnapshotValueInput(account_id=999, value=1000)]
    )

    with pytest.raises(HTTPException) as exc_info:
        create_snapshot(test_db_session, data)

    assert exc_info.value.status_code == 404
    assert "not found" in exc_info.value.detail
```

---

## Simplicity Principles

### Anti-Patterns to AVOID

❌ **NEVER:**
- Over-engineer simple features
- Add unnecessary abstractions
- Create helpers/utilities for one-time operations
- Design for hypothetical future requirements
- Build complex multi-layer architectures for simple tasks
- Generate long code blocks or entire files at once (>100 lines)
- Use placeholder comments like `// ... rest of code ...` or `# TODO`
- Add linter suppressions (`# noqa`, `// eslint-disable`, `// @ts-ignore`)

### Enforcement Rules

✅ **ALWAYS:**
- Choose the simplest practical solution
- Three similar lines > premature abstraction
- Only introduce complexity if clearly justified
- Make minimal, surgical changes
- Examine codebase for similar patterns FIRST (especially backend/app/services/* and src/lib/utils/*)
- Reuse existing components/utilities/logic
- Consistency > perfection

### Complexity Check

**Before implementing, ask:**
1. Can this be simpler?
2. Am I adding abstractions needed NOW (not future)?
3. Does similar code exist I can reuse?
4. Is this the minimal change to achieve the goal?

**If unsure:** STOP and ask for approval before proceeding.

### Pattern Drift Threats

**pandas calculations:**
- Keep educational comments for complex operations (see backend/app/services/dashboard.py)
- Prefer simple iterrows() for small datasets over complex vectorization
- Don't over-optimize - readability > performance for small data

**API handlers:**
- Keep simple: validate → call service → return response
- No middleware unless explicitly needed
- Don't add caching/retry logic without clear requirements

---

## Code Generation Rules

### ALWAYS

- Show **complete code** (no placeholders)
- **Incremental changes** - small focused steps (20-50 lines)
- **Surgical, minimal** changes only
- **Follow existing patterns** found in codebase
- **Test after each step**

### NEVER

- Generate entire long files at once
- Generate >100 lines in single response
- Make big changes in single step
- Modify code unrelated to the task
- Assume requirements without asking
- Add features beyond what's requested
- Use placeholders like `// ... rest of code ...` or `# TODO: implement`

### Incremental Development Process

**Break changes into steps:**
1. Define interfaces/types (schemas, TypeScript interfaces)
2. Implement core logic (minimal)
3. Add error handling
4. Add tests
5. Iterate

**Each step:** Review, approve, then proceed to next.

---

## Common Commands

### Development (mise)

```bash
mise run dev       # Start all services (docker-compose.dev.yml)
mise run backend   # Backend only (port 8000)
mise run frontend  # Frontend only (port 5173)
mise run down      # Stop all services
```

### Frontend

```bash
npm run dev              # Vite dev server (port 5173)
npm run build            # Production build
npm run preview          # Preview production build
npm run test             # Vitest
npm run test:coverage    # Tests with coverage
npm run lint             # oxlint + prettier check
npm run format           # prettier --write
npm run check            # svelte-check (types)
```

### Backend

```bash
cd backend
uv sync                  # Install dependencies
uv run pytest            # Run tests
uv run pytest --cov      # Tests with coverage
uv run ruff check .      # Lint
uv run ruff format .     # Format
uv run pyrefly check .   # Type check
uv run uvicorn app.main:app --reload  # Dev server (port 8000)
```

### Docker

```bash
docker-compose up -d                      # Production
docker-compose -f docker-compose.dev.yml up --build  # Development
docker-compose -f docker-compose.dev.yml down        # Stop
```

### Git Conventions

- **Branch naming:** `feat/*`, `fix/*`, `chore/*`
- **Commit format:** Conventional commits with scope + PR reference
  ```
  fix(migration): include ROR column and normalize liability values (#43)
  feat: Phase 7 - Snapshot Creation API and UI (#41)
  chore(deps): update SvelteKit to 2.49
  ```
- **Commit signing:** Use `-s -S` flags (required by global CLAUDE.md)
- **CI gates:** Linting, type checks, tests, 80% coverage

---

## Project-Specific Context

### Domain Knowledge (Polish Personal Finance)

**Asset Types:**
- **Bank accounts:** konto (checking), oszczędności (savings)
- **Retirement:** IKE, IKZE, PPK (Polish retirement schemes)
- **Investments:** akcje (stocks), obligacje (bonds), fundusze (funds)
- **Real estate:** mieszkanie (apartment), działka (land)
- **Other:** samochód (vehicle), elektronika (electronics)

**Liability Types:**
- **Mortgage:** hipoteka
- **Installments:** raty, raty 0% (0% installments)

**Core Metrics:**
- **Net Worth (Wartość Netto):** Assets - Liabilities
- **Snapshots:** Monthly net worth with all account values
- **Goals:** Short-term financial targets with progress tracking
- **Asset Allocation:** By category (bank, IKE, IKZE, real estate) and owner (Marcin, Ewa, Shared)
- **ROR:** Rate of Return on investments

**Financial Calculations:**
- Net worth trends (time-series pandas aggregations)
- Asset allocation percentages (groupby category, owner)
- Goal progress + monthly contribution requirements
- Month-over-month changes

### Known Issues & Gotchas

- **Database migrations:** Auto-initialized from SQLAlchemy models (backend/app/core/init_db.py) - no Alembic, keep simple until needed
- **Polish labels:** All UI labels in Polish - no i18n/internationalization needed
- **testcontainers:** PostgreSQL integration tests slower due to container startup (acceptable trade-off)
- **pandas iterrows():** Acceptable for small datasets (monthly snapshots) - readability > performance
- **Educational comments:** Encourage for complex pandas operations (see backend/app/services/dashboard.py lines 36-42, 44-46)
- **Currency:** All values stored as PLN (Polish Złoty) - no multi-currency support yet

### Integration Points

- **PostgreSQL 18:** Database via SQLAlchemy 2.0 (connection pooling, async support via asyncio.to_thread)
- **ECharts 6.0:** Frontend visualizations (line charts for net worth trends)
- **OpenProps 1.7:** CSS design tokens (var(--size-*), var(--color-*))
- **Docker Compose:** Local development environment (PostgreSQL + backend + frontend)
- **mise:** Tool version management (node 24, python 3.14, uv) + task runner

---

## Notes for Maintaining This File

**Do:**
- Update when architecture changes
- Add patterns as they emerge from real code
- Review during PR reviews
- Keep under 500 lines

**Don't:**
- Add generic programming advice
- Include outdated patterns
- Let it grow beyond 500 lines without splitting
- Forget to update when conventions change
