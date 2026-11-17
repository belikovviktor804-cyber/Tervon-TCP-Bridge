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

## 2026-11-17 — M1: stream adapter, проверка ожидается

- **Ветка:** `work/m1-framing`.
- **Коммиты кода:** `a839354cbbb3b358fbf83d25974296000b3f4927`, `87bb1fe46cb2f2da9d4aec22ff75ed02ff1f3061`.
- **Добавлено:** `prototype/framing/stream.go` и `prototype/framing/stream_test.go`.
- **Реализовано в исходниках:** bounded DATA framing для произвольного byte stream, FIN для half-close, запрет записи после FIN, обнаружение неожиданного EOF до FIN, чтение маленькими порциями.
- **Тесты:** добавлены тестовые сценарии; **выполнение тестов на этих коммитах пока НЕ ПОДТВЕРЖДЕНО**. Статус `UNVERIFIED`.
- **Не реализовано:** HTTP/2 transport negotiation, server/client integration, framed E2E и совместимость.
- **Production:** не изменялся. M1 остаётся `IN_PROGRESS`.
- **Следующая проверка на Homeserver:** `go test -race -count=1 ./...`, `go vet ./...`, `go test -run 'TestStream' -v ./framing`; фиксировать точный SHA.

## 2026-11-17 — M1: подтверждённый PASS stream adapter

- **Проверенный commit:** `08ab050eed7f877935eee9392bc3272ff0966e02` (`work/m1-framing`).
- **Среда:** Homeserver, изолированная рабочая копия `/opt/tervon/bridge-m0-check/build/repo/prototype`; без изменения production.
- **Команды и результаты:** `go test -race -count=1 ./...` — PASS для `tervon-tcp-bridge`, `cmd/bridge`, `framing`, `telemetry`; `go vet ./...` — без ошибок; `go test -run 'TestStream' -v ./framing` — PASS для пяти тестов (RoundTrip, RequiresFIN, FIN, Corrupt, SmallReads).
- **Git:** рабочее дерево чистое после тестов.
- **Ограничение:** это PASS **адаптера**, не полного M1. HTTP/2 negotiation и интеграционные E2E ещё не реализованы и не проверены; fuzz harness есть, длительный fuzz-run не проводился.
- **Следующий пакет:** opt-in framed endpoint, строгая версия/возможности, клиент/сервер, несовместимые версии, двусторонние E2E и документация. `/v1/stream` оставить legacy.

## 2026-11-17 — M1: framed HTTP/2 integration candidate (UNVERIFIED)

- **Code commits:** `261f841c48e9d4fda65fa7511b3b1467f2ae124d` (client/server opt-in), `5525c83c20e64904c7bad5f0cae18fc849ad5834` (CLI URL), `0e80a9e315562156afa6466a664bda1d9161391e` (integration tests).
- **Protocol:** `/v1/stream` remains legacy raw TCP; `/v2/stream` requires `Tervon-Wire-Version: 1` and `Tervon-Wire-Capability: tbf1-fin`; incompatible requests rejected with 426 before backend dial. Successful response echoes both headers. Client checks headers and does not silently downgrade.
- **Data plane:** HTTP/2 upload and download framed independently via TBF1 DATA/FIN. Payload counted before encoding / after decoding. CLI allows explicit `https://.../v2/stream` opt-in.
- **Added tests:** `TestFramedByteExact` and `TestFramedRejectVersionBeforeDial`. **Tests have not yet been run on this commit.**
- **Known risks requiring correction before M1 PASS:** framed decoder errors in server upload are not yet propagated to handler; trailing frames after FIN are not rejected; existing connection shutdown can interrupt half-close; no explicit proof of cancellation/goroutine cleanup or production compatibility. These are open issues, not PASS.
- **Next verification:** on isolated Homeserver clone run `go test -race -count=1 ./...`, `go vet ./...`, `go test -run 'TestFramed' -v ./...` and report exact SHA. Do not deploy.
- **M1:** IN_PROGRESS. Production: NONE.

## 2026-11-17 — M1: подтверждённый интеграционный PASS (частичный)

- **Проверенный commit:** `dd1ede2679c1a507ad9bdca344d13ed18d5ebc03`.
- **Среда:** Homeserver, изолированный checkout; работающие службы не затрагивались.
- **`go test -race -count=1 ./...`:** PASS для `tervon-tcp-bridge`, `cmd/bridge`, `framing`, `telemetry`.
- **`go vet ./...`:** PASS (без замечаний).
- **`go test -race -count=1 -run 'TestFramed' -v ./...`:** PASS: `TestFramedByteExact`, `TestFramedRejectVersionBeforeDial`. Остальные пакеты ожидаемо сообщили `no tests to run`.
- **`git status --short`:** чистое дерево.
- **Что доказано:** базовая byte-exact передача через opt-in `/v2/stream`, отказ при несовместимой версии до backend dial.
- **Что НЕ доказано:** корректный half-close при всех направлениях, propagation ошибок кадрирования, запрет данных после FIN, cancellation, отсутствие зависших горутин, отдельный длительный fuzz-run и совместимость со старым сервером.
- **Статус:** интеграционный smoke PASS, **M1 IN_PROGRESS**, production не изменён.

