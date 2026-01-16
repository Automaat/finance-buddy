from datetime import date

from pydantic import BaseModel, field_validator


class SnapshotValueInput(BaseModel):
    """Input for single account value in snapshot"""

    account_id: int
    value: float


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
    """Snapshot value for single account"""

    id: int
    account_id: int
    account_name: str
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
