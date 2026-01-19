from datetime import UTC, datetime
from typing import Annotated

from fastapi import APIRouter, Depends
from sqlalchemy.orm import Session

from app.core.database import get_db
from app.models import RetirementLimit
from app.schemas.retirement import (
    PPKContributionGenerateRequest,
    PPKContributionGenerateResponse,
    PPKStatsResponse,
    RetirementLimitCreate,
    RetirementLimitResponse,
    YearlyStatsResponse,
)
from app.services import retirement

router = APIRouter(prefix="/api/retirement", tags=["retirement"])


@router.get("/stats", response_model=list[YearlyStatsResponse])
def get_retirement_stats(
    db: Annotated[Session, Depends(get_db)],
    year: int | None = None,
    owner: str | None = None,
) -> list[YearlyStatsResponse]:
    """Get yearly contribution stats for all retirement accounts"""
    if year is None:
        year = datetime.now(UTC).year
    return retirement.get_yearly_stats(db, year, owner)


@router.get("/ppk-stats", response_model=list[PPKStatsResponse])
def get_ppk_stats(
    db: Annotated[Session, Depends(get_db)],
    owner: str | None = None,
) -> list[PPKStatsResponse]:
    """Get PPK contribution breakdown and returns per owner"""
    return retirement.get_ppk_stats(db, owner)


@router.post("/ppk-contributions/generate", response_model=PPKContributionGenerateResponse)
def generate_ppk_contributions(
    db: Annotated[Session, Depends(get_db)],
    data: PPKContributionGenerateRequest,
) -> PPKContributionGenerateResponse:
    """Generate monthly PPK contributions based on salary and configured rates"""
    return retirement.generate_ppk_contributions(db, data)


@router.get("/limits/{year}", response_model=list[RetirementLimitResponse])
def get_limits_for_year(
    db: Annotated[Session, Depends(get_db)],
    year: int,
) -> list[RetirementLimitResponse]:
    """Get all limits for a specific year"""
    limits = db.query(RetirementLimit).filter(RetirementLimit.year == year).all()
    return [
        RetirementLimitResponse(
            id=limit.id,
            year=limit.year,
            account_wrapper=limit.account_wrapper,
            owner=limit.owner,
            limit_amount=float(limit.limit_amount),
            notes=limit.notes,
        )
        for limit in limits
    ]


@router.put("/limits/{year}/{wrapper}/{owner}", response_model=RetirementLimitResponse)
def upsert_limit(
    db: Annotated[Session, Depends(get_db)],
    year: int,
    wrapper: str,
    owner: str,
    data: RetirementLimitCreate,
) -> RetirementLimitResponse:
    """Create or update retirement limit"""
    limit = retirement.update_limit(db, year, wrapper, owner, data.limit_amount, data.notes)
    return RetirementLimitResponse(
        id=limit.id,
        year=limit.year,
        account_wrapper=limit.account_wrapper,
        owner=limit.owner,
        limit_amount=float(limit.limit_amount),
        notes=limit.notes,
    )
