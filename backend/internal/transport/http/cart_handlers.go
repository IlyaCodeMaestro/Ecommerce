package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"ecommerce-backend/internal/domain"
	"ecommerce-backend/internal/service"

	"github.com/go-chi/chi/v5"
)

func resolveCartKey(r *http.Request) string {
	if claims := GetUserClaims(r.Context()); claims != nil && claims.UserID != "" {
		return "usr:" + claims.UserID
	}

	sessionID := strings.TrimSpace(r.Header.Get("X-Session-ID"))
	if sessionID != "" {
		return "guest:" + sessionID
	}

	return "guest:anonymous"
}

func (h *Handler) GetCart(w http.ResponseWriter, r *http.Request) {
	cartKey := resolveCartKey(r)
	cart, err := h.cartService.GetCart(r.Context(), cartKey)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to retrieve cart")
		return
	}

	respondJSON(w, http.StatusOK, cart)
}

func (h *Handler) AddCartItem(w http.ResponseWriter, r *http.Request) {
	var req domain.AddToCartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	cartKey := resolveCartKey(r)
	cart, err := h.cartService.AddItem(r.Context(), cartKey, req)
	if err != nil {
		if errors.Is(err, service.ErrInvalidQuantity) {
			respondError(w, http.StatusBadRequest, err.Error())
			return
		}
		if errors.Is(err, service.ErrProductNotFound) {
			respondError(w, http.StatusNotFound, "product not found")
			return
		}
		if errors.Is(err, service.ErrInsufficientStock) {
			respondError(w, http.StatusConflict, "insufficient product stock")
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, cart)
}

func (h *Handler) UpdateCartItem(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	productID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid product id")
		return
	}

	var req domain.UpdateCartItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	cartKey := resolveCartKey(r)
	cart, err := h.cartService.UpdateItem(r.Context(), cartKey, productID, req.Quantity)
	if err != nil {
		if errors.Is(err, service.ErrProductNotFound) {
			respondError(w, http.StatusNotFound, "product not found")
			return
		}
		if errors.Is(err, service.ErrInsufficientStock) {
			respondError(w, http.StatusConflict, "insufficient product stock")
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, cart)
}

func (h *Handler) RemoveCartItem(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	productID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid product id")
		return
	}

	cartKey := resolveCartKey(r)
	cart, err := h.cartService.RemoveItem(r.Context(), cartKey, productID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to remove cart item")
		return
	}

	respondJSON(w, http.StatusOK, cart)
}

func (h *Handler) ClearCart(w http.ResponseWriter, r *http.Request) {
	cartKey := resolveCartKey(r)
	if err := h.cartService.ClearCart(r.Context(), cartKey); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to clear cart")
		return
	}

	respondJSON(w, http.StatusOK, &domain.Cart{
		CartKey: cartKey,
		Items:   []domain.CartItem{},
	})
}

func (h *Handler) MergeCart(w http.ResponseWriter, r *http.Request) {
	claims := GetUserClaims(r.Context())
	if claims == nil || claims.UserID == "" {
		respondError(w, http.StatusUnauthorized, "authentication required to merge cart")
		return
	}

	var req domain.MergeCartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	userCartKey := "usr:" + claims.UserID
	cart, err := h.cartService.MergeCart(r.Context(), userCartKey, req.Items)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to merge cart")
		return
	}

	respondJSON(w, http.StatusOK, cart)
}

