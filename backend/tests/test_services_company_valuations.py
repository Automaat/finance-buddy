"""Tests for company valuation service + paper value computation."""

from datetime import date

import pytest
from fastapi import HTTPException
from pydantic import ValidationError

from app.schemas.company_valuations import (
    CompanyValuationCreate,
    CompanyValuationUpdate,
)
from app.services.company_valuations import (
    create_company_valuation,
    delete_company_valuation,
    get_all_company_valuations,
    get_company_valuation,
    get_latest_valuation,
    update_company_valuation,
)
from app.services.equity_grants import get_equity_grant
from tests.factories import create_test_company_valuation, create_test_equity_grant


def _valid_create(**overrides) -> CompanyValuationCreate:
    base = {
        "company": "Co X",
        "date": date(2024, 6, 1),
        "currency": "USD",
        "fmv_per_share": 10.0,
        "source": "409a",
    }
    base.update(overrides)
    return CompanyValuationCreate(**base)


def test_create_valuation_success(test_db_session):
    result = create_company_valuation(test_db_session, _valid_create())
    assert result.id is not None
    assert result.fmv_per_share == 10.0
    assert result.source == "409a"


def test_fmv_low_cannot_exceed_base():
    with pytest.raises(ValidationError):
        _valid_create(fmv_low=12.0)


def test_fmv_high_cannot_be_below_base():
    with pytest.raises(ValidationError):
        _valid_create(fmv_high=8.0)


def test_get_all_orders_by_date_desc(test_db_session):
    create_test_company_valuation(test_db_session, valuation_date=date(2024, 1, 1))
    create_test_company_valuation(test_db_session, valuation_date=date(2024, 12, 1))
    create_test_company_valuation(test_db_session, valuation_date=date(2024, 6, 1))

    result = get_all_company_valuations(test_db_session)
    dates = [v.date for v in result.company_valuations]
    assert dates == [date(2024, 12, 1), date(2024, 6, 1), date(2024, 1, 1)]


def test_filter_by_company(test_db_session):
    create_test_company_valuation(test_db_session, company="A")
    create_test_company_valuation(test_db_session, company="B")

    result = get_all_company_valuations(test_db_session, company="A")
    assert result.total_count == 1


def test_latest_valuation_lookup(test_db_session):
    create_test_company_valuation(
        test_db_session, company="X", valuation_date=date(2024, 1, 1), fmv_per_share=5
    )
    create_test_company_valuation(
        test_db_session, company="X", valuation_date=date(2024, 12, 1), fmv_per_share=10
    )

    latest = get_latest_valuation(test_db_session, "X")
    assert latest is not None
    assert float(latest.fmv_per_share) == 10


def test_latest_with_as_of_date(test_db_session):
    create_test_company_valuation(
        test_db_session, company="X", valuation_date=date(2024, 1, 1), fmv_per_share=5
    )
    create_test_company_valuation(
        test_db_session, company="X", valuation_date=date(2024, 12, 1), fmv_per_share=10
    )

    latest = get_latest_valuation(test_db_session, "X", on_date=date(2024, 6, 1))
    assert latest is not None
    assert float(latest.fmv_per_share) == 5


def test_update_valuation(test_db_session):
    valuation = create_test_company_valuation(test_db_session)
    result = update_company_valuation(
        test_db_session, valuation.id, CompanyValuationUpdate(fmv_per_share=15.0)
    )
    assert result.fmv_per_share == 15.0


def test_update_range_integrity(test_db_session):
    valuation = create_test_company_valuation(test_db_session, fmv_per_share=10, fmv_high=12)
    with pytest.raises(HTTPException) as exc:
        update_company_valuation(
            test_db_session,
            valuation.id,
            CompanyValuationUpdate(fmv_per_share=15),  # now > fmv_high
        )
    assert exc.value.status_code == 422


def test_delete_soft_deletes(test_db_session):
    valuation = create_test_company_valuation(test_db_session)
    delete_company_valuation(test_db_session, valuation.id)

    with pytest.raises(HTTPException) as exc:
        get_company_valuation(test_db_session, valuation.id)
    assert exc.value.status_code == 404


