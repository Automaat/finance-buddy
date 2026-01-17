from fastapi import HTTPException
from sqlalchemy import desc, select
from sqlalchemy.orm import Session

from app.models import Asset, Snapshot, SnapshotValue
from app.schemas.assets import AssetCreate, AssetResponse, AssetsListResponse, AssetUpdate


def get_all_assets(db: Session) -> AssetsListResponse:
    """Get all active assets with their latest snapshot values"""
    # Get all active assets
    assets = db.execute(select(Asset).where(Asset.is_active.is_(True))).scalars().all()

    # Get latest snapshot
    latest_snapshot = db.execute(
        select(Snapshot).order_by(desc(Snapshot.date)).limit(1)
    ).scalar_one_or_none()

    # Get latest values if snapshot exists
    latest_values = {}
    if latest_snapshot:
        values = db.execute(
            select(SnapshotValue).where(
                SnapshotValue.snapshot_id == latest_snapshot.id,
                SnapshotValue.asset_id.is_not(None),
            )
        ).scalars()
        latest_values = {sv.asset_id: float(sv.value) for sv in values}

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
    existing = (
        db.execute(
            select(Asset).where(
                Asset.name == data.name,
                Asset.is_active.is_(True),
            )
        )
        .scalars()
        .first()
    )
    if existing:
        raise HTTPException(status_code=400, detail=f"Active asset '{data.name}' already exists")

    asset = Asset(
        name=data.name,
        is_active=True,
    )
    db.add(asset)
    db.commit()
    db.refresh(asset)

    return AssetResponse(
        id=asset.id,
        name=asset.name,
        is_active=asset.is_active,
        created_at=asset.created_at,
        current_value=0.0,
    )


def update_asset(db: Session, asset_id: int, data: AssetUpdate) -> AssetResponse:
    """Update existing asset"""
    asset = db.execute(select(Asset).where(Asset.id == asset_id)).scalar_one_or_none()

    if not asset:
        raise HTTPException(status_code=404, detail=f"Asset with id {asset_id} not found")

    # Check for duplicate name if changing name
    if data.name and data.name != asset.name:
        existing = (
            db.execute(
                select(Asset).where(
                    Asset.name == data.name,
                    Asset.is_active.is_(True),
                    Asset.id != asset_id,
                )
            )
            .scalars()
            .first()
        )
        if existing:
            raise HTTPException(
                status_code=400, detail=f"Active asset '{data.name}' already exists"
            )

    # Update fields
    if data.name is not None:
        asset.name = data.name

    db.commit()
    db.refresh(asset)

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
    asset = db.execute(select(Asset).where(Asset.id == asset_id)).scalar_one_or_none()

    if not asset:
        raise HTTPException(status_code=404, detail=f"Asset with id {asset_id} not found")

    # Idempotent: if already deleted, return early
    if not asset.is_active:
        return

    asset.is_active = False
    db.commit()