## 2026-11-17 — M1: исправление ошибочного half-close и проверка FIN (UNVERIFIED)

- **Базовый проверенный SHA:** `dd1ede2679c1a507ad9bdca344d13ed18d5ebc03`, smoke PASS на Homeserver (см. запись выше).
- **Изменения кода:** `590747f43101bbd35b1806e6f72d3e0177a490ee` — server upload теперь закрывает backend TCP при ошибке декодирования/записи вместо выдачи успешного half-close; `a6364506ec5b147c8b920e53a1077c1404c87a42` — decoder требует EOF после FIN и отклоняет последующие байты; `ce0e4422a31a0baa820b461423fd3c6ee603150a` — тесты лишних байтов и второго FIN.
- **Вопрос проверки:** отклоняются ли данные после FIN; не маскируется ли повреждённый upload под штатное завершение backend.
- **Статус:** код записан в GitHub; тесты на новых коммитах **не запускались**, PASS не заявлен.
- **Открыто:** decoder ждёт транспортного EOF после FIN (возможна задержка при некорректном peer); HTTP/2 cancellation/half-close E2E, обработка ошибок download, проверки на утечки горутин и fuzz-run. Не развёртывать.
- **Проверка:** `go test -race -count=1 ./...`, `go vet ./...`, `go test -run 'TestStream|TestFramed' -v ./...` на изолированной рабочей копии с точным SHA.
- **M1:** IN_PROGRESS. **Production:** NONE.

## 2026-11-17 — M1: подтверждённый PASS негативных FIN-тестов

- **Проверенный SHA:** `572d2173eaa6bfcc0743fcfd858249b2f56ef12a` (`work/m1-framing`).
- **Среда:** Homeserver, изолированная рабочая копия, без воздействия на действующие службы.
- **`go test -race -count=1 ./...`:** PASS, четыре пакета (`tervon-tcp-bridge`, `cmd/bridge`, `framing`, `telemetry`).
- **`go vet ./...`:** PASS, вывод ошибок отсутствует.
- **`go test -count=1 -run 'TestStream|TestFramed' -v ./...`:** PASS, включая `TestStreamRejectTrailingAfterFIN`, `TestStreamRejectSecondFIN`, `TestFramedByteExact`, `TestFramedRejectVersionBeforeDial`, а также тесты ограничения concurrent streams. Предупреждения `no tests to run` в двух пакетах ожидаемы.
- **Git:** `git status --short` пустой.
- **Доказано:** корректность проверенных сценариев FIN/negotiation и отсутствие выявленных race в текущем тестовом наборе. **Не доказано:** все сетевые failure paths, полный half-close, отмена и отсутствие утечек, длительный fuzz-run.
- **Статус:** текущий пакет PASS, **M1 IN_PROGRESS**, production без изменений.

## 2026-11-17 — M1: HTTP/2 cancellation regression test (UNVERIFIED)

- **Code commit:** `0658d413cd5e9a604e61d291a9f20892286b08bd`.
- **Изменение:** `prototype/framed_integration_test.go` — добавлен `TestFramedCancellationReleasesHandler`: поднять тестовый HTTP/2 mTLS bridge и TCP echo backend, проверить обмен данными, отменить родительский context, потребовать завершения `HandleContext` за 4 секунды.
- **Вопрос:** освобождается ли обработчик активного framed-соединения при отмене контекста после успешного обмена.
- **Результат:** исходник теста добавлен; исполнение на данном commit **не подтверждено**. Нет оснований считать cancellation PASS.
- **Открыто:** полноценный half-close и upload/download errors, отсутствие утечек при отказах, fuzz-run; требуется повторная race-проверка.
- **Статус:** M1 IN_PROGRESS; production NONE. Следующий шаг — запуск тестов на Homeserver, исправление ошибок в M1.

## 2026-11-17 — M1: подтверждённый PASS framed cancellation

- **Проверенный SHA:** `35ba4ec198d1b5dd27caa0f4e0da583b2a979fbc` (`work/m1-framing`).
- **Среда:** Homeserver, изолированная рабочая копия; production без изменений.
- **`go test -race -count=1 ./...`:** PASS, четыре пакета (`tervon-tcp-bridge`, `cmd/bridge`, `framing`, `telemetry`).
- **`go vet ./...`:** PASS без замечаний.
- **`go test -race -count=1 -run 'TestFramedCancellationReleasesHandler' -v .`:** PASS (0.14 s), handler завершается после context cancellation в проверенном HTTP/2 mTLS echo-сценарии.
- **`git status --short`:** пустой вывод.
- **Ограничения:** единичный успешный cancellation-сценарий не подтверждает отсутствие утечек при всех ошибках. Полный half-close, failure paths, fuzz-run и совместимость остаются открытыми.
- **Статус:** частичный PASS; M1 IN_PROGRESS; не release.

