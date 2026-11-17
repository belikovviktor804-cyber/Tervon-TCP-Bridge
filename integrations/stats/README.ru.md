# Интеграция Tervon TCP Bridge со stats.xrayvless.online

Статус: **код адаптера подготовлен; подключение к рабочему сайту НЕ выполнено**.
Текущий модуль разработки Bridge: M1. До проверки CI, источников данных,
авторизации сайта и плана отката запрещено объявлять интеграцию завершённой.

## Подтверждённая архитектура сайта (Homeserver, read-only)

- `xray-blocker-dashboard.service` (действующий systemd override):
  `uvicorn unified_monitor:app --host 127.0.0.1 --port 8893 --workers 2`.
- `/opt/xray-blocker/web-dashboard/unified_monitor.py` динамически загружает
  `/opt/xray-infrastructure-monitor/infrastructure.py` как `_infra_module`.
- `GET /infrastructure/api/infrastructure` вызывает `_infra_module.collect()`.
- `collect()` возвращает семь ключей:
  `generated_at`, `source`, `summary`, `burst_observatory`,
  `balancers`, `outbounds`, `diagnostics`.
- Публичный nginx, действующие страницы и прочие API не менять без отдельного
  утверждённого плана. Не редактировать `app.py` или каталоги `backup-*`.

## Реализовано в этом репозитории

- `prototype/telemetry/prometheus.go`: `GET /metrics` для Prometheus
  на существующем **только loopback** listener; исходный JSON
  `GET /metrics/v1` сохранён.
- `integrations/stats/bridge_metrics.py`: независимый Python-адаптер
  `BridgeMetricsCollector` для JSON `/metrics/v1`.
  Разрешены только буквальные loopback IP, отключены прокси и HTTP redirect,
  ограничены размер ответа (64 KiB) и timeout (до 2 секунд).
  На выходе только проверенные агрегаты, без IP, токенов, идентификаторов,
  содержимого соединений или текста ошибок от удалённого узла.
  Неуспешный опрос узла возвращает `status=unavailable` и не прерывает
  сбор другого узла. Никакие FastAPI routes не регистрируются.
- `integrations/stats/test_bridge_metrics.py`: изолированные HTTP-тесты
  двух узлов, недоступности, некорректного JSON, redirect, размера и SSRF.
- GitHub Actions запускает эти Python-тесты и Go-проверки.

## Предполагаемое подключение (ещё НЕ РАЗВЁРНУТО)

1. Homeserver Bridge с разрешённым `metrics_listen` отдаёт
   `127.0.0.1:<port>/metrics/v1` и `/metrics`.
   Пока поле не настроено в действующей конфигурации, endpoint недоступен.
2. Finserver Bridge имеет собственный локальный listener. Его статистику
   разрешено читать на Homeserver **только** после настройки проверенного
   защищённого канала с локальным endpoint на Homeserver. Не публиковать
   Finserver metrics в интернет и не создавать неаутентифицированный relay.
3. Отдельный серверный сборщик опрашивает оба локальных endpoint и
   возвращает `tervon.bridge.dashboard.v1` с раздельными `homeserver`
   и `finserver`. Нет источника — показывать `unavailable`, не нули.
4. После аудита контроля доступа сайта добавить отдельную защищённую
   страницу Bridge и backend API. Не делать новый публичный endpoint,
   пока не проверена серверная авторизация. Вёрстка обязана экранировать
   текстовые значения и не раскрывать внутренние адреса.
5. Проверить в изоляции тесты, схему, отказ одного узла, timeout,
   нагрузку и rollback. Только затем согласовать установку и
   перезапуск работающих служб, отдельно проверив HTTPS с внешнего
   узла через **публичный DNS/IP**, не `127.0.0.1`.

## Ограничения

- Prometheus `rate(tervon_bridge_upload_bytes_total[1m])` и
  `rate(tervon_bridge_download_bytes_total[1m])` — полезные байты/с,
  не физический трафик интерфейса.
- JSON `upload_bytes_per_second` зависит от интервала вызовов Snapshot
  в Bridge, поэтому в адаптере намеренно не используется.
- Padding/Jitter до M2/M3: `NOT_IMPLEMENTED`, не `ACTIVE`.
- Адаптер не является доставкой данных с Finserver, не является
  полноценным сервисом и не включает авторизацию сайта.
- Реальные пути и конфигурация Finserver, действующий механизм
  авторизации страницы и полная реализация текущего infrastructure
  collector пока не проверены. Эти проверки обязательны перед patch.
