from datetime import UTC, datetime

from sqlalchemy import select
from sqlalchemy.orm import Session

from app.models.app_config import AppConfig
from app.models.persona import Persona
from app.models.salary_record import SalaryRecord
from app.schemas.zus import (
    SalaryHistoryEntry,
    ZusCalculatorInputs,
    ZusCalculatorResponse,
    ZusPrefillResponse,
    ZusSensitivityScenario,
    ZusYearlyProjection,
)

# ZUS contribution rates (% of gross salary)
CONTRIBUTION_RATE_KONTO = 0.1222  # 12.22% goes to individual account (konto)
CONTRIBUTION_RATE_SUBKONTO_NO_OFE = 0.0730  # 7.30% to subkonto when no OFE
CONTRIBUTION_RATE_SUBKONTO_OFE = 0.0438  # 4.38% to subkonto with OFE (rest goes to OFE)
TOTAL_CONTRIBUTION_RATE = 0.1952  # 19.52% total pension contribution

# 30x average salary cap — annual limit on contribution base (2026 estimate)
CAP_30X_2026 = 282_600.0
CAP_GROWTH_RATE = 0.05  # ~5% annual growth of cap

# GUS 2025 life expectancy in months (dalsze trwanie życia) by retirement age
LIFE_EXPECTANCY_MONTHS: dict[int, float] = {
    55: 318.0,
    56: 312.0,
    57: 306.0,
    58: 300.0,
    59: 294.0,
    60: 266.4,
    61: 260.4,
    62: 254.4,
    63: 248.4,
    64: 234.6,
    65: 220.8,
    66: 214.8,
    67: 208.8,
    68: 202.8,
    69: 196.8,
    70: 190.8,
}

# Pension tax parameters (2026)
PIT_RATE = 0.12  # 12% PIT
HEALTH_RATE = 0.09  # 9% health insurance
ANNUAL_FREE_THRESHOLD = 30_000.0  # Tax-free amount


def apply_pension_tax(monthly_gross: float) -> float:
    """Calculate net pension from gross pension.

    Pension taxation: 12% PIT (with 30k/year free threshold) + 9% health insurance.
    Health insurance is deducted from gross before PIT calculation.
    """
    annual_gross = monthly_gross * 12

    # Health insurance (9% of gross, not tax-deductible for pensioners)
    annual_health = annual_gross * HEALTH_RATE

    # PIT: 12% on (gross - free threshold), but not below 0
    taxable_income = max(0.0, annual_gross - ANNUAL_FREE_THRESHOLD)
    annual_pit = taxable_income * PIT_RATE

    annual_net = annual_gross - annual_health - annual_pit
    return max(0, annual_net / 12)


def get_life_expectancy(retirement_age: int) -> float:
    """Look up life expectancy months from GUS table, interpolate if needed."""
    if retirement_age in LIFE_EXPECTANCY_MONTHS:
        return LIFE_EXPECTANCY_MONTHS[retirement_age]
    # Clamp to table bounds
    if retirement_age < 55:
        return LIFE_EXPECTANCY_MONTHS[55]
    if retirement_age > 70:
        return LIFE_EXPECTANCY_MONTHS[70]
    # Linear interpolation between two nearest ages
    lower = max(k for k in LIFE_EXPECTANCY_MONTHS if k <= retirement_age)
    upper = min(k for k in LIFE_EXPECTANCY_MONTHS if k >= retirement_age)
    if lower == upper:
        return LIFE_EXPECTANCY_MONTHS[lower]
    frac = (retirement_age - lower) / (upper - lower)
    return LIFE_EXPECTANCY_MONTHS[lower] + frac * (
        LIFE_EXPECTANCY_MONTHS[upper] - LIFE_EXPECTANCY_MONTHS[lower]
    )


