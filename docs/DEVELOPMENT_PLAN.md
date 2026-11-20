# Tervon TCP Bridge — единый план разработки (v1.0.1 → Android)

**Статус:** утверждённый порядок работы, 2026-11-17. **Единственный рабочий план для следующих изменений.**
**Репозиторий:** `git@github.com:tervon-software/Tervon-TCP-Bridge.git`.
**Важно:** пользователь разрешил публикацию исходников 2026-11-17. Go-прототип, CLI, тесты и workflow перенесены в публичную ветку `main` коммитом `1528ebd76309aa5330e4cde88a183e81f4cce074` из ветки `docs/working-prototype-release` (`f04280c77e54dc39ac29660a397925b0989fb625`). Публикация кода **не равна** подтверждённому PASS, готовности релиза или воспроизводимой сборке.

## Цель

Развивать существующий Tervon TCP Bridge, сначала Linux Encapsulator/Decapsulator, затем переносить **проверенные** наработки в TervonVPN Android. Первый этап — v1.0.1 с адаптивными Padding/Jitter для устойчивых длительных потоков, без постоянного изменения коротких и интерактивных соединений. Мост не анализирует содержимое VPN-протокола. Абсолютная «невидимость» для DPI не обещается; эффект проверяется измерениями.

## Неизменные правила работы

1. **Один модуль за раз:** выбрать модуль → сформулировать критерии готовности → реализовать → тестировать → зафиксировать commit и доказательства → закрыть модуль → только потом следующий. Не переключаться между незавершёнными модулями.
2. **Пакетная разработка:** объединять в один логический пакет изменения **внутри текущего модуля** (код + тесты + документация + проверка), если это не мешает отладке и ревью. Независимые задачи одного модуля можно выполнять параллельно; интеграция и закрытие — единые.
3. **GitHub только SSH** для Git: `git@github.com:tervon-software/Tervon-TCP-Bridge.git`. Разработка и коммиты в GitHub; сборки при необходимости на Homeserver. **Работающие службы не трогать.**
4. Использовать готовые, поддерживаемые библиотеки для TLS/HTTP2/криптографии; не реализовывать криптографию самостоятельно. Не вводить несколько библиотек для одной задачи без обоснования.
5. Сначала проверять текущий код и фактическое состояние; не придумывать интерфейсы и параметры. Изменения протокола должны иметь version/capability negotiation, отказ на несовместимых версиях и тесты совместимости.
6. Исходники Bridge разрешено публиковать в этом репозитории; **не публиковать** секреты, реальные IP, сертификаты, UUID и посторонние приватные проекты. Для публичных документов использовать «главный сервер каскада» вместо имени конкретного домашнего хоста.
7. Каждая проверка отвечает на конкретный вопрос и фиксирует команду, commit, среду, результат PASS/FAIL и ограничения. «Код написан» ≠ «испытан» ≠ «выпущен».
8. Если текущий модуль заблокирован, явно записать блокер; **не перепрыгивать** на следующий без согласования.

## Архитектура и границы

- **Android (позднее)** → Bridge Encapsulator → **главный сервер каскада** (Bridge Decapsulator, Xray, маршрутизатор, блокировщик рекламы).
- **Главный сервер каскада** → Bridge Encapsulator → **зарубежный балансировщик** (Bridge Decapsulator, приём/передача и балансировка нагрузки) → зарубежные выходные ноды.
- Текущий Bridge переносит **TCP**, не нативный UDP/Hysteria2. Android-интеграция только после проверки Linux-моста.
- Padding — согласованное транспортное кадрирование с удалением служебных байтов до backend. Jitter — управление отправкой, не изменение payload. Не вставлять байты в непрозрачный TCP-поток.
- Адаптация по статистике размеров, скорости и интервалов **без DPI-инспекции содержимого**. Продолжительный устойчивый поток может активировать обработку; интерактивный/короткий — OFF. Видеотрафик может быть порционным, поэтому не классифицировать по названию приложения.
- Гистерезис и отдельные направления передачи; немедленное снижение/отключение при росте RTT, очередей, потерях/перегрузке. Жёсткие лимиты задержек и накладного трафика. Значения лимитов утверждаются по результатам тестов, а не принимаются как доказанные.

