from typing import Annotated

from fastapi import APIRouter, Depends
from sqlalchemy import select
from sqlalchemy.orm import Session

from app.core.database import get_db
from app.models.app_config import AppConfig
from app.models.persona import Persona
from app.models.salary_record import SalaryRecord
from app.schemas.simulations import (
    MortgageVsInvestInputs,
    MortgageVsInvestResponse,
    PrefillBalances,
    PrefillResponse,
    SimulationInputs,
    SimulationResponse,
)
from app.services import simulations as sim_service

router = APIRouter(prefix="/api/simulations", tags=["simulations"])


@router.post("/mortgage-vs-invest", response_model=MortgageVsInvestResponse)
def simulate_mortgage_vs_invest(inputs: MortgageVsInvestInputs) -> MortgageVsInvestResponse:
    """Compare overpaying mortgage vs investing extra amount"""
    return sim_service.simulate_mortgage_vs_invest(inputs)


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

    retirement_age = config.retirement_age if config else 67

    # Fetch PPK rates from Persona table
    personas = db.execute(select(Persona).order_by(Persona.name)).scalars().all()
    ppk_rates = {
        p.name.lower(): {
            "employee": float(p.ppk_employee_rate),
            "employer": float(p.ppk_employer_rate),
        }
        for p in personas
    }

    # Fetch latest salary records per persona
    monthly_salaries: dict[str, float | None] = {}
    for persona in personas:
        salary = (
            db.execute(
                select(SalaryRecord)
                .where(SalaryRecord.owner == persona.name, SalaryRecord.is_active.is_(True))
                .order_by(SalaryRecord.date.desc())
            )
            .scalars()
            .first()
        )
        monthly_salaries[persona.name.lower()] = float(salary.gross_amount) if salary else None

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