def calculate_zus_pension(inputs: ZusCalculatorInputs) -> ZusCalculatorResponse:
    """Core ZUS pension calculation.

    Formula: Emerytura = (Kapitał Początkowy + Suma Zwaloryzowanych Składek) / Dalsze Trwanie Życia

    Steps per year:
    1. Determine salary (from history or projected)
    2. Apply 30x cap
    3. Compute konto + subkonto contributions
    4. Valorize existing balances
    5. Add new contributions
    6. Apply suwak bezpieczeństwa (10 years before retirement)
    """
    today = datetime.now(UTC).date()
    current_year = today.year
    birth_year = inputs.birth_date.year
    retirement_year = birth_year + inputs.retirement_age

    subkonto_rate = (
        CONTRIBUTION_RATE_SUBKONTO_OFE if inputs.has_ofe else CONTRIBUTION_RATE_SUBKONTO_NO_OFE
    )

    # Build salary lookup from history
    salary_by_year: dict[int, float] = {}
    for entry in inputs.salary_history:
        salary_by_year[entry.year] = entry.annual_gross

    konto_balance = 0.0
    subkonto_balance = 0.0
    projections: list[ZusYearlyProjection] = []
    last_salary = inputs.current_gross_monthly_salary * 12

    # Valorize kapitał początkowy alongside konto balance
    kapital_valorized = inputs.kapital_poczatkowy

    for year in range(inputs.work_start_year, retirement_year):
        age = year - birth_year

        # Determine annual gross salary for this year
        if year in salary_by_year:
            annual_salary = salary_by_year[year]
        elif year <= current_year:
            # Past year with no record — use current salary as estimate
            annual_salary = inputs.current_gross_monthly_salary * 12
        else:
            # Future year — project from current salary with growth
            years_ahead = year - current_year
            annual_salary = (
                inputs.current_gross_monthly_salary
                * 12
                * (1 + inputs.salary_growth_rate / 100) ** years_ahead
            )

        last_salary = annual_salary

        # 30x cap grows annually from 2026 base
        years_from_base = year - 2026
        cap = CAP_30X_2026 * (1 + CAP_GROWTH_RATE) ** max(0, years_from_base)
        salary_capped = annual_salary > cap
        capped_salary = min(annual_salary, cap)

        # Contributions
        contribution_konto = capped_salary * CONTRIBUTION_RATE_KONTO
        contribution_subkonto = capped_salary * subkonto_rate

        # Suwak bezpieczeństwa: 10 years before retirement, subkonto contributions → konto
        suwak_active = (retirement_year - year) <= 10
        if suwak_active:
            contribution_konto += contribution_subkonto
            contribution_subkonto = 0.0

        # Valorize existing balances (applied before adding new contributions)
        konto_balance *= 1 + inputs.valorization_rate_konto / 100
        subkonto_balance *= 1 + inputs.valorization_rate_subkonto / 100
        kapital_valorized *= 1 + inputs.valorization_rate_konto / 100

        # Add new contributions after valorization
        konto_balance += contribution_konto
        subkonto_balance += contribution_subkonto

        projections.append(
            ZusYearlyProjection(
                year=year,
                age=age,
                annual_gross_salary=round(annual_salary, 2),
                salary_capped=salary_capped,
                contribution_konto=round(contribution_konto, 2),
                contribution_subkonto=round(contribution_subkonto, 2),
                konto_balance=round(konto_balance, 2),
                subkonto_balance=round(subkonto_balance, 2),
                total_balance=round(konto_balance + subkonto_balance + kapital_valorized, 2),
            )
        )

    # At retirement
    life_expectancy = get_life_expectancy(inputs.retirement_age)
    total_capital = kapital_valorized + konto_balance + subkonto_balance
    monthly_pension_gross = total_capital / life_expectancy
    monthly_pension_net = apply_pension_tax(monthly_pension_gross)
    replacement_rate = (monthly_pension_gross / (last_salary / 12) * 100) if last_salary > 0 else 0

    # Sensitivity analysis: ±2% valorization
    sensitivity = _build_sensitivity(inputs, retirement_year, salary_by_year, life_expectancy)

    return ZusCalculatorResponse(
        inputs=inputs,
        yearly_projections=projections,
        life_expectancy_months=life_expectancy,
        konto_at_retirement=round(konto_balance, 2),
        subkonto_at_retirement=round(subkonto_balance, 2),
        kapital_poczatkowy_valorized=round(kapital_valorized, 2),
        total_capital=round(total_capital, 2),
        monthly_pension_gross=round(monthly_pension_gross, 2),
        monthly_pension_net=round(monthly_pension_net, 2),
        replacement_rate=round(replacement_rate, 2),
        last_gross_salary=round(last_salary / 12, 2),
        sensitivity=sensitivity,
    )


