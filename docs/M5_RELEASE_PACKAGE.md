# M5 — актуальный релизный пакет (обновление после двухузлового cutover PASS)

**Дата:** 2026-11-19. Оператор подтвердил запуск на обоих серверах M5 с нулевой квотой сессий: `M5_DUAL_RESULT=PASS_TWO_NODE_CUTOVER`, отдельные SHA PASS, `max_sessions=0` на обоих узлах, public telemetry PASS. **GitHub Release v1.0.1 ещё не опубликован**.

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

`PUBLICATION_DONE=NO` в операторском отчёте. На GitHub проверено: релизы содержат `v1.0.0`, но нет `v1.0.1` (2026-11-19). Инсталляционный workflow статически подготовлен, его CI PASS пока не проверен. Для официального релиза проверить GitHub Actions `m5-installer.yml` и другие необходимые release gates, исходную лицензию/условия распространения, а также окончательный новый пакет. После этого загрузить три assets в GitHub Release `v1.0.1` через GitHub API; не публиковать бинарник с другим SHA или закрытые конфигурации/ключи. Документировать реальный Git tag SHA отдельно от исходного коммита сборки. Публикация **не подтверждена**.

## Исторический статус

До исправления М4 использовал жёсткую квоту 100 сессий. Пользовательский 180-секундный тест обнаружил 313 отклонённых подключений на Homeserver, несмотря на 0 ошибок транспорта. Оператор потребовал снять лимит; код исправлен, ветка протестирована (vet/unit/race/build PASS), а затем новый бинарник установлен одновременно на обоих узлах (LIVE SWITCH PASS, `max_sessions=0`). Новых post-cutover 180-секундных нагрузочных измерений пока не проводили; не приписывать старой нагрузке свойства нового экземпляра.
