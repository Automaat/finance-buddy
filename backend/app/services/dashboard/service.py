"""
Dashboard service: aggregate-backed hot paths with raw fallback.

Hot paths (net_worth_history, current totals, allocation) read from
snapshot_aggregates — O(snapshots × owners) rows, no merge needed.

Raw paths (time series, savings-rate sub-computation) still use the full
merged DataFrame, but are only loaded when investment accounts are present
or when no aggregates exist (migration period / tests using factories).
"""

from collections import defaultdict

import numpy as np
import pandas as pd
from sqlalchemy import select
from sqlalchemy.orm import Session

from app.models import Account, AppConfig, Asset, Snapshot, SnapshotValue, Transaction
from app.models.snapshot_aggregate import SnapshotAggregate
from app.schemas.dashboard import (
    AccountWrapperBreakdown,
    AllocationAnalysis,
    AllocationBreakdown,
    AllocationItem,
    CategoryTimeSeries,
    DashboardResponse,
    MetricCards,
    NetWorthPoint,
    RebalancingSuggestion,
    TileDelta,
    TileDeltas,
    WrapperTimeSeries,
)
from app.services.dashboard.metrics import (
    _calculate_debt_to_income,
    _calculate_hour_of_life_cost,
    _calculate_hour_of_work_cost,
    compute_tile_deltas,
)
from app.services.dashboard.time_series import (
    build_category_time_series,
    build_investment_time_series,
    build_wrapper_time_series,
)

_INVESTMENT_CATEGORIES = {"stock", "bond", "fund", "etf", "gold", "ppk"}


# ---------------------------------------------------------------------------
# Public entry point
# ---------------------------------------------------------------------------


def get_dashboard_data(db: Session) -> DashboardResponse:
    """Return dashboard metrics.

    Uses snapshot_aggregates for net_worth_history, current totals, and
    allocation when aggregates exist. Falls back to the raw-path implementation
    when the table is empty (migration period, or tests that create data via
    factories without triggering the service layer).
    """
    agg_rows = _load_aggregate_rows(db)
    if not agg_rows:
        return _get_dashboard_data_raw(db)
    return _get_dashboard_data_from_aggregates(db, agg_rows)


# ---------------------------------------------------------------------------
# Aggregate path
# ---------------------------------------------------------------------------


def _load_aggregate_rows(db: Session) -> list[SnapshotAggregate]:
    return list(db.execute(select(SnapshotAggregate)).scalars().all())


