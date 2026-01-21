# TCP комунікація між сервісами в Podman Swarm

## Архітектура комунікації

В Podman Swarm сервіси можуть комунікувати між собою трьома способами:

1. **Через Ingress (HTTP/HTTPS)** - для зовнішнього трафіку
2. **Через DNS резолвінг** - для внутрішньої комунікації між сервісами (рекомендовано)
3. **Пряма TCP комунікація через Service Discovery API** - для внутрішньої комунікації між сервісами

## DNS резолвінг сервісів (рекомендований спосіб)

### Як це працює

Podman Swarm надає вбудований DNS сервер, який автоматично резолвить імена сервісів в IP адреси. Це найпростіший та найзручніший спосіб для контейнерів знаходити інші сервіси.

**Переваги DNS резолвінгу:**
- ✅ Стандартний підхід (сумісний з Kubernetes)
- ✅ Не потрібно знати Service Discovery API
- ✅ Працює з будь-якою мовою програмування
- ✅ Автоматичне балансування через кілька A-записів
- ✅ Підтримка SRV записів для портів

### Формат DNS імен

Сервіси доступні за наступними DNS іменами:

```
<service-name>.<namespace>.cluster.local
<service-name>.<namespace>.svc.cluster.local  # Kubernetes сумісність
```

**Приклади:**
- `postgres-service.default.cluster.local` → резолвиться в IP адреси всіх ендпоінтів
- `redis.cache.cluster.local` → резолвиться в IP адреси Redis сервісу
- `api.production.cluster.local` → резолвиться в IP адреси API сервісу

### Автоматична конфігурація

Кожен контейнер автоматично налаштовується для використання DNS сервера кластера:
- DNS сервер запускається на кожній ноді
- Контейнери отримують DNS сервер через `--dns` опцію Podman
- Запити на `cluster.local` обробляються локально
- Інші запити пересилаються до upstream DNS серверів

### Приклад використання

#### Python приклад

```python
import socket

# Просто використовуйте DNS ім'я
host = "postgres-service.default.cluster.local"
port = 5432

# Стандартний DNS резолвінг
ip = socket.gethostbyname(host)
print(f"Resolved {host} to {ip}")

# Підключення до PostgreSQL
import psycopg2
conn = psycopg2.connect(
    host=host,  # DNS ім'я автоматично резолвиться
    port=port,
    database="mydb",
    user="postgres"
)
```

#### Go приклад

```go
package main

import (
    "net"
    "fmt"
)

func main() {
    // DNS резолвінг
    host := "postgres-service.default.cluster.local"
    addrs, err := net.LookupHost(host)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Resolved %s to: %v\n", host, addrs)
    
    // Підключення до першого IP
    conn, err := net.Dial("tcp", fmt.Sprintf("%s:5432", addrs[0]))
    if err != nil {
        log.Fatal(err)
    }
    defer conn.Close()
}
```

#### Node.js приклад

```javascript
const dns = require('dns');
const net = require('net');

// DNS резолвінг
const host = 'postgres-service.default.cluster.local';
dns.lookup(host, (err, address) => {
    if (err) {
        console.error(err);
        return;
    }
    
    console.log(`Resolved ${host} to ${address}`);
    
    // Підключення
    const socket = net.createConnection(5432, address);
});
```

#### Java приклад

```java
import java.net.InetAddress;

String host = "postgres-service.default.cluster.local";
InetAddress address = InetAddress.getByName(host);
System.out.println("Resolved to: " + address.getHostAddress());

// JDBC connection string
String jdbcUrl = "jdbc:postgresql://" + host + ":5432/mydb";
```

### SRV записи для портів

Для отримання інформації про порти можна використовувати SRV записи:

```
_<port-name>._<protocol>.<service-name>.<namespace>.cluster.local
```

**Приклад:**
```
_http._tcp.api-service.default.cluster.local
```

### Балансування навантаження

DNS сервер повертає кілька A-записів для кожного сервісу (по одному на кожен здоровий ендпоінт). Більшість DNS клієнтів автоматично обирають один з них (round-robin або random).

**Приклад:**
```bash
# DNS запит повертає кілька IP адрес
$ dig postgres-service.default.cluster.local

;; ANSWER SECTION:
postgres-service.default.cluster.local. 60 IN A 10.0.1.1
postgres-service.default.cluster.local. 60 IN A 10.0.1.2
postgres-service.default.cluster.local. 60 IN A 10.0.1.3
```

### Налаштування upstream DNS

DNS сервер автоматично пересилає запити, які не належать до `cluster.local`, до зовнішніх DNS серверів.

