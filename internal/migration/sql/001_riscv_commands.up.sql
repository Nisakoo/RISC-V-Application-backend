CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    login VARCHAR(64) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(16) NOT NULL CHECK (role IN ('user', 'moderator')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE riscv_commands (
    id BIGSERIAL PRIMARY KEY,
    mnemonic VARCHAR(16) NOT NULL,
    assembly_example VARCHAR(128) NOT NULL DEFAULT '',
    description VARCHAR(1000) NOT NULL DEFAULT '',
    format VARCHAR(1) NOT NULL DEFAULT '' CHECK (format IN ('', 'R', 'I', 'S', 'B', 'U', 'J')),
    operand_count SMALLINT NOT NULL DEFAULT 0 CHECK (operand_count BETWEEN 0 AND 3),
    length_bits SMALLINT NOT NULL DEFAULT 32 CHECK (length_bits IN (16, 32, 48, 64)),
    image_key VARCHAR(255) NOT NULL,
    video_key VARCHAR(255) NOT NULL,
    status VARCHAR(16) NOT NULL CHECK (status IN ('draft', 'published', 'deleted')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    published_at TIMESTAMPTZ,
    creator_id BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    CONSTRAINT riscv_commands_publication_fields CHECK (
        status = 'draft'
        OR (assembly_example <> '' AND description <> '' AND format <> '' AND published_at IS NOT NULL)
    )
);

CREATE UNIQUE INDEX riscv_commands_one_draft_per_creator
    ON riscv_commands (creator_id)
    WHERE status = 'draft';

CREATE INDEX riscv_commands_feed_order
    ON riscv_commands (status, operand_count, id);

CREATE TABLE riscv_command_likes (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    riscv_command_id BIGINT NOT NULL REFERENCES riscv_commands(id) ON DELETE RESTRICT,
    CONSTRAINT riscv_command_likes_user_command_unique UNIQUE (user_id, riscv_command_id)
);

INSERT INTO users (id, login, password_hash, role) VALUES
    (1, 'riscv_moderator', '$2a$10$lab2AuthenticationIsImplementedLater000000000000000000', 'moderator'),
    (2, 'pipeline_author', '$2a$10$lab2AuthenticationIsImplementedLater000000000000000000', 'user'),
    (3, 'rv32_student', '$2a$10$lab2AuthenticationIsImplementedLater000000000000000000', 'user'),
    (4, 'compiler_engineer', '$2a$10$lab2AuthenticationIsImplementedLater000000000000000000', 'user'),
    (5, 'embedded_builder', '$2a$10$lab2AuthenticationIsImplementedLater000000000000000000', 'user'),
    (6, 'isa_researcher', '$2a$10$lab2AuthenticationIsImplementedLater000000000000000000', 'user');

SELECT setval(pg_get_serial_sequence('users', 'id'), (SELECT MAX(id) FROM users));

INSERT INTO riscv_commands (
    id, mnemonic, assembly_example, description, format, operand_count, length_bits,
    image_key, video_key, status, creator_id, published_at
) VALUES
    (1, 'ADD', 'add x3, x1, x2', 'Складывает значения исходных регистров rs1 и rs2, затем сохраняет сумму в целевой регистр rd.', 'R', 3, 32, 'images/ADD.png', 'video/ADD.mp4', 'published', 1, CURRENT_TIMESTAMP),
    (4, 'SUB', 'sub x3, x1, x2', 'Вычитает значение регистра rs2 из регистра rs1 и записывает полученную разность в целевой регистр rd.', 'R', 3, 32, 'images/SUB.png', 'video/SUB.mp4', 'published', 1, CURRENT_TIMESTAMP),
    (9, 'ADDI', 'addi x2, x1, 3', 'Складывает значение регистра rs1 со знаково расширенным immediate и сохраняет результат в rd.', 'I', 3, 32, 'images/ADDI.png', 'video/ADDI.mp4', 'published', 1, CURRENT_TIMESTAMP),
    (15, 'BEQ', 'beq x1, x2, label', 'Сравнивает значения rs1 и rs2 и при равенстве выполняет переход по заданному смещению.', 'B', 3, 32, 'images/BEQ.png', 'video/BEQ.mp4', 'published', 1, CURRENT_TIMESTAMP),
    (23, 'LUI', 'lui x5, 0x12345', 'Помещает двадцатибитное immediate в старшие разряды целевого регистра rd.', 'U', 2, 32, 'images/LUI.png', 'video/LUI.mp4', 'published', 1, CURRENT_TIMESTAMP),
    (42, 'JAL', 'jal x1, label', 'Выполняет безусловный переход и сохраняет адрес следующей команды в регистре rd.', 'J', 2, 32, 'images/JAL.png', 'video/JAL.mp4', 'published', 1, CURRENT_TIMESTAMP),
    (77, 'AND', '', '', '', 0, 32, 'images/default.svg', 'video/ADD.mp4', 'draft', 2, NULL),
    (88, 'OR', 'or x3, x1, x2', 'Выполняет побитовое логическое ИЛИ над значениями двух исходных регистров.', 'R', 3, 32, 'images/default.svg', 'video/ADD.mp4', 'deleted', 1, CURRENT_TIMESTAMP);

SELECT setval(pg_get_serial_sequence('riscv_commands', 'id'), (SELECT MAX(id) FROM riscv_commands));

INSERT INTO riscv_command_likes (user_id, riscv_command_id) VALUES
    (2, 1), (3, 1), (4, 1), (5, 1),
    (2, 4), (3, 4), (6, 4),
    (2, 9), (4, 9), (5, 9),
    (3, 15), (6, 15),
    (2, 23), (5, 23),
    (4, 42), (6, 42);
