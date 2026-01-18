from datetime import date

from pydantic import BaseModel


class NetWorthPoint(BaseModel):
    date: date
    value: float


class AllocationItem(BaseModel):
    category: str
    owner: str | None
    value: float


class MetricCards(BaseModel):
    property_sqm: float
    emergency_fund_months: float
    retirement_income_monthly: float
    mortgage_remaining: float
    mortgage_months_left: int
    mortgage_years_left: float
    retirement_total: float
    investment_contributions: float
    investment_returns: float


class AllocationBreakdown(BaseModel):
    category: str
    current_value: float
    current_percentage: float
    target_percentage: float
    difference: float


class AccountWrapperBreakdown(BaseModel):
    wrapper: str
    value: float
    percentage: float


class RebalancingSuggestion(BaseModel):
    category: str
    action: str  # "buy" or "sell"
    amount: float


class AllocationAnalysis(BaseModel):
    by_category: list[AllocationBreakdown]
    by_wrapper: list[AccountWrapperBreakdown]
    rebalancing: list[RebalancingSuggestion]
    total_investment_value: float


class InvestmentTimeSeriesPoint(BaseModel):
    date: date
    value: float
    contributions: float
    returns: float


class WrapperTimeSeries(BaseModel):
    ike: list[InvestmentTimeSeriesPoint]
    ikze: list[InvestmentTimeSeriesPoint]
    ppk: list[InvestmentTimeSeriesPoint]


class CategoryTimeSeries(BaseModel):
    stock: list[InvestmentTimeSeriesPoint]
    bond: list[InvestmentTimeSeriesPoint]


class DashboardResponse(BaseModel):
    net_worth_history: list[NetWorthPoint]
    current_net_worth: float
    change_vs_last_month: float
    total_assets: float
    total_liabilities: float
    allocation: list[AllocationItem]
    retirement_account_value: float
    metric_cards: MetricCards
    allocation_analysis: AllocationAnalysis
    investment_time_series: list[InvestmentTimeSeriesPoint]
    wrapper_time_series: WrapperTimeSeries
    category_time_series: CategoryTimeSeries
