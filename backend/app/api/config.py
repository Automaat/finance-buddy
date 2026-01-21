from typing import Annotated

from fastapi import APIRouter, Depends, HTTPException
from sqlalchemy.orm import Session

from app.core.database import get_db
from app.schemas.config import ConfigCreate, ConfigResponse
from app.services.config import get_config, upsert_config

router = APIRouter(prefix="/api/config", tags=["config"])


@router.get("", response_model=ConfigResponse)
def get_app_config(db: Annotated[Session, Depends(get_db)]) -> ConfigResponse:
    """Get app configuration"""
    config = get_config(db)
    if not config:
        raise HTTPException(status_code=404, detail="Configuration not initialized")
    return config


@router.put("", response_model=ConfigResponse)
def update_app_config(
    data: ConfigCreate,
    db: Annotated[Session, Depends(get_db)],
) -> ConfigResponse:
    """Create or update app configuration"""
    return upsert_config(db, data)
