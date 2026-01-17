from datetime import datetime

from pydantic import BaseModel, field_validator


class AssetCreate(BaseModel):
    name: str

    @field_validator("name")
    @classmethod
    def validate_name(cls, v: str) -> str:
        if not v or not v.strip():
            raise ValueError("Name cannot be empty")
        return v.strip()


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
        if v is not None:
            if not v or not v.strip():
                raise ValueError("Name cannot be empty")
            return v.strip()
        return v


class AssetsListResponse(BaseModel):
    assets: list[AssetResponse]
