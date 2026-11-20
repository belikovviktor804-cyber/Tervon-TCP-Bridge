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


### M6 — approved permanent Android ingress conversion (2026-11-20)

Operator explicitly instructed: **one permanent installation, no more lab/temporary installations**, and remove abandoned temporary PKI after verified cutover. Observed prior local TLS 1.3 verification PASS against the existing loopback-bound lab Decapsulator (`M6_MTLS_EXIT=0`, `Verification: OK`, peer CN `bridge.android.test`), but this alone does not demonstrate HTTP/2 data plane, Android enrollment, or a production certificate.

GitHub M6 changes prepared, **NOT YET DEPLOYED**:
- `integrations/android-ingress/compose.yaml` has `restart: unless-stopped`, `pull_policy: never`, and no implicit build from obsolete prototype. Retains original Compose project/service, hardened container restrictions, loopback-only `127.0.0.1:18443`, and Xray backend `host.docker.internal:30000`.
- `integrations/android-ingress/prepare-production-pki.sh` initializes a **brand-new operator-named production PKI directory** once (never overwrites), separate ten-year server/client CAs, one-year server leaf with operator-controlled FQDN SAN, immutable location for CA keys outside container, protected server key mounted read-only. Starts with a deny-all client certificate fingerprint placeholder required by current release parser; no unregistered Android device will be allowed.
- `integrations/android-ingress/README.ru.md` defines controlled in-place conversion of **same** ingress container, explicit certification renewal and lab cleanup after verified cutover.

**Not yet PASS / operator input needed:** actual externally resolvable controlled Android Bridge DNS name for server leaf/SNI; creation of production secrets under `/etc/tervon/bridge-android`; controlled recreate and runtime validation; secure certificate issuance + atomic ACL update and Android E2E. Do not invent a domain from camouflage SNI or expose HTTPS publicly before authorization is in place. User's requirement is permanent installation, not to skip appropriate acceptance checks. Existing Russia–Finland Bridge, live VLESS TCP :30000, and protection services remain unchanged.


### M6 — operator confirmed permanent Android ingress hostname (2026-11-20)

Operator confirms ownership/creation of **`bridge.xrayvless.online`** as the intended dedicated Android Bridge HTTPS entry. No need to ask again or infer SNI from generic VLESS camouflage hostnames. This is a **hostname selection**, not proof of public DNS resolution, TLS certificate issuance, public port accessibility, or live Android end-to-end connectivity.

The M6 Compose configuration now defaults to **`/etc/tervon/bridge-android/config`** for its read-only volume, avoiding reliance on a shell-exported laboratory path after restarts and later Compose operations. `restart: unless-stopped`, `pull_policy: never`, and no implicit image build remain. **GitHub only, NOT YET APPLIED** on Homeserver. Once operator updates the checkout, perform one-time production PKI initialization with FQDN `bridge.xrayvless.online`, recreate **the same existing Android Decapsulator only**, prove that it mounts the permanent config and serves a valid matching server certificate, then remove **only** the explicitly identified laboratory PKI directory after successful cutover. Do not publish server/client CA private keys or service secrets. Access is deny-all until client enrollment and verified ACL are provided; no production Android connection is claimed.


### M6 — permanent Android-facing Decapsulator runtime confirmed (2026-11-20)

Operator supplied Homeserver `docker inspect`, `ss`, and `openssl x509` after in-place conversion of the **same** ingress container. **Observed:** `STATE=running`, `RESTART=unless-stopped`, bind mount `MOUNT=/etc/tervon/bridge-android/config:/config`; TCP listener at `127.0.0.1:18443` via docker-proxy; production leaf certificate `subject=CN=bridge.xrayvless.online`, `subjectAltName=DNS:bridge.xrayvless.online`, `notBefore=Nov 20 07:48:35 2026 GMT`, `notAfter=Nov 20 07:48:35 2027 GMT`. **PERMANENT CONTAINER START/CONFIG/LISTENER AND LEAF SAN: PASS.** The container no longer mounts laboratory configuration. This evidence does not show a fresh TLS handshake against the permanent certificate, HTTP/2 forwarding, remote port reachability or Android enrollment; those remain NOT_TESTED. User instruction: remove **only** obsolete `/opt/tervon-bridge-m6-lab` after exact-path/mount verification; do not leave temporary PKI on host. Existing Russia–Finland Bridge and direct VLESS `30000` remain untouched. Production ACL is deny-all pending verified client registration. Next scope in M6: operator-controlled secure enrollment, live ACL updates, external HTTPS ingress and Android client E2E without adding further test deployments.


