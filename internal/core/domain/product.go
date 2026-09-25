package domain

import "github.com/google/uuid"

type Product struct {
	Id               uuid.UUID
	CompanyId        uuid.UUID
	PartnerProductId *string
	Name             string
	Price            int64
	Description      string
	CurrentCount     int
	IsAvailable      bool
}

type ProductPatch struct {
	Name         *string
	Price        *int64
	Description  *string
	CurrentCount *int
	IsAvailable  *bool
}

type BasketProduct struct {
	Product
	ConfirmedPrice int64
	ChosenCount    int
}
