package models

import (
	"time"

	"github.com/google/uuid"
)

type CompanyModel struct {
	Id       uuid.UUID
	Name     string
	IsActive bool
	Url      string
}

type ProductModel struct {
	Id               uuid.UUID
	CompanyId        uuid.UUID
	PartnerProductId *string
	Name             string
	Price            int64
	Description      string
	CurrentCount     int
	IsAvailable      bool
}
type BasketProductModel struct {
	Id             uuid.UUID
	Name           string
	Description    string
	CurrentPrice   int64
	ConfirmedPrice int64
	IsAvailable    bool
	ChosenCount    int
	CurrentCount   int
}

type ProductForSendModel struct {
	OrderId          uuid.UUID
	CompanyURL       string
	PartnerProductId string
	Count            int
}

type OrderModel struct {
	Id              uuid.UUID
	UserId          uuid.UUID
	CompanyId       uuid.UUID
	Status          string
	RejectionReason *string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type OrderProductModel struct {
	Id    uuid.UUID
	Name  string
	Price int64
	Count int
}
