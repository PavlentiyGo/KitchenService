package models

import (
	"foodDelivery/internal/core/domain"
)

func CompanyDomainFromModel(company CompanyModel) domain.Company {
	return domain.Company{
		Id:       company.Id,
		Name:     company.Name,
		IsActive: company.IsActive,
		Url:      company.Url,
	}
}

func ProductDomainFromModel(product ProductModel) domain.Product {
	return domain.Product{
		Id:               product.Id,
		CompanyId:        product.CompanyId,
		PartnerProductId: product.PartnerProductId,
		Name:             product.Name,
		Price:            product.Price,
		Description:      product.Description,
		CurrentCount:     product.CurrentCount,
		IsAvailable:      product.IsAvailable,
	}
}

func BasketProductDomainFromModel(product BasketProductModel) domain.BasketProduct {
	return domain.BasketProduct{
		Product: domain.Product{
			Id:           product.Id,
			Name:         product.Name,
			Price:        product.CurrentPrice,
			Description:  product.Description,
			IsAvailable:  product.IsAvailable,
			CurrentCount: product.CurrentCount,
		},
		ConfirmedPrice: product.ConfirmedPrice,
		ChosenCount:    product.ChosenCount,
	}
}

func ProductForSendDomainFromModel(product ProductForSendModel) domain.ProductForSend {
	return domain.ProductForSend{
		PartnerProductId: product.PartnerProductId,
		Count:            product.Count,
	}
}

func OrderForSendDomainFromModel(product ProductForSendModel) domain.OrderForSend {
	return domain.OrderForSend{
		OrderId:    product.OrderId,
		CompanyUrl: product.CompanyURL,
		Products: []domain.ProductForSend{
			ProductForSendDomainFromModel(product),
		},
	}
}

func OrderProductDomainFromModel(product OrderProductModel) domain.OrderProduct {
	return domain.OrderProduct{
		Id:    product.Id,
		Name:  product.Name,
		Price: product.Price,
		Count: product.Count,
	}
}

func OrderRowDomainFromModels(order OrderModel, product OrderProductModel) domain.OrderRow {
	return domain.OrderRow{
		OrderId:         order.Id,
		UserId:          order.UserId,
		CompanyId:       order.CompanyId,
		Status:          domain.OrderStatus(order.Status),
		RejectionReason: order.RejectionReason,
		Product:         OrderProductDomainFromModel(product),
		CreatedAt:       order.CreatedAt,
		UpdatedAt:       order.UpdatedAt,
	}
}
