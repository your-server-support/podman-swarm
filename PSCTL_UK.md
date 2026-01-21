# psctl - Podman Swarm CLI

`psctl` - це інструмент командного рядка для управління Podman Swarm кластерами, натхненний `kubectl`.

## Встановлення

### Збірка з Вихідного Коду

```bash
# Збудувати psctl
make build-psctl

# Або збудувати і агент, і psctl
make build-all

# Встановити в $GOPATH/bin
make install
```

### Розташування Бінарного Файлу

Після збірки бінарний файл буде доступний:
- `./psctl` (в корені проекту)
- `$GOPATH/bin/psctl` (після `make install`)

## Конфігурація

`psctl` можна налаштувати трьома способами (за пріоритетом):

1. **Прапорці командного рядка**: `--server`, `--token`, `--namespace`
2. **Змінні оточення**: `PSCTL_SERVER`, `PSCTL_TOKEN`
3. **Файл конфігурації**: `~/.psctl/config`

### Налаштування Конфігурації

```bash
# Встановити URL API сервера
psctl config set-server http://localhost:8080

# Встановити токен авторизації
psctl config set-token <ваш-токен>

# Переглянути поточну конфігурацію
psctl config view

# Отримати розташування файлу конфігурації
psctl config get-location
```

### Формат Файлу Конфігурації

`~/.psctl/config`:
```yaml
server: http://localhost:8080
token: ваш-api-токен
```

## Використання

### Основні Команди

```bash
# Застосувати маніфест
psctl apply -f deployment.yaml

# Отримати ресурси
psctl get pods
psctl get deployments
psctl get services
psctl get nodes

# Отримати конкретний ресурс
psctl get pod nginx-0
psctl get deployment nginx

# Отримати ресурси в конкретному namespace
psctl get pods -n production

# Видалити ресурси
psctl delete deployment nginx
psctl delete service nginx-service

# Описати ресурс
psctl describe pod nginx-0
psctl describe deployment nginx

# Отримати логи (заглушка)
psctl logs nginx-0
```

### Формати Виводу

```bash
# Табличний вивід за замовчуванням
psctl get pods

# JSON вивід
psctl get pods -o json

# Показати мітки
psctl get pods --show-labels
```

### Авторизація

Якщо на сервері увімкнена авторизація API, передайте токен:

```bash
# Через прапорець
psctl get pods --token <ваш-токен>

# Через конфігурацію
psctl config set-token <ваш-токен>
psctl get pods

# Через змінну оточення
export PSCTL_TOKEN=<ваш-токен>
psctl get pods
```

## Довідник Команд

### apply

Застосувати конфігурацію до ресурсів з файлу.

```bash
psctl apply -f <filename>
psctl apply -f deployment.yaml
cat deployment.yaml | psctl apply -f -
```

**Прапорці:**
- `-f, --filename`: Ім'я файлу або `-` для stdin (обов'язково)
- `-n, --namespace`: Namespace (за замовчуванням: "default")
- `--server`: URL API сервера
- `--token`: Токен авторизації

### get

Відобразити один або декілька ресурсів.

```bash
psctl get <resource> [name]
psctl get pods
psctl get pod nginx-0
psctl get deployments -n production
```

**Ресурси:**
- `pods`, `pod`, `po`: Список або отримати поди
- `deployments`, `deployment`, `deploy`: Список або отримати деплойменти
- `services`, `service`, `svc`: Список або отримати сервіси
- `nodes`, `node`: Список нод

**Прапорці:**
- `-o, --output`: Формат виводу (json|yaml)
- `--show-labels`: Показати мітки ресурсів
- `-n, --namespace`: Namespace (за замовчуванням: "default")
- `--server`: URL API сервера
- `--token`: Токен авторизації

### delete

Видалити ресурс за ім'ям.

```bash
psctl delete <resource> <name>
psctl delete deployment nginx
psctl delete service nginx-service -n production
```

**Прапорці:**
- `-n, --namespace`: Namespace (за замовчуванням: "default")
- `--server`: URL API сервера
- `--token`: Токен авторизації

### describe

Показати детальну інформацію про конкретний ресурс.

```bash
psctl describe <resource> <name>
psctl describe pod nginx-0
psctl describe deployment nginx
psctl describe node node-1
```

**Прапорці:**
- `-n, --namespace`: Namespace (за замовчуванням: "default")
- `--server`: URL API сервера
- `--token`: Токен авторизації

### logs

Вивести логи поду (заглушка - не повністю реалізовано).

```bash
psctl logs <pod-name>
psctl logs nginx-0
```