## M0 — перенос исходников в main (2026-11-17)

- **Источник:** `docs/working-prototype-release`, commit `f04280c77e54dc39ac29660a397925b0989fb625`.
- **Цель:** `main`, перенос commit `1528ebd76309aa5330e4cde88a183e81f4cce074`; сохранены текущие README, документация, примеры, systemd и план.
- **Добавлено:** 31 файл: `prototype/` (Go bridge, CLI, unit/integration tests), `tests/homeserver/` (изолированные тестовые сценарии), `.github/workflows/go-prototype.yml`.
- **Проверка:** GitHub подтвердил создание commit и успешное обновление `main`; выборочная проверка основных исходников и скриптов на очевидные секреты не обнаружила приватных ключей/токенов. **Полный security review ещё нужен.**
- **Статус M0:** `IN_PROGRESS`. Источник найден и перенесён, но сборка, тесты на **точном commit** и воспроизводимость бинарника v1.0.0 пока не подтверждены. Не объявлять PASS.
- **Дальше только M0:** проверить дерево после переноса, карту функций, конфигурацию и протокол, выполнить `go test`, `go test -race`, `go vet`, сборку и зафиксировать результаты с точным SHA. Службы не затрагивать.

## Очередь модулей: строго по порядку

### M0 — Инвентаризация, мониторинговый интерфейс, базовая линия и эксплуатационная проверка [ЗАВЕРШЁН: эксплуатационный PASS, ограничения ниже]
**Сделать одним пакетом:** инвентаризировать текущий Go-код, TLS/mTLS/HTTP2, JSON, протокол и тесты; сверить исходники с v1.0.0; реализовать безопасный экспорт базовых метрик и тесты; собрать воспроизводимый бинарник; провести изолированные тесты, затем согласованное развёртывание и сквозную проверку реального TCP-трафика. Если связи нет — диагностировать и исправлять в M0 до подтверждённого результата.
**Мониторинг:** Bridge предоставляет только локальный read-only интерфейс сбора статистики (по умолчанию 127.0.0.1, не публичный endpoint). Отдельный контролируемый коллектор передаёт агрегированные данные на `stats.xrayvless.online`; интеграция с сайтом требует аудита его текущей архитектуры и отдельного безопасного шага без нарушения действующих служб. Не открывать метрики наружу, не экспортировать сертификаты, ключи, идентификаторы клиентов, IP-адреса или содержимое потоков. Для сайта предусмотреть строгую аутентификацию и разграничение доступа.
**Базовые метрики M0:** build version/commit, uptime, readiness, активные/завершённые/отклонённые TCP-сессии, полезные байты по направлениям, текущая скорость на окнах измерения, ошибки по ограниченному набору причин (TLS handshake, auth, backend dial, HTTP status, timeout, cancellation, shutdown), число активных HTTP/2-сессий при доступности точного счётчика, лимиты и занятость, задержки установки соединений (histogram p50/p95/p99), backend availability по безопасному идентификатору, runtime memory/goroutines. Трафик измерять отдельно как полезные байты и накладные байты только там, где они достоверно наблюдаемы. Не называть полезные байты полным сетевым трафиком. TLS version/cipher/ALPN/mTLS status — только проверенные, агрегированные наблюдения; не выдавать конфигурацию за факт успешного соединения.
**Будущие поля (M2/M3):** Padding и Jitter в состояниях NOT_IMPLEMENTED/OFF/ACTIVE/ERROR, отдельно upload/download; padding bytes/overhead%, jitter delay and disable reason, budgets, hysteresis transitions. Пока функциональность не реализована, не показывать ACTIVE. RTT/потери/ретрансляции отображать только при наличии корректного источника измерений и указывать слой измерения.
**Надёжность телеметрии:** никакой блокировки data plane из-за мониторинга, ограниченные метки и память, атомарные счётчики, отсутствие бесконечной кардинальности, ограничение частоты сбора, отдельные тесты race/нагрузки и отказа коллектора; мониторинг не должен менять байты TCP-потока.
**Готово, когда:** исходники и commit подтверждены; есть карта модулей, базовые тесты и перечень рисков; мониторинговый контракт реализован и протестирован; бинарник собран; изолированные проверки PASS; после отдельной безопасной установки подтверждена реальная двусторонняя TCP-связь и наблюдаемость. Если развёртывание пока невозможно, M0 остаётся IN_PROGRESS/BLOCKED, а не PASS.

