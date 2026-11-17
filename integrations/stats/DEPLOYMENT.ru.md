# План безопасной установки Bridge Stats на Homeserver

## Актуальные решения и результат установки (2026-11-17)

- **Доступ:** оператор выбрал публичную страницу `/bridge/` **без Basic Auth**. Ранее описанные ниже проверки `401/200` и копирование `auth_basic_user_file` **устарели для Bridge**; действующие `/admin/` и `/admin/api/` остаются защищёнными и не меняются.
- **Фактический первый запуск:** `ADMIN_HTPASSWD=PASS`, но мгновенный запрос к 127.0.0.1:8894 дал `ConnectionRefusedError`, затем `INSTALL_STATUS=ROLLBACK`. Журнал systemd показал `Started`, сразу `Stopping` и `Stopped`, без traceback приложения. Файл unit и каталог app удалены, служба inactive, порт свободен. Причина вероятно race с Uvicorn, но не доказана.
- **Исправление:** `wait_for_site.py` делает ограниченное ожидание до 12 секунд с повторными запросами **только** при `ConnectionRefusedError`, проверяет active unit и JSON schema/единственный Homeserver/`unavailable`; при других ошибках и истечении срока блокирует установку и откатывает только новую службу. После CI PASS повторить установку, зафиксировать `NEW_SERVICE=PASS` и локальный listener.
- **Nginx:** применять только после подтверждения `NEW_SERVICE=PASS` и отдельного аудита; обновлённый `nginx-bridge.conf.example` имеет `auth_basic off` **только внутри новых /bridge locations**. Сохранить существующую авторизацию `/admin/`. Проверять внешние 200 без credentials, 404 для сырого `/metrics/v1`, а также отсутствие регрессий действующих страниц. Не раскрывать конфиденциальные показатели, IP/идентификаторы клиентов, секреты.


**СТАТУС: оператор разрешил подключение мониторинга 2026-11-17. Установка выполняется поэтапно, с проверками и откатом; команды не выполнялись на Homeserver автоматически.**
Цель: отдельная защищённая страница `/bridge/` на существующем
`stats.xrayvless.online`. Только локальный Homeserver Encapsulator.
Не изменять `unified_monitor.py`, `infrastructure.py`,
`xray-blocker-dashboard.service`, существующие `/admin/` и
`/infrastructure/`. Не подключаться к Finserver.

## Известные исходные факты (read-only, 2026-11-17)

- nginx `/admin/` и `/admin/api/`: Basic Auth, upstream 8000.
- nginx `/infrastructure/`: upstream 8893, отдельной auth-директивы
  в проверенном location нет.
- 8894: `ss -H -ltn '( sport = :8894 )'` не показал слушателя.
- Существующая Python venv: FastAPI 0.139.0, Uvicorn 0.50.2.
- У работающего Bridge ещё нет включённого локального
  `metrics_listen`. До его включения API должен возвращать
  `status=unavailable`, а не выдуманные метрики.

## Этап 0. Подтверждение предпосылок — без изменений

1. Проверить текущий commit ветки `work/m1-framing`, CI PASS,
   отсутствие изменений и секретов в `integrations/stats/`.
   Для git использовать только SSH remote
   `git@github.com:tervon-software/Tervon-TCP-Bridge.git`.
2. Проверить, что нет существующей службы
   `tervon-bridge-stats.service`, порта 8894, занятого каталога
   `/opt/tervon/bridge-stats` или конфликтующих nginx locations.
3. Проверить системный `systemd` и его поддержку
   `IPAddressDeny`/`IPAddressAllow`; не ослаблять sandbox
   при ошибке, а фиксировать блокер.
4. Проверить доступность существующей venv и каталога приложения
   для `DynamicUser=yes`; не менять разрешения действующей venv.
5. Проверить реальный `auth_basic_user_file` в `/admin/`
   **локально**, не публикуя путь или содержимое в GitHub/чат.
6. Сделать резервную копию изменяемого nginx server config в
   согласованном постоянном каталоге с ограниченными правами.
   Записать sha256 и команду отката.

### Автоматизированный read-only preflight

Из чистого SSH checkout ветки `work/m1-framing` на Homeserver
запустить `python3 integrations/stats/preflight_homeserver.py`.
Скрипт не пишет на диск и не раскрывает путь/содержимое
`auth_basic_user_file`. Любой `BLOCKED` — остановка
установки и фиксация причины. Строка `DYNAMIC_USER_ACCESS: CHECK`
остаётся отдельным обязательным gate, даже при коде выхода 0.
Точный SHA и результат preflight зафиксировать в журнале.

