# M5 — Linux v1.0.1: финальная приёмка

**Утверждено оператором 2026-11-19:** M4 — уже работающий Linux Tervon TCP Bridge **v1.0.1**, установленный на Homeserver и Finserver. M5 проверяет именно этот экземпляр без смены бинарников или конфигурации. M6 — `HOLD / DISCUSSION_REQUIRED`.

## Статусы и порядок

| Пакет | Приёмка | Статус |
|---|---|---|
| M5.1 | Фиксация M4=v1.0.1, source commit, бинарного SHA и ранее подтверждённой рабочей VPN/Stats | **CLOSED / PASS** |
| **M5.2+M5.3** | **ОДНА объединённая проверка** TCP correctness + Padding/Jitter (≤4 мс на кадр; ≤20 мс за 1 сек) + короткие и длинные потоки, конкурентность/fairness, RTT, CPU/RSS/goroutines | **OFFLINE 6/6 PASS + PASSIVE LIVE PASS + ADAPTIVE POLICY 6/6 PASS / REAL NETWORK PENDING** |
| M5.4 | Сверка надёжности и rollback с уже пройденными M4; проверка только отсутствующего | NOT_STARTED |
| M5.5 | Проверяемый GitHub release package для **того же** M4=v1.0.1 | NOT_STARTED |
| M5.6 | Единственный официальный GitHub Release `v1.0.1` | NOT_STARTED |

## Принцип объединённой проверки

Один пакет, один отчёт и однозначный итог: `PASS` только при всех обязательных доказательствах, иначе `FAIL` или `BLOCKED` с конкретной причиной. **Нельзя программно обещать 100% PASS до запуска или считать работоспособный VPN доказательством любого возможного численного свойства.** Но не повторяем ранее пройденные M4 проверки ради бюрократии.

- **M4 evidence accepted:** операторский PASS VPN и Stats, установленные идентичные бинарники с SHA256 `5035217a89476117894efe1a44db5dcdd1242773d7fd9461170f2cf6ccb98a90`; source `8d0aa3777c209d2825a1c11dfbae9487893cb7f5`, M4 closure `eb4fde88d6477defb58f8237fd0526c292e93cfe`.
- **Source regression check:** только исходники M4, unit/integration/race тесты транспортного слоя, Padding/Jitter и проверка ограничений политики. Изолированное исполнение не является повторной установкой, не затрагивает работающую службу.
- **Численные Jitter:** тестовые значения решения `jitter.Observe` должны быть ≤4 мс на кадр и суммарно ≤20 мс в отдельном фиксированном секундном окне на поток/направление. Отдельно учитывать, что таймер/планировщик ОС не гарантирует точно такую же задержку в сетевом RTT.
- **Реальная сеть и ресурсы:** для RTT, fairness и CPU/RSS/goroutines требуются реальные показания на соответствующем уровне. Публичный Stats screenshot подтверждает работу сервиса и наблюдаемые счётчики, но не максимумы каждого кадра, распределение RTT или отсутствие starvation во всех сессиях. Не создавать фиктивные наблюдения; результат `BLOCKED`, если обязательные измерения пока не доступны.
- **Безопасность:** никаких остановок/рестартов `Bridge`, `Stats`, `nginx`, `x-ui`, Xray и других работающих служб; никаких private configs, сертификатов или ключей в GitHub. Трафик пользователей не генерировать без явно согласованного разрешения.

## Объединённый скрипт

`tests/m5/run-m5-23-batch.sh` выполняет полностью изолированные Go тесты текущего M4-source и выводит `M5_23_OFFLINE=PASS` только если подтверждены все offline gates. Затем явно выводит `M5_23_FINAL=BLOCKED_LIVE_MEASUREMENTS`, пока нет независимого комплекта live measurements. Нельзя интерпретировать первый PASS как полный M5.2+M5.3 PASS. Один запуск/журнал. Вторая часть единой приёмки фиксируется только на основании фактических данных, не придумывается и не подменяется прошлой версией.

## Зафиксированные отмены

Любые сравнения с предыдущими версиями **отменены** решением оператора. Подготовленный ранее `tests/m5/run-m5-1-source-batch.sh` не запускать: он создан под неверную концепцию сравнительной пересборки. Исторические протоколы сохранены только для аудита.

## Следующее действие

Выполнить объединённый offline + evidence-inspection batch, получить один отчёт. Если обнаружится недостаток инструментирования для реальных RTT/Jitter/fairness, зафиксировать конкретный пробел **внутри объединённого пакета**, а не открывать другой модуль. M5.4 не начинать до подтверждённого PASS M5.2+M5.3.

