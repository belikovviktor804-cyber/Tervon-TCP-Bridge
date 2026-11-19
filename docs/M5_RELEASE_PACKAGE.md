# M5 — актуальный релизный пакет (обновление после двухузлового cutover PASS)

**Дата:** 2026-11-19. Оператор подтвердил запуск на обоих серверах M5 с нулевой квотой сессий: `M5_DUAL_RESULT=PASS_TWO_NODE_CUTOVER`, отдельные SHA PASS, `max_sessions=0` на обоих узлах, public telemetry PASS. **GitHub Release v1.0.1 официально опубликован 2026-11-19 — RELEASED/PASS.**

- Исходный код: `96c136702d10b6fb9a16aa2893f6b978e8b14e86`.
- Проверенный новый бинарный SHA256: `4f997b60bd777516339512ac71ddbb89b96d5bf9af979286ff3bd63690ea525b`.
- Подготовленный и установленный кандидат: `/opt/tervon-tcp-bridge/m5/unbounded/stage-qSxWhELI/dist/tervon-bridge-linux-amd64`.
- Release staging `/opt/tervon-tcp-bridge/m5/release-no-quota/` — **операторский PASS 2026-11-19** (`RELEASE_PREP=PASS_VERIFIED_DEPLOYED_M5`, exit 0). Повторный запуск откажется перезаписывать пакет. Не требовал сборки или перезапуска служб.
- Публиковать ТОЛЬКО новый M5 бинарник; прежний M4 SHA `5035217a89476117894efe1a44db5dcdd1242773d7fd9461170f2cf6ccb98a90` из `/opt/tervon-tcp-bridge/m5/release/` устарел и не используется.

## Подготовка новых assets (без изменения работающей системы)

```bash
cd /opt/tervon-tcp-bridge/m5/source || exit 1
git pull --ff-only origin work/m5-linux-release
bash install/prepare-release-assets.sh
cd /opt/tervon-tcp-bridge/m5/release-no-quota || exit 1
sha256sum -c SHA256SUMS
cat BUILD_INFO
```

При PASS создаются `tervon-bridge-linux-amd64`, `SHA256SUMS`, `BUILD_INFO`. Скрипт проверяет SHA и отказывается перезаписывать уже существующий новый пакет. GitHub Release не создаёт. Установщик `install/install-bridge.sh` теперь использует новый SHA и примеры конфигурации `max_local_connections=0`, `max_streams=0`. Не запускать его поверх действующего Bridge.

## Публикация

Исторический пакетный отчёт фиксировал `PUBLICATION_DONE=NO` **до** публикации. После него GitHub API подтвердил опубликованный `v1.0.1`, см. итог ниже. Инсталляционный workflow успешно проверен: GitHub Actions run 44528805350 = SUCCESS (новый SHA, нулевые лимиты в шаблонах, отрицательная проверка offline asset, Bash syntax). Для официального релиза проверить GitHub Actions `m5-installer.yml` и другие необходимые release gates, исходную лицензию/условия распространения, а также окончательный новый пакет. После этого загрузить три assets в GitHub Release `v1.0.1` через GitHub API; не публиковать бинарник с другим SHA или закрытые конфигурации/ключи. Документировать реальный Git tag SHA отдельно от исходного коммита сборки. Публикация **не подтверждена**.

## Исторический статус

До исправления М4 использовал жёсткую квоту 100 сессий. Пользовательский 180-секундный тест обнаружил 313 отклонённых подключений на Homeserver, несмотря на 0 ошибок транспорта. Оператор потребовал снять лимит; код исправлен, ветка протестирована (vet/unit/race/build PASS), а затем новый бинарник установлен одновременно на обоих узлах (LIVE SWITCH PASS, `max_sessions=0`). Новых post-cutover 180-секундных нагрузочных измерений пока не проводили; не приписывать старой нагрузке свойства нового экземпляра.

## Проверенные и незавершённые условия выпуска (2026-11-19)

- Installer CI: PASS, https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44528805350
- Cutover offline CI: PASS, https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44524847060
- M5 dual-node live deployment and release asset preparation: operator PASS, SHA `4f997b60bd777516339512ac71ddbb89b96d5bf9af979286ff3bd63690ea525b`
- Passive post-cutover 30-second observation: tool committed, host execution pending; cannot claim post-upgrade real-user TCP throughput from staging
- Quantitative real-network RTT / per-stream fairness and actual wallclock Jitter maxima: NOT_MEASURED; M5.2+M5.3 final gate not marked full PASS
- LICENSE absent at repository root. No license or public redistribution permission silently invented; require operator decision
- Official GitHub Release v1.0.1: NOT_RELEASED; existing v1.0.0 only. Do not publish until approved gates satisfied

