# M3 — восстановление мониторинга Tervon TCP Bridge

**Статус:** подготовлена диагностика; **исправление на Homeserver ещё не применено**. Это обязательный результат M3 по решению оператора от 2026-11-18. Авторитетный статус — `docs/DEVELOPMENT_PLAN.md`.

## Цепочка данных

`tervon-bridge-client.service` (существующий M2, конфигурация `/etc/tervon-tcp-bridge/client.json`) → опциональный локальный `metrics_listen` → `GET /metrics/v1` → `tervon-bridge-stats.service`, `TERVON_BRIDGE_METRICS_URL` из `/etc/tervon/bridge-stats.env` → локальный сайт `127.0.0.1:8894/bridge/api/status` → существующий nginx → публичный `https://stats.xrayvless.online/bridge/`.

Последнее наблюдение оператора до M3: Bridge Stats работает, но метрики на странице отсутствуют. Более ранний снимок конфигурации указывает на **отключённый** `metrics_listen`; текущее состояние на сервере ещё не считано. Нельзя считать причину доказанной до диагностики.

## Этап A — read-only Homeserver

Клонирование исходников только по SSH в отдельный каталог; работающий бинарник и конфигурацию **не изменять**:

```bash
git clone --branch work/m3-adaptive-jitter \
  git@github.com:tervon-software/Tervon-TCP-Bridge.git \
  /opt/tervon-tcp-bridge/m3/source

python3 /opt/tervon-tcp-bridge/m3/source/tests/homeserver/diagnose_m3_monitoring.py
```

Если отдельный checkout уже существует, не клонировать повторно; после проверки статуса рабочей копии безопасно обновить из GitHub по SSH.

Диагностический JSON выводит **только** статус двух заданных служб, факт настройки loopback `metrics_listen`, доступность личного `/metrics/v1`, признак совпадения endpoint из `/etc/tervon/bridge-stats.env` и ответ локального Bridge Stats API. Он **не** выводит адреса соединений, IP клиентов, сертификаты, содержимое конфигураций или счётчики клиента. Ничего не перезапускает, не открывает порты и не записывает.

**Интерпретация:** `bridge_metrics_config=DISABLED_OR_INVALID` — источник не задан либо некорректен; `bridge_metrics_local=UNAVAILABLE` при `ENABLED_LOOPBACK` — проверить listener/журнал Bridge; `stats_source_env=NOT_CONFIGURED` или не `MATCH` — проверить URL существующей Stats-службы; `stats_bridge_node=UNAVAILABLE` при локальном `bridge_metrics_local=PASS` — проблема на стороне Stats-сборщика/его env или работающей версии приложения. Не включать новый порт без проверки конфликта.

## Этап B — восстановление без переписывания инфраструктуры

После подтверждения причины подготовить узкий операторский патч **только** к двум уже существующим компонентам Bridge: поле `metrics_listen` на Homeserver, если оно отключено, и `TERVON_BRIDGE_METRICS_URL` только в файле окружения Bridge Stats, если отсутствует/не совпадает. Выбирать и проверять свободный loopback-порт отдельно; не считать `8895` гарантированно свободным. Сохранять закрытые резервные копии, SHA исполняемого файла M2 и прежние JSON/окружение; откатить эти изменения при ошибке рестарта или local API. Не редактировать общую службу инфраструктуры, nginx, Xray, Guard, подписку `:2096` или Finserver.

Обновить только отдельные файлы Bridge Stats из ветки M3 для отображения PAD и тестировать их отдельно. Не публиковать закрытый `/metrics/v1` в интернет. При сомнении в безопасности отката — блокировать установку и запросить минимальные фактические данные.

## Этап C — подтверждение

1. Homeserver: существующие Bridge/Stats units `active`, `/metrics/v1` доступен **только на выбранном loopback listener**, схема `tervon.bridge.telemetry.v1`, счётчики реальны и растут при обычной работе; Stats API `tervon.bridge.dashboard.v1`, `nodes.homeserver.status=ok`.
2. **Внешняя сеть**, например Finserver или пользовательский ПК: запросить **фактическое** `https://stats.xrayvless.online/bridge/` и `/bridge/api/status` по DNS-имени без `--resolve`, `-k`, VPN DNS override и localhost подмены. Проверить HTTPS-сертификат, статус и отображение, по разрешённой модели доступа.
3. Оператор подтверждает, что Bridge/VPN продолжает работать, подписки и другие службы не затронуты. Зафиксировать отдельные результаты: `MONITORING_CODE_CI_PASS`, `MONITORING_LOCAL_PASS`, `MONITORING_EXTERNAL_PASS`. Только после всех трёх — `MONITORING_PASS`.

## Состояния

- Подготовленная проверка и CI — не равны исправлению на Homeserver.
- Публичный HTTPS-test — не равен закрытому loopback-test для источника метрик; оба необходимы.
- М3 в целом не может считаться PASS до отдельно проверенных Jitter, его Fairness/RTT/очередей, мониторинга и контролируемого live результата.
