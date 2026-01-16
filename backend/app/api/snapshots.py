from fastapi import APIRouter, Depends
from sqlalchemy.orm import Session

from app.core.database import get_db
from app.schemas.snapshots import SnapshotCreate, SnapshotListItem, SnapshotResponse
from app.services import snapshots

router = APIRouter(prefix="/api/snapshots", tags=["snapshots"])


@router.post("", response_model=SnapshotResponse, status_code=201)
def create_snapshot(data: SnapshotCreate, db: Session = Depends(get_db)) -> SnapshotResponse:  # noqa: B008
    """Create new snapshot with all account values"""
    return snapshots.create_snapshot(db, data)


@router.get("", response_model=list[SnapshotListItem])
def get_snapshots(db: Session = Depends(get_db)) -> list[SnapshotListItem]:  # noqa: B008
    """Get all snapshots ordered by date descending"""
    return snapshots.get_all_snapshots(db)


@router.get("/{snapshot_id}", response_model=SnapshotResponse)
def get_snapshot(snapshot_id: int, db: Session = Depends(get_db)) -> SnapshotResponse:  # noqa: B008
    """Get single snapshot with all account values"""
    return snapshots.get_snapshot_by_id(db, snapshot_id)
