"""Metric-card calculations for the dashboard (savings rate, ratios, hourly costs)."""

import numpy as np
import pandas as pd
from sqlalchemy import select
from sqlalchemy.orm import Session

from app.models import AppConfig, SalaryRecord

# Time constants for hourly cost calculations
MONTHLY_WORK_HOURS = 160  # Standard monthly work hours
MONTHLY_LIFE_HOURS = 730  # Total hours per month (24h × 365d / 12m)


def _calculate_savings_rate(
    snapshots_df: pd.DataFrame, df: pd.DataFrame, db: Session
) -> float | None:
    """
    Calculate average monthly savings rate over last 3 months.
    Formula: (avg_monthly_net_worth_delta / avg_monthly_gross_salary) × 100

    Requires: last 4 snapshots (to calculate 3 deltas) and 3+ salary records
    """
    # Need at least 4 snapshots to calculate 3 deltas
    if len(snapshots_df) < 4:
        return None

    # Vectorized signed-value (recompute defensively — direct callers may pass a raw df)
    last_4_ids = snapshots_df.tail(4)["id"].tolist()
    sub = df[df["snapshot_id"].isin(last_4_ids)].copy()
    if "signed_value" not in sub.columns:
        has_name = sub["name"] if "name" in sub.columns else pd.Series(False, index=sub.index)
        asset_mask = sub["asset_id"].notna() & has_name.notna()
        account_mask = sub["account_id"].notna() & sub["type"].notna()
        sign = np.where(
            account_mask,
            np.where(sub["type"] == "asset", 1, -1),
            np.where(asset_mask, 1, 0),
        )
        sub["signed_value"] = sub["value"].astype(float) * sign

    nw_by_snap = sub.groupby("snapshot_id")["signed_value"].sum()
    net_worth_values = [float(nw_by_snap.get(sid, 0.0)) for sid in last_4_ids]

    deltas = [
        net_worth_values[i] - net_worth_values[i - 1] for i in range(1, len(net_worth_values))
    ]
    avg_delta = sum(deltas) / len(deltas)

    # Get last 3 salary records
    salaries = (
        db.execute(
            select(SalaryRecord)
            .where(SalaryRecord.is_active.is_(True))
            .order_by(SalaryRecord.date.desc())
            .limit(3)
        )
        .scalars()
        .all()
    )

    if not salaries or len(salaries) < 3:
        return None

    avg_salary = sum(float(s.gross_amount) for s in salaries) / len(salaries)

    if avg_salary == 0:
        return None

    return (avg_delta / avg_salary) * 100


def _get_latest_active_salary(db: Session) -> SalaryRecord | None:
    """Get latest active salary record."""
    return (
        db.execute(
            select(SalaryRecord)
            .where(SalaryRecord.is_active.is_(True))
            .order_by(SalaryRecord.date.desc())
        )
        .scalars()
        .first()
    )


def _calculate_debt_to_income(db: Session) -> float | None:
    """
    Calculate debt-to-income ratio.
    Formula: (monthly_mortgage_payment / latest_gross_salary) × 100
    """
    # Get app config
    config = db.execute(select(AppConfig).where(AppConfig.id == 1)).scalar_one_or_none()
    if not config or not config.monthly_mortgage_payment:
        return None

    # Get latest salary
    latest_salary = _get_latest_active_salary(db)

    if not latest_salary or latest_salary.gross_amount == 0:
        return None

    return (float(config.monthly_mortgage_payment) / float(latest_salary.gross_amount)) * 100


def _calculate_hour_of_work_cost(db: Session) -> float | None:
    """Calculate cost of one work hour (gross_salary / 160h)"""
    latest_salary = _get_latest_active_salary(db)

    if not latest_salary or latest_salary.gross_amount == 0:
        return None

    return float(latest_salary.gross_amount) / MONTHLY_WORK_HOURS


def _calculate_hour_of_life_cost(db: Session) -> float | None:
    """Calculate cost of one life hour (gross_salary / 730h)"""
    latest_salary = _get_latest_active_salary(db)

    if not latest_salary or latest_salary.gross_amount == 0:
        return None

    return float(latest_salary.gross_amount) / MONTHLY_LIFE_HOURS