## Результат единого прогона — 2026-11-19, подтверждён оператором

- Точно протестирован исходный код действующей M4=v1.0.1: M4 source `8d0aa3777c209d2825a1c11dfbae9487893cb7f5`; HEAD тестового checkout `3a6e1c5d0ce9be96fd5a2f7069c2256e65caf52f`; Go `go1.25.5 linux/amd64`.
- Время начала `2026-11-19T09:27:17Z`; журнал `/opt/tervon-tcp-bridge/m5/results/m5.23-8xLZEb4p/summary.log`.
- **Операторский PASS 6/6:** `M5_23_VET=PASS`, `M5_23_BUILD=PASS`, `M5_23_INTEGRATION=PASS`, `M5_23_RACE=PASS`, `M5_23_JITTER_REPEAT=PASS`, `M5_23_FRAMING_REPEAT=PASS`; `M5_23_OFFLINE=PASS`. Предыдущее свидетельство работающего VPN/Stats: `M5_23_LIVE_TCP=OPERATOR_M4_PASS_EXISTING`.
- Конечный `M5_23_FINAL=BLOCKED_LIVE_MEASUREMENTS` и exit code 3 — **предусмотренная блокировка критериев, а не сбой тестов или сервиса**. Скрипт устанавливает этот код намеренно при отсутствии реальных численных наблюдений.
- **Осталось измерить, не повторяя офлайн-прогон:** сетевой RTT, отсутствие starvation/fairness конкурентных потоков, реальный профиль CPU/RSS/goroutines, максимум задержек Jitter в live path (либо чётко разграничить строго доказанный policy bound и непроверенные wall-clock задержки). Не выдавать full PASS без этих данных.
- Активный объединённый этап `M5.2+M5.3=OFFLINE_PASS / LIVE_MEASUREMENTS_PENDING`. M5.4 остаётся NOT_STARTED; M6 HOLD. Работающие службы не изменялись.

## Короткая пассивная проверка v1.0.1 — подготовлена, ещё не выполнена

`tests/m5/run-m5-23-live-short.sh`: один 60-секундный read-only прогон существующего M3 public dashboard sampler (6 интервалов по 10 секунд), проверка SHA и неизменности PID действующего M4-бинарника, снимки RSS/CPU, динамика сессий/счётчиков Padding/Jitter/ошибок с обеих нод. Не генерирует пользовательский TCP, не трогает сервисы/конфигурации; не повторяет 6/6 уже пройденных тестов. Пассивная телеметрия доказывает наблюдаемую стабильность за 60 секунд, **не** RTT, fairness или max per-frame/window jitter. `SOURCE_COMMITTED / LIVE_NOT_RUN`.

## М5.2+М5.3 — 60-секундное пассивное испытание: OPERATOR PASS (2026-11-19)

Оператор выполнил `tests/m5/run-m5-23-live-short.sh`. Источник: Homeserver, начало `2026-11-19T09:33:48Z`, журнал `/opt/tervon-tcp-bridge/m5/results/m5.23-live-sPZnEtyL/evidence.jsonl`; код возврата `0`. `M5_LIVE_RUNNING_M4_SHA=PASS`, `M5_LIVE_SAMPLER=PASS_PASSIVE_TWO_NODE_COUNTER_CONTINUITY`, `M5_LIVE_SAMPLES=7`, `M5_LIVE=PASS_PASSIVE_OBSERVATION`.

| Показатель | Homeserver | Finserver |
|---|---:|---:|
| Активные сессии до → после | 17 → 15 | 17 → 15 |
| Новые сессии | 7 | 7 |
| Прирост ошибок | 0 | 0 |

Параметры процесса Homeserver: CPU `0.6% → 0.6%`; RSS `20644 → 21128 KiB` (+484 KiB). За минутное окно `M5_LIVE_JITTER_FRAMES=0`, `M5_LIVE_PADDING_FRAMES=0`, поэтому реальное срабатывание адаптивных механизмов этим прогоном **не проверено**. Измерения реального RTT/fairness/максимума задержки на кадр `NOT_MEASURED`. Прежний source suite **6/6 PASS**, эксплуатация VPN/Stats ранее оператором подтверждена. **Текущая приёмка: OFFLINE_PASS + PASSIVE_LIVE_PASS; количественные Jitter/RTT/fairness ещё не PASS.** Никаких перезапусков или замен работающих служб.

## M5.2+M5.3 — пакет численного тестирования адаптивного Jitter (ГОТОВ, НЕ ЗАПУЩЕН)

