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


### M6 — traffic API adapter and quota integration committed (2026-11-20)

Reviewed active M6 plan and existing `xui` read-only adapter. Upstream 3X-UI API docs confirm `GET /panel/api/clients/traffic/{email}` returning a `ClientTraffic` object (up/down counters, email). Added `integrations/android-ingress/xui/traffic.go` and `traffic_test.go` on `work/m6-android-ingress`. The new internal GET-only method requires matching email and *present, integer* `up`/`down` (missing/null/malformed counters fail closed). `CheckEligibilityByEmail` first fetches `obj.client`, validates status/expiry, requests traffic only for nonzero quotas, and calls the existing `CheckSubscriptionEligibility`; no guessed zero-traffic fallback. Tests cover under/at quota, missing counters, wrong identity, malformed/null traffic, unlimited quota, HTTP 404 and invalid path identifiers. **SOURCE COMMITTED, CI PASS NOT YET VERIFIED, installed panel traffic response NOT YET OBSERVED.** No certificate signing or enrollment authentication implemented, no Docker/3X-UI/Xray/production Bridge changes. Before accepting this M6 sub-step, run source-package `go test -race -count=1 ./...` and one sanitised HTTP read of the actual traffic response structure with the existing token, without displaying subscriber data.


### M6 — operator race-test FAIL and minimal validator correction (2026-11-20)

Operator ran exact source from commit `89cddc4` on Homeserver: `go test -race -count=1 ./...` in `integrations/android-ingress/xui` **FAIL** at `TestTrafficLookupRejectsInvalidIdentifier` for a newline-containing email: HTTP 404 instead of input rejection. Root cause: the newly added `traffic.go` identifier filter used an incorrectly escaped character string and failed to reject the control character before making the request. Corrected **only** `validateEmail` to reject ASCII control characters, DEL and path delimiters via `strings.IndexFunc`, commit `a4078f5b70afee2fe8b923604da616a3fd4d05d2`. Independent Go 1.23.2 micro-check on same predicate confirmed newline, slash, backslash, question mark and hash are rejected. **Full source package re-test after fix NOT_YET_CONFIRMED**; request one run only, not repeat prior unrelated checks. Operator also correctly notes that Xray already denies VPN traffic when a VLESS subscription is no longer valid. Bridge must NOT independently duplicate traffic quota and expiry enforcement for every forwarded stream. 3X-UI entitlement checks, if needed, belong at the **one-time enrollment/certificate authorization boundary**, not in TCP forwarding; whether to retain the unused quota adapter should be decided within M6 without architecture churn. Live services unchanged.


### M6 — Homeserver Go race tests PASS after validator fix (2026-11-20)

Operator updated worktree by SSH Git `git pull --ff-only origin work/m6-android-ingress` from `89cddc4` to **`d5bba1c`** and ran `go test -race -count=1 ./...` in `integrations/android-ingress/xui`. **Result `ok github.com/tervon-software/Tervon-TCP-Bridge/integrations/android-ingress/xui 1.305s`; PASS.** The preceding test failure for newline in the traffic lookup identifier is resolved. No need to repeat this test without a code change. This is test PASS for the xui package only, not completion of M6, release, or real Android enrollment. Operator-approved simplification: existing Xray/panel remains the authority for subscription validity and whether actual VPN traffic is accepted; Bridge merely transports it and must not mirror traffic/quota checking in the stream path. Separate secure certificate issuance still requires proof of client authorization; ACL/CSR and Android end-to-end remain incomplete. No existing services changed.


### M6 — device CSR signer and staged ACL for FIRST controlled Android test (2026-11-20)

On current branch, after operator-confirmed `go test -race` PASS of xui adapter (`d5bba1c`), added two **internal library primitives**, `integrations/android-ingress/xui/certificate.go` and `acl.go`, plus `certificate_test.go` and `acl_test.go`. The signer uses Go standard `crypto/x509` with signed P-256 CSR, a matching private client CA, at most 24h clientAuth-only leaf, and returns public PEM + SHA256 DER leaf fingerprint; no phone private key is created or accepted by the server. `AddDeviceToACL` constructs a **staged JSON byte array** preserving Bridge v1.0.1 transport options, requiring an existing service/route and a canonical SHA256 leaf fingerprint. It deliberately does not write live configuration, reload Docker or expose a public enrollment API. There is NO claim of authenticated Android enrollment or production deployment: calling the signer must be preceded by trusted operator action or a separately implemented, authenticated enrollment gate. v1.0.1 authorization remains startup-only; any controlled activation must affect only the dedicated Android ingress, never the already-working Russia–Finland bridge or Xray. Tests were committed but **the new code's package test PASS is NOT YET confirmed**; request one exact-source `go test -race -count=1 ./...` and fix failures before proceeding. Next M6: connect this to a small operator-approved first-device enrollment/activation path, then Android socket protection and end-to-end traffic test. No running services changed in this turn.


### M6 — Homeserver package race tests PASS for CSR and ACL primitives (2026-11-20)

Operator fast-forwarded current SSH worktree from `d5bba1c` to **`2a424d4`** on `work/m6-android-ingress` and ran `cd integrations/android-ingress/xui && go test -race -count=1 ./...`. **Result: `ok github.com/tervon-software/Tervon-TCP-Bridge/integrations/android-ingress/xui 1.351s` — PASS**. This includes newly committed `certificate.go`, `certificate_test.go`, `acl.go`, and `acl_test.go`, as well as prior xui tests. Repeat only after code changes or a specifically identified defect. This is unit/race test PASS of the package, **not** mTLS enrollment, deployment, ACL activation or Android end-to-end PASS. No running services changed by the commands. Next M6 step is a narrowly scoped, operator-authorized first-device registration flow and an activation plan affecting only the dedicated Android ingress; never assume that successful CSR parsing alone authenticates a device.


