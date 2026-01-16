"""
Dashboard service using pandas for financial calculations.
Demonstrates: groupby, pivot, merge, aggregations, time series
"""

import pandas as pd
from sqlalchemy import select
from sqlalchemy.orm import Session

from app.models import Account, Snapshot, SnapshotValue
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
    accounts_query = select(Account).where(Account.is_active.is_(True))
    accounts_df = pd.read_sql(accounts_query, db.get_bind())

    snapshots_query = select(Snapshot).order_by(Snapshot.date)
    snapshots_df = pd.read_sql(snapshots_query, db.get_bind())

    values_query = select(SnapshotValue)
    values_df = pd.read_sql(values_query, db.get_bind())

    # pandas: merge() - Join snapshot values with accounts
    # Similar to SQL: SELECT * FROM snapshot_values JOIN accounts
    df = values_df.merge(
        accounts_df, left_on="account_id", right_on="id", suffixes=("", "_account")
    )
    df = df.merge(snapshots_df, left_on="snapshot_id", right_on="id", suffixes=("", "_snapshot"))

    # Calculate net worth per snapshot
    # pandas: groupby() + vectorized aggregations - Efficient aggregation per group
    net_worth_by_date = (
        df.groupby(["date", "type"])["value"].sum().unstack(fill_value=0).reset_index()
    )
    # Ensure missing asset/liability columns are treated as zero
    if "asset" not in net_worth_by_date.columns:
        net_worth_by_date["asset"] = 0.0
    if "liability" not in net_worth_by_date.columns:
        net_worth_by_date["liability"] = 0.0
    net_worth_by_date["net_worth"] = net_worth_by_date["asset"] - net_worth_by_date["liability"]
    net_worth_by_date = net_worth_by_date[["date", "net_worth"]]

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

        # pandas: groupby() + sum() - Aggregate by type
        totals_by_type = latest_df.groupby("type")["value"].sum()

        total_assets = float(totals_by_type.get("asset", 0))
        total_liabilities = float(totals_by_type.get("liability", 0))

        # Asset allocation
        # pandas: Query filter + groupby multiple columns
        assets_df = latest_df[latest_df["type"] == "asset"]

        # pandas: groupby() with multiple columns
        allocation_df = assets_df.groupby(["category", "owner"])["value"].sum().reset_index()

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
