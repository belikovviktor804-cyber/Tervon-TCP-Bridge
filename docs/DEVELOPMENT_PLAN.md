# Tervon TCP Bridge — единый план разработки (v1.0.1 → Android)

**Статус:** утверждённый порядок работы, актуализировано 2026-11-19. **Единственный рабочий план для следующих изменений.**

> **M4 TLS13+H2 PRE-UPLOAD SECURITY GATE — CI PASS, 2026-11-19:** новый реальный тест обнаружил, что стандартный `net/http.Transport` может использовать HTTP/1.1 fallback даже при `NextProtos=["h2"]`; CI #44468361183 сначала правильно упал, потому что тестовый HTTP/1.1 handler получил запрос. Исправление `NewClient` через **штатный `crypto/tls.Dialer.DialContext` + проверку версии TLS 1.3 и ALPN `h2` прямо в `DialTLSContext` до передачи HTTP-заголовков/полезных VPN-данных**, с 10-секундным deadline и без мутации исходной TLS-конфигурации вызывающего кода. Проверки с TLS1.2-only и TLS1.3+h1-only серверами, byte-exact mTLS/HTTP2 и regressions: [Go checks #44468661056](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44468661056) **SUCCESS**, исходник `6814c57130326f5ecbbea7280f57117074eeee0b`; [M4 Linux candidate #44468661092](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44468661092) **SUCCESS (artifact only)**. **Все более старые M4-кандидаты до этого исправления НЕ ПРОДВИГАТЬ В РЕЛИЗ.** M4 исходники/CI по этому security gate PASS, но **HOST STAGING NOT RUN / PRODUCTION UNTOUCHED / M4 MODULE IN_PROGRESS / M5 NOT STARTED**. Текущая инструкция `docs/M4_ACCEPTANCE.md`.


> **M4 BATCH 3 RESOURCE CI PASS + GUARDED HOMESERVER STAGING READY (2026-11-19):** на `work/m4-http2-reliability` тест `prototype/m4_churn_test.go` прошёл `Go prototype checks #44466925122` **SUCCESS**: 3×8 авторизованных TLS1.3/h2 потоков, проверка байтов, нулевых остаточных активных сессий и возврата горутин. Отдельная `M4 Linux candidate #44467184331` **SUCCESS** (бинарник, SHA256SUMS, BUILD_INFO; без развёртывания). `tests/homeserver/stage-m4-candidate.sh` готов для *позднейшего однократного изолированного запуска* на Homeserver: только GitHub SSH, отдельный путь `/opt/tervon-tcp-bridge/m4`, точный SHA, build + `go test -race -run '^TestM4'`; проверка скрипта и полный Go CI `#44467642316` **SUCCESS** (commit `24e6352a76b9759afa0c1c07b3b71c8ad70053b9`). Документ `docs/M4_ACCEPTANCE.md` содержит запуск. **HOST STAGING NOT RUN; LIVE SERVICES UNTOUCHED; M4 STILL IN_PROGRESS; M5 NOT STARTED**. Подтверждённые M3 измерения сохраняются, недоказанные RTT/fairness лимиты всё ещё блокируют релиз в M5.


