package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/reb-felipe/eventcounter/pkg/config"
	"github.com/reb-felipe/eventcounter/pkg/service"
)

func main() {
	log.Println("Contador de Eventos")

	// carrega arquivo .env
	err := godotenv.Load() 
	if err != nil {
		log.Println("Aviso: Erro ao carregar arquivo .env")
	}

	// carregar configuração
	settings, err := config.Load()
	if err != nil {
		log.Fatal(" Erro de carregamento de configuração:", err)
	}

	// contexto para desligamento 
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// captura de sinais do sistema
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Println("Sinal de desligamento, finalizando...")
		cancel()
	}()

	// inicializa serviço
	counterService, err := service.New(settings)
	if err != nil {
		log.Fatal("Erro de criação do serviço:", err)
	}

	log.Println("Serviço rodando... aguardando mensagens")

	// roda até receber sinal
	if err := counterService.Run(ctx); err != nil {
		log.Fatal("Erro em execução:", err)
	}

	log.Println("Serviço finalizado!")
}
