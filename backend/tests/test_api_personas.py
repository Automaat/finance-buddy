from app.models.persona import Persona
from tests.factories import create_test_account


def test_list_personas_empty(test_client):
    """Test GET /api/personas returns empty list"""
    response = test_client.get("/api/personas")

    assert response.status_code == 200
    assert response.json() == []


def test_list_personas_ordered_by_name(test_client, test_db_session):
    """Test GET /api/personas returns personas ordered by name"""
    test_db_session.add(Persona(name="Ewa"))
    test_db_session.add(Persona(name="Marcin"))
    test_db_session.commit()

    response = test_client.get("/api/personas")

    assert response.status_code == 200
    data = response.json()
    assert len(data) == 2
    assert data[0]["name"] == "Ewa"
    assert data[1]["name"] == "Marcin"
    assert "ppk_employee_rate" in data[0]
    assert "ppk_employer_rate" in data[0]
    assert "created_at" in data[0]


def test_create_persona_success(test_client):
    """Test POST /api/personas creates a new persona"""
    payload = {"name": "Marcin", "ppk_employee_rate": "2.0", "ppk_employer_rate": "1.5"}

    response = test_client.post("/api/personas", json=payload)

    assert response.status_code == 201
    data = response.json()
    assert data["name"] == "Marcin"
    assert float(data["ppk_employee_rate"]) == 2.0
    assert float(data["ppk_employer_rate"]) == 1.5
    assert "id" in data
    assert "created_at" in data


def test_create_persona_default_rates(test_client):
    """Test POST /api/personas uses default PPK rates"""
    payload = {"name": "Ewa"}

    response = test_client.post("/api/personas", json=payload)

    assert response.status_code == 201
    data = response.json()
    assert float(data["ppk_employee_rate"]) == 2.0
    assert float(data["ppk_employer_rate"]) == 1.5


def test_create_persona_duplicate_returns_409(test_client, test_db_session):
    """Test POST /api/personas with duplicate name returns 409"""
    test_db_session.add(Persona(name="Marcin"))
    test_db_session.commit()

    payload = {"name": "Marcin"}

    response = test_client.post("/api/personas", json=payload)

    assert response.status_code == 409
    assert "already exists" in response.json()["detail"]


def test_update_persona_success(test_client, test_db_session):
    """Test PUT /api/personas/{id} updates persona fields"""
    persona = Persona(name="Marcin")
    test_db_session.add(persona)
    test_db_session.commit()

    payload = {"ppk_employee_rate": "3.0", "ppk_employer_rate": "2.0"}

    response = test_client.put(f"/api/personas/{persona.id}", json=payload)

    assert response.status_code == 200
    data = response.json()
    assert data["name"] == "Marcin"
    assert float(data["ppk_employee_rate"]) == 3.0
    assert float(data["ppk_employer_rate"]) == 2.0


def test_update_persona_name(test_client, test_db_session):
    """Test PUT /api/personas/{id} updates persona name"""
    persona = Persona(name="Marcin")
    test_db_session.add(persona)
    test_db_session.commit()

    payload = {"name": "Marcin Updated"}

    response = test_client.put(f"/api/personas/{persona.id}", json=payload)

    assert response.status_code == 200
    assert response.json()["name"] == "Marcin Updated"


def test_update_persona_not_found(test_client):
    """Test PUT /api/personas/999 returns 404"""
    response = test_client.put("/api/personas/999", json={"name": "Test"})

    assert response.status_code == 404
    assert "not found" in response.json()["detail"]


def test_update_persona_duplicate_name_returns_409(test_client, test_db_session):
    """Test PUT /api/personas/{id} with duplicate name returns 409"""
    p1 = Persona(name="Marcin")
    p2 = Persona(name="Ewa")
    test_db_session.add(p1)
    test_db_session.add(p2)
    test_db_session.commit()

    response = test_client.put(f"/api/personas/{p2.id}", json={"name": "Marcin"})

    assert response.status_code == 409
    assert "already exists" in response.json()["detail"]


def test_delete_persona_success(test_client, test_db_session):
    """Test DELETE /api/personas/{id} deletes persona"""
    persona = Persona(name="Marcin")
    test_db_session.add(persona)
    test_db_session.commit()
    persona_id = persona.id

    response = test_client.delete(f"/api/personas/{persona_id}")

    assert response.status_code == 204

    remaining = test_db_session.query(Persona).filter(Persona.id == persona_id).first()
    assert remaining is None


def test_delete_persona_not_found(test_client):
    """Test DELETE /api/personas/999 returns 404"""
    response = test_client.delete("/api/personas/999")

    assert response.status_code == 404
    assert "not found" in response.json()["detail"]


def test_delete_persona_with_accounts_returns_409(test_client, test_db_session):
    """Test DELETE /api/personas/{id} fails when accounts reference it"""
    persona = Persona(name="Marcin")
    test_db_session.add(persona)
    test_db_session.commit()

    create_test_account(test_db_session, name="Bank", owner="Marcin")

    response = test_client.delete(f"/api/personas/{persona.id}")

    assert response.status_code == 409
    assert "accounts reference it" in response.json()["detail"]
