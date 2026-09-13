# ER-диаграмма ЛР2

Диаграмма соответствует миграциям из `internal/migration/sql`. Готовый проект для открытия и дальнейшего оформления в StarUML находится в файле [`ER.mdj`](ER.mdj); ниже сохранено независимое текстовое представление схемы.

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
        timestamptz created_at
    }

    RISC_V_COMMANDS {
        bigint id PK
        varchar mnemonic "16 символов"
        varchar assembly_example "128 символов"
        varchar description "1000 символов"
        varchar format "R I S B U J"
        smallint operand_count "от 0 до 3"
        smallint length_bits "16 32 48 64"
        varchar image_key "255 символов"
        varchar video_key "255 символов"
        varchar status "draft published deleted"
        timestamptz created_at
        timestamptz published_at "может отсутствовать"
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
- `image_key` и `video_key` содержат только ключи объектов MinIO.
