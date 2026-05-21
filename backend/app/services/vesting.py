"""Equity vesting math. Pure functions, no DB access.

Supports standard schemes (cliff + linear at chosen frequency) and arbitrary
custom schedules expressed as a list of `{month, pct}` events.

Custom schedule semantics:
    [{"month": 12, "pct": 25}, {"month": 24, "pct": 25}, ...]
    Each entry is an *increment* of total shares vesting at that absolute
    month index after vest_start. Summing all pct values should typically
    equal 100; the calculator just sums those whose month <= elapsed.

Double-trigger RSUs: when `requires_liquidity_event` is True, vesting math
runs as usual but the result is gated to 0 until `liquidity_event_date` is
reached. After the trigger date, the time-based vested count is returned.
"""

from dataclasses import dataclass
from datetime import date
from typing import Any

from app.core.enums import VestingFrequency


@dataclass(frozen=True)
class VestingSchedule:
    total_shares: int
    vest_start_date: date
    vest_cliff_months: int
    vest_total_months: int
    vest_frequency: VestingFrequency
    vest_custom_schedule: list[dict[str, Any]] | None = None
    requires_liquidity_event: bool = False
    liquidity_event_date: date | None = None


_FREQ_MONTHS: dict[VestingFrequency, int] = {
    VestingFrequency.MONTHLY: 1,
    VestingFrequency.QUARTERLY: 3,
    VestingFrequency.YEARLY: 12,
}


def months_between(start: date, end: date) -> int:
    """Whole months elapsed from `start` to `end`. Negative if end < start.

    Uses anniversary semantics: if `end.day < start.day`, the current month
    hasn't completed yet.
    """
    months = (end.year - start.year) * 12 + (end.month - start.month)
    if end.day < start.day:
        months -= 1
    return months


def vested_shares_at(schedule: VestingSchedule, on_date: date) -> int:
    """Compute vested share count at a point in time.

    Returns 0 before vest_start or before cliff. For double-trigger grants,
    returns 0 until liquidity_event_date is set and reached.
    """
    if schedule.requires_liquidity_event:
        if schedule.liquidity_event_date is None:
            return 0
        if on_date < schedule.liquidity_event_date:
            return 0

    if on_date < schedule.vest_start_date:
        return 0

    elapsed_months = months_between(schedule.vest_start_date, on_date)
    if elapsed_months < schedule.vest_cliff_months:
        return 0

    capped = min(elapsed_months, schedule.vest_total_months)

    if schedule.vest_custom_schedule:
        total_pct = sum(
            float(event.get("pct", 0))
            for event in schedule.vest_custom_schedule
            if int(event.get("month", 0)) <= capped
        )
        return min(
            schedule.total_shares,
            int(schedule.total_shares * total_pct / 100),
        )

    if schedule.vest_total_months <= 0:
        return schedule.total_shares

    freq_months = _FREQ_MONTHS[schedule.vest_frequency]

    # Months that have *vested* — only events at multiples of freq_months
    # starting from the cliff anniversary contribute.
    months_after_cliff = capped - schedule.vest_cliff_months
    extra_periods = months_after_cliff // freq_months
    vesting_month_count = schedule.vest_cliff_months + extra_periods * freq_months
    vesting_month_count = min(vesting_month_count, schedule.vest_total_months)

    return int(schedule.total_shares * vesting_month_count / schedule.vest_total_months)


def vesting_progress_pct(schedule: VestingSchedule, on_date: date) -> float:
    """Convenience: vested fraction as a percentage (0–100)."""
    if schedule.total_shares <= 0:
        return 0.0
    vested = vested_shares_at(schedule, on_date)
    return vested / schedule.total_shares * 100