**Конфігурація:**
```bash
# Використати Google DNS (за замовчуванням)
--upstream-dns=8.8.8.8:53,8.8.4.4:53

# Використати Cloudflare DNS
--upstream-dns=1.1.1.1:53,1.0.0.1:53

# Використати системний резолвер
--upstream-dns=127.0.0.1:53
```

**Приклад:**
- `postgres-service.default.cluster.local` → обробляється локально
- `google.com` → пересилається до upstream DNS
- `github.com` → пересилається до upstream DNS

## TCP комунікація через Service Discovery API

### Як це працює

Коли сервіс A хоче з'єднатися з сервісом B:

1. **Service Discovery запит**: Сервіс A запитує адреси сервісу B через Service Discovery
2. **Отримання ендпоінтів**: Service Discovery повертає список всіх здорових ендпоінтів сервісу B
3. **Вибір ендпоінта**: Сервіс A обирає ендпоінт (round-robin або інший алгоритм)
4. **TCP з'єднання**: Пряме TCP з'єднання до обраного ендпоінта

### Приклад потоку

```
Service A (Pod на Node 1) хоче з'єднатися з Service B

1. Service A → Service Discovery API
   GET /api/v1/services/namespace/service-b/endpoints

2. Service Discovery повертає:
   [
     {NodeName: "node1", Address: "10.0.1.1", Port: 5432},
     {NodeName: "node2", Address: "10.0.1.2", Port: 5432},
     {NodeName: "node3", Address: "10.0.1.3", Port: 5432}
   ]

3. Service A обирає ендпоінт (наприклад, node2:5432)

4. Service A встановлює TCP з'єднання:
   conn, err := net.Dial("tcp", "10.0.1.2:5432")
```

## Реалізація

### Service Discovery API

Service Discovery надає API для отримання адрес сервісів:

```go
// Отримати всі адреси сервісу
addresses, err := discovery.GetServiceAddresses("postgres-service", "default")
// Повертає: ["10.0.1.1:5432", "10.0.1.2:5432", "10.0.1.3:5432"]

// Отримати детальну інформацію про ендпоінти
endpoints, err := discovery.GetServiceEndpoints("postgres-service", "default")
// Повертає: []*ServiceEndpoint з NodeName, Address, Port, Healthy
```

### Приклад використання в коді

### Варіант 1: DNS резолвінг (рекомендовано)

**Найпростіший спосіб** - просто використовуйте DNS імена:

```go
// Просто використовуйте DNS ім'я - все інше працює автоматично
host := "postgres-service.default.cluster.local"
port := 5432

conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", host, port))
// DNS автоматично резолвиться в IP адресу
```

### Варіант 2: Пряме використання Service Discovery (всередині кластеру)

```go
package main

import (
    "net"
    "fmt"
    "github.com/your-server-support/podman-swarm/internal/discovery"
)

func connectToService(discovery *discovery.Discovery, serviceName, namespace string) (net.Conn, error) {
    // Отримати адреси сервісу
    addresses, err := discovery.GetServiceAddresses(serviceName, namespace)
    if err != nil {
        return nil, fmt.Errorf("service not found: %w", err)
    }

    // Round-robin: обираємо перший ендпоінт
    // В production можна використовувати більш складний алгоритм
    target := addresses[0]

    // Встановити TCP з'єднання
    conn, err := net.Dial("tcp", target)
    if err != nil {
        return nil, fmt.Errorf("failed to connect: %w", err)
    }

    return conn, nil
}
```

### Варіант 3: Через API (для зовнішніх клієнтів або подів)

```go
// Використовувати HTTP API для отримання адрес
resp, err := http.Get("http://node1:8080/api/v1/services/default/postgres-service/addresses")
// Отримати: {"addresses": ["10.0.1.1:5432", "10.0.1.2:5432"]}

// Потім встановити TCP з'єднання
conn, err := net.Dial("tcp", "10.0.1.1:5432")
```

Повний приклад клієнта див. в `examples/service-client.go`

## Особливості реалізації

### 1. Пряме TCP з'єднання

Сервіси встановлюють **прямі TCP з'єднання** до подів:
- Немає проміжних проксі
- Мінімальна затримка
- Підтримка будь-якого протоколу поверх TCP (HTTP, gRPC, PostgreSQL, Redis, тощо)

### 2. Розташування подів

```
Service A (Node 1) → Service B (Node 2)
   ↓                    ↓
TCP Connection: 10.0.1.1:xxxx → 10.0.1.2:5432
```

### 3. Балансування навантаження

