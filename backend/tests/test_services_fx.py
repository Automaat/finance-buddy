"""Tests for FX rate lookup, cache, and NBP integration.

NBP calls are mocked via monkeypatch on _fetch_nbp_range — we never hit
the network in tests.
"""

from datetime import date
from decimal import Decimal

import pytest

from app.models import FxRate
from app.services import fx as fx_module
from app.services.fx import NbpRate, get_fx_rate_to_pln, to_pln


def _stub_nbp(monkeypatch: pytest.MonkeyPatch, rates: list[NbpRate]) -> list[int]:
    """Replace _fetch_nbp_range with a stub. Returns call counter."""
    calls: list[int] = []

    def fake_fetch(_currency: str, _start: date, _end: date) -> list[NbpRate]:
        calls.append(1)
        return rates

    monkeypatch.setattr(fx_module, "_fetch_nbp_range", fake_fetch)
    return calls


def test_pln_returns_one_without_db_call(test_db_session):
    assert get_fx_rate_to_pln(test_db_session, "PLN") == Decimal("1")


def test_first_fetch_persists_rate(test_db_session, monkeypatch):
    _stub_nbp(
        monkeypatch,
        [NbpRate(effective_date=date(2024, 6, 1), rate_pln=Decimal("4.0625"))],
    )

    result = get_fx_rate_to_pln(test_db_session, "USD", on_date=date(2024, 6, 5))

    assert result == Decimal("4.0625")
    stored = (
        test_db_session.query(FxRate).filter_by(currency="USD").order_by(FxRate.date.desc()).first()
    )
    assert stored is not None
    assert stored.rate_pln == Decimal("4.0625")


def test_cache_hit_avoids_fetch(test_db_session, monkeypatch):
    test_db_session.add(FxRate(date=date(2024, 6, 5), currency="USD", rate_pln=Decimal("4.0")))
    test_db_session.commit()

    calls = _stub_nbp(monkeypatch, [])

    result = get_fx_rate_to_pln(test_db_session, "USD", on_date=date(2024, 6, 7))
    assert result == Decimal("4.0")
    assert calls == []  # cache was fresh enough


def test_stale_cache_triggers_refresh(test_db_session, monkeypatch):
    test_db_session.add(FxRate(date=date(2024, 1, 1), currency="USD", rate_pln=Decimal("4.0")))
    test_db_session.commit()

    _stub_nbp(
        monkeypatch,
        [NbpRate(effective_date=date(2024, 6, 1), rate_pln=Decimal("4.5"))],
    )

    result = get_fx_rate_to_pln(test_db_session, "USD", on_date=date(2024, 6, 5))
    assert result == Decimal("4.5")


def test_nbp_outage_returns_stale_cache(test_db_session, monkeypatch):
    """If NBP fails, fall back to whatever's in cache."""
    test_db_session.add(FxRate(date=date(2024, 1, 1), currency="USD", rate_pln=Decimal("4.0")))
    test_db_session.commit()

    _stub_nbp(monkeypatch, [])  # no rates returned

    result = get_fx_rate_to_pln(test_db_session, "USD", on_date=date(2024, 6, 5))
    assert result == Decimal("4.0")


def test_nbp_outage_and_empty_cache_returns_none(test_db_session, monkeypatch):
    _stub_nbp(monkeypatch, [])
    result = get_fx_rate_to_pln(test_db_session, "USD", on_date=date(2024, 6, 5))
    assert result is None


def test_picks_latest_rate_le_target_date(test_db_session, monkeypatch):
    _stub_nbp(
        monkeypatch,
        [
            NbpRate(effective_date=date(2024, 5, 30), rate_pln=Decimal("4.0")),
            NbpRate(effective_date=date(2024, 5, 31), rate_pln=Decimal("4.1")),
            NbpRate(effective_date=date(2024, 6, 3), rate_pln=Decimal("4.2")),
        ],
    )

    result = get_fx_rate_to_pln(test_db_session, "USD", on_date=date(2024, 6, 1))
    # 2024-06-1 is a Saturday in NBP terms; latest ≤ that is May 31
    assert result == Decimal("4.1")


def test_to_pln_helper():
    assert to_pln(100, "PLN", None) == 100.0
    assert to_pln(None, "USD", Decimal("4")) is None
    assert to_pln(100, "USD", None) is None
    assert to_pln(100, "USD", Decimal("4.0625")) == 406.25