def _get_dashboard_data_from_aggregates(
    db: Session, agg_rows: list[SnapshotAggregate]
) -> DashboardResponse:
    # --- Net worth history (one point per snapshot, sum across owners) ---
    snapshot_nw: dict = defaultdict(float)
    for row in agg_rows:
        snapshot_nw[row.snapshot_id] += float(row.net_worth)

    snap_objs = (
        db.execute(select(Snapshot).where(Snapshot.id.in_(list(snapshot_nw)))).scalars().all()
    )
    snap_date: dict = {s.id: s.date for s in snap_objs}

    sorted_sids = sorted(snapshot_nw, key=lambda sid: snap_date[sid])
    net_worth_history = [
        NetWorthPoint(date=snap_date[sid], value=snapshot_nw[sid]) for sid in sorted_sids
    ]
    current_net_worth = float(net_worth_history[-1].value)
    last_month_net_worth = float(net_worth_history[-2].value) if len(net_worth_history) > 1 else 0.0

    # --- Latest snapshot (by highest snapshot_id) ---
    latest_sid: int = max(row.snapshot_id for row in agg_rows)
    latest_rows = [r for r in agg_rows if r.snapshot_id == latest_sid]

    total_assets = sum(float(r.total_assets) for r in latest_rows)
    total_liabilities = sum(float(r.total_liabilities) for r in latest_rows)

    allocation = sorted(
        [
            AllocationItem(
                category=item["category"],
                owner=row.owner,
                value=float(item["value"]),
            )
            for row in latest_rows
            for item in (row.allocation_json or [])
        ],
        key=lambda x: (x.category, x.owner),
    )

    # --- Shared data (always loaded) ---
    accounts_df = pd.read_sql(select(Account).where(Account.is_active.is_(True)), db.get_bind())
    app_config = db.execute(select(AppConfig).where(AppConfig.id == 1)).scalar_one_or_none()

    retirement_account_value = 0.0
    metric_cards: MetricCards
    allocation_analysis: AllocationAnalysis
    transactions_with_accounts_df = pd.DataFrame()

    if app_config:
        # Narrow load: only the latest snapshot's values (not a full table scan)
        latest_sv_df = _load_raw_for_metrics(db, latest_sid)

        if not latest_sv_df.empty:
            latest_df = latest_sv_df.merge(
                accounts_df,
                left_on="account_id",
                right_on="id",
                how="left",
                suffixes=("", "_account"),
            )
        else:
            latest_df = pd.DataFrame()

        if not latest_df.empty and "type" in latest_df.columns:
            retirement_df = latest_df[
                pd.notna(latest_df["account_id"])
                & (latest_df["type"] == "asset")
                & (latest_df["purpose"] == "retirement")
            ]
            retirement_account_value = float(retirement_df["value"].sum())

        # Transactions — shared between metric cards and time series
        transactions_df = _load_raw_for_transactions(db)
        if not transactions_df.empty:
            transactions_with_accounts_df = transactions_df.merge(
                accounts_df, left_on="account_id", right_on="id", how="left"
            )
            sign = np.where(
                transactions_with_accounts_df["transaction_type"] == "withdrawal", -1.0, 1.0
            )
            transactions_with_accounts_df["signed_amount"] = (
                transactions_with_accounts_df["amount"].astype(float) * sign
            )

        metric_cards = _build_metric_cards_aggregate(
            agg_rows,
            latest_df,
            accounts_df,
            transactions_with_accounts_df,
            app_config,
            retirement_account_value,
            db,
        )

        allocation_analysis = _build_allocation_analysis(latest_df, app_config)
    else:
        metric_cards = MetricCards(
            property_sqm=0,
            emergency_fund_months=0,
            retirement_income_monthly=0,
            mortgage_remaining=0,
            mortgage_months_left=0,
            mortgage_years_left=0,
            retirement_total=0,
            investment_contributions=0,
            investment_returns=0,
            savings_rate=None,
            debt_to_income_ratio=None,
            hour_of_work_cost=None,
            hour_of_life_cost=None,
        )
        allocation_analysis = AllocationAnalysis(
            by_category=[],
            by_wrapper=[],
            rebalancing=[],
            total_investment_value=0,
        )

    # --- Time series (only when investment accounts exist) ---
    has_investment = not accounts_df.empty and bool(
        accounts_df["category"].isin(_INVESTMENT_CATEGORIES).any()
    )

    if has_investment:
        assets_df = pd.read_sql(select(Asset).where(Asset.is_active.is_(True)), db.get_bind())
        snapshots_df = pd.read_sql(select(Snapshot).order_by(Snapshot.date), db.get_bind())
        values_df = pd.read_sql(select(SnapshotValue), db.get_bind())
        df = _build_merged_df(values_df, assets_df, accounts_df, snapshots_df)

        if app_config is None:
            # Transactions not yet loaded
            transactions_df = _load_raw_for_transactions(db)
            if not transactions_df.empty:
                transactions_with_accounts_df = transactions_df.merge(
                    accounts_df, left_on="account_id", right_on="id", how="left"
                )
                sign = np.where(
                    transactions_with_accounts_df["transaction_type"] == "withdrawal", -1.0, 1.0
                )
                transactions_with_accounts_df["signed_amount"] = (
                    transactions_with_accounts_df["amount"].astype(float) * sign
                )

        investment_time_series = build_investment_time_series(
            df, snapshots_df, transactions_with_accounts_df
        )
        wrapper_time_series = build_wrapper_time_series(
            df, snapshots_df, transactions_with_accounts_df
        )
        category_time_series = build_category_time_series(
            df, snapshots_df, transactions_with_accounts_df
        )
    else:
        investment_time_series = []
        wrapper_time_series = WrapperTimeSeries(ike=[], ikze=[], ppk=[])
        category_time_series = CategoryTimeSeries(stock=[], bond=[])

    return DashboardResponse(
        net_worth_history=net_worth_history,
        current_net_worth=current_net_worth,
        change_vs_last_month=current_net_worth - last_month_net_worth,
        total_assets=total_assets,
        total_liabilities=total_liabilities,
        allocation=allocation,
        retirement_account_value=retirement_account_value,
        metric_cards=metric_cards,
        allocation_analysis=allocation_analysis,
        investment_time_series=investment_time_series,
        wrapper_time_series=wrapper_time_series,
        category_time_series=category_time_series,
    )


