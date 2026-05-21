"""Polish gross-to-net salary calculator.

Pure functions, year-keyed constants. Models the most common employment
contract types under Polish tax law:

- UOP (Umowa o pracę): ZUS pracownika (13.71%) + składka zdrowotna 9%
  + PIT 12% / 32% with 30k PLN kwota wolna (free amount).
- B2B liniowy 19%: flat 19% PIT, składka zdrowotna 4.9% partially deductible.
  Assumes ZUS is paid separately (not modeled here).
- B2B ryczałt 12% (IT default): flat % on revenue, tiered składka zdrowotna.
- UZ (zlecenie): ZUS + składka zdrowotna + PIT progresywny with 20% KUP.
- UoD (dzieło): no ZUS / no health, PIT progresywny with 20% KUP.

All functions return annualized breakdowns; pass `gross_monthly * 12` semantics
by accepting a monthly amount and multiplying internally.

Constants are stable since the 2022 PIT reform; the year argument lets callers
opt into future rate changes without code edits.
"""

from dataclasses import dataclass
from decimal import Decimal

from app.core.enums import ContractType


@dataclass(frozen=True)
class TaxConstants:
    year: int
    zus_emerytalne: Decimal
    zus_rentowe: Decimal
    zus_chorobowe: Decimal
    zus_cap_annual: Decimal  # 30× annual average salary — caps emerytalne+rentowe
    health_rate: Decimal
    pit_low_rate: Decimal
    pit_high_rate: Decimal
    pit_threshold_annual: Decimal
    free_amount_annual: Decimal
    kup_monthly: Decimal
    b2b_liniowy_rate: Decimal
    b2b_liniowy_health_rate: Decimal
    b2b_liniowy_health_deduction_cap_annual: Decimal
    ryczalt_it_rate: Decimal
    ryczalt_health_low_monthly: Decimal
    ryczalt_health_mid_monthly: Decimal
    ryczalt_health_high_monthly: Decimal
    ryczalt_low_threshold_annual: Decimal
    ryczalt_high_threshold_annual: Decimal


_CONSTANTS_2024 = TaxConstants(
    year=2024,
    zus_emerytalne=Decimal("0.0976"),
    zus_rentowe=Decimal("0.015"),
    zus_chorobowe=Decimal("0.0245"),
    zus_cap_annual=Decimal("282600"),
    health_rate=Decimal("0.09"),
    pit_low_rate=Decimal("0.12"),
    pit_high_rate=Decimal("0.32"),
    pit_threshold_annual=Decimal("120000"),
    free_amount_annual=Decimal("30000"),
    kup_monthly=Decimal("250"),
    b2b_liniowy_rate=Decimal("0.19"),
    b2b_liniowy_health_rate=Decimal("0.049"),
    b2b_liniowy_health_deduction_cap_annual=Decimal("11600"),
    ryczalt_it_rate=Decimal("0.12"),
    ryczalt_health_low_monthly=Decimal("419.46"),
    ryczalt_health_mid_monthly=Decimal("699.11"),
    ryczalt_health_high_monthly=Decimal("1258.39"),
    ryczalt_low_threshold_annual=Decimal("60000"),
    ryczalt_high_threshold_annual=Decimal("300000"),
)


_BY_YEAR: dict[int, TaxConstants] = {
    2024: _CONSTANTS_2024,
    2025: _CONSTANTS_2024,
    2026: _CONSTANTS_2024,
}


def constants_for(year: int) -> TaxConstants:
    """Look up tax constants for a given year, falling back to latest known."""
    if year in _BY_YEAR:
        return _BY_YEAR[year]
    latest = max(_BY_YEAR.keys())
    return _BY_YEAR[latest]


@dataclass(frozen=True)
class NetBreakdown:
    gross_annual: Decimal
    zus_annual: Decimal
    health_annual: Decimal
    pit_annual: Decimal
    net_annual: Decimal


def _round2(value: Decimal) -> Decimal:
    return value.quantize(Decimal("0.01"))


def _progressive_pit(taxable: Decimal, c: TaxConstants) -> Decimal:
    """Progressive PIT after free amount: 12% then 32% above pit_threshold.

    `taxable` is the income base already reduced by ZUS + KUP. The free amount
    is applied here (not by callers).
    """
    if taxable <= c.free_amount_annual:
        return Decimal("0")
    after_free = taxable - c.free_amount_annual
    low_band_cap = c.pit_threshold_annual - c.free_amount_annual
    if after_free <= low_band_cap:
        return after_free * c.pit_low_rate
    high_band = after_free - low_band_cap
    return low_band_cap * c.pit_low_rate + high_band * c.pit_high_rate


def _zus_employee_annual(gross_annual: Decimal, c: TaxConstants) -> Decimal:
    """ZUS pracownika. Emerytalne + rentowe are capped at 30x annual average;
    chorobowe applies to full gross."""
    capped = min(gross_annual, c.zus_cap_annual)
    return capped * c.zus_emerytalne + capped * c.zus_rentowe + gross_annual * c.zus_chorobowe


