from datetime import datetime

from pydantic import BaseModel, field_validator

from app.utils.validators import validate_not_empty_string


class AssetCreate(BaseModel):
    name: str

    @field_validator("name")
    @classmethod
    def validate_name(cls, v: str) -> str:
        return validate_not_empty_string(v)  # type: ignore[return-value]


class AssetResponse(BaseModel):
    id: int
    name: str
    is_active: bool
    created_at: datetime
    current_value: float


class AssetUpdate(BaseModel):
    name: str | None = None

    @field_validator("name")
    @classmethod
    def validate_name(cls, v: str | None) -> str | None:
        return validate_not_empty_string(v)


class AssetsListResponse(BaseModel):
    assets: list[AssetResponse]