def _load_raw_for_metrics(db: Session, latest_snapshot_id: int) -> pd.DataFrame:
    """Load SnapshotValues for the latest snapshot only (narrow query)."""
    return pd.read_sql(
        select(SnapshotValue).where(SnapshotValue.snapshot_id == latest_snapshot_id),
        db.get_bind(),
    )


def _load_raw_for_transactions(db: Session) -> pd.DataFrame:
    return pd.read_sql(select(Transaction).where(Transaction.is_active.is_(True)), db.get_bind())


def _build_merged_df(
    values_df: pd.DataFrame,
    assets_df: pd.DataFrame,
    accounts_df: pd.DataFrame,
    snapshots_df: pd.DataFrame,
) -> pd.DataFrame:
    """Full merge for time series functions (mirrors original lines 63-70)."""
    df = values_df.merge(
        assets_df, left_on="asset_id", right_on="id", how="left", suffixes=("", "_asset")
    )
    df = df.merge(
        accounts_df, left_on="account_id", right_on="id", how="left", suffixes=("", "_account")
    )
    df = df.merge(snapshots_df, left_on="snapshot_id", right_on="id", suffixes=("", "_snapshot"))

    asset_mask = df["asset_id"].notna() & df["name"].notna()
    account_mask = df["account_id"].notna() & df["type"].notna()
    sign = np.where(
        account_mask,
        np.where(df["type"] == "asset", 1, -1),
        np.where(asset_mask, 1, 0),
    )
    df["signed_value"] = df["value"].astype(float) * sign
    return df


def _compute_savings_rate_from_aggregates(
    agg_rows: list[SnapshotAggregate], db: Session
) -> float | None:
    """Savings rate from aggregate net_worth deltas — no SnapshotValue scan."""
    from app.models import SalaryRecord

    # Group by snapshot_id first to avoid summing across multiple same-month snapshots
    snapshot_nw_sr: dict = defaultdict(float)
    snapshot_month_sr: dict = {}
    for row in agg_rows:
        snapshot_nw_sr[row.snapshot_id] += float(row.net_worth)
        snapshot_month_sr[row.snapshot_id] = row.month

    # Per month, take the latest snapshot (highest snapshot_id = most recent)
    month_latest_sid: dict = {}
    for sid, month in snapshot_month_sr.items():
        if month not in month_latest_sid or sid > month_latest_sid[month]:
            month_latest_sid[month] = sid

    month_nw: dict = {month: snapshot_nw_sr[sid] for month, sid in month_latest_sid.items()}

    sorted_months = sorted(month_nw)
    if len(sorted_months) < 4:
        return None

    last_4 = sorted_months[-4:]
    nw_vals = [month_nw[m] for m in last_4]
    deltas = [nw_vals[i] - nw_vals[i - 1] for i in range(1, 4)]
    avg_delta = sum(deltas) / 3

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


