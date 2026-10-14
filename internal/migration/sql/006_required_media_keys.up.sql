-- url изображения и видео обязательны для каждой команды RISC-V.
-- Черновикам без медиа назначаются заведомо отсутствующие в MinIO ключи:
-- для них SSR-сервер показывает фото и видео по умолчанию.
UPDATE riscv_commands
SET image_key = 'images/missing/' || id || '.png'
WHERE image_key IS NULL OR image_key = '';

UPDATE riscv_commands
SET video_key = 'video/missing/' || id || '.mp4'
WHERE video_key IS NULL OR video_key = '';

ALTER TABLE riscv_commands
    ALTER COLUMN image_key SET NOT NULL,
    ALTER COLUMN video_key SET NOT NULL,
    ADD CONSTRAINT riscv_commands_media_keys_not_empty CHECK (image_key <> '' AND video_key <> '');
