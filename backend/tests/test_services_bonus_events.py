"""Tests for bonus event service layer."""

from datetime import date

import pytest
from fastapi import HTTPException

from app.schemas.bonus_events import BonusEventCreate, BonusEventUpdate
from app.services.bonus_events import (
    create_bonus_event,
    delete_bonus_event,
    get_all_bonus_events,
    get_bonus_event,
    update_bonus_event,
)
from tests.factories import create_test_bonus_event


def test_create_bonus_event_success(test_db_session):
    data = BonusEventCreate(
        date=date(2024, 12, 15),
        amount=20000.0,
        currency="PLN",
        type="annual",
        company="Test Co",
        owner="Marcin",
        contract_type="UOP",
        notes="Q4 bonus",
    )

    result = create_bonus_event(test_db_session, data)

    assert result.id is not None
    assert result.amount == 20000.0
    assert result.currency == "PLN"
    assert result.type == "annual"
    assert result.notes == "Q4 bonus"
    assert result.is_active is True


def test_get_all_bonus_events_returns_active_only(test_db_session):
    create_test_bonus_event(test_db_session, bonus_date=date(2024, 12, 15))
    create_test_bonus_event(
        test_db_session,
        bonus_date=date(2024, 6, 1),
        amount=5000.0,
        is_active=False,
    )

    result = get_all_bonus_events(test_db_session)

    assert result.total_count == 1
    assert result.bonus_events[0].amount == 20000.0


def test_get_all_bonus_events_orders_by_date_desc(test_db_session):
    create_test_bonus_event(test_db_session, bonus_date=date(2024, 1, 15), amount=1000.0)
    create_test_bonus_event(test_db_session, bonus_date=date(2024, 12, 15), amount=2000.0)
    create_test_bonus_event(test_db_session, bonus_date=date(2024, 6, 15), amount=3000.0)

    result = get_all_bonus_events(test_db_session)

    assert [e.amount for e in result.bonus_events] == [2000.0, 3000.0, 1000.0]


def test_get_all_bonus_events_filters(test_db_session):
    create_test_bonus_event(
        test_db_session,
        bonus_date=date(2024, 3, 1),
        owner="Marcin",
        company="A",
    )
    create_test_bonus_event(
        test_db_session,
        bonus_date=date(2024, 6, 1),
        owner="Ewa",
        company="B",
    )
    create_test_bonus_event(
        test_db_session,
        bonus_date=date(2024, 9, 1),
        owner="Marcin",
        company="B",
    )

    by_owner = get_all_bonus_events(test_db_session, owner="Marcin")
    assert by_owner.total_count == 2

    by_company = get_all_bonus_events(test_db_session, company="B")
    assert by_company.total_count == 2

    by_range = get_all_bonus_events(
        test_db_session, date_from=date(2024, 5, 1), date_to=date(2024, 7, 1)
    )
    assert by_range.total_count == 1
    assert by_range.bonus_events[0].owner == "Ewa"


def test_get_bonus_event_success(test_db_session):
    created = create_test_bonus_event(test_db_session)

    result = get_bonus_event(test_db_session, created.id)

    assert result.id == created.id
    assert result.amount == 20000.0


def test_get_bonus_event_not_found(test_db_session):
    with pytest.raises(HTTPException) as exc:
        get_bonus_event(test_db_session, 9999)
    assert exc.value.status_code == 404


def test_get_bonus_event_soft_deleted_returns_404(test_db_session):
    created = create_test_bonus_event(test_db_session, is_active=False)

    with pytest.raises(HTTPException) as exc:
        get_bonus_event(test_db_session, created.id)
    assert exc.value.status_code == 404


def test_update_bonus_event_partial(test_db_session):
    created = create_test_bonus_event(test_db_session)

    result = update_bonus_event(
        test_db_session,
        created.id,
        BonusEventUpdate(amount=33000.0, notes="updated"),
    )

    assert result.amount == 33000.0
    assert result.notes == "updated"
    assert result.currency == "PLN"
    assert result.type == "annual"


def test_update_bonus_event_not_found(test_db_session):
    with pytest.raises(HTTPException) as exc:
        update_bonus_event(test_db_session, 9999, BonusEventUpdate(amount=100.0))
    assert exc.value.status_code == 404


def test_delete_bonus_event_soft_deletes(test_db_session):
    created = create_test_bonus_event(test_db_session)

    delete_bonus_event(test_db_session, created.id)

    with pytest.raises(HTTPException) as exc:
        get_bonus_event(test_db_session, created.id)
    assert exc.value.status_code == 404


def test_available_companies_distinct(test_db_session):
    create_test_bonus_event(test_db_session, bonus_date=date(2024, 1, 1), company="A")
    create_test_bonus_event(test_db_session, bonus_date=date(2024, 2, 1), company="B")
    create_test_bonus_event(test_db_session, bonus_date=date(2024, 3, 1), company="A")

    result = get_all_bonus_events(test_db_session)

    assert result.available_companies == ["A", "B"]
