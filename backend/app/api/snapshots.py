from typing import Annotated

from fastapi import APIRouter, Depends
from sqlalchemy.orm import Session

from app.core.database import get_db
from app.schemas.snapshots import SnapshotCreate, SnapshotListItem, SnapshotResponse, SnapshotUpdate
from app.services import snapshots

router = APIRouter(prefix="/api/snapshots", tags=["snapshots"])


@router.post("", response_model=SnapshotResponse, status_code=201)
def create_snapshot(
    data: SnapshotCreate, db: Annotated[Session, Depends(get_db)]
) -> SnapshotResponse:
    """Create new snapshot with all account values"""
    return snapshots.create_snapshot(db, data)


@router.get("", response_model=list[SnapshotListItem])
def get_snapshots(db: Annotated[Session, Depends(get_db)]) -> list[SnapshotListItem]:
    """Get all snapshots ordered by date descending"""
    return snapshots.get_all_snapshots(db)


@router.get("/{snapshot_id}", response_model=SnapshotResponse)
def get_snapshot(snapshot_id: int, db: Annotated[Session, Depends(get_db)]) -> SnapshotResponse:
    """Get single snapshot with all account values"""
    return snapshots.get_snapshot_by_id(db, snapshot_id)


@router.put("/{snapshot_id}", response_model=SnapshotResponse)
def update_snapshot(
    snapshot_id: int, data: SnapshotUpdate, db: Annotated[Session, Depends(get_db)]
) -> SnapshotResponse:
    """Update existing snapshot"""
    return snapshots.update_snapshot(db, snapshot_id, data)
