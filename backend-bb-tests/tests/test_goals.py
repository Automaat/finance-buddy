"""Black-box tests for /api/goals — CRUD coverage against the seeded backend."""

from __future__ import annotations

from collections.abc import Iterator
from contextlib import contextmanager

import httpx
import pytest

from _golden import assert_matches_golden


@contextmanager
def _temp_goal(client: httpx.Client, name: str, **overrides: object) -> Iterator[dict]:
    """Create a goal, yield its body, hard-delete afterwards."""
    payload: dict = {
        "name": name,
        "target_amount": 5000.0,
        "target_date": "2030-12-31",
        "current_amount": 0.0,
        "monthly_contribution": 0.0,
    }
    payload.update(overrides)
    response = client.post("/api/goals", json=payload)
    assert response.status_code == 201, response.text
    body = response.json()
    try:
        yield body
    finally:
        client.delete(f"/api/goals/{body['id']}")


@pytest.mark.golden
def test_list_goals_matches_golden(client: httpx.Client, update_golden: bool) -> None:
    response = client.get("/api/goals")
    assert response.status_code == 200, response.text
    assert_matches_golden("goals_list", response.json(), update=update_golden)


def test_get_goal_by_id_happy_path(client: httpx.Client, request: pytest.FixtureRequest) -> None:
    unique_name = f"bb-test-{request.node.name}-goal"
    with _temp_goal(client, unique_name, target_amount=10000.0, current_amount=2500.0) as goal:
        response = client.get(f"/api/goals/{goal['id']}")
        assert response.status_code == 200, response.text
        body = response.json()
        assert body["id"] == goal["id"]
        assert body["name"] == unique_name
        assert body["target_amount"] == 10000.0
        assert body["current_amount"] == 2500.0
        assert body["remaining_amount"] == 7500.0


def test_create_goal_happy_path(client: httpx.Client, request: pytest.FixtureRequest) -> None:
    unique_name = f"bb-test-{request.node.name}-goal"
    created_id: int | None = None
    try:
        response = client.post(
            "/api/goals",
            json={
                "name": unique_name,
                "target_amount": 10000.0,
                "target_date": "2030-12-31",
                "current_amount": 1000.0,
                "monthly_contribution": 100.0,
            },
        )
        assert response.status_code == 201, response.text
        body = response.json()
        created_id = int(body["id"])
        assert body["name"] == unique_name
        assert body["target_amount"] == 10000.0
        assert body["current_amount"] == 1000.0
        assert body["remaining_amount"] == 9000.0
    finally:
        if created_id is not None:
            client.delete(f"/api/goals/{created_id}")


def test_create_goal_validation_error(client: httpx.Client) -> None:
    response = client.post(
        "/api/goals",
        json={
            "name": "bb-test-goal-invalid",
            "target_amount": -100.0,
            "target_date": "2030-12-31",
        },
    )
    assert response.status_code >= 400, response.text
    assert "detail" in response.json()


def test_update_goal_happy_path(client: httpx.Client, request: pytest.FixtureRequest) -> None:
    unique_name = f"bb-test-{request.node.name}-goal"
    renamed = f"{unique_name}-renamed"
    with _temp_goal(client, unique_name, target_amount=5000.0) as goal:
        response = client.put(
            f"/api/goals/{goal['id']}",
            json={"name": renamed, "current_amount": 2500.0},
        )
        assert response.status_code == 200, response.text
        body = response.json()
        assert body["id"] == goal["id"]
        assert body["name"] == renamed
        assert body["current_amount"] == 2500.0


def test_update_goal_validation_error(client: httpx.Client, request: pytest.FixtureRequest) -> None:
    unique_name = f"bb-test-{request.node.name}-goal"
    with _temp_goal(client, unique_name) as goal:
        response = client.put(f"/api/goals/{goal['id']}", json={"target_amount": -1.0})
        assert response.status_code >= 400, response.text
        assert "detail" in response.json()


def test_delete_goal_happy_path(client: httpx.Client, request: pytest.FixtureRequest) -> None:
    unique_name = f"bb-test-{request.node.name}-goal"
    create_response = client.post(
        "/api/goals",
        json={
            "name": unique_name,
            "target_amount": 1000.0,
            "target_date": "2030-12-31",
        },
    )
    assert create_response.status_code == 201, create_response.text
    created_id = int(create_response.json()["id"])

    response = client.delete(f"/api/goals/{created_id}")
    assert response.status_code == 204, response.text

    follow_up = client.get(f"/api/goals/{created_id}")
    assert follow_up.status_code == 404, follow_up.text