### M6 — manual first-device staging CLI added (2026-11-20)

Operator approved an **initial controlled enrollment of one test Android**, ahead of any public self-service registrar, strictly limited to dedicated Android ingress. On `work/m6-android-ingress`, added `integrations/android-ingress/xui/cmd/m6-first-device/main.go` and unit tests. It is a root-only local operator command that reads an **Android-generated, signed P-256 CSR** and verifies an explicitly supplied device common name; reads existing **client CA** (private key stays on Homeserver) and current Android ingress server config; signs one 24-hour clientAuth leaf through existing library; creates a **new, private, non-overwriting staging folder** directly under `/etc/tervon/bridge-android/private/` containing `android-client.crt`, `leaf.sha256` and `server.staged.json`. It **does not write** `/etc/tervon/bridge-android/config/server.json`, restart or modify Docker, expose a public registration port, or touch Xray/3X-UI/Linux Russia–Finland Bridge. It is explicitly operator-controlled; no Android claim/auth API exists yet. Staging is **NOT deployment** and not proof of live Android mTLS or VPN connectivity. An Android-native key/CSR generation mechanism suitable for the installed Bridge TLS client is still needed; do not pretend the Android application already has it. CLI unit tests committed, package CI/Homeserver PASS **not yet confirmed after this change**. Next verify `go test -race -count=1 ./...` in the exact xui module and then arrange Android CSR, first staged issuance, and separately controlled activation of *only* the dedicated Android ingress following the result.


### M6 — first-device staging CLI tests PASS on Homeserver (2026-11-20)

Operator fetched SSH worktree `work/m6-android-ingress`, fast-forwarding from `2a424d4` to **`3a6fc6e`**. Ran exact-source `cd integrations/android-ingress/xui && go test -race -count=1 ./...` on Homeserver. Confirmed **PASS** in both packages: `.../xui` **1.346s**, `.../xui/cmd/m6-first-device` **1.020s**. The staging CLI and library unit/race tests are confirmed; **no real Android CSR issued, no operator certificate issued, no ACL installed, no Docker activation and no Android E2E**. The first real step must obtain a signed CSR from an Android-held private key (no private key transfer), then operator-only staging; do not assume current Android application already supports CSR generation. No need to repeat these tests absent code changes.

 
### M6 — recovered Android integration evidence and actual current blocker (2026-11-20)

Context recovery based on actual source-archive inspection plus earlier operator reports (not new tests). Earlier work crosscompiled **release Bridge v1.0.1** to Android ARM64, packaged its **Go executable ELF** as `android/app/src/main/jniLibs/arm64-v8a/libtervonbridge.so`, and proved execution on a real device via `RESULT=PASS_EXEC_ANDROID_ARM64` (CLI `-config`). **No JNI exported API**: Android starts it with `ProcessBuilder`. A local **Android ARM64-to-Android ARM64 Bridge loopback** achieved `RESULT=PASS_ANDROID_LOOPBACK_MTLS_TCP`: 16,384 byte exact pass over HTTP/2, TLS 1.3, mTLS, Padding/Jitter. These are Android local proofs, **NOT Android-to-Russian-ingress E2E**. A previous post-HEV-JNI-fix APK was built with working VPN/internet; preserve that HEV/Xray/DNS path.

Recovered actual Kotlin source artifacts from existing staged packages:
- Stage 1: `BridgeConfigManager.kt` stores Bridge executable/config and isolated app-private `filesDir/bridge/config/bridge.json` and `pki`; `BridgeCoreManager.kt` handles process lifecycle via `ProcessBuilder`; `BridgeVpnIntegration.kt` currently only checks a loopback TCP target. **It does not implement Xray routing.**
- Stage 3: `BridgeRuntimeControl.kt` exposes Flutter `installClientConfig/start/stop/status`, invokes `-config`, and **reports `vpnRouteIntegrated=false`**; runtime UI stores JSON and starts Bridge only on demand. The previously built APK's exact current installed sources must be inspected before any modifications; staged sources are not proof of installed version.
- Stage 4: historical conversation references a proposed `tervonvpndpi-enrollment-stage4.tar.gz` / `BridgeDeviceIdentity.kt` and an AES-GCM/Android-Keystore-wrapped P-256 identity design. **These are NOT verified source artifacts**: operator searched `/opt`, `/root` and `/home` on Homeserver for Stage 4 archives/source files and found none. Operator confirms Stage 4 was **never integrated into TervonVPN DPI nor compiled into an APK**. Prior claim that existing Stage 4 sources/archive had been recovered was unsupported and is withdrawn. **Stage 4 status: SOURCE_NOT_FOUND / NOT_INTEGRATED / NOT_COMPILED / NOT_TESTED**. The design details are only a prior proposal until independently confirmed in actual source. Do not replace existing ProcessBuilder/PEM mechanism with JNI without a demonstrated need.
- Server PKI generation already exists in `prepare-production-pki.sh` (OpenSSL P-256 CA/server leaf); the Bridge v1.0.1 process **loads**, rather than generates, PEM keypair from `cert_file/key_file/ca_file`. This resolves the prior confusion about built-in key provisioning. Production Android-facing Docker ingress is permanent, loopback `127.0.0.1:18443`, cert SAN `bridge.xrayvless.online`, route to existing Xray TCP `:30000`, ACL **deny-all**; local Docker-to-Xray TCP was confirmed separately. Never repeat prior proofs without reason.

**M6 next blocker:** need the *actual current* TervonVPN DPI source at `/opt/tervonvpndpi/android/app/src/main/kotlin/com/tervonvpndpi/app` and Flutter routing implementation, since only historical staged archives and operator grep signatures were available; applying historical `apply.py` blindly would risk overwriting working app. Inspect the current source snapshot before any code patch, then make the **minimum additive wiring** for already-existing device identity → authenticated registrar → PEM-compatible private client config → Bridge readiness → route toggle with rollback. Confirm the precise mechanism of enrollment credential authentication and how the external Android ingress will be exposed before live certificate issuance; no extra parallel VPN/account system. Work only in M6; keep current Linux Bridge and production VPN services untouched.


