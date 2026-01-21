from typing import Annotated

from fastapi import APIRouter, Depends
from sqlalchemy import select
from sqlalchemy.orm import Session

from app.core.database import get_db
from app.models.app_config import AppConfig
from app.schemas.simulations import (
    PrefillBalances,
    PrefillResponse,
    SimulationInputs,
    SimulationResponse,
)
from app.services import simulations as sim_service

router = APIRouter(prefix="/api/simulations", tags=["simulations"])


@router.post("/retirement", response_model=SimulationResponse)
def simulate_retirement(
    inputs: SimulationInputs,
    db: Annotated[Session, Depends(get_db)],
) -> SimulationResponse:
    """Calculate retirement account projections"""
    return sim_service.run_simulation(db, inputs)


@router.get("/prefill", response_model=PrefillResponse)
def get_prefill_data(db: Annotated[Session, Depends(get_db)]) -> PrefillResponse:
    """Get current balances and config for form prefill"""
    balances_dict = sim_service.fetch_current_balances(db)
    config = db.execute(select(AppConfig)).scalar_one_or_none()
    current_age = sim_service.get_age_from_config(db)

    # Provide defaults if config not initialized
    retirement_age = config.retirement_age if config else 67

    # Fetch PPK rates from AppConfig
    ppk_rates = {}
    if config:
        ppk_rates = {
            "marcin": {
                "employee": float(config.ppk_employee_rate_marcin),
                "employer": float(config.ppk_employer_rate_marcin),
            },
            "ewa": {
                "employee": float(config.ppk_employee_rate_ewa),
                "employer": float(config.ppk_employer_rate_ewa),
            },
        }

    # Fetch latest salary records
    from app.models.salary_record import SalaryRecord

    salary_marcin = (
        db.execute(
            select(SalaryRecord)
            .where(SalaryRecord.owner == "Marcin", SalaryRecord.is_active.is_(True))
            .order_by(SalaryRecord.date.desc())
        )
        .scalars()
        .first()
    )

    salary_ewa = (
        db.execute(
            select(SalaryRecord)
            .where(SalaryRecord.owner == "Ewa", SalaryRecord.is_active.is_(True))
            .order_by(SalaryRecord.date.desc())
        )
        .scalars()
        .first()
    )

    monthly_salaries = {
        "marcin": float(salary_marcin.gross_amount) if salary_marcin else None,
        "ewa": float(salary_ewa.gross_amount) if salary_ewa else None,
    }

    # Fetch PPK balances
    ppk_balances = sim_service.fetch_ppk_balances(db)

    return PrefillResponse(
        current_age=current_age,
        retirement_age=retirement_age,
        balances=PrefillBalances(**balances_dict),
        ppk_rates=ppk_rates,
        monthly_salaries=monthly_salaries,
        ppk_balances=ppk_balances,
    )
