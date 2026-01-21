from typing import Annotated

from fastapi import APIRouter, Depends
from sqlalchemy.orm import Session

from app.core.database import get_db
from app.schemas.dashboard import DashboardResponse
from app.services.dashboard import get_dashboard_data

router = APIRouter(prefix="/api/dashboard", tags=["dashboard"])


@router.get("", response_model=DashboardResponse)
def get_dashboard(db: Annotated[Session, Depends(get_db)]) -> DashboardResponse:
    """Get dashboard data with net worth history and allocation"""
    return get_dashboard_data(db)