### M6 — correction: Stage 4 artifacts not found on Homeserver (2026-11-20)

Operator reports read-only `find` searches in `/opt`, `/root` and `/home` for `*stage4*`, `*enrollment*` and `BridgeDeviceIdentity.kt` returned **no files**. Earlier documentation asserted an existing staged archive and Kotlin implementation without source evidence. **Retract that assertion**: Stage 4 design may have been discussed, but actual source and APK incorporation are **not confirmed**. Operator separately confirms there was no Stage 4 APK compilation. This evidence does not prove files never existed elsewhere; for M6 planning treat Stage 4 as **NOT_IMPLEMENTED until source is located or new approved implementation is completed**. Prior Android ARM64 bridge runtime/loopback proofs and server-side Go unit-test PASS are separate unaffected results. No executable code or services modified by this documentation correction.


### M6 — Stage 4 Android source files CREATED in GitHub (2026-11-20)

Following operator instruction to **create the missing Stage 4 files**, new isolated sources were committed in `integrations/android-ingress/stage4/`: `BridgeDeviceIdentity.kt`, `BridgeDeviceCsr.kt`, `BridgeDeviceSelfCheck.kt`, `README.ru.md`. Working branch: **`work/m6-android-ingress`**; code stage commit through `90450c5a774ddd43938d1e0dc513c6dc89b3f3d6`. These are newly authored files and are **not** recovered/previously implemented artifacts. They do not alter installed TervonVPN DPI and do not touch running Linux Bridge/Xray/Docker services.

Implementation: app-scoped exportable P-256 PKCS#8 identity encrypted using AES-256/GCM with the Android Keystore non-exportable **symmetric wrapping key**, atomically stored under `noBackupFilesDir`; stable SPKI, SHA256withECDSA challenge signature, PKCS#10 PEM CSR via Bouncy Castle `bcpkix` (required dependency still needs version/Gradle compatibility confirmation). `BridgeDeviceSelfCheck` is **source only**, not a completed test. Since the Go Bridge is a separate `ProcessBuilder` ELF that uses `tls.LoadX509KeyPair`, an app-private ephemeral PEM handoff and cleanup will still be needed; no claim that the *EC client private key* is non-exportable. No own crypto/ASN.1 implementation. No registrar/Android API, ACL runtime activation, protected external VPN socket or Xray route switching added in this stage.

**Status:** `STAGE4_SOURCE_CREATED`, `ANDROID_INTEGRATION_NOT_STARTED`, `ANDROID_COMPILATION_NOT_RUN`, `DEVICE_TEST_NOT_RUN`, `ANDROID_E2E_NOT_RUN`. Prior server Go `go test -race` PASS stays valid for unchanged code only; do not report Stage 4 PASS or rerun unrelated suites. Next within M6: inspect exact current `/opt/tervonvpndpi` Gradle/Kotlin/Flutter code, integrate these new files minimally, add authenticated enrollment and private-key runtime PEM handoff, compile on Homeserver, fix concrete errors, then device E2E. No live services changed.


### M6 — reviewed actual TervonVPN DPI sources; guarded Stage 4 integration authored (2026-11-20)

Operator supplied `tervonvpndpi-stage4-review.tar.gz`, containing **15 current project files** (Kotlin Bridge stack, actual Gradle Kotlin DSL, Flutter Bridge settings/runtime screens, home/settings screens). Source review confirms: `BridgeCoreManager` calls Android `ProcessBuilder` with the existing ELF; `BridgeConfigManager` uses app-private `filesDir/bridge` and `-config` JSON; `BridgeRuntimeControl` already exposes `status/installClientConfig/start/stop` and declares `vpnRouteIntegrated=false`; `BridgeVpnIntegration` does NOT reroute Xray, but notes the app UID is excluded from VPN routing. Existing Flutter Bridge screen and `BridgeControlChannel` form an isolated test integration point. Current Gradle `applicationId=com.tervonvpndpi.app`, arm64-v8a only, JDK17, and release-signing environment; no PKCS#10 Bouncy Castle dependency yet. Actual code, not historical stage assumptions, is the basis for changes.

Committed `integrations/android-ingress/stage4/apply_to_tervonvpndpi.py` plus README on branch `work/m6-android-ingress`. Tool runs `--check` or explicit `--apply`; requires exact SHA256 bytes of operator-provided Gradle, BridgeControlChannel, and Bridge Settings Dart; fails before writes if anything differs, stages only the three new Kotlin Stage 4 identity/CSR/selfcheck files, one Bouncy Castle bcpkix dependency (1.87), one self-check MethodChannel case, and one Flutter diagnostic button. Does not alter `BridgeCoreManager`, HEV, Xray, VpnForegroundService, signing configuration, versioning or publication. Static check of **all four source patch anchors** against supplied archive: each exists exactly once. **No claim of Python installer --check/--apply PASS, APK compilation, Android self-check or M6 E2E yet.** The current approved next gate is Homeserver `--check`, then guarded `--apply`, followed by a focused Android/Gradle compilation with signing unchanged and instrumented device check. Do not re-run unaffected Linux Bridge/Go tests. Public enrollment API/auth and Android-to-server E2E remain separate incomplete steps within M6; server Docker and all production services untouched.


### M6 — operator-reported Android Gradle compilation PASS, Stage 4 inclusion unverified (2026-11-20)