## 2026-11-17 — M1: TCP half-close integration candidate (UNVERIFIED)

- **Code commit:** `f47c58a6a8aaf988c60d02d484e90bdba7575331`.
- **Добавлен:** `TestFramedHalfCloseResponseAfterUploadEOF` в `prototype/framed_integration_test.go`.
- **Сценарий:** реальный локальный TCP peer передаёт запрос и делает `CloseWrite`; backend читает до EOF и только затем отвечает; клиент обязан получить полный ответ, обработчик и backend обязаны завершиться. Проверка выполняется через тестовый HTTP/2 mTLS bridge с `/v2/stream`.
- **Вопрос:** сохраняется ли асимметричный TCP half-close при framed-обёртке, без преждевременного закрытия противоположного направления.
- **Результат:** тест написан и закоммичен; **не выполнялся на Homeserver**. Возможный FAIL означает ошибку жизненного цикла, которую необходимо исправить до закрытия M1.
- **Проверить:** `go test -race -count=1 ./...`, `go vet ./...`, `go test -race -count=1 -run 'TestFramedHalfCloseResponseAfterUploadEOF' -v .`, точный SHA и чистое дерево.
- **Открыто:** error paths, строгая отмена и завершение горутин при сбоях, fuzz-run, совместимость. **M1 IN_PROGRESS; production NONE**.

## 2026-11-17 — M1: подтверждённый PASS TCP half-close E2E

- **Проверенный commit:** `053d24ae17afe55225a5d22eac8a64e93c25d807` (`work/m1-framing`).
- **Среда:** Homeserver, изолированная рабочая копия; действующие службы не затронуты.
- **`go test -race -count=1 ./...`:** PASS для четырёх пакетов.
- **`go vet ./...`:** PASS.
- **`go test -race -count=1 -run '^TestFramedHalfCloseResponseAfterUploadEOF$' -v .`:** PASS, 0.16 s.
- **Сценарий:** реальный локальный TCP peer вызывает `CloseWrite`; backend читает запрос до EOF, затем отправляет ответ; клиент получает весь ответ, handler и backend завершаются.
- **`git status --short`:** чистое дерево.
- **Ограничение:** один подтверждённый E2E half-close не заменяет тестирование ошибок передачи, leak/cancellation при отказах, fuzz-run и совместимости.
- **Статус:** пакет PASS, **M1 IN_PROGRESS**, не выпуск.

## 2026-11-17 — M1: truncated HTTP/2 download negative test (UNVERIFIED)

- **Code commit:** `bc92c41406addfc42cf9ae356c9399c2be268bcd`.
- **Добавлено:** `TestFramedRejectsTruncatedDownload` в `prototype/framed_integration_test.go`. Изолированный mTLS HTTP/2 test server объявляет v2 и корректные negotiation headers, но возвращает неполный TBF1 header без FIN. Клиент должен вернуть `framing.ErrMalformed`, не зависнув.
- **Вопрос:** обнаруживается ли повреждённый download как ошибка, а не штатный EOF.
- **Результат:** исходник записан в GitHub; запуск на данном commit не подтверждён.
- **Следующий шаг:** проверить `go test -race -count=1 ./...`, `go vet ./...`, `go test -race -count=1 -run '^TestFramedRejectsTruncatedDownload$' -v .` на Homeserver, зафиксировать точный SHA; исправить FAIL до следующего пакета.
- **Открыто:** malformed upload E2E, failure-path cleanup, fuzz-run, compatibility; **M1 IN_PROGRESS**, production NONE.

## 2026-11-17 — M1: объединённый пакет негативных проверок (ожидает полного PASS)

