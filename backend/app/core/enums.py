"""Domain enums for finance-buddy application."""

from enum import Enum


class AccountType(str, Enum):
    """Account type classification."""

    ASSET = "asset"
    LIABILITY = "liability"


class Category(str, Enum):
    """Account category for asset allocation."""

    BANK = "bank"
    SAVING_ACCOUNT = "saving_account"
    STOCK = "stock"
    BOND = "bond"
    GOLD = "gold"
    REAL_ESTATE = "real_estate"
    PPK = "ppk"
    FUND = "fund"
    ETF = "etf"
    VEHICLE = "vehicle"
    MORTGAGE = "mortgage"
    INSTALLMENT = "installment"
    OTHER = "other"


class Wrapper(str, Enum):
    """Polish retirement account wrappers."""

    IKE = "IKE"
    IKZE = "IKZE"
    PPK = "PPK"


class Purpose(str, Enum):
    """Account purpose classification."""

    RETIREMENT = "retirement"
    EMERGENCY_FUND = "emergency_fund"
    GENERAL = "general"


class DebtType(str, Enum):
    """Debt classification."""

    MORTGAGE = "mortgage"
    INSTALLMENT_0PERCENT = "installment_0percent"


class TransactionType(str, Enum):
    """Retirement transaction types."""

    EMPLOYEE = "employee"
    EMPLOYER = "employer"
    WITHDRAWAL = "withdrawal"
    GOVERNMENT = "government"


class ContractType(str, Enum):
    """Polish employment contract types."""

    UOP = "UOP"  # Umowa o pracę (employment contract)
    UZ = "UZ"  # Umowa zlecenie (contract of mandate)
    UOD = "UoD"  # Umowa o dzieło (contract for specific work)
    B2B = "B2B"  # Business to business
