from typing import Annotated

from fastapi import APIRouter, Depends
from sqlalchemy.orm import Session

from app.core.database import get_db
from app.schemas.investment import CategoryStatsResponse
from app.services import investment

router = APIRouter(prefix="/api/investment", tags=["investment"])


@router.get("/stock-stats", response_model=CategoryStatsResponse)
def get_stock_stats(
    db: Annotated[Session, Depends(get_db)],
) -> CategoryStatsResponse:
    """Get aggregate statistics for stock investments"""
    return investment.get_category_stats(db, "stock")


@router.get("/bond-stats", response_model=CategoryStatsResponse)
def get_bond_stats(
    db: Annotated[Session, Depends(get_db)],
) -> CategoryStatsResponse:
    """Get aggregate statistics for bond investments"""
    return investment.get_category_stats(db, "bond")
