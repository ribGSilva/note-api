package notes

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/ribgsilva/note-api/business/v1/note"
	"github.com/ribgsilva/note-api/sys"
	"gocloud.dev/pubsub"
)

func Consume(ctx context.Context, sub *pubsub.Subscription, maxWorkers int) error {
	logger := sys.R.Log
	workers := make(chan int, maxWorkers)

	var err error
	for {
		var message, err = sub.Receive(ctx)
		if err != nil {
			break
		}

		go func(m *pubsub.Message) {
			workers <- 1
			defer func() { <-workers }()
			defer m.Ack()

			logger.Infof("message received: %s", string(m.Body))
			var e note.Event
			if err := json.Unmarshal(m.Body, &e); err != nil {
				logger.Error("failed to parse body: ", err)
				return
			}

			switch e.Type {
			case "create":
				var c note.NewNote
				marshal, _ := json.Marshal(e.Data)
				_ = json.Unmarshal(marshal, &c)

				if err := note.Create(ctx, c); err != nil {
					logger.Errorf("failed to create event %+v: err: %s", e.Data, err)
				}
			default:
				logger.Error("unknown event type: ", e.Type)
			}
		}(message)
	}

	for w := 0; w < maxWorkers; w++ {
		workers <- 1
	}

	if !errors.Is(err, context.Canceled) {
		return err
	}

	return nil
}
