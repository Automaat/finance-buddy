from datetime import date

from pydantic import BaseModel, field_validator, model_validator


class SnapshotValueInput(BaseModel):
    """Input for single account or asset value in snapshot"""

    asset_id: int | None = None
    account_id: int | None = None
    value: float

    @model_validator(mode="after")
    def validate_exactly_one_id(self) -> "SnapshotValueInput":
        if self.asset_id is None and self.account_id is None:
            raise ValueError("Either asset_id or account_id must be provided")
        if self.asset_id is not None and self.account_id is not None:
            raise ValueError("Only one of asset_id or account_id can be provided")
        return self


class SnapshotCreate(BaseModel):
    """Request to create new snapshot"""

    date: date
    notes: str | None = None
    values: list[SnapshotValueInput]

    @field_validator("values")
    @classmethod
    def validate_values_not_empty(cls, v: list[SnapshotValueInput]) -> list[SnapshotValueInput]:
        if len(v) == 0:
            raise ValueError("Snapshot must contain at least one account value")
        return v


class SnapshotValueResponse(BaseModel):
    """Snapshot value for single account or asset"""

    id: int
    asset_id: int | None = None
    asset_name: str | None = None
    account_id: int | None = None
    account_name: str | None = None
    value: float


class SnapshotResponse(BaseModel):
    """Snapshot with all account values"""

    id: int
    date: date
    notes: str | None
    values: list[SnapshotValueResponse]


class SnapshotListItem(BaseModel):
    """Snapshot summary for list view"""

    id: int
    date: date
    notes: str | None
    total_net_worth: float
