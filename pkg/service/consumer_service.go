package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/rabbitmq/amqp091-go"
	"github.com/reb-felipe/eventcounter/pkg/config"
	"github.com/reb-felipe/eventcounter/pkg/eventcounter"
)

const (
	queueName = "eventcountertest"
	shutdownTimeout = 5 * time.Second
)

// implementação
type EventCounterService struct {
	cfg		*config.Config
	mu       sync.Mutex
	wg       sync.WaitGroup
	counts   map[eventcounter.EventType]map[string]int
	processedMessages map[string]bool
}

var _ eventcounter.Consumer = (*EventCounterService)(nil)

// cria uma nova instância serviço
func New(cfg *config.Config) (*EventCounterService, error) {
	return &EventCounterService{
		cfg: cfg,
		counts: map[eventcounter.EventType]map[string]int{
			eventcounter.EventCreated: make(map[string]int),
			eventcounter.EventUpdated: make(map[string]int),
			eventcounter.EventDeleted: make(map[string]int),
		},
		processedMessages: make(map[string]bool),
	}, nil
}

// inicia ciclo de vida.
func (s *EventCounterService) Run(ctx context.Context) error {
	conn, err := amqp091.Dial(s.cfg.RabbitMQURL)
	if err != nil {
		return fmt.Errorf("falha ao conectar ao RabbitMQ: %w", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		return fmt.Errorf("falha ao abrir um canal: %w", err)
	}
	defer ch.Close()

	shutdownTimer := time.NewTimer(shutdownTimeout)
	if !shutdownTimer.Stop() {
		<-shutdownTimer.C
	}

	doneChan := make(chan struct{})

	go func() {
		select {
		case <-shutdownTimer.C:
			log.Printf("Nenhuma mensagem recebida por %s. A iniciar o desligamento...", shutdownTimeout)
			close(doneChan)
		case <-ctx.Done():
			log.Println("Contexto cancelado. A iniciar o desligamento...")
			close(doneChan)
		}
	}()

	log.Println("A começar a consumir mensagens...")

	for {
		select {
		case <-doneChan:
			return s.shutdown()
		default:
			msg, ok, err := ch.Get(queueName, false)
			if err != nil {
				log.Printf("Erro ao obter mensagem: %v. A tentar novamente...", err)
				time.Sleep(1 * time.Second)
				continue
			}
			if !ok {
				// pausa
				time.Sleep(200 * time.Millisecond) 
				continue
			}
			shutdownTimer.Reset(shutdownTimeout)
			s.processMessage(ctx, msg)
		}
	}
}

func (s *EventCounterService) processMessage(ctx context.Context, msg amqp091.Delivery) {
	parts := strings.Split(msg.RoutingKey, ".")
	if len(parts) != 3 || parts[1] != "event" {
		log.Printf("Routing key mal formatada: %s. A rejeitar mensagem.", msg.RoutingKey)
		msg.Nack(false, false)
		return
	}
	uid := parts[0]
	eventType := eventcounter.EventType(parts[2])

	var body eventcounter.PublishedMessage
	if err := json.Unmarshal(msg.Body, &body); err != nil {
		log.Printf("Erro ao descodificar corpo da mensagem: %v. A rejeitar.", err)
		msg.Nack(false, false)
		return
	}
	messageID := body.ID

	s.mu.Lock()
	if s.processedMessages[messageID] {
		s.mu.Unlock()
		log.Printf("Mensagem duplicada (ID: %s). A ignorar.", messageID)
		msg.Ack(false)
		return
	}
	s.processedMessages[messageID] = true
	s.mu.Unlock()

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		var err error
		switch eventType {
		case eventcounter.EventCreated:
			err = s.Created(ctx, uid)
		case eventcounter.EventUpdated:
			err = s.Updated(ctx, uid)
		case eventcounter.EventDeleted:
			err = s.Deleted(ctx, uid)
		default:
			log.Printf("Tipo de evento desconhecido: %s", eventType)
			msg.Nack(false, false)
			return
		}
		if err != nil {
			log.Printf("Erro ao processar evento '%s' para o utilizador '%s': %v", eventType, uid, err)
			msg.Nack(false, false)
		} else {
			msg.Ack(false)
		}
	}()
}

func (s *EventCounterService) Created(ctx context.Context, uid string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.counts[eventcounter.EventCreated][uid]++
	log.Printf("[CREATED] Utilizador: %s", uid)
	return nil
}

func (s *EventCounterService) Updated(ctx context.Context, uid string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.counts[eventcounter.EventUpdated][uid]++
	log.Printf("[UPDATED] Utilizador: %s", uid)
	return nil
}

func (s *EventCounterService) Deleted(ctx context.Context, uid string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.counts[eventcounter.EventDeleted][uid]++
	log.Printf("[DELETED] Utilizador: %s", uid)
	return nil
}

func (s *EventCounterService) shutdown() error {
	log.Println("Aguadando goroutines de processamento terminarem")
	s.wg.Wait()
	log.Println("Todas terminaram.")
	return s.writeResultsToJSON()
}

func (s *EventCounterService) writeResultsToJSON() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	outputDir := "output"
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Println("Erro ao criar diretório output:", err)
		return err
	}

	log.Println("Salvando arquivos JSON no diretório...")
	for eventType, userCounts := range s.counts {
		
		if len(userCounts) == 0 {
			continue
		}

		fileName := fmt.Sprintf("%s.json", eventType)
		fullPath := filepath.Join(outputDir, fileName)

		file, err := json.MarshalIndent(userCounts, "", "  ")
		if err != nil {
			log.Printf("Erro ao fazer marshal do JSON %s: %v\n", eventType, err)
			continue
		}


		if err := os.WriteFile(fullPath, file, 0644); err != nil {
			log.Printf("Erro ao escrever em %s: %v\n", fullPath, err)
			continue
		}
		log.Printf("Resultados para '%s' guardados em %s", eventType, fullPath)
	}
	return nil
}