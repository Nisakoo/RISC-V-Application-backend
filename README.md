# Лабораторная работа №4

**Цель работы**: завершение бэкенда для `SPA`.

**Задание**: добавление авторизации и `swagger` в веб-сервис, подготовка ТЗ.

Реализовать методы бэкенда для `аутентификации` и `регистрации`. Авторизация для `Golang` — через `JWT` и `blacklist`. Автозаполнение пользователя в таблице команд RISC-V при создании новой. Добавить описание методов для `swagger`.

Без авторизации в `Swagger` должно быть доступно только чтение-получение данных через API, с авторизацией — методы `создателя`.

**Порядок показа**: выполнить метод аутентификации через `swagger` в режиме инкогнито, получить список команд RISC-V в `swagger`; `JWT` из ответа аутентификации использовать в заголовке `Authorization` остальных запросов через `insomnia`/`postman`. Получение всех команд RISC-V работает независимо от токена. PUT публикации команды для гостя возвращает 401/403, для авторизованного пользователя выполняется успешно и обновляет поля. Показать содержимое `Redis` через `cmd`.

## Авторизация

Метод `POST /api/users/authentication` проверяет логин и bcrypt-хеш пароля и возвращает JWT доступа (HS256) со сроком действия `JWT_TTL`. В токене хранятся идентификатор пользователя (`sub`), `login`, `role` и уникальный идентификатор токена (`jti`). Клиент передаёт токен заголовком `Authorization: Bearer <token>`.

Проверку выполняет middleware:

- методы чтения (список и лента) доступны всем; при действительном токене в ответе заполняются признаки `is_creator` и `is_liked`, иначе запрос выполняется от имени гостя;
- методы создателя требуют действительный токен: без него, с неверной подписью, с истёкшим или отозванным токеном возвращается `401`, при недопустимой роли — `403`;
- создатель черновика, владелец удаляемой команды и автор лайка определяются только по JWT.

`POST /api/users/logout` помещает `jti` токена в `blacklist` Redis до истечения срока действия токена, после чего токен не принимается.

Документация Swagger доступна по адресу `http://localhost:8080/swagger/index.html`.

## HTTP-методы

| Метод | URL | Доступ | Описание |
| --- | --- | --- | --- |
| `GET` | `/api/riscv_commands` | Гость, пользователь | Список опубликованных команд с фильтрацией по `min_operands` и `max_operands` (0–3); у каждой команды признак `is_creator` (0/1): создатель совпадает с пользователем из JWT. |
| `GET` | `/api/riscv_commands/feed` | Гость, пользователь | Лента опубликованных команд; необязательный `id` (без него — первая команда), с `id` и `next=true` — следующая команда после указанной. Ответ содержит признак `is_liked` (0/1) и `next_id`. |
| `GET` | `/api/riscv_commands/draft` | Пользователь | Черновик пользователя из JWT без передачи его ID. |
| `POST` | `/api/riscv_commands` | Пользователь | Создание черновика из `mnemonic` и файлов `image`, `video` в `multipart/form-data`; файлы сохраняются в MinIO, создатель берётся из JWT. |
| `PUT` | `/api/riscv_commands/{id}/publish` | Пользователь | Публикация своего черновика с полями `description`, `operand_count`, `length_bits`. |
| `DELETE` | `/api/riscv_commands/{id}` | Пользователь | Логическое удаление своей команды, `204` без тела. |
| `POST` | `/api/riscv_commands/{id}/like` | Пользователь | Установка (`like: 1`) или отмена (`like: 0`) лайка опубликованной команды; ответ содержит `like` и `like_count`. |
| `POST` | `/api/users` | Гость | Регистрация пользователя с ролью `user` по `login` и `password`. |
| `POST` | `/api/users/authentication` | Гость | Аутентификация: `access_token`, `token_type`, `expires_in` (секунды), `login`, `role`. |
| `POST` | `/api/users/logout` | Пользователь | Деавторизация: токен попадает в `blacklist` Redis, `204` без тела. |

«Пользователь» — роли `user` и `moderator`. Команда в ответах содержит `id`, `mnemonic`, `description`, `operand_count`, `length_bits`, `image_key`, `video_key`, `image_url`, `video_url`, `status`, `created_at`, `published_at`, `like_count` и логин создателя `creator_login`. Ошибка возвращается объектом `{"message": "…"}`.

## Таблицы PostgreSQL

- `users`: `id` (`BIGINT`, первичный ключ), `login` (`VARCHAR(64)`, уникальный), `password_hash` (`VARCHAR(255)`), `role` (`VARCHAR(16)`, `user` или `moderator`).
- `riscv_commands`: `id` (`BIGINT`, первичный ключ), `mnemonic` (`VARCHAR(16)`), `description` (`VARCHAR(1000)`), `operand_count` (`SMALLINT`), `length_bits` (`SMALLINT`), `image_key` и `video_key` (`VARCHAR(255)`), `status` (`VARCHAR(16)`: `draft`, `published`, `deleted`), `created_at` и `published_at` (`TIMESTAMPTZ`), `creator_id` (`BIGINT`, внешний ключ на `users.id`). Для черновика `description`, `operand_count`, `length_bits`, ключи медиа и `published_at` могут быть пустыми; на одного пользователя допускается не более одного черновика.
- `riscv_command_likes`: `id` (`BIGINT`, первичный ключ), `user_id` (`BIGINT`, внешний ключ на `users.id`), `riscv_command_id` (`BIGINT`, внешний ключ на `riscv_commands.id`). Пара `user_id` и `riscv_command_id` уникальна.

## Хранилище Redis

- `riscv.jwt_blacklist.<jti>` — строка с логином владельца отозванного JWT; ключ удаляется автоматически по истечении срока действия токена.
