"""Tests for the equity grants service."""

from datetime import date

import pytest
from fastapi import HTTPException
from pydantic import ValidationError

from app.schemas.equity_grants import EquityGrantCreate, EquityGrantUpdate
from app.services.equity_grants import (
    create_equity_grant,
    delete_equity_grant,
    get_all_equity_grants,
    get_equity_grant,
    update_equity_grant,
)
from tests.factories import create_test_equity_grant


def _valid_create(**overrides) -> EquityGrantCreate:
    base = {
        "grant_date": date(2024, 1, 1),
        "type": "rsu",
        "company": "Co X",
        "owner": "Marcin",
        "total_shares": 4800,
        "currency": "USD",
        "vest_start_date": date(2024, 1, 1),
        "vest_cliff_months": 12,
        "vest_total_months": 48,
        "vest_frequency": "monthly",
    }
    base.update(overrides)
    return EquityGrantCreate(**base)


def test_create_rsu_grant_success(test_db_session):
    result = create_equity_grant(test_db_session, _valid_create())

    assert result.id is not None
    assert result.type == "rsu"
    assert result.total_shares == 4800
    assert result.strike_price is None
    assert result.is_active is True
    assert 0 <= result.vesting_progress_pct <= 100


def test_create_option_requires_strike():
    with pytest.raises(ValidationError):
        _valid_create(type="option", strike_price=None)


def test_create_option_with_strike(test_db_session):
    result = create_equity_grant(test_db_session, _valid_create(type="option", strike_price=5.5))
    assert result.strike_price == 5.5


def test_cliff_cannot_exceed_total():
    with pytest.raises(ValidationError):
        _valid_create(vest_cliff_months=60, vest_total_months=48)


def test_get_all_returns_active_only(test_db_session):
    create_test_equity_grant(test_db_session)
    create_test_equity_grant(test_db_session, is_active=False)

    result = get_all_equity_grants(test_db_session)
    assert result.total_count == 1


def test_get_all_filters(test_db_session):
    create_test_equity_grant(test_db_session, owner="Marcin", company="A")
    create_test_equity_grant(test_db_session, owner="Ewa", company="A")
    create_test_equity_grant(test_db_session, owner="Marcin", company="B")

    by_owner = get_all_equity_grants(test_db_session, owner="Marcin")
    assert by_owner.total_count == 2

    by_company = get_all_equity_grants(test_db_session, company="A")
    assert by_company.total_count == 2


def test_available_companies_distinct(test_db_session):
    create_test_equity_grant(test_db_session, company="A")
    create_test_equity_grant(test_db_session, company="B")
    create_test_equity_grant(test_db_session, company="A")

    result = get_all_equity_grants(test_db_session)
    assert result.available_companies == ["A", "B"]


def test_get_equity_grant_not_found(test_db_session):
    with pytest.raises(HTTPException) as exc:
        get_equity_grant(test_db_session, 9999)
    assert exc.value.status_code == 404


def test_get_soft_deleted_returns_404(test_db_session):
    grant = create_test_equity_grant(test_db_session, is_active=False)
    with pytest.raises(HTTPException) as exc:
        get_equity_grant(test_db_session, grant.id)
    assert exc.value.status_code == 404


def test_update_grant(test_db_session):
    grant = create_test_equity_grant(test_db_session)

    result = update_equity_grant(
        test_db_session,
        grant.id,
        EquityGrantUpdate(total_shares=5000, notes="adjusted"),
    )
    assert result.total_shares == 5000
    assert result.notes == "adjusted"


def test_update_cliff_overflow_rejected(test_db_session):
    grant = create_test_equity_grant(test_db_session)

    with pytest.raises(HTTPException) as exc:
        update_equity_grant(
            test_db_session,
            grant.id,
            EquityGrantUpdate(vest_cliff_months=100),
        )
    assert exc.value.status_code == 422


def test_update_switch_to_option_without_strike_rejected(test_db_session):
    """RSU → option requires a strike price, even via PATCH."""
    grant = create_test_equity_grant(test_db_session, grant_type="rsu", strike_price=None)

    with pytest.raises(HTTPException) as exc:
        update_equity_grant(test_db_session, grant.id, EquityGrantUpdate(type="option"))
    assert exc.value.status_code == 422


def test_update_switch_to_rsu_clears_strike(test_db_session):
    """Option → RSU should drop the now-irrelevant strike price."""
    grant = create_test_equity_grant(test_db_session, grant_type="option", strike_price=5.0)

    result = update_equity_grant(test_db_session, grant.id, EquityGrantUpdate(type="rsu"))
    assert result.type == "rsu"
    assert result.strike_price is None


def test_delete_soft_deletes(test_db_session):
    grant = create_test_equity_grant(test_db_session)
    delete_equity_grant(test_db_session, grant.id)

    with pytest.raises(HTTPException) as exc:
        get_equity_grant(test_db_session, grant.id)
    assert exc.value.status_code == 404


def test_response_includes_computed_vested(test_db_session):
    """Response should include vested_shares_today and progress."""
    # Grant from 2020 with 4yr/1yr/monthly — fully vested by now
    grant = create_test_equity_grant(
        test_db_session,
        vest_start_date=date(2020, 1, 1),
        vest_cliff_months=12,
        vest_total_months=48,
        total_shares=1000,
    )

    result = get_equity_grant(test_db_session, grant.id)
    assert result.vested_shares_today == 1000
    assert result.vesting_progress_pct == 100.0


def test_double_trigger_returns_zero_until_event(test_db_session):
    grant = create_test_equity_grant(
        test_db_session,
        vest_start_date=date(2020, 1, 1),
        requires_liquidity_event=True,
        liquidity_event_date=None,
    )
    result = get_equity_grant(test_db_session, grant.id)
    assert result.vested_shares_today == 0
