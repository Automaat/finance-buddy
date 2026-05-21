"""FX rate lookup and NBP fetcher.

Caches NBP table A rates in `fx_rates`. On cache miss, fetches a recent
range from NBP (which only publishes rates on business days) and stores
the most recent rate ≤ requested date.

NBP table A is the standard reference rate published each business day
(no buy/sell spread). It's what tax authorities use, so it matches what
end users encounter on PIT calculations and on bank statements.

PLN always returns 1.0 without DB or network access.
"""

from __future__ import annotations

import logging
from dataclasses import dataclass
from datetime import UTC, date, datetime, timedelta
from decimal import Decimal

import httpx
from sqlalchemy import desc, select
from sqlalchemy.exc import IntegrityError
from sqlalchemy.orm import Session

from app.models import FxRate

logger = logging.getLogger(__name__)

NBP_BASE_URL = "https://api.nbp.pl/api/exchangerates/rates/A"
_LOOKBACK_DAYS = 10
_CACHE_TOLERANCE_DAYS = 7
_HTTP_TIMEOUT_SECONDS = 5.0


@dataclass(frozen=True)
class NbpRate:
    effective_date: date
    rate_pln: Decimal


def _fetch_nbp_range(currency: str, start: date, end: date) -> list[NbpRate]:
    """Fetch all NBP table A rates for currency in [start, end] inclusive.

    Returns empty list on 404 (no rates in window) or network errors.
    Network errors are logged; a transient FX outage shouldn't break the
    salaries page.
    """
    url = f"{NBP_BASE_URL}/{currency}/{start.isoformat()}/{end.isoformat()}/"
    try:
        response = httpx.get(url, params={"format": "json"}, timeout=_HTTP_TIMEOUT_SECONDS)
    except httpx.HTTPError as e:
        logger.warning("NBP fetch failed for %s %s..%s: %s", currency, start, end, e)
        return []

    if response.status_code == 404:
        return []
    if response.status_code != 200:
        logger.warning(
            "NBP fetch failed for %s %s..%s: HTTP %d",
            currency,
            start,
            end,
            response.status_code,
        )
        return []

    try:
        payload = response.json()
    except ValueError as e:
        logger.warning("NBP returned invalid JSON: %s", e)
        return []

    rates: list[NbpRate] = []
    for entry in payload.get("rates", []):
        try:
            rates.append(
                NbpRate(
                    effective_date=date.fromisoformat(entry["effectiveDate"]),
                    rate_pln=Decimal(str(entry["mid"])),
                )
            )
        except (KeyError, ValueError):
            continue
    return rates


def get_fx_rate_to_pln(db: Session, currency: str, on_date: date | None = None) -> Decimal | None:
    """Return PLN-per-unit rate for currency on/before on_date.

    PLN passes through as 1. Returns None only if no rate could be fetched
    or found in cache (e.g. permanent NBP outage on first call).
    """
    code = currency.strip().upper()
    if code == "PLN":
        return Decimal("1")

    if on_date is None:
        on_date = datetime.now(UTC).date()

    # Look for the most recent cached rate on/before the requested date.
    cached = db.execute(
        select(FxRate)
        .where(FxRate.currency == code, FxRate.date <= on_date)
        .order_by(desc(FxRate.date))
        .limit(1)
    ).scalar_one_or_none()

    if cached is not None:
        age_days = (on_date - cached.date).days
        if age_days <= _CACHE_TOLERANCE_DAYS:
            return cached.rate_pln

    # Cache miss or stale — fetch a window ending on the target date.
    start = on_date - timedelta(days=_LOOKBACK_DAYS)
    rates = _fetch_nbp_range(code, start, on_date)
    if not rates:
        return cached.rate_pln if cached is not None else None

    rates.sort(key=lambda r: r.effective_date)
    for rate in rates:
        if rate.effective_date > on_date:
            continue
        _persist_rate(db, code, rate)

    refreshed = db.execute(
        select(FxRate)
        .where(FxRate.currency == code, FxRate.date <= on_date)
        .order_by(desc(FxRate.date))
        .limit(1)
    ).scalar_one_or_none()
    if refreshed is not None:
        return refreshed.rate_pln
    return cached.rate_pln if cached is not None else None


def _persist_rate(db: Session, currency: str, rate: NbpRate) -> None:
    """Insert one NBP rate, ignoring duplicates."""
    record = FxRate(
        date=rate.effective_date,
        currency=currency,
        rate_pln=rate.rate_pln,
    )
    try:
        db.add(record)
        db.commit()
    except IntegrityError:
        db.rollback()


def to_pln(amount: Decimal | float | None, currency: str, rate_pln: Decimal | None) -> float | None:
    """Convert `amount` in `currency` to PLN using `rate_pln`.

    Returns None if amount or rate is missing. PLN amounts pass through
    when rate is None but currency is PLN.
    """
    if amount is None:
        return None
    if currency.upper() == "PLN":
        return float(amount)
    if rate_pln is None:
        return None
    return float(Decimal(str(amount)) * rate_pln)
