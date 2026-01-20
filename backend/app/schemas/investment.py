from pydantic import BaseModel

from app.core.enums import Category


class CategoryStatsResponse(BaseModel):
    category: Category
    total_value: float
    total_contributed: float
    returns: float
    roi_percentage: float
