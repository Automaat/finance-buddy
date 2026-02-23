"""Pydantic validator utilities for common validation patterns."""

from datetime import UTC, date, datetime


def validate_positive_amount(v: float, field_name: str = "Amount") -> float:
    """
    Validate amount is positive (> 0).

    Args:
        v: Amount value
        field_name: Name of the field for error message (default: "Amount")

    Returns:
        Validated amount

    Raises:
        ValueError: If amount <= 0
    """
    if v <= 0:
        raise ValueError(f"{field_name} must be greater than 0")
    return v


def validate_non_negative_amount(v: float, field_name: str = "Amount") -> float:
    """
    Validate amount is non-negative (>= 0).

    Args:
        v: Amount value
        field_name: Name of the field for error message (default: "Amount")

    Returns:
        Validated amount

    Raises:
        ValueError: If amount < 0
    """
    if v < 0:
        raise ValueError(f"{field_name} must be non-negative")
    return v


def validate_not_future_date(v: date, field_name: str = "Date") -> date:
    """
    Validate date is not in the future.

    Args:
        v: Date value
        field_name: Name of the field for error message (default: "Date")

    Returns:
        Validated date

    Raises:
        ValueError: If date > today (UTC)
    """
    if v > datetime.now(UTC).date():
        raise ValueError(f"{field_name} cannot be in the future")
    return v


def validate_not_empty_string(v: str | None, field_name: str = "Name") -> str | None:
    """
    Validate string is not empty (after stripping whitespace).

    Handles nullable fields - returns None if input is None.

    Args:
        v: String value (nullable)
        field_name: Name of the field for error message (default: "Name")

    Returns:
        Stripped string or None

    Raises:
        ValueError: If string is empty after stripping
    """
    if v is None:
        return None
    stripped = v.strip()
    if not stripped:
        raise ValueError(f"{field_name} cannot be empty")
    return stripped
