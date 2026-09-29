# MongoDB Practice Notes

---

## 1. Подготовка тестовых данных

Seeder создал:

```text
10000 users
200000 events
```

Команда запуска:

```bash
go run ./cmd/seed
```

Результат:

```text
Inserted 10000 users
Inserted 200000 events
Seed completed
```

### Подключение к MongoDB в Docker

```bash
docker exec -it eventflow-mongodb mongosh
```

В этой практике использовались:

```text
MongoDB 8.0.32
Mongosh 2.11.1
```

Переход в базу:

```javascript
use eventflow
```

Проверка количества документов:

```javascript
db.users.countDocuments()
```

```text
10000
```

```javascript
db.events.countDocuments()
```

```text
200000
```

---

## 2. Базовый поиск: filter + sort + limit

Пример поиска событий конкретного пользователя, конкретного типа и только после определённой даты:

```javascript
db.events.find({
    user_id: "u00001",
    event_type: "message_sent",
    timestamp: {
        $gte: ISODate("2026-09-01T00:00:00Z")
    }
})
.sort({
    timestamp: -1
})
.limit(50)
```

Что здесь используется:

- `find()` — поиск документов.
- `user_id` и `event_type` — обычные equality filters.
- `$gte` — range condition: `timestamp >= указанная дата`.
- `.sort({timestamp: -1})` — сортировка по убыванию, новые события первыми.
- `.limit(50)` — вернуть максимум 50 документов.

Пример найденного документа:

```javascript
{
    _id: ObjectId("..."),
    event_type: "message_sent",
    user_id: "u00001",
    user_snapshot: {
        country: "DE",
        plan: "basic"
    },
    timestamp: ISODate("2026-09-20T22:48:03.825Z"),
    properties: {
        length: 1,
        chat_id: "chat_0",
        channel: "web"
    }
}
```

---

# 3. Performance и indexes

## 3.1. Какие indexes были изначально

```javascript
db.events.getIndexes()
```

Результат:

```javascript
[
    {
        v: 2,
        key: { _id: 1 },
        name: "_id_"
    }
]
```

`_id_` — автоматический index, который MongoDB создаёт для `_id`.

На поля `user_id`, `event_type` и `timestamp` custom indexes пока отсутствовали.

---

## 3.2. `explain("executionStats")` без custom index

Запрос:

```javascript
db.events
  .find({
      user_id: "u00001",
      event_type: "message_sent"
  })
  .sort({
      timestamp: -1
  })
  .limit(50)
  .explain("executionStats")
```

Ключевая часть плана:

```text
SORT
└── COLLSCAN
```

Ключевые показатели:

```text
nReturned:          7
totalKeysExamined:  0
totalDocsExamined:  200000
executionTimeMillis: 66
```

### Что означает `COLLSCAN`

`COLLSCAN` = Collection Scan.

MongoDB не имеет подходящего index и поэтому проходит по всей collection:

```text
200000 документов
↓
проверка условий
↓
7 подходящих документов
```

Это дорого на больших объёмах данных.

Также MongoDB пришлось отдельно выполнить:

```text
SORT timestamp DESC
```

потому что нужного порядка в index не было.

---

## 3.3. Single-field index

Создаём index только по `user_id`:

```javascript
db.events.createIndex({
    user_id: 1
})
```

MongoDB создала:

```text
user_id_1
```

Проверка:

```javascript
db.events.getIndexes()
```

```javascript
[
    { key: { _id: 1 }, name: "_id_" },
    { key: { user_id: 1 }, name: "user_id_1" }
]
```

Повторяем тот же `explain()`.

Ключевой execution plan:

```text
SORT
└── FETCH
    └── IXSCAN user_id_1
```

Показатели:

```text
nReturned:          7
totalKeysExamined:  20
totalDocsExamined:  20
executionTimeMillis: 1
```

### Что изменилось

Раньше:

```text
200000 documents examined
```

Теперь:

```text
20 index keys
20 documents
7 result documents
```

