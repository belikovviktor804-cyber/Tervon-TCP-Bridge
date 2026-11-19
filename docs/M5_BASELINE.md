# M5.1 — Linux baseline и протокол сравнения v1.0.0 → v1.0.1

**Дата:** 2026-11-19  
**Ветка:** `work/m5-linux-release`  
**Начало M5:** M4 closure `eb4fde88d6477defb58f8237fd0526c292e93cfe`  
**Статус:** `IN_PROGRESS / RELEASE_ARTIFACT_IDENTITY_PASS`; никаких новых benchmark PASS этот документ не заявляет.  
**Порядок:** `docs/DEVELOPMENT_PLAN.md` → `docs/M5_ACCEPTANCE.md` → этот baseline.

## 1. Подтверждённые исходные артефакты

| Объект | Происхождение | SHA-256 / commit | Уровень доказательства |
| --- | --- | --- | --- |
| Старый сохранённый Linux-бинарник, документированный в `docs/RELEASE.ru.md` и `docs/INSTALL-BINARY.ru.md` | Homeserver, ранее проверенный экземпляр | `5c813bae9cb0c68e3d5ac1e9afb0556f13cb270b369a86ae8ffc0d20adb086e2` | Исторический checksum, **не** повторно проверенный на хосте в M5 |
| VCS metadata старого бинарника | `docs/RELEASE.ru.md` | `83c98482048d130041e89f545db5885ade090f2b`; `vcs.modified=true` | **Нельзя** считать воспроизводимой сборкой из чистого commit |
| Сохранённый M3 rollback-бинарник | обе стороны Bridge | `df3bcdfc70dfa8f238af454cd9fa1e852188b61233799972fd38bde48c8aa499` | M4 prepare и два хостовых SHA PASS; rollback fault injection отдельно не подтверждён |
| Действующий Linux M4 candidate на двух узлах | исходники `8d0aa3777c209d2825a1c11dfbae9487893cb7f5` | `5035217a89476117894efe1a44db5dcdd1242773d7fd9461170f2cf6ccb98a90` | stage + install/process SHA PASS, операторский VPN/Stats PASS, **не** performance benchmark |
| M5 branch base | завершённый M4 | `eb4fde88d6477defb58f8237fd0526c292e93cfe` | GitHub provenance |

