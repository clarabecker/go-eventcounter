# Contador de Eventos

Este projeto é a implementação de um microsserviço em Go que atende a um desafio técnico.  
O serviço consome eventos de uma fila no **RabbitMQ**, realiza a contagem de eventos por tipo (`created`, `updated`, `deleted`) e por utilizador, e persiste os resultados em arquivos JSON após um período de inatividade.

---

## Como Executar

### Pré-requisitos
- Go (1.18+)  
- Docker e Docker Compose  

### 1. Instalar Dependências
```bash
go mod tidy
```
### 2. Subir Ambiente
```bash
make env-up
```
### 3. Publicar Mensagem
```bash
make generator-publish
```
### 4. Executar 
```bash
go run ./cmd/consumer/main.go
```
### 5. Desligar ambiente 
```bash
make env-down
```
## Executar Teste de Integração 

- É necessário realizar a etapa 2 e 3 do passo a passo anterior. 

- Declaração da variável
```bash
export RABBITMQ_URL="amqp://guest:guest@localhost:5672"
```
- Execução do teste 
```bash
 go test ./... -v
