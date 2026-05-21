from datetime import UTC, date, datetime
from decimal import Decimal

from sqlalchemy import Boolean, Numeric, String
from sqlalchemy.orm import Mapped, mapped_column

from app.core.database import Base


class CompanyValuation(Base):
    __tablename__ = "company_valuations"

    id: Mapped[int] = mapped_column(primary_key=True)
    company: Mapped[str] = mapped_column(String(200))
    date: Mapped[date]
    currency: Mapped[str] = mapped_column(String(3), default="USD")

    fmv_per_share: Mapped[Decimal] = mapped_column(Numeric(15, 4))
    fmv_low: Mapped[Decimal | None] = mapped_column(Numeric(15, 4), nullable=True)
    fmv_high: Mapped[Decimal | None] = mapped_column(Numeric(15, 4), nullable=True)

    source: Mapped[str] = mapped_column(String(30))
    common_stock_discount_pct: Mapped[Decimal | None] = mapped_column(Numeric(5, 2), nullable=True)

    notes: Mapped[str | None] = mapped_column(String(500), default=None)
    is_active: Mapped[bool] = mapped_column(Boolean, default=True)
    created_at: Mapped[datetime] = mapped_column(default=lambda: datetime.now(UTC))
