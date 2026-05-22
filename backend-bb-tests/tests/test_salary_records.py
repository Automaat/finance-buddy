"""Black-box tests for /api/salaries — salary record CRUD."""

from __future__ import annotations

import httpx
import pytest

from fixtures.seed import COMPANY_MARCIN_EMPLOYER, PERSONA_MARCIN


def test_list_salaries_includes_seeded(client: httpx.Client, owner_ids: dict[str, int]) -> None:
    response = client.get("/api/salaries")
    assert response.status_code == 200, response.text
    body = response.json()
    assert body["total_count"] >= 3
    owners = {r["owner_user_id"] for r in body["salary_records"]}
    assert owner_ids[PERSONA_MARCIN] in owners
    assert COMPANY_MARCIN_EMPLOYER in body["available_companies"]


def test_get_salary_by_id_returns_seeded_record(client: httpx.Client) -> None:
    listing = client.get("/api/salaries").json()
    sample = listing["salary_records"][0]
    response = client.get(f"/api/salaries/{sample['id']}")
    assert response.status_code == 200, response.text
    body = response.json()
    assert body["id"] == sample["id"]
    assert body["owner_user_id"] == sample["owner_user_id"]
    assert body["gross_amount"] == sample["gross_amount"]


def test_get_salary_not_found(client: httpx.Client) -> None:
    response = client.get("/api/salaries/999999")
    assert response.status_code == 404, response.text
    assert "detail" in response.json()


def test_create_salary_happy_path(
    client: httpx.Client, request: pytest.FixtureRequest, owner_ids: dict[str, int]
) -> None:
    created_id: int | None = None
    try:
        payload = {
            "date": "2024-03-31",
            "gross_amount": 17500.0,
            "contract_type": "UOP",
            "company": f"bb-test-{request.node.name}",
            "owner_user_id": owner_ids[PERSONA_MARCIN],
        }
        response = client.post("/api/salaries", json=payload)
        assert response.status_code == 201, response.text
        body = response.json()
        created_id = body["id"]
        assert body["gross_amount"] == 17500.0
        assert body["company"] == payload["company"]
        assert body["owner_user_id"] == owner_ids[PERSONA_MARCIN]
        assert body["is_active"] is True
    finally:
        if created_id is not None:
            client.delete(f"/api/salaries/{created_id}")


def test_create_salary_validation_error(client: httpx.Client, owner_ids: dict[str, int]) -> None:
    payload = {
        "date": "2024-03-31",
        "gross_amount": -100.0,
        "contract_type": "UOP",
        "company": "bb-test-invalid",
        "owner_user_id": owner_ids[PERSONA_MARCIN],
    }
    response = client.post("/api/salaries", json=payload)
    assert response.status_code >= 400, response.text
    assert "detail" in response.json()


def test_update_salary_happy_path(
    client: httpx.Client, request: pytest.FixtureRequest, owner_ids: dict[str, int]
) -> None:
    created_id: int | None = None
    try:
        create_payload = {
            "date": "2024-04-30",
            "gross_amount": 16000.0,
            "contract_type": "UOP",
            "company": f"bb-test-{request.node.name}",
            "owner_user_id": owner_ids[PERSONA_MARCIN],
        }
        created = client.post("/api/salaries", json=create_payload)
        assert created.status_code == 201, created.text
        created_id = created.json()["id"]

        response = client.patch(
            f"/api/salaries/{created_id}",
            json={"gross_amount": 16500.0},
        )
        assert response.status_code == 200, response.text
        assert response.json()["gross_amount"] == 16500.0
    finally:
        if created_id is not None:
            client.delete(f"/api/salaries/{created_id}")


def test_update_salary_validation_error(
    client: httpx.Client, request: pytest.FixtureRequest, owner_ids: dict[str, int]
) -> None:
    created_id: int | None = None
    try:
        create_payload = {
            "date": "2024-05-31",
            "gross_amount": 16000.0,
            "contract_type": "UOP",
            "company": f"bb-test-{request.node.name}",
            "owner_user_id": owner_ids[PERSONA_MARCIN],
        }
        created = client.post("/api/salaries", json=create_payload)
        assert created.status_code == 201, created.text
        created_id = created.json()["id"]

        response = client.patch(
            f"/api/salaries/{created_id}",
            json={"gross_amount": -1.0},
        )
        assert response.status_code >= 400, response.text
        assert "detail" in response.json()
    finally:
        if created_id is not None:
            client.delete(f"/api/salaries/{created_id}")


def test_delete_salary_happy_path(
    client: httpx.Client, request: pytest.FixtureRequest, owner_ids: dict[str, int]
) -> None:
    create_payload = {
        "date": "2024-06-30",
        "gross_amount": 15000.0,
        "contract_type": "UOP",
        "company": f"bb-test-{request.node.name}",
        "owner_user_id": owner_ids[PERSONA_MARCIN],
    }
    created = client.post("/api/salaries", json=create_payload)
    assert created.status_code == 201, created.text
    created_id = created.json()["id"]

    response = client.delete(f"/api/salaries/{created_id}")
    assert response.status_code == 204, response.text

    follow = client.get(f"/api/salaries/{created_id}")
    assert follow.status_code == 404, follow.text