Operator executed `cd /opt/tervonvpndpi/android && ./gradlew :app:compileDebugKotlin --no-daemon` on Homeserver. Output: **BUILD SUCCESSFUL in 1m 17s; 72 actionable tasks (67 executed, 5 up-to-date)**, including `:app:compileFlutterBuildDebug` and `:app:compileDebugKotlin`. Warnings only about AGP 9 / Kotlin plugin deprecations, Java 8 source level of dependencies, and pre-existing Kotlin code; no compilation errors reported. **ANDROID_DEBUG_KOTLIN_COMPILATION=PASS for the current working tree.** However this log does not prove that the newly authored Stage 4 Kotlin files/dependency/button were applied; no operator output for `apply_to_tervonvpndpi.py --check` or `--apply` was provided. **STAGE4_INCLUDED_IN_COMPILE=UNVERIFIED.** Do not repeat the same Gradle build without checking inclusion or identifying a code change. Next read-only gate: verify all three Stage 4 Kotlin files exist under actual package and grep Gradle/Flutter/channel for Stage 4 markers. No APK packaging, signing, Android device self-check, or Android-to-ingress E2E PASS claimed. No running service changes.


### M6 — operator verified Stage 4 integration markers in application (2026-11-20)

Following successful `:app:compileDebugKotlin`, operator supplied read-only grep output from `/opt/tervonvpndpi`: `android/app/build.gradle.kts:95` contains `implementation("org.bouncycastle:bcpkix-jdk18on:1.87")`; `BridgeControlChannel.kt:28` contains `"stage4IdentitySelfCheck"`; `lib/features/home/bridge_settings_screen.dart:77` defines `_checkStage4Identity()` and line 156 connects the button. **Gradle/MethodChannel/Flutter integration markers PRESENT** in actual Android working tree. Output for `ls .../BridgeDevice*.kt` was not included, so three Kotlin Stage 4 source files are **not independently confirmed present** by this report. The earlier Kotlin compilation PASS stands, but do not claim that source files were compiled until their existence and included Kotlin path is verified. No APK packaging, device self-check, enrollment, or mobile E2E verified; no service changes. Next gate: verify three Stage 4 source filenames, then APK package and device self-check without rerunning already-passed unchanged Gradle compilation.


### M6 — Stage 4 source inclusion + debug Kotlin compilation PASS (2026-11-20)

Operator confirmed actual source presence under `/opt/tervonvpndpi/android/app/src/main/kotlin/com/tervonvpndpi/app`: `BridgeDeviceCsr.kt` (1578 bytes), `BridgeDeviceIdentity.kt` (7861 bytes), `BridgeDeviceSelfCheck.kt` (2192 bytes), all listed Nov 20 15:18. Earlier reported Gradle/MethodChannel/Flutter markers are present and `./gradlew :app:compileDebugKotlin --no-daemon` completed `BUILD SUCCESSFUL in 1m 17s` (72 tasks). **STAGE4_SOURCE_PRESENT=PASS; STAGE4_DEBUG_KOTLIN_COMPILE=PASS** for reviewed working tree; no new compilation requested absent changes. **Not yet confirmed:** APK packaging/signature, on-device identity/CSR self-check, enrollment, ACL activation, external ingress, Android→Decapsulator→Xray E2E. Next prepare APK using existing project workflow without version bump, release publishing or service changes, then on-device manual Stage 4 self-check.


### M6 — Android APK packaging failure: Bouncy Castle duplicate license resource (2026-11-20)

Operator attempted an **assembleRelease** build on Homeserver (separate from the earlier successful `:app:compileDebugKotlin`), and `:app:mergeReleaseJavaResource` failed after 1m36s: 3 Bouncy Castle 1.87 artifacts (`bcpkix`, `bcutil`, `bcprov`) each supply `META-INF/LICENSE.md`. This is a **Java-resource merge conflict**, not a Kotlin compilation or X.509 runtime failure. **STAGE4_RELEASE_APK_PACKAGE=FAIL**; no APK release/signing PASS. Reviewed actual operator archive `android/app/build.gradle.kts`, which already contains a `packaging { jniLibs { ... } }` block. Approved minimal fix is to add **only** `resources { pickFirsts += "META-INF/LICENSE.md" }` to that existing packaging block, retaining one LICENSE.md metadata entry; no replacement of the other Bouncy Castle artifacts, no changes to signing, versions, minify, native libs or live VPN. New guarded single-file script `integrations/android-ingress/stage4/fix_bc_license_merge.py` checks the exact SHA-256 of the previously Stage4-integrated Gradle file and offers `--check` / `--apply`; original first-run Stage4 installer also updated to include the same packaging rule. This is **CODE_COMMITTED / HOST_PATCH_NOT_RUN / REBUILD_NOT_RUN**. Next: operator SSH pulls current branch, runs guarded --check then --apply on `/opt/tervonvpndpi`, and **retries only the failed APK packaging build** (no need for separate Kotlin test or unrelated service tests). Existing Linux Bridge/Xray/Android Docker services untouched.


### M6 — on-device Stage 4 CSR button has no visible result; isolated UI correction (2026-11-20)

Operator reports release APK **built successfully** after prior Bouncy Castle packaging fix, but tapping Flutter **«Проверить ключ и CSR»** does not show a visible result and briefly blinks **«Сохранить настройки»**. This is not a Stage 4 cryptographic PASS or FAIL. Reviewed actual archived `lib/features/home/bridge_settings_screen.dart` and approved Stage 4 injection: `_checkStage4Identity()` sets the **shared** `_busy` state (disables Save Settings) and writes to shared `_message` displayed lower in the general diagnostics card. This explains the button blink and can obscure the result, but does **not** prove whether Kotlin self-check itself succeeded. Added **one-file, guarded** script `integrations/android-ingress/stage4/fix_stage4_selfcheck_ui.py` (commit `c6859e911c5a888e7195b89f9939dadbf40a6a39`): it requires exact original Stage4 Dart method/state/button snippets; `--check` is read-only, `--apply` atomically modifies only `lib/features/home/bridge_settings_screen.dart`, replacing shared busy/result use with `_stage4Busy` and `_stage4Result`, and renders PASS/FAIL or sanitized PlatformException code/message **immediately beneath Stage 4 button**. No Kotlin/Gradle/crypto, Save Settings handler, Xray, HEV, VPN route, build signing or live services changed. GitHub content readback confirms expected Dart interpolation preserved. **SOURCE_PATCH_COMMITTED; HOMESERVER_APPLY_NOT_RUN; NEW_APK_BUILD_NOT_RUN; PHONE_SELF_CHECK_NOT_CONFIRMED**. Next operator SSH pulls, `--check` then `--apply`, rebuilds APK by existing approved method, installs and taps Stage4 button; capture the exact displayed result. Do not claim actual CSR pass based on UI behavior.


