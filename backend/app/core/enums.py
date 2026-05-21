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


class BonusType(str, Enum):
    """Compensation bonus event types."""

    ANNUAL = "annual"  # Annual performance bonus
    SIGNON = "signon"  # Sign-on / joining bonus
    SPOT = "spot"  # Discretionary spot bonus
    RETENTION = "retention"  # Retention / stay bonus


class EquityGrantType(str, Enum):
    """Equity grant instrument."""

    OPTION = "option"  # Stock options (ISO/NSO/ESOP)
    RSU = "rsu"  # Restricted stock units


class VestingFrequency(str, Enum):
    """Vesting cadence after the cliff."""

    MONTHLY = "monthly"
    QUARTERLY = "quarterly"
    YEARLY = "yearly"


class EquityTaxTreatment(str, Enum):
    """Polish tax treatment for equity grants."""

    CAPITAL_GAINS_19 = "capital_gains_19"  # art. 24 ust. 11 — deferred to sale, 19%
    EMPLOYMENT_INCOME = "employment_income"  # PIT 12/32 + ZUS at vest


class ValuationSource(str, Enum):
    """Source of a private-company FMV estimate."""

    A_409A = "409a"  # Official 409A appraisal
    PREFERRED_ROUND = "preferred_round"  # Last preferred share round price
    TENDER = "tender"  # Secondary tender offer
    ESTIMATE = "estimate"  # User estimate / unverified