## Этап 1. Изолированное приложение — разрешён оператором

**Исполняемый этап (отдельный процесс bash, НЕ `source`):**

```bash
cd /opt/tervon/bridge-m0-check/build/repo
# Только если чистый SSH checkout work/m1-framing:
git fetch origin work/m1-framing
git merge --ff-only FETCH_HEAD
sudo bash integrations/stats/install-site-service.sh
```

Команды не содержат `exit` для интерактивной оболочки; при ошибке
отдельный скрипт возвращает ненулевой код и выполняет rollback
**только новой службы**. Перед установкой скрипт запускает
`preflight_homeserver.py`, не создаёт `bridge-stats.env`,
не включает локальную метрику Bridge и требует `unavailable`
в новом API. Проверяет, что порт 8894 слушает только
`127.0.0.1`. Он **не меняет nginx**. Не выполнять при
`BLOCKED` и не игнорировать `DYNAMIC_USER_ACCESS: CHECK`
без оценки доступа: это дополнительная проверка при первом
запуске под DynamicUser.

**Важное:** `sudo bash` запускает скрипт отдельным процессом,
поэтому его `exit` не закрывает интерактивную SSH-сессию.
После `NEW_SERVICE=PASS` проверить внешний шаг nginx отдельно.



1. Установить **только** файлы `bridge_site.py`,
   `bridge_metrics.py`, `bridge_panel.html`, `bridge_panel.js`,
   `bridge_panel.css` в отдельный read-only каталог
   `/opt/tervon/bridge-stats`. Не ставить новые Python пакеты.
2. Подготовить `tervon-bridge-stats.service` из шаблона.
   До включения выполнить `systemd-analyze verify` и проверить
   параметры безопасности. Убедиться, что bind только 127.0.0.1:8894.
3. Протестировать локально `/bridge/` и
   `/bridge/api/status`; статус API при неактивной метрике
   `unavailable`. Никакой доступности извне на этом этапе.
4. Если приложение не стартует, откатить только новую службу;
   не менять работающий dashboard или nginx.

## Этап 2. Nginx — отдельное разрешение и проверка

1. Добавить ровно два новых location из
   `nginx-bridge.conf.example` внутрь **действующего HTTPS server**
   `stats.xrayvless.online`; заменить placeholder реальным
   локальным путём htpasswd из `/admin/`.
2. И точный `/bridge`, и `^~ /bridge/` должны требовать Basic Auth.
   Точный `/bridge` проксируется к `/bridge/` после авторизации;
   не использовать nginx `return 30x` до auth phase.
   Не добавлять общих `auth_basic` на другие страницы.
3. Выполнить `nginx -t`; при FAIL ничего не перезагружать,
   восстановить конфигурацию. При PASS согласовать `reload`.
4. Проверить с **внешней сети** реальный публичный DNS, TLS/SNI:
   `/bridge/` и `/bridge/api/status` без credentials → 401,
   с разрешёнными credentials → 200; проверить
   `/bridge`, JS/CSS и обходы альтернативных путей.
   Никогда не подменять публичный DNS loopback при внешнем тесте.
5. Убедиться, что прежние `/admin/` и `/infrastructure/`
   работают без изменений. При регрессии — немедленный откат
   только нового nginx location и новой службы.

## Этап 3. Метрики Bridge — отдельно после подтверждения M1

1. Согласовать безопасное включение `metrics_listen` только
   на Homeserver, на отдельном приватном loopback-порту.
   Перед любым перезапуском действующего Bridge иметь резервную
   конфигурацию, процедуру отката и контроль TCP-соединений.
2. Проверить JSON `tervon.bridge.telemetry.v1` и отсутствие
   клиентских данных. Не включать метрики Finserver.
3. Установить `TERVON_BRIDGE_METRICS_URL` локально в конфигурации
   **новой** stats-службы и перезапустить только её.
4. Сверить отображаемые счётчики с локальным источником,
   ошибки и деградацию, затем провести внешний тест авторизации.

## Критерии PASS

- Python/JS/Go CI PASS на точном SHA, изолированный ASGI PASS.
- Порт 8894 слушает только loopback; новый процесс ограничен
  sandbox, остальные службы не затронуты.
- Нельзя получить страницу или API без Basic Auth; внешний 401/200
  подтверждён на настоящем публичном DNS.
- Существующий инфраструктурный API и страницы без регрессий.
- Пока метрики Bridge не активированы — `unavailable`.
  После активации — реальные агрегаты только Homeserver.
- Зафиксированы время, commit, команды, результаты и откат.
  **До этого код не считается развёрнутым или выпущенным.**
