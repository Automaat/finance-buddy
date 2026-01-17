"""
Dashboard service using pandas for financial calculations.
Demonstrates: groupby, pivot, merge, aggregations, time series
"""

import pandas as pd
from sqlalchemy import select
from sqlalchemy.orm import Session

from app.models import Account, Asset, Snapshot, SnapshotValue
from app.schemas.dashboard import AllocationItem, DashboardResponse, NetWorthPoint


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

    # Calculate net worth per snapshot
    # pandas: Calculate signed value based on whether it's an asset or liability
    # Assets (from Asset table) contribute positively
    # Accounts depend on account.type (asset=+, liability=-)
    def calculate_signed_value(row):
        if pd.notna(row["asset_id"]):
            # From Asset table - always positive
            return row["value"]
        if pd.notna(row["account_id"]):
            # From Account table - depends on type
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
    else:
        total_assets = 0
        total_liabilities = 0
        allocation = []

    return DashboardResponse(
        net_worth_history=net_worth_history,
        current_net_worth=current_net_worth,
        change_vs_last_month=current_net_worth - last_month_net_worth,
        total_assets=total_assets,
        total_liabilities=total_liabilities,
        allocation=allocation,
    )
