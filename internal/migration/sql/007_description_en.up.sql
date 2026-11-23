-- Краткое описание команды на английском (до 100 символов) нужно для поиска карточек по изображению
-- в нативном приложении: модель SigLIP сравнивает изображение только с английским текстом.
-- Описания 50–100 символов называют команду и её схему, как на изображении карточки.
ALTER TABLE riscv_commands
    ADD COLUMN description_en VARCHAR(100) NOT NULL DEFAULT '';

UPDATE riscv_commands SET description_en = 'Diagram of add x3, x1, x2: two registers are summed with a plus block into x3.' WHERE id = 1;
UPDATE riscv_commands SET description_en = 'Diagram of sub x3, x1, x2: register x2 is subtracted from x1 and stored in x3.' WHERE id = 4;
UPDATE riscv_commands SET description_en = 'Diagram of addi x2, x1, 3: register x1 plus the immediate value 3 stored in x2.' WHERE id = 9;
UPDATE riscv_commands SET description_en = 'Diagram: is x1 equal to x2, True goes to label, False goes to PC + 4, beq x1, x2, label.' WHERE id = 15;
UPDATE riscv_commands SET description_en = 'Diagram: imm 0x12345 shifted left by 12 bits becomes 0x12345000 in x5, lui x5, 0x12345.' WHERE id = 23;
UPDATE riscv_commands SET description_en = 'Diagram of jal x1, label: jump to label and save the return address in x1.' WHERE id = 42;
