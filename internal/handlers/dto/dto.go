package dto

import "time"

type PutBasketProductRequest struct {
	Count int `json:"count" validate:"required,gt=0" example:"2"`
}

type ConfirmProductsPriceRequest struct {
	Products []ProductPriceRequest `json:"products" validate:"required,min=1,dive"`
}

type ProductPriceRequest struct {
	Id    string `json:"id" validate:"required,uuid" example:"33333333-3333-4333-8333-000000000001"`
	Price int64  `json:"price" validate:"required,gt=0" example:"49900"`
}

type GetCompaniesResponse struct {
	Companies []CompanyDto `json:"companies"`
}

type CompanyDto struct {
	Id       string `json:"id" example:"11111111-1111-4111-8111-111111111111"`
	Name     string `json:"name" example:"Demo company"`
	IsActive bool   `json:"is_active" example:"true"`
}

type GetCompanyProductsResponse struct {
	Products []ProductDto `json:"products"`
}

type ProductDto struct {
	Id          string `json:"id" example:"33333333-3333-4333-8333-000000000001"`
	Name        string `json:"name" example:"Пицца Маргарита"`
	Price       int64  `json:"price" example:"49900"`
	Description string `json:"description" example:"Томатный соус, моцарелла, томаты и базилик"`
	IsAvailable bool   `json:"is_available" example:"true"`
}

type UserBasketDto struct {
	Products           []BasketProductDto `json:"products"`
	FullPrice          int64              `json:"full_price" example:"99800"`
	FullConfirmedPrice int64              `json:"full_confirmed_price" example:"99800"`
}

type BasketProductDto struct {
	Id                 string `json:"id" example:"33333333-3333-4333-8333-000000000001"`
	Name               string `json:"name" example:"Пицца Маргарита"`
	Price              int64  `json:"price" example:"49900"`
	LastConfirmedPrice int64  `json:"last_confirmed_price" example:"49900"`
	ChosenCount        int    `json:"chosen_count" example:"2"`
	CurrentCount       int    `json:"current_count" example:"18"`
	Description        string `json:"description" example:"Томатный соус, моцарелла, томаты и базилик"`
	IsAvailable        bool   `json:"is_available" example:"true"`
}

type CreateOrderRequest struct {
	Products []ProductPriceRequest `json:"products" validate:"required,min=1,dive"`
}

type CreateCompanyProductRequest struct {
	PartnerProductId string `json:"partner_product_id" validate:"required,max=255" example:"pizza-margherita"`
	Name             string `json:"name" validate:"required,max=255" example:"Пицца Маргарита"`
	Price            int64  `json:"price" validate:"required,gt=0" example:"49900"`
	Description      string `json:"description" validate:"max=255" example:"Томатный соус, моцарелла, томаты и базилик"`
	CurrentCount     int    `json:"current_count" validate:"gte=0" example:"18"`
}

type PatchCompanyProductRequest struct {
	Name         *string `json:"name" validate:"omitempty,max=255" example:"Пицца Маргарита"`
	Price        *int64  `json:"price" validate:"omitempty,gt=0" example:"49900"`
	Description  *string `json:"description" validate:"omitempty,max=255" example:"Томатный соус, моцарелла, томаты и базилик"`
	CurrentCount *int    `json:"current_count" validate:"omitempty,gte=0" example:"18"`
	IsAvailable  *bool   `json:"is_available" example:"true"`
}

type ChangeOrderStatusRequest struct {
	Status string `json:"status" validate:"required,oneof=created rejected accepted preparing completed" enums:"created,rejected,accepted,preparing,completed" example:"accepted"`
}

type CompanyProductDto struct {
	Id               string  `json:"id" example:"33333333-3333-4333-8333-000000000001"`
	PartnerProductId *string `json:"partner_product_id" example:"pizza-margherita"`
	Name             string  `json:"name" example:"Пицца Маргарита"`
	Price            int64   `json:"price" example:"49900"`
	Description      string  `json:"description" example:"Томатный соус, моцарелла, томаты и базилик"`
	CurrentCount     int     `json:"current_count" example:"18"`
	IsAvailable      bool    `json:"is_available" example:"true"`
}

type GetMyOrdersResponse struct {
	Orders []OrderDto `json:"orders"`
}

type OrderDto struct {
	Id              string            `json:"id" example:"44444444-4444-4444-8444-444444444444"`
	CompanyId       string            `json:"company_id" example:"11111111-1111-4111-8111-111111111111"`
	Status          string            `json:"status" enums:"created,rejected,accepted,preparing,completed" example:"accepted"`
	RejectionReason *string           `json:"rejection_reason,omitempty" example:"Недостаточно товара"`
	Products        []OrderProductDto `json:"products"`
	FullPrice       int64             `json:"full_price" example:"99800"`
	CreatedAt       time.Time         `json:"created_at" example:"2026-09-08T12:00:00Z"`
	UpdatedAt       time.Time         `json:"updated_at" example:"2026-09-08T12:05:00Z"`
}

type OrderProductDto struct {
	Id    string `json:"id" example:"33333333-3333-4333-8333-000000000001"`
	Name  string `json:"name" example:"Пицца Маргарита"`
	Price int64  `json:"price" example:"49900"`
	Count int    `json:"count" example:"2"`
}