### M6 — production enrollment contract recorded (2026-11-20)

A new M6-only document `integrations/android-ingress/ENROLLMENT_CONTRACT.ru.md` defines integration with the **existing** VLESS subscription authority, CSR signing with separate client CA, authenticated enrollment and per-leaf SHA-256 authorization, constrained revocation, Android protected socket/routing/rollback, and separation of public HTTPS bootstrap from Bridge mTLS. **Documentation only / proposed interface, not code PASS or deployed.** The installed permanent Decapsulator remains deny-all and loopback-bound. The single outstanding factual dependency for implementation is the actual source of subscription authorization on Homeserver (Xray config vs subscription service/panel); user identifiers, VLESS UUID and private keys must never be copied into GitHub or chat. Next inspect the existing Xray process/config *location only*, then implement server registration and app integration without additional temporary deployments.


### M6 — identified 3x-ui authorization database schema (2026-11-20)

Operator read the running Xray metadata (`/usr/local/x-ui/bin/xray-linux-amd64` with `-c bin/config.json`, CWD `/usr/local/x-ui`) and established that 3x-ui stores data in `/etc/x-ui/x-ui.db`. Read-only SQLite schema inspection (no records or secrets disclosed) confirmed: `clients(id, email, sub_id, uuid, enable, expiry_time, total_gb, limit_hwid, ...)`, `client_inbounds(client_id, inbound_id, ...)`, `client_traffics(inbound_id, enable, email, expiry_time, total, up, down, ...)`, `inbounds(id, port, protocol, enable, expiry_time, settings, ...)`, plus `client_hwids`, `settings`, `hosts` and `api_tokens`. This identifies the **existing source of record** for subscription eligibility, avoiding a duplicate account database. Schema names alone do NOT prove actual relationship cardinality, subscription authentication method, units of expiry, or correct quota policy. No query for UUID, password, secret, subscription ID or token values is authorized.

Next within M6: one aggregated read-only validation of the linkage for port 30000 (only counts/boolean flags, no identifiers) before implementing the registrar; keep the installed ingress isolated and deny-all until enrollment and ACL handling are ready.


### M6 — 3x-ui port 30000 authorization join not established (2026-11-20)

Operator executed read-only aggregate SQLite queries against `/etc/x-ui/x-ui.db` for `inbounds.port=30000 AND inbounds.protocol='vless'`. Results: `VLESS INBOUND 30000=0`, `CLIENT RELATIONS=0`, `ENABLED CLIENT RELATIONS=0`. **This invalidates the unverified assumption** that the active listener can be authorized directly by a `client_inbounds` join filtered on these exact values. It does not establish that Xray is down, that there are no users, or that the 3x-ui database is unused. Server socket `*:30000` was independently confirmed earlier. **AUTHORIZATION_SOURCE=BLOCKED_PENDING_MAPPING**; no registrar may issue certificates based on this join. Next targeted read-only comparison: print sanitized inbound port/protocol metadata from active `/usr/local/x-ui/bin/config.json` and aggregate SQLite inbound port/protocol counts without exposing UUID, subscriber identifiers or secrets. No service changes.


### M6 — confirmed empty 3x-ui SQLite tables despite active Xray config (2026-11-20)

Operator read metadata only from `/usr/local/x-ui/bin/config.json`, verifying configured Xray inbounds: `54321/tunnel` and `10811/socks` bound on `127.0.0.1`, plus `55636/vless` and `30000/vless` bound on `0.0.0.0`. In contrast, `/etc/x-ui/x-ui.db` read-only totals were `clients=0`, `client_inbounds=0`, `inbounds=0`, with no grouped inbounds. This decisively **disproves** that the currently inspected database stores those active Xray inbound/client records; it does not show whether a different 3x-ui DB or an independent subscription manager is responsible, and does not mean the running VLESS ports lack clients. **M6 authorization-source discovery BLOCKED** until actual subscription authority identified. Next perform metadata-only inspection of 3x-ui service process/config launch and potential other DB locations, avoid printing secrets from Xray JSON or databases. Do not bind enrollment to empty `/etc/x-ui/x-ui.db`, modify working Xray, or create a parallel account registry.


