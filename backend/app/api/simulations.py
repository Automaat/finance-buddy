from fastapi import APIRouter, Depends
from sqlalchemy import select
from sqlalchemy.orm import Session

from app.core.database import get_db
from app.models.app_config import AppConfig
from app.schemas.simulations import SimulationInputs, SimulationResponse
from app.services import simulations as sim_service

router = APIRouter(prefix="/api/simulations", tags=["simulations"])


@router.post("/retirement", response_model=SimulationResponse)
def simulate_retirement(
    inputs: SimulationInputs, db: Session = Depends(get_db)
) -> SimulationResponse:
    """Calculate retirement account projections"""
    return sim_service.run_simulation(db, inputs)


@router.get("/prefill")
def get_prefill_data(db: Session = Depends(get_db)):
    """Get current balances and config for form prefill"""
    balances = sim_service.fetch_current_balances(db)
    config = db.execute(select(AppConfig)).scalar_one()
    current_age = sim_service.get_age_from_config(db)

    return {
        "current_age": current_age,
        "retirement_age": config.retirement_age,
        "balances": balances,
    }
