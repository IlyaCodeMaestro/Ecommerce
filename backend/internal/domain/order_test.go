package domain

import (
	"testing"
)

func TestOrderStatus_CanTransitionTo(t *testing.T) {
	tests := []struct {
		current  OrderStatus
		next     OrderStatus
		expected bool
	}{
		// Idempotent transitions
		{OrderStatusAccepted, OrderStatusAccepted, true},
		{OrderStatusPaid, OrderStatusPaid, true},
		{OrderStatusCancelled, OrderStatusCancelled, true},

		// Valid forward progressions
		{OrderStatusAccepted, OrderStatusPaid, true},
		{OrderStatusAccepted, OrderStatusCancelled, true},
		{OrderStatusAccepted, OrderStatusFailed, true},
		{OrderStatusPending, OrderStatusPaid, true},
		{OrderStatusPending, OrderStatusCancelled, true},
		{OrderStatusPaid, OrderStatusProcessing, true},
		{OrderStatusPaid, OrderStatusCancelled, true},
		{OrderStatusPaid, OrderStatusRefunded, true},
		{OrderStatusProcessing, OrderStatusShipped, true},
		{OrderStatusProcessing, OrderStatusCancelled, true},
		{OrderStatusShipped, OrderStatusDelivered, true},
		{OrderStatusShipped, OrderStatusCompleted, true},
		{OrderStatusShipped, OrderStatusRefunded, true},
		{OrderStatusDelivered, OrderStatusRefunded, true},

		// Invalid transitions
		{OrderStatusDelivered, OrderStatusPending, false},
		{OrderStatusDelivered, OrderStatusPaid, false},
		{OrderStatusCancelled, OrderStatusPaid, false},
		{OrderStatusCancelled, OrderStatusProcessing, false},
		{OrderStatusRefunded, OrderStatusDelivered, false},
		{OrderStatusFailed, OrderStatusPaid, false},
		{OrderStatusAccepted, OrderStatusDelivered, false},
		{OrderStatusPaid, OrderStatusDelivered, false},
	}

	for _, tt := range tests {
		got := tt.current.CanTransitionTo(tt.next)
		if got != tt.expected {
			t.Errorf("expected %s.CanTransitionTo(%s) = %v, got %v", tt.current, tt.next, tt.expected, got)
		}
	}
}