### M6 — Stage 4 UI-only hotfix applied on Homeserver (2026-11-20)

Operator fast-forwarded checkout of `work/m6-android-ingress` from `48cd21b` to `5de70c6`, then executed `python3 integrations/android-ingress/stage4/fix_stage4_selfcheck_ui.py --app-root /opt/tervonvpndpi --check`: `STAGE4_UI_PATCH=READY`, `STAGE4_UI_CHANGED_FILES=1`, `STAGE4_UI_FILES_UNCHANGED=YES` (**preflight PASS**). Immediately executed same script with `--apply`: `STAGE4_UI_PATCH=APPLIED`, `STAGE4_UI_BUILD=NOT_RUN` (**source patch APPLY PASS**). Only `lib/features/home/bridge_settings_screen.dart` was targeted; changes to app services, VPN configuration, signing or publication not reported. **Do not re-run this one-shot patch or repeat previous tests without cause.** Next use exact previously working APK build/signing procedure (without version bump/publication) and validate visually on device that Stage4 result appears below its button and Save Settings does not blink; **APK rebuild / runtime self-check still NOT_CONFIRMED**.


### M6 — Stage 4 Android on-device P-256 identity and CSR self-check PASS (2026-11-20)

Operator provided a screenshot of the installed TervonVPN DPI Bridge Settings screen on a real Android device after rebuilding the APK with the Stage4 UI fix. Under the button «Проверить ключ и CSR (Stage 4)» it visibly displays **`Stage 4: PASS — ключ устройства, подпись и CSR`**; immediately below, status reads **«Настройки загружены; TCP-маршрутизация пока не активирована»**. This is **STAGE4_ANDROID_IDENTITY_CSR_SELF_CHECK=PASS** and **STAGE4_UI_RESULT_DISPLAY=PASS**, based on the app's explicit self-check (stable public key across instances, ECDSA challenge verification, PKCS#10 CSR public-key match and signature verification). It is **not** proof of server-side certificate issuance, subscription authorization, mTLS handshake to production, external HTTPS ingress, ACL activation, VPN routing, or Android-to-Russian-server TCP E2E. No change to working Linux Bridge/Xray/Docker services. Stage4 *identity and CSR self-check* may be closed PASS; continue M6 only with authenticated enrollment, server ACL and real routing after review. No need to repeat successful Stage4 self-check without a code change or specific failure.


### M6 — operator-approved active-server address reconciliation and per-server certificates (2026-11-20)

**Requirement, NOT YET IMPLEMENTED / NOT TESTED.** TervonVPN DPI Bridge uses a device-scoped P-256 identity independent of VPN subscriptions, while retaining **individual client mTLS certificates and Bridge settings per authorized Bridge server**. The active subscription supplies the current server/transport address and source of trusted Bridge discovery information. When Android initiates a connection or switches **protocol, server, or subscription**, it must re-read the actual active profile's address (not a cached server from a prior session), resolve its identity against the app's trusted Bridge server registry, and compare it against the currently selected Bridge server.

- **Same current Bridge server:** no certificate switching, re-enrollment, or unnecessary Bridge process restart. Maintain the current route if enabled.
- **Different known Bridge server:** choose the corresponding stored certificate and configuration, verify certificate validity and server reachability as part of activation, and switch the Bridge to the selected server. If no valid certificate is stored, obtain an individual client certificate through the authorized enrollment flow, then save it for future switching. Retain the same independent device P-256 identity.
- **Unknown server / subscription without an authorized Bridge:** automatically switch **Bridge off**, use the ordinary connection path as supported by the selected profile, and never send the certificate of another server to an unknown endpoint.
- **Known but unreachable server:** distinguish connectivity failure from unknown server identity; show an error and do not delete stored certificates or silently treat it as a different server.
- **Manual Bridge setting:** these automatic reconciliations do not authorize enabling Bridge when the user has manually turned bypass off. Preserve existing Xray/HEV/working Linux Bridge paths.
- **Status:** report whether Bridge transport is selected/active separately from whether underlying VLESS credentials and Internet traffic succeed. A successful Bridge session does not guarantee a working VPN subscription.

Use actual connection/profile-switch events instead of a high-frequency background network poll; do not derive trust solely from arbitrary camouflage SNI or URL. A registry match must refer to a trusted Bridge server identity and its TLS name, not merely an IP/DNS string that may collide. Do not implement a new transport or duplicate subscription account system. This decision governs remaining M6 enrollment, certificate storage, external socket/VpnService protect and Xray route integration. **Stage4 Android CSR self-check already PASS; multi-server selection, enrollment, reachability and Android E2E remain NOT_IMPLEMENTED / NOT_TESTED.** No executable code or running service changed in this documentation step.


### M6 — correction: offline server lookup and explicitly user-initiated certificate enrollment (2026-11-20)

**Supersedes the automatic-enrollment wording in the preceding M6 address-reconciliation checkpoint. Approved requirement, NOT YET IMPLEMENTED / NOT TESTED.**