### M1 — Версионирование и безопасное кадрирование [ТЕКУЩИЙ]
**Пакет:** формат кадров и парсер с ограничениями длины/памяти; negotiation; совместимость OFF с v1.0.0 либо явный безопасный отказ; fuzz/negative tests; байтовая идентичность full duplex, half-close, EOF, cancellation.
**Готово, когда:** все тесты и совместимость подтверждены на конкретном commit.

### M2 — Адаптивный Padding
**Пакет:** пассивные метрики, детектор устойчивых потоков, гистерезис, бюджеты overhead, раздельная адаптация направлений, OFF для интерактивного трафика; unit/integration tests.
**Готово, когда:** нулевая порча данных, измерены overhead и ресурсы, отсутствует деградация на коротких потоках сверх утверждённых пределов.

### M3 — Адаптивный Jitter
**Пакет:** ограниченное управление отправкой, отмена задержек при интерактивности/перегрузке, fairness между потоками, метрики, тесты RTT/потерь/очередей.
**Готово, когда:** доказаны границы задержки, нет starvation и зависших горутин/соединений.

### M4 — Надёжность и соответствие HTTPS/2
**Пакет:** управление жизненным циклом соединений, тайм-ауты, TLS/HTTP2 compliance, лимиты ресурсов, корректная обработка сетевых отказов; контроль утечек и тесты под нагрузкой. Не подменять этот модуль недоказанными заявлениями о DPI-невидимости.
**Готово, когда:** есть воспроизводимые PASS для отказов, нагрузки и корректного завершения.

### M5 — Измерения, интеграция, выпуск v1.0.1
**Пакет:** сравнение v1.0.0 и v1.0.1 на идентичных маршрутах; короткие запросы, видеообразный ровный поток, порционный поток, скачивание, RTT/потери/переключения сети; реальный TCP VPN; документация, JSON, release notes, SHA256SUMS.
**Готово, когда:** подтверждены точные результаты, нет регрессий, создан проверенный релиз. До этого не объявлять v1.0.1 выпущенным.

### M6 — Android (только после M5)
**Пакет:** аудит TervonVPN VpnService и ядра, поддерживаемый способ переиспользования Bridge, защищённое хранение ключей, защита сокетов от VPN-loop, переключатель «Обход DPI», тесты мобильных сетей. Без изменения действующего APK до отдельного решения.

## Бэклог после v1.0.1 (не выполнять вместо текущего модуля)

- TLS fingerprint consistency и HTTP/2 protocol compliance — измерять признаки, не обещать неотличимость от браузера.
- Connection lifecycle/recovery, TLS session resumption, обработка смены сети.
- Управляемое HTTP/2 multiplexing и оценка head-of-line blocking.
- DNS/route leak protection, path MTU, буферы, безопасный fallback.
- Реальный HTTPS-сервис/reverse proxy на главном сервере — отдельный проектный и эксплуатационный этап, без вмешательства в работающие службы.
- QUIC/HTTP3/UDP не добавлять в текущий TCP scope без отдельного решения.

## Уточнение M0 — 2026-11-17, решение пользователя

