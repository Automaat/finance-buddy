"""Black-box tests for /api/bonuses — bonus event CRUD."""

from __future__ import annotations

import httpx
import pytest

from fixtures.seed import COMPANY_MARCIN_EMPLOYER, PERSONA_MARCIN


def test_list_bonuses_includes_seeded(client: httpx.Client) -> None:
    response = client.get("/api/bonuses")
    assert response.status_code == 200, response.text
    body = response.json()
    assert body["total_count"] >= 2
    owners = {r["owner"] for r in body["bonus_events"]}
    assert PERSONA_MARCIN in owners
    assert COMPANY_MARCIN_EMPLOYER in body["available_companies"]


def test_get_bonus_by_id_returns_seeded_record(client: httpx.Client) -> None:
    listing = client.get("/api/bonuses").json()
    sample = listing["bonus_events"][0]
    response = client.get(f"/api/bonuses/{sample['id']}")
    assert response.status_code == 200, response.text
    body = response.json()
    assert body["id"] == sample["id"]
    assert body["owner"] == sample["owner"]
    assert body["amount"] == sample["amount"]


def test_get_bonus_not_found(client: httpx.Client) -> None:
    response = client.get("/api/bonuses/999999")
    assert response.status_code == 404, response.text
    assert "detail" in response.json()


def test_create_bonus_happy_path(client: httpx.Client, request: pytest.FixtureRequest) -> None:
    created_id: int | None = None
    try:
        payload = {
            "date": "2025-09-15",
            "amount": 3000.0,
            "currency": "PLN",
            "type": "spot",
            "company": f"bb-test-{request.node.name}",
            "owner": PERSONA_MARCIN,
            "contract_type": "UOP",
            "notes": "test spot bonus",
        }
        response = client.post("/api/bonuses", json=payload)
        assert response.status_code == 201, response.text
        body = response.json()
        created_id = body["id"]
        assert body["amount"] == 3000.0
        assert body["currency"] == "PLN"
        assert body["type"] == "spot"
        assert body["is_active"] is True
    finally:
        if created_id is not None:
            client.delete(f"/api/bonuses/{created_id}")


def test_create_bonus_validation_error(client: httpx.Client) -> None:
    payload = {
        "date": "2025-09-15",
        "amount": 1000.0,
        "currency": "ZZZ",
        "type": "spot",
        "company": "bb-test-bad-currency",
        "owner": PERSONA_MARCIN,
        "contract_type": "UOP",
    }
    response = client.post("/api/bonuses", json=payload)
    assert response.status_code >= 400, response.text
    assert "detail" in response.json()


def test_update_bonus_happy_path(client: httpx.Client, request: pytest.FixtureRequest) -> None:
    created_id: int | None = None
    try:
        create_payload = {
            "date": "2025-10-10",
            "amount": 2000.0,
            "currency": "PLN",
            "type": "spot",
            "company": f"bb-test-{request.node.name}",
            "owner": PERSONA_MARCIN,
            "contract_type": "UOP",
        }
        created = client.post("/api/bonuses", json=create_payload)
        assert created.status_code == 201, created.text
        created_id = created.json()["id"]

        response = client.patch(
            f"/api/bonuses/{created_id}",
            json={"amount": 2500.0, "notes": "updated"},
        )
        assert response.status_code == 200, response.text
        body = response.json()
        assert body["amount"] == 2500.0
        assert body["notes"] == "updated"
    finally:
        if created_id is not None:
            client.delete(f"/api/bonuses/{created_id}")


def test_update_bonus_validation_error(
    client: httpx.Client, request: pytest.FixtureRequest
) -> None:
    created_id: int | None = None
    try:
        create_payload = {
            "date": "2025-10-11",
            "amount": 2000.0,
            "currency": "PLN",
            "type": "spot",
            "company": f"bb-test-{request.node.name}",
            "owner": PERSONA_MARCIN,
            "contract_type": "UOP",
        }
        created = client.post("/api/bonuses", json=create_payload)
        assert created.status_code == 201, created.text
        created_id = created.json()["id"]

        response = client.patch(
            f"/api/bonuses/{created_id}",
            json={"amount": -50.0},
        )
        assert response.status_code >= 400, response.text
        assert "detail" in response.json()
    finally:
        if created_id is not None:
            client.delete(f"/api/bonuses/{created_id}")


def test_delete_bonus_happy_path(client: httpx.Client, request: pytest.FixtureRequest) -> None:
    create_payload = {
        "date": "2025-10-12",
        "amount": 1500.0,
        "currency": "PLN",
        "type": "spot",
        "company": f"bb-test-{request.node.name}",
        "owner": PERSONA_MARCIN,
        "contract_type": "UOP",
    }
    created = client.post("/api/bonuses", json=create_payload)
    assert created.status_code == 201, created.text
    created_id = created.json()["id"]

    response = client.delete(f"/api/bonuses/{created_id}")
    assert response.status_code == 204, response.text

    follow = client.get(f"/api/bonuses/{created_id}")
    assert follow.status_code == 404, follow.text