1. Current server identity/address is obtained **locally** from the selected, already stored VPN profile/subscription metadata, not by DNS lookup or reaching the Bridge server. The app compares it with a locally cached, trusted registry of known Bridge identities. A broken DNS name or unreachable network cannot prevent this local comparison. DNS/transport reachability is tested only when trying to connect, independently of local identity lookup.
2. **Bridge manually OFF:** changing protocol, server, or subscription must NOT start Bridge, attempt Bridge DNS/reachability checks, register an identity, or request a certificate. A subscription/server supporting Bridge does **not** automatically activate or enroll the client.
3. **User manually requests Bridge activation:** resolve active profile to a trusted Bridge server record locally. If it is unknown or has no Bridge support, remain OFF and show a clear reason; do not attempt enrollment against unknown endpoints. If known with valid stored certificate, use it. If a known server lacks a valid certificate, show a registration action; **CSR submission, registration and issuance happen only after the user explicitly presses the registration button**, never on profile changes or automatically on Connect.
4. **Bridge already ON and protocol/server/subscription changes:** re-read the active local address. Same known server: no action or unnecessary process restart. Different known server with a stored valid certificate: automatically select that certificate and matching config (do not re-enroll). Different known server with missing/expired certificate: do NOT auto-enroll; notify user and require manual registration/renewal. Unknown server: switch Bridge OFF and let the normal connection path operate.
5. **Known server but DNS/host unreachable:** preserve known identity, certificate and registration, show a connection error. Do not delete keys, fall back to a certificate from another server, or treat DNS failure as an unknown server.
6. An optional network probe may be part of actual Bridge connection activation, but **no continuous scanning/polling** and no unsolicited network requests while Bridge is OFF. Reachability and selected-server identity are distinct states.

Security: trust is established from verified server metadata, not from arbitrary attacker-controlled URL or camouflage SNI. The per-device P-256 private key remains independent from subscriptions; distinct per-server mTLS certificates are retained. Existing Stage4 Android identity/CSR self-check remains PASS and unchanged. **No implementation or server service changes** in this documentation-only correction.


### M6 — offline trusted Bridge-server reconciliation source and tests committed (2026-11-20)

Operator explicitly approved the manual-enrollment/offline-address-lookup design immediately above and authorized implementation in the current M6 module. Working branch: `work/m6-android-ingress`. Added `integrations/android-ingress/stage4/BridgeServerReconciler.kt` (commit `d98e661ff946af43ae6e3153c934785fede556a8`) and `BridgeServerReconcilerTest.kt` (commit `cd4e0929767346512ce880da959f0cf62e56b00c`). This is an **isolated pure Kotlin policy component**, not yet integrated into the app, and has NO DNS, network, API, file, signing or VPN side effects.

Decision cases implemented: Bridge manually OFF => IDLE_OFF (before profile lookup); current known endpoint with valid saved certificate => KEEP_CURRENT; different known endpoint with valid saved certificate => SELECT_SAVED; known endpoint without current valid cert => REQUEST_USER_REGISTRATION **without issuing one**; unknown => TURN_OFF_UNKNOWN; invalid endpoint => TURN_OFF_INVALID_PROFILE; duplicate trusted mapping => TURN_OFF_AMBIGUOUS. Endpoint matches actual selected VPN destination host+port against a **caller-supplied trusted server catalog**; certificate lookup is keyed by stable trusted server ID and expiry. Unreachable DNS is not interpreted as unknown because the policy performs **no DNS resolution**. This is not an automatically trusted registry: authenticated catalog provisioning remains a separate required integration step.

Read-only source investigation of operator-provided actual 15-file Android archive: `BridgeRuntimeSettings.kt` contains only manual preferences; `BridgeConfigManager.kt` handles app-private config; `BridgeVpnIntegration.kt` validates local TCP target but does not alter Xray; `home_screen.dart` owns selection events and consults `ExternalServerSelection` / `SubParser`. These external service source files and complete active profile resolution are **not in the supplied archive**. It would be unsafe to guess those APIs or wire automatic route changes without inspecting them. Do **not** inject reconciliation into Bridge Settings alone; keep working VPN/HEV and release signing unchanged.

**Current status:** `RECONCILER_SOURCE_COMMITTED / TEST_SOURCE_COMMITTED`; `KOTLIN_TEST_EXECUTION_NOT_CONFIRMED`; `APP_WIRING_NOT_IMPLEMENTED`; `ON_DEVICE_SWITCHING_NOT_TESTED`; `ENROLLMENT_NOT_IMPLEMENTED`. A sandbox compiler exists but direct GitHub raw download failed due DNS, so there is no honest PASS from that attempt. Next gate: Kotlin/JVM compile/test exact GitHub sources in the existing SSH Homeserver checkout; after PASS inspect actual `lib/vpn/external_server_selection.dart`, `lib/vpn/subscription_service_name.dart`, related protocol selection and Xray start/pause event paths before wiring. No production services or Android APK changed by these GitHub commits.


### M6 — offline Kotlin reconciler compilation and isolated scenario test PASS on Homeserver (2026-11-20)

Operator executed exact source checkout `work/m6-android-ingress` up to `797904b` and compiled `integrations/android-ingress/stage4/BridgeServerReconciler.kt` + `BridgeServerReconcilerTest.kt` using the pre-existing Gradle-cached Kotlin compiler 2.3.20 under Java 25, without installing packages or affecting production services. Standalone compiler was initially invoked with incomplete classpath (`kotlin-reflect 2.3.20` missing; then `kotlinx.coroutines.CoroutineScope` missing). Operator corrected classpath using cached `kotlin-reflect 2.2.10` and `kotlinx-coroutines-core-jvm 1.8.0`. **Final observed evidence:**
```text
COMPILE_EXIT_CODE=0
M6_OFFLINE_BRIDGE_RECONCILIATION=PASS
TEST_EXIT_CODE=0
```
Java 25 native-access/Unsafe warnings were nonfatal. **Status: ISOLATED_RECONCILER=PASS** (source compilation + local scenario assertions). This does **not** establish Android app event wiring, actual trusted catalog loading, real certificate storage, live DNS reachability, enrollment, Xray routing, APK/device E2E or remote server connection. Do not repeat this successful isolated test without relevant source changes. Next step in this SAME M6 module: inspect actual active-profile selection APIs in `/opt/tervonvpndpi` read-only before implementing minimal event hookup and separate per-server certificate storage. Never guess source APIs or alter running services.


