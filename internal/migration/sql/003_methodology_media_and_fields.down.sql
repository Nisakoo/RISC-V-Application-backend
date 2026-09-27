ALTER TABLE riscv_commands
    DROP CONSTRAINT riscv_commands_publication_fields;

ALTER TABLE riscv_commands
    ADD COLUMN assembly_example VARCHAR(128) NOT NULL DEFAULT '',
    ADD COLUMN length_bits SMALLINT NOT NULL DEFAULT 32 CHECK (length_bits IN (16, 32, 48, 64));

UPDATE riscv_commands
SET assembly_example = LOWER(mnemonic) || ' operands'
WHERE status = 'published';

ALTER TABLE riscv_commands
    ALTER COLUMN image_key DROP DEFAULT,
    ALTER COLUMN video_key DROP DEFAULT;

ALTER TABLE riscv_commands
    ADD CONSTRAINT riscv_commands_publication_fields CHECK (
        status IN ('draft', 'deleted')
        OR (assembly_example <> '' AND description <> '' AND format <> '' AND published_at IS NOT NULL)
    );