`tests/m5/run-m5-adaptive-batch.sh` копирует **неизменённую M4 policy.go** в отдельный каталог и запускает дважды под `-race` три изолированных теста из `tests/m5/m5_adaptive_policy_test.go`: (1) sustained eligible traffic со строгими 4 мс/кадр и 20 мс/сек на виртуальном времени, (2) исключение коротких запросов, перегрузки и idle, (3) восемь независимых конкурентных состояний. Тесты не генерируют live VPN-трафик и не перезапускают службы. Отдельно от политики **реальные RTT и фактическая wall-clock delay по сети этим пакетом НЕ измеряются**; даже `M5_ADAPTIVE_POLICY=PASS` не является автоматическим полным PASS M5.2+M5.3. `SOURCE_COMMITTED / OPERATOR_RESULT_PENDING`.

## Численная приёмка Jitter policy — OPERATOR PASS (2026-11-19)

- Проверка `tests/m5/run-m5-adaptive-batch.sh`, checkout HEAD `87b2f2385813c5d82b6c15dfbe6c559248e0dea4`, ровно неизменённый M4 policy source `8d0aa3777c209d2825a1c11dfbae9487893cb7f5`; запуск Homeserver `2026-11-19T09:40:35Z`.
- Под `go test -race -count=2` завершились **6/6 PASS**, exit 0: `TestM5SustainedWindowCaps` ×2; `TestM5InteractiveBacklogAndIdleBypass` ×2; `TestM5CompetingStreamsHaveIndependentBudgets` ×2.
- В sustained тестах по виртуальному времени `eligible_frames=527 / 529`, `max_frame_ns=3969448 / 3994250` (оба ≤4000000), `max_window_ns=20000000` (≤20000000). Интерактивный/idle/backlog обход PASS; 8 независимых jitter-состояний PASS. `M5_ADAPTIVE_POLICY=PASS`, `M5_ADAPTIVE_EXIT_CODE=0`.
- Журнал `/opt/tervon-tcp-bridge/m5/results/m5-adaptive-6bhWTQVD/tests.log`.
- **Подтверждены строгие численные лимиты решения policy**, но не фактические сетевые wall-clock задержки, real RTT и fairness между конкурирующими TCP-сессиями. Общий статус объединённого этапа: `OFFLINE_TCP_PASS + LIVE_PASSIVE_PASS + ADAPTIVE_POLICY_PASS / REAL_NETWORK_MEASUREMENTS_PENDING`. Изменений работающих служб нет.

## M5.2+M5.3 — сетевой loopback пакет с реальным TLS1.3 / HTTP2 (подготовлен, не выполнен)

Созданы `tests/m5/m5_loopback_network_test.go` и `tests/m5/run-m5-network-batch.sh`: проверка **неизменённого M4** на настоящих loopback TCP сокетах с тестовыми mTLS сертификатами, /v2/stream, активными Padding/Rate/Jitter policies, четырьмя конкурирующими длительными потоками по 128 кадров, 24 короткими запросами на их фоне, byte-exact echo и наблюдением Jitter/PAD counters. Замеряются application roundtrip p50/p95/max, завершённость всех потоков и коротких запросов, диагностический Go heap. `p95 ≤1s` — только локальный test threshold против зависания, **не** допуск Internet RTT. Пакет создаёт всё в одноразовом каталоге; никаких серверных конфигураций, service restarts, действующего трафика. `SOURCE_COMMITTED / OPERATOR_NOT_RUN`.

**Разделение доказательств:** jitter policy 4/20ms уже PASS, а фактические RTT на Homeserver↔Finserver и end-to-end межпоточная fairness действующих пользователей этим loopback тестом не измеряются. Local loopback RTT — проверка транспортной реализации и отсутствия starvation в контролируемой среде. Финальное заключение по реальной межсерверной сети допускается только по её наблюдениям; не подменять её loopback.

## M5 loopback first run — 2026-11-19: PARTIAL PASS / test correction needed

First run 4×128-frame loopback flows and 24 short probes completed successfully; p50/p95/max=1.516/2.216/2.614 ms; client/server PAD frames=212/208, byte exact and 4/4 fairness observed. `JitterUploadFrames=0`, `JitterDownloadFrames=0`, test exited 1 due to required activation. Review of `framing/stream.go`, `padding/rate.go`, `jitter/policy.go`: rate requires two completed 1s windows, jitter another 2s eligible sustained after rate activation. Original 3.45s run cannot reliably hit both thresholds. Adjusted **test only**, 320 frames and 25s timeout. Recheck pending, no production change. This is an observed TEST FAIL, not yet a production code defect.
