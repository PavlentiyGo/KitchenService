package domain

import (
	"time"

	"github.com/google/uuid"
)

type OrderStatus string

const (
	OrderStatusCreated   OrderStatus = "created"
	OrderStatusRejected  OrderStatus = "rejected"
	OrderStatusAccepted  OrderStatus = "accepted"
	OrderStatusPreparing OrderStatus = "preparing"
	OrderStatusCompleted OrderStatus = "completed"
)

type Order struct {
	Id              uuid.UUID
	UserId          uuid.UUID
	CompanyId       uuid.UUID
	Status          OrderStatus
	RejectionReason *string
	Products        []OrderProduct
	FullPrice       int64
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type OrderProduct struct {
	Id    uuid.UUID
	Name  string
	Price int64
	Count int
}

type OrderRow struct {
	OrderId         uuid.UUID
	UserId          uuid.UUID
	CompanyId       uuid.UUID
	Status          OrderStatus
	RejectionReason *string
	Product         OrderProduct
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type OrderForSend struct {
	OrderId    uuid.UUID
	CompanyUrl string
	Products   []ProductForSend
}

type ProductForSend struct {
	PartnerProductId string
	Count            int
}
