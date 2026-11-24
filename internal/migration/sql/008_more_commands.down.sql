DELETE FROM riscv_command_likes WHERE riscv_command_id IN (
    SELECT id FROM riscv_commands WHERE image_key LIKE 'images/missing/%' AND image_key = 'images/missing/' || mnemonic || '.png' AND mnemonic IN ('AND','OR','XOR','SLL','SRL','SRA','SLT','SLTU','ANDI','ORI','SLLI','LW','SW','BNE','BLT','JALR','AUIPC')
);
DELETE FROM riscv_commands
WHERE image_key LIKE 'images/missing/%' AND image_key = 'images/missing/' || mnemonic || '.png' AND mnemonic IN ('AND','OR','XOR','SLL','SRL','SRA','SLT','SLTU','ANDI','ORI','SLLI','LW','SW','BNE','BLT','JALR','AUIPC');
UPDATE riscv_commands SET description_en = ''
WHERE (mnemonic = 'SLTI' AND description_en LIKE 'Instruction slti x2, x1, 5:%')
   OR (mnemonic = 'XORI' AND description_en LIKE 'Instruction xori x2, x1, 15:%');
