"""Tests for persona schemas validation"""

from decimal import Decimal

import pytest
from pydantic import ValidationError

from app.schemas.personas import PersonaCreate, PersonaUpdate


class TestPersonaCreateValidation:
    """Test PersonaCreate schema validators"""

    def test_valid_persona_create(self):
        """Test creating valid persona"""
        data = PersonaCreate(
            name="Marcin", ppk_employee_rate=Decimal("2.0"), ppk_employer_rate=Decimal("1.5")
        )
        assert data.name == "Marcin"
        assert data.ppk_employee_rate == Decimal("2.0")
        assert data.ppk_employer_rate == Decimal("1.5")

    def test_name_stripped(self):
        """Test name whitespace is stripped"""
        data = PersonaCreate(name="  Marcin  ")
        assert data.name == "Marcin"

    def test_empty_name_fails(self):
        """Test empty name raises validation error"""
        with pytest.raises(ValidationError, match="Name cannot be empty"):
            PersonaCreate(name="   ")

    def test_ppk_rate_below_minimum_fails(self):
        """Test PPK rate below 0.5 raises validation error"""
        with pytest.raises(ValidationError, match="PPK rate must be between"):
            PersonaCreate(name="Marcin", ppk_employee_rate=Decimal("0.4"))

    def test_ppk_rate_above_maximum_fails(self):
        """Test PPK rate above 4.0 raises validation error"""
        with pytest.raises(ValidationError, match="PPK rate must be between"):
            PersonaCreate(name="Marcin", ppk_employer_rate=Decimal("4.1"))

    def test_ppk_rate_at_boundary_valid(self):
        """Test PPK rate at boundary values 0.5 and 4.0 is valid"""
        data = PersonaCreate(
            name="Test", ppk_employee_rate=Decimal("0.5"), ppk_employer_rate=Decimal("4.0")
        )
        assert data.ppk_employee_rate == Decimal("0.5")
        assert data.ppk_employer_rate == Decimal("4.0")


class TestPersonaUpdateValidation:
    """Test PersonaUpdate schema validators"""

    def test_valid_partial_update(self):
        """Test partial update with only some fields"""
        data = PersonaUpdate(ppk_employee_rate=Decimal("3.0"))
        assert data.name is None
        assert data.ppk_employee_rate == Decimal("3.0")
        assert data.ppk_employer_rate is None

    def test_name_stripped_on_update(self):
        """Test name whitespace is stripped on update"""
        data = PersonaUpdate(name="  Ewa  ")
        assert data.name == "Ewa"

    def test_empty_name_fails_on_update(self):
        """Test empty name raises validation error on update"""
        with pytest.raises(ValidationError, match="Name cannot be empty"):
            PersonaUpdate(name="   ")

    def test_none_name_allowed(self):
        """Test None name is allowed (partial update)"""
        data = PersonaUpdate(name=None)
        assert data.name is None

    def test_ppk_rate_none_allowed(self):
        """Test None PPK rates are allowed (partial update)"""
        data = PersonaUpdate(ppk_employee_rate=None, ppk_employer_rate=None)
        assert data.ppk_employee_rate is None
        assert data.ppk_employer_rate is None

    def test_ppk_rate_below_minimum_fails_on_update(self):
        """Test PPK rate below 0.5 raises validation error on update"""
        with pytest.raises(ValidationError, match="PPK rate must be between"):
            PersonaUpdate(ppk_employee_rate=Decimal("0.4"))

    def test_ppk_rate_above_maximum_fails_on_update(self):
        """Test PPK rate above 4.0 raises validation error on update"""
        with pytest.raises(ValidationError, match="PPK rate must be between"):
            PersonaUpdate(ppk_employer_rate=Decimal("4.5"))
