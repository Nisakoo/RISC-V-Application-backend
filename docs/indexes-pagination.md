# Дополнительное задание: индексы, пагинация и лента в Redis

Ветка `indexes` (backend — от `auth`, frontend — от `pwa`). Задание описано в `course-docs/README.md`
(«Индексы в БД», +4 балла) и `course-docs/tutorials/pagination_index/README.md`.

## Что реализовано

| Часть задания | Реализация |
|---|---|
| Более 100 000 команд | `make seed` (`cmd/seed`): один запрос `generate_series`, 3% команд удалены логически, 4% без операндов; `make seed-clean` удаляет сгенерированные команды |
| Пагинация на бэкенде | `GET /api/riscv_commands?page=&limit=`: `limit` по умолчанию 12, максимум 100; ответ `{total, page, limit, data}` |
| Пагинация на фронтенде | компонент `Pagination`, страница плитки `/riscv_commands/grid?min_operands=&max_operands=&page=`, состояние `page` и `total` в Redux |
| Индекс | `riscv_commands_feed_order (status, operand_count, id)` из миграции 001 покрывает фильтр плитки |
| Лента в Redis | `GET /api/riscv_commands/feed/batch`: 100 случайных неповторяющихся команд сессии с порядковыми номерами |

Первая команда ленты (`feed` без `id`) раньше загружала весь каталог; теперь это запрос `LIMIT 1`
(`GetFirstVisible`): на 100 000 команд время упало с 284 мс до 7 мс.

## Метод ленты сессии

`GET /api/riscv_commands/feed/batch?session_id=<id>&refresh=<true|false>`, доступен гостю.

- Без `session_id` создаётся сессия (32 hex-символа), ключ в Redis — `riscv.feed.<session_id>`, значение — JSON
  `{session_id, batch_number, first_position, ids}`, время жизни 30 минут.
- Повторный запрос с тем же `session_id` возвращает ту же порцию из Redis.
- `refresh=true` формирует следующую порцию; нумерация продолжается (101–200 и так далее).
- Команды выбираются `ORDER BY random() LIMIT 100` среди опубликованных, поэтому в порции они не повторяются.
  Данные команд и лайки читаются из PostgreSQL по сохранённым идентификаторам, в Redis лежит только порядок.
- Ответ: `{session_id, batch_number, first_position, items[]}`, элемент — поля команды, `position` и `is_liked`.
- Ошибки: 400 (некорректные `session_id` или `refresh`), 404 (нет опубликованных команд), 503 (Redis недоступен).

Во фронтенде порция хранится в Redux (`feedBatch`), «Следующий» идёт по ней без запросов, а после последней
команды запрашивается новая порция. Идентификатор сессии лежит в `sessionStorage`. Если backend недоступен,
гость видит прежнюю ленту с mock-данными.

## Порядок показа

1. Запустить стенд и добавить данные:

```bash
make migrate-up
make seed
make lab7-run
```

2. Открыть плитку `https://localhost:3000/riscv_commands/grid`: пагинация, фильтр по операндам и номер страницы в URL.
3. Открыть ленту: у команды виден номер `№ N`, «Следующий» не делает запросов, после 100-й команды приходит новая порция.
4. Показать порцию в Redis (пароль берётся из `backend/docker/.env`):

```bash
docker exec riscv-redis redis-cli -a "$REDIS_PASSWORD" --scan --pattern 'riscv.feed.*'
docker exec riscv-redis redis-cli -a "$REDIS_PASSWORD" get riscv.feed.<session_id>
```

5. Сравнить время запроса с индексом и без него (`psql` через `docker exec -it riscv-postgres psql -U riscv_app -d riscv_catalog`):

```sql
ANALYZE riscv_commands;
EXPLAIN ANALYZE SELECT * FROM riscv_commands
 WHERE status = 'published' AND operand_count = 0 ORDER BY id LIMIT 12 OFFSET 2000;

DROP INDEX riscv_commands_feed_order;
EXPLAIN ANALYZE SELECT * FROM riscv_commands
 WHERE status = 'published' AND operand_count = 0 ORDER BY id LIMIT 12 OFFSET 2000;

CREATE INDEX riscv_commands_feed_order ON riscv_commands (status, operand_count, id);
```

Результат на 100 000 командах: с индексом `Index Scan using riscv_commands_feed_order`, 3,4 мс; без индекса
`Index Scan using riscv_commands_pkey` с фильтрацией, 16,4 мс (в 5 раз медленнее).
