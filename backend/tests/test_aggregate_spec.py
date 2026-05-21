"""Tests for the pure aggregate spec (no ORM, no DB)."""

from dataclasses import dataclass
from datetime import date
from decimal import Decimal

from app.services.aggregate_spec import _first_of_month, compute_aggregates


@dataclass
class _Snap:
    id: int
    date: date


@dataclass
class _SV:
    asset_id: int | None
    account_id: int | None
    value: Decimal


@dataclass
class _Account:
    id: int
    owner: str
    type: str
    category: str


@dataclass
class _Asset:
    id: int


def test_empty_snapshot_returns_shared_zero_row():
    snap = _Snap(id=1, date=date(2024, 1, 1))
    rows = compute_aggregates(snap, [], [], [])
    assert len(rows) == 1
    row = rows[0]
    assert row.owner == "Shared"
    assert row.total_assets == Decimal("0")
    assert row.total_liabilities == Decimal("0")
    assert row.net_worth == Decimal("0")
    assert row.allocation == []
    assert row.month == date(2024, 1, 1)
    assert row.snapshot_id == 1


def test_asset_table_entry_goes_to_shared():
    snap = _Snap(id=1, date=date(2024, 3, 15))
    sv = _SV(asset_id=10, account_id=None, value=Decimal("5000"))
    asset = _Asset(id=10)
    rows = compute_aggregates(snap, [sv], [], [asset])
    assert len(rows) == 1
    assert rows[0].owner == "Shared"
    assert rows[0].total_assets == Decimal("5000")
    assert rows[0].total_liabilities == Decimal("0")
    assert rows[0].net_worth == Decimal("5000")
    # Month is first of March
    assert rows[0].month == date(2024, 3, 1)


def test_inactive_asset_is_ignored():
    snap = _Snap(id=1, date=date(2024, 1, 1))
    sv = _SV(asset_id=99, account_id=None, value=Decimal("1000"))
    # asset_rows is empty → 99 not in active_asset_ids
    rows = compute_aggregates(snap, [sv], [], [])
    assert len(rows) == 1
    assert rows[0].owner == "Shared"
    assert rows[0].total_assets == Decimal("0")


def test_account_asset_contributes_positively():
    snap = _Snap(id=1, date=date(2024, 1, 1))
    acct = _Account(id=1, owner="Marcin", type="asset", category="bank")
    sv = _SV(asset_id=None, account_id=1, value=Decimal("10000"))
    rows = compute_aggregates(snap, [sv], [acct], [])
    assert len(rows) == 1
    assert rows[0].owner == "Marcin"
    assert rows[0].total_assets == Decimal("10000")
    assert rows[0].total_liabilities == Decimal("0")
    assert rows[0].net_worth == Decimal("10000")
    assert rows[0].allocation == [{"category": "bank", "value": Decimal("10000")}]


def test_account_liability_contributes_to_liabilities():
    snap = _Snap(id=1, date=date(2024, 1, 1))
    acct = _Account(id=2, owner="Shared", type="liability", category="mortgage")
    sv = _SV(asset_id=None, account_id=2, value=Decimal("200000"))
    rows = compute_aggregates(snap, [sv], [acct], [])
    assert len(rows) == 1
    assert rows[0].owner == "Shared"
    assert rows[0].total_assets == Decimal("0")
    assert rows[0].total_liabilities == Decimal("200000")
    assert rows[0].net_worth == Decimal("-200000")
    assert rows[0].allocation == []


def test_multi_owner_produces_separate_rows():
    snap = _Snap(id=1, date=date(2024, 1, 1))
    acct_m = _Account(id=1, owner="Marcin", type="asset", category="bank")
    acct_e = _Account(id=2, owner="Ewa", type="asset", category="stock")
    svs = [
        _SV(asset_id=None, account_id=1, value=Decimal("5000")),
        _SV(asset_id=None, account_id=2, value=Decimal("3000")),
    ]
    rows = compute_aggregates(snap, svs, [acct_m, acct_e], [])
    assert len(rows) == 2
    owners = {r.owner for r in rows}
    assert owners == {"Marcin", "Ewa"}
    for row in rows:
        if row.owner == "Marcin":
            assert row.total_assets == Decimal("5000")
            assert row.allocation == [{"category": "bank", "value": Decimal("5000")}]
        else:
            assert row.total_assets == Decimal("3000")
            assert row.allocation == [{"category": "stock", "value": Decimal("3000")}]


def test_allocation_sorted_by_category():
    snap = _Snap(id=1, date=date(2024, 1, 1))
    acct = _Account(id=1, owner="Marcin", type="asset", category="zzz")
    acct2 = _Account(id=2, owner="Marcin", type="asset", category="aaa")
    svs = [
        _SV(asset_id=None, account_id=1, value=Decimal("100")),
        _SV(asset_id=None, account_id=2, value=Decimal("200")),
    ]
    rows = compute_aggregates(snap, svs, [acct, acct2], [])
    assert len(rows) == 1
    alloc = rows[0].allocation
    assert alloc[0]["category"] == "aaa"
    assert alloc[1]["category"] == "zzz"


def test_asset_table_and_account_mixed():
    snap = _Snap(id=1, date=date(2024, 1, 1))
    asset = _Asset(id=5)
    acct = _Account(id=1, owner="Marcin", type="asset", category="bank")
    svs = [
        _SV(asset_id=5, account_id=None, value=Decimal("2000")),
        _SV(asset_id=None, account_id=1, value=Decimal("8000")),
    ]
    rows = compute_aggregates(snap, svs, [acct], [asset])
    owners = {r.owner: r for r in rows}
    assert "Shared" in owners
    assert "Marcin" in owners
    assert owners["Shared"].total_assets == Decimal("2000")
    assert owners["Shared"].allocation == []
    assert owners["Marcin"].total_assets == Decimal("8000")


def test_inactive_account_is_ignored():
    snap = _Snap(id=1, date=date(2024, 1, 1))
    acct_active = _Account(id=1, owner="Marcin", type="asset", category="bank")
    svs = [
        _SV(asset_id=None, account_id=1, value=Decimal("1000")),
        _SV(asset_id=None, account_id=99, value=Decimal("999")),  # no matching account
    ]
    rows = compute_aggregates(snap, svs, [acct_active], [])
    assert len(rows) == 1
    assert rows[0].total_assets == Decimal("1000")


def test_first_of_month():
    assert _first_of_month(date(2024, 3, 15)) == date(2024, 3, 1)
    assert _first_of_month(date(2024, 1, 1)) == date(2024, 1, 1)
    assert _first_of_month(date(2024, 12, 31)) == date(2024, 12, 1)


def test_signed_value_net_worth():
    snap = _Snap(id=1, date=date(2024, 1, 1))
    asset_acct = _Account(id=1, owner="Marcin", type="asset", category="bank")
    liab_acct = _Account(id=2, owner="Marcin", type="liability", category="mortgage")
    svs = [
        _SV(asset_id=None, account_id=1, value=Decimal("100000")),
        _SV(asset_id=None, account_id=2, value=Decimal("30000")),
    ]
    rows = compute_aggregates(snap, svs, [asset_acct, liab_acct], [])
    assert len(rows) == 1
    row = rows[0]
    assert row.total_assets == Decimal("100000")
    assert row.total_liabilities == Decimal("30000")
    assert row.net_worth == Decimal("70000")
