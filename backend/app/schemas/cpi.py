from datetime import date

from pydantic import BaseModel


class CpiPoint(BaseModel):
    year: int
    yoy_rate: float
    cumulative_index: float


class CpiSeriesResponse(BaseModel):
    points: list[CpiPoint]
    base_year: int | None
    latest_year: int | None
    source: str = "GUS-BDL-217230"


class AdjustRequest(BaseModel):
    amount: float
    from_date: date
    to_date: date


class AdjustResponse(BaseModel):
    original_amount: float
    adjusted_amount: float
    factor: float
    from_date: date
    to_date: date
    as_of_year: int


class RefreshResponse(BaseModel):
    rows_written: int
    latest_year: int | None
