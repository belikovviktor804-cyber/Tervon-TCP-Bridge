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

- **CI recovery verified:** [run #44044975373](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44044975373) on `307c59564f3a49317227488e32c2e8f7b57a7719` — **SUCCESS/PASS** (bash syntax, build, vet, unit, race, repeated M1 regressions, FuzzReadFrame, FuzzStreamDecoder). This does not exercise the opt-in installed M0/v1.0.0 binary compatibility checks.


## 2026-11-17 — Homeserver M1 verification: deployed M0 compatibility PASS

- User ran isolated checkout at `/opt/tervon/bridge-m0-check/build/repo`, branch `work/m1-framing`, SHA **`5ae795f0a6f342724f6dc894f0e82b7f4402f867`**, Go **1.25.5 linux/amd64**.
- `go build ./...` **PASS**; `go vet ./...` **PASS**; `go test -race -count=1 -timeout=120s ./...` **PASS**; repeated M1 race regressions `-count=3` **PASS**.
- `FuzzReadFrame` 30s / **1,336,459 executions PASS**; `FuzzStreamDecoder` 30s / **1,258,600 executions PASS**. No crashes reported.
- `TERVON_M0_BIN=/opt/tervon-tcp-bridge/bridge` — checksum-pinned copied installed M0, subprocess loopback mTLS HTTP/2 /v1/stream: **legacy_client_to_current_server PASS** (65,536 byte-exact bytes) and **current_client_to_legacy_server PASS** (65,536 byte-exact bytes). `M1_REAL_M0=PASS`.
- `M1_VERIFY_RESULT=PASS`, `M1_VERIFY_SHA=5ae795f0a6f342724f6dc894f0e82b7f4402f867`, `M1_VERIFY_PRODUCTION_CHANGES=NONE`, `M1_VERIFY_NOTE=code_level_only_no_production_deployment`.
- `M1_REAL_V100=NOT_RUN`: published v1.0.0 executable not supplied to the opt-in test. Separate historical-release compatibility gate or documented equivalent evidence remains open. Custom non-cancelable `io.ReadCloser` limitation also remains under assessment. **M1 IN_PROGRESS**, M2 not started.


## 2026-11-17 — M1 deployment preparation requested; no production changes

- User authorized a controlled M1 installation with a backup of the working M0. The M1 code's Homeserver compatibility PASS remains tied to `5ae795f0a6f342724f6dc894f0e82b7f4402f867`.
- Added guarded installer `tests/homeserver/deploy-m1-with-m0-backup.sh`: read-only service and running SHA inspection; root-only M0 binary/unit/config backup; checksum-gated candidate; atomic replacement; restart of one explicitly selected bridge unit; running-process SHA verification; automatic and manual rollback. Corrected recovery behavior for failed restart and interrupted deployment.
- CI workflow now checks installer Bash syntax; **this does not constitute an actual install or rollback E2E test**.
- Added `docs/DEPLOY-M1-WITH-M0-BACKUP.ru.md`: Finserver first, Homeserver second, /v1 preserved during binary upgrade, /v2 enablement separate, real TCP verification mandatory.
- **Status: PREPARED_NOT_DEPLOYED**. Finserver unit/path/SHA not yet provided. Production services untouched. Installation and rollback PASS cannot be claimed yet.


## 2026-11-17 — Live-host preflight reveals actual Finserver M0 executable

- Finserver user evidence: `tervon-bridge-server.service` active, MainPID `35365`, ExecStart **`/opt/tervon-bridge-linux-amd64 -config /etc/tervon-tcp-bridge/server.json`**. `/proc/35365/exe` SHA-256 `fbc927f33cd0198329bb21ff0b40a5d4e8240bc9a92d3d62c1d9c4e4ec30fc0c` = active M0. `/opt/tervon-tcp-bridge/bridge` SHA-256 `5c813bae9cb0c68e3d5ac1e9afb0556f13cb270b369a86ae8ffc0d20adb086e2` = **inactive historical v1.0.0**.
- Homeserver `inspect` PASS: active and running `/opt/tervon-tcp-bridge/bridge` SHA `fbc927f33cd0198329bb21ff0b40a5d4e8240bc9a92d3d62c1d9c4e4ec30fc0c`, PID `2304526`.
- Separate Finserver services: echo-test and SOCKS5 production active, SOCKS5 lab failed. None are to be changed during M1.
- **Deployment safety correction** `e1e1975cbec0070e761ce4718dd1cf783b070045`, `bd062278e78a502208927f011a0f5760472521f2`: deployment script resolves live binary from selected systemd ExecStart, restricts allowed paths, checks running/on-disk hashes, backs up and restores the actual binary path. This prevents overwriting inactive v1.0.0 on Finserver.
- User directive: no new test environments or unnecessary test files on hosts; deploy directly to live servers with M0 backup and rollback, verifying real traffic. **Not yet deployed**.

- **Corrected live-path installer CI PASS:** [GitHub Actions #44046731395](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44046731395), tested script/code commit `bd062278e78a502208927f011a0f5760472521f2`, completed SUCCESS (Bash syntax, Go build/vet/unit/race, repeated M1 regression, bounded fuzz). This is **not** a live deployment/rollback test. Next: install on Finserver first, then Homeserver, verify real traffic and keep M0 backups.


## 2026-11-17 — First live Finserver M1 activation FAILED; M0 automatic rollback PASS

- User-confirmed preflight on Finserver: `tervon-bridge-server.service`, active executable `/opt/tervon-bridge-linux-amd64`, PID `35365`, disk and process M0 SHA `fbc927f33cd0198329bb21ff0b40a5d4e8240bc9a92d3d62c1d9c4e4ec30fc0c`.
- User transferred M1 candidate and confirmed matching SHA across Homeserver and Finserver. Candidate SHA `e42545d56ad05e94280466ddf92dfbcaa4955d716511f839de2eec0ae21675e5`.
- Live `install tervon-bridge-server.service` created backup `/opt/tervon-tcp-bridge/backups/m0-qVNyeBJK` and attempted restart. **`FAIL: M1 restart/verification`**. Installer automatically restored backed-up M0 and reported **`M0_AUTO_ROLLBACK=PASS`**.
- **No M1 deployment PASS**. Root cause not yet known: could be process start failure or SHA/liveness verification. Need read-only Finserver journal/systemd/process SHA evidence before modifying anything. **Do not repeat install** or update Homeserver until failure is explained and corrected.
- Separate working SOCKS5/echo services were not targeted by the installer.

- **Read-only Finserver diagnosis after rollback:** systemd `ActiveState=active`, `SubState=running`, PID `37153`, `NRestarts=0`, `ExecMainStatus=0`. Both active executable and backup `m0-qVNyeBJK/bridge` SHA `fbc927f33cd0198329bb21ff0b40a5d4e8240bc9a92d3d62c1d9c4e4ec30fc0c` — M0 restoration verified.
- Journal at 2026-11-17 08:41:26 UTC: old server shutdown reports `server graceful shutdown incomplete: context deadline exceeded`, exits 1. At 08:41:31 systemd reports `Started` new server, then `Stopping` and `Started` again due to installer's rollback. **No new M1 application failure is visible in this excerpt.** Most likely deployment false-negative: immediate `systemctl restart` return or process verification; precise cause not yet proven.
- Installer correction `f97ecd8086fe561072cd9def85a2691a7d527db8`: bounded retry to observe active MainPID and expected /proc PID SHA, 2s stable observation; `systemctl restart` nonzero is recorded but not sufficient to fail if new process is verified. Same principle for rollback. **Awaiting CI and live re-deployment verification; not yet PASS.**
- **Revised installer CI PASS:** [GitHub Actions #44049124303](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44049124303), source SHA `f97ecd8086fe561072cd9def85a2691a7d527db8`, completed SUCCESS: bash syntax, Go build, vet, unit, race, repeated regression, two bounded fuzz runs. Live Finserver retry not yet executed; `M1_INSTALL` remains FAIL until new evidence.

- **Operator command issue (not a service failure):** A proposed live retry used `bash <<'BASH'` with an internal `ssh` command reading stdin. After remote SHA check requested a password, SSH could consume the rest of Bash's heredoc, terminating the operator script before `inspect` or `install`. User output stopped at `===== VERIFY M1 CANDIDATE =====` and SSH password prompt; **no evidence of a second deployment**. Corrected operator guidance: avoid the heredoc; use `ssh -n` for read-only remote checksum commands, and `ssh ... < deploy-m1-with-m0-backup.sh` only when intentionally feeding script to remote Bash. Confirm Finserver state before retry.

- **Second live Finserver install: `M1_INSTALL=PASS`.** User confirmed `BACKUP_DIRECTORY=/opt/tervon-tcp-bridge/backups/m0-mDC0QnYM`, `OLD_SHA256=fbc927f33cd0198329bb21ff0b40a5d4e8240bc9a92d3d62c1d9c4e4ec30fc0c`, `NEW_SHA256=e42545d56ad05e94280466ddf92dfbcaa4955d716511f839de2eec0ae21675e5`, `CONFIG_CHANGED=NO`. Deployment script verified active PID SHA; no evidence yet for live /v1 stream E2E after installation. **Finserver M1 binary installed, E2E pending. Homeserver remains M0; TBF1 /v2 inactive.**

- **Live mixed-version traffic PASS confirmed by operator:** Homeserver `tervon-bridge-client.service` `ActiveState=active`, `SubState=running`, PID `2304526` (M0). Finserver `tervon-bridge-server.service` `ActiveState=active`, `SubState=running`, PID `37309`, running SHA `e42545d56ad05e94280466ddf92dfbcaa4955d716511f839de2eec0ae21675e5` (M1). User confirmed Internet connectivity over the bridge. Finserver journal included a previous `Failed with result 'exit-code'` at 08:49:13, not evidence of present process failure. **M0→M1 real-world connectivity PASS; M1 on Homeserver NOT YET INSTALLED, /v2 TBF1 NOT ENABLED.**
- **Operator requested existing site integration** `stats.xrayvless.online` for Bridge monitoring. Existing M0 plan already calls for loopback read-only `/metrics/v1` and controlled aggregation. Per sequential-module rule, defer modifying site until M1 is fully verified/closed. Audit existing site backend/auth/data flow before implementation; never expose metrics publicly or touch unrelated running services.

- **Live Homeserver M1 binary installation PASS (2026-11-17, user output):** `tervon-bridge-client.service` original binary `/opt/tervon-tcp-bridge/bridge` running M0 PID `2304526` SHA `fbc927f33cd0198329bb21ff0b40a5d4e8240bc9a92d3d62c1d9c4e4ec30fc0c` preflight PASS; M1 candidate SHA verified `e42545d56ad05e94280466ddf92dfbcaa4955d716511f839de2eec0ae21675e5`; backup `/opt/tervon-tcp-bridge/backups/m0-5RY4qRMn`; `M1_INSTALL=PASS`, `CONFIG_CHANGED=NO`. Finserver M1 installation already PASS with backup `m0-mDC0QnYM`. **Both binaries now installed M1; post-client-restart M1→M1 live TCP traffic verification NOT YET REPORTED.** `/v1/stream` retained, TBF1 `/v2/stream` not enabled. Do not declare module closure yet.

- **Live M1→M1 /v1/stream operational PASS (user-confirmed, 2026-11-17):** Homeserver `tervon-bridge-client.service` PID `2946701`, installed and running SHA `e42545d56ad05e94280466ddf92dfbcaa4955d716511f839de2eec0ae21675e5`; Finserver `tervon-bridge-server.service` PID `37309`, active/running, same SHA. Operator confirmed Internet connectivity through live bridge. Client journal `journalctl -u tervon-bridge-client.service --since "5 minutes ago" -p warning` returned `-- No entries --`. **Live M1→M1 compatibility PASS on /v1; framed /v2 NOT YET VERIFIED or enabled; no quantitative throughput benchmark.** Next: inspect exact client config path safely, backup + atomic `remote_url` change with bounded rollback, activate /v2 and verify traffic before M1 closure.

- **M1 guarded TBF1 cutover code CI PASS:** GitHub Actions [run #44051517938](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44051517938), workflow SHA `64295bd4abacc4c394335f1fcc67bf7ce5e2b05b`, completed success. New `test_activate_m1_framing.py` passed, alongside Go build/vet/unit/race/repeated M1 regressions/bounded fuzz. Live client config path confirmed `/etc/tervon-tcp-bridge/client.json`, mode client, /v1/stream. Runbook `docs/DEPLOY-M1-WITH-M0-BACKUP.ru.md` updated. **/v2 activation NOT YET RUN; live E2E remains PENDING.**

- **Live TBF1 /v2 activation and operator connectivity PASS (2026-11-17):** Homeserver `/etc/tervon-tcp-bridge/client.json` switched by committed `tests/homeserver/activate-m1-framing.py`; local unit tests 4/4 PASS; `CONFIG_BACKUP_DIRECTORY=/opt/tervon-tcp-bridge/backups/m1-framing-c6_jvgv6`; `V2_ACTIVATION=PASS_PROCESS_ONLY`, PID `2988996`. Operator separately confirmed **Internet works after /v2 cutover**. Server M1 already supports both /v1 and /v2 automatically in `prototype/bridge.go:Server.ServeHTTP`; no Finserver config change or restart is necessary. Finserver post-cutover process/journal read-only verification remains optional/pending. This is a live functional check, not a quantified throughput or long-duration stability benchmark.

- **Finserver post-TBF1 cutover check PASS (2026-11-17):** user executed `systemctl show tervon-bridge-server.service -p ActiveState -p SubState -p MainPID` and `journalctl -u tervon-bridge-server.service --since "5 minutes ago" -p warning` on Finserver. Result `MainPID=37309`, `ActiveState=active`, `SubState=running`, `-- No entries --`. Together with operator-confirmed Internet on Homeserver /v2, **M1→M1 /v2 live functional PASS**. Still no long-duration benchmark. **M1 closure pending separately documented v1.0.0 binary compatibility gate and early-EOF 500ms limitation evaluation**; do not mark all criteria complete or begin M2.

- **Monitoring integration discovery (2026-11-17, READ_ONLY):** operator inspected Homeserver `/etc/tervon-tcp-bridge/client.json`: `metrics_listen` is absent (`NOT_CONFIGURED`), so local Bridge telemetry endpoint is not enabled in the running client. Existing `prometheus.service` and `prometheus-node-exporter.service` are active; nginx site reference found in `/etc/nginx/sites-enabled/stats-temp`. HTTPS request from Homeserver to `stats.xrayvless.online` timed out after 5 seconds; site health/cause not yet determined. Unrelated `tervon-bridge-socks5-client-lab.service` is failed, not to be modified as part of monitoring. **Monitoring site integration: NOT_CONNECTED / diagnosis in progress**; no service or configuration changed. Next: read-only audit of nginx local site, loopback HTTPS, Prometheus scrape jobs and listening ports; then plan controlled metrics enablement with backup/rollback and no exposure beyond loopback.

- **External-site diagnosis correction and next gate (2026-11-17, READ_ONLY):** operator's `nginx -t` PASS confirms syntax only. Site config has HTTP/HTTPS vhosts and existing admin/infrastructure/API proxy routes; Prometheus and Node Exporter are running, Prometheus scrape jobs include xray, xray-sqlite, xray-blocked but no Bridge. Nginx listens on 80/443, Prometheus on 9090, node-exporter on 9100. A previous `curl --resolve stats.xrayvless.online:443:127.0.0.1` returned a certificate-name mismatch: **INVALID evidence for public-site certificate health** because it replaced the external route with loopback. The earlier public-name request made **from Homeserver** timed out: **not sufficient** to conclude external site is down (possible NAT/hairpin/routing). Operator requires public DNS/real external path from an outside host, without loopback substitution. Next evidence: HTTPS/TLS/DNS from Finserver over real public route, plus read-only mapping of the existing site backend. Bridge `metrics_listen` is still NOT_CONFIGURED, monitoring NOT_CONNECTED. No service changes, M1 remains active; do not claim a completed integration.

- **Public site externally verified from Finserver (2026-11-17, READ_ONLY):** public DNS for `stats.xrayvless.online` resolved; `curl` over real external route returned HTTP 200, TLS verify result 0, elapsed ~0.405s. Presented certificate subject/SAN matches `stats.xrayvless.online`, valid 2026-09-22 to 2026-12-21. Do not publish the public IP as a new hard-coded dependency. Existing nginx site proxy targets: admin routes to loopback port 8000; other site/infrastructure routes to loopback port 8893 (application identity and data ingestion not yet verified). **Public HTTPS PASS, Bridge monitoring NOT_CONNECTED, metrics_listen NOT_CONFIGURED**. Next read-only audit of listener owners, service units and existing Prometheus integration before implementing changes. No changes to running services.

- **Existing stats site backend ownership identified (2026-11-17, READ_ONLY):** nginx proxy target `127.0.0.1:8000` belongs to `hy2xray-web.service` (uvicorn, working directory `/opt/hy2xray_bot`); `127.0.0.1:8893` belongs to `xray-blocker-dashboard.service` (uvicorn/python3, working directory `/opt/xray-blocker/web-dashboard`). Both reported active/running since 2026-11-15. This identifies existing application owners, **not** yet the application integration contract. Next: inspect read-only route definitions, data sources and deployment structure without exposing tokens or private code, then design minimal additive Bridge monitoring integration. No running service touched; M1 remains active; monitoring NOT_CONNECTED.

- **Existing dashboard source layout identified (2026-11-17, READ_ONLY):** `xray-blocker-dashboard.service` remains active (main PID 2102), unit file `/etc/systemd/system/xray-blocker-dashboard.service`. Its working directory `/opt/xray-blocker/web-dashboard` includes `app.py`, `unified_monitor.py`, `aggregator.py`, `server_live.py`, active templates in `templates/`, and monitoring-related modules. Several `backup-*` directories also exist; **do not modify or treat backup copies as active code**. No dashboard source contents or route contracts were supplied yet. Next inspect only active route declarations, template integration and data source usage, without exposing secrets or modifying runtime. Bridge metrics export remains disabled, integration NOT_CONNECTED; M1 remains active.

- **Dashboard route map (2026-11-17, READ_ONLY):** operator AST-inspected active source files. Both `app.py` and `unified_monitor.py` define `GET /` and `GET /api/stats`; `unified_monitor.py` additionally defines `GET /infrastructure/`, `GET /infrastructure/api/infrastructure`, `GET /infrastructure/health`, and AI page. `aggregator.py` provides CrowdSec and Fail2Ban aggregation functions. This is a route declaration inventory only; actual systemd entrypoint, integration data contract and site templates still require read-only confirmation. No Bridge telemetry endpoint enabled, no monitoring connected, no running services changed.

- **Active stats site entrypoint and monitoring code package (2026-11-17):** operator read-only unit inspection showed the effective overridden `ExecStart` is `uvicorn unified_monitor:app --host 127.0.0.1 --port 8893 --workers 2`, not the obsolete `app:app --host 0.0.0.0` line. Active infrastructure endpoints call `collect()` and serve `templates/index.html`. The site uses shared navigation in templates. **Do not patch inactive `app.py` or backups.**
- **Bridge telemetry source changes, NOT YET VERIFIED (2026-11-17):** commits `98d8e575`, `6b321f79`, `80202496` add fixed-cardinality Prometheus text `GET /metrics` to the same private loopback listener as existing JSON `GET /metrics/v1`, with new compatibility/exposition tests. Docs updated `d0a9d090`. No new dependency or public listener. This is **CODE WRITTEN / CI IN_PROGRESS**, not PASS, not deployed, not connected to Prometheus or the site. Existing running services unchanged. Need CI PASS, then isolated Homeserver tests and read-only check of site authorization/data source before safe activation. **M1 still active**.

- **Prometheus telemetry GitHub Actions PASS (2026-11-17):** workflow `Go prototype checks` [run #44057777781](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44057777781) completed `success` for source commit `80202496e1d3fabfdbbcd9151acf301923d5cca5`. This verifies CI for code with private `/metrics` exposition and new tests, **not** Homeserver/Finserver deployment or site integration. Operator's AST inspection of active `unified_monitor.py` found `infrastructure_api` and `infrastructure_health` route handlers but no top-level `from ... import collect`; `collect()` source still unidentified. Next inspect function-level imports/assignments and resolve actual collector before editing site. Existing services unchanged; M1 active.

- **Confirmed infrastructure collector path (2026-11-17, READ_ONLY):** live `unified_monitor.py` lines 366–371 dynamically load `/opt/xray-infrastructure-monitor/infrastructure.py` via `importlib.util.spec_from_file_location("stats_infrastructure_collector", ...)`, bind it to `_infra_module`, and route `GET /infrastructure/api/infrastructure` to `_infra_module.collect()`. This is the actual collector, not `aggregator.py` or `app.py`. **Do not edit production collector before reviewing its source contracts, data shape, authorization, and rollback.** No deployment or site changes; M1 active.

- **Isolated stats adapter code package (2026-11-17, not deployed):** confirmed active `/opt/xray-infrastructure-monitor/infrastructure.py` collector exposes `collect()` with keys `generated_at/source/summary/burst_observatory/balancers/outbounds/diagnostics`. Rather than mutate its contract or live dashboard, new `integrations/stats/bridge_metrics.py` and `test_bridge_metrics.py` implement an isolated, loopback-only two-node JSON telemetry reader, with fixed field allowlist, bounded HTTP reads/timeouts, no redirects/proxy, and failure isolation. `.github/workflows/go-prototype.yml` now runs isolated Python tests for `integrations/stats/**`; integration constraints documented in `integrations/stats/README.ru.md`. Commits `e8fe484e`, `a1e7e221`, `9bc38a88`, `3de91114`. **CI for adapter is still IN_PROGRESS** at [run #44058713599](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44058713599); no PASS claim yet. Live Bridge metrics listener not enabled; no Finserver secure transport or website authorization integration. M1 remains active; running services unchanged.

- **Isolated stats adapter test step PASS (2026-11-17):** in [GitHub Actions run #44058713599](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44058713599), job #133954644962, step `Test isolated stats collector (no production services)` reported `completed/success`. Full workflow was still `in_progress` at observation; this is a **step-level PASS**, not full CI PASS, deployment, or live site integration.

- **User scope correction (2026-11-17):** stats.xrayvless.online monitors **only the local Homeserver Encapsulator**. No Finserver metrics, relay, remote collector, tunnel, or Finserver dashboard linkage. Commits `bcbbde42` and `72ef13b2` restrict `BridgeMetricsCollector` to one loopback endpoint, preserve output `nodes.homeserver` only, reject remote-node configuration, and adjust tests; `a08931a7` updates deployment docs. Intermediate CI run [#44059184537](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44059184537) failed because tests had not yet been updated at that commit; superseding run [#44059210311](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44059210311) is the verification gate, status pending at this record. No production changes. M1 active.

- **Homeserver-only stats UI code package (2026-11-17):** after user ruled out Finserver telemetry, [CI run #44059210311](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44059210311) completed **success** for `72ef13b204499f7ede6a958bd16e9fedbbb05bfb`, validating the one-node adapter and Go regression suite. Prepared embeddable `integrations/stats/bridge_panel.html`, `bridge_panel.js`, `bridge_panel.css` and `test_bridge_panel.cjs` (commits `a28677e3`, `b85bdcdf`, `7407b54d`, `46687e72`), with same-origin `/bridge/api/status` fetch, textContent-only rendering, explicit unavailable state and no remote node data. Workflow updated in `ebb1da6e` to run `node --check` and `node --test` without npm packages. **Isolated collector and JS panel test steps PASS** in [run #44075613005](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44075613005); full workflow still IN_PROGRESS at time of entry. These files are NOT deployed; API route is not yet installed. Next blocker: inspect live stats site authentication and route wiring read-only, then implement authenticated backend and isolated tests. No running services modified; M1 remains active.

- **Homeserver-only stats UI full GitHub Actions PASS (2026-11-17):** [run #44075613005](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44075613005) completed `success` for source commit `ebb1da6e6aa642e099e8a73d2587fedb71bc6c28`. Includes isolated Python collector tests, Node.js panel tests, Go build/vet/unit/race/framing/fuzz checks. The later CSS/docs commits do not change tested JS/Python/Go behavior. Plan explicitly restricts monitoring to Homeserver in `ebb5152a`. **Status: CODE WRITTEN + CI PASS; NOT DEPLOYED; authenticated site API and production integration NOT IMPLEMENTED.** Next gate: read-only audit of active site authorization and safe route attachment. M1 remains active, running services unchanged.

- **Stats-site authentication audit, read-only (2026-11-17):** operator's AST inspection of active `unified_monitor.py` reports `APP_GLOBAL_DEPENDENCIES=False` and `Depends=0` for `infrastructure_index`, `infrastructure_api`, `infrastructure_health`. Nginx `/etc/nginx/sites-enabled/stats-temp` contains `auth_basic` under `location /admin/` and `location /admin/api/`, but no displayed `auth_basic` or `auth_request` under `location = /infrastructure` or `location /infrastructure/`. This is **not a complete authentication proof**: inherited config, other middleware or external controls remain unverified. **SECURITY GATE:** do not attach `/bridge/api/status` or the panel to existing unprotected-looking infrastructure routes; review the effective nginx upstream/auth scope and application middleware first, then enforce and test explicit authentication for Bridge. Existing `/infrastructure/api/infrastructure` contract remains unchanged; no production modifications. M1 active.

- **Site routing inventory and isolated protected Bridge app (2026-11-17):** operator read-only nginx inspection confirms `/admin/` and `/admin/api/` proxy to local port 8000 with `auth_basic` and `auth_basic_user_file`; `/infrastructure/` proxies to local 8893 with no auth directive in the shown location. Instead of changing the active `unified_monitor.py`, prepared separate `integrations/stats/bridge_site.py` FastAPI app with only `/bridge/`, `/bridge/api/status`, local JS/CSS, strict CSP and no-store; unconfigured Bridge metrics return `unavailable`. Proposed `nginx-bridge.conf.example` requires `auth_basic` on `location ^~ /bridge/`, reuses the existing admin htpasswd file locally (no path/credentials committed), and proxies to a proposed **loopback-only** port 8894, subject to port availability verification. Added ASGI and static nginx template tests `test_bridge_site.py`, `test_nginx_bridge_template.py` and CI FastAPI dependency. Commits `28bf62b6`, `ff1b87d6`, `8edb33d8`, `8764a756`, `ff9d8603`, `354fcad2`. **Status: CODE WRITTEN, CI PENDING** in [run #44077311198](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44077311198); no nginx reload, service install, metrics activation, public exposure, or website deployment. Security gate: effective nginx auth/route review, `nginx -t`, and external unauthenticated 401 / authenticated 200 before release. M1 remains active.

- **Isolated protected Bridge site CI PASS (2026-11-17):** [GitHub Actions run #44077311198](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44077311198) completed `success` on commit `ff9d8603528685601498a7310aaf5414b07dfd32`. Checks included Python collector, FastAPI ASGI routes/CSP, nginx auth-template static tests, Node.js panel, Go build/vet/unit/race/framing/fuzz. Later `354fcad2` and `538f9bb1` changed only documentation. **Code + CI PASS; NOT DEPLOYED; nginx Basic Auth has NOT been tested on live site; local 8894 port and Bridge metrics listener not verified.** Do not claim public-site integration or M1 closure. No production service touched.

- **Homeserver stats preflight and service template (2026-11-17):** operator read-only check `ss -H -ltn '( sport = :8894 )'` produced no listeners; existing site venv reports `fastapi=0.139.0`, `uvicorn=0.50.2`. This is **PRECHECK PASS for current port/dependency snapshot**, not installation. Added `integrations/stats/tervon-bridge-stats.service.example` (separate loopback-only Uvicorn worker, DynamicUser, strict filesystem/network sandbox), `test_service_template.py`, and tightened `nginx-bridge.conf.example` so even exact `/bridge` redirect requires Basic Auth, with regression test. Commits `b4dbd6c0`, `3c8adfea`, `515822ef`, `38a695e2`; README updated in `263f18e3`. GitHub Actions [run #44077891019](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44077891019) **IN_PROGRESS** at recording. Before deployment still need verify DynamicUser file/venv access, systemd IPAddressDeny support, actual htpasswd path privately, and agree backup/rollback. No running service or nginx changed; M1 remains active.

- **Stats deployment staging and rollback plan (2026-11-17):** operator's read-only Homeserver output confirms port 8894 had no listener and existing venv includes FastAPI 0.139.0 / Uvicorn 0.50.2. Added sandboxed `tervon-bridge-stats.service.example`, protected exact `/bridge` redirect and `/bridge/` nginx template, unit/template regression tests, and approval-gated `integrations/stats/DEPLOYMENT.ru.md` describing staged isolated app, external Basic Auth 401/200, rollback, and separate metrics activation. Commits `b4dbd6c0`, `3c8adfea`, `515822ef`, `38a695e2`, `263f18e3`, `ddb1b143`. GitHub Actions [run #44077891019](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44077891019) was still in progress at time of entry. **No files installed on Homeserver, no systemd unit started, no nginx changes/reload, no Bridge metrics activation.** M1 active.

- **Stats sandbox unit and auth template CI PASS (2026-11-17):** [run #44077891019](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44077891019) completed `success` for source commit `38a695e2cf6d3c98e5cea159a35cb139834d6550`, including Python stats/unit/nginx tests, Node.js UI tests and Go build/vet/unit/race/framing/fuzz. Later changes `263f18e3`, `f3715b3b`, `ddb1b143`, `8e094ef7` were documentation only. **Status CODE + CI PASS, staging runbook ready; NOT INSTALLED / NOT DEPLOYED / NOT LIVE-VERIFIED.** Next requires operator approval for new service and nginx changes; protect running services. M1 remains active.

- **2026-11-17 — Homeserver-only stats deployment gate, code and tests:** operator approved continuing staged deployment preparation. Review of nginx template found a security error: `return 308` in exact `location = /bridge` executes during rewrite phase before Basic Auth, so the redirect could be served without authentication. Fixed exact path to authenticated `proxy_pass http://127.0.0.1:8894/bridge/` and updated regression test (`0f93e339`, `228f643b`). Added **read-only** `integrations/stats/preflight_homeserver.py` and mocked tests (`a6a21f6b`, `75fe4a02`), then changed Git status invocation to `git --no-optional-locks` (`56705af5`, `882618fd`) to avoid optional index writes. Preflight checks SSH origin, clean M1 branch, port 8894, no conflicting unit/app, FastAPI/Uvicorn, systemd >=235 and admin htpasswd existence without printing the path or contents. Updated `integrations/stats/README.ru.md`, `DEPLOYMENT.ru.md` and plan. **Status: preflight code written and CI in progress; no on-host preflight output yet; no install, nginx reload or running service modification.** Next: obtain Homeserver read-only preflight output, resolve blockers, then stage new isolated service without touching active monitoring. M1 active.

## 2026-11-17 — обязательная коррекция статуса M1 по замечанию оператора

- **ФАКТИЧЕСКОЕ ТЕКУЩЕЕ СОСТОЯНИЕ:** M1 **установлен и работает на обоих узлах**, Homeserver и Finserver. Исполняемый бинарник на обоих узлах ранее подтверждён SHA-256 `e42545d56ad05e94280466ddf92dfbcaa4955d716511f839de2eec0ae21675e5`. Клиент Homeserver переключён на TBF1 `/v2/stream`; оператор подтвердил работающий интернет, Finserver подтверждён active/running без предупреждений на финальной проверке. **M1 DEPLOYED + LIVE CONNECTIVITY PASS**. Это не равно релизу v1.0.1 или длительному performance benchmark.
- **УСТАРЕВШЕЕ ТРЕБОВАНИЕ СНЯТО:** исторический опубликованный v1.0.0 и M0 **не являются рабочими версиями**; отдельный opt-in тест `M1_REAL_V100` и совместимость с историческим бинарником v1.0.0 **не являются обязательными критериями M1**. Старые строки `NOT_RUN`, `NOT_DEPLOYED`, `Production: NONE`, `M0 работает` и «M1 требует проверки v1.0.0» в предшествующих хронологических записях относятся только к прошлым контрольным точкам, **не к настоящему состоянию**.
- **Открытый инженерный вопрос отдельно от эксплуатационного PASS:** оценка 500ms fail-closed при раннем backend EOF и ограничений завершения для кастомного `io.ReadCloser`, игнорирующего `Close`. Это не доказанная неисправность работающего M1; не выдавать этот вопрос за отсутствие установки или сетевого PASS. Состояние формального закрытия M1 определяется именно этой проверкой, не v1.0.0.
- **Сайт:** подготовленные исходники Bridge stats ещё **не развёрнуты**; работающий сайт и Bridge в ходе этой коррекции **не изменялись**.
- **Причина коррекции:** предыдущий ответ ошибочно использовал ранние исторические критерии как актуальные и повторно потребовал v1.0.0 compatibility. `docs/DEVELOPMENT_PLAN.md` обновлён с явным актуальным статусом и пометками исторических записей. При следующей работе читать **последнюю** контрольную точку и подтверждённое фактическое состояние прежде, чем назначать gate.

## 2026-11-17 — оператор разрешил подключение Homeserver-only Bridge Stats

- **Scope:** только локальные агрегированные метрики Homeserver Encapsulator, не Finserver. M1 уже работает на обоих узлах с TBF1 `/v2/stream`; не устанавливать M1 повторно и не трогать действующие службы.
- **Подготовлено:** `integrations/stats/install-site-service.sh` (commit `a0a10fc0`) — отдельный `sudo bash` процесс, read-only preflight, проверка конфликтов, установка только `/opt/tervon/bridge-stats` и нового systemd unit, `systemd-analyze verify`, старт **только** `tervon-bridge-stats.service`, проверка loopback API с `status=unavailable` и строгая проверка привязки 127.0.0.1:8894, автоматический rollback новых файлов/unit при FAIL. Не меняет nginx, существующий dashboard или Bridge.
- **Тесты и CI:** `test_install_site_service.py` (commit `e078de9c`) проверяет ограничения и откат статически, workflow `bash -n` (commit `83abe30f`); commit `faf382c0` усилил проверку всех слушателей только на IPv4 loopback. [CI run #44080564990](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44080564990) был **IN_PROGRESS** при фиксации; проверка shell syntax, Python и JS на момент фиксации прошли. `DEPLOYMENT.ru.md` обновлён (commit `448b217e`).
- **Статус:** CODE PREPARED, **НЕ УСТАНОВЛЕНО** на Homeserver, нет вывода `NEW_SERVICE=PASS`, nginx не менялся, реальные метрики ещё не включены. Следующий шаг — дождаться CI PASS, оператор запускает из чистого SSH checkout отдельный скрипт; при FAIL — блокер и диагностика, не переходить к nginx.

- **Installer CI PASS (2026-11-17):** [GitHub Actions #44080564990](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44080564990), source SHA `faf382c080afb52e8b567cec21c5fd0b95d2248b`, completed **SUCCESS**. `bash -n` installer, Python tests including static install safety, JS panel, Go build/vet/unit/race/regression/fuzz all PASS. Later changes `448b217e`, `88dbb194`, `85888372` documentation only. **Installer is verified in CI, but NOT EXECUTED on Homeserver.** The new service, nginx routing and local metrics remain uninstalled/unconfigured until actual operator command and verified outputs.

## 2026-11-17 — первый запуск Homeserver stats installer: BLOCKED на read-only preflight

- **Оператор выполнил** SSH `git fetch` + `merge --ff-only` ветки `work/m1-framing`: локальный HEAD `67b01d81ea47e3a465a77160f19b2aed536e040c`, checkout чистый, SSH origin подтверждён.
- **Preflight:** `GIT_SSH_REMOTE=PASS`, `GIT_BRANCH=PASS`, `GIT_HEAD=PASS`, `GIT_CLEAN=PASS`, `PORT_8894=PASS`, `NEW_UNIT=PASS`, `APP_DIRECTORY=PASS`, `PYTHON_DEPS=PASS`, `SYSTEMD_VERSION=PASS`; **`ADMIN_HTPASSWD=BLOCKED`** («literal admin auth file not verified»); `DYNAMIC_USER_ACCESS=CHECK`.
- **Installer:** `INSTALL_STATUS=ROLLBACK (exit=2)`. Блокировка произошла **до** создания каталога/unit и запуска нового процесса. Действующие nginx, M1 Bridge и dashboard не изменялись. Причина может быть связана с чрезмерно узким распознаванием `location /admin/` в preflight; отсутствие auth не доказано.
- **Следующий шаг:** только read-only локальная диагностика фактического nginx `location`, `auth_basic`, `auth_basic_user_file` без раскрытия паролей и пути к htpasswd; затем исправление preflight по доказанному формату и тесты. Не обходить auth gate, не устанавливать сервис и не переходить к nginx до PASS.

## 2026-11-17 — устранение ложного ADMIN_HTPASSWD BLOCKED (после read-only диагностики)

- **Homeserver read-only факт:** nginx site config существует, имя `stats.xrayvless.online` найдено; обнаружены 2 location с `/admin`, рядом с обоими `auth_basic` и `auth_basic_user_file`, без `include`. Старый preflight exact regex `location\\s+/admin/\\s*\\{...` дал `LOCATION_MATCH=False`, `AUTH_DIRECTIVE_MATCH=False`; это доказывает **несоответствие распознавания location**, но не отсутствие auth и не конкретный nginx modifier.
- **Исправление:** commit `e2bf9937` — preflight распознаёт `location /admin/`, `location ^~ /admin/`, `location = /admin/`, требует явный `auth_basic` не `off`, буквальный абсолютный существующий regular `auth_basic_user_file`; при неподдерживаемом синтаксисе остаётся BLOCKED. Не печатает путь, пароль или хеш.
- **Регрессии:** commit `638db4b4` — mocked tests на обычный, `^~`, `=` и fail-closed случаи: `off`, закомментированный auth, отсутствующий/переменный файл, другой location, вложенные блоки. GitHub Actions #44082064519 **IN_PROGRESS** при записи. Нет изменений работающих nginx, M1, dashboard; **установка по-прежнему NOT DEPLOYED**. Следующий шаг после CI PASS — повторить read-only preflight на Homeserver, при PASS разрешить отдельную установку новой stats-службы; при BLOCKED выяснить фактический синтаксис без раскрытия секретов.

## 2026-11-17 — повторный Homeserver preflight после исправления nginx matcher: BLOCKED

- **Оператор:** fast-forward `67b01d8 → b35a49f`; повторный read-only `preflight_homeserver.py` выполнен из чистого SSH checkout `work/m1-framing`.
- **PASS:** SSH origin, branch, HEAD, clean tree, port 8894 free, new unit absent, app directory free, Python deps, systemd version. **BLOCKED:** `ADMIN_HTPASSWD` («explicit /admin/ auth and regular literal file not verified»). `DYNAMIC_USER_ACCESS=CHECK`. **NO_CHANGES=PASS**. Инсталлятор **не запускался повторно**.
- **CI:** GitHub Actions #44082064519 для исправления `638db4b4` завершился **SUCCESS**, но on-host preflight всё ещё BLOCKED: CI PASS **не** означает совместимость с фактическим nginx синтаксисом. Предыдущее предположение о `^~`/`=` как причине не доказано.
- **Блокер:** нужна ещё одна ограниченная read-only диагностика: форма `location` и фактические auth directives/файл в соответствующих блоках; не выводить реальные пути, учётные данные или хеши. До выяснения причины не изменять nginx, Bridge, dashboard, не обходить auth gate.
