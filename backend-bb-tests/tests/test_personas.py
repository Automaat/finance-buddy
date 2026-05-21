"""Smoke tests for /api/personas — covers the seeded personas list."""

from __future__ import annotations

import httpx


def test_list_personas_includes_seeded(client: httpx.Client) -> None:
    response = client.get("/api/personas")
    assert response.status_code == 200, response.text
    names = {p["name"] for p in response.json()}
    assert {"Marcin", "Ewa"}.issubset(names)


def test_persona_response_shape(client: httpx.Client) -> None:
    response = client.get("/api/personas")
    assert response.status_code == 200
    persona = response.json()[0]
    required = {"id", "name", "ppk_employee_rate", "ppk_employer_rate", "created_at"}
    missing = required - persona.keys()
    assert not missing, f"Persona response is missing fields: {sorted(missing)}"
