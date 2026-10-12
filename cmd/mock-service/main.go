package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"foodDelivery/internal/handlers/dto"
	"log"
	"math/rand"
	"net/http"
)

type orderRequest struct {
	OrderID  string `json:"order_id"`
	Products []prod `json:"products"`
}

type prod struct {
	ID    string `json:"product_id"`
	Count int    `json:"count"`
}

type handler struct {
	cfg *MockConfig
}

const (
	demoCompanyID  = "11111111-1111-4111-8111-111111111111"
	mainAPIBaseURL = "http://app:8080/api/v1"
	mockServerAddr = ":8081"
)

func main() {
	cfg := NewConfigMust()
	handlers := handler{cfg: cfg}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /orders", handlers.handleOrder)

	log.Printf("mock company server is listening on %s", mockServerAddr)
	server := &http.Server{
		Addr:              mockServerAddr,
		Handler:           mux,
		ReadHeaderTimeout: cfg.RequestTimeout,
	}
	err := server.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}

func (h *handler) handleOrder(w http.ResponseWriter, r *http.Request) {

	var order orderRequest
	err := json.NewDecoder(r.Body).Decode(&order)
	if err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	random := rand.Intn(100)
	if random < h.cfg.FailurePercent {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = h.changeOrderStatus(r.Context(), dto.ChangeOrderStatusRequest{Status: "accepted"}, order.OrderID)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}
func (h *handler) changeOrderStatus(ctx context.Context, status dto.ChangeOrderStatusRequest, orderId string) error {
	body, err := json.Marshal(status)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, h.cfg.RequestTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPatch,
		fmt.Sprintf("%s/company/orders/%s/status", mainAPIBaseURL, orderId),
		bytes.NewBuffer(body),
	)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", demoCompanyID)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		err = resp.Body.Close()
		if err != nil {
			log.Printf("failed to close response body: %v", err)
		}
	}()
	return nil
}
