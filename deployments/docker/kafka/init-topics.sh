#!/bin/bash

# Kafka Broker 地址 (在 Docker 网络中)
KAFKA_BOOTSTRAP_SERVER="kafka:9092"

echo "Waiting for Kafka to be ready..."
# 简单的重试逻辑，直到 kafka-topics.sh 能连上
until kafka-topics.sh --bootstrap-server $KAFKA_BOOTSTRAP_SERVER --list > /dev/null 2>&1; do
  echo "Kafka is not ready yet - sleeping"
  sleep 2
done

echo "Kafka is ready! Creating topics..."

# 定义 Topic 及其分区数 (副本数目前只能为 1，因为是单节点 Kafka)
# 格式: "Topic名称:分区数"
TOPICS=(
  "message-topic:4"
  "group-topic:2"
  "user-topic:2"
  "recovery-topic:1"
)

for TOPIC_CONF in "${TOPICS[@]}"; do
  TOPIC=$(echo $TOPIC_CONF | cut -d: -f1)
  PARTITIONS=$(echo $TOPIC_CONF | cut -d: -f2)
  
  echo "Creating topic: $TOPIC with $PARTITIONS partitions..."
  kafka-topics.sh --bootstrap-server $KAFKA_BOOTSTRAP_SERVER \
    --create \
    --if-not-exists \
    --topic $TOPIC \
    --partitions $PARTITIONS \
    --replication-factor 1
done

echo "Topic initialization completed successfully!"
