package providers

import (
	"context"

	"github.com/dhanifudin/pakai/internal/schema"
)

// Provider is the interface that all usage providers must implement.
type Provider interface {
	// ID returns the unique identifier for this provider.
	ID() string
	// Fetch retrieves the current usage data from the provider.
	Fetch(ctx context.Context) (*schema.Usage, error)
}