### M6 — operator surfaced 3X-UI 3.x OpenAPI as enrollment source (2026-11-20)

Operator confirmed panel executable `/usr/local/x-ui/x-ui`, service `WorkingDirectory=/usr/local/x-ui`, optional environment file `/etc/default/x-ui`; observed `/etc/x-ui/x-ui.db` last modified 2026-11-14 and all inspected `clients`, `client_inbounds`, `inbounds` counts zero. The operator additionally surfaced **3X-UI 3.x OpenAPI 3.0** for the deployed panel, with authenticated `/panel/api/*` endpoints and API token scopes `admin`, `monitor`, `node-sync`, `node-admin` (full rights per scope NOT YET verified). This is a better candidate integration boundary than directly reading the empty SQLite database or copying live Xray config. **API integration proposed, not implemented or tested.** Next: inspect only the locally served OpenAPI schema to identify exact read-only client/subscription eligibility endpoints and required minimal scopes; do not print or store the secret panel base path, Bearer tokens, UUIDs, or credentials in GitHub. Never blindly assume monitor scope can read all client data. Existing services remain untouched.


### M6 — local 3X-UI web-route confirmed, API route unresolved (2026-11-20)

Operator tested local panel access through HTTPS with domain resolution forced to `127.0.0.1:5588`. Results: `/panel/api/clients/list` → HTTP 404; panel-prefixed `/panel/api/clients/list` → HTTP 404; **the actual configured panel base path** → HTTP **200**. The local x-ui listener exists and binary contains API route strings, but neither tested API path succeeded. **Panel web route PASS; authenticated API path NOT_CONFIRMED.** Do not guess more paths or assert a token will solve a 404. Next inspect an actual same-origin browser Network request from the functioning panel (method + path suffix + HTTP status, redact the private base path and all cookies/tokens). No changes to running panel or Android Decapsulator. The private panel prefix is deliberately omitted from Git.


### M6 — authenticated 3X-UI client GET route confirmed in browser (2026-11-20)

Operator captured real browser Network request for an existing diagnostic client: `GET /panel/api/clients/get/{email}` with HTTP **200 OK**, using an authorized 3X-UI browser session. This **confirms that specific client lookup endpoint works under session authorization**, unlike earlier unauthenticated local curl attempts which returned HTTP 404. The exact panel prefix, account identity, server IP and response body are intentionally excluded from GitHub documentation. Do not infer 404 was definitively due to missing auth; do not infer scopes or response schema. **Browser client GET route PASS; API-token access, eligibility fields, certificate enrollment and Android end-to-end NOT_TESTED.** Next agree a read-only scoped API credential and examine redacted response field names for eligibility (enabled, expiry/quota/subscription), without exposing credentials and without touching live Xray or Bridge services.


### M6 — 3X-UI API token scope verified against upstream documentation (2026-11-20)

Authenticated browser `GET /panel/api/clients/get/{email}` is already confirmed HTTP 200. Cross-checked upstream 3X-UI documentation (`docs/content/docs/en/reference/api/api-tokens.mdx`, `frontend/src/pages/api-docs/endpoints.ts`, `docs/architecture.md`): `monitor` token is limited to status/metrics, **not arbitrary client records**; client lookup requires an admin-capable token or panel session. **Do not deploy a `monitor` token for enrollment on an unproven assumption; never deliver an admin token to Android.** Proposed privilege boundary: dedicated server-side registrar isolated on Homeserver holds a tightly protected backend credential, queries only the necessary client eligibility fields, authenticates Android using proof of the existing managed subscription credential bound to a CSR, and only then issues clientAuth certificates and manages per-leaf ACL. The proof mechanism, admin credential risk and response schema require explicit design review before implementation; **no code or deploy PASS**. Do not print raw client data, panel base path or secrets. Existing Bridge/Xray unchanged.


### M6 — new 3X-UI API bearer credential verified from Homeserver (2026-11-20)

