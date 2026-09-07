package service

import (
	"context"
	"testing"

	"ecommerce-backend/internal/domain"
)

func TestCart_Recalculate(t *testing.T) {
	cart := &domain.Cart{
		CartKey: "test-cart-1",
		Items: []domain.CartItem{
			{
				ProductID: 1,
				Name:      "Item 1",
				Price:     100.0,
				Quantity:  2,
			},
			{
				ProductID: 2,
				Name:      "Item 2",
				Price:     49.5,
				Quantity:  3,
			},
		},
	}

	cart.Recalculate()

	if cart.TotalItems != 5 {
		t.Errorf("expected total items 5, got %d", cart.TotalItems)
	}

	expectedAmount := (100.0 * 2) + (49.5 * 3) // 200 + 148.5 = 348.5
	if cart.TotalAmount != expectedAmount {
		t.Errorf("expected total amount %.2f, got %.2f", expectedAmount, cart.TotalAmount)
	}

	if cart.Items[0].Subtotal != 200.0 {
		t.Errorf("expected item 0 subtotal 200, got %.2f", cart.Items[0].Subtotal)
	}
	if cart.Items[1].Subtotal != 148.5 {
		t.Errorf("expected item 1 subtotal 148.5, got %.2f", cart.Items[1].Subtotal)
	}
}

func TestCartService_QuantityValidation(t *testing.T) {
	svc := NewCartService(nil, nil)

	_, err := svc.AddItem(context.Background(), "cart-test", domain.AddToCartRequest{
		ProductID: 1,
		Quantity:  0,
	})
	if err != ErrInvalidQuantity {
		t.Errorf("expected ErrInvalidQuantity for 0 qty, got %v", err)
	}

	_, err = svc.AddItem(context.Background(), "cart-test", domain.AddToCartRequest{
		ProductID: 1,
		Quantity:  -5,
	})
	if err != ErrInvalidQuantity {
		t.Errorf("expected ErrInvalidQuantity for negative qty, got %v", err)
	}
}