def net_uop(gross_monthly: Decimal, year: int) -> NetBreakdown:
    """Compute annual net pay from gross monthly UoP salary."""
    c = constants_for(year)
    gross_annual = gross_monthly * 12

    zus_annual = _zus_employee_annual(gross_annual, c)

    after_zus = gross_annual - zus_annual
    health_annual = after_zus * c.health_rate

    kup_annual = c.kup_monthly * 12
    taxable = max(Decimal("0"), after_zus - kup_annual)
    pit_annual = _progressive_pit(taxable, c)

    net_annual = gross_annual - zus_annual - health_annual - pit_annual

    return NetBreakdown(
        gross_annual=_round2(gross_annual),
        zus_annual=_round2(zus_annual),
        health_annual=_round2(health_annual),
        pit_annual=_round2(pit_annual),
        net_annual=_round2(net_annual),
    )


def net_uz(gross_monthly: Decimal, year: int) -> NetBreakdown:
    """Umowa zlecenie. ZUS + health + progressive PIT with 20% KUP on revenue."""
    c = constants_for(year)
    gross_annual = gross_monthly * 12

    zus_annual = _zus_employee_annual(gross_annual, c)

    after_zus = gross_annual - zus_annual
    health_annual = after_zus * c.health_rate

    kup_annual = gross_annual * Decimal("0.20")
    taxable = max(Decimal("0"), after_zus - kup_annual)
    pit_annual = _progressive_pit(taxable, c)

    net_annual = gross_annual - zus_annual - health_annual - pit_annual

    return NetBreakdown(
        gross_annual=_round2(gross_annual),
        zus_annual=_round2(zus_annual),
        health_annual=_round2(health_annual),
        pit_annual=_round2(pit_annual),
        net_annual=_round2(net_annual),
    )


def net_uod(gross_monthly: Decimal, year: int) -> NetBreakdown:
    """Umowa o dzieło. No ZUS, no health, progressive PIT with 20% KUP."""
    c = constants_for(year)
    gross_annual = gross_monthly * 12

    kup_annual = gross_annual * Decimal("0.20")
    taxable = max(Decimal("0"), gross_annual - kup_annual)
    pit_annual = _progressive_pit(taxable, c)

    net_annual = gross_annual - pit_annual

    return NetBreakdown(
        gross_annual=_round2(gross_annual),
        zus_annual=Decimal("0.00"),
        health_annual=Decimal("0.00"),
        pit_annual=_round2(pit_annual),
        net_annual=_round2(net_annual),
    )


def net_b2b_liniowy(revenue_monthly: Decimal, year: int) -> NetBreakdown:
    """B2B 19% liniowy. ZUS is paid separately and not modeled here."""
    c = constants_for(year)
    revenue_annual = revenue_monthly * 12

    health_annual = revenue_annual * c.b2b_liniowy_health_rate
    deductible = min(health_annual, c.b2b_liniowy_health_deduction_cap_annual)
    pit_base = max(Decimal("0"), revenue_annual - deductible)
    pit_annual = pit_base * c.b2b_liniowy_rate

    net_annual = revenue_annual - health_annual - pit_annual

    return NetBreakdown(
        gross_annual=_round2(revenue_annual),
        zus_annual=Decimal("0.00"),
        health_annual=_round2(health_annual),
        pit_annual=_round2(pit_annual),
        net_annual=_round2(net_annual),
    )


def net_b2b_ryczalt(
    revenue_monthly: Decimal, year: int, rate: Decimal | None = None
) -> NetBreakdown:
    """B2B ryczałt. Defaults to 12% (typical for IT). Tiered składka zdrowotna."""
    c = constants_for(year)
    revenue_annual = revenue_monthly * 12
    effective_rate = rate if rate is not None else c.ryczalt_it_rate

    if revenue_annual <= c.ryczalt_low_threshold_annual:
        health_monthly = c.ryczalt_health_low_monthly
    elif revenue_annual <= c.ryczalt_high_threshold_annual:
        health_monthly = c.ryczalt_health_mid_monthly
    else:
        health_monthly = c.ryczalt_health_high_monthly

    health_annual = health_monthly * 12
    pit_annual = revenue_annual * effective_rate
    net_annual = revenue_annual - health_annual - pit_annual

    return NetBreakdown(
        gross_annual=_round2(revenue_annual),
        zus_annual=Decimal("0.00"),
        health_annual=_round2(health_annual),
        pit_annual=_round2(pit_annual),
        net_annual=_round2(net_annual),
    )


def gross_to_net(
    gross_monthly: Decimal,
    contract_type: ContractType,
    year: int,
) -> NetBreakdown:
    """Dispatch by contract type. B2B defaults to liniowy 19%."""
    if contract_type == ContractType.UOP:
        return net_uop(gross_monthly, year)
    if contract_type == ContractType.B2B:
        return net_b2b_liniowy(gross_monthly, year)
    if contract_type == ContractType.UZ:
        return net_uz(gross_monthly, year)
    if contract_type == ContractType.UOD:
        return net_uod(gross_monthly, year)
    raise ValueError(f"Unknown contract type: {contract_type}")
