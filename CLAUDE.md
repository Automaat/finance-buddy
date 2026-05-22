# Finance-Buddy

Self-hosted personal finance dashboard for tracking net worth, goals, and investments. Replaces "Finansowa Forteca.xlsx" spreadsheet with automated tracking and visualizations.

**Tech Stack:** SvelteKit 2.60, Svelte 5 (runes), TypeScript 6, Go 1.26 (chi + pgx + shopspring/decimal), PostgreSQL 18

**Purpose:** Track Polish personal finance (IKE/IKZE/PPK retirement accounts), monitor net worth trends, manage financial goals, analyze asset allocation.

> The backend was migrated from FastAPI/Python to Go endpoint-by-endpoint; the
> Python backend has been decommissioned. `migration/` documents that effort.

> Version numbers below are indicative. `package.json` and `backend-go/go.mod` are the single source of truth - check them when exact versions matter.

---

## Project Structure

### Directory Layout

```
/backend-go/       # Go backend (chi router + pgx)
  /cmd/api/        # main — server wiring, scheduler, schema bootstrap
  /internal/
    /<domain>/     # one package per endpoint group: store.go (pgx),
                   # handler.go (chi), validation.go, errors.go
    /db/           # pool wiring + schema.sql (baseline schema, applied
                   # on first start) + scheduler-facing helpers
    /server/       # route registration
    /scheduler/    # in-process CPI monthly-refresh job
/backend-bb-tests/ # black-box regression suite (pytest) against backend-go
/migration/        # migration record: cutover docs + archived proxy
/src/              # SvelteKit frontend
  /routes/         # File-based routing (+page.svelte, +page.ts)
  /lib/
    /components/   # UI components
    /utils/        # Calculations, formatting
/static/           # Assets
/build/            # Frontend build output
```

### Key Modules

- **Dashboard** (backend-go/internal/dashboard/) - net worth, allocation, time series; aggregate-backed hot path + raw fallback
- **Snapshots** (backend-go/internal/snapshots/) - monthly net worth snapshots + aggregate recompute
- **Schema bootstrap** (backend-go/internal/db/schema.go) - applies schema.sql to an empty database on startup
- **Frontend Dashboard** (src/routes/+page.svelte) - ECharts visualizations, net worth trends
- **Financial Calculations** (src/lib/utils/calculations.ts) - Net worth, goal progress, month-over-month changes

---

## Development Workflow

### Before Coding

1. **ASK clarifying questions** until 95% confident about requirements
2. **Research existing patterns** in codebase (especially backend-go/internal/* for backend, src/lib/* for frontend utils)
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

- **Formatter:** prettier 3.8
- **Linter:** oxlint 1.65
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

### Svelte 5 Runes (mandatory — no legacy syntax)

- **Props:** `let { foo, bar = default }: Props = $props()` with typed `interface Props`. Pages: `data: PageData` from `./$types`.
- **State:** reassigned UI-driving locals use `$state(...)`. Bare `let` only for non-reactive values (e.g. `bind:this` refs read only in `onMount`).
- **Derived:** `const x = $derived(expr)` for pure computed values; `$derived.by(() => {...})` for multi-statement.
- **Effects:** `$effect(() => {...})` for side effects. Never read+write the same state inside one effect (infinite loop).
- **Events:** attributes not directives — `onclick={fn}`, `onsubmit={fn}`. No modifiers: call `event.preventDefault()` inside the handler.
- **Bindable:** child-written props need `$bindable(default)` (e.g. snapshot modal `bind:` props).
- **One-time prop reads** in `$state(...)` initializers: wrap in `untrack(() => ...)` to avoid `state_referenced_locally`.
- `Modal.svelte` and snapshot modals are reference runes implementations.

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

## Go Conventions (Backend)

### Code Style

- **Formatter:** gofmt
- **Linter:** golangci-lint (`backend-go/.golangci.yml`) + nilaway
- **Naming:** Go standard — MixedCaps, exported = PascalCase
- **No named return values** — enforced by the `nonamedreturns` linter
- **Indentation:** tabs

### Linter Errors

**NEVER** use linter-suppression directives (enforced by a hook). Fix the
root cause — sentinel errors instead of `(nil, nil)`, value returns instead
of pointers for nilaway, etc.

### Backend Patterns

- **One package per endpoint group** under `internal/<domain>/`: `store.go`
  (pgx queries), `handler.go` (chi handlers + wire types), `validation.go`,
  `errors.go` (sentinel errors).
- **Money:** `shopspring/decimal`, parsed from raw JSON bytes (never via
  `float64`) to preserve Numeric precision.
- **Null-vs-missing** PATCH/PUT semantics: `map[string]json.RawMessage`.
- **Schema:** `internal/db/schema.sql` is the frozen baseline, applied to an
  empty database on startup by `ApplySchema`. Future schema changes are made
  directly there.

### Testing

- **Go unit tests:** `go test ./...` in `backend-go/`.
- **Black-box suite:** `backend-bb-tests/` (pytest) drives a real backend-go
  over HTTP against a real Postgres — the regression oracle. conftest builds
  + launches backend-go and seeds fixtures; `golden/` snapshots gate response
  shape. Run with `cd backend-bb-tests && uv run pytest`.

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
- Examine codebase for similar patterns FIRST (especially backend-go/internal/* and src/lib/utils/*)
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

**Backend calculations:**
- Keep aggregation code straightforward — plain loops over small datasets
  (monthly snapshots) beat clever optimization
- Don't over-optimize - readability > performance for small data

**API handlers:**
- Keep simple: validate → call store → return response
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

### Backend (Go)

```bash
cd backend-go
go build ./cmd/api       # Build
go test ./...            # Unit tests
gofmt -w .               # Format
golangci-lint run ./...  # Lint
go run ./cmd/api         # Dev server (port 8000; needs DATABASE_URL)
```

### Black-box tests

```bash
cd backend-bb-tests
uv run pytest            # Builds + launches backend-go, seeds, runs the suite
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
- Net worth trends (time-series aggregations)
- Asset allocation percentages (grouped by category, owner)
- Goal progress + monthly contribution requirements
- Month-over-month changes

### Known Issues & Gotchas

- **Database schema:** `backend-go/internal/db/schema.sql` is the frozen baseline (a pg_dump of the final Alembic head). `ApplySchema` applies it to an empty database on startup and no-ops when tables already exist, so production is never re-DDL'd. Future schema changes are made directly in `schema.sql`.
- **Polish labels:** All UI labels in Polish - no i18n/internationalization needed
- **testcontainers:** the black-box suite spins a real Postgres — slower startup, acceptable trade-off
- **Currency:** All values stored as PLN (Polish Złoty) - no multi-currency support yet
- **CPI scheduler:** `internal/scheduler` refreshes CPI from GUS monthly (16th, 04:00 Europe/Warsaw) + on startup if stale — single-instance only.

### Integration Points

- **PostgreSQL 18:** Database via pgx (connection pooling)
- **ECharts 6.0:** Frontend visualizations (line charts for net worth trends)
- **OpenProps 1.7:** CSS design tokens (var(--size-*), var(--color-*))
- **Docker Compose:** Local development environment (PostgreSQL + backend-go + frontend)
- **mise:** Tool version management (node 24, go 1.26) + task runner
- **GUS BDL API:** monthly CPI source for inflation math

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