- **Последний подтверждённый Homeserver SHA:** `385b869ec22be9be40f5a5ad77cb79f9e597d651`. `go test -race -count=1 ./...`: PASS (4 пакета); `go vet ./...`: PASS; `TestFramedRejectsTruncatedDownload`: PASS (0.12 s); рабочее дерево чистое.
- **Новый кодовый commit:** `18debb5c8b32661b23a9c14873fc545499e3d478`.
- **Пакет тестов:** `TestStreamEncoderWriteFailureIsTerminal`, `TestStreamEncoderCloseWriteFailureIsTerminal`, `TestStreamRejectTruncatedPayload` и `FuzzStreamDecoder`. Проверяют отказ writer, запрет продолжения потока после ошибки, обрыв payload и свойство принятого decoder потока: ровно один конечный FIN без хвоста.
- **Статус нового кода:** закоммичен, но не протестирован на Homeserver. Fuzz harness добавлен, длительный fuzz-прогон не подтверждён.
- **Для итогового прохода M1:** `go test -race -count=1 ./...`; `go vet ./...`; `go test -run 'TestFramed|TestStream' -count=1 -v ./...`; `go test -run '^$' -fuzz '^FuzzReadFrame$' -fuzztime=30s ./framing`; `go test -run '^$' -fuzz '^FuzzStreamDecoder$' -fuzztime=30s ./framing`; `git status --short`. Каждый fuzz target запускается отдельно.
- **Не закрыто:** malformed upload end-to-end, подтверждение корректной обработки ошибок на обоих направлениях, полная проверка lifecycle/cleanup и совместимости с legacy. **M1 IN_PROGRESS**, production NONE. Не объявлять M1 PASS по одному тесту.

## 2026-11-17 — M1: единый кандидат для интеграционной и негативной проверки

- **Branch:** `work/m1-framing`; предыдущий подтверждённый PASS: `385b869ec22be9be40f5a5ad77cb79f9e597d651`.
- **Новые кодовые commits:** `18debb5c8b32661b23a9c14873fc545499e3d478` (отказы writer, обрыв payload, `FuzzStreamDecoder`), `a503950e2c16923a1d2d927f1746326219d228b6` (HTTP/2 malformed upload и отказ negotiation ответа).
- **Новые E2E-тесты:** `TestFramedMalformedUploadClosesBackend` — отправка усечённого TBF1-заголовка вместо FIN и контроль завершения backend-соединения; `TestFramedRejectsMissingResponseNegotiation` — клиент не принимает HTTP 200 без обязательных заголовков v2.
- **Статус:** написано/закоммичено, **не проверено**. Проверка malformed upload подтверждает закрытие backend, но отдельно ещё не доказывает способ закрытия (TCP RST против FIN) и отсутствие передачи повреждённых байтов.
- **Следующий единый прогон:** полный `go test -race -count=1 ./...`, `go vet ./...`, все `TestFramed|TestStream`, отдельные fuzz targets по 30s, точный SHA, чистый git. При FAIL — исправление в M1 без перехода к M2.
- **Нерешённое:** анализ ошибок download/write после отправки HTTP 200, строгий cleanup goroutines при ошибках и cancellation, полноценная проверка совместимости v1. **M1 IN_PROGRESS**, production NONE.

## 2026-11-17 — M1: запрет ложного FIN при ошибке download (UNVERIFIED)

- **Commit кода:** `9f9fafdb463587db2b0ec40b70e1bbac1640af11` в `work/m1-framing`.
- **Изменение:** `prototype/bridge.go` теперь сохраняет результат `io.Copy` из backend в HTTP response. При ошибке чтения backend или записи в HTTP response **не отправляет TBF1 FIN**, закрывает backend; при успешной передаче отправляет FIN, проверяя результат его записи.
- **Цель:** не маркировать повреждённую передачу как штатный EOF. HTTP status уже может быть 200, поэтому отменить ответ статусом невозможно; корректность сигнализации ошибки проверяется по отсутствию FIN.
- **Ограничения:** код не проходил race/E2E на новом SHA; обработка ошибок upload и подтверждение отсутствия утечек требуют отдельной проверки. Тесты из commits `18debb5c` и `a503950e` также пока UNVERIFIED.
- **Следующий шаг:** завершить M1 код/тесты, затем единый итоговый прогон на Homeserver; production не менять. **M1 IN_PROGRESS**.

## 2026-11-17 — M1: client upload cleanup на всех путях выхода (UNVERIFIED)

- **Code commit:** `d70b8f06df571eaeb25f34295270592ac939a6e6`.
- **Изменение:** `Client.HandleContext` теперь использует единый отложенный cleanup после запуска upload goroutine: отмена контекста, закрытие TCP и обоих концов pipe, ожидание завершения upload до 2 секунд. Это применяется также при отказе HTTP, неверном ответе negotiation, повреждённом download и штатном EOF.
- **Цель:** не оставлять незавершённую upload goroutine на ранних путях выхода, ранее не ожидавших её завершения.
- **Ограничение:** timeout явно возвращается как ошибка, но сам по себе не доказывает отсутствия goroutine leak. Новое поведение ещё не проверено; нужен единый race/E2E/fuzz прогон и исправление любых FAIL.
- **Статус:** M1 IN_PROGRESS; production NONE; переход к M2 запрещён до подтверждённого PASS.

## 2026-11-17 — M1: server-side malformed upload / FIN guard (UNVERIFIED)

