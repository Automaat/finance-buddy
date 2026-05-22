"""Black-box tests for /api/equity-grants — equity grant CRUD."""

from __future__ import annotations

import httpx
import pytest

from fixtures.seed import COMPANY_MARCIN_EMPLOYER, PERSONA_MARCIN


def test_list_equity_grants_includes_seeded(
    client: httpx.Client, owner_ids: dict[str, int]
) -> None:
    response = client.get("/api/equity-grants")
    assert response.status_code == 200, response.text
    body = response.json()
    assert body["total_count"] >= 1
    owners = {r["owner_user_id"] for r in body["equity_grants"]}
    assert owner_ids[PERSONA_MARCIN] in owners
    assert COMPANY_MARCIN_EMPLOYER in body["available_companies"]


def test_get_equity_grant_by_id_returns_seeded_record(client: httpx.Client) -> None:
    listing = client.get("/api/equity-grants").json()
    sample = listing["equity_grants"][0]
    response = client.get(f"/api/equity-grants/{sample['id']}")
    assert response.status_code == 200, response.text
    body = response.json()
    assert body["id"] == sample["id"]
    assert body["owner_user_id"] == sample["owner_user_id"]
    assert body["total_shares"] == sample["total_shares"]


def test_get_equity_grant_not_found(client: httpx.Client) -> None:
    response = client.get("/api/equity-grants/999999")
    assert response.status_code == 404, response.text
    assert "detail" in response.json()


def test_create_equity_grant_happy_path(
    client: httpx.Client, request: pytest.FixtureRequest, owner_ids: dict[str, int]
) -> None:
    created_id: int | None = None
    try:
        payload = {
            "grant_date": "2024-02-01",
            "type": "rsu",
            "company": f"bb-test-{request.node.name}",
            "owner_user_id": owner_ids[PERSONA_MARCIN],
            "total_shares": 1200,
            "currency": "USD",
            "vest_start_date": "2024-02-01",
            "vest_cliff_months": 12,
            "vest_total_months": 48,
            "vest_frequency": "monthly",
            "tax_treatment": "capital_gains_19",
            "notes": "test grant",
        }
        response = client.post("/api/equity-grants", json=payload)
        assert response.status_code == 201, response.text
        body = response.json()
        created_id = body["id"]
        assert body["total_shares"] == 1200
        assert body["type"] == "rsu"
        assert body["owner_user_id"] == owner_ids[PERSONA_MARCIN]
        assert body["is_active"] is True
    finally:
        if created_id is not None:
            client.delete(f"/api/equity-grants/{created_id}")


def test_create_equity_grant_validation_error(
    client: httpx.Client, owner_ids: dict[str, int]
) -> None:
    payload = {
        "grant_date": "2024-02-01",
        "type": "rsu",
        "company": "bb-test-bad",
        "owner_user_id": owner_ids[PERSONA_MARCIN],
        "total_shares": 0,
        "currency": "USD",
        "vest_start_date": "2024-02-01",
        "vest_cliff_months": 12,
        "vest_total_months": 48,
        "vest_frequency": "monthly",
        "tax_treatment": "capital_gains_19",
    }
    response = client.post("/api/equity-grants", json=payload)
    assert response.status_code >= 400, response.text
    assert "detail" in response.json()


def test_update_equity_grant_happy_path(
    client: httpx.Client, request: pytest.FixtureRequest, owner_ids: dict[str, int]
) -> None:
    created_id: int | None = None
    try:
        create_payload = {
            "grant_date": "2024-03-01",
            "type": "rsu",
            "company": f"bb-test-{request.node.name}",
            "owner_user_id": owner_ids[PERSONA_MARCIN],
            "total_shares": 800,
            "currency": "USD",
            "vest_start_date": "2024-03-01",
            "vest_cliff_months": 12,
            "vest_total_months": 48,
            "vest_frequency": "monthly",
            "tax_treatment": "capital_gains_19",
        }
        created = client.post("/api/equity-grants", json=create_payload)
        assert created.status_code == 201, created.text
        created_id = created.json()["id"]

        response = client.patch(
            f"/api/equity-grants/{created_id}",
            json={"notes": "updated notes", "total_shares": 900},
        )
        assert response.status_code == 200, response.text
        body = response.json()
        assert body["notes"] == "updated notes"
        assert body["total_shares"] == 900
    finally:
        if created_id is not None:
            client.delete(f"/api/equity-grants/{created_id}")


def test_update_equity_grant_validation_error(
    client: httpx.Client, request: pytest.FixtureRequest, owner_ids: dict[str, int]
) -> None:
    created_id: int | None = None
    try:
        create_payload = {
            "grant_date": "2024-04-01",
            "type": "rsu",
            "company": f"bb-test-{request.node.name}",
            "owner_user_id": owner_ids[PERSONA_MARCIN],
            "total_shares": 800,
            "currency": "USD",
            "vest_start_date": "2024-04-01",
            "vest_cliff_months": 12,
            "vest_total_months": 48,
            "vest_frequency": "monthly",
            "tax_treatment": "capital_gains_19",
        }
        created = client.post("/api/equity-grants", json=create_payload)
        assert created.status_code == 201, created.text
        created_id = created.json()["id"]

        response = client.patch(
            f"/api/equity-grants/{created_id}",
            json={"total_shares": -10},
        )
        assert response.status_code >= 400, response.text
        assert "detail" in response.json()
    finally:
        if created_id is not None:
            client.delete(f"/api/equity-grants/{created_id}")


def test_delete_equity_grant_happy_path(
    client: httpx.Client, request: pytest.FixtureRequest, owner_ids: dict[str, int]
) -> None:
    create_payload = {
        "grant_date": "2024-05-01",
        "type": "rsu",
        "company": f"bb-test-{request.node.name}",
        "owner_user_id": owner_ids[PERSONA_MARCIN],
        "total_shares": 500,
        "currency": "USD",
        "vest_start_date": "2024-05-01",
        "vest_cliff_months": 12,
        "vest_total_months": 48,
        "vest_frequency": "monthly",
        "tax_treatment": "capital_gains_19",
    }
    created = client.post("/api/equity-grants", json=create_payload)
    assert created.status_code == 201, created.text
    created_id = created.json()["id"]

    response = client.delete(f"/api/equity-grants/{created_id}")
    assert response.status_code == 204, response.text

    follow = client.get(f"/api/equity-grants/{created_id}")
    assert follow.status_code == 404, follow.text