def _build_metric_cards_aggregate(
    agg_rows: list[SnapshotAggregate],
    latest_df: pd.DataFrame,
    accounts_df: pd.DataFrame,
    transactions_with_accounts_df: pd.DataFrame,
    app_config: AppConfig,
    retirement_account_value: float,
    db: Session,
) -> MetricCards:
    """Metric cards using narrow latest_df (no full SnapshotValue scan)."""
    if latest_df.empty:
        return MetricCards(
            property_sqm=0,
            emergency_fund_months=0,
            retirement_income_monthly=0,
            mortgage_remaining=0,
            mortgage_months_left=0,
            mortgage_years_left=0,
            retirement_total=0,
            investment_contributions=0,
            investment_returns=0,
            savings_rate=None,
            debt_to_income_ratio=None,
            hour_of_work_cost=None,
            hour_of_life_cost=None,
        )

    real_estate_accounts = accounts_df[
        (accounts_df["category"] == "real_estate")
        & (accounts_df["owner"].isin(["Marcin", "Shared"]))
    ]
    total_property_sqm = float(real_estate_accounts["square_meters"].fillna(0).sum())

    real_estate_values = latest_df[
        pd.notna(latest_df["account_id"])
        & (latest_df["type"] == "asset")
        & (latest_df["account_id"].isin(real_estate_accounts["id"]))
    ]
    property_value = float(real_estate_values["value"].sum())

    mortgage_df = latest_df[
        pd.notna(latest_df["account_id"])
        & (latest_df["type"] == "liability")
        & (latest_df["category"].isin(["housing", "mortgage"]))
    ]
    mortgage_remaining = float(mortgage_df["value"].sum())

    if property_value > 0:
        equity_percentage = (property_value - mortgage_remaining) / property_value
        property_sqm = total_property_sqm * max(0.0, equity_percentage)
    else:
        property_sqm = 0.0

    emergency_fund_df = latest_df[
        pd.notna(latest_df["account_id"])
        & (latest_df["type"] == "asset")
        & (latest_df["purpose"] == "emergency_fund")
    ]
    emergency_fund_value = float(emergency_fund_df["value"].sum())
    emergency_fund_months = (
        emergency_fund_value / float(app_config.monthly_expenses)
        if app_config.monthly_expenses > 0
        else 0
    )

    retirement_income_monthly = (retirement_account_value * 0.04) / 12

    mortgage_months_left = int(
        abs(mortgage_remaining) / float(app_config.monthly_mortgage_payment)
        if app_config.monthly_mortgage_payment > 0
        else 0
    )
    mortgage_years_left = mortgage_months_left / 12
    retirement_total = retirement_account_value

    # Investment contributions from transactions (shared DataFrame)
    investment_contributions = 0.0
    investment_returns = 0.0
    if not transactions_with_accounts_df.empty:
        # Find the latest snapshot date from aggregate rows
        from app.models import Snapshot as SnapModel

        latest_snap = db.execute(
            select(SnapModel).where(SnapModel.id == max(r.snapshot_id for r in agg_rows))
        ).scalar_one_or_none()
        if latest_snap is not None:
            investment_trans = transactions_with_accounts_df[
                (transactions_with_accounts_df["purpose"] == "retirement")
                & (transactions_with_accounts_df["date"] <= latest_snap.date)
            ].copy()
            investment_contributions = float(investment_trans["signed_amount"].sum())
            investment_returns = retirement_account_value - investment_contributions

    savings_rate = _compute_savings_rate_from_aggregates(agg_rows, db)
    debt_to_income_ratio = _calculate_debt_to_income(db)
    hour_of_work_cost = _calculate_hour_of_work_cost(db)
    hour_of_life_cost = _calculate_hour_of_life_cost(db)

    return MetricCards(
        property_sqm=property_sqm,
        emergency_fund_months=emergency_fund_months,
        retirement_income_monthly=retirement_income_monthly,
        mortgage_remaining=mortgage_remaining,
        mortgage_months_left=mortgage_months_left,
        mortgage_years_left=mortgage_years_left,
        retirement_total=retirement_total,
        investment_contributions=investment_contributions,
        investment_returns=investment_returns,
        savings_rate=savings_rate,
        debt_to_income_ratio=debt_to_income_ratio,
        hour_of_work_cost=hour_of_work_cost,
        hour_of_life_cost=hour_of_life_cost,
    )