- **Проверенный HEAD до записи:** `08fa345a4d492ed8dad3397533bbe014bc40990a`.
- **Подтверждено чтением кода:** `prototype/bridge.go` реализует `Server.ServeHTTP`, `Client.HandleContext`, `Client.ListenAndServe`; `prototype/cmd/bridge/main.go` настраивает TLS 1.3, mTLS, HTTP/2, JSON и лимиты. Специального мониторингового интерфейса в прочитанных файлах нет. Padding/Jitter не подтверждены как реализованные.
- **Решение:** дополнить M0 экспортом статистики для будущего подключения `stats.xrayvless.online` и закрывать M0 только после сборки, изолированных тестов, безопасного ввода в эксплуатацию и подтверждения двусторонней связи.
- **Статус:** `IN_PROGRESS`; это фиксация требований и первичного аудита, **не реализация метрик, не PASS и не развёртывание**. Существующие службы не затронуты.
- **Ближайший шаг M0:** разработать контракт и базовые инструменты телеметрии с тестами, затем проверить сборку и сценарии в изоляции. Перед установкой получить фактическую схему сервера/сайта и план безостановочного переключения.

## Формат отчёта после каждого пакета

```text
Модуль: M?
Статус: NOT_STARTED / IN_PROGRESS / BLOCKED / PASS
Репозиторий/ветка/commit:
Изменённые файлы:
Что реализовано:
Какие вопросы проверяли:
Команды и среда тестов:
Результаты и ограничения:
Безопасность/совместимость:
Влияние на работающие службы: NONE (либо явно согласованное)
Следующий шаг В ЭТОМ ЖЕ модуле:
Условие закрытия модуля:
```

## Протокол сохранения контекста для ассистента

1. **В начале каждого нового чата** открыть этот файл в GitHub, проверить текущую ветку и commit, последний отчёт; не полагаться только на память диалога.
2. **После каждого значимого изменения** обновить здесь статус текущего модуля и/или журнал `docs/DEVELOPMENT_LOG.md` (если он существует), закоммитить с кодом или отдельным docs-коммитом.
3. Всегда различать: утверждённое требование, гипотезу, реализованное, протестированное, выпущенное. Не объявлять PASS без результата.
4. Не переключаться на другой модуль при незакрытом текущем. При блокере остановиться, записать его и запросить ровно недостающий ресурс/решение.
5. **Источник истины — этот файл и проверенный Git commit**, а не краткая память ChatGPT. В память сохранить только короткое правило-переходник к документу.

## Короткий промпт для Memory / Personalization

> Для проекта Tervon TCP Bridge единственный обязательный план разработки — `docs/DEVELOPMENT_PLAN.md` в `git@github.com:tervon-software/Tervon-TCP-Bridge.git`. В начале каждой работы открывай этот файл, сверяй ветку, commit, текущий модуль, критерии готовности и последний результат. Работай строго над одним модулем до завершения и доказанного PASS; не перепрыгивай на другие. Используй пакетную разработку внутри модуля (код + тесты + документация), когда это безопасно. После изменений обновляй статус и коммить. Git только SSH; сборки при необходимости на Homeserver; работающие службы не трогать. Не путай планы, код, тесты и релизы; не придумывай результаты. Сначала Linux Bridge v1.0.1 с адаптивными Padding/Jitter, затем Android. Если контекст неполон, восстанавливай его из Git, а не из догадок.


### M0 — подтверждённый локальный E2E расширенной телеметрии (2026-11-17)

- Два независимых процесса Bridge: PASS.
- Двусторонняя передача 1 МиБ в каждом направлении: PASS.
- TLS 1.3, ALPN h2, mTLS и TCP half-close: PASS.
- Расширенный JSON-контракт `/metrics/v1` на обеих сторонах: PASS.
- Сервер: 2 успешных backend dial, 2 измерения задержки, 2 подтверждённых TLS/HTTP2/mTLS-запроса.
- Race-тесты, go vet, сборка и восстановление тестового бинарника: PASS.
- Build metadata пока `dev/unknown`; выпуск не подтверждён.
- Реальные двухсерверные испытания не выполнены.
- **M0 остаётся IN_PROGRESS. M1 не начинать.**

