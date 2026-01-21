# Безпека Podman Swarm

## Принципи Безпеки

Podman Swarm дотримується наступних основних принципів безпеки:

1. **Принцип Мінімальних Привілеїв** - Кожен компонент працює з мінімально необхідними правами
2. **Rootless за Замовчуванням** - Використовує rootless можливості Podman для посиленої безпеки
3. **Безпечний за Замовчуванням** - Функції безпеки увімкнені з коробки
4. **Захист в Глибину** - Множинні рівні безпеки (шифрування, автентифікація, авторизація)
5. **Zero Trust** - Вся комунікація автентифікована та зашифрована

## Шифрування комунікації

Podman Swarm підтримує шифрування комунікації між нодами на двох рівнях:

### 1. Шифрування повідомлень (Message-level encryption)

Всі повідомлення між нодами шифруються за допомогою AES-256-GCM. Ключ шифрування може бути вказаний через параметр `--encryption-key` або згенерований автоматично для першої ноди.

**Використання:**
```bash
# Перша нода (генерує ключ автоматично)
./podman-swarm-agent --node-name=node1

# Інші ноди (використовують той самий ключ)
./podman-swarm-agent --node-name=node2 \
  --join=node1:7946 \
  --join-token=<TOKEN> \
  --encryption-key=<KEY>
```

### 2. TLS шифрування (Transport-level encryption)

Для додаткового захисту можна використовувати TLS сертифікати:

```bash
./podman-swarm-agent \
  --node-name=node1 \
  --tls-cert=/path/to/cert.pem \
  --tls-key=/path/to/key.pem \
  --tls-ca=/path/to/ca.pem
```

## Join Token (Токен приєднання)

Система токенів працює аналогічно Docker Swarm:

### Генерація токену

Перша нода автоматично генерує join token при старті:

```bash
./podman-swarm-agent --node-name=node1
# Output: Generated join token: <TOKEN>
```

### Використання токену для приєднання

```bash
./podman-swarm-agent \
  --node-name=node2 \
  --join=node1:7946 \
  --join-token=<TOKEN>
```

### Валідація токену

- Токен валідується при спробі приєднання до кластеру
- Невірний токен блокує приєднання ноди
- Токени можуть бути відкликані через API

## DNS Whitelist

Podman Swarm підтримує білий список зовнішніх доменів для контролю DNS резолвінгу:

### Налаштування whitelist

```bash
# Через API
curl -X PUT http://localhost:8080/api/v1/dns/whitelist \
  -H "Content-Type: application/json" \
  -d '{
    "enabled": true,
    "hosts": ["google.com", "github.com", "docker.io"]
  }'
```

### Особливості

- **За замовчуванням**: Whitelist вимкнений, дозволено резолвити всі домени
- **Підтримка піддоменів**: Якщо дозволено `example.com`, то `api.example.com` також дозволено
- **CNAME перевірка**: Перевіряються всі CNAME цілі в DNS відповідях
- **Блокування**: Запити до недозволених доменів повертають `RcodeRefused`

### Приклад використання

```bash
# Отримати поточну конфігурацію
curl http://localhost:8080/api/v1/dns/whitelist

# Додати хост
curl -X POST http://localhost:8080/api/v1/dns/whitelist/hosts \
  -H "Content-Type: application/json" \
  -d '{"host": "example.com"}'

# Видалити хост
curl -X DELETE http://localhost:8080/api/v1/dns/whitelist/hosts/example.com
```

## Рекомендації по безпеці

1. **Використовуйте сильні ключі шифрування:**
   ```bash
   # Генерація випадкового ключа
   openssl rand -base64 32
   ```

2. **Зберігайте ключі безпечно:**
   - Не комітьте ключі в git
   - Використовуйте секретні менеджери (HashiCorp Vault, AWS Secrets Manager, etc.)
   - Обмежте доступ до файлів з ключами (chmod 600)

3. **Використовуйте TLS сертифікати:**
   - Використовуйте сертифікати від довірених CA
   - Регулярно оновлюйте сертифікати
   - Використовуйте окремі сертифікати для кожної ноди

4. **Обмежте мережевий доступ:**
   - Використовуйте firewall для обмеження доступу до порту кластеру (7946)
   - Використовуйте VPN або приватні мережі для комунікації між нодами

5. **Регулярно ротуйте токени:**
   - Генеруйте нові токени періодично
   - Відкликайте старі токени через API

6. **Використовуйте DNS whitelist:**
   - Увімкніть whitelist для обмеження зовнішніх DNS запитів
   - Додавайте тільки необхідні домени
   - Регулярно перевіряйте список дозволених доменів

## Приклад конфігурації для production

```bash
# node1 (перша нода)
./podman-swarm-agent \
  --node-name=node1 \
  --bind-addr=10.0.1.1:7946 \
  --encryption-key=$(cat /etc/podman-swarm/encryption.key) \
  --tls-cert=/etc/podman-swarm/certs/node1.crt \
  --tls-key=/etc/podman-swarm/certs/node1.key \
  --tls-ca=/etc/podman-swarm/certs/ca.crt

# node2 (приєднання)
./podman-swarm-agent \
  --node-name=node2 \
  --bind-addr=10.0.1.2:7946 \
  --join=node1:7946 \
  --join-token=$(cat /etc/podman-swarm/join.token) \
  --encryption-key=$(cat /etc/podman-swarm/encryption.key) \
  --tls-cert=/etc/podman-swarm/certs/node2.crt \
  --tls-key=/etc/podman-swarm/certs/node2.key \
  --tls-ca=/etc/podman-swarm/certs/ca.crt
```
