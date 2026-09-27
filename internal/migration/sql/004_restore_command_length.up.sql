ALTER TABLE riscv_commands
    ADD COLUMN length_bits SMALLINT NOT NULL DEFAULT 32
    CHECK (length_bits IN (16, 32, 48, 64));
