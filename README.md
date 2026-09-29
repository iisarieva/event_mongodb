# EventFlow 🚀
Pet project на **Go + MongoDB** для практического изучения MongoDB и её отличий от SQL.

## 💡 Идея

Сервис хранит пользователей и 3 типа событий:

- `user_registered`
- `message_sent`
- `payment_completed`

У каждого события своя структура `properties`, поэтому на проекте можно потренировать flexible schema, индексы, aggregation и моделирование документов.

Пример:

```json
{
  "event_type": "message_sent",
  "user_id": "u1",
  "timestamp": "2026-09-22T10:10:00Z",
  "properties": {
    "chat_id": "c10",
    "channel": "telegram",
    "length": 145
  }
}
```

## 🛠 Стек

- Go
- MongoDB
- MongoDB Go Driver
- Docker Compose

## ✅ TODO

### 1. База
- [x] Поднять MongoDB через Docker
- [x] Подключить Go-приложение к MongoDB
- [x] Настроить один общий `mongo.Client`
- [x] Создать коллекции `users` и `events`

### 2. CRUD
- [x] Создание события
- [x] Получение события по ID
- [x] Получение списка событий
- [x] Фильтрация по `user_id`, `event_type`, датам
- [x] Удаление события
- [x] Добавить sorting / limit / projection
- [x] Обновление события через `$set`

### 3. Flexible schema
- [x] Реализовать 3 типа событий с разными `properties`
- [x] Добавить MongoDB schema validation
- [x] Проверить поведение при некорректных данных

### 4. Modeling
- [x] Добавить `users`
- [x] Сравнить reference через `user_id` и embedded `user_snapshot`
- [x] Разобраться с denormalization
- [x] Описать, почему выбран конкретный вариант

### 5. Arrays
- [x] Добавить `tags` пользователю
- [x] Фильтрация по тегам
- [x] Попробовать `$in`, `$all`, `$elemMatch`
- [ ] Создать multikey index

### 6. Performance и индексы
- [x] Сгенерировать ~10 000 users и ~200 000 events
- [x] Выполнить запрос без индекса
- [x] Проверить `explain("executionStats")`
- [x] Найти `COLLSCAN`
- [x] Добавить single-field index
- [x] Добавить compound index
- [x] Получить `IXSCAN`
- [x] Разобраться с ESR: Equality → Sort → Range
- [x] Сравнить performance до и после индекса

### 7. Aggregation
- [x] Количество событий по типам
- [x] Количество сообщений по channel
- [x] Количество платежей по plan
- [x] Использовать `$match`, `$group`, `$sort`, `$project`
- [x] Попробовать `$lookup` между `events` и `users`

### 8. Atomicity и transactions
- [x] Добавить `events_count` пользователю
- [x] Воспроизвести race condition через concurrent goroutines
- [x] Исправить через `$inc`
- [x] Попробовать MongoDB transaction
- [x] Разобраться, когда transaction действительно нужна

### 9. Дополнительные MongoDB-фичи
- [x] Настроить context и timeouts в Go
- [x] Разобраться с connection pool
- [x] Поднять Replica Set локально
