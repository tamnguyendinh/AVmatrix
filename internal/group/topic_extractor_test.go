package group

import "testing"

func TestExtractTopicContractsAcrossBrokersAndLanguages(t *testing.T) {
	tmpDir := t.TempDir()
	writeGroupFile(t, tmpDir, "src/EventHandler.java", `@KafkaListener(topics = "user.created")
public void handleUserCreated(ConsumerRecord<String, String> record) {}`)
	writeGroupFile(t, tmpDir, "src/EventPublisher.java", `public class EventPublisher {
    public void publish() {
        kafkaTemplate.send("user.created", payload);
    }
}`)
	writeGroupFile(t, tmpDir, "src/consumer.ts", `await consumer.subscribe({ topic: 'order.placed', fromBeginning: true });
await consumer.run({ eachMessage: async () => {} });`)
	writeGroupFile(t, tmpDir, "src/producer.ts", `await producer.send({ topic: 'order.placed', messages: [{ value: JSON.stringify(order) }] });`)
	writeGroupFile(t, tmpDir, "src/OrderListener.java", `@RabbitListener(queues = "order-queue")
public void processOrder(OrderMessage msg) {}`)
	writeGroupFile(t, tmpDir, "src/Publisher.java", `rabbitTemplate.convertAndSend("order-exchange", "order.new", payload);`)
	writeGroupFile(t, tmpDir, "src/worker.ts", `channel.consume("task-queue", (msg) => {
  console.log(msg.content.toString());
});`)
	writeGroupFile(t, tmpDir, "src/rabbit-publisher.ts", `channel.publish("events", "user.signup", Buffer.from(JSON.stringify(data)));
channel.sendToQueue("job-queue", Buffer.from(msg));`)
	writeGroupFile(t, tmpDir, "src/stream.go", `package main
func main() {
  js.Publish("orders.created", payload)
  js.Subscribe("orders.created", handler)
  nc.Subscribe("updates.weather", func(m *nats.Msg) {})
  nc.Publish("updates.weather", []byte("sunny"))
}`)
	writeGroupFile(t, tmpDir, "src/subscriber.py", `nc = await nats.connect()
await nc.subscribe("orders.created", cb=handler)
await nc.publish("orders.created", payload)`)
	writeGroupFile(t, tmpDir, "internal/consumer.go", `package consumer
partConsumer, _ := consumer.ConsumePartition("inventory.update", 0, sarama.OffsetNewest)`)
	writeGroupFile(t, tmpDir, "internal/publisher.go", `package publisher
producer, _ := sarama.NewSyncProducer(brokers, cfg)
for _, item := range items {
    msg1 := &sarama.ProducerMessage{Topic: "order.created"}
    msg2 := &sarama.ProducerMessage{Topic: "order.shipped"}
    _ = msg1
    _ = msg2
}
writer := &kafka.Writer{Topic: "inventory.update"}
reader := kafka.NewReader(kafka.ReaderConfig{Topic: "inventory.update"})`)
	writeGroupFile(t, tmpDir, "app/consumer.py", `from kafka import KafkaConsumer
consumer = KafkaConsumer('payment.processed', bootstrap_servers=['localhost:9092'])`)
	writeGroupFile(t, tmpDir, "app/producer.py", `from kafka import KafkaProducer
producer.send('payment.processed', value=msg)`)

	contracts, err := ExtractTopicContracts(tmpDir)
	if err != nil {
		t.Fatalf("ExtractTopicContracts() error = %v", err)
	}
	for _, want := range []struct {
		id     string
		role   string
		broker string
	}{
		{"topic::user.created", "consumer", "kafka"},
		{"topic::user.created", "provider", "kafka"},
		{"topic::order.placed", "consumer", "kafka"},
		{"topic::order.placed", "provider", "kafka"},
		{"topic::order-queue", "consumer", "rabbitmq"},
		{"topic::order-exchange", "provider", "rabbitmq"},
		{"topic::task-queue", "consumer", "rabbitmq"},
		{"topic::events", "provider", "rabbitmq"},
		{"topic::job-queue", "provider", "rabbitmq"},
		{"topic::orders.created", "consumer", "nats"},
		{"topic::orders.created", "provider", "nats"},
		{"topic::updates.weather", "consumer", "nats"},
		{"topic::updates.weather", "provider", "nats"},
		{"topic::inventory.update", "consumer", "kafka"},
		{"topic::inventory.update", "provider", "kafka"},
		{"topic::order.created", "provider", "kafka"},
		{"topic::order.shipped", "provider", "kafka"},
		{"topic::payment.processed", "consumer", "kafka"},
		{"topic::payment.processed", "provider", "kafka"},
	} {
		contract := findTopicContract(contracts, want.id, want.role)
		if contract == nil {
			t.Fatalf("missing %s %s\nall=%v", want.role, want.id, contractIDs(contracts))
		}
		if contract.Meta["broker"] != want.broker {
			t.Fatalf("%s %s broker = %#v, want %s", want.role, want.id, contract.Meta["broker"], want.broker)
		}
	}
}

func TestExtractTopicContractsDedupesAndSkipsGoTestFiles(t *testing.T) {
	tmpDir := t.TempDir()
	writeGroupFile(t, tmpDir, "src/events.ts", `await producer.send({ topic: 'user.created', messages: [] });
await producer.send({ topic: 'user.created', messages: [] });
await producer.send({ topic: 'user.deleted', messages: [] });
await consumer.subscribe({ topic: 'order.placed' });`)
	writeGroupFile(t, tmpDir, "src/orders_test.go", `package main
func TestFake(t *testing.T) {
  consumer.ConsumePartition("fake-topic", 0, sarama.OffsetNewest)
}`)

	contracts, err := ExtractTopicContracts(tmpDir)
	if err != nil {
		t.Fatalf("ExtractTopicContracts() error = %v", err)
	}
	if findTopicContract(contracts, "topic::fake-topic", "consumer") != nil {
		t.Fatalf("go test file emitted topic contract: %#v", contracts)
	}
	if countTopicContracts(contracts, "topic::user.created", "provider") != 1 {
		t.Fatalf("duplicate producer was not deduped: %#v", contracts)
	}
	if findTopicContract(contracts, "topic::user.deleted", "provider") == nil || findTopicContract(contracts, "topic::order.placed", "consumer") == nil {
		t.Fatalf("expected producer/consumer contracts missing: %#v", contracts)
	}
}

func TestExtractTopicContractsEmptyRepo(t *testing.T) {
	tmpDir := t.TempDir()
	writeGroupFile(t, tmpDir, "src/index.ts", "console.log('hello')")
	contracts, err := ExtractTopicContracts(tmpDir)
	if err != nil {
		t.Fatalf("ExtractTopicContracts() error = %v", err)
	}
	if len(contracts) != 0 {
		t.Fatalf("empty topic repo contracts = %#v", contracts)
	}
}

func findTopicContract(contracts []StoredContract, contractID string, role string) *StoredContract {
	for i := range contracts {
		if contracts[i].ContractID == contractID && contracts[i].Role == role {
			return &contracts[i]
		}
	}
	return nil
}

func countTopicContracts(contracts []StoredContract, contractID string, role string) int {
	count := 0
	for _, contract := range contracts {
		if contract.ContractID == contractID && contract.Role == role {
			count++
		}
	}
	return count
}
