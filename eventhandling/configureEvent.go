package eventhandling

import (
	"context"

	cloudevents "github.com/cloudevents/sdk-go"
)

func GotEvent(ctx context.Context, event cloudevents.Event) error {
	return nil
}
