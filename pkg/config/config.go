package config

import (
	"errors"
	"os"
)

type Config struct {
	RabbitMQURL      string
	RabbitMQExchange string
}

func Load() (*Config, error) {
	// padrão usado pelo makefile
	rabbitURL := os.Getenv("RABBITMQ_URI")
	if rabbitURL == "" {
		// variável comum
		rabbitURL = os.Getenv("RABBITMQ_URL")
	}
	if rabbitURL == "" {
		return nil, errors.New("variável de ambiente RABBITMQ_URI ou RABBITMQ_URL não definida")
	}

	rabbitExchange := os.Getenv("RABBITMQ_EXCHANGE")
	if rabbitExchange == "" {
		return nil, errors.New("variável de ambiente RABBITMQ_EXCHANGE não definida")
	}

	logMsg := "Configuração carregada: " + rabbitURL + " / " + rabbitExchange
	println(logMsg) // opcional: só para feedback rápido
	return &Config{
		RabbitMQURL:      rabbitURL,
		RabbitMQExchange: rabbitExchange,
	}, nil
}