class TestPaperValueOnGrant:
    def test_rsu_paper_value_uses_fmv(self, test_db_session):
        grant = create_test_equity_grant(
            test_db_session,
            company="Co X",
            currency="USD",
            grant_type="rsu",
            total_shares=1000,
            vest_start_date=date(2020, 1, 1),  # fully vested by now
            vest_cliff_months=12,
            vest_total_months=48,
        )
        create_test_company_valuation(
            test_db_session, company="Co X", currency="USD", fmv_per_share=10
        )

        result = get_equity_grant(test_db_session, grant.id)
        assert result.vested_shares_today == 1000
        assert result.paper_value_base == 10_000.0
        assert result.paper_value_currency == "USD"

    def test_option_uses_intrinsic_value(self, test_db_session):
        grant = create_test_equity_grant(
            test_db_session,
            company="Co X",
            currency="USD",
            grant_type="option",
            strike_price=4.0,
            total_shares=1000,
            vest_start_date=date(2020, 1, 1),
            vest_cliff_months=12,
            vest_total_months=48,
        )
        create_test_company_valuation(
            test_db_session, company="Co X", currency="USD", fmv_per_share=10
        )

        result = get_equity_grant(test_db_session, grant.id)
        assert result.paper_value_base == 6_000.0  # (10 - 4) * 1000

    def test_underwater_option_zero(self, test_db_session):
        grant = create_test_equity_grant(
            test_db_session,
            company="Co X",
            currency="USD",
            grant_type="option",
            strike_price=20.0,
            total_shares=1000,
            vest_start_date=date(2020, 1, 1),
            vest_cliff_months=12,
            vest_total_months=48,
        )
        create_test_company_valuation(
            test_db_session, company="Co X", currency="USD", fmv_per_share=10
        )

        result = get_equity_grant(test_db_session, grant.id)
        assert result.paper_value_base == 0.0

    def test_range_values(self, test_db_session):
        grant = create_test_equity_grant(
            test_db_session,
            company="Co X",
            currency="USD",
            grant_type="rsu",
            total_shares=1000,
            vest_start_date=date(2020, 1, 1),
            vest_cliff_months=12,
            vest_total_months=48,
        )
        create_test_company_valuation(
            test_db_session,
            company="Co X",
            currency="USD",
            fmv_per_share=10,
            fmv_low=8,
            fmv_high=14,
        )

        result = get_equity_grant(test_db_session, grant.id)
        assert result.paper_value_low == 8_000.0
        assert result.paper_value_base == 10_000.0
        assert result.paper_value_high == 14_000.0

    def test_no_valuation_returns_none(self, test_db_session):
        grant = create_test_equity_grant(
            test_db_session,
            company="Co X",
            vest_start_date=date(2020, 1, 1),
            vest_cliff_months=12,
            vest_total_months=48,
        )

        result = get_equity_grant(test_db_session, grant.id)
        assert result.paper_value_base is None
        assert result.valuation_date is None

    def test_zero_vested_returns_none(self, test_db_session):
        """Before cliff, no paper value even if valuation exists."""
        grant = create_test_equity_grant(
            test_db_session,
            company="Co X",
            currency="USD",
            vest_start_date=date(2030, 1, 1),  # future
        )
        create_test_company_valuation(
            test_db_session, company="Co X", currency="USD", fmv_per_share=10
        )

        result = get_equity_grant(test_db_session, grant.id)
        assert result.vested_shares_today == 0
        assert result.paper_value_base is None

    def test_currency_mismatch_leaves_value_none(self, test_db_session):
        """Grant USD vs valuation EUR — paper value blank until FX (Phase 5)."""
        grant = create_test_equity_grant(
            test_db_session,
            company="Co X",
            currency="USD",
            vest_start_date=date(2020, 1, 1),
            vest_cliff_months=12,
            vest_total_months=48,
        )
        create_test_company_valuation(
            test_db_session, company="Co X", currency="EUR", fmv_per_share=10
        )

        result = get_equity_grant(test_db_session, grant.id)
        assert result.paper_value_base is None
        # Date and source still surfaced so UI can show "valuation exists but FX missing"
        assert result.valuation_date is not None