## Фактический результат M0 — 2026-11-17

- Go unit/race tests: PASS для всех трёх пакетов.
- `go vet ./...`: PASS.
- Локальный E2E: PASS, двусторонняя побайтовая передача 1048576 байт
  в каждом направлении, включая half-close.
- TLS 1.3, HTTP/2 ALPN h2, mTLS: локальные проверки PASS.
- Локальный контракт телеметрии и счётчики полезных байтов: PASS.
- Finserver: основной и SOCKS5 Decapsulator обновлены, оба процесса
  проверены по SHA-256: PASS.
- Главный сервер каскада: Encapsulator обновлён, процесс проверен
  по SHA-256: PASS.
- SHA-256 установленного кандидата:
  `fbc927f33cd0198329bb21ff0b40a5d4e8240bc9a92d3d62c1d9c4e4ec30fc0c`.
- Реальная эксплуатация: пользователь подтвердил доступность интернета,
  Telegram и YouTube, хорошую отзывчивость без прежних фризов.
- Резервные копии v1.0.0 сохранены на обоих серверах.
- Ограничение: отдельные измерения производительности, потерь,
  RTT и проверка локального endpoint метрик на production
  не представлены. Эксплуатационное подтверждение не является
  измеренным performance benchmark.
- Изменения M0 не означают выпуск v1.0.1.
- Решение: перейти к M1 без изменения работающих служб.


## M6 — Android ingress Decapsulator: operator-approved implementation checkpoint (2026-11-20)

**This checkpoint supersedes the historical M6 HOLD / NOT_STARTED wording above for the newly approved scope only.** Linux v1.0.1 release and existing Russia → Finland Bridge were confirmed operational by the operator. The operator approved a separate Android-facing Decapsulator on the Russian server, independent of the live Encapsulator, with VLESS TCP port 30000 remaining open for direct connections. Android DPI bypass is opt-in/manual; no Wi-Fi/mobile auto-switching. Existing live Xray/Bridge/Stats services must remain untouched.

- **Working branch:** `work/m6-android-ingress` (from `main`); **current module M6**, implementation in progress.
- **Confirmed host inventory from operator:** Xray listening `*:30000`; `tervon-bridge-client.service` active; `tervon-bridge-stats.service` and `tervon-bridge-fin-metrics-tunnel.service` active; Docker container `xray-exporter` running.
- **Implemented in GitHub (non-deployed):** `integrations/android-ingress/Dockerfile`, `compose.yaml`, `server.json.example`, `README.ru.md`. Opt-in Compose profile, loopback-only host port binding, isolated Decapsulator routes to `host.docker.internal:30000`, nonroot read-only container, explicit placeholder authorized fingerprint. **No server runtime changed.**
- **Verified:** all four files read back from GitHub; JSON example parses, role is server and dummy ACL fingerprint length is 64 hexadecimal chars. **Not verified:** Docker Compose validation/build, backend reachability, client enrollment, mTLS handshake, Android switching, or E2E TCP. Do not report PASS for these.
- **Known blocker:** Bridge server checks `authorized` certificate SHA-256 fingerprints at startup. The intended first-launch enrollment/automatic certificate issuance cannot work simply by distributing JSON; an operator-controlled, authenticated way of provisioning and updating the new Decapsulator's authorization is needed without altering the running Russia → Finland connection. An unauthenticated public certificate-issuing API is not acceptable.
- **Next within M6:** verify the free ingress port, Docker bridge gateway and access to existing VLESS without changing services; then test isolated container and agree minimal safe enrollment authorization. Use the chosen subscription SNI only when it actually names the intended discovery endpoint; do not assume third-party camouflage domains support Bridge.
- **User-facing UI text:** «Если в вашем регионе нет блокировок VPN, рекомендуем оставить обход DPI выключенным для экономии заряда аккумулятора.»


### M6 — read-only host network preflight (2026-11-20)