Operator created a **new token in the reinstalled, active 3X-UI panel**, stored the Bearer header in a root-only private file under `/etc/tervon/bridge-android/private/`, and verified `GET /panel/api/clients/get/{email}` through `curl --header @<private-header-file>` via local `127.0.0.1:5588` HTTPS with intact hostname/TLS verification. **Observed `M6_XUI_API_HTTP=200`. Result: 3X-UI backend token authentication and client GET route PASS** for the existing diagnostic account. No token values, subscriber details, private panel base path or external service address are recorded in Git. Previous bot tokens are obsolete after panel reinstall and must never be reused. This proves API access, **not** what response fields mean, client eligibility logic, CSR proof-of-possession, certificate issuance, dynamic ACL updates, or Android end-to-end. Next: implement M6-only registrar integration after confirming redacted response field names and secure enrollment auth; never put token in Android or Docker ingress, preserve existing Linux Bridge, Xray and 3X-UI services.


### M6 — 3X-UI client list read-only schema confirmed (2026-11-20)

Operator queried live panel using the new private bearer header via local TLS. `GET /panel/api/clients/list` returned HTTP 200, JSON parsed, `obj` array of **2 client entries**. Observed fields and types on first entry: `id:int`, `email:str`, `subId:str`, `uuid:str`, `limitHwid:int`, `totalGB:int`, `expiryTime:int`, `enable:bool`. Separate `GET /panel/api/clients/get/Diag` returned HTTP 200 with `obj:dict`, whose nested field names were **not** yet identified by the allowlisted read-only inspection. **Client listing / first-entry field schema PASS**, client eligibility semantics and individual-object nesting NOT_TESTED. Proposed server-side mapping uses 3X-UI `id` for database identity and `subId` for subscription lookup, with an independent device identity/certificate fingerprint; this is a proposal, not proven enrollment authentication. Avoid raw `uuid`, `subId`, `email`, subscriber values, panel prefix, tokens or private keys in logs/GitHub. Continue with minimal redacted key/type inspection and policy determination before registrar coding, preserve production services.


### M6 — 3X-UI individual client object nesting confirmed (2026-11-20)

Operator performed one more redacted, read-only lookup for the diagnostic record using the dedicated server-side Bearer token. `success: True`, `obj: dict`, `obj.client: dict` and nested fields/types confirmed: `id:int`, `email:str`, `subId:str`, `uuid:str`, `limitHwid:int`, `totalGB:int`, `expiryTime:int`, `enable:bool`. **Individual-client response shape PASS.** No raw customer or credential values collected. No usage counters (`up`/`down`) were observed in the allowlisted traversal; their absence from output is not proof they are absent throughout the response. Next task is NOT additional redundant fetches: define precise eligibility semantics and separate authentication proof from lookup. `email`, `id`, `subId` and `uuid` are identifiers, not by themselves enrollment authentication. Proposed mapping: server-only client `id` as database identity, independent device identifier and mTLS leaf fingerprint; do not write Android CSR-signing logic until enrollment proof and revocation rules are agreed. No changes to running services.


### M6 — enrollment authorization design: verified API surface vs proposed proof (2026-11-20)

Read active M6 contract and existing operator-confirmed `obj.client` schema. Upstream 3X-UI docs explicitly publish read-only `GET /panel/api/clients/traffic/{email}` in addition to already confirmed `GET /panel/api/clients/get/{email}` and `/clients/list`; published upstream examples represent `totalGB` as a byte quota and `expiryTime` as milliseconds since epoch, with `0` meaning unlimited quota/expiry. **These precise semantics are DOCUMENTED UPSTREAM, NOT yet tested on installed panel revision.** Use existing 3X-UI for entitlement/usage rather than creating a parallel subscription database. `enable` is required but quota/expiry and attachment/eligibility require validation against deployed implementation.

