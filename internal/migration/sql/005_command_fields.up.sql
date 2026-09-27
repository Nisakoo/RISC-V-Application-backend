ALTER TABLE riscv_commands
    DROP CONSTRAINT riscv_commands_publication_fields;

ALTER TABLE riscv_commands
    DROP COLUMN format,
    ALTER COLUMN description DROP NOT NULL,
    ALTER COLUMN description DROP DEFAULT,
    ALTER COLUMN operand_count DROP NOT NULL,
    ALTER COLUMN operand_count DROP DEFAULT,
    ALTER COLUMN length_bits DROP NOT NULL,
    ALTER COLUMN length_bits DROP DEFAULT,
    ALTER COLUMN image_key DROP NOT NULL,
    ALTER COLUMN image_key DROP DEFAULT,
    ALTER COLUMN video_key DROP NOT NULL,
    ALTER COLUMN video_key DROP DEFAULT;

UPDATE riscv_commands
SET description = NULL,
    operand_count = NULL,
    length_bits = NULL,
    image_key = NULL,
    video_key = NULL
WHERE status = 'draft';

ALTER TABLE riscv_commands
    ADD CONSTRAINT riscv_commands_publication_fields CHECK (
        status IN ('draft', 'deleted')
        OR (
            description IS NOT NULL AND description <> ''
            AND operand_count IS NOT NULL
            AND length_bits IS NOT NULL
            AND published_at IS NOT NULL
        )
    );

ALTER TABLE users DROP COLUMN created_at;
