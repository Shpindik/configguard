# ConfigGuard

Утилита на Go: анализирует YAML/JSON-конфиг веб-приложения и находит опасные
настройки (debug-режим, пароли в открытом виде, `0.0.0.0` без ограничений,
отключённый TLS, слабые алгоритмы, слишком широкие права доступа). Работает
как CLI, HTTP REST API и gRPC API.

## Архитектура

Проект собран по DDD: зависимости идут строго внутрь, домен ничего не знает
про способ доставки (CLI/HTTP/gRPC) и про конкретный формат файла.

```
domain          — правила и модель конфига, только stdlib, ничего не знает
  issue/         Level (LOW/MEDIUM/HIGH), Issue, сортировка по серьёзности
  config/        Config — дерево конфига (map[string]any/[]any) + Walk/FindKey
  rule/          интерфейсы Rule и FileAwareRule, Registry
  rule/builtin/  сами правила (по файлу на правило)

application     — сценарии использования, дирижирует domain+infrastructure
  scanner/       Service.ScanFile/ScanReader → Report
  dirscan/       рекурсивный обход директории, worker pool

infrastructure  — реализации интерфейсов domain/application
  parser/        JSON- и YAML-парсеры в общее дерево Config

interfaces      — точки входа, решают про stdout/exit code/HTTP/gRPC
  cli/           cobra-команды scan и serve
  httpapi/       REST API
  grpcapi/       gRPC API

api/proto/            .proto-контракт gRPC
internal/genproto/    сгенерированный код (protoc-gen-go/-grpc)
```

**Почему так расширяется.** Новое правило — это новый файл в
`internal/domain/rule/builtin/` (реализует `Rule.Check(*config.Config) []issue.Issue`,
опционально ещё `CheckFile(fs.FileInfo, path)` из `FileAwareRule`, если нужны
метаданные файла) плюс одна строчка в `builtin.All()`. Новый формат конфига —
это `parser.Parser` в `infrastructure/parser` плюс регистрация в
`parser.NewRegistry(...)` в `cmd/configguard/main.go`. Ни то, ни другое не
трогает остальной код.

**Как правило видит конфиг.** `config.Config.Walk` обходит распарсенное дерево
в глубину (ключи map сортируются — вывод детерминирован) и отдаёт каждому
узлу его `Node{Path, Key, Value}`. Правила матчатся по имени ключа
(`password`, `host`, `tls`, `digest-algorithm`, ...), а не по жёсткой схеме —
поэтому работают на любом по форме конфиге, не только на примерах из ТЗ.

## Правила

| Правило | Что ищет | Уровень |
|---|---|---|
| `debug-logging` | `level: debug` / `debug: true` | LOW |
| `plaintext-password` | `password/secret/token/...` открытым текстом (не `${...}`/`env:`) | HIGH |
| `unrestricted-bind` | `host/bind/address: 0.0.0.0` или `::` | MEDIUM |
| `tls-disabled` | `tls.enabled: false`, `insecure_skip_verify: true` и аналоги | HIGH |
| `weak-algorithm` | `algorithm/cipher/digest/hash: MD5\|SHA1\|DES\|3DES\|RC4` | HIGH |
| `permissive-file-mode` | `mode/permissions/chmod`, дающие запись группе/всем | MEDIUM/HIGH |
| `file-permission` | реальные права **самого файла конфига** (`os.Stat`) — world-writable/readable | HIGH/MEDIUM |

`file-permission` — единственное правило, которому нужен реальный файл на
диске: оно реализует `FileAwareRule` и не выполняется при `--stdin` или при
проверке через HTTP/gRPC (там нет файла, есть только байты).

## Многопоточность

- **Рекурсивный обход директории** (`internal/application/dirscan`) — классический
  worker pool: `filepath.WalkDir` сначала собирает пути к конфигам, затем N
  горутин (по умолчанию `runtime.NumCPU()`) разбирают их параллельно, читая
  из канала задач (fan-out) и складывая результаты в общий канал (fan-in),
  синхронизация через `sync.WaitGroup`. `context.Context` пробрасывается для
  досрочной остановки.
- **HTTP и gRPC серверы** — каждый входящий запрос стандартная библиотека
  обслуживает в отдельной горутине, так что параллельные проверки конфигов
  через API не блокируют друг друга.
- **`serve --http --grpc`** — оба сервера поднимаются и останавливаются
  параллельно (`sync.WaitGroup`), не дожидаясь друг друга.

## Graceful shutdown

`configguard serve` слушает `SIGINT`/`SIGTERM` через `signal.NotifyContext`.
По сигналу:
- HTTP: `http.Server.Shutdown(ctx)` с таймаутом 10с — новые соединения не
  принимаются, активные запросы успевают завершиться.
- gRPC: `grpc.Server.GracefulStop()` с тем же таймаутом, при превышении —
  принудительный `Stop()`.

Оба останавливаются одновременно, процесс завершается только когда оба
подтвердили остановку (или истёк таймаут).

## Запуск

```bash
make build        # bin/configguard
```

### CLI

```bash
# файл
./bin/configguard scan testdata/weak-algorithm.yaml

# stdin
echo '{"log":{"level":"debug"}}' | ./bin/configguard scan --stdin

# директория рекурсивно
./bin/configguard scan testdata/

# не выходить с ошибкой при найденных проблемах
./bin/configguard scan -s testdata/multi-issue.yaml
```

Флаги: `-s/--silent` — exit code 0 даже если проблемы найдены;
`--stdin` — читать конфиг из stdin вместо файла. Если хотя бы одна проблема
найдена и `--silent` не передан — exit code 1 (в директории — если хоть один
файл с проблемами).

### HTTP REST API

```bash
./bin/configguard serve --http :8080
curl -X POST localhost:8080/api/v1/scan -d @testdata/multi-issue.yaml
curl localhost:8080/healthz
```

`POST /api/v1/scan` — тело: сырой конфиг, `Content-Type: application/json`
или `application/yaml` (по умолчанию JSON). Ответ 200 всегда, кроме
нечитаемого/невалидного тела (400): `{"source":"request","issues":[...],"has_issues":true}`.

### gRPC API

```bash
./bin/configguard serve --grpc :9090
```

Сервис `configguard.v1.ScannerService/Scan` (контракт в
`api/proto/configguard/v1/configguard.proto`): `ScanRequest{config, format}` →
`ScanResponse{issues, has_issues}`. Оба сервера можно поднять одновременно:
`serve --http :8080 --grpc :9090`.

### Docker

```bash
make docker-build
docker run --rm configguard scan --stdin < testdata/weak-algorithm.yaml

# CMD в образе по умолчанию поднимает и HTTP, и gRPC — пробрасывай оба порта
docker run --rm -p 8080:8080 -p 9090:9090 configguard

# либо переопредели команду явно, если нужен только один сервер
docker run --rm -p 8080:8080 configguard serve --http :8080
```

## Тесты

```bash
make test   # go test ./... -race
```

Покрыто: каждое правило (позитив/негатив), `scanner.Service` (в том числе
дословные примеры из ТЗ и FileAwareRule), `dirscan` worker pool, HTTP-хендлер
(`httptest`).

## Известные ограничения

- `permissive-file-mode` понимает права как строку-восьмеричное число
  (`"0777"`, `"777"`) или как уже готовое число прав доступа — если в
  конфиге десятичное число, не совпадающее по смыслу с восьмеричным
  (нетипичный случай), результат будет неверным.
