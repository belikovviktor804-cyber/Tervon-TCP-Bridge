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
