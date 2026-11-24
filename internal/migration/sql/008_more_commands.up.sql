-- Дополнительные команды RISC-V для проверки пагинации и поиска по изображению в нативном приложении.
-- У команд нет собственных схем: ключи указывают на заведомо отсутствующие объекты, поэтому показывается
-- изображение по умолчанию. Английские описания (74–99 символов) называют команду и результат её выполнения.
-- Миграция повторяема: уже существующие команды (по мнемонике и ключу изображения) не дублируются.
INSERT INTO riscv_commands (mnemonic, description, operand_count, length_bits, image_key, video_key, status, published_at, creator_id, description_en)
SELECT v.mnemonic, v.description, v.operand_count, 32, v.image_key, v.video_key, 'published', CURRENT_TIMESTAMP, 1, v.description_en
FROM (VALUES
    ('AND', 'Выполняет побитовое логическое И над значениями регистров rs1 и rs2 и записывает результат в целевой регистр rd.', 3, 'images/missing/AND.png', 'video/missing/AND.mp4', 'Instruction and x3, x1, x2: bitwise AND of registers x1 and x2 is written to x3.'),
    ('OR', 'Выполняет побитовое логическое ИЛИ над значениями регистров rs1 и rs2 и записывает результат в целевой регистр rd.', 3, 'images/missing/OR.png', 'video/missing/OR.mp4', 'Instruction or x3, x1, x2: bitwise OR of registers x1 and x2 is written to x3.'),
    ('XOR', 'Выполняет побитовое исключающее ИЛИ над значениями регистров rs1 и rs2 и записывает результат в rd.', 3, 'images/missing/XOR.png', 'video/missing/XOR.mp4', 'Instruction xor x3, x1, x2: bitwise XOR of registers x1 and x2 is written to x3.'),
    ('SLL', 'Сдвигает значение регистра rs1 влево на число бит из младших пяти разрядов rs2 и записывает результат в rd.', 3, 'images/missing/SLL.png', 'video/missing/SLL.mp4', 'Instruction sll x3, x1, x2: x1 is shifted left by the amount in x2 and stored in x3.'),
    ('SRL', 'Логически сдвигает значение регистра rs1 вправо на число бит из младших пяти разрядов rs2, результат записывается в rd.', 3, 'images/missing/SRL.png', 'video/missing/SRL.mp4', 'Instruction srl x3, x1, x2: x1 is logically shifted right by x2 bits and stored in x3.'),
    ('SRA', 'Арифметически сдвигает значение регистра rs1 вправо на число бит из rs2 с сохранением знака и записывает результат в rd.', 3, 'images/missing/SRA.png', 'video/missing/SRA.mp4', 'Instruction sra x3, x1, x2: x1 is arithmetically shifted right by x2 and stored in x3.'),
    ('SLT', 'Сравнивает значения регистров rs1 и rs2 как знаковые числа и записывает в rd 1, если rs1 меньше rs2, иначе 0.', 3, 'images/missing/SLT.png', 'video/missing/SLT.mp4', 'Instruction slt x3, x1, x2: x3 is 1 if signed x1 is less than signed x2, otherwise 0.'),
    ('SLTU', 'Сравнивает значения регистров rs1 и rs2 как беззнаковые числа и записывает в rd 1, если rs1 меньше rs2, иначе 0.', 3, 'images/missing/SLTU.png', 'video/missing/SLTU.mp4', 'Instruction sltu x3, x1, x2: x3 is 1 if unsigned x1 is less than unsigned x2, otherwise 0.'),
    ('ANDI', 'Выполняет побитовое логическое И значения регистра rs1 со знаково расширенным immediate и записывает результат в rd.', 3, 'images/missing/ANDI.png', 'video/missing/ANDI.mp4', 'Instruction andi x2, x1, 15: bitwise AND of register x1 and the immediate 15 is written to x2.'),
    ('ORI', 'Выполняет побитовое логическое ИЛИ значения регистра rs1 со знаково расширенным immediate и записывает результат в rd.', 3, 'images/missing/ORI.png', 'video/missing/ORI.mp4', 'Instruction ori x2, x1, 15: bitwise OR of register x1 and the immediate 15 is written to x2.'),
    ('SLLI', 'Сдвигает значение регистра rs1 влево на константное число бит shamt и записывает результат в целевой регистр rd.', 3, 'images/missing/SLLI.png', 'video/missing/SLLI.mp4', 'Instruction slli x2, x1, 3: register x1 is shifted left by the constant 3 and stored in x2.'),
    ('LW', 'Загружает из памяти 32-битное слово по адресу rs1 плюс смещение и записывает его в целевой регистр rd.', 3, 'images/missing/LW.png', 'video/missing/LW.mp4', 'Instruction lw x2, 8(x1): the word at memory address x1 + 8 is loaded into register x2.'),
    ('SW', 'Сохраняет 32-битное значение регистра rs2 в память по адресу rs1 плюс знаково расширенное смещение.', 3, 'images/missing/SW.png', 'video/missing/SW.mp4', 'Instruction sw x2, 8(x1): the word from register x2 is stored at memory address x1 + 8.'),
    ('BNE', 'Сравнивает значения rs1 и rs2 и при неравенстве выполняет переход по заданному смещению относительно PC.', 3, 'images/missing/BNE.png', 'video/missing/BNE.mp4', 'Instruction bne x1, x2, label: if x1 is not equal to x2, jump to label, otherwise go to PC + 4.'),
    ('BLT', 'Сравнивает значения rs1 и rs2 как знаковые числа и при rs1 меньше rs2 выполняет переход по заданному смещению.', 3, 'images/missing/BLT.png', 'video/missing/BLT.mp4', 'Instruction blt x1, x2, label: if signed x1 is less than x2, jump to label, otherwise go to PC + 4.'),
    ('JALR', 'Выполняет косвенный переход по адресу rs1 плюс смещение и сохраняет адрес следующей команды в регистре rd.', 3, 'images/missing/JALR.png', 'video/missing/JALR.mp4', 'Instruction jalr x1, 0(x5): jump to the address x5 + 0 and save the return address in x1.'),
    ('AUIPC', 'Прибавляет двадцатибитное immediate, сдвинутое на 12 бит влево, к адресу текущей команды и записывает сумму в rd.', 2, 'images/missing/AUIPC.png', 'video/missing/AUIPC.mp4', 'Instruction auipc x5, 0x1: immediate 0x1 shifted left by 12 is added to PC and stored in x5.')
) AS v (mnemonic, description, operand_count, image_key, video_key, description_en)
WHERE NOT EXISTS (
    SELECT 1 FROM riscv_commands c WHERE c.mnemonic = v.mnemonic AND c.image_key = v.image_key
);

-- Английские описания для команд, добавленных после миграции 007
UPDATE riscv_commands SET description_en = 'Instruction slti x2, x1, 5: x2 is 1 if signed x1 is less than the immediate 5, otherwise 0.'
WHERE mnemonic = 'SLTI' AND status = 'published' AND description_en = '';
UPDATE riscv_commands SET description_en = 'Instruction xori x2, x1, 15: bitwise XOR of register x1 and the immediate 15 is written to x2.'
WHERE mnemonic = 'XORI' AND status = 'published' AND description_en = '';
