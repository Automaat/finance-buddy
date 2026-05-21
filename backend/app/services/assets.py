from fastapi import HTTPException
from sqlalchemy import desc, select
from sqlalchemy.exc import IntegrityError
from sqlalchemy.orm import Session

from app.models import Asset, Snapshot, SnapshotValue
from app.schemas.assets import AssetCreate, AssetResponse, AssetsListResponse, AssetUpdate
from app.services.snapshot_aggregates import recompute_for_snapshots
from app.utils.db_helpers import (
    check_duplicate_name,
    get_latest_snapshot_values_batch_for_assets,
    get_or_404,
)


def _affected_snapshot_ids_for_asset(db: Session, asset_id: int) -> list[int]:
    """Return snapshot IDs that have a SnapshotValue for this asset."""
    return list(
        db.execute(
            select(SnapshotValue.snapshot_id).where(SnapshotValue.asset_id == asset_id).distinct()
        )
        .scalars()
        .all()
    )


def get_all_assets(db: Session) -> AssetsListResponse:
    """Get all active assets with their latest snapshot values"""
    assets = db.execute(select(Asset).where(Asset.is_active.is_(True))).scalars().all()

    asset_ids = [asset.id for asset in assets]
    latest_values = get_latest_snapshot_values_batch_for_assets(db, asset_ids)

    asset_responses = []
    for asset in assets:
        asset_response = AssetResponse(
            id=asset.id,
            name=asset.name,
            is_active=asset.is_active,
            created_at=asset.created_at,
            current_value=latest_values.get(asset.id, 0.0),
        )
        asset_responses.append(asset_response)

    return AssetsListResponse(assets=asset_responses)


def create_asset(db: Session, data: AssetCreate) -> AssetResponse:
    """Create new asset"""
    check_duplicate_name(db, Asset, data.name)

    asset = Asset(
        name=data.name,
        is_active=True,
    )

    try:
        db.add(asset)
        db.commit()
        db.refresh(asset)
    except IntegrityError as e:
        db.rollback()
        raise HTTPException(
            status_code=500,
            detail="Failed to create asset due to database integrity error",
        ) from e

    return AssetResponse(
        id=asset.id,
        name=asset.name,
        is_active=asset.is_active,
        created_at=asset.created_at,
        current_value=0.0,
    )


def update_asset(db: Session, asset_id: int, data: AssetUpdate) -> AssetResponse:
    """Update existing asset (name-only; no aggregate recompute needed)"""
    asset = get_or_404(db, Asset, asset_id)

    if data.name and data.name != asset.name:
        check_duplicate_name(db, Asset, data.name, exclude_id=asset_id)

    if data.name is not None:
        asset.name = data.name

    try:
        db.commit()
        db.refresh(asset)
    except IntegrityError as e:
        db.rollback()
        raise HTTPException(
            status_code=500,
            detail="Failed to update asset due to database integrity error",
        ) from e

    latest_snapshot = db.execute(
        select(Snapshot).order_by(desc(Snapshot.date)).limit(1)
    ).scalar_one_or_none()

    current_value = 0.0
    if latest_snapshot:
        snapshot_value = db.execute(
            select(SnapshotValue).where(
                SnapshotValue.snapshot_id == latest_snapshot.id,
                SnapshotValue.asset_id == asset.id,
            )
        ).scalar_one_or_none()
        if snapshot_value:
            current_value = float(snapshot_value.value)

    return AssetResponse(
        id=asset.id,
        name=asset.name,
        is_active=asset.is_active,
        created_at=asset.created_at,
        current_value=current_value,
    )


def delete_asset(db: Session, asset_id: int) -> None:
    """Soft delete asset and recompute affected aggregates"""
    asset = get_or_404(db, Asset, asset_id)

    if not asset.is_active:
        return

    affected_ids = _affected_snapshot_ids_for_asset(db, asset_id)

    asset.is_active = False
    db.flush()  # Persist is_active=False before recompute queries assets

    if affected_ids:
        recompute_for_snapshots(db, affected_ids)

    db.commit()
