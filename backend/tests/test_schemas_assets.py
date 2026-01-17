import pytest
from pydantic import ValidationError

from app.schemas.assets import AssetCreate, AssetUpdate


def test_asset_create_valid():
    """Test creating AssetCreate with valid name"""
    asset = AssetCreate(name="Car")
    assert asset.name == "Car"


def test_asset_create_strips_whitespace():
    """Test AssetCreate strips whitespace from name"""
    asset = AssetCreate(name="  Car  ")
    assert asset.name == "Car"


def test_asset_create_empty_name():
    """Test AssetCreate rejects empty name"""
    with pytest.raises(ValidationError) as exc_info:
        AssetCreate(name="")

    assert "Name cannot be empty" in str(exc_info.value)


def test_asset_create_whitespace_only_name():
    """Test AssetCreate rejects whitespace-only name"""
    with pytest.raises(ValidationError) as exc_info:
        AssetCreate(name="   ")

    assert "Name cannot be empty" in str(exc_info.value)


def test_asset_update_valid():
    """Test creating AssetUpdate with valid name"""
    asset = AssetUpdate(name="New Name")
    assert asset.name == "New Name"


def test_asset_update_none_name():
    """Test AssetUpdate allows None name"""
    asset = AssetUpdate(name=None)
    assert asset.name is None


def test_asset_update_strips_whitespace():
    """Test AssetUpdate strips whitespace from name"""
    asset = AssetUpdate(name="  New Name  ")
    assert asset.name == "New Name"


def test_asset_update_empty_name():
    """Test AssetUpdate rejects empty name"""
    with pytest.raises(ValidationError) as exc_info:
        AssetUpdate(name="")

    assert "Name cannot be empty" in str(exc_info.value)


def test_asset_update_whitespace_only_name():
    """Test AssetUpdate rejects whitespace-only name"""
    with pytest.raises(ValidationError) as exc_info:
        AssetUpdate(name="   ")

    assert "Name cannot be empty" in str(exc_info.value)
