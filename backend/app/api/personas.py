from typing import Annotated

from fastapi import APIRouter, Depends, HTTPException
from sqlalchemy.exc import IntegrityError
from sqlalchemy.orm import Session

from app.core.database import get_db
from app.models.account import Account
from app.models.persona import Persona
from app.schemas.personas import PersonaCreate, PersonaResponse, PersonaUpdate

router = APIRouter(prefix="/api/personas", tags=["personas"])


@router.get("", response_model=list[PersonaResponse])
def list_personas(db: Annotated[Session, Depends(get_db)]) -> list[PersonaResponse]:
    personas = db.query(Persona).order_by(Persona.name).all()
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
    persona = db.query(Persona).filter(Persona.id == persona_id).first()
    if not persona:
        raise HTTPException(status_code=404, detail="Persona not found")

    if data.name is not None:
        persona.name = data.name
    if data.ppk_employee_rate is not None:
        persona.ppk_employee_rate = data.ppk_employee_rate
    if data.ppk_employer_rate is not None:
        persona.ppk_employer_rate = data.ppk_employer_rate

    try:
        db.commit()
        db.refresh(persona)
    except IntegrityError as e:
        db.rollback()
        raise HTTPException(status_code=409, detail=f"Persona '{data.name}' already exists") from e
    return PersonaResponse.model_validate(persona, from_attributes=True)


@router.delete("/{persona_id}", status_code=204)
def delete_persona(persona_id: int, db: Annotated[Session, Depends(get_db)]) -> None:
    persona = db.query(Persona).filter(Persona.id == persona_id).first()
    if not persona:
        raise HTTPException(status_code=404, detail="Persona not found")

    account_count = db.query(Account).filter(Account.owner == persona.name).count()
    if account_count > 0:
        raise HTTPException(
            status_code=409,
            detail=f"Cannot delete persona '{persona.name}': {account_count} accounts reference it",
        )

    db.delete(persona)
    db.commit()
