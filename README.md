# Практическое задание 8
## Шишков А.Д. ЭФМО-02-22
## Тема
Настройка GitHub Actions для сборки приложения.

## Цель
Освоить основы CI/CD для backend-проекта на Go, настроить автоматический pipeline для запуска тестов, сборки и упаковки Docker-образа.

---

## 1. Что такое CI и CD

**CI (Continuous Integration)** — непрерывная интеграция. После каждого коммита система автоматически:
- устанавливает зависимости;
- запускает тесты;
- выполняет сборку;
- проверяет, что изменения не сломали проект.

**CD (Continuous Delivery / Deployment)** — непрерывная доставка. После успешной CI автоматически:
- собирается Docker-образ;
- публикуется в registry;
- (опционально) разворачивается на сервере.

CI отвечает за качество кода, CD — за доставку результата.

---

## 2. Структура pipeline

Pipeline состоит из двух job, выполняемых последовательно:

```
push/PR в main
    │
    ▼
┌──────────────────┐
│  test-and-build  │
│                  │
│  1. Checkout     │
│  2. Setup Go     │
│  3. Dependencies │
│  4. go test      │
│  5. go build     │
└────────┬─────────┘
         │ (только при успехе)
         ▼
┌──────────────────┐
│  docker-build    │
│                  │
│  1. Checkout     │
│  2. Buildx       │
│  3. docker build │
└──────────────────┘
```

**test-and-build** — проверяет код: скачивает зависимости, запускает unit-тесты, компилирует оба сервиса (tasks и auth).

**docker-build** — запускается только после успешного прохождения тестов (`needs: test-and-build`), собирает Docker-образ через multi-stage build.

---

## 3. YAML-файл pipeline

Файл `.github/workflows/ci.yml`:

```yaml
name: CI Pipeline

on:
  push:
    branches: [ "main", "master" ]
  pull_request:
    branches: [ "main", "master" ]

jobs:
  test-and-build:
    runs-on: ubuntu-latest

    steps:
      - name: Checkout repository
        uses: actions/checkout@v4

      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.23'

      - name: Show Go version
        run: go version

      - name: Download dependencies
        run: go mod download

      - name: Run tests
        run: go test ./...

      - name: Build tasks service
        run: go build ./services/tasks/cmd/tasks

      - name: Build auth service
        run: go build ./services/auth/cmd/auth

  docker-build:
    runs-on: ubuntu-latest
    needs: test-and-build

    steps:
      - name: Checkout repository
        uses: actions/checkout@v4

      - name: Set up Docker Buildx
        uses: docker/setup-buildx-action@v3

      - name: Build Docker image
        run: docker build -t techip-tasks:${{ github.sha }} -f services/tasks/Dockerfile .
```

### Пояснение шагов

| Шаг | Назначение |
|-----|-----------|
| `actions/checkout@v4` | Клонирует репозиторий в runner |
| `actions/setup-go@v5` | Устанавливает Go 1.23 |
| `go mod download` | Загружает зависимости из go.mod |
| `go test ./...` | Запускает все unit-тесты проекта |
| `go build ./services/tasks/cmd/tasks` | Компилирует сервис tasks |
| `go build ./services/auth/cmd/auth` | Компилирует сервис auth |
| `docker/setup-buildx-action@v3` | Настраивает Docker Buildx |
| `docker build` | Собирает Docker-образ с multi-stage build |

---

## 4. Unit-тесты

Для pipeline созданы unit-тесты бизнес-логики сервиса tasks (`services/tasks/internal/service/task_test.go`). Используется mock-реализация интерфейса `TaskRepository`:

```go
type mockRepo struct {
    tasks map[string]*Task
}
```

Покрытые сценарии:

| Тест | Что проверяет |
|------|--------------|
| `TestCreate` | Создание задачи, генерация ID, начальный статус |
| `TestGetAll` | Получение списка всех задач |
| `TestGetByID` | Поиск задачи по ID |
| `TestGetByID_NotFound` | Ошибка при несуществующем ID |
| `TestUpdate` | Обновление полей задачи |
| `TestUpdate_NotFound` | Ошибка при обновлении несуществующей задачи |
| `TestUpdate_PartialFields` | Частичное обновление (только указанные поля) |
| `TestDelete` | Удаление задачи |
| `TestDelete_NotFound` | Ошибка при удалении несуществующей задачи |
| `TestSearchByTitle` | Поиск задач по заголовку |

---

## 5. Формирование тега Docker-образа

Тег формируется автоматически на основе SHA коммита:

```yaml
docker build -t techip-tasks:${{ github.sha }} .
```

Это обеспечивает:
- **уникальность** — каждый коммит получает свой образ;
- **трассируемость** — по тегу можно найти точный коммит;
- **воспроизводимость** — повторная сборка того же коммита даёт тот же тег.

---

## 6. Хранение секретов

Секреты (токены, пароли, SSH-ключи) хранятся в **GitHub Secrets** (Settings → Secrets and variables → Actions), а не в репозитории.

Использование в pipeline:

```yaml
- name: Login to registry
  run: echo "${{ secrets.REGISTRY_PASSWORD }}" | docker login -u "${{ secrets.REGISTRY_USERNAME }}" --password-stdin ghcr.io
```

Принципы:
- секреты **не попадают** в репозиторий и не отображаются в логах;
- передаются как переменные окружения только во время выполнения pipeline;
- для публикации Docker-образа в registry потребуются `REGISTRY_USERNAME` и `REGISTRY_PASSWORD`.

---

## 7. Публикация образа в registry (опционально)

Для публикации образа в GitHub Container Registry (ghcr.io) job `docker-build` дополняется шагами:

```yaml
- name: Login to GHCR
  run: echo "${{ secrets.GHCR_TOKEN }}" | docker login ghcr.io -u ${{ github.actor }} --password-stdin

- name: Build and tag image
  run: docker build -t ghcr.io/${{ github.repository }}/tasks:${{ github.sha }} -f services/tasks/Dockerfile .

- name: Push image
  run: docker push ghcr.io/${{ github.repository }}/tasks:${{ github.sha }}
```

После публикации образ можно развернуть на сервере:

```bash
docker pull ghcr.io/<owner>/pz1.2/tasks:<tag>
docker compose up -d
```

---

## 8. Демонстрация

**Pipeline успешно выполнен:**

![CI Pipeline](docs/images/pz8_pipeline.png)
