package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/platform/backend/internal/repository"
	"github.com/platform/backend/internal/service"
)

type OrderHandler struct {
	orderService *service.OrderService
	queries      *repository.Queries
}

func NewOrderHandler(orderService *service.OrderService, queries *repository.Queries) *OrderHandler {
	return &OrderHandler{
		orderService: orderService,
		queries:      queries,
	}
}

type CreateOrderRequest struct {
	CustomerID  string          `json:"customer_id"`
	AmountCents int64           `json:"amount_cents"`
	Currency    string          `json:"currency"`
	Metadata    json.RawMessage `json:"metadata"`
}

type UpdateOrderStatusRequest struct {
	Status string `json:"status"`
}

// CreateOrder maneja la creación de la orden e inserta atómicamente el evento outbox.
func (h *OrderHandler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	var req CreateOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Payload JSON inválido: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Sanitización de entradas del usuario
	customerID := strings.TrimSpace(req.CustomerID)
	currency := strings.ToUpper(strings.TrimSpace(req.Currency))
	if currency == "" {
		currency = "USD"
	}

	order, err := h.orderService.CreateOrderWithOutbox(r.Context(), service.CreateOrderInput{
		CustomerID:  customerID,
		AmountCents: req.AmountCents,
		Currency:    currency,
		Metadata:    req.Metadata,
	})
	if err != nil {
		http.Error(w, "Error procesando orden: "+err.Error(), http.StatusUnprocessableEntity)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(order)
}

// UpdateStatus permite a los agentes de LangGraph actualizar el estado de la entidad (Zero Trust: vía API Go).
func (h *OrderHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch && r.Method != http.MethodPut {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	// Extracción de ID desde el path /api/v1/orders/{id}/status
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 4 {
		http.Error(w, "Ruta inválida", http.StatusBadRequest)
		return
	}

	orderID, err := uuid.Parse(parts[3])
	if err != nil {
		http.Error(w, "UUID inválido", http.StatusBadRequest)
		return
	}

	var req UpdateOrderStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Payload JSON inválido", http.StatusBadRequest)
		return
	}

	updated, err := h.queries.UpdateOrderStatus(r.Context(), repository.UpdateOrderStatusParams{
		ID:     orderID,
		Status: req.Status,
	})
	if err != nil {
		http.Error(w, "Error actualizando estado de la orden: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(updated)
}

// GetByID consulta la entidad de negocio requerida por los agentes.
func (h *OrderHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 4 {
		http.Error(w, "Ruta inválida", http.StatusBadRequest)
		return
	}

	orderID, err := uuid.Parse(parts[3])
	if err != nil {
		http.Error(w, "UUID inválido", http.StatusBadRequest)
		return
	}

	order, err := h.queries.GetOrderByID(r.Context(), orderID)
	if err != nil {
		http.Error(w, "Orden no encontrada", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(order)
}
