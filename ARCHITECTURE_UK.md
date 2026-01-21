# Архітектура Podman Swarm

## Загальний огляд

Podman Swarm - це кластерний оркестратор для Podman, який забезпечує сумісність з Kubernetes маніфестами та peer-to-peer архітектуру.

## Компоненти

### 1. Cluster (internal/cluster)
- **Призначення**: Керування peer-to-peer кластером
- **Технологія**: HashiCorp Memberlist
- **Функції**:
  - Об'єднання нод у кластер
  - Відстеження стану нод
  - Розповсюдження повідомлень між нодами

### 2. Parser (internal/parser)
- **Призначення**: Парсинг Kubernetes маніфестів
- **Підтримувані ресурси**:
  - Deployment
  - Service
  - Ingress
- **Функції**:
  - Конвертація YAML в Kubernetes об'єкти
  - Витягування інформації про поди, сервіси та ingress

### 3. Scheduler (internal/scheduler)
- **Призначення**: Розподіл подів по нодах
- **Стратегії**:
  - Random selection (за замовчуванням)
  - Node selector (для прив'язки до конкретних нод)
- **Функції**:
  - Вибір ноди для поду
  - Відстеження розподілу подів

### 4. Podman Client (internal/podman)
- **Призначення**: Інтеграція з Podman
- **Функції**:
  - Створення контейнерів
  - Запуск/зупинка контейнерів
  - Отримання статусу
  - Pull образів

### 5. Service Discovery (internal/discovery)
- **Призначення**: Service discovery на основі memberlist
- **Технологія**: Власна реалізація з синхронізацією через memberlist broadcast
- **Функції**:
  - Реєстрація сервісів локально на кожній ноді
  - Синхронізація через memberlist broadcast
  - Пошук сервісів з автоматичним балансуванням
  - Health checking ендпоінтів
  - Відстеження змін сервісів

### 6. DNS Server (internal/dns)
- **Призначення**: DNS резолвінг сервісів та зовнішніх доменів
- **Функції**:
  - Резолвінг сервісів через DNS імена (формат: `service.namespace.cluster.local`)
  - Підтримка A та SRV записів
  - Forwarding зовнішніх DNS запитів до upstream DNS серверів
  - DNS whitelist для контролю зовнішніх доменів
  - Підтримка CNAME записів в whitelist
- **Технологія**: miekg/dns

### 7. Ingress Controller (internal/ingress)
- **Призначення**: Роутинг HTTP/HTTPS трафіку
- **Функції**:
  - Обробка Ingress правил
  - Reverse proxy до сервісів
  - Балансування навантаження

### 8. API Server (internal/api)
- **Призначення**: REST API для керування кластером
- **Ендпоінти**:
  - `POST /api/v1/manifests` - Застосування маніфесту
  - `DELETE /api/v1/manifests/:namespace/:name` - Видалення ресурсу
  - `GET /api/v1/pods` - Список подів
  - `GET /api/v1/deployments` - Список деплойментів
  - `GET /api/v1/services` - Список сервісів
  - `GET /api/v1/services/:namespace/:name/endpoints` - Ендпоінти сервісу
  - `GET /api/v1/services/:namespace/:name/addresses` - Адреси сервісу
  - `GET /api/v1/nodes` - Список нод
  - `GET /api/v1/dns/whitelist` - Отримати DNS whitelist
  - `PUT /api/v1/dns/whitelist` - Встановити DNS whitelist
  - `POST /api/v1/dns/whitelist/hosts` - Додати хост до whitelist
  - `DELETE /api/v1/dns/whitelist/hosts/:host` - Видалити хост з whitelist

## Потік роботи

### Деплоймент Deployment

1. Користувач відправляє Kubernetes маніфест на API
2. Parser парсить маніфест і витягує інформацію про Deployment
3. Scheduler визначає, на яку ноду розмістити кожен под
4. На відповідній ноді Podman Client створює контейнер
5. Контейнер запускається
6. Статус оновлюється в Scheduler

### Деплоймент Service

1. Parser витягує інформацію про Service
2. Service реєструється локально через Service Discovery
3. Всі поди, що відповідають selector, реєструються як ендпоінти
4. Інформація про сервіси синхронізується між нодами через memberlist
5. DNS сервер автоматично резолвить сервіс через DNS ім'я (`service.namespace.cluster.local`)

### Деплоймент Ingress

1. Parser витягує інформацію про Ingress
2. Ingress Controller додає правила роутингу
3. Запити до вказаного хосту/шляху проксуються до відповідного сервісу

## Балансування навантаження

- **DNS level**: Кілька A-записів для кожного сервісу (по одному на ендпоінт)
- **Service level**: Round-robin між подами сервісу через власний service discovery
- **Ingress level**: Round-robin між сервісами через Ingress Controller з локальною оптимізацією

## Персистентність

- Поди можуть бути прив'язані до конкретної ноди через `nodeSelector`
- Це дозволяє використовувати локальні volumes для персистентних даних

## Масштабування

- Всі ноди рівноправні (peer-to-peer)
- Нова нода може приєднатися до кластеру через параметр `--join`
- Scheduler автоматично розподіляє нові поди по всіх нодах
- DNS сервер автоматично синхронізує інформацію про сервіси між нодами

## DNS резолвінг

- **Внутрішні сервіси**: Резолвляться через DNS імена формату `service.namespace.cluster.local`
- **Зовнішні домени**: Forwarding до upstream DNS серверів (за замовчуванням 8.8.8.8, 8.8.4.4)
- **Whitelist**: Можливість обмежити зовнішні DNS запити білим списком доменів
- **CNAME підтримка**: Перевірка CNAME записів в whitelist для безпеки
