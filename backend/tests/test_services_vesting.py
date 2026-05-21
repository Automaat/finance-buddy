"""Tests for the equity vesting calculator."""

from datetime import date

from app.core.enums import VestingFrequency
from app.services.vesting import (
    VestingSchedule,
    months_between,
    vested_shares_at,
    vesting_progress_pct,
)


class TestMonthsBetween:
    def test_same_day_anniversary(self):
        assert months_between(date(2024, 1, 15), date(2025, 1, 15)) == 12

    def test_one_day_before_anniversary(self):
        # Anniversary semantics: day 14 of same month doesn't fully complete month 12
        assert months_between(date(2024, 1, 15), date(2025, 1, 14)) == 11

    def test_one_day_after_anniversary(self):
        assert months_between(date(2024, 1, 15), date(2025, 1, 16)) == 12

    def test_zero_for_same_date(self):
        assert months_between(date(2024, 1, 15), date(2024, 1, 15)) == 0

    def test_negative_for_past(self):
        assert months_between(date(2024, 6, 1), date(2024, 5, 1)) == -1


def _standard_4_1_monthly(total_shares: int = 4800) -> VestingSchedule:
    """Common preset: 4yr, 1yr cliff, monthly. 4800 chosen for nice math."""
    return VestingSchedule(
        total_shares=total_shares,
        vest_start_date=date(2024, 1, 1),
        vest_cliff_months=12,
        vest_total_months=48,
        vest_frequency=VestingFrequency.MONTHLY,
    )


class TestStandardMonthlyVesting:
    def test_before_start_returns_zero(self):
        sched = _standard_4_1_monthly()
        assert vested_shares_at(sched, date(2023, 12, 31)) == 0

    def test_before_cliff_returns_zero(self):
        sched = _standard_4_1_monthly()
        assert vested_shares_at(sched, date(2024, 12, 31)) == 0

    def test_at_cliff_returns_25pct(self):
        sched = _standard_4_1_monthly(4800)
        # 12 months → 12/48 = 25% of 4800 = 1200
        assert vested_shares_at(sched, date(2025, 1, 1)) == 1200

    def test_one_month_after_cliff(self):
        sched = _standard_4_1_monthly(4800)
        # 13 months → 13/48 of 4800 = 1300
        assert vested_shares_at(sched, date(2025, 2, 1)) == 1300

    def test_fully_vested_at_total_months(self):
        sched = _standard_4_1_monthly(4800)
        assert vested_shares_at(sched, date(2028, 1, 1)) == 4800

    def test_caps_at_total_after_full_period(self):
        sched = _standard_4_1_monthly(4800)
        assert vested_shares_at(sched, date(2030, 1, 1)) == 4800


class TestQuarterlyVesting:
    def test_quarterly_post_cliff(self):
        sched = VestingSchedule(
            total_shares=4800,
            vest_start_date=date(2024, 1, 1),
            vest_cliff_months=12,
            vest_total_months=48,
            vest_frequency=VestingFrequency.QUARTERLY,
        )
        # At month 12: 25%
        assert vested_shares_at(sched, date(2025, 1, 1)) == 1200
        # At month 13-14: still 1200 (waiting next quarter)
        assert vested_shares_at(sched, date(2025, 2, 1)) == 1200
        assert vested_shares_at(sched, date(2025, 3, 31)) == 1200
        # At month 15: extra quarter = 15/48 of 4800 = 1500
        assert vested_shares_at(sched, date(2025, 4, 1)) == 1500


class TestYearlyVesting:
    def test_yearly_steps(self):
        sched = VestingSchedule(
            total_shares=400,
            vest_start_date=date(2024, 1, 1),
            vest_cliff_months=12,
            vest_total_months=48,
            vest_frequency=VestingFrequency.YEARLY,
        )
        assert vested_shares_at(sched, date(2025, 1, 1)) == 100
        assert vested_shares_at(sched, date(2025, 6, 1)) == 100
        assert vested_shares_at(sched, date(2026, 1, 1)) == 200
        assert vested_shares_at(sched, date(2028, 1, 1)) == 400


