from datetime import date

from pydantic import BaseModel


class SnapshotValueInput(BaseModel):
    """Input for single account value in snapshot"""

    account_id: int
    value: float


class SnapshotCreate(BaseModel):
    """Request to create new snapshot"""

    date: date
    notes: str | None = None
    values: list[SnapshotValueInput]


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
