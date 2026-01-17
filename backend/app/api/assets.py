from fastapi import APIRouter, Depends
from sqlalchemy.orm import Session

from app.core.database import get_db
from app.schemas.assets import AssetCreate, AssetResponse, AssetsListResponse, AssetUpdate
from app.services import assets

router = APIRouter(prefix="/api/assets", tags=["assets"])


@router.get("", response_model=AssetsListResponse)
def get_assets(db: Session = Depends(get_db)) -> AssetsListResponse:  # noqa: B008
    """Get all active assets with their latest snapshot values"""
    return assets.get_all_assets(db)


@router.post("", response_model=AssetResponse, status_code=201)
def create_asset(data: AssetCreate, db: Session = Depends(get_db)) -> AssetResponse:  # noqa: B008
    """Create new asset"""
    return assets.create_asset(db, data)


@router.put("/{asset_id}", response_model=AssetResponse)
def update_asset(
    asset_id: int,
    data: AssetUpdate,
    db: Session = Depends(get_db),  # noqa: B008
) -> AssetResponse:
    """Update existing asset"""
    return assets.update_asset(db, asset_id, data)


@router.delete("/{asset_id}", status_code=204)
def delete_asset(asset_id: int, db: Session = Depends(get_db)) -> None:  # noqa: B008
    """Delete asset (soft delete by setting is_active=False)"""
    assets.delete_asset(db, asset_id)
