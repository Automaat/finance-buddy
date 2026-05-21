"""Metric-card calculations for the dashboard (savings rate, ratios, hourly costs)."""

from datetime import timedelta

import numpy as np
import pandas as pd
from sqlalchemy import select
from sqlalchemy.orm import Session

from app.models import AppConfig, SalaryRecord
from app.schemas.dashboard import DeltaValue, TileDelta, TileDeltas

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
        # Asset-table rows require both asset_id and a joined name; with no name
        # column the join never happened, so no row qualifies as asset-table.
        if "name" in sub.columns:
            asset_mask = sub["asset_id"].notna() & sub["name"].notna()
        else:
            asset_mask = pd.Series(False, index=sub.index)
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


_EMPTY_TILE_DELTAS = TileDeltas(
    net_worth=TileDelta(mom=None, yoy=None),
    assets=TileDelta(mom=None, yoy=None),
    liabilities=TileDelta(mom=None, yoy=None),
)


def _per_snapshot_totals(df: pd.DataFrame) -> pd.DataFrame:
    """Aggregate (assets, liabilities, net_worth) per snapshot."""
    if df.empty:
        return pd.DataFrame(columns=["snapshot_id", "date", "assets", "liabilities", "net_worth"])

    name_present = df["name"].notna() if "name" in df.columns else pd.Series(False, index=df.index)
    asset_table_mask = df["asset_id"].notna() & name_present
    account_asset_mask = df["account_id"].notna() & (df["type"] == "asset")
    account_liab_mask = df["account_id"].notna() & (df["type"] == "liability")

    value = df["value"].astype(float)
    work = df[["snapshot_id", "date"]].copy()
    work["assets"] = value.where(asset_table_mask | account_asset_mask, 0.0)
    work["liabilities"] = value.where(account_liab_mask, 0.0)

    grouped = work.groupby(["snapshot_id", "date"])[["assets", "liabilities"]].sum().reset_index()
    grouped["net_worth"] = grouped["assets"] - grouped["liabilities"]
    return grouped.sort_values(["date", "snapshot_id"]).reset_index(drop=True)


def _pick_baseline(totals: pd.DataFrame, low: object, high: object) -> pd.Series | None:
    """Latest row with low <= date <= high; deterministic tie-break by snapshot_id desc."""
    window = totals[(totals["date"] >= low) & (totals["date"] <= high)]
    if window.empty:
        return None
    max_date = window["date"].max()
    same_day = window[window["date"] == max_date]
    return same_day.sort_values("snapshot_id", ascending=False).iloc[0]


def _delta(current: float, baseline: pd.Series | None, field: str) -> DeltaValue | None:
    if baseline is None:
        return None
    base = float(baseline[field])
    abs_change = current - base
    pct = (abs_change / abs(base)) * 100 if base != 0 else None
    return DeltaValue(absolute=float(abs_change), percentage=pct)


def compute_tile_deltas(df: pd.DataFrame, snapshots_df: pd.DataFrame) -> TileDeltas:
    """Compute MoM/YoY deltas for net worth, assets, liabilities.

    MoM window = [latest - 45d, latest - 15d]
    YoY window = [latest - 395d, latest - 335d]
    Latest snapshot is excluded from baseline candidates.
    """
    if df.empty or snapshots_df.empty:
        return _EMPTY_TILE_DELTAS

    totals = _per_snapshot_totals(df)
    if totals.empty:
        return _EMPTY_TILE_DELTAS

    current = totals.iloc[-1]
    prior = totals.iloc[:-1]
    latest_date = current["date"]

    mom_base = _pick_baseline(
        prior, latest_date - timedelta(days=45), latest_date - timedelta(days=15)
    )
    yoy_base = _pick_baseline(
        prior, latest_date - timedelta(days=395), latest_date - timedelta(days=335)
    )

    def _tile(field: str) -> TileDelta:
        cur = float(current[field])
        return TileDelta(mom=_delta(cur, mom_base, field), yoy=_delta(cur, yoy_base, field))

    return TileDeltas(
        net_worth=_tile("net_worth"),
        assets=_tile("assets"),
        liabilities=_tile("liabilities"),
    )
