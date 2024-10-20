package consumer

import (
	"encoding/json"
	"fmt"
	"sync"

	mongo "webhook-consumer/internal/mongo"

	kafka "github.com/segmentio/kafka-go"
)

func ProcessMessage(m kafka.Message, wg *sync.WaitGroup, sem chan struct{}) {
	defer wg.Done()
	defer func() { <-sem }()
	const TAG = "webhook-consumer"

	msg := KafkaMessage{}

	msg.ca = m.Time
	msg.offset = 0

	var err error = nil

	if err = json.Unmarshal(m.Value, &msg); err != nil {
		return
	}

	var payload Pyld

	switch v := msg.Pyld.(type) {
	case string:
		// Optionally, you can further unmarshal if it's JSON in a string form
		if err = json.Unmarshal([]byte(v), &payload); err != nil {
			return
		}
	case map[string]interface{}:
		// Convert the map into a structured object if needed
		// jsonBytes, _ := json.Marshal(v)
		// if err = json.Unmarshal(jsonBytes, &payload); err != nil {
		// 	return
		// }
		payload = v
	default:
		fmt.Println("Unknown pyld type")
		return
	}

	if msg.Meta["uniqueWebhook"] == 1 {
		isUnique, err := mongo.IsUniqueCallAttempt(msg.CidNum, msg.CallNum, msg.UID)

		if !isUnique {
			return
		}

		if err != nil {
			logObject := mongo.ErrorLog{
				Priority:  "error",
				Message:   "Error occurred while checking for unique call",
				UserID:    0,
				Tag:       TAG,
				Exception: err,
				Env: map[string]interface{}{
					"topic":     m.Topic,
					"logObject": msg,
				},
			}
			mongo.InsertErrorLog(&logObject)
		}

	}

	prepareRequest(&msg, payload)
}
