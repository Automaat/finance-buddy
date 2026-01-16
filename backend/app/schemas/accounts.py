from datetime import datetime

from pydantic import BaseModel


class AccountResponse(BaseModel):
    id: int
    name: str
    type: str
    category: str
    owner: str
    currency: str
    is_active: bool
    created_at: datetime
    current_value: float


class AccountsListResponse(BaseModel):
    assets: list[AccountResponse]
    liabilities: list[AccountResponse]
