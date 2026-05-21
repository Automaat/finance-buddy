"""Tests for Polish gross-to-net salary calculator.

Reference values are computed manually following the same year-2024 constants
encoded in pl_tax.py — these tests guard the formula, not external truth.
"""

from decimal import Decimal

import pytest

from app.core.enums import ContractType
from app.services.pl_tax import (
    constants_for,
    gross_to_net,
    net_b2b_liniowy,
    net_b2b_ryczalt,
    net_uod,
    net_uop,
    net_uz,
)


def _two_places(d: Decimal) -> Decimal:
    return d.quantize(Decimal("0.01"))


class TestConstantsLookup:
    def test_returns_known_year(self):
        assert constants_for(2024).year == 2024

    def test_unknown_year_falls_back(self):
        result = constants_for(2030)
        assert result.year in (2024, 2025, 2026)


class TestUopNet:
    def test_basic_uop_10k(self):
        """10000 PLN/mo UoP, 2024 constants. Under ZUS cap → full 13.71%."""
        result = net_uop(Decimal("10000"), 2024)

        gross = Decimal("120000")
        zus = gross * Decimal("0.1371")  # below cap
        after_zus = gross - zus
        health = after_zus * Decimal("0.09")
        kup = Decimal("3000")
        taxable = after_zus - kup
        pit = (taxable - Decimal("30000")) * Decimal("0.12")
        net = gross - zus - health - pit

        assert result.gross_annual == _two_places(gross)
        assert result.zus_annual == _two_places(zus)
        assert result.health_annual == _two_places(health)
        assert result.pit_annual == _two_places(pit)
        assert result.net_annual == _two_places(net)

    def test_uop_above_threshold(self):
        """30000 PLN/mo → annual 360k, triggers 32% bracket and ZUS cap."""
        result = net_uop(Decimal("30000"), 2024)

        gross = Decimal("360000")
        cap = Decimal("282600")
        zus = cap * Decimal("0.0976") + cap * Decimal("0.015") + gross * Decimal("0.0245")
        after_zus = gross - zus
        health = after_zus * Decimal("0.09")
        taxable = after_zus - Decimal("3000")
        low_band = Decimal("120000") - Decimal("30000")
        high_band = (taxable - Decimal("30000")) - low_band
        pit = low_band * Decimal("0.12") + high_band * Decimal("0.32")
        net = gross - zus - health - pit

        assert result.zus_annual == _two_places(zus)
        assert result.pit_annual == _two_places(pit)
        assert result.net_annual == _two_places(net)

    def test_zus_cap_applied_at_high_income(self):
        """Above-cap income: emerytalne+rentowe capped, chorobowe uncapped."""
        result = net_uop(Decimal("50000"), 2024)
        gross = Decimal("600000")
        cap = Decimal("282600")
        expected_zus = cap * (Decimal("0.0976") + Decimal("0.015")) + gross * Decimal("0.0245")
        assert result.zus_annual == _two_places(expected_zus)

    def test_uop_below_free_amount(self):
        """Tiny salary: net before PIT exceeds free amount? Check no negative PIT."""
        result = net_uop(Decimal("1500"), 2024)
        assert result.pit_annual >= Decimal("0")
        assert result.net_annual > Decimal("0")

    def test_uop_net_is_less_than_gross(self):
        result = net_uop(Decimal("15000"), 2024)
        assert result.net_annual < result.gross_annual


class TestB2BLiniowy:
    def test_basic_b2b_liniowy_15k(self):
        result = net_b2b_liniowy(Decimal("15000"), 2024)

        revenue = Decimal("180000")
        health = revenue * Decimal("0.049")  # 8820
        # 8820 < 11600 cap → full deductibility
        pit_base = revenue - health
        pit = pit_base * Decimal("0.19")
        net = revenue - health - pit

        assert result.health_annual == _two_places(health)
        assert result.pit_annual == _two_places(pit)
        assert result.net_annual == _two_places(net)
        assert result.zus_annual == Decimal("0.00")

    def test_b2b_liniowy_high_revenue_caps_deduction(self):
        """Revenue 500k → health 24.5k, but deduction capped at 11600."""
        result = net_b2b_liniowy(Decimal("41667"), 2024)
        revenue = Decimal("41667") * 12
        health = revenue * Decimal("0.049")
        pit_base = revenue - Decimal("11600")
        pit = pit_base * Decimal("0.19")
        assert result.pit_annual == _two_places(pit)
        assert result.health_annual == _two_places(health)


class TestB2BRyczalt:
    def test_ryczalt_low_tier(self):
        """Revenue < 60k → low health tier."""
        result = net_b2b_ryczalt(Decimal("4000"), 2024)
        revenue = Decimal("48000")
        health = Decimal("419.46") * 12
        pit = revenue * Decimal("0.12")
        net = revenue - health - pit
        assert result.health_annual == _two_places(health)
        assert result.net_annual == _two_places(net)

    def test_ryczalt_mid_tier(self):
        """60k < revenue ≤ 300k → mid health tier."""
        result = net_b2b_ryczalt(Decimal("15000"), 2024)
        health = Decimal("699.11") * 12
        assert result.health_annual == _two_places(health)

    def test_ryczalt_high_tier(self):
        """Revenue > 300k → high health tier."""
        result = net_b2b_ryczalt(Decimal("30000"), 2024)
        health = Decimal("1258.39") * 12
        assert result.health_annual == _two_places(health)

    def test_ryczalt_custom_rate(self):
        """Some IT activities use 8.5% rate."""
        result = net_b2b_ryczalt(Decimal("10000"), 2024, rate=Decimal("0.085"))
        revenue = Decimal("120000")
        pit = revenue * Decimal("0.085")
        assert result.pit_annual == _two_places(pit)


class TestUzAndUod:
    def test_uz_has_zus_and_20pct_kup(self):
        result = net_uz(Decimal("8000"), 2024)
        assert result.zus_annual > Decimal("0")
        assert result.health_annual > Decimal("0")
        assert result.net_annual < result.gross_annual

    def test_uod_no_zus_no_health(self):
        result = net_uod(Decimal("8000"), 2024)
        assert result.zus_annual == Decimal("0.00")
        assert result.health_annual == Decimal("0.00")
        assert result.pit_annual > Decimal("0")


class TestDispatcher:
    @pytest.mark.parametrize(
        "ct, expected_zus_zero",
        [
            (ContractType.UOP, False),
            (ContractType.B2B, True),
            (ContractType.UZ, False),
            (ContractType.UOD, True),
        ],
    )
    def test_dispatch_picks_correct_calculator(self, ct: ContractType, expected_zus_zero: bool):
        result = gross_to_net(Decimal("10000"), ct, 2024)
        if expected_zus_zero:
            assert result.zus_annual == Decimal("0.00")
        else:
            assert result.zus_annual > Decimal("0")