Клієнтський сервіс сам обирає ендпоінт:
- Round-robin
- Random selection
- Health-aware (тільки здорові ендпоінти)
- Weighted selection (якщо реалізовано)

### 4. Локальна оптимізація

Якщо под знаходиться на тій самій ноді:
```go
if endpoint.NodeName == localNodeName {
    // Використовувати localhost для меншої затримки
    target = fmt.Sprintf("localhost:%d", endpoint.Port)
} else {
    // Використовувати IP адресу ноди
    target = fmt.Sprintf("%s:%d", endpoint.Address, endpoint.Port)
}
```

## Приклади використання

### Приклад 1: PostgreSQL з'єднання (через DNS)

```go
// Найпростіший спосіб - використовуйте DNS ім'я
dsn := "host=postgres-service.default.cluster.local port=5432 user=postgres dbname=mydb sslmode=disable"
db, err := sql.Open("postgres", dsn)
// PostgreSQL драйвер автоматично резолвить DNS ім'я
```

**Альтернатива через Service Discovery:**
```go
// Отримати адреси PostgreSQL сервісу
addresses, err := discovery.GetServiceAddresses("postgres", "default")
if err != nil {
    log.Fatal(err)
}

// Підключитися до першого ендпоінта
dsn := fmt.Sprintf("host=%s port=%s user=postgres dbname=mydb sslmode=disable",
    strings.Split(addresses[0], ":")[0],
    strings.Split(addresses[0], ":")[1])

db, err := sql.Open("postgres", dsn)
```

### Приклад 2: Redis з'єднання (через DNS)

```go
// Найпростіший спосіб - використовуйте DNS ім'я
client := redis.NewClient(&redis.Options{
    Addr: "redis-service.default.cluster.local:6379",
})
// Redis клієнт автоматично резолвить DNS ім'я
```

**Альтернатива через Service Discovery:**
```go
// Отримати адреси Redis сервісу
endpoints, err := discovery.GetServiceEndpoints("redis", "default")
if err != nil {
    log.Fatal(err)
}

// Обрати здоровий ендпоінт
var selectedEndpoint *discovery.ServiceEndpoint
for _, ep := range endpoints {
    if ep.Healthy {
        selectedEndpoint = ep
        break
    }
}

// Підключитися
client := redis.NewClient(&redis.Options{
    Addr: fmt.Sprintf("%s:%d", selectedEndpoint.Address, selectedEndpoint.Port),
})
```

### Приклад 3: gRPC з'єднання (через DNS)

```go
// Найпростіший спосіб - використовуйте DNS ім'я
conn, err := grpc.Dial("grpc-service.default.cluster.local:50051", grpc.WithInsecure())
if err != nil {
    log.Fatal(err)
}
defer conn.Close()

client := pb.NewMyServiceClient(conn)
// gRPC автоматично резолвить DNS ім'я
```

**Альтернатива через Service Discovery:**
```go
// Отримати адреси gRPC сервісу
addresses, err := discovery.GetServiceAddresses("grpc-service", "default")
if err != nil {
    log.Fatal(err)
}

// Підключитися до сервісу
conn, err := grpc.Dial(addresses[0], grpc.WithInsecure())
if err != nil {
    log.Fatal(err)
}
defer conn.Close()

client := pb.NewMyServiceClient(conn)
```

## Синхронізація Service Discovery

### Автоматична синхронізація

Кожна нода має повну картину всіх сервісів:

1. **Реєстрація**: При створенні Service, всі поди реєструються в Service Discovery
2. **Broadcast**: Інформація про сервіс розсилається через memberlist
3. **Синхронізація**: Всі ноди отримують оновлення та синхронізують локальний реєстр
4. **Health Check**: Кожні 10 секунд перевіряється здоров'я ендпоінтів

### Оновлення в реальному часі

- При додаванні нового поду → автоматично додається до Service Discovery
- При видаленні поду → автоматично видаляється з Service Discovery
- При падінні поду → позначається як нездоровий через 30 секунд

## Порівняння способів комунікації

### DNS резолвінг (рекомендовано)

```
Service A → DNS запит → DNS сервер → Service Discovery → Real Node IP:Port → Direct TCP → Pod
```

**Переваги:**
- ✅ Стандартний підхід (сумісний з Kubernetes)
- ✅ Працює з будь-якою мовою програмування
- ✅ Не потрібно знати Service Discovery API
- ✅ Автоматичне балансування через кілька A-записів
- ✅ Підтримка SRV записів

