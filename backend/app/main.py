from contextlib import asynccontextmanager

from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware

from app.api import dashboard
from app.core.config import settings
from app.core.init_db import init_db


@asynccontextmanager
async def lifespan(_app: FastAPI):
    # Startup: Initialize database tables
    init_db()
    yield
    # Shutdown: cleanup if needed


app = FastAPI(title="Finance Buddy API", version="1.0.0", lifespan=lifespan)

app.add_middleware(
    CORSMiddleware,
    allow_origins=settings.cors_origins.split(","),
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# Include routers
app.include_router(dashboard.router)


@app.get("/health")
def health_check() -> dict[str, str]:
    return {"status": "ok"}
