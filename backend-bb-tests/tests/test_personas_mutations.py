"""Black-box tests for /api/personas mutations — list endpoint is covered in test_personas.py."""

from __future__ import annotations

import httpx
import pytest


def _find_persona_id(client: httpx.Client, name: str) -> int | None:
    response = client.get("/api/personas")
    assert response.status_code == 200, response.text
    for persona in response.json():
        if persona["name"] == name:
            return int(persona["id"])
    return None


def test_create_persona_happy_path(client: httpx.Client, request: pytest.FixtureRequest) -> None:
    unique_name = f"bb-test-{request.node.name}-persona"
    created_id: int | None = None
    try:
        response = client.post(
            "/api/personas",
            json={
                "name": unique_name,
                "ppk_employee_rate": "2.0",
                "ppk_employer_rate": "1.5",
            },
        )
        assert response.status_code == 201, response.text
        body = response.json()
        created_id = int(body["id"])
        assert body["name"] == unique_name
        # API serialises Decimal with trailing zeros preserved (Decimal("2.00")).
        assert body["ppk_employee_rate"] in {"2.0", "2.00"}
        assert body["ppk_employer_rate"] in {"1.5", "1.50"}
    finally:
        if created_id is not None:
            client.delete(f"/api/personas/{created_id}")


def test_create_persona_validation_error(client: httpx.Client) -> None:
    response = client.post(
        "/api/personas",
        json={"name": "bb-test-invalid", "ppk_employee_rate": "99.0"},
    )
    assert response.status_code >= 400, response.text
    assert "detail" in response.json()


def test_update_persona_happy_path(client: httpx.Client, request: pytest.FixtureRequest) -> None:
    unique_name = f"bb-test-{request.node.name}-persona"
    renamed = f"{unique_name}-renamed"
    create_response = client.post("/api/personas", json={"name": unique_name})
    assert create_response.status_code == 201, create_response.text
    created_id = int(create_response.json()["id"])

    try:
        response = client.put(
            f"/api/personas/{created_id}",
            json={"name": renamed, "ppk_employee_rate": "3.0"},
        )
        assert response.status_code == 200, response.text
        body = response.json()
        assert body["id"] == created_id
        assert body["name"] == renamed
        assert body["ppk_employee_rate"] in {"3.0", "3.00"}
    finally:
        # Look up by current name in case the rename succeeded.
        cleanup_id = _find_persona_id(client, renamed) or created_id
        client.delete(f"/api/personas/{cleanup_id}")


def test_update_persona_validation_error(
    client: httpx.Client, request: pytest.FixtureRequest
) -> None:
    unique_name = f"bb-test-{request.node.name}-persona"
    create_response = client.post("/api/personas", json={"name": unique_name})
    assert create_response.status_code == 201, create_response.text
    created_id = int(create_response.json()["id"])

    try:
        response = client.put(
            f"/api/personas/{created_id}",
            json={"ppk_employer_rate": "0.0"},
        )
        assert response.status_code >= 400, response.text
        assert "detail" in response.json()
    finally:
        client.delete(f"/api/personas/{created_id}")


def test_delete_persona_happy_path(client: httpx.Client, request: pytest.FixtureRequest) -> None:
    unique_name = f"bb-test-{request.node.name}-persona"
    create_response = client.post("/api/personas", json={"name": unique_name})
    assert create_response.status_code == 201, create_response.text
    created_id = int(create_response.json()["id"])

    response = client.delete(f"/api/personas/{created_id}")
    assert response.status_code == 204, response.text

    assert _find_persona_id(client, unique_name) is None
