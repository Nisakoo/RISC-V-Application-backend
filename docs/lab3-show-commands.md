## 2

```bash
curl -i 'http://localhost:8080/api/riscv_commands?min_operands=2&max_operands=3'
```

## 3

```bash
curl -sS -X POST \
  -F 'mnemonic=XORI' \
  -F 'image=@backend/assets/images/ADD.png' \
  -F 'video=@backend/assets/video/ADD.mp4' \
  'http://localhost:8080/api/riscv_commands' \
  | tee /tmp/riscv-lab3-created.json | jq .
RISC_V_COMMAND_ID=$(jq -r '.id' /tmp/riscv-lab3-created.json)
```

## 4

```bash
curl -i 'http://localhost:8080/api/riscv_commands/draft'
```

## 5

```bash
curl -i -X PUT \
  -H 'Content-Type: application/json' \
  -d '{"description":"Выполняет побитовое исключающее ИЛИ с immediate и записывает результат в rd.","operand_count":3,"length_bits":32}' \
  "http://localhost:8080/api/riscv_commands/$RISC_V_COMMAND_ID/publish"
```

## 6

```bash
curl -i 'http://localhost:8080/api/riscv_commands/feed'
```

## 7

```bash
curl -i "http://localhost:8080/api/riscv_commands/feed?id=$RISC_V_COMMAND_ID"
```

## 8

```bash
curl -i -X POST \
  -H 'Content-Type: application/json' \
  -d '{"like":1}' \
  "http://localhost:8080/api/riscv_commands/$RISC_V_COMMAND_ID/like"
```

## 9

```bash
curl -i -X DELETE "http://localhost:8080/api/riscv_commands/$RISC_V_COMMAND_ID"
```

## 10

```bash
RISC_V_REGISTRATION_PASSWORD=$(openssl rand -base64 18)
curl -i -X POST \
  -H 'Content-Type: application/json' \
  -d "$(jq -nc --arg login "riscv_demo_$(date +%s)" --arg password "$RISC_V_REGISTRATION_PASSWORD" '{login:$login,password:$password}')" \
  'http://localhost:8080/api/users'
```

## 11

```sql
SELECT id, mnemonic, status, image_key, video_key, created_at, published_at
FROM riscv_commands
ORDER BY id DESC
LIMIT 10;
```

## 12

```sql
SELECT id, user_id, riscv_command_id
FROM riscv_command_likes
ORDER BY id DESC
LIMIT 10;
```

## 13

```sql
SELECT id, login, role
FROM users
ORDER BY id DESC
LIMIT 10;
```
