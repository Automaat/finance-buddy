from datetime import UTC, datetime
from decimal import Decimal

from sqlalchemy import DateTime, Numeric, String
from sqlalchemy.orm import Mapped, mapped_column

from app.core.database import Base


class CpiIndex(Base):
    """Annual Polish CPI (year-over-year rate) from GUS BDL variable 217230.

    ``yoy_rate`` is the value as published by GUS (e.g. 114.4 means the price
    level rose 14.4% versus the previous year). Fixed-base indices and
    arbitrary-date inflation factors are derived at query time in
    ``app.services.inflation`` — keeping raw data here makes refreshes
    idempotent and re-derivation trivial.
    """

    __tablename__ = "cpi_index"

    year: Mapped[int] = mapped_column(primary_key=True)
    yoy_rate: Mapped[Decimal] = mapped_column(Numeric(8, 4))
    source: Mapped[str] = mapped_column(String(64), default="GUS-BDL-217230")
    fetched_at: Mapped[datetime] = mapped_column(
        DateTime(timezone=True), default=lambda: datetime.now(UTC)
    )
