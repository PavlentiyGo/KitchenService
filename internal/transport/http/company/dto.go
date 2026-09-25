package transport_dto

type OrderDto struct {
	OrderId  string       `json:"order_id"`
	Products []ProductDto `json:"products"`
}
type ProductDto struct {
	ProductId string `json:"product_id"`
	Count     int    `json:"count"`
}