MongoDB использовала `IXSCAN` (Index Scan): она сразу нашла в index документы пользователя `u00001`.

Но index содержит только `user_id`, поэтому MongoDB всё ещё должна:
1. получить 20 найденных документов (`FETCH`);
2. проверить у них `event_type`;
3. оставить 7 подходящих;
4. отдельно выполнить `SORT` по `timestamp`.

То есть single-field index уже очень полезен, но запрос покрывает не полностью.

---

## 3.4. Compound index

Создаём один index сразу по нескольким полям:

```javascript
db.events.createIndex({
    user_id: 1,
    event_type: 1,
    timestamp: -1
})
```

Получили:

```text
user_id_1_event_type_1_timestamp_-1
```

Проверка:

```javascript
db.events.getIndexes()
```

```javascript
[
    { key: { _id: 1 }, name: "_id_" },
    { key: { user_id: 1 }, name: "user_id_1" },
    {
        key: {
            user_id: 1,
            event_type: 1,
            timestamp: -1
        },
        name: "user_id_1_event_type_1_timestamp_-1"
    }
]
```

Повторяем тот же запрос:

```javascript
db.events
  .find({
      user_id: "u00001",
      event_type: "message_sent"
  })
  .sort({
      timestamp: -1
  })
  .limit(50)
  .explain("executionStats")
```

Ключевой execution plan:

```text
LIMIT
└── FETCH
    └── IXSCAN user_id_1_event_type_1_timestamp_-1
```

Показатели:

```text
nReturned:          7
totalKeysExamined:  7
totalDocsExamined:  7
executionTimeMillis: 3
```

### Почему стало лучше

MongoDB теперь может сразу искать сочетание:

```text
user_id = u00001
+
event_type = message_sent
```

Поэтому вместо:

```text
20 documents → filter → 7
```

она получает:

```text
7 index keys
→
7 documents
→
7 results
```

Отдельный `SORT` тоже исчез.

Причина: поле `timestamp` уже находится в compound index в порядке:

```javascript
timestamp: -1
```

Поэтому MongoDB читает подходящую часть index сразу в нужном порядке.

### Почему `FETCH` всё ещё есть

Index содержит:

```text
user_id
event_type
timestamp
```

Но `find()` возвращает полный документ, включая:

```text
properties
user_snapshot
...
```

Эти данные нужно прочитать из самой collection, поэтому после `IXSCAN` остаётся `FETCH`.

---

# 4. ESR: Equality → Sort → Range

Практический index:

```javascript
{
    user_id: 1,
    event_type: 1,
    timestamp: -1
}
```

Для запроса:

```javascript
{
    user_id: "u00001",
    event_type: "message_sent"
}
```

и:

```javascript
.sort({
    timestamp: -1
})
```

получаем:

```text
user_id     → Equality
event_type  → Equality
timestamp   → Sort
```

То есть:

```text
E → E → S
```

---

## 4.1. Добавляем Range

Сначала проверяем диапазон дат в collection:

```javascript
db.events.aggregate([
    {
        $group: {
            _id: null,
            min: { $min: "$timestamp" },
            max: { $max: "$timestamp" }
        }
    }
])
```

Получили:

```text
min: 2026-08-28T15:48:03.825Z
max: 2026-09-27T14:48:03.825Z
```

Теперь запрос с range filter:

```javascript
db.events
  .find({
      user_id: "u00001",
      event_type: "message_sent",
      timestamp: {
          $gte: ISODate("2026-09-15T00:00:00Z")
      }
  })
  .sort({
      timestamp: -1
  })
  .limit(50)
  .explain("executionStats")
```

Результат:

```text
nReturned:          2
totalKeysExamined:  2
totalDocsExamined:  2
executionTimeMillis: 1
```

Execution plan:

```text
LIMIT
└── FETCH
    └── IXSCAN compound index
```

Отдельного `SORT` опять нет.

### `indexBounds`

Для `user_id`:

```text
["u00001", "u00001"]
```

Для `event_type`:

```text
["message_sent", "message_sent"]
```

Это точные Equality bounds.

Для `timestamp` MongoDB ограничила диапазон снизу датой `2026-09-15`.

Поскольку index имеет `timestamp: -1`, MongoDB идёт по нему от новых дат к старым и останавливается на границе.

Итого поле `timestamp` здесь выполняет две задачи:

```text
Sort + Range
```

---

## 4.2. Сравнение performance

| Вариант | Keys examined | Docs examined | Returned | Дополнительный SORT |
|---|---:|---:|---:|---|
| Без custom index | 0 | 200000 | 7 | Да |
| `{user_id: 1}` | 20 | 20 | 7 | Да |
| Compound index | 7 | 7 | 7 | Нет |
| Compound + date range | 2 | 2 | 2 | Нет |

Главный вывод:

```text
200000 docs → 20 docs → 7 docs
```

Количество просмотренных данных уменьшилось радикально.

### Важно про `executionTimeMillis`

В одном запуске получилось:

```text
COLLSCAN:       66 ms
single index:    1 ms
compound:        3 ms
range:           1 ms
```

Нельзя делать вывод, что `1 ms` обязательно лучше `3 ms`.

На локальной MongoDB такие маленькие значения сильно зависят от cache, ОС и query planning.

Для оценки качества index полезнее смотреть прежде всего на:

```text
totalDocsExamined
totalKeysExamined
nReturned
execution plan
```

---

# 5. Aggregation Pipeline

`aggregation pipeline` — последовательность stages, через которые проходят документы.

Пример:

```text
documents
↓
$match
↓
$group
↓
$sort
↓
$project
↓
result
```

SQL-аналоги:

| MongoDB | Примерный SQL-аналог |
|---|---|
| `$match` | `WHERE` |
| `$group` | `GROUP BY` |
| `$sort` | `ORDER BY` |
| `$project` | `SELECT` / aliases |
| `$lookup` | `JOIN` |
| `$unwind` | разворачивание элементов array |

---

## 5.1. Количество events по типам

```javascript
db.events.aggregate([
    {
        $group: {
            _id: "$event_type",
            count: {
                $sum: 1
            }
        }
    }
])
```

Результат:

```javascript
[
    { _id: "payment_completed", count: 66667 },
    { _id: "message_sent", count: 66667 },
    { _id: "user_registered", count: 66666 }
]
```

В `$group`:

```javascript
_id: "$event_type"
```

означает grouping key, примерно как:

```sql
GROUP BY event_type
```

А:

```javascript
count: {
    $sum: 1
}
```

аналогично:

```sql
COUNT(*)
```

---

## 5.2. `$sort` после `$group`

```javascript
db.events.aggregate([
    {
        $group: {
            _id: "$event_type",
            count: {
                $sum: 1
            }
        }
    },
    {
        $sort: {
            count: -1
        }
    }
])
```

`-1` означает descending order.

---

## 5.3. `$project`

```javascript
db.events.aggregate([
    {
        $group: {
            _id: "$event_type",
            count: {
                $sum: 1
            }
        }
    },
    {
        $sort: {
            count: -1
        }
    },
    {
        $project: {
            _id: 0,
            event_type: "$_id",
            count: 1
        }
    }
])
```

Результат:

```javascript
[
    { count: 66667, event_type: "payment_completed" },
    { count: 66667, event_type: "message_sent" },
    { count: 66666, event_type: "user_registered" }
]
```

Что делает `$project`:

```javascript
_id: 0
```

убирает `_id`.

```javascript
count: 1
```

оставляет `count`.

```javascript
event_type: "$_id"
```

создаёт новое поле `event_type` и кладёт туда значение старого `_id`.

---

## 5.4. `$match` перед `$group`

Статистика только для событий после 15 сентября:

```javascript
db.events.aggregate([
    {
        $match: {
            timestamp: {
                $gte: ISODate("2026-09-15T00:00:00Z")
            }
        }
    },
    {
        $group: {
            _id: "$event_type",
            count: {
                $sum: 1
            }
        }
    },
    {
        $sort: {
            count: -1
        }
    },
    {
        $project: {
            _id: 0,
            event_type: "$_id",
            count: 1
        }
    }
])
```