> **M4 BATCH 2 NETWORK FAILURE CI PASS / BATCH 3 CHURN PENDING — 2026-11-19:** isolated `prototype/m4_failure_modes_test.go` verifies timed TLS handshake stall, h2 response header stall with context cancellation, rejection of oversized HTTP2 response headers. `prototype/m4_long_lived_stream_test.go` verifies a valid TCP stream stays byte-exact across a data gap longer than response-header timeout; setup-time limits are not body timeouts. Commit `c2e44c058158324dcb900f066270739969f9fdc1`, GitHub [Go prototype checks #44466460566](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44466460566) **SUCCESS** (unit/race/build/vet and legacy regressions). Next `prototype/m4_churn_test.go` (SHA `9ce54ac557856e6596e04cf1f5141a0b020fe566`) exercises repeated authenticated h2 streams with M3 mechanisms and bounded drain/goroutine checks; CI #44466925122 pending and not falsely PASS. **M4 remains IN_PROGRESS, no host deployment and M5 not started.**


> **M4 HTTPS/2 RELIABILITY — FIRST SOURCE BATCH CI PASS (2026-11-19):** `work/m4-http2-reliability`, source SHA `35067ac3bc59837e606a9aa6d63803e523abb2c7`, [Go prototype checks #44464477666](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44464477666) **SUCCESS** (unit/race/vet/build, TLS1.2/ALPN downgrade rejection, header/idle limits and M1–M3 regressions). Защита: TLS 1.3 + HTTP/2 ALPN h2 с обеих сторон; TLS handshake 10s, response headers 15s/32KiB, HTTP server headers 32KiB, idle 60s, без абсолютного body timeout для долгих TCP VPN сессий. Тест admission fixture обновлён для TLS13/h2 после CI failure; проверка безопасности не ослаблена. **РЕАЛЬНЫЕ СЛУЖБЫ НЕ ИЗМЕНЯЛИСЬ; M4 BATCH1 CODE/CI PASS, M4 MODULE IN_PROGRESS**, ещё нужны controlled failure/resource/load и host acceptance. `docs/M4_ACCEPTANCE.md` — текущий чеклист, Linux v1.0.1 не выпущен.


> **АКТИВНЫЙ МОДУЛЬ M4 — 2026-11-19, решение оператора:** оператор подтвердил, что вопрос M3 Padding/Jitter закрыт, и явно поручил **перейти к M4**. Приняты подтверждённые M3 подэтапы: live PAD/Jitter (9 240 кадров PAD отправлено/принято; 1 781 Jitter задержанных кадров), нулевые ошибки, UI, восстановление SSH-туннеля и 180-секундные публичные телеметрические срезы. **M3 закрыт по решению оператора для перехода**, но недоказанные количественные пределы индивидуального Jitter (<=4мс/кадр и <=20мс/фиксированное окно на поток), реальные RTT, fairness и завершение контролируемых потоков НЕ считать техническим PASS: занести в обязательную финальную интеграционную приёмку **M5** до релиза v1.0.1. Этот явный перенос — исключение из исходного строгого последовательного требования, не фиктивные результаты. **Текущий модуль: M4 HTTPS/2 Reliability**, ветка `work/m4-http2-reliability` от последнего принятого M3 `b97ad886258d5668f40370bef73013cbcc59d7e0`. На M4 работаем только в GitHub; живые службы не трогать до отдельного контролируемого испытания. Никакого выпуска v1.0.1 до M5 PASS.


> **M3 PRODUCTION PAD/JITTER LIVE ACTIVATION — PASS 2026-11-19:** оператор выполнил `tervon-m3-pad-jitter-live.service` (`--duration 180 --interval 10 --require-activation`, UTC 05:08:53–05:12:28), 19 внешних HTTPS-срезов через Finserver, systemd успешно завершился, `PASS_PAD_AND_JITTER_ACTIVATED`, все error deltas 0. На Finserver отправлено **9 240 PAD кадров, 4 448 642 PAD payload bytes**, на Homeserver принято столько же; суммарные 18 480 — **двусторонний учёт тех же кадров**, а не уникальные пакеты. Finserver применил **1 781 Jitter задержку**, всего **1 519 999 202 нс** (среднее ~0,854 мс/кадр). Оба узла: начальных сессий 29, новых 70, завершённых 62, конечных 37; отклонено 0; Finserver backend dial success +70, failure +0. **Критерий реальной активации Padding/Jitter закрыт, повторять не нужно.** Для полного M3 остаются доказательство **настоящих end-to-end RTT и fairness** под совместным потоком/короткими запросами, реальных максимальных задержек **4ms/frame и 20ms/fixed-second-window на поток** (агрегаты не позволяют), и drain контролируемых сессий. **M3 IN_PROGRESS, M4 NOT STARTED, v1.0.1 NOT RELEASED.** Исторические записи ниже с `LIVE ACTIVATION PENDING` устарели.


> **M3 PAD/JITTER LIVE ACTIVATION — 2026-11-19 (CODE WRITTEN, LIVE NOT RUN):** после двух пассивных наблюдений с `PAD/Jitter delta=0` оператор потребовал реальной проверки активации. В `tests/homeserver/m3_passive_live_evidence.py` добавлен `--require-activation` с fail-closed статусами `PASS_PAD_AND_JITTER_ACTIVATED`, `NO_OBSERVED_PAD_OR_JITTER_ACTIVATION`, `PAD_ONLY_JITTER_NOT_OBSERVED`, `JITTER_ONLY_PAD_NOT_OBSERVED`; при отсутствии обоих механизмов нет ложного PASS. Добавлены mocked unit tests, инструкция `docs/M3_GATE_CLOSURE.md` по созданию непрерывной загрузки с **действующего TCP-VPN-клиента через Bridge**. Сам инструмент не создаёт трафик, не затрагивает службы и не измеряет per-stream RTT/fairness. **SOURCE CI PASS: Go prototype checks #44462201616 SUCCESS; M3 Linux candidate #44462201682 SUCCESS (artifact only). LIVE ACTIVATION PENDING; полный M3 PASS не заявлен.**


> **M3 PASSIVE TWO-NODE LIVE TELEMETRY PASS — 2026-11-19:** оператор выполнил `tervon-m3-passive-live.service` с `--duration 180 --interval 10`. 19 внешних HTTPS-срезов с Finserver, успешное завершение systemd unit и `PASS_PASSIVE_TWO_NODE_COUNTER_CONTINUITY`; оба узла: начальные active=17, финальные active=27, +61 начатых, +51 завершённых, отклонено 0, новые ошибки 0; 18 промежуточных срезов показали совпадение текущих active Homeserver/Finserver. Finserver: +61 успешных backend dial, +61 TLS1.3/H2/mTLS requests. **За весь интервал PAD/Jitter delta=0 на обоих узлах.** Это PASS непрерывности телеметрии, **не нагрузочный PASS**: реальные RTT, fairness, задержки 4 мс на кадр / 20 мс на фиксированное окно, полезный активированный Padding/Jitter и очистка сессий после контролируемой нагрузки не проверены. Повторять пассивный тест для подтверждения доступности не нужно. **M3 IN_PROGRESS / M4 NOT STARTED / v1.0.1 NOT RELEASED.**


> **M3 LIVE EVIDENCE SAMPLER — 2026-11-19 (SOURCE ONLY):** в текущей ветке добавлены `tests/homeserver/m3_passive_live_evidence.py` и изолированные `test_m3_passive_live_evidence.py`. Сбор за 180 секунд (19 внешних HTTPS-срезов) производится на Homeserver командой SSH к Finserver, который обращается только к публичному `stats.xrayvless.online:443` через закреплённый внешний IPv4 `87.103.198.219`, проверяет сертификат и реальность обеих нод. Код не изменяет working services, не содержит заглушек и не заявляет RTT/fairness из счётчиков. CI [Go checks #44460373997](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44460373997) **SUCCESS**, включая mocked parser/дельты; [M3 Linux candidate #44460374223](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44460374223) **SUCCESS (artifact only)**. **SOURCE TESTED; HOST SAMPLING NOT RUN; LIVE RTT/FAIRNESS GATE OPEN.** Инструкции: `docs/M3_GATE_CLOSURE.md`. SSH recovery уже LIVE PASS, повторять его не нужно. 


> **LIVE M3 TUNNEL RECOVERY PASS — 2026-11-19:** оператор выполнил detached `tervon-m3-tunnel-recovery.service` на Homeserver с исходником `dfa2bfc1`. `SIGTERM_MAINPID_ONLY`: SSH PID `1474240 → 1672169` за 6с; `M3_TUNNEL_RESTART=PASS`; Finserver подтвердил публичный двухузловой HTTPS API `M3_TUNNEL_RECOVERY_PUBLIC_API=PASS` через 8с. `BRIDGE_RESTART=NONE`, `STATS_RESTART=NONE`, `NGINX_RELOAD=NONE`; испытательный юнит завершился успешно. **Туннельный критерий M3 ЗАКРЫТ и не требует повторения.** Остаётся единственный финальный M3 gate — воспроизводимая длительная реальная нагрузка с RTT/fairness, проверкой границ Jitter 4ms/frame и 20ms/fixed window на поток и отсутствия зависших сессий; синтетические CI-тесты не подменяют live evidence. До этого **M3 IN_PROGRESS, M4 NOT STARTED, Linux v1.0.1 NOT RELEASED**. Старые исторические записи ниже с `recovery pending` не являются актуальным статусом.


> **M3 GATE CLOSURE TOOLING — 2026-11-19:** после поручения продолжать и актуализировать документацию в GitHub подготовлены `tests/homeserver/m3-managed-tunnel-recovery.sh` (только systemd SSH metrics tunnel, 45с, detached, восстановление при ошибке, а после смены SSH PID — автоматическая проверка публичного HTTPS API обоих узлов **с Finserver через 87.103.198.219**; **НЕ ЗАПУЩЕН на Homeserver**) и `docs/M3_GATE_CLOSURE.md` (операторский сценарий и внешний HTTPS API контроль с Finserver). Усилен `prototype/jitter/fairness_test.go` синтетической 30-секундной многопоточной проверкой фиксированных окон 4ms/frame / 20ms/window; эти тесты не подтверждают реальный RTT. CI **SUCCESS**: Go checks #44455023918, M3 Linux candidate #44455023901 (только артефакт). Host recovery fault injection и длительная нагрузка не выполнялись. **Финальный M3 PASS пока не получен; M4/M5/M6 не начинались.** Не повторять принятые проверки панели. 

> **ЗАПРОС ПЕРЕХОДА К M4–M6 (2026-11-19):** оператор подтвердил работоспособность публичной двухузловой Stats-панели и потребовал продолжить M4, M5, M6. **Порядок выпуска из этого документа остаётся M3 → M4 → M5 → Linux v1.0.1 → M6 Android**; M6 не является обязательным условием Linux v1.0.1. **M3 не имеет полного финального PASS** по двум ранее установленным обязательным критериям: (1) подтверждение managed SSH tunnel recovery после разрыва; (2) продолжительные измерения под нагрузкой/RTT/fairness, в том числе per-stream bound 4ms и 20ms/сек. Имеются только отдельные PASS: live M3 data plane, счётчики PAD/Jitter, managed tunnel startup, публичный API, browser UI. Не смешивать эти PASS с полным завершением M3 и не выпускать v1.0.1 раньше M5. Рабочие службы не трогать.


> **АКТИВНАЯ ЗАДАЧА M3 UI (2026-11-19):** устранение текстовых заглушек на публичном сайте. В GitHub изменены `bridge_panel.html`, `bridge_panel.js`, `test_bridge_panel.cjs`: удалены фиктивные Homeserver-карточки Backend dial/TLS/latency; реальные Backend dial и TLS остаются Finserver, добавлена измеренная Finserver latency p50/p95/p99 с явным количеством наблюдений (0 замеров, если измерений нет). **CI SUCCESS; LOCAL UI PASS; PUBLIC BROWSER VISUAL PASS (operator screenshot 2026-11-19 10:30).** UI-only установка уже выполнена через `update-m3-ui.py`: `M3_STATS_UI_LOCAL=PASS`, `M3_UI_FIX=PASS`, backup `/opt/tervon/bridge-stats/.ui-backups/m3-ui-nz5ji82x`; Bridge, Stats, nginx, SSH-туннель не перезапускались. Публичный браузер показал обе секции и реальные счётчики: Homeserver 29 сессий, ошибки 0; Finserver 31 сессия, Backend dial 1035/0, TLS 1035, p50/p95/p99 1/1/1 мс (1035), Jitter 357 кадров / 407.55 мс. UI gate закрыт, не проверять и не устанавливать повторно. Остальные M3 критерии ниже неизменны.


> **ТЕКУЩИЙ M3 GATE — 2026-11-19: PUBLIC TWO-NODE API PASS.** Homeserver + Finserver M3 `rate1/jit1` действуют, PAD/Jitter live-счётчики подтверждены на двух узлах. `tervon-bridge-fin-metrics-tunnel.service` через OpenSSH доступен как `active/enabled` и ранее выдержал cutover-v2 PASS. Stats двухузловой API локально и **через внешний IPv4 87.103.198.219:443** с Finserver успешно подтверждён: TLS с сертификатом на `stats.xrayvless.online` валиден, HTTP/2 200, `homeserver=ok (26 sessions)`, `finserver=ok (26 sessions)`, `M3_PUBLIC_TWO_NODE_API=PASS`. Бэкап Stats: `/opt/tervon/bridge-stats/.two-node-backups/m3-ty70nt1x`. GitHub CI после коррекции `ClearAllForwardings=no` — SUCCESS. **Осталось:** проверить две секции в браузере и статические assets, автоматическое восстановление systemd SSH после разрыва, длительные latency/fairness и per-stream PAD/Jitter limits. Не переустанавливать работающие службы, не переходить к M4, не выпускать v1.0.1 без итоговых испытаний. Прежние записи ниже — исторические.


> **ПОСЛЕДНЕЕ ОПЕРАТОРСКОЕ ПОДТВЕРЖДЕНИЕ — 2026-11-18, M3 Stats installed:** Homeserver SSH checkout `work/m3-adaptive-jitter` обновлён fast-forward до `5015a3b6f5eff704aa0cd317ecb69228f49b4c80`. Оператор запустил **последовательно** `integrations/stats/update-m3-ui.py` и `update-m3-collector.py`. Реальный вывод: `M3_STATS_UI_LOCAL=PASS`, `UI_BACKUP=/opt/tervon/bridge-stats/.ui-backups/m3-ui-6th3s12p`; `M3_COLLECTOR_LOCAL=PASS`, `COLLECTOR_BACKUP=/opt/tervon/bridge-stats/.collector-backups/m3-00et2d9c`; `BRIDGE_RESTART=NONE`, `STATS_RESTART=YES` (только во втором сценарии), `NGINX_RELOAD=NONE`, `EXTERNAL_HTTPS=NOT_VERIFIED`. **Состояние: M3_STATS_UI_LOCAL=PASS, M3_STATS_COLLECTOR_LOCAL=PASS (operator evidence), старый работающий M2 не менялся; внешний HTTPS и M3 rate1/jit1 LIVE_NOT_VERIFIED.** Не выполнять повторно установку метрик и не требовать прежние шаги. Оставить обе резервные копии на месте. При прямой попытке внешнего HTTPS-чтения инструментарий не получил доступный ответ; это не является ни PASS, ни доказательством отказа сервиса. **Текущий модуль M3 IN_PROGRESS, не RELEASED; следующий gate — реальный data-plane M3 на двух узлах с backup/rollback и измерениями.** Примечания ниже о `HOST NOT RUN` и `UI NOT INSTALLED` — только исторические снимки до этого подтверждения.


> **M3 STATS CODE PASS (2026-11-18; production unchanged):** полный [Go prototype checks #44308674234](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44308674234) — **SUCCESS, 36/36** на точном кодовом SHA `1661d93000f11dfe74a12543efffd23df04084e1`. Проверены отдельно новые `update-m3-collector.py` + `test_update_m3_collector.py` (включая mock rollback), UI и прежние Go M1/M2/M3 тесты/race/fuzz. Collector читает только локальные источники без прокси/HTTP redirect, не публикует сырой endpoint; допускает ограниченный restart **только Stats** при операторском запуске, восстанавливает файл при ошибке. UI-only скрипт ранее прошёл CI. **Это CODE/CI PASS, не INSTALL/LIVE PASS**. На Homeserver M2 и его ранее подключённые метрики продолжают работать; M3 rate1/jit1 live, независимый внешний HTTPS и полный мониторинг остаются NOT_VERIFIED. Не объявлять M3 завершённым до живого двухузлового и Stats acceptance.


> **M3 Stats collector gate (2026-11-18):** На фоне работающего M2 и ранее подтверждённого `MONITORING_LOCAL=PASS` создан отдельный `integrations/stats/update-m3-collector.py` и offline тесты `test_update_m3_collector.py` (коммиты `5ffebb2`, `12f49f6`, `c69f026`, `7e5bc2a`, `681117e`). Это **CODE PREPARED, NEW CI PENDING, HOST NOT RUN**. Он не касается Bridge/Finserver/nginx; при отдельно согласованном применении может перезапустить только существующий `tervon-bridge-stats.service`, с backup/rollback и ограниченной проверкой. После PASS CI: UI-only обновление → отдельно Stats collector → реальная внешняя проверка → двухузловые тесты M3 rate1/jit1. До подтверждённых живых результатов M3 IN_PROGRESS, v1.0.1 NOT RELEASED.


> **M3 UI corrected CI gate PASS (2026-11-18):** [Go prototype checks #44306965079](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44306965079) completed **SUCCESS** at exact implementation SHA `ee68d50f2910d3125ce39e091d0ab3d0d3099db4`: all 36 steps succeeded, including stats Python, Node panel cases (configured/unknown OFF), rollback tests, Go build/vet/unit/race and M1/M2/M3 regressions/fuzz. Previous FAIL on `0038e082` was an outdated JS fixture expectation (`Активно / Выкл.` instead of `Активно / Нет активности`), corrected by `ee68d50f`. **CODE CI PASS only; UI/adapter M3 not installed**, working M2 and stats unchanged. The confirmed M3 UI hot-update changes only HTML/JS; existing Python collector remains a separate deployment gate. 


> **M3 мониторинг — текущий подтверждённый этап (2026-11-18):** [Go prototype checks #44306152672](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44306152672) завершился **SUCCESS** на коммите `669fa32e`, в том числе Python-тесты stats, Node.js-тесты панели, тесты UI-only установки/rollback, Go build/vet/unit/race и транспортные регрессии; отдельный M3 Linux candidate [#44306153058](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44306153058) также SUCCESS. **Это CI PASS кода, не live deployment.** Дополнительный пакет `34bb3e2` / `0038e08` исправляет неопределённый признак `padding_configured` в старом бинарнике и тестирует отличие «Нет активности» от «Выключено»; его **полный CI ещё не подтверждён**. В production по-прежнему M2, метрики подключены, работающие службы не менялись. Далее после CI: UI-only update с backup/rollback, локальная и независимая внешняя проверка, отдельно live испытания M3 rate1/jit1.


> **ТЕКУЩАЯ КОНТРОЛЬНАЯ ТОЧКА M3 (2026-11-18, приоритет над историей M1/M2 ниже):** Рабочая ветка разработки — `work/m3-adaptive-jitter`; оператор подтвердил установленный на Homeserver **M2, Padding включён**, метрики уже подключены, источник и локальный dashboard работали (`MONITORING_LOCAL=PASS`), публичный скриншот показывает реальные агрегаты. M2 не заменять. Требование M3 `rate1`: **≥1 Мбит/с в каждом отдельном направлении потока на протяжении двух последовательных полных 1-секундных окон**; выход ниже 0,7 Мбит/с, idle-reset >1 секунды, wire overhead ≤5%. Код/тесты `rate1`/ `jit1` в M3, production M2 использует прежний `pad1`. **Не выдавать CI за живое испытание M3.**
>
> **Пакет отображения метрик M3 (код в GitHub, NOT DEPLOYED):** Панель теперь различает отсутствие измерения на Encapsulator (backend dial, TLS) и ноль, отсутствие latency samples и реальный latency=0; показывает PAD состояние каждого направления и PAD кадры/переходы, а также подготовленные ранее PAD байты/overhead и Jitter. Изменения панели/JS и Node-регрессии: `0e4f97e`, `85c2564`, `83b44ce`. До GitHub CI PASS и отдельной установки на Homeserver: **CODE WRITTEN, TEST RESULT PENDING, LIVE UNCHANGED**. Не перезапускать Bridge, не удалять stats env, не активировать метрики повторно. Следующий M3 gate: CI → безопасное развёртывание только файлов панели с backup/rollback → внешний браузер + local source verification; отдельно контролируемые испытания `rate1/jit1` на двух узлах.


> **ИСТОРИЯ ДО M3 (не текущий статус):** M1 закрыт только в поддерживаемой области стандартного HTTP/2 транспорта по существующим CI/Homeserver/live PASS; ограничение нестандартного io.ReadCloser зафиксировано в `docs/M1_SCOPE_AND_CLOSURE.md` (commit `af015d30329a06d174a43959800af9b0d58c3ba4`). Это не новый test PASS. Текущий модуль — M2 Adaptive Padding. Изменения M2 до проверок не включать в production. Релиз v1.0.1 НЕ ВЫПУЩЕН.

> **ИСТОРИЧЕСКОЕ СОСТОЯНИЕ НА 2026-11-17 (не текущая версия M3):** на **обоих действующих узлах установлен и работает M1** (SHA-256 исполняемого бинарника `e42545d56ad05e94280466ddf92dfbcaa4955d716511f839de2eec0ae21675e5`); Homeserver использует **TBF1 `/v2/stream`**, оператор подтвердил работающий интернет, Finserver работает без предупреждений при финальной проверке. **Ни M0, ни опубликованный v1.0.0 не являются действующей версией моста.** Старые записи о M0/предварительном развёртывании ниже — хронология, не текущее состояние. **Совместимость с историческим v1.0.0 НЕ является обязательным gate M1** и не должна запрашиваться повторно. M1: **DEPLOYED + LIVE CONNECTIVITY PASS**; оценка 500ms fail-closed/непрерываемого `io.ReadCloser` остаётся отдельным инженерным вопросом без доказанного PASS. Служба Bridge Stats на Homeserver: **DEPLOYED + LOCAL PASS** (127.0.0.1:8894); публичный маршрут сайта `/bridge/`: **INSTALLED ON HOMESERVER + NGINX SYNTAX/RELOAD PASS, EXTERNAL DNS/TLS NOT VERIFIED**; телеметрия работающего M1: **NOT ENABLED**. Следующий шаг определять по актуальным открытым вопросам, а не по старым `NOT_RUN`.
**Репозиторий:** `git@github.com:tervon-software/Tervon-TCP-Bridge.git`.
**Важно:** пользователь разрешил публикацию исходников 2026-11-17. Go-прототип, CLI, тесты и workflow перенесены в публичную ветку `main` коммитом `1528ebd76309aa5330e4cde88a183e81f4cce074` из ветки `docs/working-prototype-release` (`f04280c77e54dc39ac29660a397925b0989fb625`). Публикация кода **не равна** подтверждённому PASS, готовности релиза или воспроизводимой сборке.

## Цель

Развивать существующий Tervon TCP Bridge, сначала Linux Encapsulator/Decapsulator, затем переносить **проверенные** наработки в TervonVPN Android. Первый этап — v1.0.1 с адаптивными Padding/Jitter для устойчивых длительных потоков, без постоянного изменения коротких и интерактивных соединений. Мост не анализирует содержимое VPN-протокола. Абсолютная «невидимость» для DPI не обещается; эффект проверяется измерениями.

## Утверждённая модель версий и релиза v1.0.1 (уточнение 2026-11-17)

- **v1.0.0** — исторический опубликованный GitHub Release; это исходная версия, а не новый релиз каждого промежуточного модуля.
- **v1.0.1** — **единый планируемый следующий релиз** после завершения всей очереди модулей M0, M1, M2, M3, M4 и M5. Пока все модули не завершены и финальные испытания не подтверждены, v1.0.1 **НЕ ВЫПУЩЕНА**; не создавать промежуточные релизы/теги под M0 или M1.
- **M0, M1, M2 и далее — внутренние этапы разработки v1.0.1**, не номера версий, не отдельные публичные релизы. Для каждого модуля: реализация → тесты → согласованное контролируемое испытание **на действующих Homeserver/Finserver** с резервной копией и откатом → фактический PASS → GitHub commit и документация → следующий модуль. Не создавать новые отдельные испытательные серверы вместо согласованных живых проверок.
- Успешный живой трафик M0→M1 и M1→M1 подтверждает соответствующие проверенные маршруты. Не превращать проверку исторического опубликованного v1.0.0 в отдельный релизный цикл или повод нарушать порядок модулей. Если отдельный критерий M1 остаётся открытым, фиксировать его точно и завершать в рамках M1.
- **Публикация:** только после сборки всех модулей, финальной интеграционной проверки, оформления release notes и контрольных сумм публикуется **один GitHub Release v1.0.1**. Публикация коммитов исходников в GitHub и установка промежуточного бинарника на живой сервер — не выпуск версии.

## Обязательное правило диагностики внешних сайтов и мониторинга (решение пользователя, 2026-11-17)

- **Проверять доступность `stats.xrayvless.online` и других внешних сайтов по их настоящему публичному DNS-имени и внешнему адресу, из внешней сети** (например, с зарубежного узла или пользовательского устройства вне Homeserver). Проверять реальный публичный маршрут, TLS SNI и сертификат на нём.
- **Не подменять публичный адрес на `127.0.0.1`, `--resolve ...:127.0.0.1`, `localhost`, внутренний IP или loopback при проверке доступности сайта.** Это другая точка входа, способная показать чужой виртуальный хост/сертификат и дать ложный вывод. Не предлагать такой тест снова.
- Результат запроса с самого Homeserver к публичному имени **не равен** проверке из внешней сети: на него могут влиять маршрутизация, NAT hairpin, DNS и локальные правила. Для вывода о доступности извне требуется именно внешний наблюдатель.
- **Разделять два разных вида проверок:** локальный loopback `/metrics/v1` Bridge допустим и обязателен для проверки **закрытого интерфейса сбора телеметрии**, но **не используется как замена внешней проверке веб-сайта**. Не публиковать приватный endpoint метрик.
- Любая диагностика сначала read-only; не менять nginx, DNS, firewall, TLS, Prometheus и работающий Bridge без отдельного плана и согласованного шага.

## Подтверждённое применение nginx Bridge Stats (2026-11-17)

- **Homeserver операторский вывод, git SHA `694f79076af9be5a55d2d0bb8120ee5ebad377cf`:** `NGINX_SYNTAX=PASS`, `NGINX_RELOAD=PASS`, `PUBLIC_BRIDGE_LOCATIONS=INSTALLED`, `BACKUP=AVAILABLE_FOR_ROLLBACK`, `EXTERNAL_DNS_TLS_CHECK=NOT_YET_VERIFIED`.
- **Статус:** новая публичная `/bridge/` конфигурация **INSTALLED + NGINX RELOAD PASS**, без Basic Auth; изолированная stats-служба на Homeserver **DEPLOYED + LOCAL PASS**; **внешний публичный DNS/TLS/HTTP из независимой сети NOT VERIFIED**, локальные метрики действующего Bridge M1 **NOT ENABLED**. Существующий `/admin/` должен оставаться под Basic Auth, требуется внешний регрессионный тест `401`; `/infrastructure/` и M1 не менялись установщиком.
- **Следующий gate:** внешний клиент с настоящим публичным DNS/SNI без `--resolve` и без `-k` проверяет `GET /bridge/` HTTP 200, `GET /bridge/api/status` HTTP 200 с единственным Homeserver `unavailable`, `GET /admin/` HTTP 401 без credentials, `GET /metrics/v1` не 200. Сайт не считать внешне доступным до фактического PASS; затем отдельно проектировать безопасное включение локального `metrics_listen` на работающем M1 без нарушения data plane.

## Read-only nginx audit и подготовка публичного маршрута (2026-11-17)

- **Фактический аудит оператора:** `/etc/nginx/sites-enabled/stats-temp` — обычный файл, не symlink; два `server` (строки 1 и 15), два `server_name stats.xrayvless.online` (строки 4 и 18), один HTTPS `listen 443` (строка 16), `/admin/` locations (26, 45), `/infrastructure/` (70, 74), includes (22, 91, 103). **Нет** Bridge locations и прокси на 8894. `nginx -t` **PASS**, предупреждение `variables_hash` не блокирует.
- **Готовится:** `integrations/stats/install-site-nginx.py` — точечная вставка одного `include` только в HTTPS server, отдельный snippet с публичными GET/HEAD `/bridge` и `/bridge/`, `auth_basic off` только в новых locations, резервная копия root-only, `nginx -t`, контролируемый reload, откат `rollback`. Проверяет git SSH branch/clean, stats service active и loopback API, отсутствие конфликтов в эффективном `nginx -T`, точный layout; любое отклонение блокирует. Не меняет M1, dashboard, /admin, DNS, TLS или firewall.
- **Статус:** код и изолированные тесты в GitHub; до полного CI PASS и фактического выполнения на Homeserver nginx **UNCHANGED**, публичный маршрут **NOT DEPLOYED**. После установки требуется внешний тест реального публичного DNS/TLS из внешней сети, не `--resolve` на localhost.

## Подтверждённая установка изолированного Bridge Stats (2026-11-17)

- **Homeserver installer PASS** на commit `33a44290921393467d462d285fd9ace300241892`: `GIT_SSH_REMOTE/GIT_BRANCH/GIT_HEAD/GIT_CLEAN/PORT_8894/NEW_UNIT/APP_DIRECTORY/PYTHON_DEPS/SYSTEMD_VERSION/ADMIN_HTPASSWD=PASS`, `DYNAMIC_USER_ACCESS=CHECK` (проверен фактическим успешным запуском и HTTP), `NO_CHANGES=PASS`. После systemd start: `LOOPBACK_STATUS=PASS`, `METRICS_STATUS=UNAVAILABLE_EXPECTED`, `LOOPBACK_LISTENER=PASS`, `NEW_SERVICE=PASS`, `NGINX=UNCHANGED`, `BRIDGE_DATA_PLANE=UNCHANGED`, `WEBSITE_INTEGRATION=NOT_YET_ENABLED`. Unit включён в автозапуск; доступ только 127.0.0.1:8894.
- **Фактические статусы:** отдельная stats-служба **DEPLOYED + LOCAL PASS**; внешний `/bridge/` **NOT DEPLOYED**; локальная телеметрия работающего M1 **NOT ENABLED**. Не считать `unavailable` реальными метриками и не объявлять мониторинг полностью подключённым.
- **Следующий шаг в текущем M1:** read-only аудит действующего nginx-конфига, проверка отсутствия конфликтующих locations, затем отдельное безопасное добавление только `/bridge` и `/bridge/` без авторизации с backup, `nginx -t`, контролируемым reload и rollback. Сохранить `/admin/` Basic Auth, `/infrastructure/`, M1 и dashboard. Реальный публичный доступ проверять только из внешней сети по настоящему DNS/TLS.

## Изменение решения оператора по доступу к Bridge Stats (2026-11-17, позднее)

- **Текущее решение:** публичные `/bridge`, `/bridge/` и `/bridge/api/status` на `stats.xrayvless.online` **без HTTP Basic Auth**. Ранее записанные требования Basic Auth для **Bridge** ниже являются историческими и заменены этим решением. **`/admin/`, `/admin/api/` и их авторизация не меняются.** Существующий `/infrastructure/` не менять.
- Публиковать только заранее ограниченные агрегаты Homeserver, без IP-адресов, клиентских идентификаторов, токенов, сертификатов, секретов, backend URL, внутренних адресов и сырого `/metrics/v1`. Источник `/metrics/v1` остаётся **только на loopback**, Finserver не подключать. Принять во внимание, что публичные агрегаты раскрывают активность и состояние сервиса.
- Этапность не меняется: сначала подтверждённый PASS отдельной stats-службы на 127.0.0.1:8894; только затем отдельный контролируемый nginx-этап с `nginx -t`, проверкой реального внешнего DNS и откатом. **Не отключать/не обходить авторизацию `/admin/`.**
- **Уточнение по отказу запуска:** journal systemd показал `Started → Stopping → Stopped` в пределах одной секунды, без ошибки приложения; unit и app удалены, 8894 свободен. Возможная причина — однократный HTTP probe до готовности Uvicorn. Исправление в текущей ветке: ограниченный `wait_for_site.py` (до 12 с, только localhost, retry исключительно ConnectionRefused), тесты и CI; пока **NOT DEPLOYED**. Внешний nginx **не менять**, пока нет фактического `NEW_SERVICE=PASS`.
- **Последний фактический Homeserver installer:** checkout SHA `8da8eb70dc30084705e8eb833f7aeff2093675d3`, все обязательные preflight PASS, `DYNAMIC_USER_ACCESS=CHECK`; `systemctl start` + `is-active` прошли, но loopback `/bridge/api/status` дал `ConnectionRefusedError`; `INSTALL_STATUS=ROLLBACK (exit=1)`. Новая stats-служба **NOT DEPLOYED**. До нового запуска — read-only journal диагностика. Работающие Bridge/nginx/dashboard не менять.

## Утверждённая область статистики сайта (2026-11-17)

- `stats.xrayvless.online` получает **только локальные метрики Encapsulator на Homeserver**. Finserver в мониторинге сайта не участвует: не создавать удалённый сборщик, туннель, relay, внешний metrics listener или показатели Finserver.
- Подготовленный адаптер `integrations/stats/bridge_metrics.py` читает один приватный loopback `/metrics/v1`; подготовленные `bridge_panel.html`, `bridge_panel.js`, `bridge_panel.css` — **только исходники, не установка**.
- Перед добавлением `/bridge/api/status` проверить фактическую авторизацию сайта. Существующий `/infrastructure/api/infrastructure` и его семипольный контракт не изменять.
- Проверка nginx на Homeserver подтвердила `/admin/` и `/admin/api/` с HTTP Basic Auth и upstream 8000, а `/infrastructure/` с upstream 8893 без видимой авторизации. Для Bridge выбран **отдельный** `/bridge/` с тем же локальным nginx `auth_basic_user_file` и **отдельным** loopback-only FastAPI приложением `integrations/stats/bridge_site.py`. Не встраивать в действующий `unified_monitor.py` и не менять инфраструктурный API. Порт 8894 — кандидат, не подтверждён как свободный. Шаблон `integrations/stats/nginx-bridge.conf.example` не применять без аудита и согласования; обязательны `nginx -t`, внешний 401 без учётных данных и 200 с учётными данными, откат.
- **Read-only Homeserver preflight 2026-11-17:** TCP port 8894 not listening; existing venv FastAPI 0.139.0, Uvicorn 0.50.2. Prepared `integrations/stats/tervon-bridge-stats.service.example` with loopback-only bind and systemd sandbox. Exact `/bridge` and `/bridge/` both require nginx Basic Auth in template; exact `/bridge` uses proxy_pass, not rewrite-phase `return`, to avoid an auth bypass. This is a staging design, **not installed**. Before installation verify DynamicUser read access, systemd network sandbox support, nginx htpasswd reuse, then explicit approval for any service/nginx changes and external auth checks.
- **Read-only deployment gate:** `integrations/stats/preflight_homeserver.py` checks the Homeserver SSH checkout, clean M1 branch, port 8894, absence of existing service/app, local venv, systemd version and existing admin htpasswd without exposing paths/secrets. Any `BLOCKED` prevents installation; `DYNAMIC_USER_ACCESS: CHECK` remains a separate gate. Script does not touch running services; `git --no-optional-locks status` avoids index writes. Before any installation obtain the operator's actual Homeserver output and confirm no blockers.
- **Разрешение оператора 2026-11-17:** подключить мониторинг Homeserver. Подготовлен `integrations/stats/install-site-service.sh`: проверяет preflight, устанавливает только новый sandboxed `tervon-bridge-stats.service` на 127.0.0.1:8894, проверяет локальный API (`unavailable` до активации метрик), автоматически откатывает только новую службу при FAIL. Скрипт **не изменяет nginx и не перезапускает Bridge**. Требуются подтверждённый CI PASS на SHA, исполнение на Homeserver и проверка фактического результата. После PASS отдельный контролируемый шаг nginx с Basic Auth и rollback, затем отдельное подключение локального `metrics_listen`. Не считать код установленным до вывода `NEW_SERVICE=PASS`.
- Полный PASS CI адаптера не равен внедрению на сайте. Действующие службы, nginx и Bridge не изменять без согласованного плана и отката. M1 остаётся текущим модулем.

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
- **Исторический статус M0 на момент переноса:** `IN_PROGRESS`. **Не текущий статус:** M0 впоследствии прошёл эксплуатационные проверки и был заменён M1 на обоих действующих узлах. Исторический опубликованный v1.0.0 не используется.
- **Исторический план на момент переноса (выполненные впоследствии шаги отражены ниже):** проверить дерево, функции, конфигурацию, протокол, выполнить build/test/race/vet и зафиксировать SHA. Не трактовать как текущую задачу.

## Очередь модулей: строго по порядку

### M0 — Инвентаризация, мониторинговый интерфейс, базовая линия и эксплуатационная проверка [ЗАВЕРШЁН: эксплуатационный PASS, ограничения ниже]
**Сделать одним пакетом:** инвентаризировать текущий Go-код, TLS/mTLS/HTTP2, JSON, протокол и тесты; сверить исходники с v1.0.0; реализовать безопасный экспорт базовых метрик и тесты; собрать воспроизводимый бинарник; провести изолированные тесты, затем согласованное развёртывание и сквозную проверку реального TCP-трафика. Если связи нет — диагностировать и исправлять в M0 до подтверждённого результата.
**Мониторинг:** Bridge предоставляет только локальный read-only интерфейс сбора статистики (по умолчанию 127.0.0.1, не публичный endpoint). Отдельный контролируемый коллектор передаёт агрегированные данные на `stats.xrayvless.online`; интеграция с сайтом требует аудита его текущей архитектуры и отдельного безопасного шага без нарушения действующих служб. Не открывать метрики наружу, не экспортировать сертификаты, ключи, идентификаторы клиентов, IP-адреса или содержимое потоков. Для сайта предусмотреть строгую аутентификацию и разграничение доступа.
**Базовые метрики M0:** build version/commit, uptime, readiness, активные/завершённые/отклонённые TCP-сессии, полезные байты по направлениям, текущая скорость на окнах измерения, ошибки по ограниченному набору причин (TLS handshake, auth, backend dial, HTTP status, timeout, cancellation, shutdown), число активных HTTP/2-сессий при доступности точного счётчика, лимиты и занятость, задержки установки соединений (histogram p50/p95/p99), backend availability по безопасному идентификатору, runtime memory/goroutines. Трафик измерять отдельно как полезные байты и накладные байты только там, где они достоверно наблюдаемы. Не называть полезные байты полным сетевым трафиком. TLS version/cipher/ALPN/mTLS status — только проверенные, агрегированные наблюдения; не выдавать конфигурацию за факт успешного соединения.
**Будущие поля (M2/M3):** Padding и Jitter в состояниях NOT_IMPLEMENTED/OFF/ACTIVE/ERROR, отдельно upload/download; padding bytes/overhead%, jitter delay and disable reason, budgets, hysteresis transitions. Пока функциональность не реализована, не показывать ACTIVE. RTT/потери/ретрансляции отображать только при наличии корректного источника измерений и указывать слой измерения.
**Надёжность телеметрии:** никакой блокировки data plane из-за мониторинга, ограниченные метки и память, атомарные счётчики, отсутствие бесконечной кардинальности, ограничение частоты сбора, отдельные тесты race/нагрузки и отказа коллектора; мониторинг не должен менять байты TCP-потока.
**Готово, когда:** исходники и commit подтверждены; есть карта модулей, базовые тесты и перечень рисков; мониторинговый контракт реализован и протестирован; бинарник собран; изолированные проверки PASS; после отдельной безопасной установки подтверждена реальная двусторонняя TCP-связь и наблюдаемость. Если развёртывание пока невозможно, M0 остаётся IN_PROGRESS/BLOCKED, а не PASS.

### M1 — Версионирование и безопасное кадрирование [ЗАВЕРШЁН В РАМКАХ HTTP/2; СМ. docs/M1_SCOPE_AND_CLOSURE.md]

#### M1 — контрольная точка 2026-11-17 (GitHub source of truth)

- **Ветка:** `work/m1-framing`. **Текущий кодовый commit:** `b7e620646921503d728eb8a21e281ef31f0327f0` (добавлен E2E early backend EOF с открытым HTTP/2 upload). **CI workflow commit:** `6c9aef45f9700c629c779bdd300b65f0e53e8b7c`.
- **Подтверждённый GitHub Actions PASS:** [run #44036862629](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44036862629), HEAD `6c9aef45f9700c629c779bdd300b65f0e53e8b7c`, Ubuntu runner: `go build ./...`, `go vet ./...`, `go test -count=1 -timeout=60s ./...`, `go test -race -count=1 -timeout=120s ./...`, `go test -race -count=3 -timeout=180s -run 'TestFramed|TestLegacyV1StreamRemainsByteExact|TestStream' ./...` — PASS.
- **Fuzz:** на том же CI run `FuzzReadFrame` 15s / 722070 executions — PASS; `FuzzStreamDecoder` 15s / 718491 executions — PASS. Это ограниченный CI fuzz, не длительная нагрузочная проверка.
- **Проверенные сценарии:** TBF1 framing/FIN, negative parser tests, version/capability negotiation, binary byte-exact framed/legacy v1, TCP half-close, truncated download, malformed upload, cancellation, early backend EOF до request FIN, завершение server handler при открытом HTTP/2 request body. Проверки выполнялись в CI на тестовых loopback backend и HTTP/2 mTLS.
- **Изменение lifecycle:** `prototype/bridge.go` теперь ожидает server upload worker после закрытия `r.Body` и backend. Проверенный тестами стандартный HTTP/2 путь завершился. Произвольный кастомный `io.ReadCloser`, который игнорирует `Close` и блокирует `Read`, по-прежнему может зависнуть; универсальная гарантия для таких источников не заявляется.
- **Повторный CI PASS с Homeserver-скриптом:** [run #44037187498](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44037187498), commit `9ad9265b175f114178007321049811d5d595c11c` — сборка, vet, unit, race, 3x targeted race, оба fuzz 15s и `bash -n ../tests/homeserver/verify-m1.sh` прошли. **Последний подтверждённый CI SHA:** `9ad9265b175f114178007321049811d5d595c11c`.
- **Подготовлена воспроизводимая проверка Homeserver:** `tests/homeserver/verify-m1.sh` (commits `4cc39b7d0dd2c3f461b41bc960bb13f3505faffd`, `da40bb9da134631551581015bfeb2bfbb56a8d34`). Только изолированный чистый checkout, проверка SSH remote и точного remote HEAD, build/vet/race/targeted и fuzz по 30s, без развёртывания. Сам скрипт на Homeserver ещё не запускался.
- **Homeserver PASS — подтверждён пользователем 2026-11-17:** `e2313ab8375be3d9a7934e8b1c462a17e1fa3286`, Go `1.25.5 linux/amd64`, isolated checkout `/opt/tervon/bridge-m0-check/build/repo`, `M1_VERIFY_RESULT=PASS`. `go build ./...`, `go vet ./...`, `go test -race -count=1 -timeout=120s ./...` (4 пакета), `go test -race -count=3 -timeout=180s -run 'TestFramed|TestLegacyV1StreamRemainsByteExact|TestStream' ./...` — PASS; `FuzzReadFrame` 30s / **1 347 631 executions**, `FuzzStreamDecoder` 31s / **1 211 473 executions**, оба PASS. Скрипт вывел `M1_VERIFY_PRODUCTION_CHANGES=NONE` и итоговый `M1_VERIFY_NOTE=code_level_only_no_production_deployment`. Это доказанный кодовый PASS, а не выпуск.
- **Новый кодовый пакет после Homeserver PASS:** `e26792c61d91b605d43bf6ff76f370f667c7ddab` — `prototype/legacy_wire_compat_test.go`: отдельные сценарии legacy-style HTTP/2 raw клиент → текущий сервер и текущий клиент → legacy-style raw сервер. Проверяют отсутствие TBF1 negotiation headers и двустороннюю байтовую идентичность на `/v1/stream`. **Ограничение:** legacy сторона смоделирована по wire-контракту, а не запущена из исторического бинарника v1.0.0.
- **Обновлённые race-гейты:** `dd09b1cf2b1311d896ffb1ffd08daa38ea43374c` и `ce551cfeed2c4ecdcf7dbc760a7e45fa7e16281b` включили `TestLegacy*` в 3-кратный race-прогон GitHub CI и Homeserver-скрипта. [GitHub Actions #44039980495](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44039980495) — **PASS** на кодовом HEAD `ce551cfeed2c4ecdcf7dbc760a7e45fa7e16281b`: build, vet, unit, race, 3x targeted, оба fuzz 15s, bash syntax. Новые тесты **ещё не запускались на Homeserver**; предыдущий Homeserver PASS относится только к `e2313ab...`.
- **Повторный Homeserver PASS, пользовательский протокол:** commit `8aedaf9e518e9cfbb60529d4ff32c0acb90dabad`, Go `1.25.5 linux/amd64`, isolated checkout, `M1_VERIFY_RESULT=PASS`. `go build ./...`, `go vet ./...`, полный `go test -race -count=1 -timeout=120s ./...` (4 пакета), 3-кратный targeted `TestFramed|TestLegacy|TestStream` с race — PASS. `FuzzReadFrame` 30s / **1 326 453** executions, PASS, 11 corpus; `FuzzStreamDecoder` 30s / **1 315 708** executions, PASS, corpus вырос с 15 до 19. `M1_VERIFY_PRODUCTION_CHANGES=NONE`. Это **Homeserver CODE PASS**, не испытание реального v1.0.0 binary и не production release.
- **Новый реальный release-compat gate (ещё НЕ ПРОВЕРЕН):** официальный [GitHub Release v1.0.0](https://github.com/tervon-software/Tervon-TCP-Bridge/releases/tag/v1.0.0) содержит `tervon-bridge-linux-amd64`, SHA-256 `5c813bae9cb0c68e3d5ac1e9afb0556f13cb270b369a86ae8ffc0d20adb086e2`. Добавлен opt-in `prototype/cmd/bridge/legacy_release_compat_test.go` (commit `3f951e173051834786b37ff920081e2409747e16`), который проверяет этот SHA, запускает **копию** реального v1.0.0 binary и текущий CLI в отдельных процессах с эфемерным тестовым CA, localhost TCP echo, проверяет оба направления old-client/new-server и new-client/old-server. В `tests/homeserver/verify-m1.sh` (commit `cd9c1632ede701b5485b42d39f532317944e0bc1`) при `TERVON_V100_BIN=/local/path` запускается тест и выводится `M1_REAL_V100=PASS` либо FAIL; без переменной `M1_REAL_V100=NOT_RUN`. **Тест ещё не исполнялся с бинарником**, не объявлять совместимость доказанной.
- **Исправленный CI PASS:** [GitHub Actions #44041281072](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44041281072), SHA `797f2d93726ded3ef27632ee2e4d590a005ec4fa`, **SUCCESS**: bash syntax, build, vet, unit, race, 3x regression и оба ограниченных fuzz. Это подтверждает компиляцию opt-in теста и исправление shell; **не подтверждает запуск с v1.0.0 binary**, потому что `TERVON_V100_BIN` не задан.
- **Исторический gate (пересмотрен после эксплуатационного PASS):** проверка опубликованного v1.0.0 **снята с обязательных критериев M1**: он не используется на действующих узлах, которые уже работают M1↔M1 через TBF1 `/v2/stream`. Опциональный тест с v1.0.0 остаётся только историческим инструментом, не блокирует разработку. **Открытый инженерный вопрос:** оценить 500ms fail-closed при раннем backend EOF и явно документировать пределы завершения для `io.ReadCloser`, игнорирующего `Close`; не заявлять доказанного PASS этого свойства. Действующие службы не трогать без отдельного согласования.
- **Историческая контрольная точка до установки:** Production: NONE. **Актуально:** M1 установлен на обоих узлах и работает; TBF1 `/v2/stream` live connectivity PASS. `gofmt -l` в раннем CI был non-blocking; не считать его отдельным PASS.

#### M1 — корректирующий аудит 2026-11-17 (после восстановления контекста M0)

- **Историческая контрольная база до установки M1:** M0 тогда работал на Homeserver/Finserver с SHA-256 `fbc927f33cd0198329bb21ff0b40a5d4e8240bc9a92d3d62c1d9c4e4ec30fc0c`, отличным от опубликованного v1.0.0 (`5c813bae9cb0c68e3d5ac1e9afb0556f13cb270b369a86ae8ffc0d20adb086e2`). **Сейчас оба рабочих узла на M1**; старое предписание «M0 не трогать» не актуально.
- **Обнаруженная проблема M1:** клиент при ошибке декодирования /v2/stream делал `CloseWrite()` на локальном TCP, что могло представить повреждённую загрузку как обычный EOF. Исправление `22a2bfeb11953350891780b1a10993d3c77b4c28`: при ошибке framed-download `SetLinger(0)` перед закрытием TCP; тест на реальном TCP `prototype/framed_tcp_abort_test.go` (commit `dcf3b5b27ff795ee076003cea2607b6c4fcc080b`).
- **Исправление несовпадения эталонов:** `8c84e6e56628955af3d036370d7ea51c9e703ae6` — раздельные opt-in бинарные E2E проверки `TERVON_M0_BIN` и `TERVON_V100_BIN` с обязательной проверкой **соответствующего** SHA-256 и копированием в изолированный временный каталог; оба направления на /v1/stream.
- **Homeserver gate:** `tests/homeserver/verify-m1.sh` добавил `M1_REAL_M0=PASS/NOT_RUN`; первоначальная вставка в `5dc48cf87097f556055d6c6d3d62234b42715cb4` ошибочно повредила shell, GitHub Actions #44044892316 **FAIL** на bash syntax. Исправление `307c59564f3a49317227488e32c2e8f7b57a7719`. **Подтверждённый CI PASS:** [run #44044975373](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44044975373), SHA `307c59564f3a49317227488e32c2e8f7b57a7719`: bash syntax, build, vet, unit, race, 3x targeted и оба bounded fuzz — SUCCESS. Opt-in реальные M0/v1.0.0 E2E без переменных **NOT_RUN**.
- **Homeserver PASS подтверждён пользователем на точном SHA `5ae795f0a6f342724f6dc894f0e82b7f4402f867`:** `go build ./...`, `go vet ./...`, `go test -race -count=1`, повторные race-регрессии (3 прогона), `FuzzReadFrame` 30s / 1 336 459 executions, `FuzzStreamDecoder` 30s / 1 258 600 executions, `M1_REAL_M0=PASS` (65536 байт byte-exact в обоих направлениях TLS1.3/mTLS/HTTP2 `/v1/stream`), `M1_VERIFY_RESULT=PASS`, `M1_VERIFY_PRODUCTION_CHANGES=NONE`. Это **изолированная кодовая проверка**, не развёртывание M1. `M1_REAL_V100=NOT_RUN`.
- **Актуализация после установки M1:** проверка с опубликованным v1.0.0 **не требуется и не блокирует M1**; реальные рабочие узлы M1↔M1, TBF1 `/v2/stream` подтверждён. Отдельно остаётся оценка 500ms/непрерываемого `io.ReadCloser`. Не путать это незакрытое свойство с уже подтверждённой эксплуатационной работой M1.

#### M1 — подготовка развёртывания с backup M0 (2026-11-17)

- **Решение пользователя:** установить M1 на рабочие узлы, сохранив возможность отката к M0.
- **Реализована только подготовка:** `tests/homeserver/deploy-m1-with-m0-backup.sh` (preflight/inspect, checksum-pinned install, root-only backup бинарника/unit/конфигурации, atomic replace, restart только выбранного bridge unit, проверка SHA запущенного процесса, автоматический и ручной rollback). Инструкция `docs/DEPLOY-M1-WITH-M0-BACKUP.ru.md`.
- **Порядок:** Finserver Decapsulator → подтверждение работы /v1/stream → Homeserver Encapsulator → подтверждение реального TCP. TBF1 /v2/stream включать только отдельной контролируемой сменой `remote_url` после PASS обоих узлов.
- **Фактическое состояние:** `PREPARED_NOT_DEPLOYED`. Пользователь подтвердил: Homeserver активный M0 `/opt/tervon-tcp-bridge/bridge` SHA `fbc927f3...30fc0c`; Finserver активный M0 **`/opt/tervon-bridge-linux-amd64`** с тем же SHA, тогда как `/opt/tervon-tcp-bridge/bridge` на Finserver — **неиспользуемый v1.0.0** SHA `5c813bae...b086e2`. Скрипт исправлен на чтение реального ExecStart и сохранение пути в backup manifest. Пользователь запретил дополнительные тестовые окружения; только живое поэтапное развёртывание с обязательным backup/rollback. Скрипт ещё **не исполнялся на реальных узлах**. Рестарт bridge unit прервёт активные соединения; никаких иных служб не трогать. Не путать подготовленный сценарий и эксплуатационный PASS.

- **Первая реальная установка Finserver 2026-11-17:** `M1_INSTALL=FAIL`, автоматический `M0_AUTO_ROLLBACK=PASS`, backup `/opt/tervon-tcp-bridge/backups/m0-qVNyeBJK`. M0 повторно подтверждён на диске и в процессе PID 37153. В журнале старый M0 завершился с `graceful shutdown incomplete` и exit 1; новый процесс был отмечен systemd как Started, но установщик немедленно откатил. Причина не доказана окончательно; вероятен ложный FAIL установщика. Исправление `f97ecd8086fe561072cd9def85a2691a7d527db8` — bounded PID+SHA verification независимо от exit code restart; ждём CI и повторного контролируемого развёртывания. **Статус M1: NOT_DEPLOYED; Homeserver M0 не изменён.**

- **Повторная установка Finserver 2026-11-17:** пользователь подтвердил `M1_INSTALL=PASS`, `CONFIG_CHANGED=NO`, новый бинарник SHA `e42545d56ad05e94280466ddf92dfbcaa4955d716511f839de2eec0ae21675e5`, резервная копия M0 `/opt/tervon-tcp-bridge/backups/m0-mDC0QnYM`. **Только установка серверного бинарника PASS**; проверка реального `/v1/stream` и TCP-трафика ещё требуется. Homeserver по-прежнему M0, `/v2/stream` не активирован. Исторические записи о предыдущем FAIL сохраняются.

- **Сквозная проверка действующего маршрута 2026-11-17:** Homeserver client M0 PID 2304526 active; Finserver server M1 PID 37309 active, SHA `e42545d56ad05e94280466ddf92dfbcaa4955d716511f839de2eec0ae21675e5`; пользователь подтвердил работоспособность интернета. **M0→M1 `/v1/stream` эксплуатационный PASS** (без количественных измерений). Следующий шаг в M1 — контролируемая установка M1 на Homeserver с backup M0, затем E2E, и только потом отдельная активация `/v2/stream`. Запрос интеграции Bridge-мониторинга в `stats.xrayvless.online` зарегистрирован, но изменение сайта отложено до закрытия M1 согласно правилу одного модуля.

- **Установка Homeserver M1 2026-11-17:** пользователь подтвердил preflight M0 PID 2304526, бинарник `/opt/tervon-tcp-bridge/bridge` SHA `fbc927f33cd0198329bb21ff0b40a5d4e8240bc9a92d3d62c1d9c4e4ec30fc0c`, кандидат M1 SHA `e42545d56ad05e94280466ddf92dfbcaa4955d716511f839de2eec0ae21675e5`, backup `/opt/tervon-tcp-bridge/backups/m0-5RY4qRMn`, `M1_INSTALL=PASS`, `CONFIG_CHANGED=NO`. **Оба бинарника M1 установлены**, но реальный E2E **M1→M1 после рестарта Homeserver ещё не подтверждён**; `/v1/stream` сохранён, `/v2/stream` ещё не активирован. Предыдущие записи `NOT_DEPLOYED` выше относятся к историческим контрольным точкам, не к текущему состоянию.

- **M1→M1 `/v1/stream` эксплуатационный PASS 2026-11-17:** Homeserver PID 2946701, SHA бинарника на диске и процесса `e42545d56ad05e94280466ddf92dfbcaa4955d716511f839de2eec0ae21675e5`; Finserver PID 37309 active/running с тем же SHA. Пользователь подтвердил интернет через мост, журнал предупреждений клиента за последние 5 минут пуст. **M1 бинарники установлены на обоих узлах и `/v1` работает.** TBF1 `/v2/stream` **ещё не активирован и не проверен**; M1 остаётся IN_PROGRESS до безопасного включения `/v2`, E2E PASS и фиксации результата.

- **Подготовка безопасного включения TBF1:** пользователь подтвердил фактический путь клиентского конфига `/etc/tervon-tcp-bridge/client.json`, `mode=client`, `remote_url` путь `/v1/stream`. В ветке M1 добавлены `tests/homeserver/activate-m1-framing.py` (только клиент, backup конфигурации с закрытым доступом, атомарная замена `remote_url`, рестарт только client unit, проверка PID+SHA, автооткат при ошибке запуска, явный ручной rollback при отсутствии трафика), unit-тесты и CI workflow. **Тесты нового скрипта и реальная активация `/v2` ещё не подтверждены.** Проверка PID+SHA не доказывает сетевой E2E; пользователь должен отдельно подтвердить интернет на `/v2`.

- **TBF1 `/v2/stream` эксплуатационная проверка 2026-11-17:** на Homeserver выполнены 4/4 unit-теста переключателя (PASS), `activate-m1-framing.py enable` сообщил `V2_ACTIVATION=PASS_PROCESS_ONLY`, PID 2988996, backup `/opt/tervon-tcp-bridge/backups/m1-framing-c6_jvgv6`; пользователь отдельно подтвердил, что интернет после переключения работает. **TBF1 `/v2` live connectivity PASS (качественная проверка)**. В `prototype/bridge.go` сервер M1 одновременно принимает `/v1/stream` и `/v2/stream` с согласованием TBF1, поэтому Finserver не требует отдельного переключения и перезапуска. Не заявлять количественную производительность или длительную стабильность без измерений.

- **Финальная эксплуатационная проверка Finserver после `/v2` 2026-11-17:** `tervon-bridge-server.service` PID 37309, `active/running`, `journalctl -p warning --since '5 minutes ago'` — пусто. Совместно с пользовательским подтверждением интернета на Homeserver TBF1 `/v2` — **эксплуатационный PASS по доступности**, без длительного benchmark. **Актуализация критериев:** проверка исторического бинарника v1.0.0 **не требуется**, поскольку оба рабочих узла используют M1; `M1_REAL_V100=NOT_RUN` — историческая запись, **не блокер**. M1 **DEPLOYED, TBF1 `/v2` LIVE PASS**. Оценка 500ms fail-closed/непрерываемого `io.ReadCloser` остаётся отдельно без подтверждённого PASS; не выдавать её за отказ работающего моста.

**Пакет:** формат кадров и парсер с ограничениями длины/памяти; negotiation; совместимость OFF с v1.0.0 либо явный безопасный отказ; fuzz/negative tests; байтовая идентичность full duplex, half-close, EOF, cancellation.
**Готово, когда:** все тесты и совместимость подтверждены на конкретном commit.

#### M1 — журнал фактической реализации (2026-11-17)

- **Текущая ветка:** `work/m1-framing`; базовый коммит M1: `f508f1b025eb49eb07c48b8befb10d10d0a4964d`.
- **Реализовано:** `prototype/framing/framing.go` — формат TBF1 (версия 1), типы DATA/FIN, 10-байтовый заголовок, ограничение payload 64 KiB, проверки magic/version/type/length и коротких чтений/записей. Непрозрачный payload не анализируется.
- **Тесты:** `prototype/framing/framing_test.go` — round-trip, отрицательные случаи, truncation и fuzz harness. На Homeserver пользователь выполнил `go test -race -count=1 ./...` (PASS для четырёх пакетов) и `go vet ./...` (PASS без сообщений) после установки кодека. Непрерывный fuzz-прогон не подтверждён.
- **Подтверждение GitHub:** коммит `f508f1b025eb49eb07c48b8befb10d10d0a4964d` опубликован в `work/m1-framing`; файлы кодека прочитаны из GitHub.
- **Исторический список на раннем этапе M1 (не актуальное состояние):** тогда не были проверены negotiation/capability, интеграция, framed full-duplex/half-close/EOF/cancellation и длительные fuzz-тесты. **Позднее negotiation, интеграция, framed E2E и ограниченные fuzz получили CI PASS, а M1 `/v2` — эксплуатационный PASS** (см. записи выше). Совместимость с v1.0.0 не требуется.
- **Совместимость:** действующий `/v1/stream` не изменён; framed режим пока не подключён и не развёрнут. Работающие службы не затронуты.
- **Статус M1 (актуальный, 2026-11-17):** **DEPLOYED на Homeserver и Finserver, TBF1 `/v2/stream` LIVE CONNECTIVITY PASS**; это не выпуск v1.0.1. Ранее перечисленные version/negotiation, integration и negative/E2E tests реализованы и проверены. Единственный отдельно отмеченный инженерный вопрос перед формальным закрытием — 500ms fail-closed/непрерываемый `io.ReadCloser`. **Не требовать v1.0.0 compatibility.**
- **Приоритет записей:** исторические утверждения M0 ниже в документе отражают состояние на момент записи; текущие статусы задаются заголовками очереди модулей и этим журналом. Не трактовать старое «M1 не начинать» как действующее ограничение после завершения M0.

### M2 — Адаптивный Padding [ИСТОРИЧЕСКАЯ ЗАПИСЬ; текущая разработка M3]

#### External HTTPS redirect confirmed by operator (2026-11-18)

- Operator confirms that an ordinary external visit to `https://vp.xrayvless.online/` now redirects to `https://ya.ru/`. The implementation uses existing Homeserver Nginx with the domain's existing TLS certificate; no Caddy/Docker/new listener was requested. This is an **operator-reported redirect functionality PASS**, not independent network tracing by ChatGPT.
- Separate subscription endpoint on port 2096 and Bridge have not been reported as changed, but independent post-change regression validation of those endpoints has not been supplied; do not claim those checks PASS without evidence.
- **Open M2 issue:** monitoring previously reported not working, root cause unknown. Complete M2 monitoring diagnostics/fix/verification before M3; the new redirect does not close this issue.



#### Operator request — existing nginx HTTPS redirect and M3 progression (2026-11-18; NOT EXECUTED)

- Operator requests that an ordinary browser visit to `https://vp.xrayvless.online/` receive an HTTP 302 redirect to `https://ya.ru/`, while existing subscription route on port 2096 and existing Tervon Bridge traffic remain unchanged. Port 443 is already occupied; do not introduce another listener or modify unrelated virtual hosts. Before any change, inspect exact Nginx server block/listeners on Homeserver, back up only the affected configuration, validate `nginx -t`, reload existing Nginx and verify redirect and subscriptions separately. **No actual Nginx host inspection or change has been performed by the assistant.**
- Operator additionally requests continuation to `M3 Adaptive Jitter`. Per confirmed sequential-development gate, `M2` is still open due to reported broken monitoring and missing telemetry confirmation; the operator's request is recorded but M3 work does **not** start until the M2 closure criteria and outstanding monitoring regression are resolved. No release claim.



#### M2 — operator confirms connectivity PASS; monitoring regression reported (2026-11-18)

- After both existing Homeserver and Finserver M2 binaries were installed and `padding_enabled=true` activated, operator explicitly confirmed **`Всё работает`** (normal existing service connectivity). Record **M2 live connectivity PASS (operator qualitative report)**, NOT a quantified throughput/latency/jitter/long-running acceptance PASS. Do not conflate process health and actual PAD frame counter activity; the latter has not been observed yet.
- Operator simultaneously reported **`Мониторинг только не работает`**. Exact failing monitoring surface, telemetry chain and error details have not been supplied; register **MONITORING=REPORTED_BROKEN / ROOT_CAUSE_UNKNOWN**, not fixed and not validated. Relevant existing Bridge Stats and service instrumentation must be inspected read-only before proposing changes. Preserve production Bridge, nginx and other services; do not change unrelated monitoring components without the user's approval.
- Deployment/activation private backups: Homeserver binary `/opt/tervon-tcp-bridge/backups/m0-aV1CqZ3C`, client JSON `/opt/tervon-tcp-bridge/backups/m2pad-client-XpjLETZN`; Finserver binary `/opt/tervon-tcp-bridge/backups/m0-cSx4CMGr`, server JSON `/opt/tervon-tcp-bridge/backups/m2pad-server-bHxygPhh`.
- **Current status:** `M2 DEPLOYED / PAD ENABLED / NORMAL CONNECTIVITY PASS (operator) / MONITORING BROKEN (operator) / M2 OPEN / NOT RELEASED`. Continue resolving the monitoring issue within M2; M3 remains blocked.



#### M2 — both production endpoints PAD configuration enabled (2026-11-18; connectivity unverified)

- **Homeserver:** operator reports direct `tervon-bridge-client.service` activation `M2_PADDING=ENABLED`, `ACTIVATION_COMMAND_EXIT_CODE=0`, `mode=client`, existing TBF1 HTTPS `/v2/stream`, `padding_enabled` previously false, restarted PID `2276833`, running SHA-256 `64c3a7161d7d03b74ec70ea6ae2ceb97992e86d2dfa65e8662193ffc0bc7d42c`, configuration backup `/opt/tervon-tcp-bridge/backups/m2pad-client-XpjLETZN`. Prior binary M1 backup `/opt/tervon-tcp-bridge/backups/m0-aV1CqZ3C` retained.
- **Finserver:** prior operator report `M2_PADDING=ENABLED`, `ACTIVATION_COMMAND_EXIT_CODE=0`, new PID `46322`, same running SHA-256, configuration backup `/opt/tervon-tcp-bridge/backups/m2pad-server-bHxygPhh`. Prior M1 binary backup `/opt/tervon-tcp-bridge/backups/m0-cSx4CMGr` retained.
- **Confirmed:** M2 binary installed and both configs opt in to PAD at startup. These observations come from operator-provided commands and outputs; no direct host connection from ChatGPT. **Not yet confirmed:** live successful client→server `tbf1-fin,pad1` negotiation, normal Internet traffic through the bridge after client activation, production performance. A clean process restart proves startup, not throughput or application-layer connectivity.
- **Status:** `M2 DEPLOYED / PADDING CONFIG ENABLED / CONNECTIVITY PENDING / LIVE PERFORMANCE PENDING / NOT_RELEASED`. Do not mark complete or start M3 until an operator-reported normal connection PASS is recorded and acceptance conditions are explicitly resolved. No lab/test service was started.



#### M2 — Finserver PAD enabled by operator (2026-11-18; client pending)

- Operator executed direct server JSON activation for existing `tervon-bridge-server.service` on Finserver. Reported `CONFIG_MODE=server`, prior `padding_enabled=False`, `M2_PADDING=ENABLED`, new PID `46322`, running SHA-256 `64c3a7161d7d03b74ec70ea6ae2ceb97992e86d2dfa65e8662193ffc0bc7d42c` and `ACTIVATION_COMMAND_EXIT_CODE=0`. Private configuration backup `/opt/tervon-tcp-bridge/backups/m2pad-server-bHxygPhh` retained.
- **Deployment distinction:** Finserver server-side PAD capability is enabled and process hash verified; Homeserver client-side `padding_enabled` **NOT YET ENABLED**. No observed pad1 request/response or normal-client connectivity PASS was reported, so E2E PAD **LIVE PENDING**. Next action: enable Homeserver client only after verifying its `/v2/stream` URL, back up its private JSON, confirm exact M2 running hash, and restart the client with bounded automatic config rollback on startup failure. Never change unrelated services.



#### M2 — operator-reported binary installation on both production nodes (2026-11-18; PADDING NOT ENABLED)

- **Homeserver:** operator supplied installer output `M1_INSTALL=PASS`, `CONFIG_CHANGED=NO` and `INSTALL_EXIT_CODE` not recorded in first output. Running executable separately confirmed at PID `2229132` as SHA-256 `64c3a7161d7d03b74ec70ea6ae2ceb97992e86d2dfa65e8662193ffc0bc7d42c`; retained prior M1 backup directory `/opt/tervon-tcp-bridge/backups/m0-aV1CqZ3C`. This is a direct M2 **binary installation** despite legacy installer output name `M1_INSTALL`.
- **Finserver:** operator ran explicit existing-unit preflight and install for `tervon-bridge-server.service` (live executable `/opt/tervon-bridge-linux-amd64`). Installer output confirmed `M1_INSTALL=PASS`, `CONFIG_CHANGED=NO`, `INSTALL_EXIT_CODE=0`; candidate M2 hash `64c3a7161d7d03b74ec70ea6ae2ceb97992e86d2dfa65e8662193ffc0bc7d42c`; retained prior M1 backup `/opt/tervon-tcp-bridge/backups/m0-cSx4CMGr`. **No separate post-install running /proc PID/hash supplied yet**; do not silently equate script PASS with independent runtime hash verification.
- **Operator console incident:** earlier installation attempts exited because `sudo` does not exist on Finserver (user is root), returning exit code 127; interactive shell `exit 1` / inherited `set -e` could close terminal. Successful Finserver install used `bash` with no `sudo` and safe shell flow. Do not recommend `exit` or `set -e` for interactive root-console instructions.
- **Current stage:** M2 binary INSTALL PASS as reported by operator on each node, **`padding_enabled` not changed** and pad1 live negotiation **NOT RUN / NOT VERIFIED**. Do not claim operational M2 Padding PASS or release. Next safe action is staged enablement on server first, client second with preserved configs and rollback. No unrelated production units touched. M3 remains blocked.



#### M2 — operator-approved direct deployment intent (2026-11-18, NOT EXECUTED)

- **Operator decision:** install M2 directly on the existing Homeserver ↔ Finserver Bridge as soon as it is installable; do not create, launch or require a separate test bench and do not make optional test procedures a condition of that operator decision. The operator accepts a direct production change rather than an isolated trial.
- **Safety of the actual change remains mandatory:** preserve verified, checksum-pinned current M1 binary and confidential configuration on both nodes, confirm the new M2 build provenance and compatibility, activate matching `padding_enabled` only at both endpoints in a safe sequence, verify systemd readiness and basic existing production connectivity immediately after restart, and automatically restore M1 if deployment or startup fails. These are integral installation and recovery steps, not a separate test program. Never change Xray, nginx, firewall, other services, or create lab listeners.
- **Execution blocker at this checkpoint:** no remote shell/SSH or server command-execution connector is currently available to this ChatGPT session. GitHub connector access can edit source/docs and inspect CI but cannot install or restart services on Homeserver or Finserver. No M2 host binary/config or last-minute on-host hashes have been inspected. Therefore authorization is recorded **conditionally on technical readiness**; **installation NOT EXECUTED**, no production service changed.
- **Do not misreport status:** code CI PASS is not the same as installed, operationally verified or released; M2 remains active and M3 blocked. The direct deployment decision supersedes earlier requirements for an independent test stand, but does not waive backup and automatic rollback.



#### M2 — read-only M1 host baseline collector prepared (2026-11-18, CI PASS; NOT RUN ON HOSTS)

- **Tool:** `tests/homeserver/collect_m2_baseline.py` (source SHA introduced at `5f87b507cbffc738332490cbb06f2a510da3f722`) — Python standard library only; takes explicitly chosen `--role` (`homeserver` or `finserver`) and existing systemd service unit name. Executes only `systemctl show` with allowlisted status properties, reads current `/proc/<PID>/exe` for SHA-256 and `/proc/<PID>/status` for RSS. Does NOT read Bridge configuration, TLS private keys, connection payloads, command lines, process environment or IP addresses; no shell, no service mutations, no files written by collector.
- **Safety and boundaries:** refuses inactive or malformed PIDs and invalid unit names; if running executable SHA differs from the last confirmed M1 SHA (`e42545d56ad05e94280466ddf92dfbcaa4955d716511f839de2eec0ae21675e5`), reports mismatch **without overwriting anything**. CPU %, RTT, loss/retransmits, Padding live state and all metrics not actually collected remain explicitly `NOT_MEASURED`. JSON output is for private operator review; do not upload identifiable operational snapshots to public GitHub.
- **Tests:** `tests/homeserver/test_collect_m2_baseline.py` (commit `2ade03ebc057844a8f8c3d70a0abdc0c9e1ab770`) mocks systemctl and uses fake procfs fixtures; validates output whitelist, SHA/RSS, command safety, rejection of invalid units, stopped services and absent metrics. CI workflow changes at `a24db02237ecbfd7692386cd1e204f78c4fee4e3`.
- **Verified CI:** [GitHub Actions #44240086746](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44240086746), source SHA `a24db02237ecbfd7692386cd1e204f78c4fee4e3`, job **SUCCESS**. Dedicated mocked collector test **PASS**, plus Go build/vet/unit/race, 3x unmodified policy, M1/M2 regression, bounded fuzz.
- **Operation:** `docs/M2_LIVE_ACCEPTANCE.md` explains optional operator-only collection. **Script NOT EXECUTED on Homeserver or Finserver**; no live M1 baseline gathered and no live M2 upgrade authorized. M2 **CODE PASS / LIVE PENDING**. No service touched, no M3, no v1.0.1 release.



#### M2 — repair intermittent default-policy test telemetry snapshot (2026-11-18; CI PASS)

- **Failure evidence:** GitHub Actions [#44233867705](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44233867705) on docs-only HEAD `89bef654d07532e46026bc003a01679684c633c0`: **FAIL** at `TestM2DefaultPolicyIdleResume`, message `PAD emitted after idle gap on an interactive frame`. Build/vet passed. This superseded the earlier single-run success as reliability evidence and was treated as a test-gate blocker.
- **Root cause and correction:** a complete echoed DATA payload may arrive before the *server* finishes writing the subsequent PAD frame and invoking its telemetry callback. The regression captured its pre-idle PAD baseline immediately following the echoed DATA, so it sometimes counted a delayed PAD from the prior eligible DATA frame as new PAD after the later short interactive request. Code commit `f190584e26abf61564505649e3a013453ed7a8d3` moves the baseline snapshot to **after** the >1s quiet period, immediately before the short interactive frame; the actual default policy, application traffic and PAD wire enforcement are unchanged.
- **Initial repaired CI:** [#44239131439](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44239131439), code SHA `f190584e26abf61564505649e3a013453ed7a8d3`, full job **SUCCESS** with build/vet/unit/race/M1-M2 regressions/fuzz.
- **Repeated CI gate:** workflow step `go test -race -count=3 -v -timeout=180s -run '^TestM2DefaultPolicyIdleResume$' .` at SHA `c9b369d3ed63b7fabbb9d17008414e0467d5e27a`. [GitHub Actions #44239483601](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44239483601), job **SUCCESS**, three repetitions **PASS** (7.31 / 7.20 / 7.28s), each 245,888 useful bytes per direction and 2,610 PAD TBF1 wire bytes per direction. No race report.
- **Scope:** regression verifies the CI fixture and conservative default PAD timing; does not prove production network behavior or lifetime resource stability. **CODE PASS / LIVE PENDING**. No M3, no release and no production changes.



#### M2 — default policy sustained/idle/reactivation regression (2026-11-18, CI PASS)

- **Documentation integrity:** prior `docs/DEVELOPMENT_PLAN.md` entry about `TestM2RepeatedSessionWaves` was damaged by an unsafe text replacement (unintended interpretation of `$'`), while source code and CI remained intact. Restored full plan content from the preceding verified document revision, reinserted the actual wave test evidence without string interpolation, and confirmed exactly one M2 session-wave section and one M3 heading. Repair commit: `67f86b5ef675d80f440e1e72930edee358cf7514`.
- **New real-policy test:** `prototype/padding_default_policy_test.go` at `b0336d2e4f0fa8bc9c31f8d7204b88145cfccae8` uses **unmodified `padding.DefaultPolicy()`** with actual wall clock: MinDuration=2 seconds, MinSamples=5, MinUsefulBytes=64 KiB, MaxInterSampleGap=1 second and RatioPercent=5. Authenticated in-process HTTP/2 + echo TCP verifies initial no-PAD, sustained upload and download activation, a >1s gap followed by an interactive frame that disables PAD without new overhead, and fresh-burst reactivation instead of inheriting prior burst duration. It verifies byte-exact echo, terminal session/active state OFF and cumulative PAD TBF1 wire budget.
- **Latest verified code SHA:** `b0336d2e4f0fa8bc9c31f8d7204b88145cfccae8`; [GitHub Actions #44233416106](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44233416106), job **SUCCESS**: build/vet/unit/race, repeated M1 and M2 tests, bounded PAD and M1 fuzz. No override of default detector thresholds in the new test.
- **Scope/remaining:** test exercises a single synthetic in-process CI connection and ~seconds-long pauses; not proof of production route loss/RTT/CPU/RSS, multi-hour stability or absence of long-lived leaks. Production M1 binaries/config remain unchanged; no lab units. M2 CODE CI PASS / LIVE PENDING / v1.0.1 NOT_RELEASED; M3 BLOCKED.



#### M2 — repeated HTTP/2 session-wave lifecycle and accounting (2026-11-18, CI PASS)

- **Code:** `prototype/padding_session_churn_test.go` committed at `4395dc8cac020601edd7723d541f8b09649df7fa`. Ten sequential waves of six simultaneous connections; six bidirectional 8192-byte echo exchanges per connection, all through one reusable authenticated HTTP/2 client. After **each wave**, verify Started=Completed, SessionsActive=0 and PAD states OFF on client/server, exact application byte counters, cumulative PAD wire overhead <= configured 5%. Test uses only GitHub CI in-process fixtures.
- **Dedicated CI step:** `go test -race -count=3 -v -timeout=150s -run '^TestM2RepeatedSessionWaves$' .` committed at `2c2ad62d6e40dd26296bfe9942a515df35279193`. [GitHub Actions #44227694584](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44227694584) job **SUCCESS**. Full build/vet/unit/race and existing M1/M2 regression and fuzz remain PASS.
- **Three detailed PASS results:** 60 sessions/run; useful bytes per direction **2,949,120**; actual PAD wire overhead per direction **147,420** bytes (within 5%). Process-wide TotalAlloc deltas **37,529,600 / 34,028,200 / 39,717,616 bytes** and HeapAlloc at end **5,933,416 / 5,686,824 / 7,373,664 bytes**. All three session-wave runs PASS (0.35 / 0.26 / 0.58 s); no race failures.
- **Limits:** repeated correctness test is not a long-running soak test or proof of absence of goroutine/memory leaks; process memory includes TLS and test infrastructure. No real route RTT/loss/retransmission or CPU/RSS measurements. **LIVE_PENDING**, no production changes, no M3, v1.0.1 NOT_RELEASED.



#### M2 — cancellation isolation under shared mTLS HTTP/2 (2026-11-18, CI PASS)

- **New regression:** `prototype/padding_cancel_isolation_test.go` at `6804bc4624b4d915e2a81bebbd2296d08f0e7d73`. A PAD-enabled stream exchanges valid bytes, then is canceled via `HandleContext` while a second stream uses the same HTTP/2 client. The canceled handler must return an error rather than successful FIN; the second completes byte-exact echo and normal close, both sides eventually report `SessionsActive=0` and PAD ACTIVE/OFF states OFF.
- **Repeat CI:** `.github/workflows/go-prototype.yml` at `cc1459995541e79acdfe6a31b83ae80df3a865fc` includes 3 repetitions of the cancellation isolation regression under Go race detector (in addition to full build/vet/unit/race, existing M1/M2 regressions, bounded fuzz).
- **Verified code SHA:** `cc1459995541e79acdfe6a31b83ae80df3a865fc`. [GitHub Actions #44223298969](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44223298969): full job **SUCCESS**. Named cancellation regression PASS on all three invocations; each logged 4505 PAD wire bytes emitted on upload and download across these sessions, zero remaining active client/server sessions and both PAD states OFF. No race report observed.
- **Evidence limits:** test uses in-process authenticated localhost HTTP/2 fixtures only; a 3x cancellation regression cannot establish lifetime/no-leak behavior under long-lived public network loss. Real Homeserver/Finserver RTT, loss/retransmissions, resource trends and approved rollback still unverified. Neither production node was touched; no new laboratory units started.
- **Status:** CODE CI PASS / LIVE PENDING / RELEASE NONE. Current module remains M2, M3 blocked.



#### M2 — concurrent multiplexed PAD stream regression (2026-11-18, CI PASS)

- **Implementation:** `prototype/padding_concurrency_test.go` at commit `ddbd28549be52a93457f2b0028175eff9a4f1d48`. Uses the existing in-process authenticated HTTP/2 test helper with 12 concurrently active independent TCP sessions, 12 bidirectional 8192-byte echo exchanges per session, independent data patterns, aggregate telemetry and explicit session end. No Homeserver/Finserver service or lab unit used.
- **CI reinforcement:** `.github/workflows/go-prototype.yml` at `a22f3a71a1c0af8739b3ae09333e0d021c7a0734` runs 3x race-instrumented verbose concurrency regression in addition to full build/vet/unit/race, repeated M1+M2, bounded fuzz, latency and encrypted-byte diagnostics.
- **Verified CI:** [GitHub Actions run #44221520831](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44221520831), **SUCCESS** on code SHA `a22f3a71a1c0af8739b3ae09333e0d021c7a0734`. Repeated 3x concurrency results: useful bytes each direction **1,179,648**; actual PAD wire overhead each direction **58,980 bytes** (just under 5%), byte-exact echo, all client/server sessions active=0 and Padding ACTIVE states cleared. No data corruption or race failure observed.
- **Diagnostic memory logs (three samples):** process-wide TotalAlloc deltas **10,099,504 / 9,761,000 / 10,282,048 bytes** and ending HeapAlloc **3,798,592 / 3,461,424 / 4,553,360 bytes**. Goroutine snapshot before=4 and after=7 (includes HTTP/2 transports/test process): **NOT** evidence of a goroutine leak or of no leak, because it is not a post-idle stabilized count. Not a per-session memory figure.
- **Scope:** CI localhost mTLS HTTP/2 multiplexing only. No proof of real route RTT/loss/retransmissions or sustained 24/7 behavior. No production service modified. M2 remains **CODE_PASS / LIVE_PENDING**. Do not transition to M3 or release v1.0.1 until controlled live gate and rollback PASS.



#### M2 — controlled live acceptance preparation (2026-11-18, PREPARED ONLY)

- **Documentation:** `docs/M2_LIVE_ACCEPTANCE.md` added at `986bff28de1ad7a5999ad8f9fe157444e68612ae`. It specifies a **read-only** per-node baseline (running PID/executable SHA, service health, nonsecret settings, production traffic/CPU/RSS/error/RTT sources), explicit operator authorization before production changes, agreed acceptance thresholds and checksum-pinned M1 rollback requirements.
- **Current state:** checklist created; **no host data collected**, no synthetic/laboratory services run, no production binary/config changed. The current source evidence remains CI PASS #44218917305 at `025e8c71f6f7500ce96b08305015faa3a7a3be6e`, plus docs-only commits.
- **User constraints honored:** live production services remain untouched. A normal `continue` request is not authorization to replace a running service or to run new test units. Live confirmation remains a distinct gate from CI. M2 IN_PROGRESS / LIVE_PENDING / v1.0.1 NOT_RELEASED; M3 BLOCKED.



#### M2 — in-process latency / allocation diagnostics (2026-11-18, CI PASS)

- **New code/test:** `prototype/padding_latency_diagnostic_test.go`, commit `879de1bae1aead55235e95b89d78dbf091882608`. Four 48-exchange workloads (M1/M2 × 128 B/16 KiB) run through authenticated loopback HTTP/2 over TLS with an echo backend; exact payload equality and PAD behavior are asserted. Report nearest-rank p50/p95/p99, process-wide `TotalAlloc` delta and ending `HeapAlloc` as diagnostics, not service-level resource guarantees.
- **CI:** `.github/workflows/go-prototype.yml` adds a verbose named diagnostics step at commit `025e8c71f6f7500ce96b08305015faa3a7a3be6e`. [GitHub Actions run #44218917305](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44218917305), job conclusion **SUCCESS**: full build, vet, unit, race, repeated M1/M2, three bounded fuzz targets and existing codec benchmarks. Verified source SHA `025e8c71f6f7500ce96b08305015faa3a7a3be6e`.
- **One CI sample (48 echo exchanges each):** M1 128 B p50=114.112 µs, p95=148.224 µs, p99=4.035413 ms, total allocated delta=480824 B, heap end=1972232 B; M2 128 B p50=100.471 µs, p95=132.249 µs, p99=3.46089 ms, total delta=376232 B, heap end=3037192 B. **No PAD generated in the short case.**
- **One CI sample sustained 16 KiB:** M1 p50=158.288 µs, p95=225.108 µs, p99=3.585938 ms, total allocated delta=2363256 B, heap end=2465400 B; M2 p50=192.51 µs, p95=366.3 µs, p99=3.48132 ms, total delta=2439968 B, heap end=2579880 B. **PAD wire bytes upload/download=24534/24534 B** on M2; M1=0.
- **Limitations:** four serial, order-dependent tests on one shared CI process, tiny sample and loopback network, no induced RTT/loss, unsynchronized GC and no CPU isolation. Values cannot establish production p95/p99, statistical regression, absolute heap consumption, or live-route SLO. `TotalAlloc` counts the whole process including testing and TLS, not a per-connection allocation. No thresholds derived from these data alone.
- **Open gates:** real Homeserver ↔ Finserver route evidence, accepted latency/CPU/memory/loss limits, signed off restore/rollback plan and controlled production verification; **no existing units touched, no M3, no v1.0.1 release**.



#### M2 — decoder receiving-side wire cap and PAD write failure (2026-11-18, CI PASS)

- **Fix:** `prototype/framing/stream.go` commit `3e24ad53f17fd1f91ebfb6cf4a210237b56d7294` now includes the ten-byte TBF1 PAD header in the cumulative receiving-side **25%** protective budget. Previously the receiving-side limit counted PAD payload only, allowing overhead greater than 25% when PAD frames are small. Sender's stricter default 5% wire budget remains unchanged.
- **Tests:** `prototype/framing/padding_test.go` commit `45e108ad0a621df92064004769d91c73921533d8` validates rejection of 25-byte PAD payload + 10-byte header with 100 useful bytes (35% wire overhead), accepts 15-byte payload + 10-byte header (exactly 25%), and injects PAD header write failure. Failed PAD write must surface an error, close encoder, never forge FIN and never increment the emitted-PAD telemetry callback.
- **Verified code SHA:** `45e108ad0a621df92064004769d91c73921533d8`. [GitHub Actions #44218024674](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44218024674), job **SUCCESS**. Includes build/vet/unit/race, repeated M1+M2 tests, encrypted HTTP2 diagnostics, codec benchmarks and bounded fuzz. No production services changed.
- **Remaining:** release gate still **NOT MET**. Need per-route CPU/heap, latency p95/p99, short-flow behavior, loss/retransmission evidence, approved rollback and production verification. No M3, no v1.0.1 release.



#### M2 — encrypted HTTP/2 + mTLS wire accounting (2026-11-18, CI PASS)

- `prototype/padding_encrypted_wire_test.go` at `c3467cc139e69a63c00271df256031d3776632f0` measures actual TCP byte counts at the listener **below TLS**, using independent M1/M2 mTLS HTTP/2 sessions and identical echo payload in both directions. The test confirms exact useful bytes and that M2 emits PAD both ways, but purposely does not assert a fixed ratio on total TLS bytes (scheduler/record segmentation vary).
- Initial integration test CI [#44216532330](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44216532330) succeeded. Separate verbose diagnostics added by `59484421f1a682daf0bc596c200dd9edeaf0064b`; latest full CI [#44216825423](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44216825423) job **SUCCESS** (build/vet/unit/race, repeated M1/M2, bounded PAD/M1 fuzz, benchmark).
- **Measured on GitHub Actions Go 1.23.0 runner, one sample:** useful TCP bytes per direction **196608**. M1 encrypted network bytes: client→server **201365**, server→client **201345**. M2: client→server **207793**, server→client **207841**. Difference **6428 bytes (3.27% useful)** and **6496 bytes (3.30% useful)** respectively. M2 emitted PAD wire overhead **5742 bytes in each direction**, including PAD frame headers. Differences beyond PAD wire overhead include altered TLS/HTTP2 framing and timing, not an isolated constant overhead attribution.
- These are measurements of **local encrypted TCP bytestream only** (TLS records + HTTP/2 + TBF1). They exclude IP/TCP headers, retransmissions/packet loss, genuine cross-site RTT, congestion, real server resource contention and long-term statistics. Do NOT generalize to actual VPN network latency or throughput. No representative repeated distribution or bounds available yet.
- **Diagnostic CPU:** detector 18.04 ns/op, 0 B/op, 0 allocs; short M1 45.40 ns/op, 32 B/op, 2 allocs; short M2 120.5 ns/op, 256 B/op, 4 allocs; sustained M2 PAD 1553 ns/16KiB, 543 B/op, 3 allocs (single hosted runner, io.Discard codec-only).
- **Gates still OPEN:** real RTT/p95/p99 latency, CPU/heap under traffic, packet loss/retransmissions, short/interactive behavior on supported live Homeserver/Finserver, accepted thresholds, checksum-pinned backup/rollback. **No production changes or M3; v1.0.1 NOT_RELEASED.**



#### M2 — serialized TBF1 overhead verification (2026-11-18, CI PASS)

- New `prototype/framing/padding_wire_test.go` commit `0dfcad1ca9417602c9f5ba7654c404d03c904c41`: independently parses complete TBF1 frames to compute useful DATA bytes, DATA/PAD/FIN frame counts, total PAD payload and 10-byte headers. Cases: short interactive, sustained, sustained-to-short pulse with idle gap.
- **Confirmed CI:** [run #44211511127](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44211511127), code SHA `0dfcad1ca9417602c9f5ba7654c404d03c904c41`, job SUCCESS; build/vet/unit/race, M1/M2 repeat regressions, bounded fuzz all PASS. Test verifies PAD TBF1 wire overhead <= configured percentage; short-flow wire exactly equals M1 format; full byte-exact decode and one FIN.
- **Diagnostic benchmark at this run:** detector 28.69 ns/op, 0 B/op, 0 allocs; short M1 67.98 ns/op / 32 B/op / 2 allocs, short M2 164.3 ns/op / 256 B/op / 4 allocs; sustained M1 35.76 ns/op / 16 B/op / 1 alloc, M2 PAD 2443 ns/op / 543 B/op / 3 allocs. Values from one hosted runner run are noisy; do not claim regression or improvement across hosts/runs as definitive. Codec benchmarks use `io.Discard` and deliberately exclude HTTP/2/TLS/network.
- **Remaining M2 acceptance:** an agreed controlled measurement of short/intermittent latency, actual HTTP/2/TLS overhead and process CPU/memory/RTT/loss under supported live service conditions, with backup, rollback and evidence. **No deployment, no M3, v1.0.1 NOT_RELEASED.**



#### M2 — short-flow fast path and comparative benchmarks (2026-11-18, CI PASS)

- **Baseline PASS:** [GitHub Actions run #44200769174](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44200769174), SHA `318ad9bfb3d1fcbb0b198dfa717b4a30860ec723`, SUCCESS (build/vet/unit/race/repeated M1+M2/fuzz + comparative benchmarks). Go 1.23.0 linux/amd64 Ubuntu hosted runner.
- **Baseline codec-only benchmarks (not network):** short 128B M1 51.72 ns/op, 32 B/op, 2 allocs/op; M2-enabled short 166.2 ns/op, 256 B/op, 4 allocs/op. Sustained 16KiB M1 26.73 ns/op; M2 PAD 2096 ns/op, 543 B/op, 3 allocs/op. Not representative of VPN throughput; writes used `io.Discard`.
- **Optimization:** `padding.State.ObserveSmallFrame` fast-path keeps short/interactivity below `MinFrameBytes` OFF, counts useful bytes and resets sustained burst classifier without `time.Now`. Encoder checks it before full sampling. Tests verify 1000 small frames cannot activate, followed by fresh eligible burst and OFF transition.
- **Optimized CI PASS:** [GitHub Actions run #44201265799](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44201265799), exact source SHA `f2c6e4cad6b1189001510cbbd0d935faf978b131`, SUCCESS: build/vet/unit/race, repeated M1 and M2 tests, three bounded fuzz cases. Go 1.23.0 linux/amd64 hosted runner.
- **Optimized benchmark:** M1 short 50.53 ns/op / 32 B/op / 2 allocs; M2-enabled short **122.3 ns/op / 256 B/op / 4 allocs**. M2 short CPU benchmark improved relative to 166.2 ns/op initial observation, but remains ~2.4x baseline M1 CPU in this microbenchmark. Sustained M2 PAD 2100 ns/op / 543 B/op / 3 allocs (similar to baseline). Variations across runs not statistically controlled.
- **Optimized detector microbenchmark:** 21.71 ns/op, 0 B/op, 0 allocs/op (single runner result).
- **Remaining:** measured live latency, CPU/load under real traffic and RTT/loss, short/pulsed streams E2E and full HTTP2/TLS overhead; agreed acceptance thresholds, rollback, controlled Homeserver/Finserver test. **CODE PASS does not mean M2 PASS. Production changes NONE, M3 not started, v1.0.1 NOT_RELEASED.**



#### M2 — verified integrated code gate (2026-11-18)

- **CI failure:** GitHub Actions [run #44200049897](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44200049897) at `85c10507c7b2f0c49e930091c914ae507ff80c05` — FAIL in unit tests, because two legacy telemetry endpoint tests still required `padding_state=NOT_IMPLEMENTED` after the M2 local contract changed to `OFF`. Build and vet were already PASS. Earlier malformed workflow was repaired in commit `85c10507c7b2f0c49e930091c914ae507ff80c05`.
- **Corrections:** `prototype/telemetry/http_integration_test.go` commit `604652273b432b290f279921362d4d8e894ff854`; `prototype/telemetry/listener_test.go` commit `820e705e8f5f8a2a7e8246c7a0413483c053594f`. Tests now require Padding OFF by default and Jitter NOT_IMPLEMENTED.
- **Fully verified source SHA:** `820e705e8f5f8a2a7e8246c7a0413483c053594f`, [GitHub Actions run #44200189036](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44200189036), Ubuntu runner Go **1.23.0 linux/amd64**, job `test` **SUCCESS/PASS**. Build/vet/unit/race, repeated M1 regression, repeated M2 padding policy race, repeated M2 HTTP/2 E2E race, bounded M2 PAD fuzz and both M1 fuzz targets passed.
- **Fuzz counts:** `FuzzPaddingDecoder` **1,189,092** executions / 20 corpus total; `FuzzReadFrame` **1,280,882** executions / 11 total; `FuzzStreamDecoder` **1,218,455** executions / 14 total. Bounded 15s runs, not exhaustive.
- **CPU/alloc diagnostic:** `BenchmarkPaddingObserve-4` **18.10 ns/op, 0 B/op, 0 allocs/op** on hosted runner; only detector hot path, not TLS/HTTP2/network or end-to-end overhead.
- **Remaining before M2 PASS:** quantify performance on short/intermittent/sustained traffic and full frame+TLS/HTTP2 overhead, verify actual Homeserver/Finserver controlled live route with backup and rollback, and record per-node evidence. `gofmt -l` is a **non-blocking** CI step, not a formatting PASS. **No production changes, no release, no M3.**



#### M2 — strict wire budget correction (2026-11-18, awaiting current-SHA CI)

- **Source checkpoint before this docs commit:** `d786966f96df12bcc236dc555fe5ff43ba58bdf4`; feature branch `work/m2-adaptive-padding`, draft PR #1.
- **Code:** `padding.State` now tracks PAD payload and PAD wire overhead separately. `Observe` refuses PAD unless the remaining cumulative `RatioPercent` budget covers **both** the 10-byte TBF1 PAD header and all planned PAD payload bytes. This replaces the earlier payload-only enforcement and keeps the real wire overhead within the configured ratio.
- **Tests:** `TestBudgetIncludesPaddingFrameHeader`, updated cumulative-budget and overflow tests, and `TestM2PaddingHeaderConstantMatchesFraming` to catch mismatch between frame codec and policy accounting.
- **Status:** IMPLEMENTED / TESTS WRITTEN, latest CI **NOT VERIFIED**; don't extrapolate PASS #44195491683 to post-telemetry and strict-budget changes. Not deployed; no working service changed; no release.
- **Next gate:** full latest-SHA GitHub CI and evidence, then measure performance/allocations and controlled live verification with rollback. M2 remains IN_PROGRESS.

#### M2 — correction batch: accurate wire overhead and early PAD rejection (2026-11-18)

- **Source checkpoint before documentation:** `4c86d7417c2bf4db65c29f900ead0e02f494dfb7` on `work/m2-adaptive-padding`. Draft PR #1 still open and not merged.
- **Metrics:** separate PAD payload and total PAD wire overhead (PAD payload + 10-byte TBF1 header per PAD frame); per-direction frame counts and wire overhead percentages in JSON and Prometheus; saturation-safe snapshot arithmetic. Actual TCP payload metrics remain unchanged.
- **Parser hardening:** writer rejects PAD payload >4096 bytes; receiver detects overlarge PAD directly from TBF1 header *before allocation*. Test simulates forged oversized wire header, not a permitted writer frame.
- **Tests:** updated telemetry tests for PAD frame/header accounting and uint64 saturation; adjusted frame rejection tests.
- **Latest full CI:** **NOT VERIFIED on these commits**. Last independently checked successful CI remains run #44195491683 for earlier SHA `fe4bc73f492845007c67ecc5d24c890e1b3f3f60`. GitHub workflow-run lookup currently has no newer confirmed result; do not claim latest SHA PASS.
- **Pending inside M2:** full current-SHA build/vet/unit/race/fuzz, benchmark CPU and allocations, end-to-end traffic overhead and short-flow performance, controlled live verification/rollback. **No M3 and no deployment until gates PASS.** Working M1 services unchanged.

#### M2 — next code checkpoint (2026-11-18, NOT VERIFIED on latest SHA)

- Source branch `work/m2-adaptive-padding` is still a **draft PR #1** against `work/m1-framing`.
- Fixed active-state cleanup on failed streams: `Encoder.Abort()` clears state without sending FIN; both client/server defer it for failed/incomplete sessions. Regression test checks no false FIN.
- Extended HTTP/2 mTLS integration with test-only rapid detector thresholds to require **actual PAD emission in upload and download**, verify byte-exact echo and no leaked ACTIVE after stream close. This is new code/test, **not a verified PASS**.
- Corrected classifier to reset **current-burst useful bytes** after long idle gaps, preserving cumulative byte/overhead accounting. Regression test blocks short post-idle burst from reactivating using historic bytes.
- CI now additionally runs 3x M2 race E2E, 15s M2 decoder fuzz and detector CPU/allocation benchmark (`f12a657798f840e804ee56fc579f8376123f0ad0`).
- Contract and controlled verification requirements: `docs/M2_PADDING_PROTOCOL.md`.
- **Last confirmed PASS remains** initial GitHub Actions #44195491683 at `fe4bc73...`. Latest implementation includes subsequent code/telemetry/test changes; **latest CI PASS not yet confirmed**. Homeserver and Finserver production stay on M1; no modifications performed. M2 IN_PROGRESS, v1.0.1 NOT_RELEASED.

#### M2 — telemetry extension and follow-up tests (2026-11-18, awaiting CI)

- Added aggregate PAD metrics in `prototype/telemetry/metrics.go` and `prototype/telemetry/prometheus.go`: upload/download PAD bytes, overhead% relative to application bytes, ACTIVE/OFF per direction and transition counts. The metric intentionally counts **PAD payload only**, not TBF1 frame headers, HTTP/2 or TLS overhead. No IDs, IPs, payload content or dynamic labels.
- `prototype/framing/stream.go` now emits single-writer synchronous callbacks when PAD frames are successfully written and when a stream changes ACTIVE/OFF; `prototype/bridge.go` attaches them independently for client upload and server download.
- `prototype/cmd/bridge/main.go` sets local `padding_enabled` configuration status for metrics; defaults still OFF.
- `prototype/telemetry/metrics_test.go` checks OFF/ACTIVE, overhead ratios, transitions and Prometheus newline output; `prototype/framing/padding_test.go` checks callback accounting/final OFF.
- **Status:** CODE_COMMITTED; latest post-telemetry source SHA `d75088f55f8faa16b2804c06d3951508d4410ee0` (before this docs commit); new CI/race/fuzz/benchmark results **NOT YET VERIFIED**. Earlier PASS #44195491683 covers **only** snapshot `fe4bc73...`, and does not extend to these changes. Production NONE; release NONE.
- Pending: final CI gate on latest source; review of correctness and cancellation, resource/latency measurement, controlled live deployment with backup/rollback, explicit acceptance thresholds. No M3 before M2 PASS.

#### M2 — GitHub CI verification checkpoint (2026-11-18)

- **Подтверждённый CI кодового snapshot:** `fe4bc73f492845007c67ecc5d24c890e1b3f3f60`, [GitHub Actions run #44195491683](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44195491683), job test conclusion **SUCCESS**.
- **Фактически прошли:** Go build ./..., go vet ./..., unit ./..., race ./..., повторный M1 race, повторный M2 padding policy race, два 15s старых M1 fuzz targets, syntax checks и интеграционные тесты `TestM2*` внутри общего unit/race. Ограничение: отдельных повторных M2 E2E и M2 fuzz в этом snapshot ещё нет.
- **После PASS добавлено CI усиление** `cb21a654a62f59a56aa5068554b4025c9d07769d`: 3x M2 HTTP2 race regression и 15s `FuzzPaddingDecoder`; и `8487aaf679fbe4a61bb275498783752ead827a54`: padding detector benchmark. **Результат на этом новом HEAD ещё не подтверждён.**
- **Production:** M1 без изменений, M2 не установлен; ещё нет измеренной live overhead/latency/CPU и отдельной оценки compliance/performance. **M2 IN_PROGRESS, не PASS.**

#### M2 — TBF1 PAD negotiation и decoder, кодовый пакет (2026-11-18, UNVERIFIED)

- **Ветка:** `work/m2-adaptive-padding`. Draft PR: https://github.com/tervon-software/Tervon-TCP-Bridge/pull/1 (base `work/m1-framing`, **не сливать без PASS**).
- **Реализован код (но ещё не подтверждён тестами):** `prototype/framing/framing.go` — новый TypePadding=3 только через WritePaddingFrame/ReadFrameWithPadding, legacy ReadFrame и WriteFrame сохраняют жёсткий отказ. `prototype/framing/stream.go` — NewPaddingEncoder/Decoder; PAD после DATA и перед FIN, PAD удаляется до TCP backend; crypto/rand; decoder запрещает PAD до DATA, повторный PAD, >4096 байт на PAD и >25% накопленного полезного объёма.
- **Согласование:** в `prototype/bridge.go` строгий capability `tbf1-fin,pad1` на /v2/stream, симметричные request/response заголовки, отказ 426 при неподдерживаемом PAD. Старый `tbf1-fin` остаётся default. **Никакого автоматического silent fallback**, потенциально скрывающего ошибку.
- **Конфигурация:** `prototype/cmd/bridge/main.go` принимает `padding_enabled` (по умолчанию false), только с /v2/stream на клиенте; `padding.DefaultPolicy()` — предварительные консервативные лимиты, НЕ оптимизированные экспериментально.
- **Тесты записаны:** `prototype/framing/padding_test.go`, `prototype/padding_integration_test.go`, `prototype/padding/policy_test.go`. Проверяют byte-exact, legacy rejection, повреждение/излишки PAD, CAP mismatch, полномочный HTTP/2 mTLS, короткие потоки, OFF, лимиты, изоляцию направлений.
- **Результат:** CODE_COMMITTED, **CI PASS NOT VERIFIED**, Homeserver/live NOT_RUN, production changes NONE, release NONE. Ожидаются лог GitHub CI, gofmt/vet/race/fuzz и оценка накладных затрат.
- **Незакрытые части М2:** telemetry PAD по направлениям, config policy bounds/hysteresis, проверка качества адаптации на устойчивых/порционных потоках и измерение CPU/heap/latency, контролируемая live проверка с backup/rollback.
- **Правило безопасности:** не устанавливать M2 на Homeserver/Finserver и не менять действующий TBF1 до подтверждённой проверки кода и отдельного контролируемого решения.

#### M2 — первый пакет (2026-11-18, UNVERIFIED)

- **Ветка:** `work/m2-adaptive-padding`, основана на M1 docs closure `58a9b61542749f27bef2586d9eb0db1dbaacee05`.
- **Код:** `prototype/padding/policy.go` (`e7a3502826b244aae5b16946d55850ffc04cc7ea`) — независимый от TCP payload детектор активности и накопительный процентный бюджет. Включение требует продолжительности, числа выборок и объёма; после паузы выключается. Направления должны создавать отдельные экземпляры State. Алгоритм не вводит задержку и не отправляет байты.
- **Тесты:** `prototype/padding/policy_test.go` (`a51868a1581310a0d48c4d9d9d241086e347bbba`) — короткие/интерактивные OFF, длительный поток, бюджет, idle, изоляция направлений и overflow. **Только написаны, ещё не подтверждены PASS.**
- **CI:** `.github/workflows/go-prototype.yml` (`7bbd90ad5888f8ab73df24882da0a2d200902f53`) содержит отдельный повторный race-check пакета padding. Результаты CI пока **NOT_VERIFIED**.
- **Открыто:** wire capability negotiation с согласованной поддержкой padding; frame codec без изменения полезных данных; интеграция client/server и CLI-конфига; telemetry; regression/fuzz/race/e2e; доказанное ограничение overhead/latency и отдельная согласованная реальная проверка перед вводом в эксплуатацию.
- **Статус:** M2 IN_PROGRESS, NOT_PASS, NOT_DEPLOYED, RELEASE_NONE. Production changes: NONE.


**Пакет:** пассивные метрики, детектор устойчивых потоков, гистерезис, бюджеты overhead, раздельная адаптация направлений, OFF для интерактивного трафика; unit/integration tests.
**Готово, когда:** нулевая порча данных, измерены overhead и ресурсы, отсутствует деградация на коротких потоках сверх утверждённых пределов.

### M3 — Адаптивный Jitter

#### M3 — operator-confirmed Homeserver candidate STAGING PASS (2026-11-19)

- Operator executed `tests/homeserver/stage-m3-candidate.sh` on **Homeserver**, with exact GitHub SSH branch `work/m3-adaptive-jitter` and source `ba8d6cedbac58fc9c1b4e81872e127075783a3b9`. **Actual result:** `STAGE_M3=PASS`, `STAGING_COMMAND_EXIT_CODE=0`, `tervon-bridge-m3-linux-amd64: OK`, `SERVICES_RESTARTED=NONE`, `PRODUCTION_BINARY_REPLACED=NO`.
- Immutable candidate path: `/opt/tervon-tcp-bridge/m3/stage-qG1sStTg/dist/tervon-bridge-m3-linux-amd64`; binary SHA-256 `df3bcdfc70dfa8f238af454cd9fa1e852188b61233799972fd38bde48c8aa499`. Preserve `SHA256SUMS` and `BUILD_INFO` in the staging directory for later provenance verification; do not copy over the active Bridge yet.
- **Scope of PASS:** reproducible build/staging on Homeserver only, not runtime testing or M3 release. Running Homeserver and Finserver Bridge production binaries are still M2 as far as this operator evidence establishes; no current Finserver M3 artifact or validated installation path was provided.
- **Next gate before modifying live systems:** confirm both nodes' intended unit `ExecStart`, actual active binary SHA, config path/mode, stable rollback directory and M2 backup, staged M3 SHA, corresponding bilateral `padding_enabled`, `padding_rate_enabled`, `jitter_enabled` flags, and readiness/metrics. Prepare a two-node dry-run/rollback plan, avoid unrelated services, then run controlled operator-authorized tests for 1-Mbps PAD and bounded Jitter and mark PASS only on measured evidence. **Status: M3 STAGING HOMESERVER PASS; TWO-NODE LIVE NOT EXECUTED.**



#### M3 — isolated candidate staging prepared (2026-11-19; NO HOST ACTION)

- Full previous M3 concurrency tests [#44434798348](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44434798348) **SUCCESS**, Linux M3 candidate [#44434798307](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44434798307) **SUCCESS** on precise test commit `e76a4d2b4062892b8fb7bbde39fe3251677bd765`. Confirms independent qualified 1-Mbps video versus 128B interactive traffic behavior in mocked-time encode/decode tests, **not live RTT or deployment**.
- Added `tests/homeserver/stage-m3-candidate.sh`: exact 40-character commit SHA is required as input; the existing checkout must be clean, on `work/m3-adaptive-jitter`, and have exactly SSH remote `git@github.com:tervon-software/Tervon-TCP-Bridge.git`. Fetches the pinned M3 commit via SSH and builds with Go from an immutable `git archive` in a private new `/opt/tervon-tcp-bridge/m3/stage-XXXXXXXX`. Creates candidate binary, `SHA256SUMS`, `BUILD_INFO`. **No replacement, deploy, systemctl, nginx, service restart or Finserver modification.** CI checks script syntax. Staging is **PREPARED / NOT RUN ON HOMESERVER** at this checkpoint.
- After source CI PASS for staging script, operator may run the script to stage candidate without live interruption. Before actual M3 activation, a separate guarded two-node install plan with backups and verified rollback must be approved. On any version or remote mismatch, stop; do not perform live changes. Production remains **M2**; public Stats UI/collector already **LOCAL PASS**, do not reinstall.



#### M3 — resumed after interrupted session (2026-11-19)

- Re-read current `docs/DEVELOPMENT_PLAN.md` and `docs/DEVELOPMENT_LOG.md` on `work/m3-adaptive-jitter`, rather than relying on chat memory. **Previously pending** CI [#44281963459](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44281963459) (Go/race/integration) and [#44281958023](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44281958023) (M3 Linux build candidate) both completed **SUCCESS**. No production installation is implied.
- Latest external operator evidence already in the GitHub log: M3 Stats UI+collector **HOMESERVER LOCAL PASS** at `5015a3b6f5eff704aa0cd317ecb69228f49b4c80`, with separate backup paths and **no Bridge restart**; screenshot shows real M2 PAD upload emitted (18.3 KiB/41 frames) and video/inactive status. Do **not** rerun the already-completed Stats installers or assert M3 data plane deployed.
- Added deterministic `prototype/framing/m3_competing_flow_test.go` to simulate a single-direction 1 Mbps video stream alongside 128-byte short interactive writes with independent encoder states: assert PAD+Jitter activates for qualified video only, **zero** PAD/Jitter for interactive stream, and byte-exact decoded output. The test runs ~7 simulated seconds to cover the rate and Jitter qualification windows. No real traffic, server changes or user data involved.
- Current new CI: [Go #44434798348](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44434798348) **SUCCESS / CI PASS** for source `e76a4d2b4062892b8fb7bbde39fe3251677bd765`, including new independent video/interactive flow test, unit, race, Go build/vet, M1/M2 regressions and bounded fuzz; [M3 Linux candidate #44434798307](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44434798307) **SUCCESS** (artifact only, not deployed). This is CODE PASS, not live performance/RTT or release.
- **Final staging-script CI:** [Go #44435464566](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44435464566) **SUCCESS**, including `bash -n tests/homeserver/stage-m3-candidate.sh`, new competing-flow test, full unit/race/regressions/fuzz. [M3 Linux candidate #44435464622](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44435464622) **SUCCESS**, artifact-only. No candidate staging yet executed on Homeserver. Remaining gate is real two-node M3 rate1/jit1 testing with rollback, timing and fairness, then independent external Stats HTTPS verification. **M3 IN_PROGRESS, v1.0.1 NOT RELEASED.**



#### M3 — rate1+jit1 batch validation checkpoint (2026-11-18; ACTIVE SOURCE, NO DEPLOYMENT)

- **Latest module checkpoint:** `rate1`/bilateral `jit1`, decoded PAD monitoring and all current CI checks are **CODE PASS** on `35f38cc82bf6fb7ef85ab6726654cf2010a461fa`. M3 **NOT COMPLETE**: Homeserver/Finserver opt-in rollout, real ≥1Mbps stream behavior/overhead, long-running RTT/fairness and public monitoring route still **NOT VERIFIED**. No deployed file or service changed in this source batch.
- **Flow model corrected:** video stream input may be evenly paced or transferred in larger chunks. `RateState` now uses solely useful-byte thresholds over **two fully elapsed 1-second windows**; no minimum individual TCP chunk size or number of chunks. Default activation ≥1,000,000 bits/sec (~125,000 bytes per second), exit below 700,000 bits/sec, 1-second idle reset. Tests cover 1 Mbps regular small chunks **and** a single 125,000-byte chunk per second, 4 Mbps fragmented streams, below threshold, short burst, hysteresis, isolation, saturation-safe arithmetic, PAD wire overhead ≤5%. This is an **M3 option**; installed M2 stays on original pad1.
- **Jitter correlation and fairness:** in the M3 rate-based encoder, Jitter is gated by the rate classifier, bounded to 4ms per eligible DATA frame and 20ms per stream/direction/second, and bypassed on congestion/small interactive frames without erasing sustained large-frame history. New tests cover 32 competing stream states, cancel/backpressure, interleaved tiny writes, PAD+Jitter byte-exact frame decode. It remains a conservative scheduler: a qualified stream composed solely of tiny writes can be padded but may deliberately receive **no Jitter** to avoid interactive latency.
- **Bidirectional PAD visibility:** authenticated Decoders now count received and validated PAD frames on both nodes, so Homeserver `padding_download_bytes` may increase during downstream video in M3 instead of always showing zero. Invalid PAD is not counted. Site telemetry also exposes opt-in feature flags and jitter delay counts; UI distinguishes waiting from disabled. **Changes not installed on Homeserver Stats yet.**
- **CI history:** earlier [Go checks #44280393604](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44280393604) **PASS** before latest refinements; the pre-fix batch [#44279612151](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44279612151) failed on a double-clock test and was corrected by single ingress-time sampling in `1b2c9c7cfc91ff032a967eb16a492142b2c9c2ad`. Latest combined [#44280950111](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44280950111) **SUCCESS / PASS** for source commit `35f38cc82bf6fb7ef85ab6726654cf2010a461fa`: Go build, unit, race detector, integration, fuzz, monitoring adapters and repeated M1/M2 regressions. [M3 Linux candidate #44280950001](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44280950001) also **SUCCESS** (artifact only). No live M3 deployment or external monitoring PASS is implied. [M3 Linux candidate #44280950001](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44280950001) **SUCCESS** (build artifact only, no release/install).
- **Acceptance still open:** latest-source CI PASS; validate Jitter against real competing TCP flows and RTT, fairness, cancellation/retries, upload/download, server capability mismatch, PAD activity at ≥1Mbps on the actual Homeserver/Finserver; update Bridge Stats site safely and confirm external DNS/TLS. Only then declare M3 COMPLETE / release v1.0.1 later after M4/M5.



#### M3 — 1 Mbit/s throughput policy and negotiated Jitter implementation (2026-11-18; CODE IN PROGRESS)

- Operator confirms a use case of steady video streaming at **1 Mbit/s and above**; real video applications may also transfer in bursts, so acceptance must include both regular and chunked load patterns. This modifies only M3 in GitHub, not installed M2.
- **M3 opt-in classifier:** `prototype/padding/rate.go` + `rate_test.go` introduce independent per-stream/per-direction useful-throughput classification: ≥1,000,000 bits/s across **two consecutive 1-second windows**, entry after ~2s **without any minimum number of TCP writes**; exit below 700,000 bits/s per subsequent completed window; reset on gaps greater than 1s. Small frames contribute normally instead of forcing reset. Cumulative **5% PAD wire overhead** still enforced, including frame header. Explicit `padding_rate_enabled` defaults to false; existing M2 `padding_enabled` remains the unchanged `pad1` path. Tests cover 1Mbps fragmented video, 4Mbps interspersed small writes, below-threshold, short bursts, hysteresis, inactivity, independent directions, budget and edge arithmetic.
- **M3 transport integration:** `prototype/framing/stream.go` opt-in `NewRatePaddingEncoder`. `prototype/bridge.go` accepts bilateral `tbf1-fin,pad1,rate1` and `tbf1-fin,pad1,rate1,jit1` only if enabled on matching peer; rejects unsupported capability with 426. M2 `tbf1-fin,pad1` and M1 `tbf1-fin` unchanged. `prototype/cmd/bridge/main.go` validates `padding_rate_enabled` requires `padding_enabled` and `jitter_enabled` requires rate padding. Separate authenticated HTTP/2 tests validate actual byte-exact fragmented flow, independent negotiation, 426 fail-closed, and no stream hang.
- **Metrics/UI:** `prototype/telemetry/metrics.go` exposes explicit PAD/RATE/Jitter *configured* flags and bounded jitter frame/delay counts. Stats adapter allowlists optional fields; UI distinguishes `OFF` despite `padding_configured=true` as `Ожидает подходящий поток`. These are source-code changes; they are **not yet installed** into the existing Homeserver Bridge Stats service.
- **CI provenance:** earlier opt-in rate/jitter HTTP2 batch [Go prototype checks #44278167937](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44278167937) completed **SUCCESS** and Linux build artifact run [#44278167985](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44278167985) **SUCCESS**; subsequent telemetry/UI/config extensions are undergoing another CI cycle and are **NOT PASS until checked**. On M3 branch, workflow version and artifact labels are corrected to `v1.0.1-m3-candidate` (NOT release/deployment).
- **Open:** jitter scheduler still needs performance/fairness tests with video-like chunking and short competing streams, congestion/cancellation and real multi-stream integration; monitoring UI must be installed with backup/rollback; external TLS monitoring route and live M3 rate1/Jitter must be verified on operator-approved Homeserver/Finserver deployment. **Do not publish v1.0.1 or claim M3 finished.**



#### M3 — throughput-based adaptive PAD activation target (2026-11-18)

- Operator proposes **1 Mbit/s** as the activation threshold for Padding (`Паддинг должен активироваться от 1 мбит/с по идее`), after a ~4 Mbit/s upload failed to show PAD activation. Record as the **target policy to implement and validate in M3**, not something M2 already does and not an observed PASS.
- The current M2 classifier has **no explicit bits-per-second threshold**: it classifies per-frame sustained activity and resets whenever a DATA chunk is <512 bytes. 4 Mbit/s nominal input with interleaved small frames may never activate. CI [#44276220405](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44276220405) **PASS** on code-only deterministic tests demonstrating both smooth activation and small-frame reset, not live validation.
- **M3 design gate:** calculate useful-payload throughput over a bounded time window **per stream and per direction** without inspecting payload; a ≥1,000,000 bit/s sustained rate for ~2 seconds enables consideration for PAD, with a defined hysteresis/idle reset and exclusion of short interactive bursts. PAD wire overhead remains limited to the existing 5% policy budget. Tests must cover fragmented writes, multiple streams, loss/gaps, budget, short and interactive traffic, cancellation and false activations, plus real Homeserver→Finserver uplink verification.
- Do not claim aggregate 1 Mbit/s across many unrelated TCP sessions qualifies every session; the threshold applies to one sustained stream/direction until separately validated otherwise. No production settings changed or binary deployed at this checkpoint.



#### M3 — operator observed no PAD during ~4 Mbit/s file upload (2026-11-18)

- Operator performed an actual file upload reportedly at **about 4 Mbit/s** and reports that Padding **did not activate**. Treat as **LIVE PAD ACTIVATION FAIL/NOT_OBSERVED** for the tested workload, not a blanket verdict that all traffic fails and not a proof of current code defect until packet-boundary and route evidence is collected.
- At ~4 Mbit/s, sustained eligible 32 KiB DATA frames every 64 ms should satisfy the existing M2 2-second, 64-KiB, 5-frame, <=1-s-gap classifier and emit PAD. Code-level regression tests added in `prototype/padding/policy_test.go`: `TestFourMegabitSustainedUploadActivatesPadding` and `TestFourMegabitWithSmallInterspersedFramesStaysOff`, to distinguish smooth large writes from small interleaved writes at the same approximate useful throughput.
- **Open explanations to discriminate:** (a) tested upload did not actually traverse Homeserver Bridge encoder (check `upload_bytes` delta during experiment), (b) bridge saw frequent <512-byte TCP reads causing continual burst reset, (c) each bridge stream was short/gappy despite aggregate 4 Mbit/s, (d) wrong negotiated capability/metrics source. Do **not** tune policy or declare success without determining which case is true.
- **Next low-risk check:** operator compare *existing private* `upload_bytes`, `padding_upload_frames`, `padding_upload_transitions` before and after a single known upload, without changing services. If useful bytes increase while PAD stays zero, add bounded aggregate classifier diagnostics in M3 code/tests and validate on real bridge before adjusting classification. Do not log payloads, connection identities or destinations.



#### M3 — real M2 PAD counters observed all zero on Homeserver (2026-11-18)

- Operator read **actual private Bridge JSON metrics** from `http://127.0.0.1:8895/metrics/v1`: `padding_state=OFF`, upload/download states `OFF`, upload/download PAD bytes `0`, frames `0`, transitions `0`, overhead percentages `0`. **Observed fact:** no locally accounted PAD activity since the current process started; this is not proof of a broken implementation or of a disabled configuration. Preflight had `padding_enabled=true` and M2 binary SHA verified.
- **Source-code explanation:** M2 `padding.DefaultPolicy()` activates only after **at least 2 seconds** of qualifying activity, **at least five** DATA frames each **>=512 bytes**, **at least 64KiB** of useful qualifying bytes in that streak, and gaps **<=1 second**. A small DATA frame <512 bytes or a >1-second gap resets streak; early frames are never padded. M2 client metrics measure PAD frames emitted by the **Homeserver upload encoder**, while downloads received at Homeserver are decoded; download PAD emission/telemetry is performed by the Finserver server-side encoder. Thus predominantly downstream traffic can keep all Homeserver PAD emission counters at zero.
- **Next required check:** generate an operator-controlled, naturally sustained **uplink** through the existing Bridge and compare private PAD upload bytes/frames/transitions before/after, while separately checking ordinary client connectivity. No forced synthetic traffic or production tuning without authorization. Improve M3 dashboard to distinguish configured/enabled vs active vs never-qualified/observed (do not display `Выкл.` as if user had disabled PAD). Verify Finserver-downlink telemetry independently only when deliberately enabled safely; do not infer server PAD from Homeserver zeros.
- **Status:** `M2_PAD_CONFIG_ENABLED / PAD_ACTIVITY_NOT_OBSERVED / ROOT_CAUSE_UNPROVEN`; M3 monitoring task remains open pending controlled reproduction and accurate UI status.



#### M3 — dashboard visual confirmation and remaining PAD-observability gap (2026-11-18)

- Operator supplied a screenshot of `Tervon TCP Bridge` monitoring: `Bridge готов`, 15 active sessions, 66 started / 51 completed / 0 rejected, 1.1 MiB upload / 14.5 MiB download, 0 aggregate errors, about 421 seconds uptime. This confirms a rendered dashboard with live-looking Homeserver metrics but **does not establish the observing device was in an external network or certify the HTTPS route**.
- Screenshot shows `Padding / Jitter: Выкл. / Не реализовано`, with **no PAD byte/overhead cards**. This is consistent with older deployed Bridge Stats UI (new GitHub UI assets not yet installed). The M2 `padding_state=OFF` reflects absence of currently ACTIVE qualified data streams, **not** proof that `padding_enabled` is false; local preflight previously confirmed `PADDING_CONFIG=ENABLED`. Actual PAD bytes, frames and transitions must be read from the private `/metrics/v1` source before diagnosing inactivity.
- Backend dial / connect percentiles / verified mTLS counters are measured on the Decapsulator side and can legitimately show zero on Homeserver Encapsulator. Need clear labels or conditional presentation for client-only dashboard, not fabricated values. Remaining M3 monitor work: deploy latest Stats-only UI with backup/rollback and verify counters, source freshness, public route from an external observer, and no regressions; **no M3 binary deployment yet**.

#### M3 — Homeserver telemetry link restored, operator LOCAL PASS (2026-11-18)

- Operator executed the agreed narrow monitoring recovery on active Homeserver M2, **not an M3/Jitter deployment**. Preflight and config edit: `CONFIG_PREFLIGHT=PASS`, `CONFIG_UPDATED=PASS`. Restarted only `tervon-bridge-client.service` and `tervon-bridge-stats.service`. `/etc/tervon-tcp-bridge/client.json` now uses loopback-only `metrics_listen=127.0.0.1:8895`; Stats environment `/etc/tervon/bridge-stats.env` points to `http://127.0.0.1:8895/metrics/v1`.
- **Observed local checks:** `BRIDGE_METRICS=PASS`, `STATS_API=PASS`, `MONITORING_LOCAL=PASS`, `PADDING_CONFIG=ENABLED`, `INSTALL_EXIT_CODE=0`. Operator states: `Метрики работают.` These checks validate source JSON schema/ready and local Stats dashboard API reporting `homeserver.status=ok`. They do **not** alone establish a successful external HTTPS page test or measured active PAD counters.
- **Rollback artifacts:** `/opt/tervon-tcp-bridge/backups/m3-metrics-jta0z8Y3` on Homeserver; do not delete before full acceptance. Earlier verified active M2 executable SHA-256 `64c3a7161d7d03b74ec70ea6ae2ceb97992e86d2dfa65e8662193ffc0bc7d42c`; no replacement binary in the recovery script.
- **Remaining M3-monitoring gate:** verify actual public `https://stats.xrayvless.online/bridge/` and `/bridge/api/status` **from a separate external network** against true public DNS/SNI with TLS validation, plus actual PAD display/counter activity and no subscription/Bridge regressions. Public monitoring `EXTERNAL_NOT_VERIFIED`; full M3 including Jitter `IN_PROGRESS`.



#### M3 — framing scheduler and monitoring adapter batch (2026-11-18)

- Implemented opt-in `Encoder.EnableJitter` with cancellation-aware delay bounded by per-direction Jitter policy. No `jitter_enabled` JSON setting and **no bilateral `jit1` negotiation or active data-plane integration yet**: the existing M2 production executable is unchanged, jitter OFF in production. Existing M1/M2 encoder path avoids extra timing calls when jitter is disabled.
- Fixed CI build failure in `framing/stream.go` (unused `writeStarted`) observed in [CI #44261187484](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44261187484). A later pre-monitoring code run [CI #44261987515](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44261987515) completed **PASS** for build, unit, race, M1/M2 and M3 checks. **Not** full M3 acceptance or live validation.
- Monitoring M3 implementation: `tests/homeserver/diagnose_m3_monitoring.py` emits allowlisted private-metrics and local Stats-link health flags; `test_diagnose_m3_monitoring.py` covers disabled source, linked source, confidentiality and invalid settings. `integrations/stats/bridge_metrics.py`, `bridge_panel.html` and `bridge_panel.js` now expose optional M2 PAD counters, wire-overhead percentages and directional states, without inventing values for old binaries. Python/Node tests updated. New monitoring recovery runbook: `docs/M3_MONITORING_RECOVERY.md`.
- A monitoring-mock test initially failed in [CI #44263024359](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44263024359) because the test fixture used literal backslash-n characters instead of real newline separators. Corrected in `ce38918de94baa0243f860a43f17fefc444edb56`. Latest combined CI [#44263270252](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44263270252) **PASS** (Go build, unit, race, M3 monitoring mocks, Stats Python/Node tests and existing M1/M2 regressions). This confirms code/tests only; does not establish host monitoring or active Jitter performance.
- Live monitoring remains `NOT_FIXED / HOST_DIAGNOSIS_REQUIRED`. No production changes made. Next action: operator runs read-only M3 diagnostic on existing Homeserver, then restore confirmed missing link with private backups and operator-controlled restart; verify external public site from a different network. M3 remains open.



#### M3 — verified monitoring recovery preflight (2026-11-18; NOT APPLIED)

- Operator output: `ss -H -lntp '( sport = :8895 )'` returned **no listener**; systemd `EnvironmentFiles=/etc/tervon/bridge-stats.env (ignore_errors=yes)`; live Bridge PID 2276833 executable SHA-256 `64c3a7161d7d03b74ec70ea6ae2ceb97992e86d2dfa65e8662193ffc0bc7d42c` matches M2 production SHA. This confirms free candidate port, expected Stats environment file declaration, correct running M2 binary **as of preflight only**.
- Next operator action: narrowly set `metrics_listen=127.0.0.1:8895` and `TERVON_BRIDGE_METRICS_URL=http://127.0.0.1:8895/metrics/v1`, with closed backups and automatic rollback on local failure; restart only Bridge Client and Bridge Stats units, check local telemetry/schema and Stats `homeserver.status=ok`. Expect brief interruption of existing Bridge sessions upon restart; no Nginx, 3x-ui, Finserver, or other unit change. **Not executed at this checkpoint**.

#### M3 — Homeserver read-only monitoring chain diagnosis (2026-11-18; root cause identified)

- Operator executed `tests/homeserver/diagnose_m3_monitoring.py` on existing Homeserver from the SSH checkout `/opt/tervon-tcp-bridge/m3/source`. Reported `bridge_service=ACTIVE`, `stats_service=ACTIVE`, `bridge_metrics_config=DISABLED_OR_INVALID`, `bridge_metrics_local=NOT_CHECKED`, `stats_api_local=PASS`, `stats_bridge_node=UNAVAILABLE`, `stats_source_env=NOT_CONFIGURED`, `public_site_external=NOT_TESTED`.
- **Finding:** the current client config does not contain a usable private metrics listener and the Stats service has no configured metrics source. Both sides of the telemetry linkage require installation. Existing Stats local HTTP endpoint is responsive but explicitly reports `unavailable`, as expected with no source. These are actual operator observations, not simulation; no service has been changed at this checkpoint.
- **Authorized limited recovery within M3:** after confirming a free loopback port, back up `/etc/tervon-tcp-bridge/client.json` and current Stats env state (including absence), set `metrics_listen` only on the Homeserver Bridge, set `TERVON_BRIDGE_METRICS_URL` in the Stats environment to the same loopback endpoint, restart only those two Bridge units in sequence and rollback both configs on startup or metrics/API failure. Verify active running M2 binary by SHA, keep PAD enabled, do not alter Nginx, subscriptions, Finserver or other services. Actual host fix **PENDING**. Public site check must be external.

#### M3 — operator mandates monitoring completion inside M3 (2026-11-18)

- **Operator decision:** `Продолжай. Мониторинг делай в М3, раз в М1 и М2 не смог.` Carry the previously unresolved Homeserver Bridge Stats / metrics issue into the **mandatory M3 scope**, not a future backlog. No claim that M1/M2 monitoring was fixed.
- **Code fact:** `prototype/cmd/bridge/main.go` starts its loopback-only JSON metrics HTTP listener **only** when `metrics_listen` in the existing Homeserver client configuration is non-empty. `integrations/stats/bridge_site.py` requires `TERVON_BRIDGE_METRICS_URL` to collect that listener. Earlier operator baseline showed `metrics_listen` disabled, which can explain a dashboard `unavailable` response, but current host values have not been rechecked; treat as a **diagnostic hypothesis**, not a proven root cause.
- **First diagnostic implementation:** `tests/homeserver/diagnose_m3_monitoring.py` checks the already-existing client and Bridge Stats units, reads only whether `metrics_listen` is valid loopback, tests only the declared private metrics endpoint and Stats API on `127.0.0.1:8894`, and emits only allowlisted status flags. No TLS keys, subscription tokens, config values, public IPs, or payloads printed; no changes to systemd or nginx. `test_diagnose_m3_monitoring.py` mocks both healthy and missing-source scenarios and checks no secret leak. Public `stats.xrayvless.online` must be verified **from an external network over its actual DNS**; local loopback is appropriate only for private telemetry chain testing, never as substitute for public site validation.
- **M3 monitoring acceptance gate:** actual Homeserver metrics listener active and loopback-only; current Bridge Stats API returns real `homeserver.status=ok` and current counters; page renders live data, including relevant PAD/Jitter fields; external same-domain access verified without DNS override; no regression to subscriptions, existing admin site, Nginx or working Bridge. Deployment and restart require operator-run commands with backups and recovery. **Status MONITORING=IN_PROGRESS, HOST NOT VERIFIED**.



#### M3 — first isolated policy batch CI PASS (2026-11-18)

- **Tested commit:** `53ca0b82e43c3f9550a513c53ce12cefd2e7bacf` (source and tests; no subsequent source modifications). GitHub Actions [Go prototype checks #44260427878](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44260427878): **SUCCESS**, including Go build, unit, race and existing M1/M2 regressions. Separate artifact-build [#44260427827](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44260427827): **SUCCESS**; artifact is not deployed and does not turn on jitter.
- **Verified scope:** new `prototype/jitter` policy and its cancellation/queue/budget tests build and pass in CI. The module is **NOT complete**: timing logic is not wired to active framing/HTTP2 send path, negotiation/config/telemetry and actual RTT/loss/fairness/integration checks remain. Production still runs operator-installed **M2**.
- **Next step within M3:** integrate bounded jitter with authenticated stream writing behind bilateral opt-in, ensure no delays under congestion/interactive traffic, then repeat CI/race and controlled live verification. **Status M3: POLICY CODE PASS / INTEGRATION PENDING**.

#### M3 — implementation started on operator instruction (2026-11-18; NOT PASS)

- **Branch:** `work/m3-adaptive-jitter`, branched from `work/m2-adaptive-padding`. Operator explicitly instructed: `Всё. Работаем над M3. М2 установлен и работает. Паддинг включен.` This is acceptance of **deployed M2 Padding and normal traffic**; it does not create measurement results that were not taken.
- **Unresolved M2 exception:** Bridge monitoring previously reported broken and no PAD wire metric observation was supplied. Record `MONITORING_OPEN / DEFERRED_BY_OPERATOR_TRANSITION`, not `MONITORING_PASS`. The strict original sequential completion gate therefore has an explicit outstanding exception; do not misrepresent M2 as fully verified. This exception must be resolved before final v1.0.1 release.
- **First M3 code package:** `prototype/jitter/policy.go` and `policy_test.go`. Opt-in independent per-direction burst classifier, no delay for small/interactive data or congestion, per-frame bound 4ms, per-second aggregate budget 20ms, cryptographic random duration selection, cancellation-aware wait, no timers when bypassed, no modification of user payloads. All values provisional pending validation; no active Bridge integration, config key, protocol change or server deployment in this package.
- **Completion gate unchanged:** full M3 implementation, repeatable CI/race PASS, no starvation or goroutine leaks, bounded RTT and backlog behavior, controlled operator-approved deployment, documentation and confirmed live outcome. M3 currently `IN_PROGRESS / TESTS PENDING / NOT DEPLOYED`.


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
