# Podman Swarm - Кластерний Оркестратор для Podman

> ⚠️ **Статус Проекту: Рання Розробка**  
> Цей проект перебуває в активній розробці та ще не готовий для production. API та функціонал можуть змінюватися. Використовуйте на власний ризик у production середовищах.

Кластерний оркестратор для Podman з підтримкою Kubernetes маніфестів.

## Архітектура

- **Peer-to-peer кластер**: Всі ноди рівноправні, використовується HashiCorp Memberlist для керування кластером
- **Kubernetes сумісність**: Підтримка стандартних Kubernetes маніфестів (Deployment, Service, Ingress)
- **Service Discovery**: Власна реалізація на основі memberlist для синхронізації між нодами
- **DNS резолвінг**: Вбудований DNS сервер для резолву сервісів через DNS імена (сумісний з Kubernetes)
- **DNS Whitelist**: Білий список зовнішніх доменів для контролю DNS резолвінгу
- **Ingress**: Ingress контролер на кожній ноді для роутингу запитів
- **Load Balancing**: Автоматичне балансування навантаження між подами
- **Шифрування**: AES-256-GCM шифрування всіх повідомлень між нодами
- **Join Token**: Система токенів для безпечного приєднання нод (як в Docker Swarm)
- **TLS підтримка**: Опціональне TLS шифрування на транспортному рівні

## Компоненти

- `cmd/agent` - Агент, що запускається на кожній ноді
- `internal/api` - API сервер для прийому Kubernetes маніфестів
- `internal/cluster` - Peer-to-peer кластер
- `internal/scheduler` - Scheduler для розподілу подів
- `internal/podman` - Інтеграція з Podman
- `internal/parser` - Парсер Kubernetes маніфестів
- `internal/discovery` - Service discovery (власна реалізація)
- `internal/dns` - DNS сервер для резолву сервісів та зовнішніх доменів
- `internal/ingress` - Ingress контролер
- `internal/security` - Безпека (шифрування, токени, TLS)

## Встановлення

```bash
go mod download
go build -o podman-swarm-agent ./cmd/agent
```

## Запуск

### Перша нода (створює кластер)

```bash
./podman-swarm-agent --node-name=node1 --bind-addr=0.0.0.0:7946
```

При старті буде згенеровано join token, який потрібно використати для приєднання інших нод.

### Приєднання інших нод

```bash
./podman-swarm-agent \
  --node-name=node2 \
  --bind-addr=0.0.0.0:7946 \
  --join=node1:7946 \
  --join-token=<TOKEN_FROM_NODE1>
```

### З шифруванням та TLS

```bash
# Перша нода
./podman-swarm-agent \
  --node-name=node1 \
  --bind-addr=0.0.0.0:7946 \
  --encryption-key=<ENCRYPTION_KEY> \
  --tls-cert=/path/to/cert.pem \
  --tls-key=/path/to/key.pem \
  --tls-ca=/path/to/ca.pem

# Інші ноди
./podman-swarm-agent \
  --node-name=node2 \
  --bind-addr=0.0.0.0:7946 \
  --join=node1:7946 \
  --join-token=<TOKEN> \
  --encryption-key=<ENCRYPTION_KEY> \
  --tls-cert=/path/to/cert.pem \
  --tls-key=/path/to/key.pem \
  --tls-ca=/path/to/ca.pem
```

### З DNS конфігурацією

```bash
# Налаштування DNS сервера
./podman-swarm-agent \
  --node-name=node1 \
  --dns-port=53 \
  --cluster-domain=cluster.local \
  --upstream-dns=8.8.8.8:53,8.8.4.4:53
```

Детальніше про безпеку див. [SECURITY_UK.md](SECURITY_UK.md)

## Використання

### Авторизація API

Увімкніть авторизацію API для production:

```bash
./podman-swarm-agent --enable-api-auth=true
```

Токен буде згенеровано та виведено в логах. Використовуйте його в API запитах:

```bash
# Збережіть токен у змінній
export API_TOKEN="<token-from-logs>"

# Використовуйте в запитах
curl -H "Authorization: Bearer $API_TOKEN" \
  http://localhost:8080/api/v1/pods
```

### Деплоймент маніфесту

Відправте Kubernetes маніфест на API:

```bash
# Без авторизації
curl -X POST http://localhost:8080/api/v1/manifests \
  -H "Content-Type: application/yaml" \
  --data-binary @deployment.yaml

# З авторизацією
curl -H "Authorization: Bearer $API_TOKEN" \
  -X POST http://localhost:8080/api/v1/manifests \
  -H "Content-Type: application/yaml" \
  --data-binary @deployment.yaml
```

### DNS резолвінг сервісів

Сервіси автоматично доступні через DNS імена (рекомендований спосіб):

```bash
# Сервіси резолвляться автоматично через DNS
# Приклад: postgres-service.default.cluster.local
```

### TCP комунікація між сервісами

Сервіси можуть знаходити один одного через Service Discovery API або DNS:

```bash
# Через API
curl http://localhost:8080/api/v1/services/default/postgres-service/addresses
curl http://localhost:8080/api/v1/services/default/postgres-service/endpoints

# Управління DNS whitelist
curl http://localhost:8080/api/v1/dns/whitelist
curl -X PUT http://localhost:8080/api/v1/dns/whitelist \
  -H "Content-Type: application/json" \
  -d '{"enabled": true, "hosts": ["google.com", "github.com"]}'
```

Детальніше про комунікацію між сервісами див. [SERVICE_COMMUNICATION_UK.md](SERVICE_COMMUNICATION_UK.md)

## Документація

- [TODO_UK.md](TODO_UK.md) - Roadmap розробки та заплановані функції
- [AGENTS.md](AGENTS.md) - Документація агента
- [PSCTL_UK.md](PSCTL_UK.md) - Документація CLI інструменту
- [ARCHITECTURE_UK.md](ARCHITECTURE_UK.md) - Архітектура системи
- [ROUTING_UK.md](ROUTING_UK.md) - Роутинг HTTP/HTTPS трафіку
- [SERVICE_COMMUNICATION_UK.md](SERVICE_COMMUNICATION_UK.md) - Комунікація між сервісами (DNS та TCP)
- [SECURITY_UK.md](SECURITY_UK.md) - Безпека та шифрування

Англійські версії:
- [README.md](README.md)
- [PSCTL.md](PSCTL.md)
- [ARCHITECTURE.md](ARCHITECTURE.md)
- [ROUTING.md](ROUTING.md)
- [SERVICE_COMMUNICATION.md](SERVICE_COMMUNICATION.md)
- [SECURITY.md](SECURITY.md)