Результат:

```javascript
[
    { count: 28078, event_type: "message_sent" },
    { count: 28078, event_type: "payment_completed" },
    { count: 28077, event_type: "user_registered" }
]
```

Почему `$match` удобно ставить раньше:

```text
200000 documents
↓
$match
↓
осталась только нужная часть
↓
$group работает уже с меньшим объёмом
```

Кроме того, ранний `$match` иногда может использовать index.

---

## 5.5. Количество `message_sent` по `channel`

```javascript
db.events.aggregate([
    {
        $match: {
            event_type: "message_sent"
        }
    },
    {
        $group: {
            _id: "$properties.channel",
            count: {
                $sum: 1
            }
        }
    },
    {
        $sort: {
            count: -1
        }
    },
    {
        $project: {
            _id: 0,
            channel: "$_id",
            count: 1
        }
    }
])
```

Результат:

```javascript
[
    { count: 66667, channel: "web" }
]
```

Здесь используется доступ к nested field:

```javascript
"$properties.channel"
```

Это dot notation.

> В текущем seed все `message_sent` получили `channel: "web"`, поэтому aggregation вернула только одну группу.

---

## 5.6. Количество платежей по `plan`

```javascript
db.events.aggregate([
    {
        $match: {
            event_type: "payment_completed"
        }
    },
    {
        $group: {
            _id: "$properties.plan",
            count: {
                $sum: 1
            }
        }
    },
    {
        $sort: {
            count: -1
        }
    },
    {
        $project: {
            _id: 0,
            plan: "$_id",
            count: 1
        }
    }
])
```

Результат:

```javascript
[
    { count: 23331, plan: "pro" },
    { count: 23331, plan: "free" },
    { count: 20005, plan: "basic" }
]
```

---

# 6. `$lookup`: связь `events` и `users`

В `events` хранится:

```javascript
user_id: "u00001"
```

А в `users`:

```javascript
_id: "u00001"
```

Пример `$lookup`:

```javascript
db.events.aggregate([
    {
        $match: {
            user_id: "u00001"
        }
    },
    {
        $limit: 3
    },
    {
        $lookup: {
            from: "users",
            localField: "user_id",
            foreignField: "_id",
            as: "user"
        }
    }
])
```

Параметры:

```text
from          → collection, из которой подтягиваем данные
localField    → поле текущего event
foreignField  → поле в users
as            → имя нового поля с результатом
```

То есть:

```text
events.user_id
     ↓
users._id
```

Пример результата:

```javascript
{
    event_type: "message_sent",
    user_id: "u00001",
    user_snapshot: {
        country: "DE",
        plan: "basic"
    },

    user: [
        {
            _id: "u00001",
            name: "User 1",
            country: "DE",
            plan: "basic",
            tags: ["mobile"]
        }
    ]
}
```

### Почему `user` — array

`$lookup` возвращает найденные документы как array:

```javascript
user: [
    {...}
]
```

Даже если `_id` unique и реально найден максимум один user.

---

# 7. `$unwind`

Чтобы превратить:

```javascript
user: [
    {
        ...
    }
]
```

в:

```javascript
user: {
    ...
}
```

используется:

```javascript
{
    $unwind: "$user"
}
```

Полный пример:

```javascript
db.events.aggregate([
    {
        $match: {
            user_id: "u00001"
        }
    },
    {
        $limit: 3
    },
    {
        $lookup: {
            from: "users",
            localField: "user_id",
            foreignField: "_id",
            as: "user"
        }
    },
    {
        $unwind: "$user"
    }
])
```

После `$unwind`:

```javascript
user: {
    _id: "u00001",
    name: "User 1",
    country: "DE",
    plan: "basic",
    tags: ["mobile"]
}
```

---

# 8. Reference vs embedded snapshot

