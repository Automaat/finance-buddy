"""Polish CPI service.

Pulls annual y/y CPI from GUS BDL (variable 217230 — "Wskaźnik cen towarów
i usług konsumpcyjnych ogółem", aggregateId=1, all of Poland) and exposes
helpers to adjust amounts between arbitrary dates.

Educational notes:
- ``yoy_rate`` from GUS is stored as published (e.g. 114.4 = +14.4% vs prior
  year). A fixed-base cumulative index is derived from these on the fly.
- Within a single year the index is interpolated linearly between Jan 1 of
  year N and Jan 1 of year N+1. The error vs true monthly CPI is well under
  1 percentage point per year, which is below the noise from comparing
  national CPI to HICP anyway.
- Today is rarely available from GUS (lag ~15 days after month end). We fall
  back to the latest year we have and surface the ``as_of_year`` in the API.
"""

import logging
from datetime import UTC, date, datetime, timedelta
from decimal import Decimal

import httpx
from sqlalchemy import select
from sqlalchemy.orm import Session

from app.models import CpiIndex

logger = logging.getLogger(__name__)

GUS_VARIABLE_ID = 217230
GUS_BASE_URL = "https://bdl.stat.gov.pl/api/v1"
GUS_SOURCE_TAG = f"GUS-BDL-{GUS_VARIABLE_ID}"
HTTP_TIMEOUT = httpx.Timeout(15.0, connect=5.0)


class InflationDataMissingError(RuntimeError):
    """Raised when no CPI rows are available for the requested calculation."""


async def fetch_gus_cpi() -> list[tuple[int, Decimal]]:
    """Fetch annual y/y CPI for Poland from GUS BDL.

    Returns a list of ``(year, yoy_rate)`` tuples sorted ascending by year.
    Raises ``httpx.HTTPError`` on transport/HTTP failure.
    """
    url = f"{GUS_BASE_URL}/data/by-variable/{GUS_VARIABLE_ID}"
    params = {"unit-level": 0, "format": "json", "page-size": 100}

    async with httpx.AsyncClient(timeout=HTTP_TIMEOUT) as client:
        response = await client.get(url, params=params)
        response.raise_for_status()
        payload = response.json()

    results = payload.get("results", [])
    if not results:
        return []

    raw_values = results[0].get("values", [])
    parsed: list[tuple[int, Decimal]] = []
    for entry in raw_values:
        year_str = entry.get("year")
        val = entry.get("val")
        if year_str is None or val is None:
            continue
        parsed.append((int(year_str), Decimal(str(val))))
    parsed.sort(key=lambda row: row[0])
    return parsed


def refresh_cpi_sync(db: Session, rows: list[tuple[int, Decimal]]) -> int:
    """Upsert CPI rows into the database. Returns number of rows written."""
    if not rows:
        return 0

    existing = {row.year: row for row in db.execute(select(CpiIndex)).scalars().all()}
    written = 0
    for year, yoy in rows:
        if year in existing:
            current = existing[year]
            if current.yoy_rate != yoy:
                current.yoy_rate = yoy
                current.source = GUS_SOURCE_TAG
                current.fetched_at = datetime.now(UTC)
                written += 1
        else:
            db.add(
                CpiIndex(
                    year=year,
                    yoy_rate=yoy,
                    source=GUS_SOURCE_TAG,
                    fetched_at=datetime.now(UTC),
                )
            )
            written += 1
    db.commit()
    return written


async def refresh_cpi(db: Session) -> int:
    """Fetch latest CPI from GUS and upsert into the database."""
    rows = await fetch_gus_cpi()
    return refresh_cpi_sync(db, rows)


def _load_yoy_map(db: Session) -> dict[int, Decimal]:
    return {row.year: row.yoy_rate for row in db.execute(select(CpiIndex)).scalars().all()}


def _cumulative_index(yoy_by_year: dict[int, Decimal]) -> dict[int, Decimal]:
    """Build a fixed-base index keyed by Jan 1 of each year.

    Anchored at the earliest available year with index = 100. Each subsequent
    year compounds: ``index[y] = index[y-1] * yoy[y] / 100``.
    """
    if not yoy_by_year:
        return {}
    years = sorted(yoy_by_year.keys())
    index_by_year: dict[int, Decimal] = {years[0]: Decimal("100")}
    for prev, year in zip(years, years[1:], strict=False):
        index_by_year[year] = index_by_year[prev] * yoy_by_year[year] / Decimal("100")
    return index_by_year


def _index_at_date(index_by_year: dict[int, Decimal], when: date) -> Decimal:
    """Interpolate the fixed-base index at an arbitrary calendar date.

    Linear between Jan 1 of consecutive years. Before the earliest year, clamp
    to the earliest index; after the latest year, clamp to the latest.
    """
    if not index_by_year:
        raise InflationDataMissingError("CPI table is empty")

    years = sorted(index_by_year.keys())
    if when.year < years[0]:
        return index_by_year[years[0]]
    if when.year >= years[-1]:
        return index_by_year[years[-1]]

    year_start = date(when.year, 1, 1)
    next_year_start = date(when.year + 1, 1, 1)
    span_days = (next_year_start - year_start).days
    elapsed_days = (when - year_start).days
    fraction = Decimal(elapsed_days) / Decimal(span_days)

    start_index = index_by_year[when.year]
    end_index = index_by_year[when.year + 1]
    return start_index + (end_index - start_index) * fraction


def latest_known_year(db: Session) -> int | None:
    """Return the latest year for which we have CPI data, or None."""
    return db.execute(
        select(CpiIndex.year).order_by(CpiIndex.year.desc()).limit(1)
    ).scalar_one_or_none()


def adjust(db: Session, amount: float, from_date: date, to_date: date) -> float:
    """Adjust ``amount`` from purchasing power on ``from_date`` to ``to_date``.

    Returns the equivalent amount today (or on ``to_date``). Raises
    ``InflationDataMissingError`` if no CPI rows exist.
    """
    yoy_map = _load_yoy_map(db)
    index_by_year = _cumulative_index(yoy_map)
    if not index_by_year:
        raise InflationDataMissingError("CPI table is empty")

    from_index = _index_at_date(index_by_year, from_date)
    to_index = _index_at_date(index_by_year, to_date)
    if from_index == 0:
        raise InflationDataMissingError("Source index is zero")
    factor = to_index / from_index
    return float(Decimal(str(amount)) * factor)


def cpi_series(db: Session) -> list[tuple[int, Decimal, Decimal]]:
    """Return ``(year, yoy_rate, cumulative_index)`` per year, ascending."""
    yoy_map = _load_yoy_map(db)
    index_by_year = _cumulative_index(yoy_map)
    return [(year, yoy_map[year], index_by_year[year]) for year in sorted(yoy_map.keys())]


def needs_refresh(db: Session, stale_after: timedelta = timedelta(days=7)) -> bool:
    """True if no CPI rows or the freshest ``fetched_at`` is older than the threshold."""
    latest = db.execute(
        select(CpiIndex.fetched_at).order_by(CpiIndex.fetched_at.desc()).limit(1)
    ).scalar_one_or_none()
    if latest is None:
        return True
    if latest.tzinfo is None:
        latest = latest.replace(tzinfo=UTC)
    return datetime.now(UTC) - latest > stale_after