## Авторизация официального выпуска и фактическая публикация

**Оператор 2026-11-19 прямо поручил опубликовать Linux v1.0.1** и привести публичные README/инструкции в порядок. Это разрешение на публикацию проверенного бинарника и документации, **не** заявление о PASS ранее неизмеренных сетевых RTT/per-stream fairness/live wall-clock maxima. Ограничения явно отражены в публичных release notes. Ни одна open-source лицензия не назначена: файл LICENSE отсутствует, репозиторий доступен публично, но права на копирование/модификацию кода не предполагаются автоматически.

- Публичный README в `main` обновлён, убраны ссылки на внутренний AI/development plan, добавлено клонирование `git@github.com:tervon-software/Tervon-TCP-Bridge.git`.
- Обновлены `SECURITY.md`, `docs/CONFIGURATION.ru.md`, `INSTALL-v1.0.1.ru.md`, `INSTALL-BINARY.ru.md`, `RELEASE.ru.md`, `RELEASE_NOTES_v1.0.1.ru.md`, `INSTALL.ru.md`, `TROUBLESHOOTING.ru.md`, `examples/client.json`, `examples/server.json` и установщик в `main` для новых Linux-серверов.
- [Public release preflight PASS](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44532739760) (Bash синтаксис, точный SHA, нулевые квоты, отсутствие внутренних ссылок на главной странице); installer CI PASS (run 44532252732).
- **Публикация бинарных файлов требует GitHub API от имени оператора Homeserver**, поскольку проверенный бинарник физически хранится там. Подготовлен `install/publish-v1.0.1.sh`, который использует официальную утилиту `gh` (предварительно авторизованную вне ChatGPT). Скрипт верифицирует SHA/исходный commit, создаёт тег через GitHub API, затем **draft** Release, загружает три asset, скачивает их обратно и побайтно сверяет перед снятием флага draft. Никаких production-сервисов или PKI не касается.

На Homeserver, если официальный `gh` уже доступен и авторизован:

```bash
cd /opt/tervon-tcp-bridge/m5/source || exit 1
git pull --ff-only origin work/m5-linux-release
bash install/publish-v1.0.1.sh
```

Успешный завершающий маркер: `M5_PUBLISH=PASS_PUBLIC_RELEASE` и фактическая ссылка `https://github.com/tervon-software/Tervon-TCP-Bridge/releases/tag/v1.0.1`. **До этого маркера публикацию нельзя считать завершённой.** При отсутствии `gh` или авторизации — `BLOCKED`, не подставлять токен в чат и не обходить проверку. Повторно не пересобирать и не переустанавливать Bridge.

## Итоговая публикация — RELEASED / PASS (2026-11-19)

Оператор подтвердил полное выполнение `install/publish-v1.0.1.sh`: `M5_PUBLISH_DRAFT_REMOTE_VERIFY=PASS`, `M5_PUBLISH=PASS_PUBLIC_RELEASE`, exit 0. Независимая проверка GitHub API: [v1.0.1 release](https://github.com/tervon-software/Tervon-TCP-Bridge/releases/tag/v1.0.1), `draft=false`, `prerelease=false`, published_at `2026-11-19T13:43:47Z`; тег `c77cf2685623e71705ae38f57125986fb9531fd8`. В GitHub находятся `tervon-bridge-linux-amd64` (6,869,176 байт), `SHA256SUMS` (92 байта), `BUILD_INFO` (256 байт). SHA256 бинарника `4f997b60bd777516339512ac71ddbb89b96d5bf9af979286ff3bd63690ea525b` совпадает с установленным и проверенным на двух серверах. Исходный коммит его сборки `96c136702d10b6fb9a16aa2893f6b978e8b14e86` не равен тегу, потому что релизный тег включает документацию и инструменты публикации.

**M5.5 ASSETS VERIFIED PASS; M5.6 OFFICIAL RELEASED PASS.** В публикации явно описаны не измеренные network RTT/per-stream fairness/live wall-clock max Jitter. Это остаётся исключением из набора количественных испытаний, не выданным за успешное тестирование. Следующие изменения продукта не входят в выпущенный тег. Не запускать publisher ещё раз, не пересобирать и не переустанавливать работающие мосты. M6 HOLD.