**Важное различие:** известная ссылка на [GitHub Release v1.0.0](https://github.com/tervon-software/Tervon-TCP-Bridge/releases/tag/v1.0.0) присутствует в README, и **содержимое бинарного asset теперь проверено оператором по SHA256/размеру** (см. раздел 8); target commit тега отдельно не установлен. Сравнение со старой версией допускается только после сверки точных входных артефактов и их режима запуска.

## 2. Подтверждённый тракт / границы

Используется существующий маршрут **главный сервер каскада: Encapsulator → защищённый HTTPS/2 / TLS1.3 / mTLS TCP Bridge → зарубежный Decapsulator → Xray/backend**. Подтверждено оператором использование реального VPN TCP и Stats при M4↔M4. Не публиковать IP, DNS, UUID, ключи, клиентские идентификаторы и приватные конфигурации.

Подтверждённое по предыдущим протоколам: TLS1.3/h2/mTLS, двусторонний TCP, счётчики сессий/полезных байтов, работа Padding/Jitter на живом M3 (перед M4). Не подтверждено количественно: per-frame Jitter, window cap, RTT, fairness, длинная ресурсная нагрузка. Hysteria2/UDP/QUIC **за пределами** сравнения TCP Bridge.

## 3. Сопоставимые сценарии

| Класс | Что сравниваем | Обязательная защита от ложных выводов |
| --- | --- | --- |
| Короткие и интерактивные | установление TCP, время отклика, успешность, overhead | отдельно от длинных потоков; Padding/Jitter должны оставаться OFF там, где неактивны |
| Длительный ровный поток | throughput полезных байтов, PAD/Jitter активация, RTT, CPU/RSS | одинаковые размер/длительность/ограничения и направления |
| Порционный поток | чередование burst/idle, hysteresis, отклик после паузы | не считать видео ровной передачей |
| Скачивание | download Mbps, успешные байты, overhead, stall | сравнивать тот же backend/маршрут, без параллельных неконтролируемых нагрузок |
| Конкурентные потоки | fairness, короткий запрос на фоне загрузки, starvation | фиксированные числа потоков, отдельные направления и лимиты |
| Отказ/восстановление | cancel/EOF/half-close/retry, offline сетевой разрыв | изолированный стенд; не отключать рабочий сервер или пользователей ради теста |

Для v1.0.0 и v1.0.1 сравнивать **идентичный тестовый маршрут, конфигурацию, полезный payload, количество потоков, окно наблюдения и сетевой профиль**. Нельзя использовать прод-замеры разных дат как строго сопоставимый baseline без фиксации внешних условий. Старую версию не разворачивать поверх работающей M4 ради сравнения.

## 4. Измерители и пороги

1. **Правильность:** byte-exact full duplex и half-close, без повреждения/утечки payload; на одинаковом тестовом наборе ошибок и регрессий не больше базовой версии. Неподтверждённые протоколы не перечислять как протестированные.
2. **Jitter:** по ранее утверждённым ограничениям ≤4 мс на отдельный кадр и ≤20 мс в одном фиксированном секундном окне. **Агрегированные counters не доказывают эти максимумы**; нужна инструментированная per-frame/window проверка на контролируемой версии без публикации содержимого потоков.
3. **Fairness:** отсутствие starvation у конкурентных сессий, отдельная оценка коротких запросов на фоне длительной передачи. Численный допустимый прирост RTT/latency **пока не утверждён**; не придумывать порог PASS.
4. **Ресурсы:** CPU, RSS, активные Go goroutines, sessions opened/closed, деградация на фиксированной длительности/параллелизме; фиксировать baseline и значения, не смешивать application payload с фактическим wire overhead.
5. **Сетевая среда:** RTT и loss указывать с источником/слоем измерения, timestamp и направлением. Не выводить packet loss из счётчиков HTTP2 payload.
6. **Релиз:** фиксированный исходный commit, Go version, build flags, `BUILD_INFO`, binary SHA256, `SHA256SUMS`, совместимость JSON, проверка без секретов, подписанная/проверяемая provenance когда доступна. Не объявлять выпуск до M5.6.

## 5. Незакрытые вопросы M5.1

- **RELEASE_ASSET_IDENTITY_PASS (operator, 2026-11-19):** опубликованный GitHub Release v1.0.0 asset скачан в изолированный каталог Homeserver, совпали размер и SHA256 со старым локальным образцом. **Точная привязка tag к commit отдельно не проверялась**; воспроизводимость dirty VCS binary не утверждается.
- **BENCHMARK_BASELINE_NOT_MEASURED:** нет сопоставимого, воспроизводимого v1.0.0↔v1.0.1 замера на идентичном тестовом пути. Нельзя называть текущий пользовательский VPN опыт сравнительным benchmark.
- **THRESHOLD_DECISION_PENDING:** численные допустимые регрессии RTT/throughput/CPU/RSS и длительность измерительного окна не утверждены. Jitter limits 4ms/frame и 20ms/second являются исходными release-blocking требованиями.
- **M5.1 IN_PROGRESS:** опубликованный v1.0.0 asset и пять тестов проверки происхождения теперь подтверждены оператором как PASS. Остаются условия сопоставимого baseline и решения по порогам производительности; следующий шаг внутри M5.1.

## 6. Безопасность и отчётность

Источник: GitHub API и уже записанные операторские результаты M3/M4. Команды на Homeserver/Finserver **не выполнялись**, работающие Bridge/Stats/VPN/nginx службы **не изменялись**. Фиксировать результаты каждый раз как `SOURCE`, `CI`, `ISOLATED_HOST`, `LIVE_OPERATOR` или `RELEASE`, а не объединять в недоказанный общий PASS. M6 — `HOLD / DISCUSSION_REQUIRED`.

## 7. Начало фактических испытаний M5.1 — 2026-11-19

- GitHub ref `v1.0.0` **существует**: прочитан `README.md` с этого ref. Это подтверждает наличие Git ref, но **не** выпуск/содержимое assets и не их checksum.
- Создан read-only `tests/m5/verify_release_baseline.py`: проверяет public release JSON tag/name/size/path и хеш **локально скачанного** бинарника, сопоставляет с историческим SHA. Источник JSON — публичный GitHub API или сохранённый JSON, ни ключей, ни production конфигураций не требуется. Проверка **не** выполняет сетевые/серверные mutation.
- Добавлены `tests/m5/test_verify_release_baseline.py` с синтетическими отрицательными cases (неверный tag, чужой asset URL, дублирующий asset, неверные размер/хеш), и `.github/workflows/m5-baseline.yml` для синтаксиса и offline unittest в Actions.
- **SOURCE COMMITTED / CI RESULT NOT YET VERIFIED / REAL RELEASE ASSET NOT YET CHECKED.** Ни один synthetic test не объявлять PASS до просмотра результатов CI. Это только M5.1; **M5.2 и M5.3 не начаты**.
- Верификация реального опубликованного артефакта: загрузить release binary в отдельный каталог (не использовать live binary), далее `python3 tests/m5/verify_release_baseline.py --artifact /path/to/tervon-bridge-linux-amd64`; файл release metadata загружается read-only из GitHub API. Если исторический SHA не совпадает, вывод `ARTIFACT_DIFFERS_FROM_HISTORICAL` означает разные артефакты и требует уточнения baseline, **не** доказательство поломки VPN.

## 8. Operator Homeserver v1.0.0 release-artifact verification — PASS (2026-11-19)

- Operator cloned `work/m5-linux-release` using `git@github.com:tervon-software/Tervon-TCP-Bridge.git` into isolated `/opt/tervon-tcp-bridge/m5/source`; Git remote reported complete clone.
- Downloaded `https://github.com/tervon-software/Tervon-TCP-Bridge/releases/download/v1.0.0/tervon-bridge-linux-amd64` into isolated `/opt/tervon-tcp-bridge/m5/artifacts/` with successful curl transfer (~6.44 MiB).
- Ran `python3 tests/m5/verify_release_baseline.py --artifact /opt/tervon-tcp-bridge/m5/artifacts/tervon-bridge-linux-amd64` using public release metadata and isolated artifact; operator supplied JSON:
  - `status=PASS_ARTIFACT_MATCHES_HISTORICAL`
  - `release_tag=v1.0.0`, `asset_name=tervon-bridge-linux-amd64`
  - `asset_size_matches=true`, `historical_sha_matches=true`
  - `sha256=5c813bae9cb0c68e3d5ac1e9afb0556f13cb270b369a86ae8ffc0d20adb086e2`.
- **M5.1 RELEASE_ARTIFACT_IDENTITY=OPERATOR_PASS**: public GitHub asset is byte-identical to historical SHA. No claim of reproducible build: old binary still has `vcs.modified=true` metadata. Exact tag commit target was not separately verified by this script.
- **M5.1 overall remains IN_PROGRESS:** offline CI result and baseline comparator conditions/RTT+throughput+resources acceptance thresholds not yet confirmed. No production unit restart; no controlled throughput/RTT tests were performed. Do not mistake artifact verification for full M5 PASS. M6 HOLD.

## 9. Operator offline verifier tests — PASS (2026-11-19)

- Environment: Homeserver isolated checkout `/opt/tervon-tcp-bridge/m5/source` on branch `work/m5-linux-release`, pulled by SSH fast-forward from `e9a96b3` to exact tested commit `4b0a7ac4c218755af3a18c303376faf8a740adf8`.
- Command: `python3 -m unittest discover -s tests/m5 -p 'test_*.py' -v`.
- Operator output: 5 named tests OK — different asset size, mismatched historical SHA, duplicate asset, malformed tag/external URL, valid asset selection. `Ran 5 tests in 0.001s`, `OK`, `M5_TEST_EXIT_CODE=0`.
- **M5.1 OFFLINE_VERIFIER_TESTS=HOMESERVER_PASS (5/5)**, distinct from pending GitHub Actions CI result. Prior public v1.0.0 asset identity test remains `OPERATOR_PASS`. These tests do **not** measure TCP performance, Jitter limits, fairness or CPU/RSS.
- No live Bridge, Stats, Xray or other service touched. **M5.1 overall IN_PROGRESS** pending comparability/threshold decisions; M5.2–M5.6 NOT_STARTED; M6 HOLD.

## 10. Сводная пакетная проверка M5.1 — OPERATOR PASS

Исходный commit `0909812d09dd149f872b0eee9e8e48bd85374530`; запуск `2026-11-19T09:13:58Z` на Homeserver; `M5_1_BATCH=PASS_PROVENANCE_AND_STATIC_BASELINE`, exit 0. Проверки SHA старого опубликованного бинарника, наличия объекта M4, документов, ref `v1.0.0`, примеров JSON — все PASS. Git ref tag SHA `1f104008437ba9699a54404c6da1af4acb81f651`; binary Go 1.25.5. Лог `/opt/tervon-tcp-bridge/m5/results/m5.1-2EO4eslf/report.log`. **Это provenance/static PASS, не сравнительный benchmark.**
