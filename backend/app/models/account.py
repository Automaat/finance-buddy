from datetime import UTC, datetime

from sqlalchemy import Boolean, String
from sqlalchemy.orm import Mapped, mapped_column

from app.core.database import Base


class Account(Base):
    __tablename__ = "accounts"

    id: Mapped[int] = mapped_column(primary_key=True)
    name: Mapped[str] = mapped_column(String(255))
    type: Mapped[str] = mapped_column(String(50))
    category: Mapped[str] = mapped_column(String(100))
    owner: Mapped[str] = mapped_column(String(100))
    currency: Mapped[str] = mapped_column(String(10))
    account_wrapper: Mapped[str | None] = mapped_column(String(50), nullable=True, default=None)
    purpose: Mapped[str] = mapped_column(String(50))
    is_active: Mapped[bool] = mapped_column(Boolean, default=True)
    created_at: Mapped[datetime] = mapped_column(default=lambda: datetime.now(UTC))