- **Code commit:** `870936de22e0fd4f1c4f5dc149266a3fad3cf77d` (`work/m1-framing`).
- **Изменение:** server upload goroutine атомарно фиксирует ошибку чтения/декодирования до закрытия backend. Download path проверяет этот флаг перед отправкой завершающего FIN. Цель — не выдавать успешное завершение ответа, когда ошибка upload уже обнаружена.
- **Ограничение:** проверка флага не является полной синхронизацией обоих направлений. Если backend EOF наступит до обнаружения ошибки upload, FIN ещё может уйти. Для строгого доказательства нужны adversarial E2E и согласованная state machine; не объявлять проблему закрытой.
- **Тесты:** новые commits после последнего подтверждённого `385b869ec22be9be40f5a5ad77cb79f9e597d651` не проверены. Единый прогон Homeserver запланирован после завершения M1.
- **Статус:** M1 IN_PROGRESS, production NONE. Без изменения действующих служб.

## 2026-11-17 — M1: синхронизация FIN с подтверждённым завершением upload (UNVERIFIED)

- **Code commit:** `ad33d3131a81b6dfb7070c70ceec18fe723bd87c`.
- **Изменение:** перед отправкой успешного response FIN в framed-режиме сервер теперь ожидает результат декодирования upload, включая проверку request FIN и EOF. Если upload завершился ошибкой или не завершился за 500 ms после backend EOF, ответ FIN не отправляется. Убрана недостаточная атомарная проверка, оставлявшая race window.
- **Компромисс:** backend, который закрывает ответ до завершения request upload, теперь получает fail-closed поведение вместо успешного FIN. Это требует отдельного теста и документирования совместимости. Ожидание 500 ms не гарантирует завершения goroutine, если чтение HTTP body не прерывается.
- **Статус:** только GitHub code commit, без подтверждённых race/E2E тестов. Не считать race окончательно устранённой до adversarial test. **M1 IN_PROGRESS**, production NONE.

## 2026-11-17 — M1: adversarial E2E для раннего backend EOF (UNVERIFIED)

- **Code commit:** `c95085bdbd722edf338d14bbd875285ec253aeb7`.
- **Новый тест:** `TestFramedBackendEarlyEOFDoesNotForgeFIN`: реальный TCP backend закрывается сразу после accept; клиент отправляет повреждённый TBF1 upload без FIN; проверяется, что декодер ответа отвергает поток с `framing.ErrMalformed`, а backend завершается.
- **Цель:** регрессионная проверка fail-closed решения `ad33d313` при конкуренции backend EOF и ошибки upload. Тест ещё не запускался и не доказывает все возможные interleavings.
- **Открыто:** серверный cleanup при блокирующем request-body read, полноценная v1 legacy совместимость и единый race/vet/fuzz прогон. **M1 IN_PROGRESS**; production NONE.

## 2026-11-17 — M1: серверное ожидание upload после раннего EOF (UNVERIFIED)

- **Code commit:** `6c398dc2d0ce60234d6007b73152e2d9079914f8`.
- **Исправление:** сервер учитывает, был ли результат upload goroutine действительно получен до отправки FIN. После закрытия backend и request body ожидает завершения upload, если результат ещё не получен, включая случай истечения первого 500 ms ожидания. Ранее в framed timeout path второе ожидание пропускалось.
- **Ограничение:** два ограниченных ожидания не гарантируют завершение чтения тела запроса. Нужны adversarial cancellation и goroutine lifecycle тесты; ни новый код, ни накопленные M1 commits пока не проверены единым прогоном.
- **Статус:** M1 IN_PROGRESS, production NONE. Ветка `work/m1-framing`; действующие службы не изменялись.

## 2026-11-17 — M1: legacy v1 regression and persistent checkpoint (UNVERIFIED)

- **Code commit:** `74f4abb22a2cc1fb37609a3568800ae7f1cc6caa`.
- **Tests:** `TestLegacyV1StreamRemainsByteExact` verifies unframed `/v1/stream` round-trip with binary payload and handler termination; `TestFramedMalformedUploadClosesBackend` accepts Linux TCP reset as an expected closed-connection outcome, while rejecting unrelated read errors.
- **Source of truth:** `docs/DEVELOPMENT_PLAN.md` now records active branch, code SHA, last verified PASS, pending tests, known lifecycle risk, and strict M1 exit gate.
- **Evidence:** tests not yet run at the new SHA. **M1 IN_PROGRESS**, no release, no Homeserver service changes.


## 2026-11-17 — M1: server upload lifecycle join and cancellation regression (UNVERIFIED)

