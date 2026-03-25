#!/bin/bash

# Wait for Kafka to be ready
echo "Waiting for Kafka to be ready..."
sleep 20

# Create topics
kafka-topics --bootstrap-server kafka:9092 --create --if-not-exists --topic message-topic --partitions 3 --replication-factor 1
kafka-topics --bootstrap-server kafka:9092 --create --if-not-exists --topic user-topic --partitions 1 --replication-factor 1
kafka-topics --bootstrap-server kafka:9092 --create --if-not-exists --topic group-topic --partitions 1 --replication-factor 1

echo "Topics created successfully."
