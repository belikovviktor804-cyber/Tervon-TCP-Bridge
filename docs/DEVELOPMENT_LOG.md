# Tervon TCP Bridge — журнал разработки

Журнал фиксирует проверяемые факты. Источник критериев и очередности: [DEVELOPMENT_PLAN.md](DEVELOPMENT_PLAN.md).

## 2026-11-17 — M1: foundation кадрирования

- **Ветка:** `work/m1-framing` (SSH Git remote: `git@github.com:tervon-software/Tervon-TCP-Bridge.git`).
- **Код:** `f508f1b025eb49eb07c48b8befb10d10d0a4964d`.
- **Файлы:** `prototype/framing/framing.go`, `prototype/framing/framing_test.go`.
- **Реализовано:** TBF1 v1, DATA/FIN, max payload 64 KiB, проверка структуры/длины, unit/negative/fuzz harness.
- **Среда:** Homeserver, изолированная рабочая копия `prototype/`; не production.
- **Команды:** `go list ./...`, `go test -race -count=1 ./...`, `go vet ./...`.
- **Результат:** 4 пакета обнаружены, `go test -race` PASS для всех; `go vet` без ошибок (по выводу пользователя). GitHub подтверждает наличие файлов и commit. Fuzz harness существует, отдельный длительный fuzz-run не подтверждён.
- **Не выполнено:** version/capability negotiation; HTTP/2 client/server integration; framed full duplex, half-close, EOF, cancellation, negative/compatibility E2E; итоговая проверка M1 на интеграционном commit.
- **Статус:** `IN_PROGRESS`, не PASS, не release.
- **Эксплуатация:** работающие службы не изменялись, `/v1/stream` сохранён.
- **Следующий шаг:** интеграция framed opt-in в этом же M1; тесты, исправления, документирование и коммит; закрытие только по фактическому PASS.

## Правило ведения журнала

После каждого значимого изменения в текущем модуле фиксировать branch, SHA, изменённые файлы, проверяемый вопрос, команды, среду, результат, ограничения и влияние на production. Нельзя заменять подтверждённые проверки планируемыми.