- **Branch:** `work/m1-framing`; baseline HEAD before edits: `af966bc77c719bd39b11508c9576c4239c0f60a8`.
- **Code commits:** `e3c365daea873bfad2331a84b5271392a3f9fd3d` (server upload worker join), `ddd72e2faa61d1732321c18d76089222af22e377` (cancellation E2E).
- **Changed:** `prototype/bridge.go` replaces a 500 ms best-effort join with an unconditional upload-worker join after closing the request body and backend. `prototype/framed_integration_test.go` adds `TestFramedCancelRequestBodyReaderReleasesServerWorker`.
- **Question:** can a canceled HTTP/2 upload leave the backend socket or request handler alive after an incomplete TBF1 stream?
- **Tests:** NOT RUN on these commits. The available environment has no direct Homeserver access; GitHub code write alone is not a verified test result.
- **Open risk:** `r.Body.Close()` is not proven to wake a blocked HTTP/2 body read on every failure path; unconditional join could hang if the reader cannot be interrupted. The new test checks one cancellation path, not all leak scenarios. If this test hangs, correct cancellation semantics within M1 before any module promotion.
- **Required verification on isolated Homeserver checkout:** `go test -race -count=1 ./...`, `go vet ./...`, `go test -race -count=1 -run 'TestFramedCancelRequestBodyReaderReleasesServerWorker|TestFramed|TestLegacyV1StreamRemainsByteExact' -v .`, plus separate fuzz runs for `FuzzReadFrame` and `FuzzStreamDecoder`; record exact HEAD and clean working tree.
- **Status:** M1 IN_PROGRESS / UNVERIFIED; release NONE; production services untouched. Do not proceed to M2 before integrated PASS.


## 2026-11-17 — M1: проверенный GitHub Actions PASS полного CI-пакета

