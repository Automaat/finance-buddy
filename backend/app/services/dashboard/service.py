"""
Dashboard service using pandas for financial calculations.
Demonstrates: groupby, pivot, merge, aggregations, time series
"""

import pandas as pd
from sqlalchemy import select
from sqlalchemy.orm import Session

from app.models import Account, AppConfig, Asset, Snapshot, SnapshotValue, Transaction
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
    WrapperTimeSeries,
)
from app.services.dashboard.metrics import (
    _calculate_debt_to_income,
    _calculate_hour_of_life_cost,
    _calculate_hour_of_work_cost,
    _calculate_savings_rate,
)
from app.services.dashboard.time_series import (
    build_category_time_series,
    build_investment_time_series,
    build_wrapper_time_series,
)


def get_dashboard_data(db: Session) -> DashboardResponse:
    """
    Calculate dashboard metrics using pandas.

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
    # Similar to SQL: SELECT * FROM snapshot_values LEFT JOIN assets LEFT JOIN accounts
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
        )

    # Calculate net worth per snapshot
    # pandas: Calculate signed value based on whether it's an asset or liability
    # Assets (from Asset table) contribute positively
    # Accounts depend on account.type (asset=+, liability=-)
    # Note: Inactive accounts/assets will have NaN in name/type after LEFT JOIN, exclude them
    def calculate_signed_value(row):
        if pd.notna(row["asset_id"]) and pd.notna(row.get("name")):
            # From Asset table - always positive (only if asset was in the join)
            return row["value"]
        if pd.notna(row["account_id"]) and pd.notna(row.get("type")):
            # From Account table - assets positive, liabilities negative
            return row["value"] if row["type"] == "asset" else -row["value"]
        return 0

    df["signed_value"] = df.apply(calculate_signed_value, axis=1)

    # pandas: groupby() + sum() - Aggregate net worth by date
    net_worth_by_date = df.groupby("date")["signed_value"].sum().reset_index()
    net_worth_by_date.columns = ["date", "net_worth"]

    # pandas: sort_values() - Order by date
    net_worth_by_date = net_worth_by_date.sort_values("date")

    # Convert to response format
    net_worth_history = [
        NetWorthPoint(date=row["date"], value=row["net_worth"])
        for _, row in net_worth_by_date.iterrows()
    ]

    # Current metrics (latest snapshot)
    if len(net_worth_by_date) > 0:
        current_net_worth = float(net_worth_by_date.iloc[-1]["net_worth"])
        last_month_net_worth = (
            float(net_worth_by_date.iloc[-2]["net_worth"]) if len(net_worth_by_date) > 1 else 0
        )
    else:
        current_net_worth = 0
        last_month_net_worth = 0

    # Latest snapshot data for current totals
    # Use merged df to determine latest snapshot (handles case where merge filters out snapshots)
    latest_df = pd.DataFrame()  # Initialize to satisfy type checker
    if not df.empty and "snapshot_id" in df.columns:
        latest_snapshot_id = df["snapshot_id"].max()
        latest_snapshot = snapshots_df[snapshots_df["id"] == latest_snapshot_id].iloc[0]
    else:
        latest_snapshot = None

    if latest_snapshot is not None:
        # pandas: Boolean indexing - Filter rows
        latest_df = df[df["snapshot_id"] == latest_snapshot["id"]]

        # Calculate total assets: from Asset table + from Account table where type="asset"
        # pandas: Boolean masks for filtering
        from_asset_table = latest_df[pd.notna(latest_df["asset_id"])]
        from_account_assets = latest_df[
            (pd.notna(latest_df["account_id"])) & (latest_df["type"] == "asset")
        ]
        total_assets = float(from_asset_table["value"].sum() + from_account_assets["value"].sum())

        # Calculate total liabilities: from Account table where type="liability"
        from_account_liabilities = latest_df[
            (pd.notna(latest_df["account_id"])) & (latest_df["type"] == "liability")
        ]
        total_liabilities = float(from_account_liabilities["value"].sum())

        # Asset allocation - ONLY from accounts (not from Asset table)
        # pandas: Filter for accounts with type="asset" that have category and owner
        account_assets_df = latest_df[
            (pd.notna(latest_df["account_id"]))
            & (latest_df["type"] == "asset")
            & (pd.notna(latest_df["category"]))
            & (pd.notna(latest_df["owner"]))
        ]

        # pandas: groupby() with multiple columns
        allocation_df = (
            account_assets_df.groupby(["category", "owner"])["value"].sum().reset_index()
        )

        allocation = [
            AllocationItem(category=row["category"], owner=row["owner"], value=float(row["value"]))
            for _, row in allocation_df.iterrows()
        ]

        # Calculate retirement account value (accounts with purpose='retirement')
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

    # Calculate metric cards
    # Fetch AppConfig for monthly_expenses and monthly_mortgage_payment
    app_config = db.execute(select(AppConfig).where(AppConfig.id == 1)).scalar_one_or_none()

    if app_config and latest_snapshot is not None:
        # 1. Property square meters - adjusted for mortgage equity
        # Get real estate accounts for Marcin/Shared
        real_estate_accounts = accounts_df[
            (accounts_df["category"] == "real_estate")
            & (accounts_df["owner"].isin(["Marcin", "Shared"]))
        ]
        total_property_sqm = float(real_estate_accounts["square_meters"].fillna(0).sum())

        # Get real estate value from latest snapshot
        real_estate_values = latest_df[
            (pd.notna(latest_df["account_id"]))
            & (latest_df["type"] == "asset")
            & (latest_df["account_id"].isin(real_estate_accounts["id"]))
        ]
        property_value = float(real_estate_values["value"].sum())

        # Get mortgage remaining
        mortgage_df = latest_df[
            (pd.notna(latest_df["account_id"]))
            & (latest_df["type"] == "liability")
            & (latest_df["category"].isin(["housing", "mortgage"]))
        ]
        mortgage_remaining = float(mortgage_df["value"].sum())

        # Calculate equity percentage and owned square meters
        # mortgage_remaining is positive (liability value), subtract from property value
        if property_value > 0:
            equity_percentage = (property_value - mortgage_remaining) / property_value
            property_sqm = total_property_sqm * max(0.0, equity_percentage)
        else:
            property_sqm = 0.0

        # 2. Emergency fund months - sum accounts where purpose='emergency_fund' / monthly_expenses
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

        # 3. Retirement income (4% rule) - (retirement_account_value * 0.04) / 12
        retirement_income_monthly = (retirement_account_value * 0.04) / 12

        # 4. Mortgage remaining - already calculated above for property_sqm equity calculation

        # 5. Months to mortgage payoff - mortgage_remaining / monthly_mortgage_payment
        mortgage_months_left = int(
            abs(mortgage_remaining) / float(app_config.monthly_mortgage_payment)
            if app_config.monthly_mortgage_payment > 0
            else 0
        )

        # 6. Years to payoff - months_to_payoff / 12
        mortgage_years_left = mortgage_months_left / 12

        # 7. Retirement savings - already available
        retirement_total = retirement_account_value

        # 8. Investment contributions - sum of Transaction.amount for investment accounts
        # 9. Investment returns - Current value - total contributions
        # Filter transactions up to latest snapshot date to match time series calculation
        transactions_query = select(Transaction).where(Transaction.is_active.is_(True))
        transactions_df = pd.read_sql(transactions_query, db.get_bind())

        if not transactions_df.empty and latest_snapshot is not None:
            # Join transactions with accounts to filter investment accounts
            trans_with_accounts = transactions_df.merge(
                accounts_df, left_on="account_id", right_on="id", how="left"
            )

            # Filter for retirement investment accounts (purpose='retirement')
            # to match retirement_total scope, excluding non-retirement investments
            # Only include transactions up to latest snapshot date
            investment_trans = trans_with_accounts[
                (trans_with_accounts["purpose"] == "retirement")
                & (trans_with_accounts["date"] <= latest_snapshot["date"])
            ]
            investment_trans = investment_trans.copy()
            investment_trans["signed_amount"] = investment_trans.apply(
                lambda row: -row["amount"]
                if row["transaction_type"] == "withdrawal"
                else row["amount"],
                axis=1,
            )
            investment_contributions = float(investment_trans["signed_amount"].sum())

            # Reuse already-calculated retirement account value for consistency
            investment_current_value = retirement_account_value
            investment_returns = investment_current_value - investment_contributions
        else:
            investment_contributions = 0
            investment_returns = 0

        # Calculate new metrics
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

    # Calculate allocation analysis
    if app_config and latest_snapshot is not None:
        # Define investment categories for allocation (exclude PPK)
        allocation_categories = {"stock", "bond", "fund", "etf", "gold"}

        # Get investment accounts from latest snapshot (excluding PPK)
        investment_df = latest_df[
            (pd.notna(latest_df["account_id"]))
            & (latest_df["type"] == "asset")
            & (latest_df["category"].isin(allocation_categories))
        ]

        total_investment_value = float(investment_df["value"].sum())

        # Map categories to allocation groups
        category_map = {
            "stock": "stocks",
            "fund": "stocks",
            "etf": "stocks",
            "bond": "bonds",
            "gold": "gold",
        }

        # Calculate current allocation by category
        allocation_by_cat = {}
        for _, row in investment_df.iterrows():
            alloc_group = category_map.get(row["category"], "other")
            allocation_by_cat[alloc_group] = allocation_by_cat.get(alloc_group, 0) + row["value"]

        # Calculate allocation breakdown
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

        # Calculate breakdown by account wrapper (include all investments including PPK)
        all_investment_categories = {"stock", "bond", "fund", "etf", "gold", "ppk"}
        all_investment_df = latest_df[
            (pd.notna(latest_df["account_id"]))
            & (latest_df["type"] == "asset")
            & (latest_df["category"].isin(all_investment_categories))
        ]

        wrapper_breakdown = {}
        for _, row in all_investment_df.iterrows():
            wrapper = row["account_wrapper"] if pd.notna(row["account_wrapper"]) else "Regular"
            wrapper_breakdown[wrapper] = wrapper_breakdown.get(wrapper, 0) + row["value"]

        # Calculate total for percentage (includes PPK, unlike total_investment_value)
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

        # Calculate rebalancing suggestions
        # Formula: amount_to_add = (target_value - current_value) / (1 - target_pct/100)
        # This calculates how much new money to add to reach target allocation
        # Only show "buy" suggestions for under-allocated categories
        rebalancing_suggestions = []
        for breakdown in allocation_breakdown:
            if breakdown.difference < -1:  # Only if under-allocated by more than 1%
                target_value = total_investment_value * breakdown.target_percentage / 100
                current_value = breakdown.current_value
                target_pct = breakdown.target_percentage / 100

                # Calculate how much new money to add (Excel formula)
                if target_pct < 1:  # Avoid division by zero
                    amount_to_add = (target_value - current_value) / (1 - target_pct)

                    # Only include if amount is significant and positive
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

    # Fetch and prepare transactions for time series calculations
    # Execute query once and merge with accounts to avoid repeated operations in loops
    transactions_query = select(Transaction).where(Transaction.is_active.is_(True))
    transactions_df = pd.read_sql(transactions_query, db.get_bind())
    if not transactions_df.empty:
        transactions_with_accounts_df = transactions_df.merge(
            accounts_df, left_on="account_id", right_on="id", how="left"
        )
        # Withdrawals reduce cumulative contributions — negate their amount
        transactions_with_accounts_df["signed_amount"] = transactions_with_accounts_df.apply(
            lambda row: -row["amount"]
            if row["transaction_type"] == "withdrawal"
            else row["amount"],
            axis=1,
        )
    else:
        transactions_with_accounts_df = pd.DataFrame()

    # Time series (overall, per-wrapper, per-category) — see dashboard.time_series
    investment_time_series = build_investment_time_series(
        df, snapshots_df, transactions_with_accounts_df
    )
    wrapper_time_series = build_wrapper_time_series(df, snapshots_df, transactions_with_accounts_df)
    category_time_series = build_category_time_series(
        df, snapshots_df, transactions_with_accounts_df
    )

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
