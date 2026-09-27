ALTER TABLE riscv_commands
    DROP CONSTRAINT riscv_commands_publication_fields;

ALTER TABLE riscv_commands
    DROP COLUMN assembly_example,
    DROP COLUMN length_bits;

ALTER TABLE riscv_commands
    ALTER COLUMN image_key SET DEFAULT '',
    ALTER COLUMN video_key SET DEFAULT '';

UPDATE riscv_commands
SET image_key = '', video_key = ''
WHERE status = 'draft';

ALTER TABLE riscv_commands
    ADD CONSTRAINT riscv_commands_publication_fields CHECK (
        status IN ('draft', 'deleted')
        OR (description <> '' AND format <> '' AND published_at IS NOT NULL)
    );