class TestNoCliffMonthly:
    def test_vests_from_first_month(self):
        sched = VestingSchedule(
            total_shares=4800,
            vest_start_date=date(2024, 1, 1),
            vest_cliff_months=0,
            vest_total_months=48,
            vest_frequency=VestingFrequency.MONTHLY,
        )
        assert vested_shares_at(sched, date(2024, 1, 1)) == 0  # 0 months elapsed
        assert vested_shares_at(sched, date(2024, 2, 1)) == 100  # 1/48 of 4800
        assert vested_shares_at(sched, date(2025, 1, 1)) == 1200


class TestCustomSchedule:
    def test_back_loaded_10_20_30_40(self):
        sched = VestingSchedule(
            total_shares=1000,
            vest_start_date=date(2024, 1, 1),
            vest_cliff_months=12,
            vest_total_months=48,
            vest_frequency=VestingFrequency.YEARLY,
            vest_custom_schedule=[
                {"month": 12, "pct": 10},
                {"month": 24, "pct": 20},
                {"month": 36, "pct": 30},
                {"month": 48, "pct": 40},
            ],
        )
        assert vested_shares_at(sched, date(2024, 12, 1)) == 0
        assert vested_shares_at(sched, date(2025, 1, 1)) == 100  # 10%
        assert vested_shares_at(sched, date(2026, 1, 1)) == 300  # 10+20
        assert vested_shares_at(sched, date(2027, 1, 1)) == 600  # 10+20+30
        assert vested_shares_at(sched, date(2028, 1, 1)) == 1000

    def test_custom_caps_at_total_shares(self):
        sched = VestingSchedule(
            total_shares=100,
            vest_start_date=date(2024, 1, 1),
            vest_cliff_months=0,
            vest_total_months=12,
            vest_frequency=VestingFrequency.YEARLY,
            vest_custom_schedule=[
                {"month": 6, "pct": 60},
                {"month": 12, "pct": 60},  # over-100% by mistake
            ],
        )
        assert vested_shares_at(sched, date(2025, 1, 1)) == 100


class TestDoubleTrigger:
    def test_returns_zero_without_liquidity_event(self):
        sched = VestingSchedule(
            total_shares=1000,
            vest_start_date=date(2020, 1, 1),
            vest_cliff_months=12,
            vest_total_months=48,
            vest_frequency=VestingFrequency.MONTHLY,
            requires_liquidity_event=True,
            liquidity_event_date=None,
        )
        # Time-based would say fully vested, but no liquidity event yet
        assert vested_shares_at(sched, date(2026, 1, 1)) == 0

    def test_returns_zero_before_liquidity_event(self):
        sched = VestingSchedule(
            total_shares=1000,
            vest_start_date=date(2020, 1, 1),
            vest_cliff_months=12,
            vest_total_months=48,
            vest_frequency=VestingFrequency.MONTHLY,
            requires_liquidity_event=True,
            liquidity_event_date=date(2030, 6, 1),
        )
        assert vested_shares_at(sched, date(2026, 1, 1)) == 0

    def test_returns_time_vested_after_event(self):
        sched = VestingSchedule(
            total_shares=4800,
            vest_start_date=date(2024, 1, 1),
            vest_cliff_months=12,
            vest_total_months=48,
            vest_frequency=VestingFrequency.MONTHLY,
            requires_liquidity_event=True,
            liquidity_event_date=date(2025, 6, 1),
        )
        # After event, time-based math applies: 17 months → 1700
        assert vested_shares_at(sched, date(2025, 6, 1)) == 1700


class TestProgressPercent:
    def test_progress_at_25pct(self):
        sched = _standard_4_1_monthly(4800)
        assert vesting_progress_pct(sched, date(2025, 1, 1)) == 25.0

    def test_progress_zero_for_empty_grant(self):
        sched = VestingSchedule(
            total_shares=0,
            vest_start_date=date(2024, 1, 1),
            vest_cliff_months=0,
            vest_total_months=12,
            vest_frequency=VestingFrequency.MONTHLY,
        )
        assert vesting_progress_pct(sched, date(2025, 1, 1)) == 0.0
