from typing import Annotated

from fastapi import APIRouter, Depends, Query
from sqlalchemy.orm import Session

from app.core.database import get_db
from app.schemas.zus import ZusCalculatorInputs, ZusCalculatorResponse, ZusPrefillResponse
from app.services import zus_calculator

router = APIRouter(prefix="/api/zus", tags=["zus"])


@router.post("/calculate", response_model=ZusCalculatorResponse)
def calculate_zus_pension(inputs: ZusCalculatorInputs) -> ZusCalculatorResponse:
    """Calculate projected ZUS retirement pension."""
    return zus_calculator.calculate_zus_pension(inputs)


@router.get("/prefill", response_model=ZusPrefillResponse)
def get_zus_prefill(
    db: Annotated[Session, Depends(get_db)],
    owner: str | None = Query(None),
) -> ZusPrefillResponse:
    """Prefill ZUS calculator form from salary records and config."""
    return zus_calculator.get_zus_prefill(db, owner)
