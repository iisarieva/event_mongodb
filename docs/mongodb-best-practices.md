# MongoDB Best Practices

---

## 1. Data modeling: моделировать под access patterns

В MongoDB schema стоит проектировать не как набор нормализованных SQL-таблиц, а исходя из того, **как данные будут читаться и изменяться**.

В этом проекте используются две коллекции:

- `users`
- `events`

Event хранит:

```text
user_id
user_snapshot
event_type
timestamp
properties
```

### Reference и embedded data

`user_id` используется как reference на текущего пользователя:

```text
event.user_id → users._id
```

При необходимости текущие данные пользователя можно получить через `$lookup`.

Одновременно event хранит `user_snapshot` — часть данных пользователя на момент создания события.

Пример модели: [`internal/event/model.go`](../internal/event/model.go).

Это позволяет различать:

- **current state** — данные пользователя сейчас;
- **historical state** — данные пользователя на момент события.

Такой подход является намеренной denormalization и полезен для исторической аналитики.

### Когда embed, а когда reference

**Embed** подходит, если:

- данные логически принадлежат одному document;
- они часто читаются вместе;
- важна single-document atomicity;
- embedded data не растёт бесконтрольно.

**Reference** подходит, если:

- entity существует независимо;
- данные часто обновляются отдельно;
- один объект используется во многих documents;
- требуется получать current state.

---

## 2. Flexible schema не означает отсутствие schema

MongoDB позволяет documents внутри одной collection иметь разную структуру, но application всё равно должна поддерживать понятный контракт данных.

В `events` поле `properties` зависит от `event_type`.

Например:

```text
user_registered
→ country, source

message_sent
→ chat_id, channel, length

payment_completed
→ amount, currency, plan
```

Application-side модель находится в [`internal/event/model.go`](../internal/event/model.go).

Для критичных ограничений можно дополнительно использовать MongoDB schema validation. Validator лучше хранить в version-controlled migrations, а не настраивать вручную в production.

Пример validation migration: [`internal/migrate/events_validation.go`](../internal/migrate/events_validation.go).

---

## 3. Queries: возвращать только нужные данные

При построении query стоит:

- использовать конкретные filters;
- ограничивать объём результата через `limit`;
- использовать `projection`, если полный document не нужен;
- избегать чтения всей collection без необходимости.

Пример списка events с filters, sort, limit и projection находится в:

[`internal/event/repository.go`](../internal/event/repository.go).

Для nested fields используется dot notation:

```javascript
{
    "properties.channel": "web"
}
```

---

## 4. Indexes проектируются под реальные query patterns

Index стоит добавлять под конкретный часто используемый query, а не создавать indexes «на всякий случай».

Для анализа использовать:

```javascript
.explain("executionStats")
```

Ключевые показатели:

- `COLLSCAN` — чтение collection без подходящего index;
- `IXSCAN` — чтение через index;
- `totalDocsExamined` — сколько documents реально прочитано;
- `totalKeysExamined` — сколько index entries просмотрено;
- `nReturned` — сколько documents вернул query;
- отдельный `SORT` — признак того, что порядок результата не обеспечивается index.

### Практический пример из проекта

Запрос:

```javascript
{
    user_id: "...",
    event_type: "message_sent"
}
```

с:

```javascript
.sort({ timestamp: -1 })
```

без custom index:

```text
200000 docs examined
COLLSCAN
```

с single-field index:

```javascript
{ user_id: 1 }
```

результат:

```text
20 docs examined
IXSCAN
но остались FETCH/filter и SORT
```

с compound index:

```javascript
{
    user_id: 1,
    event_type: 1,
    timestamp: -1
}
```

результат:

```text
7 docs examined
7 returned
IXSCAN
отдельный SORT исчез
```

Подробный experiment: [`mongodb-practice.md`](./mongodb-practice.md).

---

## 5. Compound indexes и ESR

Полезный guideline для порядка полей в compound index:

```text
Equality → Sort → Range
```

Пример:

```javascript
{
    user_id: "u00001",                 // Equality
    event_type: "message_sent",        // Equality
    timestamp: { $gte: someDate }      // Range
}
```

и:

```javascript
.sort({ timestamp: -1 })               // Sort
```

Для такого access pattern подходит:

