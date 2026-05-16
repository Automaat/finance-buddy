"""Dashboard service.

Public API is re-exported here so callers keep importing from
``app.services.dashboard`` after the split into focused modules. The metric
helpers are re-exported because the dashboard test suite imports them directly.
"""

from app.services.dashboard.metrics import (
    _calculate_debt_to_income,
    _calculate_hour_of_life_cost,
    _calculate_hour_of_work_cost,
    _calculate_savings_rate,
)
from app.services.dashboard.service import get_dashboard_data

__all__ = [
    "_calculate_debt_to_income",
    "_calculate_hour_of_life_cost",
    "_calculate_hour_of_work_cost",
    "_calculate_savings_rate",
    "get_dashboard_data",
]