### M6 — actual Android profile flow reviewed (operator-provided 820-line source extracts; 2026-11-20)

**Source review completed; no app integration performed in this checkpoint.** Operator supplied `m6-profile-flow-review.txt` extracted directly from `/opt/tervonvpndpi`, including `external_server_selection.dart`, `sub_parser.dart`, `home_screen.dart`, `xray_service.dart`, and `VpnForegroundService.kt`. These are actual app source excerpts, not guessed APIs.

Observed:
- `ExternalServerSelection.selected(index)` uses SharedPreferences key `external_subscription_selected_server_<index>`; `select(index,url)` persists the exact VPN node URI. `forNative(resolvedRaw,index,protocol)` requires a manually selected node; if the saved URI disappeared, it may match a replacement **by URI host only** and replace the saved URI. That host-only fallback is NOT a trusted Bridge identity or enrollment authorization.
- `home_screen.dart::_chooseExternalServer`: verifies selected link is in active downloaded subscription, detects protocol, calls `XrayService.saveDisplayedExternalNode(index,chosen,protocol)`, hot-restarts Xray if already connected, *then* commits `ExternalServerSelection.select(index,chosen)` and updates UI. On error it attempts to rollback native profile and restart Xray. **Do not route Bridge for a newly chosen server before this transaction succeeds or leaves native config consistent.**
- `_tervonChooseSubscription`: stops a currently running VPN before resolving and activating the new external subscription. Restores its per-subscription node selection, or clears stale selection. No node may be selected yet; Bridge must remain manually OFF or defer activation as appropriate, never enroll automatically.
- `_toggleConnection`: already verifies selected node belongs to the active subscription and protocol exists before starting VPN. Profile validation can be inserted in the existing start transaction but **must not break the OFF path or native rollback**.
- `XrayService.saveDisplayedExternalNode` validates active subscription and protocol and passes exact chosen URI to native `saveSubscription`. `downloadAndSaveConfigCached` calls `ExternalServerSelection.forNative`. `VpnForegroundService.startVpnInternal` loads current Xray config and starts TUN, libv2ray and HEV; no actual Bridge routing currently integrated.

**Technical consequence / unresolved integration ordering:** The existing hot-switch flow restarts Xray before committing the selected node. An asynchronous Bridge process/certificate switch after this point could temporarily send live traffic through the wrong Bridge. The integration must coordinate profile -> Bridge selection -> native restart, with explicit rollback and no external network work while manual Bridge is OFF; this must be implemented **after examining full switch/connect handlers and current Flutter↔Kotlin BridgeControlChannel/BridgeRuntimeControl signatures**. Do not assume the excerpt includes the full required method bodies, bridge toggle preferences, or protocol switch handlers. No new module started; same M6. `ISOLATED_RECONCILER=PASS`; `REAL_PROFILE_EVENT_HOOKUP=NOT_IMPLEMENTED`; `APK_BUILD=NOT_RUN`; `DEVICE_E2E=NOT_TESTED`. No production services touched.


### M6 — full profile/Bridge integration archive inspected; route integration gap confirmed (2026-11-20)

Operator supplied `m6-event-integration-review.tar.gz` (six actual Android/Flutter source files). Completed source review and recorded exact integration ordering in `integrations/android-ingress/M6_PROFILE_INTEGRATION_REVIEW.ru.md` (commit `e9585fa742617fb0f7e6f34766cf47a104f49cec`). Confirmed `BridgeRuntimeControl.status().vpnRouteIntegrated=false`, `BridgeConfigManager` currently supports one config and no per-server PKI registry, and Bridge Settings still only saves a preference flag. Also confirmed `_chooseExternalServer` hot-restarts Xray **before** persisting selection and has rollback logic. A post-restart Bridge switch would be unsafe and give a false claim of functional bypass. Preserve Xray/HEV/VpnService until trusted catalog, per-server PKI, authenticated manual enrollment, route/sock protect and coordinated rollback are implemented. **Status: PROFILE_SOURCE_REVIEW=COMPLETE; ISOLATED_RECONCILER=PASS; ACTUAL_BRIDGE_ROUTE=NOT_IMPLEMENTED; MULTISERVER_STORAGE=NOT_IMPLEMENTED; APP_EVENT_INTEGRATION=NOT_IMPLEMENTED.** This checkpoint changed documentation in GitHub only; no production server or app files touched.


### M6 — staged app-private per-server mTLS certificate store source (2026-11-20)

In the same M6 module, added candidate `integrations/android-ingress/stage4/BridgeServerCertificateStore.kt` (latest commit `93acd8fa5ad362d501b265c4c748f733f780c69c`) after inspecting actual Android BridgeRuntimeControl, BridgeConfigManager, Flutter screen and hot-switch source. The candidate uses existing Android APIs `AtomicFile`, `CertificateFactory` and `android.util.Base64` without new dependencies or runtime/process changes. It writes individual client certificates to app-private `noBackupFilesDir/bridge-server-client-certificates-v1`, indexed by SHA-256 of a **caller-provided verified trusted Bridge server ID**, and validates client-cert validity, issuer against pinned CA, clientAuth EKU, and SPKI equality to Stage4 device public key before storing/reading. It never writes/export private keys, never generates keys, does not use DNS, does not auto-enroll, and retains expired certificates for manual renewal. **Critical external prerequisite:** authenticated enrollment must provide the actual pinned CA and trusted server ID; do not feed these from untrusted subscription/SNI. This is staged source only: **CERT_STORE_SOURCE_COMMITTED / ANDROID_BUILD_NOT_RUN / CERT_STORE_TEST_NOT_RUN / NOT_INTEGRATED**. Reconciler isolated Kotlin/JVM tests remain PASS; do not claim new store passes or that VPN routing is active. No running services touched, no changes to installed APK.

