from datetime import UTC, datetime

from tests.factories import create_test_account


def test_get_goals_empty(test_client):
    """GET /api/goals returns empty list when no goals exist."""
    response = test_client.get("/api/goals")
    assert response.status_code == 200
    data = response.json()
    assert data["goals"] == []
    assert data["total_count"] == 0
    assert data["completed_count"] == 0


def test_create_goal_minimal(test_client):
    """POST /api/goals creates a goal with required fields only."""
    payload = {
        "name": "Emergency Fund",
        "target_amount": 10000,
        "target_date": "2027-12-31",
    }
    response = test_client.post("/api/goals", json=payload)
    assert response.status_code == 201
    data = response.json()
    assert data["name"] == "Emergency Fund"
    assert data["target_amount"] == 10000
    assert data["current_amount"] == 0
    assert data["progress_percent"] == 0
    assert data["remaining_amount"] == 10000
    assert data["is_completed"] is False
    assert data["account_id"] is None
    assert data["category"] is None
    assert data["projected_hit_date"] is None


def test_create_goal_with_progress(test_client):
    """POST /api/goals computes progress_percent and remaining_amount."""
    payload = {
        "name": "Vacation",
        "target_amount": 5000,
        "current_amount": 1500,
        "target_date": "2026-12-31",
    }
    response = test_client.post("/api/goals", json=payload)
    assert response.status_code == 201
    data = response.json()
    assert data["progress_percent"] == 30.0
    assert data["remaining_amount"] == 3500


def test_create_goal_with_projection(test_client):
    """POST /api/goals projects hit date when monthly_contribution > 0."""
    payload = {
        "name": "Down Payment",
        "target_amount": 50000,
        "current_amount": 10000,
        "monthly_contribution": 1000,
        "target_date": "2030-01-01",
    }
    response = test_client.post("/api/goals", json=payload)
    assert response.status_code == 201
    data = response.json()
    assert data["projected_hit_date"] is not None


def test_create_goal_with_account(test_client, test_db_session):
    """POST /api/goals links to an existing account."""
    account = create_test_account(test_db_session, name="Savings", category="saving_account")
    payload = {
        "name": "House",
        "target_amount": 100000,
        "target_date": "2030-01-01",
        "account_id": account.id,
        "category": "saving_account",
    }
    response = test_client.post("/api/goals", json=payload)
    assert response.status_code == 201
    data = response.json()
    assert data["account_id"] == account.id
    assert data["account_name"] == "Savings"
    assert data["category"] == "saving_account"


def test_create_goal_invalid_account(test_client):
    """POST /api/goals with non-existent account_id returns 404."""
    payload = {
        "name": "Bad",
        "target_amount": 1000,
        "target_date": "2027-01-01",
        "account_id": 9999,
    }
    response = test_client.post("/api/goals", json=payload)
    assert response.status_code == 404


def test_create_goal_empty_name(test_client):
    payload = {
        "name": "",
        "target_amount": 1000,
        "target_date": "2027-01-01",
    }
    response = test_client.post("/api/goals", json=payload)
    assert response.status_code == 422


def test_create_goal_zero_target(test_client):
    payload = {
        "name": "Zero",
        "target_amount": 0,
        "target_date": "2027-01-01",
    }
    response = test_client.post("/api/goals", json=payload)
    assert response.status_code == 422


def test_create_goal_negative_current(test_client):
    payload = {
        "name": "Neg",
        "target_amount": 1000,
        "current_amount": -10,
        "target_date": "2027-01-01",
    }
    response = test_client.post("/api/goals", json=payload)
    assert response.status_code == 422


def test_get_goal_by_id(test_client):
    create = test_client.post(
        "/api/goals",
        json={"name": "X", "target_amount": 100, "target_date": "2027-01-01"},
    )
    goal_id = create.json()["id"]

    response = test_client.get(f"/api/goals/{goal_id}")
    assert response.status_code == 200
    assert response.json()["name"] == "X"


def test_get_goal_not_found(test_client):
    response = test_client.get("/api/goals/9999")
    assert response.status_code == 404


def test_update_goal(test_client):
    create = test_client.post(
        "/api/goals",
        json={"name": "Old", "target_amount": 1000, "target_date": "2027-01-01"},
    )
    goal_id = create.json()["id"]

    response = test_client.put(
        f"/api/goals/{goal_id}",
        json={"name": "New", "current_amount": 500, "monthly_contribution": 100},
    )
    assert response.status_code == 200
    data = response.json()
    assert data["name"] == "New"
    assert data["current_amount"] == 500
    assert data["monthly_contribution"] == 100
    assert data["projected_hit_date"] is not None


def test_update_goal_mark_completed(test_client):
    create = test_client.post(
        "/api/goals",
        json={"name": "Almost", "target_amount": 1000, "target_date": "2027-01-01"},
    )
    goal_id = create.json()["id"]

    response = test_client.put(
        f"/api/goals/{goal_id}",
        json={"is_completed": True, "current_amount": 1000},
    )
    assert response.status_code == 200
    data = response.json()
    assert data["is_completed"] is True
    assert data["progress_percent"] == 100


def test_update_goal_unlink_account(test_client, test_db_session):
    account = create_test_account(test_db_session, name="Linked")
    create = test_client.post(
        "/api/goals",
        json={
            "name": "Linked Goal",
            "target_amount": 1000,
            "target_date": "2027-01-01",
            "account_id": account.id,
        },
    )
    goal_id = create.json()["id"]

    response = test_client.put(f"/api/goals/{goal_id}", json={"account_id": None})
    assert response.status_code == 200
    assert response.json()["account_id"] is None


def test_update_goal_not_found(test_client):
    response = test_client.put("/api/goals/9999", json={"name": "X"})
    assert response.status_code == 404


def test_delete_goal(test_client):
    create = test_client.post(
        "/api/goals",
        json={"name": "To Delete", "target_amount": 100, "target_date": "2027-01-01"},
    )
    goal_id = create.json()["id"]

    response = test_client.delete(f"/api/goals/{goal_id}")
    assert response.status_code == 204

    get_response = test_client.get(f"/api/goals/{goal_id}")
    assert get_response.status_code == 404


def test_delete_goal_not_found(test_client):
    response = test_client.delete("/api/goals/9999")
    assert response.status_code == 404


def test_get_all_goals_counts(test_client):
    test_client.post(
        "/api/goals",
        json={"name": "G1", "target_amount": 100, "target_date": "2027-01-01"},
    )
    test_client.post(
        "/api/goals",
        json={
            "name": "G2",
            "target_amount": 100,
            "current_amount": 100,
            "is_completed": True,
            "target_date": "2026-06-30",
        },
    )

    response = test_client.get("/api/goals")
    assert response.status_code == 200
    data = response.json()
    assert data["total_count"] == 2
    assert data["completed_count"] == 1


def test_create_goal_target_already_reached(test_client):
    """Goal where current >= target should have projected_hit_date set to today."""
    today_before = datetime.now(UTC).date()
    response = test_client.post(
        "/api/goals",
        json={
            "name": "Done",
            "target_amount": 1000,
            "current_amount": 1500,
            "monthly_contribution": 0,
            "target_date": "2027-01-01",
        },
    )
    today_after = datetime.now(UTC).date()
    assert response.status_code == 201
    data = response.json()
    assert data["progress_percent"] == 100
    assert data["remaining_amount"] == 0
    # Accept either side of a UTC midnight crossover during the request
    assert data["projected_hit_date"] in {today_before.isoformat(), today_after.isoformat()}
