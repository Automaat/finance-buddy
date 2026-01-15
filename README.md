# 💪 Finansowa Forteca

Self-hosted personal finance web app - beautiful dashboard for tracking net worth, investments, and financial goals.

## Tech Stack

- **Framework:** SvelteKit 2.x + TypeScript
- **Database:** PostgreSQL + Drizzle ORM
- **UI:** shadcn-svelte + Tailwind CSS
- **Charts:** Apache ECharts
- **Deployment:** Docker

## Development

### Prerequisites

- Node.js 20+
- Python 3.12+ (for Excel migration script)
- PostgreSQL (or use Docker Compose)
- [mise](https://mise.jdx.dev/) (optional, for tool version management)

### Setup

1. Install dependencies:

```bash
npm install
```

2. Copy environment variables:

```bash
cp .env.example .env
# Edit .env with your PostgreSQL credentials
```

3. Run development server:

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

### Docker

Build and run with Docker:

```bash
docker build -t finance-buddy .
docker run -d \
  --name finance-buddy \
  -p 3000:3000 \
  -e DATABASE_URL="postgresql://user:pass@host:5432/finance" \
  -e ORIGIN="https://finance.yourdomain.com" \
  -e APP_PASSWORD="your-secure-password" \
  finance-buddy
```

Or use Docker Compose:

```bash
docker-compose up -d
```

## Project Status

🚧 **Work in Progress** - Currently in Phase 1 (Project Setup)

See [plan.md](./plan.md) for full implementation roadmap.

## License

Private project
