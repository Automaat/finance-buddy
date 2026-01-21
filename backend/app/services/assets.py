from fastapi import HTTPException
from sqlalchemy import desc, select
from sqlalchemy.exc import IntegrityError
from sqlalchemy.orm import Session

from app.models import Asset, Snapshot, SnapshotValue
from app.schemas.assets import AssetCreate, AssetResponse, AssetsListResponse, AssetUpdate
from app.utils.db_helpers import (
    check_duplicate_name,
    get_latest_snapshot_values_batch_for_assets,
    get_or_404,
    soft_delete,
)


def get_all_assets(db: Session) -> AssetsListResponse:
    """Get all active assets with their latest snapshot values"""
    # Get all active assets
    assets = db.execute(select(Asset).where(Asset.is_active.is_(True))).scalars().all()

    # Batch fetch latest snapshot values for all assets
    asset_ids = [asset.id for asset in assets]
    latest_values = get_latest_snapshot_values_batch_for_assets(db, asset_ids)

    # Build response
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
    # Check for duplicate active asset name
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
        raise HTTPException(status_code=409, detail=f"Asset '{data.name}' already exists") from e

    return AssetResponse(
        id=asset.id,
        name=asset.name,
        is_active=asset.is_active,
        created_at=asset.created_at,
        current_value=0.0,
    )


def update_asset(db: Session, asset_id: int, data: AssetUpdate) -> AssetResponse:
    """Update existing asset"""
    asset = get_or_404(db, Asset, asset_id)

    # Check for duplicate name if changing name
    if data.name and data.name != asset.name:
        check_duplicate_name(db, Asset, data.name, exclude_id=asset_id)

    # Update fields
    if data.name is not None:
        asset.name = data.name

    try:
        db.commit()
        db.refresh(asset)
    except IntegrityError as e:
        db.rollback()
        raise HTTPException(
            status_code=409,
            detail=f"Asset '{data.name or asset.name}' conflicts with existing asset",
        ) from e

    # Get current value
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
    """Soft delete asset by setting is_active=False"""
    soft_delete(db, Asset, asset_id)
