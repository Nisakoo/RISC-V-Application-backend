ALTER TABLE riscv_commands
    DROP CONSTRAINT riscv_commands_media_keys_not_empty,
    ALTER COLUMN image_key DROP NOT NULL,
    ALTER COLUMN video_key DROP NOT NULL;
