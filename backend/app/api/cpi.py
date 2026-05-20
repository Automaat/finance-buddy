from fastapi import APIRouter, Depends, HTTPException
from sqlalchemy.orm import Session

from app.core.database import get_db
from app.schemas.cpi import (
    AdjustRequest,
    AdjustResponse,
    CpiPoint,
    CpiSeriesResponse,
    RefreshResponse,
)
from app.services import inflation

router = APIRouter(prefix="/api/cpi", tags=["cpi"])


@router.get("/series", response_model=CpiSeriesResponse)
def get_cpi_series(db: Session = Depends(get_db)) -> CpiSeriesResponse:
    """Return the full annual CPI series with derived cumulative index."""
    rows = inflation.cpi_series(db)
    points = [
        CpiPoint(year=year, yoy_rate=float(yoy), cumulative_index=float(idx))
        for year, yoy, idx in rows
    ]
    return CpiSeriesResponse(
        points=points,
        base_year=points[0].year if points else None,
        latest_year=points[-1].year if points else None,
    )


@router.post("/adjust", response_model=AdjustResponse)
def adjust_amount(payload: AdjustRequest, db: Session = Depends(get_db)) -> AdjustResponse:
    """Inflate (or deflate) ``amount`` between two calendar dates."""
    try:
        adjusted = inflation.adjust(db, payload.amount, payload.from_date, payload.to_date)
    except inflation.InflationDataMissingError as exc:
        raise HTTPException(status_code=503, detail=str(exc)) from exc

    as_of_year = inflation.latest_known_year(db)
    if as_of_year is None:
        raise HTTPException(status_code=503, detail="CPI table is empty")

    factor = adjusted / payload.amount if payload.amount else 0.0
    return AdjustResponse(
        original_amount=payload.amount,
        adjusted_amount=adjusted,
        factor=factor,
        from_date=payload.from_date,
        to_date=payload.to_date,
        as_of_year=as_of_year,
    )


@router.post("/refresh", response_model=RefreshResponse)
async def refresh_cpi(db: Session = Depends(get_db)) -> RefreshResponse:
    """Pull fresh CPI from GUS BDL and upsert into the database.

    Intended for development and manual recovery. In normal operation the
    in-process scheduler refreshes monthly (and on startup if stale). This
    endpoint is reachable to anyone who can hit the backend — finance-buddy
    is a single-user self-hosted app, so the worst case is an unsolicited
    network call to a public GUS endpoint. Gate it behind auth before
    exposing the backend to untrusted networks.
    """
    try:
        written = await inflation.refresh_cpi(db)
    except Exception as exc:
        raise HTTPException(status_code=502, detail=f"GUS BDL fetch failed: {exc}") from exc
    return RefreshResponse(rows_written=written, latest_year=inflation.latest_known_year(db))