**Недоліки:**
- ❌ Залежність від DNS сервера
- ❌ Можливі затримки через DNS кеш

### Service Discovery API

```
Service A → Service Discovery API → Real Node IP:Port → Direct TCP → Pod
```

**Переваги:**
- ✅ Прямий доступ до інформації про ендпоінти
- ✅ Можливість вибору конкретного ендпоінта
- ✅ Детальна інформація (health, node name, тощо)

**Недоліки:**
- ❌ Потрібно знати Service Discovery API
- ❌ Додатковий код для інтеграції

## Порівняння з Kubernetes

### Kubernetes
```
Service A → DNS → ClusterIP (Virtual IP) → kube-proxy → iptables/ipvs → Pod
```

### Podman Swarm
```
Service A → DNS → DNS сервер → Service Discovery → Real Node IP:Port → Direct TCP → Pod
```

**Переваги Podman Swarm підходу:**
- ✅ Простіша архітектура (немає віртуальних IP)
- ✅ Пряме з'єднання (менша затримка)
- ✅ Підтримка будь-якого протоколу
- ✅ Легше дебажити (видно реальні адреси)
- ✅ Сумісність з Kubernetes DNS форматом

**Недоліки:**
- ❌ Клієнтський сервіс має сам обирати ендпоінт (через DNS round-robin)
- ❌ Немає автоматичного балансування на рівні мережі (як в Kubernetes)

## Налаштування мережі

### Вимоги

Для комунікації між сервісами потрібно:

1. **DNS сервер**: Автоматично налаштовується на кожній ноді
2. **Мережева доступність**: Всі ноди повинні бути доступні одна одній
3. **Firewall**: Порти подів повинні бути відкриті між нодами
4. **Upstream DNS**: Налаштування зовнішніх DNS серверів (опціонально)

### DNS конфігурація

DNS сервер автоматично налаштовується для контейнерів:
- Контейнери отримують DNS сервер через `--dns` опцію Podman
- DNS сервер слухає на порту 53 (за замовчуванням)
- Запити на `cluster.local` обробляються локально
- Інші запити пересилаються до upstream DNS

**Налаштування upstream DNS:**
```bash
# Через змінну оточення
export UPSTREAM_DNS="8.8.8.8:53,8.8.4.4:53"

# Через параметр командного рядка
./agent --upstream-dns=1.1.1.1:53,1.0.0.1:53
```

### Приклад конфігурації firewall

```bash
# Дозволити комунікацію між нодами
iptables -A INPUT -s 10.0.1.0/24 -j ACCEPT
iptables -A OUTPUT -d 10.0.1.0/24 -j ACCEPT
```

## Діаграма TCP комунікації

```
┌─────────────────────────────────────────────────────────┐
│                    Service A (Node 1)                   │
│  ┌──────────────────────────────────────────────────┐   │
│  │  1. Запит до Service Discovery                   │   │
│  │     GetServiceAddresses("service-b", "default")  │   │
│  └──────────────────┬───────────────────────────────┘   │
│                     │                                   │
│                     ▼                                   │
│  ┌──────────────────────────────────────────────────┐   │
│  │  2. Отримує адреси:                              │   │
│  │     ["10.0.1.2:5432", "10.0.1.3:5432"]           │   │
│  └──────────────────┬───────────────────────────────┘   │
│                     │                                   │
│                     ▼                                   │
│  ┌──────────────────────────────────────────────────┐   │
│  │  3. Встановлює TCP з'єднання                     │   │
│  │     net.Dial("tcp", "10.0.1.2:5432")             │   │
│  └──────────────────┬───────────────────────────────┘   │
└──────────────────────┼──────────────────────────────────┘
                       │
                       │ TCP Connection
                       │
┌──────────────────────▼──────────────────────────────────┐
│                    Service B (Node 2)                   │
│  ┌──────────────────────────────────────────────────┐   │
│  │  Pod слухає на порту 5432                        │   │
│  │  net.Listen("tcp", ":5432")                      │   │
│  └──────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────┘
```

## Рекомендації

1. **Використовуйте DNS резолвінг** як основний спосіб комунікації між сервісами
2. **Використовуйте connection pooling** для ефективності
3. **Реалізуйте retry logic** для обробки тимчасових збоїв
4. **Використовуйте health checks** перед з'єднанням (DNS автоматично фільтрує нездорові ендпоінти)
5. **Кешуйте DNS запити** - більшість DNS клієнтів автоматично кешують результати
6. **Моніторьте затримки** між нодами для оптимізації
7. **Налаштуйте upstream DNS** для швидшого резолву зовнішніх доменів
