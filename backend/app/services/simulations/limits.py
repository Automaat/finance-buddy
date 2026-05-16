from sqlalchemy import select
from sqlalchemy.orm import Session

from app.models.retirement_limit import RetirementLimit

# Fallback contribution limits (2026 values, used when RetirementLimit table is empty)
DEFAULT_IKE_LIMIT = 28260  # Annual IKE contribution limit (PLN)
DEFAULT_IKZE_LIMIT = 11304  # Annual IKZE contribution limit (PLN)


def get_limit_for_year(db: Session, year: int, wrapper: str, owner: str) -> float:
    """Fetch contribution limit from RetirementLimit table"""
    limit = db.execute(
        select(RetirementLimit).where(
            RetirementLimit.year == year,
            RetirementLimit.account_wrapper == wrapper,
            RetirementLimit.owner == owner,
        )
    ).scalar_one_or_none()

    if not limit:
        # Fallback to defaults if limit not found in database
        if wrapper == "IKE":
            return DEFAULT_IKE_LIMIT
        if wrapper == "IKZE":
            return DEFAULT_IKZE_LIMIT
        return 0

    return float(limit.limit_amount)
