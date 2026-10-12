package transport_dto

import "foodDelivery/internal/core/domain"

func OrderDtoFromDomain(order domain.OrderForSend) OrderDto {
	products := make([]ProductDto, len(order.Products))
	for i := range order.Products {
		products[i] = ProductDto{
			ProductId: order.Products[i].PartnerProductId,
			Count:     order.Products[i].Count,
		}
	}

	return OrderDto{
		OrderId:  order.OrderId.String(),
		Products: products,
	}
}
