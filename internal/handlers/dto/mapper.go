package dto

import (
	"foodDelivery/internal/core/domain"

	"github.com/google/uuid"
)

func CompanyDtoFromDomain(company domain.Company) CompanyDto {
	return CompanyDto{
		Id:       company.Id.String(),
		Name:     company.Name,
		IsActive: company.IsActive,
	}
}

func CompaniesDtoFromDomains(companies []domain.Company) GetCompaniesResponse {
	companiesDto := make([]CompanyDto, len(companies))
	for i := range companies {
		companiesDto[i] = CompanyDtoFromDomain(companies[i])
	}
	return GetCompaniesResponse{Companies: companiesDto}
}

func ProductDtoFromDomain(product domain.Product) ProductDto {
	return ProductDto{
		Id:          product.Id.String(),
		Name:        product.Name,
		Price:       product.Price,
		Description: product.Description,
		IsAvailable: product.IsAvailable,
	}
}

func ProductsDtoFromDomains(products []domain.Product) GetCompanyProductsResponse {
	productsDto := make([]ProductDto, len(products))
	for i := range products {
		productsDto[i] = ProductDtoFromDomain(products[i])
	}
	return GetCompanyProductsResponse{Products: productsDto}
}
func BasketProductDtoFromDomain(product domain.BasketProduct) BasketProductDto {
	return BasketProductDto{
		Id:                 product.Id.String(),
		Name:               product.Name,
		Price:              product.Price,
		LastConfirmedPrice: product.ConfirmedPrice,
		ChosenCount:        product.ChosenCount,
		CurrentCount:       product.CurrentCount,
		Description:        product.Description,
		IsAvailable:        product.IsAvailable,
	}
}

func BasketProductsDtoFromDomains(products []domain.BasketProduct) []BasketProductDto {
	productsDto := make([]BasketProductDto, len(products))
	for i := range products {
		productsDto[i] = BasketProductDtoFromDomain(products[i])
	}
	return productsDto
}

func BasketDtoFromDomain(basket domain.Basket) UserBasketDto {
	return UserBasketDto{
		Products:           BasketProductsDtoFromDomains(basket.Products),
		FullPrice:          basket.FullPrice,
		FullConfirmedPrice: basket.FullConfirmedPrice,
	}
}

func ProductDomainFromDto(dto ProductDto) domain.Product {
	return domain.Product{
		Id:    uuid.MustParse(dto.Id),
		Price: dto.Price,
	}
}

func ProductsDomainFromDtos(dtos []ProductDto) []domain.Product {
	products := make([]domain.Product, len(dtos))
	for i := range dtos {
		products[i] = ProductDomainFromDto(dtos[i])
	}

	return products
}

func CompanyProductDtoFromDomain(product domain.Product) CompanyProductDto {
	return CompanyProductDto{
		Id:               product.Id.String(),
		PartnerProductId: product.PartnerProductId,
		Name:             product.Name,
		Price:            product.Price,
		Description:      product.Description,
		CurrentCount:     product.CurrentCount,
		IsAvailable:      product.IsAvailable,
	}
}

func GetMyOrdersResponseFromDomains(orders []domain.Order) GetMyOrdersResponse {
	response := GetMyOrdersResponse{
		Orders: make([]OrderDto, 0, len(orders)),
	}

	for _, order := range orders {
		response.Orders = append(response.Orders, OrderDtoFromDomain(order))
	}

	return response
}

func OrderDtoFromDomain(order domain.Order) OrderDto {
	orderDto := OrderDto{
		Id:              order.Id.String(),
		CompanyId:       order.CompanyId.String(),
		Status:          string(order.Status),
		RejectionReason: order.RejectionReason,
		Products:        make([]OrderProductDto, 0, len(order.Products)),
		FullPrice:       order.FullPrice,
		CreatedAt:       order.CreatedAt,
		UpdatedAt:       order.UpdatedAt,
	}

	for _, product := range order.Products {
		orderDto.Products = append(orderDto.Products, OrderProductDto{
			Id:    product.Id.String(),
			Name:  product.Name,
			Price: product.Price,
			Count: product.Count,
		})
	}

	return orderDto
}
