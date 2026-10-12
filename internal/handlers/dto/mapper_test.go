package dto_test

import (
	"foodDelivery/internal/core/domain"
	"foodDelivery/internal/handlers/dto"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestCompanyDtoFromDomain(t *testing.T) {
	company := domain.Company{
		Id:       uuid.MustParse("11111111-1111-4111-8111-111111111111"),
		Name:     "Demo company",
		IsActive: true,
		Url:      "http://company.example",
	}

	result := dto.CompanyDtoFromDomain(company)

	expected := dto.CompanyDto{
		Id:       "11111111-1111-4111-8111-111111111111",
		Name:     "Demo company",
		IsActive: true,
	}
	assert.Equal(t, expected, result)
}

func TestCompaniesDtoFromDomains(t *testing.T) {
	companies := []domain.Company{
		{
			Id:       uuid.MustParse("11111111-1111-4111-8111-111111111111"),
			Name:     "First company",
			IsActive: true,
		},
		{
			Id:       uuid.MustParse("11111111-1111-4111-8111-222222222222"),
			Name:     "Second company",
			IsActive: false,
		},
	}

	result := dto.CompaniesDtoFromDomains(companies)

	expected := dto.GetCompaniesResponse{
		Companies: []dto.CompanyDto{
			{
				Id:       "11111111-1111-4111-8111-111111111111",
				Name:     "First company",
				IsActive: true,
			},
			{
				Id:       "11111111-1111-4111-8111-222222222222",
				Name:     "Second company",
				IsActive: false,
			},
		},
	}
	assert.Equal(t, expected, result)
}

func TestProductDtoFromDomain(t *testing.T) {
	product := domain.Product{
		Id:           uuid.MustParse("33333333-3333-4333-8333-000000000001"),
		CompanyId:    uuid.MustParse("11111111-1111-4111-8111-111111111111"),
		Name:         "Пицца Маргарита",
		Price:        49900,
		Description:  "Томатный соус, моцарелла, томаты и базилик",
		CurrentCount: 18,
		IsAvailable:  true,
	}

	result := dto.ProductDtoFromDomain(product)

	expected := dto.ProductDto{
		Id:          "33333333-3333-4333-8333-000000000001",
		Name:        "Пицца Маргарита",
		Price:       49900,
		Description: "Томатный соус, моцарелла, томаты и базилик",
		IsAvailable: true,
	}
	assert.Equal(t, expected, result)
}

func TestProductsDtoFromDomains(t *testing.T) {
	products := []domain.Product{
		{
			Id:          uuid.MustParse("33333333-3333-4333-8333-000000000001"),
			Name:        "Пицца Маргарита",
			Price:       49900,
			Description: "Томатный соус и моцарелла",
			IsAvailable: true,
		},
		{
			Id:          uuid.MustParse("33333333-3333-4333-8333-000000000002"),
			Name:        "Пицца Пепперони",
			Price:       57900,
			Description: "Пепперони и моцарелла",
			IsAvailable: false,
		},
	}

	result := dto.ProductsDtoFromDomains(products)

	expected := dto.GetCompanyProductsResponse{
		Products: []dto.ProductDto{
			{
				Id:          "33333333-3333-4333-8333-000000000001",
				Name:        "Пицца Маргарита",
				Price:       49900,
				Description: "Томатный соус и моцарелла",
				IsAvailable: true,
			},
			{
				Id:          "33333333-3333-4333-8333-000000000002",
				Name:        "Пицца Пепперони",
				Price:       57900,
				Description: "Пепперони и моцарелла",
				IsAvailable: false,
			},
		},
	}
	assert.Equal(t, expected, result)
}

func TestBasketProductDtoFromDomain(t *testing.T) {
	product := domain.BasketProduct{
		Product: domain.Product{
			Id:           uuid.MustParse("33333333-3333-4333-8333-000000000001"),
			Name:         "Пицца Маргарита",
			Price:        49900,
			Description:  "Томатный соус и моцарелла",
			CurrentCount: 18,
			IsAvailable:  true,
		},
		ConfirmedPrice: 45900,
		ChosenCount:    2,
	}

	result := dto.BasketProductDtoFromDomain(product)

	expected := dto.BasketProductDto{
		Id:                 "33333333-3333-4333-8333-000000000001",
		Name:               "Пицца Маргарита",
		Price:              49900,
		LastConfirmedPrice: 45900,
		ChosenCount:        2,
		CurrentCount:       18,
		Description:        "Томатный соус и моцарелла",
		IsAvailable:        true,
	}
	assert.Equal(t, expected, result)
}

func TestBasketProductsDtoFromDomains(t *testing.T) {
	products := []domain.BasketProduct{
		{
			Product: domain.Product{
				Id:           uuid.MustParse("33333333-3333-4333-8333-000000000001"),
				Name:         "Пицца Маргарита",
				Price:        49900,
				Description:  "Томатный соус и моцарелла",
				CurrentCount: 18,
				IsAvailable:  true,
			},
			ConfirmedPrice: 45900,
			ChosenCount:    2,
		},
		{
			Product: domain.Product{
				Id:           uuid.MustParse("33333333-3333-4333-8333-000000000002"),
				Name:         "Пицца Пепперони",
				Price:        57900,
				Description:  "Пепперони и моцарелла",
				CurrentCount: 0,
				IsAvailable:  false,
			},
			ConfirmedPrice: 57900,
			ChosenCount:    1,
		},
	}

	result := dto.BasketProductsDtoFromDomains(products)

	expected := []dto.BasketProductDto{
		{
			Id:                 "33333333-3333-4333-8333-000000000001",
			Name:               "Пицца Маргарита",
			Price:              49900,
			LastConfirmedPrice: 45900,
			ChosenCount:        2,
			CurrentCount:       18,
			Description:        "Томатный соус и моцарелла",
			IsAvailable:        true,
		},
		{
			Id:                 "33333333-3333-4333-8333-000000000002",
			Name:               "Пицца Пепперони",
			Price:              57900,
			LastConfirmedPrice: 57900,
			ChosenCount:        1,
			CurrentCount:       0,
			Description:        "Пепперони и моцарелла",
			IsAvailable:        false,
		},
	}
	assert.Equal(t, expected, result)
}

func TestBasketDtoFromDomain(t *testing.T) {
	basket := domain.Basket{
		Products: []domain.BasketProduct{
			{
				Product: domain.Product{
					Id:           uuid.MustParse("33333333-3333-4333-8333-000000000001"),
					Name:         "Пицца Маргарита",
					Price:        49900,
					Description:  "Томатный соус и моцарелла",
					CurrentCount: 18,
					IsAvailable:  true,
				},
				ConfirmedPrice: 45900,
				ChosenCount:    2,
			},
		},
		FullPrice:          99800,
		FullConfirmedPrice: 91800,
	}

	result := dto.BasketDtoFromDomain(basket)

	expected := dto.UserBasketDto{
		Products: []dto.BasketProductDto{
			{
				Id:                 "33333333-3333-4333-8333-000000000001",
				Name:               "Пицца Маргарита",
				Price:              49900,
				LastConfirmedPrice: 45900,
				ChosenCount:        2,
				CurrentCount:       18,
				Description:        "Томатный соус и моцарелла",
				IsAvailable:        true,
			},
		},
		FullPrice:          99800,
		FullConfirmedPrice: 91800,
	}
	assert.Equal(t, expected, result)
}

func TestProductDomainFromDto(t *testing.T) {
	productDto := dto.ProductDto{
		Id:          "33333333-3333-4333-8333-000000000001",
		Name:        "Пицца Маргарита",
		Price:       49900,
		Description: "Томатный соус и моцарелла",
		IsAvailable: true,
	}

	result := dto.ProductDomainFromDto(productDto)

	expected := domain.Product{
		Id:    uuid.MustParse("33333333-3333-4333-8333-000000000001"),
		Price: 49900,
	}
	assert.Equal(t, expected, result)
}

func TestProductsDomainFromDtos(t *testing.T) {
	productDtos := []dto.ProductDto{
		{
			Id:    "33333333-3333-4333-8333-000000000001",
			Price: 49900,
		},
		{
			Id:    "33333333-3333-4333-8333-000000000002",
			Price: 57900,
		},
	}

	result := dto.ProductsDomainFromDtos(productDtos)

	expected := []domain.Product{
		{
			Id:    uuid.MustParse("33333333-3333-4333-8333-000000000001"),
			Price: 49900,
		},
		{
			Id:    uuid.MustParse("33333333-3333-4333-8333-000000000002"),
			Price: 57900,
		},
	}
	assert.Equal(t, expected, result)
}

func TestCompanyProductDtoFromDomain(t *testing.T) {
	partnerProductId := "pizza-margherita"
	product := domain.Product{
		Id:               uuid.MustParse("33333333-3333-4333-8333-000000000001"),
		CompanyId:        uuid.MustParse("11111111-1111-4111-8111-111111111111"),
		PartnerProductId: &partnerProductId,
		Name:             "Пицца Маргарита",
		Price:            49900,
		Description:      "Томатный соус и моцарелла",
		CurrentCount:     18,
		IsAvailable:      true,
	}

	result := dto.CompanyProductDtoFromDomain(product)

	expected := dto.CompanyProductDto{
		Id:               "33333333-3333-4333-8333-000000000001",
		PartnerProductId: &partnerProductId,
		Name:             "Пицца Маргарита",
		Price:            49900,
		Description:      "Томатный соус и моцарелла",
		CurrentCount:     18,
		IsAvailable:      true,
	}
	assert.Equal(t, expected, result)
}

func TestOrderDtoFromDomain(t *testing.T) {
	rejectionReason := "Недостаточно товара"
	createdAt := time.Date(2026, time.September, 8, 12, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2026, time.September, 8, 12, 5, 0, 0, time.UTC)
	order := domain.Order{
		Id:              uuid.MustParse("44444444-4444-4444-8444-444444444444"),
		UserId:          uuid.MustParse("22222222-2222-4222-8222-222222222222"),
		CompanyId:       uuid.MustParse("11111111-1111-4111-8111-111111111111"),
		Status:          domain.OrderStatusRejected,
		RejectionReason: &rejectionReason,
		Products: []domain.OrderProduct{
			{
				Id:    uuid.MustParse("33333333-3333-4333-8333-000000000001"),
				Name:  "Пицца Маргарита",
				Price: 49900,
				Count: 2,
			},
		},
		FullPrice: 99800,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}

	result := dto.OrderDtoFromDomain(order)

	expected := dto.OrderDto{
		Id:              "44444444-4444-4444-8444-444444444444",
		CompanyId:       "11111111-1111-4111-8111-111111111111",
		Status:          "rejected",
		RejectionReason: &rejectionReason,
		Products: []dto.OrderProductDto{
			{
				Id:    "33333333-3333-4333-8333-000000000001",
				Name:  "Пицца Маргарита",
				Price: 49900,
				Count: 2,
			},
		},
		FullPrice: 99800,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}
	assert.Equal(t, expected, result)
}

func TestGetMyOrdersResponseFromDomains(t *testing.T) {
	createdAt := time.Date(2026, time.September, 8, 12, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2026, time.September, 8, 12, 5, 0, 0, time.UTC)
	orders := []domain.Order{
		{
			Id:        uuid.MustParse("44444444-4444-4444-8444-444444444444"),
			CompanyId: uuid.MustParse("11111111-1111-4111-8111-111111111111"),
			Status:    domain.OrderStatusCreated,
			Products: []domain.OrderProduct{
				{
					Id:    uuid.MustParse("33333333-3333-4333-8333-000000000001"),
					Name:  "Пицца Маргарита",
					Price: 49900,
					Count: 2,
				},
			},
			FullPrice: 99800,
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		},
		{
			Id:        uuid.MustParse("44444444-4444-4444-8444-555555555555"),
			CompanyId: uuid.MustParse("11111111-1111-4111-8111-222222222222"),
			Status:    domain.OrderStatusCompleted,
			Products:  []domain.OrderProduct{},
			FullPrice: 0,
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		},
	}

	result := dto.GetMyOrdersResponseFromDomains(orders)

	expected := dto.GetMyOrdersResponse{
		Orders: []dto.OrderDto{
			{
				Id:        "44444444-4444-4444-8444-444444444444",
				CompanyId: "11111111-1111-4111-8111-111111111111",
				Status:    "created",
				Products: []dto.OrderProductDto{
					{
						Id:    "33333333-3333-4333-8333-000000000001",
						Name:  "Пицца Маргарита",
						Price: 49900,
						Count: 2,
					},
				},
				FullPrice: 99800,
				CreatedAt: createdAt,
				UpdatedAt: updatedAt,
			},
			{
				Id:        "44444444-4444-4444-8444-555555555555",
				CompanyId: "11111111-1111-4111-8111-222222222222",
				Status:    "completed",
				Products:  []dto.OrderProductDto{},
				FullPrice: 0,
				CreatedAt: createdAt,
				UpdatedAt: updatedAt,
			},
		},
	}
	assert.Equal(t, expected, result)
}