```javascript
{
    user_id: 1,
    event_type: 1,
    timestamp: -1
}
```

ESR — guideline, а не абсолютное правило. Итоговый index всегда нужно проверять через `explain("executionStats")`.

Также следует помнить, что каждый дополнительный index:

- занимает disk/RAM;
- замедляет writes;
- требует maintenance.

Поэтому indexes должны иметь понятную пользу.

---

## 6. Arrays и multikey indexes

Если indexed field содержит array, MongoDB автоматически использует multikey index.

Пример из `users`:

```text
tags: ["mobile", "vip"]
```

Для array queries полезны:

- `$in` — подходит хотя бы одно значение;
- `$all` — присутствуют все указанные значения;
- `$elemMatch` — несколько условий должны выполняться для одного элемента array.

Для array из простых strings `$elemMatch` часто не нужен. Он особенно полезен для arrays из embedded documents.

---

## 7. Aggregation Pipeline

Aggregation удобно использовать для аналитики и преобразования данных.

Часто используемые stages:

```text
$match    → фильтрация
$group    → группировка
$sort     → сортировка
$project  → формирование результата
$lookup   → получение связанных documents
$unwind   → разворачивание array
```

### Рекомендуемый подход

Если возможно, уменьшать объём данных как можно раньше:

```text
$match
↓
$group
↓
$sort / $project
```

Это уменьшает количество documents, которые должны обработать последующие stages.

Примеры aggregation по `event_type`, `channel`, `plan` и `$lookup` между `events` и `users` находятся в:

[`mongodb-practice.md`](./mongodb-practice.md).

---

## 8. Single-document operations должны использовать atomic operators

MongoDB гарантирует atomicity на уровне одного document.

Для concurrent counter нельзя делать:

```text
FindOne
↓
count + 1 в Go
↓
$set
```

При concurrency это может привести к lost update.

Вместо этого использовать atomic operator:

```javascript
{
    $inc: {
        events_count: 1
    }
}
```

Пример реализации:

[`internal/user/repository.go`](../internal/user/repository.go).


---

## 9. Transactions использовать только когда нужна multi-document atomicity

Transaction нужна, когда несколько DB operations должны выполняться по принципу:

```text
all or nothing
```

Пример из проекта:

```text
1. Insert event
2. Increment user.events_count
```

Если business rule требует, чтобы эти изменения всегда происходили вместе, операции объединяются в transaction.

Application-level orchestration находится в:

[`internal/event/service.go`](../internal/event/service.go).

Repositories при этом остаются ответственными за отдельные DB operations.

Важно:

- transaction не заменяет `$inc` и другие atomic operators;
- операции внутри одной transaction выполняются последовательно;
- не стоит использовать transaction там, где достаточно одной atomic update;
- если eventual consistency допустима, transaction может быть лишней.

---

## 10. MongoDB Go Driver: один `mongo.Client` на application

Не нужно создавать новый `mongo.Client` на каждый HTTP request.

Рекомендуемая схема:

```text
application startup
↓
one mongo.Client
↓
repositories / services
↓
all requests reuse the same client
```

`mongo.Client` управляет connection pool и рассчитан на concurrent use.

Connection pool не следует тюнинговать без реальной нагрузки и метрик.

Полезные параметры существуют (`maxPoolSize`, `minPoolSize`, idle timeout), но default values обычно являются нормальной отправной точкой.

---

## 11. Context и timeouts

MongoDB operations должны получать `context.Context`.

В HTTP flow лучше передавать request context:

```go
ctx := c.Request().Context()
```

и дальше:

```text
Handler
↓
Service
↓
Repository
↓
MongoDB
```

Если request отменён, связанная DB operation также получает signal cancellation.

Для операций, которым нужен явный deadline, можно использовать:

```go
ctx, cancel := context.WithTimeout(
    c.Request().Context(),
    3*time.Second,
)
defer cancel()
```

`context.Background()` лучше оставлять для startup/CLI задач: migrations, seed, локальные utility commands.

---

## 12. Migrations должны хранить database state в version control

Даже при flexible schema migrations полезны для:

- indexes;
- schema validators;
- backfills;
- field migrations;
- TTL indexes;
- других изменений database state.

Пример migration runner:

[`internal/migrate/migrate.go`](../internal/migrate/migrate.go).

---