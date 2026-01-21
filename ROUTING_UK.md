# Роутинг трафіку в Podman Swarm

> **Примітка**: Цей документ описує HTTP/HTTPS роутинг через Ingress.
> Для TCP комунікації між сервісами див. [SERVICE_COMMUNICATION_UK.md](SERVICE_COMMUNICATION_UK.md)

## Архітектура роутингу

Роутинг трафіку в Podman Swarm працює на двох рівнях:

### 1. Ingress Controller (Рівень 1 - Вхідний трафік)

Ingress Controller запускається на **кожній ноді** кластеру на порту 80 (за замовчуванням). Він обробляє вхідні HTTP/HTTPS запити та маршрутизує їх до відповідних сервісів.

**Потік запиту:**
```
Користувач → Ingress (Node A:80) → Service Discovery → Pod (Node B:8080)
```

### 2. Service Discovery (Рівень 2 - Внутрішній роутинг)

Service Discovery зберігає інформацію про всі поди сервісу та їх розташування в кластері. Кожна нода має повну картину всіх сервісів завдяки синхронізації через memberlist.

## Детальний процес роутингу

### Крок 1: Запит надходить на Ingress

```
HTTP Request: GET http://example.com/api/users
Host: example.com
Path: /api/users
```

### Крок 2: Ingress знаходить правило

Ingress Controller перевіряє всі зареєстровані Ingress правила та знаходить відповідне:
- Перевіряє `Host` header
- Перевіряє `Path` на відповідність (Exact, Prefix, ImplementationSpecific)
- Знаходить відповідний Service

### Крок 3: Service Discovery знаходить ендпоінти

```go
addresses, err := discovery.GetServiceAddresses("nginx-service", "default")
// Повертає: ["node1:8080", "node2:8080", "node3:8080"]
```

Service Discovery повертає список всіх здорових ендпоінтів сервісу з усіх нод кластеру.

### Крок 4: Балансування навантаження

Ingress Controller вибирає ендпоінт за алгоритмом round-robin:
- Якщо под на тій самій ноді - прямий доступ через localhost
- Якщо под на іншій ноді - проксування через HTTP до віддаленої ноди

### Крок 5: Проксування запиту

```go
// Якщо под на локальній ноді
target = "localhost:8080"

// Якщо под на віддаленій ноді
target = "node2:8080"  // Проксування через HTTP
```

## Приклад роутингу між нодами

### Сценарій: 3 ноди, 3 поди сервісу

```
Node 1:
  - Ingress Controller (port 80)
  - Pod nginx-1 (port 8080)

Node 2:
  - Ingress Controller (port 80)
  - Pod nginx-2 (port 8080)

Node 3:
  - Ingress Controller (port 80)
  - Pod nginx-3 (port 8080)
```

**Запит 1:** `GET http://example.com/` на Node 1
- Ingress на Node 1 знаходить правило для `example.com`
- Service Discovery повертає: `["node1:8080", "node2:8080", "node3:8080"]`
- Round-robin вибирає `node1:8080` (локальний)
- Проксування до `localhost:8080` ✅

**Запит 2:** `GET http://example.com/` на Node 2
- Ingress на Node 2 знаходить правило
- Service Discovery повертає ті самі адреси
- Round-robin вибирає `node2:8080` (локальний)
- Проксування до `localhost:8080` ✅

**Запит 3:** `GET http://example.com/` на Node 1
- Round-robin вибирає `node2:8080` (віддалений)
- Проксування через HTTP до `node2:8080` ✅

## Реалізація балансування навантаження

Поточна реалізація використовує **round-robin з індексом**:

```go
// Round-robin selection
idx := ic.roundRobinIdx[proxyKey]
selectedEndpoint := endpoints[idx%len(endpoints)]
ic.roundRobinIdx[proxyKey] = (idx + 1) % len(endpoints)
```

**Особливості:**
- ✅ Реальний round-robin (кожен запит обирає наступний ендпоінт)
- ✅ Health-aware routing (тільки здорові ендпоінти)
- ✅ Локальна оптимізація (локальні поди мають пріоритет)

**Планується покращення:**
- Weighted round-robin
- Least connections
- Sticky sessions
- Circuit breaker для нездорових ендпоінтів

## Особливості реалізації

### 1. Локальний vs Віддалений доступ

```go
if selectedEndpoint.NodeName == ic.localNodeName {
    // Локальний под - прямий доступ через localhost
    target = fmt.Sprintf("localhost:%d", selectedEndpoint.Port)
} else {
    // Віддалений под - проксування через HTTP до ноди
    // Address містить реальну IP адресу ноди з кластеру
    target = fmt.Sprintf("%s:%d", selectedEndpoint.Address, selectedEndpoint.Port)
}
```

**Важливо:** 
- Локальні поди доступні через `localhost:port`
- Віддалені поди доступні через `node-ip:port`
- Адреса ноди отримується з кластеру при реєстрації сервісу

### 2. Синхронізація Service Discovery

Кожна нода має повну картину сервісів:
- При реєстрації сервісу - broadcast через memberlist
- При отриманні оновлення - синхронізація локального реєстру
- Health check кожні 10 секунд

### 3. Кешування проксі

Ingress Controller кешує reverse proxy для кожного сервісу:
```go
proxies[serviceKey] = httputil.NewSingleHostReverseProxy(targetURL)
```

## Обмеження та покращення

### Поточні можливості:
1. ✅ Реальний round-robin з індексом
2. ✅ Health-aware routing (тільки здорові ендпоінти)
3. ✅ Локальна оптимізація (локальні поди)
4. ✅ Автоматична синхронізація між нодами

### Планується:
1. ⏳ Weighted balancing
2. ⏳ Least connections
3. ⏳ Sticky sessions (на основі cookies)
4. ⏳ Circuit breaker для нездорових ендпоінтів
5. ⏳ Метрики та моніторинг роутингу

## Діаграма роутингу

```
                    ┌─────────────────┐
                    │   User Request   │
                    └────────┬────────┘
                             │
                    ┌────────▼────────┐
                    │  Ingress (Node) │
                    │   Port 80/443   │
                    └────────┬────────┘
                             │
                    ┌────────▼────────┐
                    │ Service Discovery│
                    │  (Local Registry)│
                    └────────┬────────┘
                             │
            ┌────────────────┼────────────────┐
            │                │                │
    ┌───────▼──────┐  ┌──────▼──────┐  ┌──────▼──────┐
    │  Pod (Node1) │  │  Pod (Node2)│  │  Pod (Node3)│
    │   :8080      │  │   :8080     │  │   :8080     │
    └──────────────┘  └─────────────┘  └─────────────┘
```

## Налаштування

### Вимкнення Ingress на ноді

```bash
./podman-swarm-agent --enable-ingress=false
```

### Зміна порту Ingress

```bash
./podman-swarm-agent --ingress-port=8080
```

### Приклад Kubernetes Ingress маніфесту

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: example-ingress
spec:
  rules:
  - host: example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: nginx-service
            port:
              number: 80
```
