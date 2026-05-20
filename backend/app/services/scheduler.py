"""Background scheduler for periodic CPI refreshes."""

import logging
from collections.abc import AsyncIterator
from contextlib import asynccontextmanager

from apscheduler.schedulers.asyncio import AsyncIOScheduler
from apscheduler.triggers.cron import CronTrigger

from app.core.database import SessionLocal
from app.services import inflation

logger = logging.getLogger(__name__)
_scheduler: AsyncIOScheduler | None = None


async def _refresh_job() -> None:
    db = SessionLocal()
    try:
        written = await inflation.refresh_cpi(db)
        logger.info("CPI refresh complete: %s rows written", written)
    except Exception:
        logger.exception("CPI refresh failed")
    finally:
        db.close()


async def _startup_refresh_if_stale() -> None:
    """Refresh on boot only if data is missing or stale (>7 days old)."""
    db = SessionLocal()
    try:
        stale = inflation.needs_refresh(db)
    except Exception:
        logger.exception("CPI staleness check failed; skipping startup refresh")
        db.close()
        return
    db.close()

    if stale:
        logger.info("CPI data missing or stale; refreshing on startup")
        await _refresh_job()
    else:
        logger.info("CPI data fresh; skipping startup refresh")


@asynccontextmanager
async def scheduler_lifespan() -> AsyncIterator[None]:
    """Start the APScheduler on entry, shut it down on exit."""
    global _scheduler
    await _startup_refresh_if_stale()

    _scheduler = AsyncIOScheduler(timezone="Europe/Warsaw")
    # GUS publishes monthly CPI around the 15th. Run on the 16th at 04:00.
    _scheduler.add_job(
        _refresh_job,
        CronTrigger(day=16, hour=4, minute=0, timezone="Europe/Warsaw"),
        id="cpi_monthly_refresh",
        replace_existing=True,
    )
    _scheduler.start()
    logger.info("CPI scheduler started")
    try:
        yield
    finally:
        if _scheduler is not None:
            _scheduler.shutdown(wait=False)
            _scheduler = None
            logger.info("CPI scheduler stopped")
