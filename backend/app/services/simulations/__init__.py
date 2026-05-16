"""Retirement and mortgage simulation services.

Public API is re-exported here so callers keep importing from
``app.services.simulations`` after the split into focused modules.
"""

from app.services.simulations.accounts import (
    get_ppk_return_for_age,
    simulate_account,
    simulate_brokerage_account,
    simulate_ppk_account,
)
from app.services.simulations.limits import get_limit_for_year
from app.services.simulations.mortgage import simulate_mortgage_vs_invest
from app.services.simulations.prefill import (
    fetch_current_balances,
    fetch_ppk_balances,
    get_age_from_config,
)
from app.services.simulations.retirement import run_simulation

__all__ = [
    "fetch_current_balances",
    "fetch_ppk_balances",
    "get_age_from_config",
    "get_limit_for_year",
    "get_ppk_return_for_age",
    "run_simulation",
    "simulate_account",
    "simulate_brokerage_account",
    "simulate_mortgage_vs_invest",
    "simulate_ppk_account",
]
