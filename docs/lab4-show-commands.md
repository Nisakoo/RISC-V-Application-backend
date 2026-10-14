# Команды для показа лабораторной работы №4

Команды выполняются из корня проекта при запущенном `make lab4-run`.

## 1. Регистрация и аутентификация

```bash
RISC_V_LOGIN="rv32_author_$(date +%s)"
RISC_V_PASSWORD=$(openssl rand -base64 18)
curl -i -X POST -H 'Content-Type: application/json' \
  -d "$(jq -nc --arg login "$RISC_V_LOGIN" --arg password "$RISC_V_PASSWORD" '{login:$login,password:$password}')" \
  'http://localhost:8080/api/users'
RISC_V_TOKEN=$(curl -sS -X POST -H 'Content-Type: application/json' \
  -d "$(jq -nc --arg login "$RISC_V_LOGIN" --arg password "$RISC_V_PASSWORD" '{login:$login,password:$password}')" \
  'http://localhost:8080/api/users/authentication' | jq -r '.access_token')
```

## 2. Список команд независимо от токена

```bash
curl -i 'http://localhost:8080/api/riscv_commands?min_operands=2&max_operands=3'
curl -i -H "Authorization: Bearer $RISC_V_TOKEN" 'http://localhost:8080/api/riscv_commands?min_operands=2&max_operands=3'
```

## 3. Создание черновика

```bash
curl -sS -X POST -H "Authorization: Bearer $RISC_V_TOKEN" \
  -F 'mnemonic=XORI' \
  -F 'image=@backend/assets/images/ADD.png' \
  -F 'video=@backend/assets/video/ADD.mp4' \
  'http://localhost:8080/api/riscv_commands' | tee /tmp/riscv-lab4-created.json | jq .
RISC_V_COMMAND_ID=$(jq -r '.id' /tmp/riscv-lab4-created.json)
```

## 4. Публикация: гость и пользователь

```bash
curl -i -X PUT -H 'Content-Type: application/json' \
  -d '{"description":"Выполняет побитовое исключающее ИЛИ с immediate и записывает результат в rd.","operand_count":3,"length_bits":32}' \
  "http://localhost:8080/api/riscv_commands/$RISC_V_COMMAND_ID/publish"
curl -i -X PUT -H "Authorization: Bearer $RISC_V_TOKEN" -H 'Content-Type: application/json' \
  -d '{"description":"Выполняет побитовое исключающее ИЛИ с immediate и записывает результат в rd.","operand_count":3,"length_bits":32}' \
  "http://localhost:8080/api/riscv_commands/$RISC_V_COMMAND_ID/publish"
```

## 5. Лента и лайк

```bash
curl -i -X POST -H "Authorization: Bearer $RISC_V_TOKEN" -H 'Content-Type: application/json' \
  -d '{"like":1}' "http://localhost:8080/api/riscv_commands/$RISC_V_COMMAND_ID/like"
curl -i -H "Authorization: Bearer $RISC_V_TOKEN" "http://localhost:8080/api/riscv_commands/feed?id=$RISC_V_COMMAND_ID"
```

## 6. Деавторизация и Redis

```bash
curl -i -X POST -H "Authorization: Bearer $RISC_V_TOKEN" 'http://localhost:8080/api/users/logout'
curl -i -H "Authorization: Bearer $RISC_V_TOKEN" 'http://localhost:8080/api/riscv_commands/draft'
docker exec -it riscv-redis redis-cli KEYS 'riscv.jwt_blacklist.*'
```
