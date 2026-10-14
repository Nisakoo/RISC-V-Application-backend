# ER-диаграмма

Диаграмма соответствует миграциям из `internal/migration/sql`. Редактируемая версия доступна в формате [draw.io](ER.drawio) и свёрстана под лист A4 в книжной ориентации: связи проведены от первичных ключей к соответствующим внешним ключам без пересечений. Ниже сохранено независимое текстовое представление схемы.

```mermaid
erDiagram
    direction LR
    USERS ||--o{ RISC_V_COMMANDS : creates
    USERS ||--o{ RISC_V_COMMAND_LIKES : sets
    RISC_V_COMMANDS ||--o{ RISC_V_COMMAND_LIKES : receives

    USERS {
        bigint id PK
        varchar login UK "64 символа"
        varchar password_hash "255 символов"
        varchar role "user или moderator"
    }

    RISC_V_COMMANDS {
        bigint id PK
        varchar mnemonic "16 символов"
        varchar description "до 1000 символов; N"
        smallint operand_count "от 0 до 3; N"
        smallint length_bits "16 32 48 64; N"
        varchar image_key "до 255 символов"
        varchar video_key "до 255 символов"
        varchar status "draft published deleted"
        timestamptz created_at
        timestamptz published_at "N"
        bigint creator_id FK
    }

    RISC_V_COMMAND_LIKES {
        bigint id PK
        bigint user_id FK
        bigint riscv_command_id FK
    }
```

## Ограничения схемы

- `users.login` — уникальное поле;
- пара `(user_id, riscv_command_id)` — уникальная;
- для каждого `creator_id` допускается не более одной строки `riscv_commands` со статусом `draft`;
- `riscv_commands.status` принимает только `draft`, `published`, `deleted`;
- все внешние ключи используют ограничение удаления `RESTRICT`, каскадное удаление запрещено;
- `published_at` обязателен для опубликованной команды;
- пометка `N` у полей команды означает, что они допускают `NULL` для черновика;
- `operand_count` и `length_bits` — два предметных поля команды RISC-V;
- `image_key` и `video_key` (url изображения и видео) обязательны: `NOT NULL` и непустые значения. Для черновика, созданного без файлов, сохраняются ключи заведомо отсутствующих объектов `images/missing/…` и `video/missing/…`;
- для пустых ключей и недоступных объектов SSR-сервер подставляет локальные фото и видео по умолчанию.