Operator ran `docker network inspect bridge --format ...` and `ss -lntp '( sport = :18443 )'` on Russian Homeserver. Actual Docker default bridge subnet: `172.17.0.0/16`, gateway `172.17.0.1`; **no TCP listener** on candidate port `18443` at observation time. Prior observation confirmed Xray listening `*:30000`. No configuration or running service was changed.

**Scope of finding:** candidate port and Docker gateway known; *container-to-host TCP reachability remains NOT_TESTED*. The Compose template maps only `127.0.0.1:18443` on the host, not a public HTTPS endpoint. Do not claim Android can connect until an approved ingress exposure and mTLS registration path are implemented and tested. Next focused check: validate Docker Compose rendering in an isolated clone, then test temporary isolated container-to-Xray connection without touching live services.


### M6 — Docker Compose static validation PASS (2026-11-20)

Operator cloned `work/m6-android-ingress` over SSH to `/opt/tervon-tcp-bridge-m6-android-ingress` on Homeserver and ran `docker compose -f integrations/android-ingress/compose.yaml --profile android-ingress-manual config --quiet` with the intended private config directory environment variable. **Actual result: `COMPOSE_EXIT_CODE=0` (PASS).** No container was built or started, no external port was opened, and active services were not altered by these commands. Next gate: isolated Docker image build, then separately scoped container-network connectivity. A Compose syntax PASS is not an end-to-end or security PASS.


### M6 — isolated Android ingress image built (2026-11-20)

Operator supplied output after an image build on Russian Homeserver: `docker image inspect tervon-bridge-android-ingress:local` returned `IMAGE_ID=sha256:33fc8f5f41e3050823326e7db0f14e7f6ba24352d327f70d01cdaeea849ecc4b`, `ARCH=amd64`, `SIZE=5890065` bytes. `docker image ls` confirms the named image (Docker reported 20.4 MB disk usage and 5.89 MB content size). **BUILD ARTIFACT PRESENT / PASS** as confirmed by image inspect, not yet executable-runtime or E2E PASS. No running Bridge or Xray units modified by image-only build. Next test: ephemeral isolated Docker bridge-network TCP reachability to `host.docker.internal:30000` via gateway `172.17.0.1`, without starting ingress Decapsulator, sending VPN payload or touching live services.


### M6 — Docker-to-VLESS TCP reachability PASS (2026-11-20)

Operator executed an ephemeral `alpine:3.20` container using `docker run --rm --network bridge --add-host host.docker.internal:host-gateway ... nc -z -w 3 host.docker.internal 30000` on Homeserver. Image was pulled for the isolated check. **Observed result: `DOCKER_TO_VLESS=PASS`.** This confirms that a container in the default Docker bridge network can establish TCP to the existing Xray VLESS listener at port 30000 via Docker host gateway `172.17.0.1`. It does **not** yet verify VLESS payload forwarding, Bridge TLS/mTLS, HTTPS/2, or the Android application. No existing Bridge/VLESS service restart or configuration change was involved. Next M6 gate: separate Android-facing PKI, private config, isolated Decapsulator startup and local-only mTLS E2E with non-disruptive rollback. Never reuse the operational Russia–Finland Bridge keys; never publish a public unauthenticated certificate issuance API.


### M6 — short-lived lab PKI fixture prepared in GitHub (2026-11-20)

- Added `integrations/android-ingress/prepare-local-fixture.sh`, kept strictly separate from live Russia–Finland PKI; generates one-day EC P-256 server/client test certificates, temporary CA, client DER SHA-256 authorization fingerprint, server/client JSON, and minimum necessary container read permissions, **outside Git**. It refuses existing targets, requires an explicit absolute path and does not start Docker or touch running services. This is a **lab-only client private key** for checking mTLS; real Android enrollment must generate its own private key locally and submit only a public key/CSR.
- Updated `integrations/android-ingress/README.ru.md` with the isolation and run criteria. GitHub file readback succeeded. **NOT_RUN:** shell syntax verification on Homeserver, actual OpenSSL key/certificate creation, Docker startup, mTLS E2E and Android. No PASS claimed for those gates.
- Current next action within M6: operator pulls the isolated working branch and checks script syntax, then generates lab PKI in a newly absent `/opt/tervon-bridge-m6-lab` directory. Only after `M6_LAB_PKI=PASS` begin loopback-only server startup tests; never publish ingress or modify the live Bridge without a separate cutover.


