"""Investment time-series builders for the dashboard (overall, per-wrapper, per-category)."""

import pandas as pd

from app.schemas.dashboard import CategoryTimeSeries, InvestmentTimeSeriesPoint, WrapperTimeSeries


def build_investment_time_series(
    df: pd.DataFrame, snapshots_df: pd.DataFrame, transactions_with_accounts_df: pd.DataFrame
) -> list[InvestmentTimeSeriesPoint]:
    """Cumulative investment value, contributions, and returns per snapshot."""
    investment_time_series: list[InvestmentTimeSeriesPoint] = []
    if df.empty or snapshots_df.empty:
        return investment_time_series

    # Define investment categories
    investment_categories = {"stock", "bond", "fund", "etf", "gold", "ppk"}

    # For each snapshot, calculate investment metrics
    for _, snapshot_row in snapshots_df.iterrows():
        snapshot_date = snapshot_row["date"]
        snapshot_id = snapshot_row["id"]

        # Get investment values for this snapshot
        snapshot_investments = df[
            (df["snapshot_id"] == snapshot_id)
            & (pd.notna(df["account_id"]))
            & (df["type"] == "asset")
            & (df["category"].isin(investment_categories))
        ]
        investment_value = float(snapshot_investments["value"].sum())

        # Calculate cumulative contributions up to this snapshot date
        cumulative_contributions = 0
        if not transactions_with_accounts_df.empty:
            # Filter for investment transactions up to snapshot date
            investment_trans = transactions_with_accounts_df[
                (transactions_with_accounts_df["category"].isin(investment_categories))
                & (transactions_with_accounts_df["date"] <= snapshot_date)
            ]
            cumulative_contributions = float(investment_trans["signed_amount"].sum())

        # Calculate returns
        returns = investment_value - cumulative_contributions

        investment_time_series.append(
            InvestmentTimeSeriesPoint(
                date=snapshot_date,
                value=investment_value,
                contributions=cumulative_contributions,
                returns=returns,
            )
        )

    return investment_time_series


def build_wrapper_time_series(
    df: pd.DataFrame, snapshots_df: pd.DataFrame, transactions_with_accounts_df: pd.DataFrame
) -> WrapperTimeSeries:
    """Per-wrapper (IKE, IKZE, PPK) investment time series."""
    ike_series: list[InvestmentTimeSeriesPoint] = []
    ikze_series: list[InvestmentTimeSeriesPoint] = []
    ppk_series: list[InvestmentTimeSeriesPoint] = []

    if not df.empty and not snapshots_df.empty:
        investment_categories = {"stock", "bond", "fund", "etf", "gold", "ppk"}

        for _, snapshot_row in snapshots_df.iterrows():
            snapshot_date = snapshot_row["date"]
            snapshot_id = snapshot_row["id"]

            # Calculate for each wrapper
            for wrapper, series_list in [
                ("IKE", ike_series),
                ("IKZE", ikze_series),
                ("PPK", ppk_series),
            ]:
                # Get investment values for this wrapper in this snapshot
                wrapper_investments = df[
                    (df["snapshot_id"] == snapshot_id)
                    & (pd.notna(df["account_id"]))
                    & (df["type"] == "asset")
                    & (df["category"].isin(investment_categories))
                    & (df["account_wrapper"] == wrapper)
                ]
                wrapper_value = float(wrapper_investments["value"].sum())

                # Calculate cumulative contributions for this wrapper up to snapshot date
                cumulative_contributions = 0
                if not transactions_with_accounts_df.empty:
                    wrapper_trans = transactions_with_accounts_df[
                        (transactions_with_accounts_df["category"].isin(investment_categories))
                        & (transactions_with_accounts_df["account_wrapper"] == wrapper)
                        & (transactions_with_accounts_df["date"] <= snapshot_date)
                    ]
                    cumulative_contributions = float(wrapper_trans["signed_amount"].sum())

                returns = wrapper_value - cumulative_contributions

                series_list.append(
                    InvestmentTimeSeriesPoint(
                        date=snapshot_date,
                        value=wrapper_value,
                        contributions=cumulative_contributions,
                        returns=returns,
                    )
                )

    return WrapperTimeSeries(ike=ike_series, ikze=ikze_series, ppk=ppk_series)


def build_category_time_series(
    df: pd.DataFrame, snapshots_df: pd.DataFrame, transactions_with_accounts_df: pd.DataFrame
) -> CategoryTimeSeries:
    """Per-category-group (stock, bond) investment time series."""
    stock_series: list[InvestmentTimeSeriesPoint] = []
    bond_series: list[InvestmentTimeSeriesPoint] = []

    if not df.empty and not snapshots_df.empty:
        # Define category grouping (matching allocation logic)
        # Map individual categories to their group for consistent aggregation
        category_to_group = {
            "stock": "stock",
            "fund": "stock",  # fund grouped as stock
            "etf": "stock",  # etf grouped as stock
            "bond": "bond",
        }

        for _, snapshot_row in snapshots_df.iterrows():
            snapshot_date = snapshot_row["date"]
            snapshot_id = snapshot_row["id"]

            # Calculate for each category group
            for target_group, series_list in [("stock", stock_series), ("bond", bond_series)]:
                # Get all categories that map to this group
                matching_categories = [
                    cat for cat, group in category_to_group.items() if group == target_group
                ]

                # Get investment values for this category group in this snapshot
                category_investments = df[
                    (df["snapshot_id"] == snapshot_id)
                    & (pd.notna(df["account_id"]))
                    & (df["type"] == "asset")
                    & (df["category"].isin(matching_categories))
                ]
                category_value = float(category_investments["value"].sum())

                # Calculate cumulative contributions for this category group up to snapshot date
                cumulative_contributions = 0
                if not transactions_with_accounts_df.empty:
                    category_trans = transactions_with_accounts_df[
                        (transactions_with_accounts_df["category"].isin(matching_categories))
                        & (transactions_with_accounts_df["date"] <= snapshot_date)
                    ]
                    cumulative_contributions = float(category_trans["signed_amount"].sum())

                returns = category_value - cumulative_contributions

                series_list.append(
                    InvestmentTimeSeriesPoint(
                        date=snapshot_date,
                        value=category_value,
                        contributions=cumulative_contributions,
                        returns=returns,
                    )
                )

    return CategoryTimeSeries(stock=stock_series, bond=bond_series)
