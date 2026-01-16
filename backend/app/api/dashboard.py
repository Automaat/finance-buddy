from fastapi import APIRouter, Depends
from sqlalchemy.orm import Session

from app.core.database import get_db
from app.schemas.dashboard import DashboardResponse
from app.services.dashboard import get_dashboard_data

router = APIRouter(prefix="/api/dashboard", tags=["dashboard"])


@router.get("", response_model=DashboardResponse)
def get_dashboard(db: Session = Depends(get_db)) -> DashboardResponse:  # noqa: B008
    """Get dashboard data with net worth history and allocation"""
    return get_dashboard_data(db)
