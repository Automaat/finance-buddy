from fastapi import APIRouter, Depends, HTTPException
from sqlalchemy.orm import Session

from app.core.database import get_db
from app.schemas.config import ConfigCreate, ConfigResponse
from app.services.config import get_config, upsert_config

router = APIRouter(prefix="/api/config", tags=["config"])


@router.get("", response_model=ConfigResponse)
def get_app_config(db: Session = Depends(get_db)) -> ConfigResponse:  # noqa: B008
    """Get app configuration"""
    config = get_config(db)
    if not config:
        raise HTTPException(status_code=404, detail="Configuration not initialized")

    return ConfigResponse(
        id=config.id,
        birth_date=config.birth_date,
        retirement_age=config.retirement_age,
        retirement_monthly_salary=config.retirement_monthly_salary,
        allocation_real_estate=config.allocation_real_estate,
        allocation_stocks=config.allocation_stocks,
        allocation_bonds=config.allocation_bonds,
        allocation_gold=config.allocation_gold,
        allocation_commodities=config.allocation_commodities,
    )


@router.put("", response_model=ConfigResponse)
def update_app_config(
    data: ConfigCreate, db: Session = Depends(get_db)  # noqa: B008
) -> ConfigResponse:
    """Create or update app configuration"""
    return upsert_config(db, data)
