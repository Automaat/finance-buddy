"""Tests for the inflation service."""

from datetime import UTC, date, datetime, timedelta
from decimal import Decimal

import pytest

from app.models import CpiIndex
from app.services import inflation


def _seed(db, rows):
    for year, yoy in rows:
        db.add(
            CpiIndex(
                year=year,
                yoy_rate=Decimal(str(yoy)),
                source="test",
                fetched_at=datetime.now(UTC),
            )
        )
    db.commit()


def test_adjust_compounds_full_years(test_db_session):
    # yoy[2021]=110 means +10% during 2021; yoy[2022]=120 means +20% during 2022.
    # Start of 2021 (= end of 2020) -> after 2022's inflation: factor 1.10 * 1.20 = 1.32.
    _seed(
        test_db_session,
        [(2020, "100.0"), (2021, "110.0"), (2022, "120.0")],
    )
    result = inflation.adjust(test_db_session, 1000.0, date(2021, 1, 1), date(2023, 1, 1))
    assert result == pytest.approx(1320.0, rel=1e-6)


def test_adjust_interpolates_within_year(test_db_session):
    _seed(test_db_session, [(2019, "100.0"), (2020, "100.0"), (2021, "120.0")])
    # Mid-2020 (no inflation during 2020) -> mid-2021 (mid of the 20% year):
    # roughly half of the 20% raise = ~+10%.
    result = inflation.adjust(test_db_session, 1000.0, date(2020, 7, 1), date(2021, 7, 1))
    assert 1080 < result < 1120


def test_adjust_clamps_above_latest_known_year(test_db_session):
    _seed(test_db_session, [(2020, "100.0"), (2021, "110.0")])
    # Any date past 2021 clamps to idx[2021] = end of 2021 prices.
    a = inflation.adjust(test_db_session, 1.0, date(2021, 1, 1), date(2099, 1, 1))
    b = inflation.adjust(test_db_session, 1.0, date(2021, 1, 1), date(2050, 1, 1))
    assert a == pytest.approx(b, rel=1e-9)


def test_adjust_clamps_below_earliest_known_year(test_db_session):
    _seed(test_db_session, [(2020, "100.0"), (2021, "110.0")])
    # Any date before 2020 clamps to idx[2020]; date(2020,1,1) also clamps
    # because 2019 is not in the data.
    a = inflation.adjust(test_db_session, 1.0, date(2020, 1, 1), date(2021, 1, 1))
    b = inflation.adjust(test_db_session, 1.0, date(1900, 1, 1), date(2021, 1, 1))
    assert a == pytest.approx(b, rel=1e-9)


def test_adjust_raises_when_empty(test_db_session):
    with pytest.raises(inflation.InflationDataMissingError):
        inflation.adjust(test_db_session, 1000.0, date(2020, 1, 1), date(2022, 1, 1))


def test_cumulative_index_anchored_at_earliest_year():
    yoy = {2020: Decimal("100"), 2021: Decimal("110"), 2022: Decimal("120")}
    idx = inflation._cumulative_index(yoy)
    assert idx[2020] == Decimal("100")
    assert idx[2021] == Decimal("110")
    assert idx[2022] == Decimal("132")


def test_refresh_cpi_sync_upserts(test_db_session):
    rows = [(2020, Decimal("103.4")), (2021, Decimal("105.1"))]
    written = inflation.refresh_cpi_sync(test_db_session, rows)
    assert written == 2

    # Same payload -> no rewrites.
    written_again = inflation.refresh_cpi_sync(test_db_session, rows)
    assert written_again == 0

    # Changed value -> one rewrite.
    rows[0] = (2020, Decimal("103.5"))
    written_changed = inflation.refresh_cpi_sync(test_db_session, rows)
    assert written_changed == 1


def test_needs_refresh_when_empty(test_db_session):
    assert inflation.needs_refresh(test_db_session) is True


def test_needs_refresh_respects_threshold(test_db_session):
    fresh = CpiIndex(
        year=2024,
        yoy_rate=Decimal("103.6"),
        source="test",
        fetched_at=datetime.now(UTC),
    )
    test_db_session.add(fresh)
    test_db_session.commit()
    assert inflation.needs_refresh(test_db_session, stale_after=timedelta(days=7)) is False

    fresh.fetched_at = datetime.now(UTC) - timedelta(days=30)
    test_db_session.commit()
    assert inflation.needs_refresh(test_db_session, stale_after=timedelta(days=7)) is True


def test_cpi_series_returns_sorted_points(test_db_session):
    _seed(
        test_db_session,
        [(2022, "114.4"), (2020, "103.4"), (2021, "105.1")],
    )
    series = inflation.cpi_series(test_db_session)
    assert [row[0] for row in series] == [2020, 2021, 2022]
    # cumulative index for 2022 = 100 * 1.051 * 1.144
    assert series[-1][2] == pytest.approx(Decimal("120.2344"), rel=1e-4)
