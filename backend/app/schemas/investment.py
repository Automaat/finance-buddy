from pydantic import BaseModel


class CategoryStatsResponse(BaseModel):
    category: str
    total_value: float
    total_contributed: float
    returns: float
    roi_percentage: float
