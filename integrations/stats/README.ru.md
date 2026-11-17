# Локальная интеграция Tervon TCP Bridge со stats.xrayvless.online

**Утверждённый объём:** только метрики Encapsulator на Homeserver.
Finserver не участвует в сборе статистики сайта; подключение к его
метрикам, relay, SSH-туннель или публичный listener **не требуются**.

**Статус:** код адаптера в GitHub, эксплуатационная установка не выполнена.
Текущий модуль разработки Bridge: M1.

## Подтверждённая архитектура сайта

- `xray-blocker-dashboard.service`: действующий override запускает
  `uvicorn unified_monitor:app --host 127.0.0.1 --port 8893 --workers 2`.
- `/opt/xray-blocker/web-dashboard/unified_monitor.py` динамически
  загружает `/opt/xray-infrastructure-monitor/infrastructure.py`
  в переменную `_infra_module`.
- `GET /infrastructure/api/infrastructure` вызывает
  `_infra_module.collect()`, возвращающую ключи
  `generated_at`, `source`, `summary`, `burst_observatory`,
  `balancers`, `outbounds`, `diagnostics`.
- Существующий контракт инфраструктуры не изменять.

## Код в GitHub

- `prototype/telemetry/prometheus.go` — опциональный
  `GET /metrics` на приватном listener Bridge; исходный JSON
  `GET /metrics/v1` сохранён.
- `integrations/stats/bridge_metrics.py` — отдельный read-only
  `BridgeMetricsCollector(endpoint)`, принимает **один** локальный
  URL `http://127.0.0.1:<port>/metrics/v1`. Запрещает удалённые
  адреса, DNS-имена, redirect и proxy; ограничивает размер ответа
  (64 KiB), время ожидания (не более 2 секунд), набор полей.
- Возвращает `tervon.bridge.dashboard.v1` с единственным ключом
  `nodes.homeserver`, статусом `ok` или `unavailable`.
  При отсутствии источника не подставляет нули и не выводит секреты.
- `integrations/stats/test_bridge_metrics.py` — изолированные
  HTTP-тесты корректного ответа, отказа, redirect, превышения
  размера, SSRF и запрета конфигурации удалённого узла.
- `.github/workflows/go-prototype.yml` запускает тесты адаптера
  при изменении `integrations/stats/**`.

## Подготовленный интерфейс (не установлен на сайте)

- `bridge_panel.html` — HTML-фрагмент карточек метрик только Homeserver;
  не содержит навигации, токенов или внешних адресов.
- `bridge_panel.js` — получает агрегаты через **same-origin**
  `GET /bridge/api/status` с `credentials: same-origin`,
  `cache: no-store`, без перенаправлений; ограничивает ожидание
  тремя секундами, обновляет раз в 15 секунд. Для вставки текста
  используется только `textContent`, не `innerHTML`.
- `bridge_panel.css` — необязательные изолированные адаптивные стили,
  не изменяющие существующие элементы сайта.
- `test_bridge_panel.cjs` — тесты Node.js без npm-зависимостей:
  корректное отображение, запрет удалённого узла, отказ API и
  отсутствие HTML-injection sinks.
- **Внимание:** путь `/bridge/api/status` пока не существует на
  работающем сайте. Не добавлять его без проверки действующего
  механизма авторизации и совместимости с `unified_monitor.py`.
  HTML-фрагмент не должен показываться пользователю до готовности
  авторизованного backend.

## План подключения (не выполнялся)

1. Проверить свободный локальный порт и согласовать
   `metrics_listen` для **Homeserver Encapsulator**. В действующей
   конфигурации это поле пока не включено.
2. Убедиться, что локальный `GET /metrics/v1` доступен только
   на Homeserver и возвращает ожидаемый JSON. Сбор не должен
   мешать передаче TCP-трафика.
3. Проверить действующую авторизацию `stats.xrayvless.online`.
   Добавить защищённый API и страницу Bridge без изменения
   существующего `/infrastructure/api/infrastructure`.
4. Выполнить изолированные тесты страницы, ошибок, тайм-аутов,
   нагрузки и отката. Только после этого согласовать изменения
   конфигурации и перезапуск действующих служб.
5. Проверять публичный HTTPS-сайт извне по его **публичному
   DNS/IP**, не подменять проверку запросом к `127.0.0.1`.

## Ограничения

- Сайт показывает статистику только локального Bridge на Homeserver.
  Трафик, состояние и ресурсы Finserver не учитываются.
- `upload_bytes` и `download_bytes` — полезные байты, а не
  полная загрузка сетевого интерфейса.
- Скорости в Prometheus рассчитываются через `rate(...[1m])`.
  JSON-скорости, зависящие от частоты Snapshot, не используются
  в адаптере.
- Padding/Jitter до M2/M3 — `NOT_IMPLEMENTED`.
- Адаптер не регистрирует FastAPI routes и не меняет работающие
  службы. Вопросы авторизации и установки сайта ещё не закрыты.