- **Кодовый commit:** `b7e620646921503d728eb8a21e281ef31f0327f0` — `TestFramedBackendEarlyEOFWithOpenUploadReleasesHandler` проверяет ранний backend EOF при открытом HTTP/2 request body и отсутствие зависшего server handler.
- **CI commit:** `6c9aef45f9700c629c779bdd300b65f0e53e8b7c` — добавлены повторный race-прогон framed/legacy тестов и два ограниченных fuzz-target.
- **GitHub Actions:** [run #44036843167](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44036843167) — PASS на кодовом SHA; [run #44036862629](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44036862629) — PASS на workflow SHA. Ubuntu GitHub-hosted runner, production NONE.
- **Команды и результаты на run #44036862629:** `go build ./...` PASS; `go vet ./...` PASS; `go test -count=1 -timeout=60s ./...` PASS (4 пакета); `go test -race -count=1 -timeout=120s ./...` PASS (4 пакета); `go test -race -count=3 -timeout=180s -run 'TestFramed|TestLegacyV1StreamRemainsByteExact|TestStream' ./...` PASS.
- **Fuzz:** `go test -run '^$' -fuzz '^FuzzReadFrame$' -fuzztime=15s -parallel=2 -timeout=90s ./framing` — PASS, 722070 executions; `FuzzStreamDecoder` теми же параметрами — PASS, 718491 executions. Длительный fuzz-run не проводился.
- **Границы доказательств:** standard HTTP/2 mTLS loopback integration, не произвольные кастомные блокирующие `io.ReadCloser`. На Homeserver новый SHA не проверялся; v1.0.0 production не переключался. `gofmt -l` остаётся non-blocking.
- **Статус:** CI PASS / M1 IN_PROGRESS / release NONE. Следующий шаг — изолированный Homeserver итоговый прогон на точном HEAD, проверка fail-closed поведения и формальное решение о завершении M1 по критериям плана.


## 2026-11-17 — M1: изолированный Homeserver verification runner и повторный CI PASS

- **Commits:** `4cc39b7d0dd2c3f461b41bc960bb13f3505faffd` (script), `da40bb9da134631551581015bfeb2bfbb56a8d34` (remote SHA guard), `9ad9265b175f114178007321049811d5d595c11c` (CI syntax check).
- **Файл:** `tests/homeserver/verify-m1.sh`. Проверяет чистоту изолированного checkout, SSH remote, активную ветку, совпадение локального HEAD с `FETCH_HEAD` после SSH fetch; запрещает известные release/production каталоги. Запускает build, vet, full race, 3x targeted race, два fuzz-target по 30s и повторную проверку чистоты дерева. Только кодовые проверки, без systemctl, deploy, сетевых production endpoint.
- **GitHub Actions:** [run #44037187498](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44037187498) — SUCCESS/PASS на `9ad9265b175f114178007321049811d5d595c11c`, включая `bash -n` скрипта и все ранее добавленные Go/CI тесты. Это **CI PASS**, а не Homeserver PASS.
- **Следующий шаг:** на Homeserver в существующем изолированном checkout выполнить fast-forward ветки и `bash tests/homeserver/verify-m1.sh "$PWD"`; сохранить stdout/stderr, точный SHA, `M1_VERIFY_RESULT`. Если FAIL — исправить в M1 и повторить CI/проверку; не переходить к M2.
- **Статус:** M1 IN_PROGRESS, production NONE, release NONE.


## 2026-11-17 — M1: Homeserver изолированный полный кодовый PASS (пользовательский протокол)

- **Проверенный SHA:** `e2313ab8375be3d9a7934e8b1c462a17e1fa3286`, ветка `work/m1-framing`. Пользователь выполнил `bash tests/homeserver/verify-m1.sh "$PWD"` из `/opt/tervon/bridge-m0-check/build/repo`; локальный HEAD после `git fetch origin work/m1-framing` и fast-forward совпал с remote HEAD. Вывод скрипта: `M1_VERIFY_BRANCH=work/m1-framing`, `M1_VERIFY_SHA=e2313ab8375be3d9a7934e8b1c462a17e1fa3286`, `M1_VERIFY_RESULT=PASS`.
- **Среда:** Homeserver, Go `go1.25.5 linux/amd64`, isolated checkout. Скрипт зафиксировал `M1_VERIFY_ENV=isolated_checkout`, `M1_VERIFY_PRODUCTION_CHANGES=NONE`, `M1_VERIFY_NOTE=code_level_only_no_production_deployment`. Отсутствие изменения работающих служб следует из назначения скрипта; отдельный аудит состояния служб в этом прогоне не выполнялся.
- **Build:** `go build ./...` — PASS. **Vet:** `go vet ./...` — PASS.
- **Full race:** `go test -race -count=1 -timeout=120s ./...` — PASS для `tervon-tcp-bridge`, `cmd/bridge`, `framing`, `telemetry`.
- **Targeted 3x race:** `go test -race -count=3 -timeout=180s -run 'TestFramed|TestLegacyV1StreamRemainsByteExact|TestStream' ./...` — PASS (включая framed cancellation, early EOF и legacy v1 regression).
- **Fuzz:** `FuzzReadFrame` 30s — PASS, **1 347 631** executions, 11 interesting corpus total; `FuzzStreamDecoder` ~31s — PASS, **1 211 473** executions, 15 interesting corpus total. Не интерпретировать как доказательство отсутствия любых ошибок.
- **Ограничения:** проверка кодовая/loopback; отдельного cross-version E2E с реальным бинарником v1.0.0 не было; 500ms early-backend-EOF fail-closed требует отдельной оценки; production deployment не выполнялся. `M1_VERIFY_RESULT=PASS` относится к командам скрипта, не ко всем критериям релиза.
- **Решение:** M1 **CODE_VERIFIED_PASS**, но **IN_PROGRESS** до совместимости и подтверждения оставшихся критериев; M2 не начинать.


## 2026-11-17 — M1: bidirectional legacy v1 wire contract regression (CI PASS)

- **Baseline Homeserver PASS:** `e2313ab8375be3d9a7934e8b1c462a17e1fa3286`, Go 1.25.5 linux/amd64, complete `verify-m1.sh` result `M1_VERIFY_RESULT=PASS`. Сохранено в предыдущей записи; не распространять на новые изменения.
- **Code commit:** `e26792c61d91b605d43bf6ff76f370f667c7ddab`. Новый `prototype/legacy_wire_compat_test.go` проверяет старый raw HTTP/2 request без TBF1 headers против нового `/v1/stream` сервера и новый `/v1/stream` клиент против старого raw HTTP/2 server contract. Оба сценария используют тестовые сертификаты, loopback и бинарный payload. Это **wire-contract simulation**, а не фактический запуск старого v1.0.0 бинарника.
- **CI/verification commits:** `dd09b1cf2b1311d896ffb1ffd08daa38ea43374c` и `ce551cfeed2c4ecdcf7dbc760a7e45fa7e16281b` расширяют 3-кратный race-regression фильтр до `TestLegacy*` в CI и Homeserver-скрипте.
- **CI:** [run #44039980495](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44039980495) — SUCCESS/PASS на `ce551cfeed2c4ecdcf7dbc760a7e45fa7e16281b`; Go build, vet, unit, race, 3x framed/legacy race, два fuzz-target по 15s, bash syntax.
- **Pending:** новый код не прогнан на Homeserver; отдельно проверить фактическую совместимость со старым бинарником v1.0.0 в изолированном стенде, не трогая production. Оценить bounded 500ms early backend EOF fail-closed. **M1 IN_PROGRESS**, M2 не начинать.


## 2026-11-17 — M1: второй Homeserver PASS и реальный бинарный v1.0.0 compatibility gate

- **Пользовательский Homeserver PASS:** `8aedaf9e518e9cfbb60529d4ff32c0acb90dabad`, `go1.25.5 linux/amd64`, isolated checkout `/opt/tervon/bridge-m0-check/build/repo`. `M1_VERIFY_RESULT=PASS`, `M1_VERIFY_PRODUCTION_CHANGES=NONE`.
- **Проверки:** `go build ./...` PASS; `go vet ./...` PASS; `go test -race -count=1 -timeout=120s ./...` PASS (4 пакета); `go test -race -count=3 -timeout=180s -run 'TestFramed|TestLegacy|TestStream' ./...` PASS.
- **FuzzReadFrame:** 30s / **1 326 453** executions / 11 corpus / PASS. **FuzzStreamDecoder:** 30s / **1 315 708** executions / 19 corpus (4 новых interesting) / PASS. Никаких crash/fail в логе нет. Увеличение корпуса не является ошибкой.
- **Новый тест:** `prototype/cmd/bridge/legacy_release_compat_test.go` commit `3f951e173051834786b37ff920081e2409747e16`. Opt-in `TERVON_V100_BIN`: проверяет официальный SHA-256 `5c813bae9cb0c68e3d5ac1e9afb0556f13cb270b369a86ae8ffc0d20adb086e2`, копирует бинарник в `t.TempDir()`, строит текущий CLI, создаёт тестовый CA и сертификаты, запускает обе версии только в дочерних процессах на loopback, тестирует old-client/new-server и new-client/old-server с байтовым echo. **Без указанного файла тест SKIP**, поэтому обычный CI не доказывает cross-version PASS.
- **CI-скрипт:** `tests/homeserver/verify-m1.sh` дополнен необязательным gate `M1_REAL_V100=PASS/NOT_RUN`. Коммит `cd9c1632ede701b5485b42d39f532317944e0bc1` внёс синтаксическую ошибку shell; [GitHub Actions #44041214104](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44041214104) **FAIL** на шаге `bash -n`. Ошибка исправлена commit `797f2d93726ded3ef27632ee2e4d590a005ec4fa`; повторный CI ожидает подтверждения.
- **Статус:** второй Homeserver CODE PASS доказан на `8aedaf9...`. Реальная бинарная совместимость v1.0.0 ещё **NOT_RUN**. M1 IN_PROGRESS, M2 не начинать, production не затрагивать.


## 2026-11-17 — M1: CI recovery after opt-in release test harness

- **Failure evidence:** [run #44041214104](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44041214104) FAIL, malformed shell introduced by commit `cd9c1632ede701b5485b42d39f532317944e0bc1`; stopped at `bash -n` before Go tests.
- **Fix:** `797f2d93726ded3ef27632ee2e4d590a005ec4fa` restored complete conditional block and final result output in `verify-m1.sh`.
- **Reverification:** [run #44041281072](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44041281072), SHA `797f2d93726ded3ef27632ee2e4d590a005ec4fa`, **SUCCESS/PASS**: bash syntax, build, vet, unit, race, 3x targeted regression, both 15s fuzz targets.
- **Important:** opt-in real v1.0.0 compatibility test is compiled but skipped in default CI without `TERVON_V100_BIN`. It remains NOT_RUN with actual release binary. Latest user-confirmed Homeserver PASS remains `8aedaf9e518e9cfbb60529d4ff32c0acb90dabad`. **M1 IN_PROGRESS** until cross-version E2E evidence.


## 2026-11-17 — M1 read-only audit and correction batch

- Confirmed M0 is installed and operational: Homeserver `/opt/tervon-tcp-bridge/bridge`, `tervon-bridge-client.service`, SHA-256 `fbc927f33cd0198329bb21ff0b40a5d4e8240bc9a92d3d62c1d9c4e4ec30fc0c`. It differs from published v1.0.0 by design; do not misidentify it as an unknown build.
- Audited M1 against completed M0 baseline `1805f9a4117afdeba6121f893d855db7be655014`. Found incorrect clean TCP half-close after malformed framed download, incomplete installed-M0 compatibility gate, and known limitation of non-cancelable custom `io.ReadCloser`.
- Code correction `22a2bfeb11953350891780b1a10993d3c77b4c28`: framed download errors abort local TCP using `SetLinger(0)` instead of signaling successful EOF; regression test `dcf3b5b27ff795ee076003cea2607b6c4fcc080b` checks actual TCP read error.
- Test harness `8c84e6e56628955af3d036370d7ea51c9e703ae6`: separate checksum-pinned installed M0 and official v1.0.0 compatibility checks; no production processes or configuration touched.
- Initial M0 verification shell addition `5dc48cf87097f556055d6c6d3d62234b42715cb4` caused syntax failure, CI #44044892316 FAIL at `bash -n`; repaired in `307c59564f3a49317227488e32c2e8f7b57a7719`. **Re-run CI pending** at documentation time; never treat a failed run as PASS.
- No server commands executed, no production deployment, no M2 work. Next gate: CI PASS on corrected code, followed by isolated Homeserver test with `TERVON_M0_BIN=/opt/tervon-tcp-bridge/bridge`; retain M1 IN_PROGRESS until verified.
