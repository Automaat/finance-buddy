from sqlalchemy import select
from sqlalchemy.orm import Session

from app.models.app_config import AppConfig
from app.schemas.config import ConfigCreate, ConfigResponse


def get_config(db: Session) -> ConfigResponse | None:
    """Get app configuration (single record with id=1)"""
    config = db.execute(select(AppConfig).where(AppConfig.id == 1)).scalar_one_or_none()
    if not config:
        return None
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
        monthly_expenses=config.monthly_expenses,
        monthly_mortgage_payment=config.monthly_mortgage_payment,
    )


def upsert_config(db: Session, data: ConfigCreate) -> ConfigResponse:
    """Create or update app configuration.

    This function implements a singleton pattern for app configuration:
    - There is always exactly one config record with id=1
    - If config exists, it updates the existing record
    - If config doesn't exist, it creates a new record with id=1
    - This ensures consistent configuration state across the application

    Args:
        db: Database session
        data: Configuration data to save

    Returns:
        ConfigResponse with saved configuration
    """
    # Fetch AppConfig directly for update operations
    config = db.execute(select(AppConfig).where(AppConfig.id == 1)).scalar_one_or_none()

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
        config.monthly_expenses = data.monthly_expenses
        config.monthly_mortgage_payment = data.monthly_mortgage_payment
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
            monthly_expenses=data.monthly_expenses,
            monthly_mortgage_payment=data.monthly_mortgage_payment,
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
        monthly_expenses=config.monthly_expenses,
        monthly_mortgage_payment=config.monthly_mortgage_payment,
    )
