"""API tests for the /api/cpi router and salaries inflation_context."""

from datetime import UTC, date, datetime
from decimal import Decimal
from unittest.mock import patch

import httpx
import pytest

from app.models import CpiIndex, Persona
from tests.factories import create_test_salary_record


@pytest.fixture
def seeded_cpi(test_db_session):
    rows = [
        (2020, "103.4"),
        (2021, "105.1"),
        (2022, "114.4"),
        (2023, "111.4"),
        (2024, "103.6"),
        (2025, "103.6"),
    ]
    for year, yoy in rows:
        test_db_session.add(
            CpiIndex(
                year=year,
                yoy_rate=Decimal(yoy),
                source="test",
                fetched_at=datetime.now(UTC),
            )
        )
    test_db_session.commit()
    return rows


def test_cpi_series_empty(test_client):
    response = test_client.get("/api/cpi/series")
    assert response.status_code == 200
    body = response.json()
    assert body["points"] == []
    assert body["base_year"] is None
    assert body["latest_year"] is None


@pytest.mark.usefixtures("seeded_cpi")
def test_cpi_series_returns_sorted_points(test_client):
    response = test_client.get("/api/cpi/series")
    assert response.status_code == 200
    body = response.json()
    assert body["base_year"] == 2020
    assert body["latest_year"] == 2025
    years = [p["year"] for p in body["points"]]
    assert years == [2020, 2021, 2022, 2023, 2024, 2025]
    # First point's cumulative index is anchored at 100.
    assert body["points"][0]["cumulative_index"] == pytest.approx(100.0)


@pytest.mark.usefixtures("seeded_cpi")
def test_cpi_adjust_compounds(test_client):
    response = test_client.post(
        "/api/cpi/adjust",
        json={
            "amount": 10000,
            "from_date": "2022-01-01",
            "to_date": "2026-05-20",
        },
    )
    assert response.status_code == 200
    body = response.json()
    # 1.144 * 1.114 * 1.036 * 1.036 = 1.368 (~13680)
    assert body["adjusted_amount"] == pytest.approx(13680.0, rel=5e-3)
    assert body["factor"] == pytest.approx(1.368, rel=5e-3)
    assert body["as_of_year"] == 2025


def test_cpi_adjust_returns_503_when_table_empty(test_client):
    response = test_client.post(
        "/api/cpi/adjust",
        json={"amount": 1000, "from_date": "2020-01-01", "to_date": "2025-01-01"},
    )
    assert response.status_code == 503


def test_cpi_refresh_returns_502_on_network_error(test_client):
    async def boom(_db):
        raise httpx.ConnectError("simulated network failure")

    with patch("app.api.cpi.inflation.refresh_cpi", side_effect=boom):
        response = test_client.post("/api/cpi/refresh")

    assert response.status_code == 502
    assert "GUS BDL fetch failed" in response.json()["detail"]


def test_cpi_refresh_success(test_client):
    async def fake_refresh(_db):
        _db.add(
            CpiIndex(
                year=2024,
                yoy_rate=Decimal("103.6"),
                source="test",
                fetched_at=datetime.now(UTC),
            )
        )
        _db.commit()
        return 1

    with patch("app.api.cpi.inflation.refresh_cpi", side_effect=fake_refresh):
        response = test_client.post("/api/cpi/refresh")

    assert response.status_code == 200
    body = response.json()
    assert body["rows_written"] == 1
    assert body["latest_year"] == 2024


@pytest.mark.usefixtures("seeded_cpi")
def test_salaries_inflation_context_present(test_client, test_db_session):
    test_db_session.add(Persona(name="Marcin"))
    test_db_session.commit()
    create_test_salary_record(
        test_db_session,
        salary_date=date(2022, 1, 1),
        gross_amount=10000.0,
        company="Co A",
        owner="Marcin",
    )
    create_test_salary_record(
        test_db_session,
        salary_date=date(2024, 6, 1),
        gross_amount=12000.0,
        company="Co B",
        owner="Marcin",
    )

    response = test_client.get("/api/salaries")
    assert response.status_code == 200
    body = response.json()
    ctx = body["inflation_context"]["Marcin"]
    assert ctx["previous_change_date"] == "2022-01-01"
    assert ctx["last_change_date"] == "2024-06-01"
    assert ctx["previous_salary"] == 10000.0
    assert ctx["current_salary"] == 12000.0
    # Real raise should be negative — 12k didn't beat 14%+11% inflation.
    assert ctx["real_change_pln"] < 0
    assert ctx["cpi_as_of_year"] == 2025


@pytest.mark.usefixtures("seeded_cpi")
def test_salaries_inflation_context_absent_with_one_record(test_client, test_db_session):
    test_db_session.add(Persona(name="Marcin"))
    test_db_session.commit()
    create_test_salary_record(
        test_db_session,
        salary_date=date(2024, 1, 1),
        gross_amount=10000.0,
        company="Co A",
        owner="Marcin",
    )

    response = test_client.get("/api/salaries")
    assert response.status_code == 200
    assert "Marcin" not in response.json()["inflation_context"]


def test_salaries_inflation_context_absent_without_cpi(test_client, test_db_session):
    test_db_session.add(Persona(name="Marcin"))
    test_db_session.commit()
    create_test_salary_record(
        test_db_session,
        salary_date=date(2022, 1, 1),
        gross_amount=10000.0,
        company="Co A",
        owner="Marcin",
    )
    create_test_salary_record(
        test_db_session,
        salary_date=date(2024, 6, 1),
        gross_amount=12000.0,
        company="Co B",
        owner="Marcin",
    )

    response = test_client.get("/api/salaries")
    assert response.status_code == 200
    assert response.json()["inflation_context"] == {}
