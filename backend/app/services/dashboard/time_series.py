"""Investment time-series builders for the dashboard (overall, per-wrapper, per-category)."""

import numpy as np
import pandas as pd

from app.schemas.dashboard import CategoryTimeSeries, InvestmentTimeSeriesPoint, WrapperTimeSeries

INVESTMENT_CATEGORIES = {"stock", "bond", "fund", "etf", "gold", "ppk"}


def _cum_contributions_per_snapshot(trans_df: pd.DataFrame, snap_dates: np.ndarray) -> np.ndarray:
    """For each snapshot date, return cumulative signed_amount of transactions on/before it.

    Uses sorted transactions + cumsum + searchsorted — O((T + S) log T) instead
    of O(T * S) for the naïve per-snapshot filter.
    """
    if trans_df.empty:
        return np.zeros(len(snap_dates))

    sorted_trans = trans_df.sort_values("date")
    t_dates = sorted_trans["date"].to_numpy()
    cum = np.cumsum(sorted_trans["signed_amount"].astype(float).to_numpy())

    # idx = number of transactions with date <= snap_date, minus 1
    idx = np.searchsorted(t_dates, snap_dates, side="right") - 1
    return np.where(idx >= 0, cum[np.clip(idx, 0, len(cum) - 1)], 0.0)


def _build_series(
    df: pd.DataFrame,
    snapshots_df: pd.DataFrame,
    transactions_with_accounts_df: pd.DataFrame,
    value_mask: pd.Series,
    trans_mask: pd.Series | None,
) -> list[InvestmentTimeSeriesPoint]:
    """Generic per-snapshot value + cumulative-contribution series builder."""
    if df.empty or snapshots_df.empty:
        return []

    snaps = snapshots_df.sort_values("date")
    snap_ids = snaps["id"].to_numpy()
    snap_dates_arr = snaps["date"].to_numpy()

    val_by_snap = df.loc[value_mask].groupby("snapshot_id")["value"].sum().astype(float)

    if trans_mask is not None and not transactions_with_accounts_df.empty:
        filtered_trans = transactions_with_accounts_df.loc[trans_mask]
        contributions = _cum_contributions_per_snapshot(filtered_trans, snap_dates_arr)
    else:
        contributions = np.zeros(len(snap_ids))

    series: list[InvestmentTimeSeriesPoint] = []
    for snap_id, snap_date, contrib in zip(snap_ids, snap_dates_arr, contributions, strict=True):
        value = float(val_by_snap.get(snap_id, 0.0))
        c = float(contrib)
        series.append(
            InvestmentTimeSeriesPoint(
                date=snap_date,
                value=value,
                contributions=c,
                returns=value - c,
            )
        )
    return series


def build_investment_time_series(
    df: pd.DataFrame, snapshots_df: pd.DataFrame, transactions_with_accounts_df: pd.DataFrame
) -> list[InvestmentTimeSeriesPoint]:
    """Cumulative investment value, contributions, and returns per snapshot."""
    if df.empty or snapshots_df.empty:
        return []

    value_mask = (
        df["account_id"].notna()
        & (df["type"] == "asset")
        & df["category"].isin(INVESTMENT_CATEGORIES)
    )
    if not transactions_with_accounts_df.empty:
        trans_mask = transactions_with_accounts_df["category"].isin(INVESTMENT_CATEGORIES)
    else:
        trans_mask = None

    return _build_series(df, snapshots_df, transactions_with_accounts_df, value_mask, trans_mask)


def build_wrapper_time_series(
    df: pd.DataFrame, snapshots_df: pd.DataFrame, transactions_with_accounts_df: pd.DataFrame
) -> WrapperTimeSeries:
    """Per-wrapper (IKE, IKZE, PPK) investment time series."""
    wrappers = {"IKE": [], "IKZE": [], "PPK": []}

    if df.empty or snapshots_df.empty:
        return WrapperTimeSeries(ike=[], ikze=[], ppk=[])

    base_value_mask = (
        df["account_id"].notna()
        & (df["type"] == "asset")
        & df["category"].isin(INVESTMENT_CATEGORIES)
    )
    if not transactions_with_accounts_df.empty:
        base_trans_mask = transactions_with_accounts_df["category"].isin(INVESTMENT_CATEGORIES)
    else:
        base_trans_mask = None

    for wrapper in wrappers:
        v_mask = base_value_mask & (df["account_wrapper"] == wrapper)
        t_mask = (
            base_trans_mask & (transactions_with_accounts_df["account_wrapper"] == wrapper)
            if base_trans_mask is not None
            else None
        )
        wrappers[wrapper] = _build_series(
            df, snapshots_df, transactions_with_accounts_df, v_mask, t_mask
        )

    return WrapperTimeSeries(ike=wrappers["IKE"], ikze=wrappers["IKZE"], ppk=wrappers["PPK"])


def build_category_time_series(
    df: pd.DataFrame, snapshots_df: pd.DataFrame, transactions_with_accounts_df: pd.DataFrame
) -> CategoryTimeSeries:
    """Per-category-group (stock, bond) investment time series."""
    if df.empty or snapshots_df.empty:
        return CategoryTimeSeries(stock=[], bond=[])

    # stock group includes stock/fund/etf; bond group is just bond
    group_categories = {
        "stock": ["stock", "fund", "etf"],
        "bond": ["bond"],
    }

    result: dict[str, list[InvestmentTimeSeriesPoint]] = {}
    for group, categories in group_categories.items():
        v_mask = (
            df["account_id"].notna() & (df["type"] == "asset") & df["category"].isin(categories)
        )
        if not transactions_with_accounts_df.empty:
            t_mask = transactions_with_accounts_df["category"].isin(categories)
        else:
            t_mask = None
        result[group] = _build_series(
            df, snapshots_df, transactions_with_accounts_df, v_mask, t_mask
        )

    return CategoryTimeSeries(stock=result["stock"], bond=result["bond"])
