package domain

type Basket struct {
	Products           []BasketProduct
	FullPrice          int64
	FullConfirmedPrice int64
}
