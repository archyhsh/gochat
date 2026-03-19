package main

import (
	"encoding/json"
	"flag"
	"log"
	"time"

	"github.com/archyhsh/gochat/pkg/kafka"
	"github.com/archyhsh/gochat/pkg/messaging"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

var (
	redisHost    = flag.String("redis", "127.0.0.1:6379", "Redis host")
	kafkaAddr    = flag.String("kafka", "127.0.0.1:9092", "Kafka broker address")
	dlqKey       = flag.String("key", messaging.DefaultDLQKey, "DLQ redis key")
	batchSize    = flag.Int("batch", 10, "Batch size to process")
	requeueLimit = flag.Int("limit", 3, "Max retry count before permanent failure")
)

func main() {
	flag.Parse()

	rdb := redis.MustNewRedis(redis.RedisConf{
		Host: *redisHost,
		Type: "node",
	})

	producer, err := kafka.NewProducer([]string{*kafkaAddr}, "recovery-topic")
	if err != nil {
		log.Fatalf("Failed to init kafka producer: %v", err)
	}
	defer producer.Close()

	log.Printf("Starting DLQ recovery. Redis: %s, Kafka: %s, Key: %s", *redisHost, *kafkaAddr, *dlqKey)

	for {
		count := 0
		for i := 0; i < *batchSize; i++ {
			data, err := rdb.Rpop(*dlqKey)
			if err != nil || data == "" {
				break
			}

			var msg messaging.FailedMessage
			if err := json.Unmarshal([]byte(data), &msg); err != nil {
				log.Printf("Failed to unmarshal message: %v", err)
				continue
			}

			log.Printf("Recovering message: topic=%s, key=%s", msg.Topic, msg.Key)
			err = producer.SendToTopic(msg.Topic, []byte(msg.Key), []byte(msg.Value))
			if err != nil {
				log.Printf("Failed to re-send message: %v", err)
				msg.RetryCount++
				msg.Error = err.Error()

				if msg.RetryCount >= *requeueLimit {
					permKey := *dlqKey + ":permanent"
					val, _ := json.Marshal(msg)
					_, _ = rdb.Lpush(permKey, string(val))
					log.Printf("Message reached retry limit. Moved to permanent failure queue.")
				} else {
					// Requeue for next round
					val, _ := json.Marshal(msg)
					_, _ = rdb.Lpush(*dlqKey, string(val))
				}
			} else {
				log.Printf("Successfully recovered message")
				count++
			}
		}

		if count == 0 {
			log.Printf("No more messages to recover. Sleeping 10s...")
			time.Sleep(10 * time.Second)
		} else {
			log.Printf("Processed %d messages in this batch", count)
		}
	}
}