def _build_allocation_analysis(
    latest_df: pd.DataFrame,
    app_config: AppConfig,
) -> AllocationAnalysis:
    """Allocation analysis from narrow latest_df (mirrors original lines 339-453)."""
    if latest_df.empty:
        return AllocationAnalysis(
            by_category=[],
            by_wrapper=[],
            rebalancing=[],
            total_investment_value=0,
        )

    allocation_categories = {"stock", "bond", "fund", "etf", "gold"}

    investment_df = latest_df[
        pd.notna(latest_df["account_id"])
        & (latest_df["type"] == "asset")
        & (latest_df["category"].isin(allocation_categories))
    ]

    total_investment_value = float(investment_df["value"].sum())

    category_map = {
        "stock": "stocks",
        "fund": "stocks",
        "etf": "stocks",
        "bond": "bonds",
        "gold": "gold",
    }

    alloc_groups = investment_df["category"].map(category_map).fillna("other")
    allocation_by_cat = investment_df["value"].groupby(alloc_groups).sum().to_dict()

    allocation_breakdown = []
    target_allocations = {
        "stocks": app_config.allocation_stocks,
        "bonds": app_config.allocation_bonds,
        "gold": app_config.allocation_gold,
    }

    for category, target_pct in target_allocations.items():
        current_value = allocation_by_cat.get(category, 0)
        current_pct = (
            (current_value / total_investment_value * 100) if total_investment_value > 0 else 0
        )
        difference = current_pct - target_pct
        allocation_breakdown.append(
            AllocationBreakdown(
                category=category,
                current_value=float(current_value),
                current_percentage=float(current_pct),
                target_percentage=float(target_pct),
                difference=float(difference),
            )
        )

    all_investment_categories = {"stock", "bond", "fund", "etf", "gold", "ppk"}
    all_investment_df = latest_df[
        pd.notna(latest_df["account_id"])
        & (latest_df["type"] == "asset")
        & (latest_df["category"].isin(all_investment_categories))
    ]

    wrappers = all_investment_df["account_wrapper"].fillna("Regular")
    wrapper_breakdown = all_investment_df["value"].groupby(wrappers).sum().to_dict()
    all_investment_total = sum(wrapper_breakdown.values())

    wrapper_list = [
        AccountWrapperBreakdown(
            wrapper=wrapper,
            value=float(value),
            percentage=(
                float(value / all_investment_total * 100) if all_investment_total > 0 else 0
            ),
        )
        for wrapper, value in wrapper_breakdown.items()
    ]

    rebalancing_suggestions = []
    for breakdown in allocation_breakdown:
        if breakdown.difference < -1:
            target_value = total_investment_value * breakdown.target_percentage / 100
            current_value = breakdown.current_value
            target_pct = breakdown.target_percentage / 100
            if target_pct < 1:
                amount_to_add = (target_value - current_value) / (1 - target_pct)
                if amount_to_add > 100:
                    rebalancing_suggestions.append(
                        RebalancingSuggestion(
                            category=breakdown.category,
                            action="buy",
                            amount=float(amount_to_add),
                        )
                    )

    return AllocationAnalysis(
        by_category=allocation_breakdown,
        by_wrapper=wrapper_list,
        rebalancing=rebalancing_suggestions,
        total_investment_value=total_investment_value,
    )


# ---------------------------------------------------------------------------
# Raw path (fallback when aggregates not populated)
# ---------------------------------------------------------------------------


