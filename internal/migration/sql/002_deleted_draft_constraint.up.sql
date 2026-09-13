ALTER TABLE riscv_commands
    DROP CONSTRAINT riscv_commands_publication_fields;

ALTER TABLE riscv_commands
    ADD CONSTRAINT riscv_commands_publication_fields CHECK (
        status IN ('draft', 'deleted')
        OR (assembly_example <> '' AND description <> '' AND format <> '' AND published_at IS NOT NULL)
    );
