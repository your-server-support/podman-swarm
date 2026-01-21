# Приклад роутингу трафіку

## Сценарій: 3 ноди, 3 поди nginx

### Конфігурація кластеру

```
Node 1 (10.0.1.1):
  - Ingress Controller: :80
  - Pod: nginx-1 (localhost:8080)

Node 2 (10.0.1.2):
  - Ingress Controller: :80
  - Pod: nginx-2 (localhost:8080)

Node 3 (10.0.1.3):
  - Ingress Controller: :80
  - Pod: nginx-3 (localhost:8080)
```

### Kubernetes маніфест

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
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
spec:
  selector:
    app: nginx
  ports:
  - port: 80
    targetPort: 80
---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: nginx-ingress
spec:
  rules:
  - host: nginx.example.com
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

### Процес роутингу

#### Запит 1: `GET http://nginx.example.com/` на Node 1

1. **Ingress на Node 1** отримує запит
2. Знаходить правило для `nginx.example.com`
3. **Service Discovery** повертає ендпоінти:
   ```
   [
     {NodeName: "node1", Address: "10.0.1.1", Port: 80},
     {NodeName: "node2", Address: "10.0.1.2", Port: 80},
     {NodeName: "node3", Address: "10.0.1.3", Port: 80}
   ]
   ```
4. **Round-robin** (idx=0) вибирає `node1`
5. Оскільки це локальна нода → `localhost:80`
6. ✅ Запит проксується до локального поду

#### Запит 2: `GET http://nginx.example.com/` на Node 1

1. **Round-robin** (idx=1) вибирає `node2`
2. Оскільки це віддалена нода → `10.0.1.2:80`
3. ✅ Запит проксується через HTTP до Node 2
4. Node 2 отримує запит і проксує до локального поду

#### Запит 3: `GET http://nginx.example.com/` на Node 2

1. **Round-robin** (idx=2) вибирає `node3`
2. Оскільки це віддалена нода → `10.0.1.3:80`
3. ✅ Запит проксується через HTTP до Node 3

#### Запит 4: `GET http://nginx.example.com/` на Node 3

1. **Round-robin** (idx=0, цикл почався знову) вибирає `node1`
2. Оскільки це віддалена нода → `10.0.1.1:80`
3. ✅ Запит проксується через HTTP до Node 1

## Візуалізація потоку

```
Запит → Ingress (Node X) 
         ↓
    Service Discovery
         ↓
    Round-robin вибір
         ↓
    ┌────┴────┐
    │         │
Локальний  Віддалений
    │         │
localhost  node-ip:port
    │         │
    └────┬────┘
         ↓
      Pod
```

## Переваги такої архітектури

1. **Висока доступність**: Якщо одна нода падає, трафік автоматично перенаправляється на інші
2. **Балансування**: Навантаження розподіляється рівномірно між всіма подами
3. **Локальна оптимізація**: Локальні поди мають пріоритет (менше мережевих стрибків)
4. **Масштабованість**: Легко додавати нові ноди - вони автоматично включаються в балансування