### M6 — local fixture creation PASS (2026-11-20)

Operator synchronized `work/m6-android-ingress` to the Homeserver checkout and executed `bash -n integrations/android-ingress/prepare-local-fixture.sh` (observed `M6_SCRIPT_SYNTAX=PASS`) and then generated the isolated test PKI under `/opt/tervon-bridge-m6-lab` (observed `M6_LAB_PKI=PASS`, `M6_LAB_SERVER_JSON=CREATED`, `M6_LAB_CONFIG_DIR=/opt/tervon-bridge-m6-lab/config`, `M6_LAB_CLIENT_JSON=/opt/tervon-bridge-m6-lab/private/client.json`, `M6_LAB_EXPIRES_WITHIN_DAYS=1`, `M6_LAB_RUNTIME=NOT_STARTED`). This confirms script execution and OpenSSL verification of the local test certificate chain only. No new ingress container started; external port and Android/mTLS E2E remain NOT_TESTED. These one-day keys are test-only and cannot be treated as production enrollment. Next: explicitly start only the new Compose-profiled ingress container bound to `127.0.0.1:18443`, verify runtime logs/listener and perform loopback TLS test; do not touch existing Russia–Finland Bridge, VLESS or protective services.


### M6 — first local container launch failed: obsolete main-source build (2026-11-20)

Operator observed `docker compose ps` with no running ingress container, no TCP listener `:18443`, and ingress log `json: unknown field "padding_enabled"`. Verified GitHub source comparison: `main` and `work/m6-android-ingress` `prototype/cmd/bridge/main.go` lack `padding_enabled`, `jitter_enabled`, `/v2/stream`; released tag `v1.0.1` supports them. Therefore **CONTAINER_START=FAIL** due to **wrong source tree for Docker image**, not established PKI failure. This test must not be counted PASS.

- Fixed Dockerfile on M6 branch to fail closed on source trees lacking Padding/Jitter and v2.
- Documented isolated detached Git worktree build from `v1.0.1`, without changing the current M6 checkout or live Linux Bridge.
- Next: operator rebuilds image from the official tag and supplies `docker build` result. Only then recreate *new ingress container* with `--no-build --force-recreate`; retain existing test-only bind to `127.0.0.1:18443`.
- Scope: no modification to active Xray, Russia–Finland Bridge, Stats, or firewall. Automated Android enrollment remains a separate incomplete part of the same M6 module.


### M6 — isolated Android ingress container started; loopback listener confirmed (2026-11-20)

Operator executed `docker compose -f integrations/android-ingress/compose.yaml --profile android-ingress-manual up -d --no-build --no-deps --force-recreate android-bridge-decapsulator` with `ANDROID_BRIDGE_CONFIG_DIR=/opt/tervon-bridge-m6-lab/config` after rebuilding the image from released `v1.0.1` sources. Runtime output: container `android-ingress-android-bridge-decapsulator-1` **Started / Up**, host binding **127.0.0.1:18443->8443/tcp**, `ss` confirms docker-proxy LISTEN on **127.0.0.1:18443**; immediate `docker compose logs --tail=20` had no lines. **CONTAINER_START / LISTEN PASS;** mTLS handshake, HTTP/2 application exchange, VLESS payload and Android E2E **NOT YET VERIFIED**. No existing Bridge Russia–Finland or Xray service changed. Next focused test: OpenSSL client certificate + server-name verification, TLS 1.3 and ALPN `h2` against loopback listener with the test PKI; this test must not transmit VPN payload.
