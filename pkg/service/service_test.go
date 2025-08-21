package service

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rabbitmq/amqp091-go"
	"github.com/reb-felipe/eventcounter/pkg/config"
)

func TestIntegration(t *testing.T){

	url := os.Getenv("RABBITMQ_URL")
	t.Logf("debug: Lendo a variável RABBITMQ_URL: '%s'", url)

	if url == "" {
		t.Skip("Pulando: RABBITMQ_URL não definida")
	}

	// Setup 
	cfg := &config.Config{
		RabbitMQURL:      os.Getenv("RABBITMQ_URL"),
		RabbitMQExchange: "eventcountertest",
	}

	service, err := New(cfg)
	if err != nil {
		t.Fatalf("Erro ao criar o serviço: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Execição em goroutine 
	go func(){
		if err := service.Run(ctx); err != nil {
			log.Printf("Erro ao rodar serviço no teste: %v", err)
		}
	}()

	time.Sleep(1 * time.Second)

	// Mensagem de teste no RabbitMQ 
	err = publishTestMessage(cfg, "test-user-123.event.created", `{"id": "`+ uuid.NewString()+`"}`)
	if err != nil {
		t.Fatalf("Falha ao publicar mensagem de teste: %v", err)
	}

		
	time.Sleep(shutdownTimeout + 1*time.Second)

	expectedFile := filepath.Join("output", "created.json")
	defer os.Remove(expectedFile) 

	if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
		t.Fatalf("Arquivo de resultado %s não foi criado", expectedFile)
	}

	content, err := os.ReadFile(expectedFile)
	if err != nil {
		t.Fatalf("Erro ao ler o arquivo de resultado: %v", err)
	}

	var results map[string]int
	if err := json.Unmarshal(content, &results); err != nil {
		t.Fatalf("Erro ao decodificar o JSON do resultado: %v", err)
	}

	if count := results["test-user-123"]; count != 1 {
		t.Errorf("Esperado 1 evento 'Created' para test-user-123, mas obteve %d", count)
	}

}


// Função para publicar uma mensagem
func publishTestMessage(cfg *config.Config, routingKey, body string) error {
	conn, err := amqp091.Dial(cfg.RabbitMQURL)
	if err != nil {
		return err
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	return ch.PublishWithContext(
		context.Background(),
		cfg.RabbitMQExchange,
		routingKey,
		false,
		false,
		amqp091.Publishing{
			ContentType: "application/json",
			Body:        []byte(body),
		},
	)
}
