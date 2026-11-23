// Команда seed наполняет каталог большим количеством команд RISC-V для демонстрации
// пагинации, индексов и ленты в Redis. Данные создаются одним SQL-запросом generate_series.
package main

import (
	"context"
	"flag"
	"log"

	"gorm.io/gorm/logger"

	"riscv-lab/internal/config"
	"riscv-lab/internal/repository"
)

// seedImagePrefix помечает сгенерированные команды: такие ключи отсутствуют в MinIO,
// поэтому для них показываются изображение и видео по умолчанию, а команды удаляются флагом -clean
const seedImagePrefix = "images/missing/seed-"

func main() {
	count := flag.Int("count", 100000, "количество создаваемых команд")
	clean := flag.Bool("clean", false, "удалить ранее созданные команды и выйти")
	flag.Parse()

	applicationConfig, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	database, err := repository.OpenPostgres(applicationConfig.DatabaseDSN)
	if err != nil {
		log.Fatal(err)
	}
	// Журнал gorm выводил бы весь SQL-запрос сида как медленный
	database.Logger = logger.Default.LogMode(logger.Silent)
	sqlDatabase, err := database.DB()
	if err != nil {
		log.Fatalf("получение подключения PostgreSQL: %v", err)
	}
	defer sqlDatabase.Close()
	ctx := context.Background()

	if *clean {
		result := database.WithContext(ctx).Exec(
			"DELETE FROM riscv_commands WHERE image_key LIKE ?", seedImagePrefix+"%",
		)
		if result.Error != nil {
			log.Fatalf("удаление созданных команд: %v", result.Error)
		}
		log.Printf("удалено команд: %d", result.RowsAffected)
		return
	}
	if *count <= 0 {
		log.Fatal("count должен быть положительным")
	}

	// Распределение: 4% команд без операндов, 18% с одним, 28% с двумя, 50% с тремя; 3% удалено логически.
	// Создатели выбираются среди существующих пользователей, лайки не создаются.
	result := database.WithContext(ctx).Exec(`
		WITH creators AS (SELECT array_agg(id) AS ids FROM users),
		mnemonics AS (
			SELECT ARRAY['ADD','SUB','AND','OR','XOR','SLL','SRL','SRA','SLT','SLTU','ADDI','ANDI','ORI','XORI',
				'LW','LH','LB','SW','SH','SB','BEQ','BNE','BLT','BGE','JAL','JALR','LUI','AUIPC','MUL','DIV','REM',
				'FENCE','ECALL','EBREAK'] AS names
		),
		series AS (
			SELECT number, random() AS operands, random() AS removed, random() AS mnemonic, random() AS creator, random() AS age
			FROM generate_series(1, ?) AS number
		)
		INSERT INTO riscv_commands (
			mnemonic, description, operand_count, length_bits,
			image_key, video_key, status, creator_id, created_at, published_at
		)
		SELECT
			mnemonics.names[1 + floor(series.mnemonic * cardinality(mnemonics.names))::int],
			'Команда набора ' || series.number || ': демонстрационное описание для проверки пагинации и индексов.',
			CASE WHEN series.operands < 0.04 THEN 0 WHEN series.operands < 0.22 THEN 1 WHEN series.operands < 0.50 THEN 2 ELSE 3 END,
			32,
			? || series.number || '.png',
			'video/missing/seed-' || series.number || '.mp4',
			CASE WHEN series.removed < 0.03 THEN 'deleted' ELSE 'published' END,
			creators.ids[1 + floor(series.creator * cardinality(creators.ids))::int],
			CURRENT_TIMESTAMP - series.age * INTERVAL '365 days',
			CURRENT_TIMESTAMP
		FROM series CROSS JOIN creators CROSS JOIN mnemonics
	`, *count, seedImagePrefix)
	if result.Error != nil {
		log.Fatalf("создание команд: %v", result.Error)
	}
	log.Printf("создано команд: %d", result.RowsAffected)
}