def _build_sensitivity(
    inputs: ZusCalculatorInputs,
    retirement_year: int,
    salary_by_year: dict[int, float],
    life_expectancy: float,
) -> list[ZusSensitivityScenario]:
    """Run 3 scenarios: pessimistic (-2%), baseline, optimistic (+2%)."""
    scenarios = [
        ("Pesymistyczny", -2.0),
        ("Bazowy", 0.0),
        ("Optymistyczny", 2.0),
    ]
    results: list[ZusSensitivityScenario] = []

    for label, delta in scenarios:
        val_konto = inputs.valorization_rate_konto + delta
        val_subkonto = inputs.valorization_rate_subkonto + delta
        # Re-run simplified calculation with adjusted rates
        total = _quick_calculate(inputs, retirement_year, salary_by_year, val_konto, val_subkonto)
        gross = total / life_expectancy
        net = apply_pension_tax(gross)
        last_salary = _get_last_salary(inputs, retirement_year)
        rr = (gross / last_salary * 100) if last_salary > 0 else 0

        results.append(
            ZusSensitivityScenario(
                label=label,
                valorization_konto=val_konto,
                valorization_subkonto=val_subkonto,
                monthly_pension_gross=round(gross, 2),
                monthly_pension_net=round(net, 2),
                replacement_rate=round(rr, 2),
            )
        )

    return results


def _get_last_salary(inputs: ZusCalculatorInputs, retirement_year: int) -> float:
    """Get projected monthly salary at retirement."""
    current_year = datetime.now(UTC).date().year
    years_ahead = retirement_year - 1 - current_year
    if years_ahead <= 0:
        return inputs.current_gross_monthly_salary
    growth = (1 + inputs.salary_growth_rate / 100) ** years_ahead
    return inputs.current_gross_monthly_salary * growth


def _quick_calculate(
    inputs: ZusCalculatorInputs,
    retirement_year: int,
    salary_by_year: dict[int, float],
    val_konto: float,
    val_subkonto: float,
) -> float:
    """Simplified total capital calculation for sensitivity analysis."""
    current_year = datetime.now(UTC).date().year
    subkonto_rate = (
        CONTRIBUTION_RATE_SUBKONTO_OFE if inputs.has_ofe else CONTRIBUTION_RATE_SUBKONTO_NO_OFE
    )

    konto = 0.0
    subkonto = 0.0
    kapital = inputs.kapital_poczatkowy

    for year in range(inputs.work_start_year, retirement_year):
        if year in salary_by_year:
            annual_salary = salary_by_year[year]
        elif year <= current_year:
            annual_salary = inputs.current_gross_monthly_salary * 12
        else:
            years_ahead = year - current_year
            annual_salary = (
                inputs.current_gross_monthly_salary
                * 12
                * (1 + inputs.salary_growth_rate / 100) ** years_ahead
            )

        years_from_base = year - 2026
        cap = CAP_30X_2026 * (1 + CAP_GROWTH_RATE) ** max(0, years_from_base)
        capped = min(annual_salary, cap)

        c_konto = capped * CONTRIBUTION_RATE_KONTO
        c_subkonto = capped * subkonto_rate

        suwak = (retirement_year - year) <= 10
        if suwak:
            c_konto += c_subkonto
            c_subkonto = 0.0

        konto *= 1 + val_konto / 100
        subkonto *= 1 + val_subkonto / 100
        kapital *= 1 + val_konto / 100

        konto += c_konto
        subkonto += c_subkonto

    return kapital + konto + subkonto


def get_zus_prefill(db: Session, owner: str | None = None) -> ZusPrefillResponse:
    """Prefill ZUS calculator form from SalaryRecord + AppConfig."""
    config = db.execute(select(AppConfig)).scalar_one_or_none()

    # Find first persona if owner not specified
    if not owner:
        persona = db.execute(select(Persona).order_by(Persona.name)).scalars().first()
        owner = persona.name if persona else None

    if not owner:
        return ZusPrefillResponse()

    # Get latest active salary for the owner
    salary = (
        db.execute(
            select(SalaryRecord)
            .where(SalaryRecord.owner == owner, SalaryRecord.is_active.is_(True))
            .order_by(SalaryRecord.date.desc())
        )
        .scalars()
        .first()
    )

    # Get salary history — all records for the owner grouped by year
    all_salaries = (
        db.execute(
            select(SalaryRecord).where(SalaryRecord.owner == owner).order_by(SalaryRecord.date)
        )
        .scalars()
        .all()
    )

    # Build yearly salary history (use latest record per year)
    yearly: dict[int, float] = {}
    for s in all_salaries:
        yearly[s.date.year] = float(s.gross_amount) * 12

    salary_history = [SalaryHistoryEntry(year=y, annual_gross=g) for y, g in sorted(yearly.items())]

    # Determine work_start_year from earliest salary record
    work_start_year = min(yearly.keys()) if yearly else None

    return ZusPrefillResponse(
        birth_date=config.birth_date if config else None,
        retirement_age=config.retirement_age if config else 65,
        gender="M",
        current_gross_monthly_salary=float(salary.gross_amount) if salary else None,
        owner=owner,
        salary_history=salary_history,
        work_start_year=work_start_year,
    )
