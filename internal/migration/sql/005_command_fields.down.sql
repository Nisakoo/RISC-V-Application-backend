ALTER TABLE users
    ADD COLUMN created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP;

ALTER TABLE riscv_commands
    DROP CONSTRAINT riscv_commands_publication_fields;

UPDATE riscv_commands
SET description = COALESCE(description, ''),
    operand_count = COALESCE(operand_count, 0),
    length_bits = COALESCE(length_bits, 32),
    image_key = COALESCE(image_key, ''),
    video_key = COALESCE(video_key, '');

ALTER TABLE riscv_commands
    ADD COLUMN format VARCHAR(1) NOT NULL DEFAULT '' CHECK (format IN ('', 'R', 'I', 'S', 'B', 'U', 'J')),
    ALTER COLUMN description SET DEFAULT '',
    ALTER COLUMN description SET NOT NULL,
    ALTER COLUMN operand_count SET DEFAULT 0,
    ALTER COLUMN operand_count SET NOT NULL,
    ALTER COLUMN length_bits SET DEFAULT 32,
    ALTER COLUMN length_bits SET NOT NULL,
    ALTER COLUMN image_key SET DEFAULT '',
    ALTER COLUMN image_key SET NOT NULL,
    ALTER COLUMN video_key SET DEFAULT '',
    ALTER COLUMN video_key SET NOT NULL;

UPDATE riscv_commands
SET format = CASE
    WHEN mnemonic IN ('ADD', 'SUB', 'AND', 'OR') THEN 'R'
    WHEN mnemonic IN ('BEQ', 'BNE') THEN 'B'
    WHEN mnemonic = 'LUI' THEN 'U'
    WHEN mnemonic = 'JAL' THEN 'J'
    WHEN mnemonic IN ('SW', 'SH', 'SB') THEN 'S'
    ELSE 'I'
END
WHERE status = 'published';

ALTER TABLE riscv_commands
    ADD CONSTRAINT riscv_commands_publication_fields CHECK (
        status IN ('draft', 'deleted')
        OR (description <> '' AND format <> '' AND published_at IS NOT NULL)
    );
