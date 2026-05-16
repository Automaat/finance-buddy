from typing import Annotated

from fastapi import APIRouter, Depends, HTTPException
from sqlalchemy import func, select, update
from sqlalchemy.exc import IntegrityError
from sqlalchemy.orm import Session

from app.core.database import get_db
from app.models.account import Account
from app.models.debt_payment import DebtPayment
from app.models.persona import Persona
from app.models.retirement_limit import RetirementLimit
from app.models.salary_record import SalaryRecord
from app.models.transaction import Transaction
from app.schemas.personas import PersonaCreate, PersonaResponse, PersonaUpdate

router = APIRouter(prefix="/api/personas", tags=["personas"])


@router.get("", response_model=list[PersonaResponse])
def list_personas(db: Annotated[Session, Depends(get_db)]) -> list[PersonaResponse]:
    personas = db.execute(select(Persona).order_by(Persona.name)).scalars().all()
    return [PersonaResponse.model_validate(p, from_attributes=True) for p in personas]


@router.post("", response_model=PersonaResponse, status_code=201)
def create_persona(data: PersonaCreate, db: Annotated[Session, Depends(get_db)]) -> PersonaResponse:
    persona = Persona(
        name=data.name,
        ppk_employee_rate=data.ppk_employee_rate,
        ppk_employer_rate=data.ppk_employer_rate,
    )
    try:
        db.add(persona)
        db.commit()
        db.refresh(persona)
    except IntegrityError as e:
        db.rollback()
        raise HTTPException(status_code=409, detail=f"Persona '{data.name}' already exists") from e
    return PersonaResponse.model_validate(persona, from_attributes=True)


@router.put("/{persona_id}", response_model=PersonaResponse)
def update_persona(
    persona_id: int, data: PersonaUpdate, db: Annotated[Session, Depends(get_db)]
) -> PersonaResponse:
    persona = db.execute(select(Persona).where(Persona.id == persona_id)).scalar_one_or_none()
    if not persona:
        raise HTTPException(status_code=404, detail="Persona not found")

    old_name = persona.name

    if data.name is not None:
        persona.name = data.name
    if data.ppk_employee_rate is not None:
        persona.ppk_employee_rate = data.ppk_employee_rate
    if data.ppk_employer_rate is not None:
        persona.ppk_employer_rate = data.ppk_employer_rate

    try:
        if data.name is not None and data.name != old_name:
            db.execute(update(Account).where(Account.owner == old_name).values(owner=data.name))
            db.execute(
                update(Transaction).where(Transaction.owner == old_name).values(owner=data.name)
            )
            db.execute(
                update(SalaryRecord).where(SalaryRecord.owner == old_name).values(owner=data.name)
            )
            db.execute(
                update(DebtPayment).where(DebtPayment.owner == old_name).values(owner=data.name)
            )
            db.execute(
                update(RetirementLimit)
                .where(RetirementLimit.owner == old_name)
                .values(owner=data.name)
            )
        db.commit()
        db.refresh(persona)
    except IntegrityError as e:
        db.rollback()
        raise HTTPException(status_code=409, detail=f"Persona '{data.name}' already exists") from e
    return PersonaResponse.model_validate(persona, from_attributes=True)


@router.delete("/{persona_id}", status_code=204)
def delete_persona(persona_id: int, db: Annotated[Session, Depends(get_db)]) -> None:
    persona = db.execute(select(Persona).where(Persona.id == persona_id)).scalar_one_or_none()
    if not persona:
        raise HTTPException(status_code=404, detail="Persona not found")

    conflicts: list[str] = []
    account_count = db.scalar(
        select(func.count()).select_from(Account).where(Account.owner == persona.name)
    )
    if account_count:
        conflicts.append(f"{account_count} account(s)")
    transaction_count = db.scalar(
        select(func.count()).select_from(Transaction).where(Transaction.owner == persona.name)
    )
    if transaction_count:
        conflicts.append(f"{transaction_count} transaction(s)")
    salary_count = db.scalar(
        select(func.count()).select_from(SalaryRecord).where(SalaryRecord.owner == persona.name)
    )
    if salary_count:
        conflicts.append(f"{salary_count} salary record(s)")
    payment_count = db.scalar(
        select(func.count()).select_from(DebtPayment).where(DebtPayment.owner == persona.name)
    )
    if payment_count:
        conflicts.append(f"{payment_count} debt payment(s)")
    limit_count = db.scalar(
        select(func.count())
        .select_from(RetirementLimit)
        .where(RetirementLimit.owner == persona.name)
    )
    if limit_count:
        conflicts.append(f"{limit_count} retirement limit(s)")

    if conflicts:
        raise HTTPException(
            status_code=409,
            detail=f"Cannot delete persona '{persona.name}': referenced by {', '.join(conflicts)}",
        )

    db.delete(persona)
    db.commit()
