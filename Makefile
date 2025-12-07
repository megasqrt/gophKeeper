SERVER_PORT=8080
ADDRESS=localhost:${SERVER_PORT}
TEMP_FILE=$(random tempfile)
KEY=secretKey
DATABASE_URL=postgres://postgres:pwd@localhost:5432/gophkeeper?sslmode=disable
COMPOSE_FILE=docker-compose.yml
#export

echo:
	go version

run_s:
	KEY=$(KEY) DATABASE_URL=$(DATABASE_URL) go run cmd/server/main.go

run_c: copy-certs-to-client
	KEY=$(KEY) go run client/cmd/main.go
	

tests: vet
	go test ./...


vet:
	go vet -vettool=$(go run tools/staticlint/main.go) ./...

race:
	go test -v -race ./...

build:
	go build -o server cmd/server/main.go
	go build -o tui-client client/cmd/main.go

cover:
	go test -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -func=coverage.out 
#go tool cover -html=coverage.out -o coverage.html

mem_test:
	run_s &
	run_a &
	sleep 60 
	go tool pprof -http=":9090" -seconds=30 http://localhost:6060/debug/pprof/heap
	curl -o memprofile.pprof http://localhost:6060/debug/pprof/heap

pprof_compare:
	go tool pprof -top -diff_base=profiles/base.pprof profiles/result.pprof 

fmtall:
	goimports -w ./.

godoc:
	godoc -http=:6070 -goroot="/home/kan/src/ypMetrics/" -play
#http://localhost:6070/pkg/ypMetrics/internal/?m=all

protoc:
	rm -rf internal/proto/keeper.pb.go
	rm -rf internal/proto/keeper_grpc.pb.go
	protoc --go_out=. --go_opt=default_api_level=API_OPAQUE --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative internal/proto/keeper.proto

# Docker

start: stop
	podman-compose -f $(COMPOSE_FILE) up
startb: stop
	podman-compose -f $(COMPOSE_FILE) up --build
stop:
	podman-compose -f $(COMPOSE_FILE) down
startpg:
	podman-compose -f pg-compose.yml up
stoppg:
	podman-compose -f pg-compose.yml down

# Docker--build

# ==============================================================================
# Сертификаты
# Все команды выполняются в директории certs/
# ==============================================================================

# Главная цель для генерации всех сертификатов
certs-all: clean-certs certs-ca certs-server certs-client

# 1. Создание Удостоверяющего Центра (CA)
certs-ca:
	@echo "--> Generating CA key and certificate..."
	cd certs && openssl req -x509 -newkey rsa:4096 -nodes -keyout ca.key -out ca.crt -days 365 -subj "/C=RU/ST=Moscow/L=Moscow/O=GophKeeper/OU=CA/CN=GophKeeper CA"

# 2. Создание сертификата сервера, подписанного CA
certs-server:
	@echo "--> Generating server key and certificate..."
	cd certs && openssl genrsa -out server.key 2048
	cd certs && openssl req -new -key server.key -out server.csr -config server.conf
	cd certs && openssl x509 -req -in server.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out server.crt -days 365 -sha256 -extfile server.conf -extensions v3_req

# 3. Создание сертификата клиента, подписанного CA
certs-client:
	@echo "--> Generating client key and certificate..."
	cd certs && openssl genrsa -out client.key 2048
	cd certs && openssl req -new -key client.key -out client.csr -config client.conf
	cd certs && openssl x509 -req -in client.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out client.crt -days 365 -sha256 -extfile client.conf -extensions v3_req

# 4. Копирование сертификатов в директорию конфигурации клиента
copy-certs-to-client:
	@echo "--> Copying certificates to client config directory ~/.gophkeeper/certs..."
	mkdir -p ~/.gophkeeper/certs
	cp certs/ca.crt ~/.gophkeeper/certs/ca.crt
	cp certs/client.crt ~/.gophkeeper/certs/client.crt
	cp certs/client.key ~/.gophkeeper/certs/client.key

# Очистка сгенерированных сертификатов
clean-certs:
	@echo "--> Cleaning up certificates..."
	rm -f certs/*.crt certs/*.key certs/*.csr certs/*.srl

.PHONY: certs-all certs-ca certs-server certs-client clean-certs start startb stop protoc run_s run_c

tools:
	go install github.com/bufbuild/buf/cmd/buf@latest