В модели event одновременно хранятся:

```javascript
user_id: "u00001"
```

и:

```javascript
user_snapshot: {
    country: "DE",
    plan: "basic"
}
```

Это намеренная denormalization.

Чтобы сравнить текущие данные user и snapshot:

```javascript
db.events.aggregate([
    {
        $match: {
            user_id: "u00001"
        }
    },
    {
        $limit: 3
    },
    {
        $lookup: {
            from: "users",
            localField: "user_id",
            foreignField: "_id",
            as: "user"
        }
    },
    {
        $unwind: "$user"
    },
    {
        $project: {
            _id: 0,
            event_type: 1,
            user_id: 1,

            snapshot_plan: "$user_snapshot.plan",
            current_plan: "$user.plan",

            snapshot_country: "$user_snapshot.country",
            current_country: "$user.country"
        }
    }
])
```

До изменения user:

```javascript
{
    snapshot_plan: "basic",
    current_plan: "basic",
    snapshot_country: "DE",
    current_country: "DE"
}
```

---

## 8.1. Меняем текущий `plan`

```javascript
db.users.updateOne(
    {
        _id: "u00001"
    },
    {
        $set: {
            plan: "pro"
        }
    }
)
```

Результат:

```javascript
{
    acknowledged: true,
    matchedCount: 1,
    modifiedCount: 1
}
```

Повторяем aggregation.

Теперь:

```javascript
{
    snapshot_plan: "basic",
    current_plan: "pro",
    snapshot_country: "DE",
    current_country: "DE"
}
```

### Главный вывод

`$lookup` показывает текущие данные из `users`:

```text
current_plan = pro
```

`user_snapshot` сохраняет состояние на момент создания event:

```text
snapshot_plan = basic
```

То есть они отвечают на разные вопросы:

```text
$lookup:
Какой plan у пользователя сейчас?

user_snapshot:
Какой plan был у пользователя в момент события?
```

Это хороший пример, зачем MongoDB-проекты иногда намеренно используют denormalization.

---

# 9. Короткая шпаргалка по execution plan

## `COLLSCAN`

`Collection Scan`: MongoDB читает документы collection напрямую. Обычно означает, что подходящего index нет.

## `IXSCAN`

`Index Scan`: MongoDB ищет данные через index.

## `FETCH`

MongoDB нашла ссылки на документы через index и теперь читает сами BSON documents из collection.

## `SORT`

MongoDB выполняет отдельную сортировку результата. Если нужный порядок уже обеспечивается index, отдельный `SORT` может исчезнуть из execution plan.

## `LIMIT`

Ограничивает максимальное количество возвращаемых документов.

### Полезные поля `explain("executionStats")`

- `nReturned` — сколько документов реально вернули.
- `totalKeysExamined` — сколько index entries просмотрено.
- `totalDocsExamined` — сколько BSON documents пришлось прочитать.
- `executionTimeMillis` — время выполнения конкретного запуска.

Для локальных быстрых запросов не стоит оценивать performance только по миллисекундам.

---

# 10. Практические выводы

1. Не создавать indexes «на всякий случай». Сначала посмотреть реальный query pattern.
2. Для анализа запроса использовать `.explain("executionStats")`.
3. Следить прежде всего за `COLLSCAN / IXSCAN`, `totalDocsExamined`, `totalKeysExamined`, `nReturned`, `SORT`.
4. Single-field index может сильно уменьшить scan, но не обязательно полностью оптимизирует query.
5. Compound index должен проектироваться под реальные filters + sort + range.
6. ESR (`Equality → Sort → Range`) — полезный guideline, но итог всегда нужно проверять через `explain()`.
7. В aggregation pipeline порядок stages важен.
8. `$match` обычно выгодно ставить как можно раньше, если это соответствует логике pipeline.
9. `$lookup` позволяет работать с references между collections.
10. Embedded snapshot и `$lookup` могут существовать одновременно: snapshot хранит историческое состояние, `$lookup` даёт текущее состояние связанной entity.

---