Next gate within M6: verify/stage the store against the actual Android app source using a guarded, minimal integration and Android build/test, then implement trustworthy catalog and user-button enrollment; only then add coordinated profile switching and real Xray route/socket protect. Do not wire automatic profile changes while `vpnRouteIntegrated=false`.


### M6 — guarded single-file Android certificate-store installer prepared (2026-11-20)

Added `integrations/android-ingress/stage4/install_cert_store.py` (commit `6007b3268f272a1d885b6a5c6380bb74c45298cc`). It stages ONLY `BridgeServerCertificateStore.kt` to `/opt/tervonvpndpi/android/app/src/main/kotlin/com/tervonvpndpi/app/` after `--check`, refuses to overwrite a differing existing file, and leaves APK, VPN config, production services, signing and release untouched. Suggested operator sequence: fast-forward the SSH Git checkout; `python3 .../install_cert_store.py --app-root /opt/tervonvpndpi --check`; if READY, run `--apply`; then compile existing Android project `cd /opt/tervonvpndpi/android && ./gradlew :app:compileDebugKotlin --no-daemon`. **Until operator provides output, stage and build remain NOT_CONFIRMED.** This is isolated source staging for validation, NOT an active Bridge route.


### M6 — Android debug Kotlin compilation after certificate-store staging step (2026-11-20)

Operator reports `BUILD SUCCESSFUL in 48s` and `72 actionable tasks: 67 executed, 5 up-to-date` following the instructed `BridgeServerCertificateStore.kt` staging/Android `:app:compileDebugKotlin` workflow. **ANDROID_DEBUG_KOTLIN_BUILD=PASS** for the current Android working tree. Operator did not provide the separate `install_cert_store.py --check/--apply` result; therefore **CERT_STORE_FILE_PRESENT / INSTALL_APPLIED=NOT_INDEPENDENTLY_CONFIRMED** from this output alone, and no direct certificate-store behavior test has run. This is NOT a certificate issuance/ACL/mobile VPN route/E2E PASS. Do not rerun the same compile without changed code. Next stage remains trusted server catalog and explicitly user-initiated authenticated enrollment, before route switching. No production services affected by the reported build.


### M6 — signed Bridge catalog v1 isolated implementation (2026-11-20)

The operator confirmed the previously agreed **option 2: signed, remotely updatable Bridge server catalog**; do not ask for this architectural choice again. Same active branch `work/m6-android-ingress`. New source-only commits: Android `integrations/android-ingress/stage4/BridgeSignedServerCatalog.kt`, existing Bouncy Castle Ed25519 verification over exact domain-prefixed payload bytes, strict payload metadata/server endpoint parsing, `server_ca_pem` clearly separate from client-issuer CA, offline `AtomicFile` cache, same-serial conflict prevention, rollback prevention, 30-day maximum validity, and signed zero-server revocation; `integrations/android-ingress/xui/catalog/sign.go` (Go stdlib Ed25519 offline signer) and `sign_test.go` (tamper/wrong-key/domain separation/invalid request tests); guarded `stage4/install_signed_catalog.py` copies only the signed catalog and previously tested `BridgeServerReconciler.kt` without overwriting different app files; specification `integrations/android-ingress/SIGNED_CATALOG_V1.ru.md`. **No production signing key was generated or published; no trust anchor is yet pinned in APK.** Signature verification always requires caller-supplied externally pinned trusted Ed25519 public key; do not accept fetched key or arbitrary VPN subscription as trusted root.

Local independent Go 1.23.2 race-detector test using an **equivalent copy** of the signer and test cases (not the exact freshly fetched GitHub files): `go test -race -count=1 -v ./...` => two test functions and six invalid-input subcases PASS. GitHub CI run / exact-repo tests **NOT_CONFIRMED**; Android stage and compile **NOT_RUN**, catalog fetch/rotation **NOT_IMPLEMENTED**, per-server certificate enrollment **NOT_IMPLEMENTED**, Xray route and socket protect **NOT_IMPLEMENTED**. Existing Stage4 Android CSR PASS, isolated reconciler PASS and prior certificate-store staging compile PASS remain unchanged. Next install pure source only using guarded --check/--apply, compile Android once; separately agree a secure production signing-key provisioning process before adding any network updater or profile-event wiring. Do not modify working HEV, Linux Bridge v1.0.1, Xray, Docker or published APK.


### M6 — signed catalog Android Kotlin compile PASS (2026-11-20)

Operator reports after requested signed-catalog staging step: `BUILD SUCCESSFUL in 23s`, `72 actionable tasks: 6 executed, 66 up-to-date` from the existing Android Gradle compilation workflow. **SIGNED_CATALOG_ANDROID_COMPILE=PASS** for the current working tree. The separate output of `install_signed_catalog.py --check/--apply` was not provided, so **SIGNED_CATALOG_FILES_INSTALLED=NOT_INDEPENDENTLY_CONFIRMED** by this report alone. This compilation is not a signed-envelope runtime verification test, Android trust anchor activation, published APK, enrollment or actual VPN route PASS. Previous successfully compiled checks are not repeated. The next gate remains production signing-key protection/public-key pinning design and trusted signed catalog delivery, with operator approval of any new architecture/security mechanism before implementation. Live Xray, HEV, Linux Bridge and Docker left untouched.
