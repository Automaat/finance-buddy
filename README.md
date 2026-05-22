# 💪 Finansowa Forteca

Self-hosted personal finance web app - beautiful dashboard for tracking net worth, investments, and financial goals.

## Tech Stack

- **Frontend:** SvelteKit 2.60 + Svelte 5 (runes) + TypeScript 6
- **Backend:** Go (chi + pgx) — `backend-go/`
- **Database:** PostgreSQL 18
- **UI:** Tailwind CSS + OpenProps design tokens
- **Charts:** Apache ECharts 6
- **Deployment:** Docker Compose

> Exact versions live in `package.json` and `backend-go/go.mod` -
> those manifests are the single source of truth.

> The backend was originally FastAPI/Python; it was migrated to Go
> endpoint-by-endpoint and the Python backend has been decommissioned.
> `migration/` documents that effort.

## Development

### Prerequisites

- Node.js 24+
- Go 1.26+
- PostgreSQL 18 (or use Docker Compose)
- [mise](https://mise.jdx.dev/) - manages tool versions and runs tasks

### Setup

1. Install frontend dependencies:

```bash
npm install
```

2. Build the backend:

```bash
cd backend-go && go build ./... && cd ..
```

3. Copy environment variables:

```bash
cp .env.example .env
# Edit .env - set POSTGRES_PASSWORD and APP_PASSWORD
```

4. Run all services (frontend, backend, PostgreSQL):

```bash
mise run dev
```

Or run the frontend dev server alone:

```bash
npm run dev
```

### Scripts

- `npm run dev` - Start development server
- `npm run build` - Build for production
- `npm run preview` - Preview production build
- `npm run check` - Type check
- `npm run lint` - Lint code
- `npm run format` - Format code

## Deployment

### Docker Compose

`docker-compose.yml` runs the frontend, backend-go, and PostgreSQL together
from the published `ghcr.io/automaat/finance-buddy-*` images. backend-go
applies the database schema on first start.

```bash
# Required env vars (no defaults - the stack fails fast without them)
export POSTGRES_PASSWORD="a-strong-password"
export APP_PASSWORD="your-secure-password"

docker-compose up -d
```

`ORIGIN`, `CORS_ORIGINS`, and `PUBLIC_API_URL_BROWSER` can be overridden
for deployments behind a custom domain.

## Project Status

Actively used. Dashboard, snapshots, accounts, assets, debts, retirement
metrics, and salary/mortgage/ZUS simulations are all in place.

See [plan.md](./plan.md) for the original implementation roadmap.

## License

Private project