Proposed, NOT_APPROVED/NOT_IMPLEMENTED credential path: an imported *managed VLESS* profile may supply its existing high-entropy UUID as an enrollment secret over a separately authenticated public HTTPS bootstrap channel; server-only registrar checks credential against live 3X-UI client data using constant-time comparison, then checks enabled/expiry/quota, verifies Android CSR proof-of-possession via standard X.509 library, issues a short-lived distinct clientAuth certificate and maps panel client identity + device to its leaf certificate fingerprint for ingress ACL. This is a **security-sensitive tradeoff**: VLESS UUID is a reusable VPN credential and exposes that credential to the registrar if sent, so a dedicated scoped enrollment credential issued through the existing authenticated subscription is preferable when available. An `email`, `id`, `subId`, HWID, IP, or CSR alone is not authorization. Token for panel remains only on Homeserver; do not reuse stale bot credential. Panel client `id` may change after reinstalls and is not a global immutable identity. Choose credential option and ACL reload mechanism before changing executable code or live service. No code/deployment claims.


### M6 — implemented isolated read-only 3X-UI API adapter (2026-11-20)

On branch `work/m6-android-ingress`, source and mock HTTPS tests committed under `integrations/android-ingress/xui/` (Go module, `client.go`, `client_test.go`). The adapter uses GET-only requests to confirmed `/panel/api/clients/list` and `/panel/api/clients/get/{email}`, the already-provisioned private Bearer-header file, HTTPS-only URL validation, disabled redirects, strict response envelope handling, a request timeout, and fail-closed errors without leaking token or private base path. **Local Go 1.23.2 unit tests: six cases PASS** using mock TLS panel, no production network or secret. Existing panel API token GET 200 verified separately by operator, but the new Go package itself is **NOT deployed / NOT tested against live 3X-UI**, and Actions result not claimed. This is an internal M6 API-read adapter, **not** user enrollment or a completed registrar. Remaining M6: approve and implement proof of subscription ownership (independent enrollment credential preferred), live eligibility/traffic checks, CSR issuance and revocation, ingress dynamic ACL, external HTTPS and Android E2E. Live Xray, Linux Russia–Finland bridge and running Docker ingress unchanged. Do not mark M6 PASS.


### M6 — fail-closed client entitlement predicate implemented, NOT an enrollment registrar (2026-11-20)

After reviewing current `work/m6-android-ingress` source/plan, added `integrations/android-ingress/xui/eligibility.go` and `eligibility_test.go` via GitHub API. `CheckSubscriptionEligibility(ClientRecord, *UsageSnapshot, now)` validates identity, enabled flag, expiry, quota, negative limits, missing usage for limited accounts and int64 overflow. Returns denial on unknown usage instead of treating it as zero. Semantics based on current upstream 3X-UI docs: `expiryTime=0` and `totalGB=0` unlimited, negative expiry pending first-use; these details remain **NOT_CONFIRMED on the installed panel**. `limitHwid` is not yet enforced, merely rejected when invalid. No CSR issuance, proof-of-possession, dynamic ACL or service changes.

**Test evidence:** local Go 1.23.2 isolated equivalent policy tested: 14 table scenarios + 1 separate test PASS. The exact new GitHub package has NOT been built/tested in CI or on Homeserver in this turn; do not call it full M6 PASS. Next current-M6 steps: inspect actual traffic/inbound linkage with one redacted read-only GET, enforce subscription and device eligibility based on authoritative response, agree/deliver independent enrollment credential via existing subscription or choose documented tradeoff, implement secure CSR registrar and atomic ACL reload for the separate ingress, then Android E2E. Only proceed to live integration with explicit evidence. Linux v1.0.1 Russia–Finland and existing Xray untouched.


### M6 — isolated GitHub Actions test workflow added (2026-11-20)

Added `.github/workflows/test-m6-xui.yml` on current M6 branch. It scopes jobs to `integrations/android-ingress/xui/**`, runs `go test -race -count=1 ./...` on GitHub-hosted Ubuntu with read-only repository permissions, and never accesses Homeserver, 3X-UI, TLS keys or production. Workflow commit `405e7764d12a715e1f7752f7ebe556ce5c5f80bb`. **GitHub Actions PASS is NOT_CONFIRMED**: commit-associated connector returned no PR-triggered workflow runs (the connector is documented as PR-only) and public Actions page was not accessible through the web tool. Do not conflate that with CI failure or success. Separate equivalent local eligibility test PASS remains the only current check of new eligibility logic. Next inspect CI result when available and correct defects within M6 before integrating registrar.
