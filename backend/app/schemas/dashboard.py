from datetime import date

from pydantic import BaseModel


class NetWorthPoint(BaseModel):
    date: date
    value: float


class AllocationItem(BaseModel):
    category: str
    owner: str | None
    value: float


class DashboardResponse(BaseModel):
    net_worth_history: list[NetWorthPoint]
    current_net_worth: float
    change_vs_last_month: float
    total_assets: float
    total_liabilities: float
    allocation: list[AllocationItem]
    retirement_account_value: float
