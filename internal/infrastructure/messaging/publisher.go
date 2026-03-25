package messaging

import (
	natspub "github.com/crm-system-new/crm-shared/pkg/messaging/nats"
)

// NewIdentityPublisher creates a NATS publisher and ensures the IDENTITY stream exists.
func NewIdentityPublisher(natsURL string) (*natspub.Publisher, error) {
	pub, err := natspub.NewPublisher(natsURL)
	if err != nil {
		return nil, err
	}

	if err := pub.EnsureStream("IDENTITY", []string{"crm.identity.>"}); err != nil {
		pub.Close()
		return nil, err
	}

	return pub, nil
}