def _get_dashboard_data_raw(db: Session) -> DashboardResponse:
    """
    Calculate dashboard metrics using pandas (original implementation).

    Used when snapshot_aggregates is empty — during migration period or when
    tests create data via factories without going through the service layer.

    pandas features used:
    - pd.DataFrame(): Create from query results
    - df.merge(): Join DataFrames (like SQL JOIN)
    - df.groupby(): Aggregate data
    - df.pivot_table(): Reshape data
    - df.sort_values(): Order data
    """

    # Fetch all data needed
    assets_query = select(Asset).where(Asset.is_active.is_(True))
    assets_df = pd.read_sql(assets_query, db.get_bind())

    accounts_query = select(Account).where(Account.is_active.is_(True))
    accounts_df = pd.read_sql(accounts_query, db.get_bind())

    snapshots_query = select(Snapshot).order_by(Snapshot.date)
    snapshots_df = pd.read_sql(snapshots_query, db.get_bind())

    values_query = select(SnapshotValue)
    values_df = pd.read_sql(values_query, db.get_bind())

    # pandas: merge() - LEFT JOIN snapshot values with both assets and accounts
    df = values_df.merge(
        assets_df, left_on="asset_id", right_on="id", how="left", suffixes=("", "_asset")
    )
    df = df.merge(
        accounts_df, left_on="account_id", right_on="id", how="left", suffixes=("", "_account")
    )
    df = df.merge(snapshots_df, left_on="snapshot_id", right_on="id", suffixes=("", "_snapshot"))

    # Handle case when no snapshots exist
    if df.empty:
        return DashboardResponse(
            net_worth_history=[],
            current_net_worth=0,
            change_vs_last_month=0,
            total_assets=0,
            total_liabilities=0,
            allocation=[],
            retirement_account_value=0,
            metric_cards=MetricCards(
                property_sqm=0,
                emergency_fund_months=0,
                retirement_income_monthly=0,
                mortgage_remaining=0,
                mortgage_months_left=0,
                mortgage_years_left=0,
                retirement_total=0,
                investment_contributions=0,
                investment_returns=0,
            ),
            allocation_analysis=AllocationAnalysis(
                by_category=[],
                by_wrapper=[],
                rebalancing=[],
                total_investment_value=0,
            ),
            investment_time_series=[],
            wrapper_time_series=WrapperTimeSeries(ike=[], ikze=[], ppk=[]),
            category_time_series=CategoryTimeSeries(stock=[], bond=[]),
            tile_deltas=TileDeltas(
                net_worth=TileDelta(mom=None, yoy=None),
                assets=TileDelta(mom=None, yoy=None),
                liabilities=TileDelta(mom=None, yoy=None),
            ),
        )

    # Vectorized signed value
    asset_mask = df["asset_id"].notna() & df["name"].notna()
    account_mask = df["account_id"].notna() & df["type"].notna()
    sign = np.where(
        account_mask,
        np.where(df["type"] == "asset", 1, -1),
        np.where(asset_mask, 1, 0),
    )
    df["signed_value"] = df["value"].astype(float) * sign

    net_worth_by_date = df.groupby("date")["signed_value"].sum().reset_index()
    net_worth_by_date.columns = ["date", "net_worth"]
    net_worth_by_date = net_worth_by_date.sort_values("date")

    net_worth_history = [
        NetWorthPoint(date=d, value=v)
        for d, v in zip(net_worth_by_date["date"], net_worth_by_date["net_worth"], strict=True)
    ]

    if len(net_worth_by_date) > 0:
        current_net_worth = float(net_worth_by_date.iloc[-1]["net_worth"])
        last_month_net_worth = (
            float(net_worth_by_date.iloc[-2]["net_worth"]) if len(net_worth_by_date) > 1 else 0
        )
    else:
        current_net_worth = 0
        last_month_net_worth = 0

    latest_df = pd.DataFrame()
    if not df.empty and "snapshot_id" in df.columns:
        valid_ids = df["snapshot_id"].dropna().unique()
        candidates = snapshots_df[snapshots_df["id"].isin(valid_ids)]
        if candidates.empty:
            latest_snapshot = None
        else:
            latest_snapshot = candidates.sort_values(["date", "id"], ascending=[False, False]).iloc[
                0
            ]
    else:
        latest_snapshot = None

    if latest_snapshot is not None:
        latest_df = df[df["snapshot_id"] == latest_snapshot["id"]]

        from_asset_table = latest_df[pd.notna(latest_df["asset_id"])]
        from_account_assets = latest_df[
            (pd.notna(latest_df["account_id"])) & (latest_df["type"] == "asset")
        ]
        total_assets = float(from_asset_table["value"].sum() + from_account_assets["value"].sum())

        from_account_liabilities = latest_df[
            (pd.notna(latest_df["account_id"])) & (latest_df["type"] == "liability")
        ]
        total_liabilities = float(from_account_liabilities["value"].sum())

        account_assets_df = latest_df[
            (pd.notna(latest_df["account_id"]))
            & (latest_df["type"] == "asset")
            & (pd.notna(latest_df["category"]))
            & (pd.notna(latest_df["owner"]))
        ]

        allocation_df = (
            account_assets_df.groupby(["category", "owner"])["value"].sum().reset_index()
        )

        allocation = [
            AllocationItem(category=c, owner=o, value=float(v))
            for c, o, v in zip(
                allocation_df["category"],
                allocation_df["owner"],
                allocation_df["value"],
                strict=True,
            )
        ]

        retirement_accounts_df = latest_df[
            (pd.notna(latest_df["account_id"]))
            & (latest_df["type"] == "asset")
            & (latest_df["purpose"] == "retirement")
        ]
        retirement_account_value = float(retirement_accounts_df["value"].sum())
    else:
        total_assets = 0
        total_liabilities = 0
        allocation = []
        retirement_account_value = 0

    app_config = db.execute(select(AppConfig).where(AppConfig.id == 1)).scalar_one_or_none()

    if app_config and latest_snapshot is not None:
        real_estate_accounts = accounts_df[
            (accounts_df["category"] == "real_estate")
            & (accounts_df["owner"].isin(["Marcin", "Shared"]))
        ]
        total_property_sqm = float(real_estate_accounts["square_meters"].fillna(0).sum())

        real_estate_values = latest_df[
            (pd.notna(latest_df["account_id"]))
            & (latest_df["type"] == "asset")
            & (latest_df["account_id"].isin(real_estate_accounts["id"]))
        ]
        property_value = float(real_estate_values["value"].sum())

        mortgage_df = latest_df[
            (pd.notna(latest_df["account_id"]))
            & (latest_df["type"] == "liability")
            & (latest_df["category"].isin(["housing", "mortgage"]))
        ]
        mortgage_remaining = float(mortgage_df["value"].sum())

        if property_value > 0:
            equity_percentage = (property_value - mortgage_remaining) / property_value
            property_sqm = total_property_sqm * max(0.0, equity_percentage)
        else:
            property_sqm = 0.0

        emergency_fund_df = latest_df[
            (pd.notna(latest_df["account_id"]))
            & (latest_df["type"] == "asset")
            & (latest_df["purpose"] == "emergency_fund")
        ]
        emergency_fund_value = float(emergency_fund_df["value"].sum())
        emergency_fund_months = (
            emergency_fund_value / float(app_config.monthly_expenses)
            if app_config.monthly_expenses > 0
            else 0
        )

        retirement_income_monthly = (retirement_account_value * 0.04) / 12

        mortgage_months_left = int(
            abs(mortgage_remaining) / float(app_config.monthly_mortgage_payment)
            if app_config.monthly_mortgage_payment > 0
            else 0
        )

        mortgage_years_left = mortgage_months_left / 12
        retirement_total = retirement_account_value

        transactions_query = select(Transaction).where(Transaction.is_active.is_(True))
        transactions_df = pd.read_sql(transactions_query, db.get_bind())

        if not transactions_df.empty and latest_snapshot is not None:
            trans_with_accounts = transactions_df.merge(
                accounts_df, left_on="account_id", right_on="id", how="left"
            )

            investment_trans = trans_with_accounts[
                (trans_with_accounts["purpose"] == "retirement")
                & (trans_with_accounts["date"] <= latest_snapshot["date"])
            ].copy()
            sign = np.where(investment_trans["transaction_type"] == "withdrawal", -1.0, 1.0)
            investment_trans["signed_amount"] = investment_trans["amount"].astype(float) * sign
            investment_contributions = float(investment_trans["signed_amount"].sum())

            investment_current_value = retirement_account_value
            investment_returns = investment_current_value - investment_contributions
        else:
            investment_contributions = 0
            investment_returns = 0

        from app.services.dashboard.metrics import _calculate_savings_rate

        savings_rate = _calculate_savings_rate(snapshots_df, df, db)
        debt_to_income_ratio = _calculate_debt_to_income(db)
        hour_of_work_cost = _calculate_hour_of_work_cost(db)
        hour_of_life_cost = _calculate_hour_of_life_cost(db)

        metric_cards = MetricCards(
            property_sqm=property_sqm,
            emergency_fund_months=emergency_fund_months,
            retirement_income_monthly=retirement_income_monthly,
            mortgage_remaining=mortgage_remaining,
            mortgage_months_left=mortgage_months_left,
            mortgage_years_left=mortgage_years_left,
            retirement_total=retirement_total,
            investment_contributions=investment_contributions,
            investment_returns=investment_returns,
            savings_rate=savings_rate,
            debt_to_income_ratio=debt_to_income_ratio,
            hour_of_work_cost=hour_of_work_cost,
            hour_of_life_cost=hour_of_life_cost,
        )
    else:
        metric_cards = MetricCards(
            property_sqm=0,
            emergency_fund_months=0,
            retirement_income_monthly=0,
            mortgage_remaining=0,
            mortgage_months_left=0,
            mortgage_years_left=0,
            retirement_total=0,
            investment_contributions=0,
            investment_returns=0,
            savings_rate=None,
            debt_to_income_ratio=None,
            hour_of_work_cost=None,
            hour_of_life_cost=None,
        )

    if app_config and latest_snapshot is not None:
        allocation_categories = {"stock", "bond", "fund", "etf", "gold"}

        investment_df = latest_df[
            (pd.notna(latest_df["account_id"]))
            & (latest_df["type"] == "asset")
            & (latest_df["category"].isin(allocation_categories))
        ]

        total_investment_value = float(investment_df["value"].sum())

        category_map = {
            "stock": "stocks",
            "fund": "stocks",
            "etf": "stocks",
            "bond": "bonds",
            "gold": "gold",
        }

        alloc_groups = investment_df["category"].map(category_map).fillna("other")
        allocation_by_cat = investment_df["value"].groupby(alloc_groups).sum().to_dict()

        allocation_breakdown = []
        target_allocations = {
            "stocks": app_config.allocation_stocks,
            "bonds": app_config.allocation_bonds,
            "gold": app_config.allocation_gold,
        }

        for category, target_pct in target_allocations.items():
            current_value = allocation_by_cat.get(category, 0)
            current_pct = (
                (current_value / total_investment_value * 100) if total_investment_value > 0 else 0
            )
            difference = current_pct - target_pct

            allocation_breakdown.append(
                AllocationBreakdown(
                    category=category,
                    current_value=float(current_value),
                    current_percentage=float(current_pct),
                    target_percentage=float(target_pct),
                    difference=float(difference),
                )
            )

        all_investment_categories = {"stock", "bond", "fund", "etf", "gold", "ppk"}
        all_investment_df = latest_df[
            (pd.notna(latest_df["account_id"]))
            & (latest_df["type"] == "asset")
            & (latest_df["category"].isin(all_investment_categories))
        ]

        wrappers = all_investment_df["account_wrapper"].fillna("Regular")
        wrapper_breakdown = all_investment_df["value"].groupby(wrappers).sum().to_dict()

        all_investment_total = sum(wrapper_breakdown.values())

        wrapper_list = [
            AccountWrapperBreakdown(
                wrapper=wrapper,
                value=float(value),
                percentage=(
                    float(value / all_investment_total * 100) if all_investment_total > 0 else 0
                ),
            )
            for wrapper, value in wrapper_breakdown.items()
        ]

        rebalancing_suggestions = []
        for breakdown in allocation_breakdown:
            if breakdown.difference < -1:
                target_value = total_investment_value * breakdown.target_percentage / 100
                current_value = breakdown.current_value
                target_pct = breakdown.target_percentage / 100

                if target_pct < 1:
                    amount_to_add = (target_value - current_value) / (1 - target_pct)

                    if amount_to_add > 100:
                        rebalancing_suggestions.append(
                            RebalancingSuggestion(
                                category=breakdown.category,
                                action="buy",
                                amount=float(amount_to_add),
                            )
                        )

        allocation_analysis = AllocationAnalysis(
            by_category=allocation_breakdown,
            by_wrapper=wrapper_list,
            rebalancing=rebalancing_suggestions,
            total_investment_value=total_investment_value,
        )
    else:
        allocation_analysis = AllocationAnalysis(
            by_category=[],
            by_wrapper=[],
            rebalancing=[],
            total_investment_value=0,
        )

    transactions_query = select(Transaction).where(Transaction.is_active.is_(True))
    transactions_df = pd.read_sql(transactions_query, db.get_bind())
    if not transactions_df.empty:
        transactions_with_accounts_df = transactions_df.merge(
            accounts_df, left_on="account_id", right_on="id", how="left"
        )
        sign = np.where(
            transactions_with_accounts_df["transaction_type"] == "withdrawal", -1.0, 1.0
        )
        transactions_with_accounts_df["signed_amount"] = (
            transactions_with_accounts_df["amount"].astype(float) * sign
        )
    else:
        transactions_with_accounts_df = pd.DataFrame()

    investment_time_series = build_investment_time_series(
        df, snapshots_df, transactions_with_accounts_df
    )
    wrapper_time_series = build_wrapper_time_series(df, snapshots_df, transactions_with_accounts_df)
    category_time_series = build_category_time_series(
        df, snapshots_df, transactions_with_accounts_df
    )

    tile_deltas = compute_tile_deltas(df, snapshots_df)

    return DashboardResponse(
        net_worth_history=net_worth_history,
        current_net_worth=current_net_worth,
        change_vs_last_month=current_net_worth - last_month_net_worth,
        total_assets=total_assets,
        total_liabilities=total_liabilities,
        allocation=allocation,
        retirement_account_value=retirement_account_value,
        metric_cards=metric_cards,
        allocation_analysis=allocation_analysis,
        investment_time_series=investment_time_series,
        wrapper_time_series=wrapper_time_series,
        category_time_series=category_time_series,
        tile_deltas=tile_deltas,
    )
