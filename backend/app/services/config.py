from sqlalchemy import select
from sqlalchemy.orm import Session

from app.models.app_config import AppConfig
from app.schemas.config import ConfigCreate, ConfigResponse


def get_config(db: Session) -> AppConfig | None:
    """Get app configuration (single record with id=1)"""
    return db.execute(select(AppConfig).where(AppConfig.id == 1)).scalar_one_or_none()


def upsert_config(db: Session, data: ConfigCreate) -> ConfigResponse:
    """Create or update app configuration (always id=1)"""
    config = get_config(db)

    if config:
        # Update existing config
        config.birth_date = data.birth_date
        config.retirement_age = data.retirement_age
        config.retirement_monthly_salary = data.retirement_monthly_salary
        config.allocation_real_estate = data.allocation_real_estate
        config.allocation_stocks = data.allocation_stocks
        config.allocation_bonds = data.allocation_bonds
        config.allocation_gold = data.allocation_gold
        config.allocation_commodities = data.allocation_commodities
    else:
        # Create new config with id=1
        config = AppConfig(
            id=1,
            birth_date=data.birth_date,
            retirement_age=data.retirement_age,
            retirement_monthly_salary=data.retirement_monthly_salary,
            allocation_real_estate=data.allocation_real_estate,
            allocation_stocks=data.allocation_stocks,
            allocation_bonds=data.allocation_bonds,
            allocation_gold=data.allocation_gold,
            allocation_commodities=data.allocation_commodities,
        )
        db.add(config)

    db.commit()
    db.refresh(config)

    return ConfigResponse(
        id=config.id,
        birth_date=config.birth_date,
        retirement_age=config.retirement_age,
        retirement_monthly_salary=config.retirement_monthly_salary,
        allocation_real_estate=config.allocation_real_estate,
        allocation_stocks=config.allocation_stocks,
        allocation_bonds=config.allocation_bonds,
        allocation_gold=config.allocation_gold,
        allocation_commodities=config.allocation_commodities,
    )