**Прапорці:**
- `-f, --follow`: Стежити за виводом логів (не реалізовано)
- `--tail`: Кількість рядків для відображення (не реалізовано)
- `-n, --namespace`: Namespace (за замовчуванням: "default")
- `--server`: URL API сервера
- `--token`: Токен авторизації

### config

Управління конфігурацією psctl.

```bash
# Встановити URL сервера
psctl config set-server http://localhost:8080

# Встановити токен авторизації
psctl config set-token <token>

# Переглянути конфігурацію
psctl config view

# Отримати розташування файлу конфігурації
psctl config get-location
```

**Підкоманди:**
- `set-server <url>`: Встановити URL API сервера
- `set-token <token>`: Встановити токен авторизації
- `view`: Переглянути поточну конфігурацію
- `get-location`: Показати шлях до файлу конфігурації

## Приклади

### Деплоймент Застосунку

```bash
# Створити deployment.yaml
cat > deployment.yaml <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx
  namespace: default
spec:
  replicas: 3
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
      - name: nginx
        image: nginx:latest
        ports:
        - containerPort: 80
---
apiVersion: v1
kind: Service
metadata:
  name: nginx-service
  namespace: default
spec:
  selector:
    app: nginx
  ports:
  - port: 80
    targetPort: 80
EOF

# Застосувати маніфест
psctl apply -f deployment.yaml

# Перевірити деплоймент
psctl get deployments
psctl get pods
psctl get services

# Описати ресурси
psctl describe deployment nginx
psctl describe service nginx-service
```

### Управління Ресурсами Через Namespace

```bash
# Список подів у default namespace
psctl get pods

# Список подів у production namespace
psctl get pods -n production

# Видалити деплоймент у staging
psctl delete deployment myapp -n staging
```

### Використання з Авторизацією

```bash
# Налаштувати один раз
psctl config set-server http://192.168.1.100:8080
psctl config set-token abc123xyz...

# Використовувати без вказання credentials
psctl get pods
psctl apply -f app.yaml
psctl get nodes
```

### JSON Вивід та Обробка

```bash
# Отримати поди як JSON
psctl get pods -o json

# Обробити за допомогою jq
psctl get pods -o json | jq '.pods[] | select(.status=="running")'

# Порахувати запущені поди
psctl get pods -o json | jq '.pods | length'
```

## Порівняння з kubectl

`psctl` розроблений схожим до `kubectl` для Kubernetes:

| kubectl | psctl | Опис |
|---------|-------|------|
| `kubectl apply -f file.yaml` | `psctl apply -f file.yaml` | Застосувати маніфест |
| `kubectl get pods` | `psctl get pods` | Список подів |
| `kubectl get pod nginx-0` | `psctl get pod nginx-0` | Отримати конкретний под |
| `kubectl delete deployment nginx` | `psctl delete deployment nginx` | Видалити деплоймент |
| `kubectl describe pod nginx-0` | `psctl describe pod nginx-0` | Описати под |
| `kubectl logs nginx-0` | `psctl logs nginx-0` | Отримати логи |
| `kubectl get pods -n prod` | `psctl get pods -n prod` | З namespace |
| `kubectl get pods -o json` | `psctl get pods -o json` | JSON вивід |

## Вирішення Проблем

### Connection Refused

```bash
# Перевірити URL сервера
psctl config view

# Тестувати підключення
curl http://localhost:8080/api/v1/health

# Оновити URL сервера
psctl config set-server http://correct-url:8080
```

### Помилка Авторизації (401)

```bash
# Перевірити чи встановлено токен
psctl config view

# Отримати новий токен з логів агента або API
psctl config set-token <new-token>
```

### Ресурс Не Знайдено

```bash
# Перевірити namespace
psctl get pods -n default
psctl get pods -n production

# Список всіх ресурсів
psctl get pods
psctl get deployments
psctl get services
```

## Розробка

### Додавання Нових Команд

1. Створити новий файл у `internal/psctl/` (наприклад, `mycommand.go`)
2. Реалізувати функцію `NewMyCommand()` яка повертає `*cobra.Command`
3. Додати команду в `cmd/psctl/main.go`: `rootCmd.AddCommand(psctl.NewMyCommand(...))`

### Тестування

```bash
# Збудувати
make build-psctl

# Запустити
./psctl --help
./psctl apply --help
./psctl get pods
```

## Дивіться також

- [README_UK.md](README_UK.md) - Основна документація проекту
- [AGENTS.md](AGENTS.md) - Документація агента
- [ARCHITECTURE_UK.md](ARCHITECTURE_UK.md) - Огляд архітектури
- [SECURITY_UK.md](SECURITY_UK.md) - Безпека
