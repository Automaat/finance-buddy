from typing import Annotated

from fastapi import APIRouter, Depends, Query
from sqlalchemy.orm import Session

from app.core.database import get_db
from app.schemas.company_valuations import (
    CompanyValuationCreate,
    CompanyValuationResponse,
    CompanyValuationsListResponse,
    CompanyValuationUpdate,
)
from app.services import company_valuations

router = APIRouter(prefix="/api", tags=["equity"])


@router.get("/company-valuations", response_model=CompanyValuationsListResponse)
def get_all_company_valuations(
    db: Annotated[Session, Depends(get_db)],
    company: str | None = Query(None),
) -> CompanyValuationsListResponse:
    return company_valuations.get_all_company_valuations(db, company)


@router.get("/company-valuations/{valuation_id}", response_model=CompanyValuationResponse)
def get_company_valuation(
    valuation_id: int,
    db: Annotated[Session, Depends(get_db)],
) -> CompanyValuationResponse:
    return company_valuations.get_company_valuation(db, valuation_id)


@router.post("/company-valuations", response_model=CompanyValuationResponse, status_code=201)
def create_company_valuation(
    data: CompanyValuationCreate,
    db: Annotated[Session, Depends(get_db)],
) -> CompanyValuationResponse:
    return company_valuations.create_company_valuation(db, data)


@router.patch("/company-valuations/{valuation_id}", response_model=CompanyValuationResponse)
def update_company_valuation(
    valuation_id: int,
    data: CompanyValuationUpdate,
    db: Annotated[Session, Depends(get_db)],
) -> CompanyValuationResponse:
    return company_valuations.update_company_valuation(db, valuation_id, data)


@router.delete("/company-valuations/{valuation_id}", status_code=204)
def delete_company_valuation(
    valuation_id: int,
    db: Annotated[Session, Depends(get_db)],
) -> None:
    company_valuations.delete_company_valuation(db, valuation_id)
