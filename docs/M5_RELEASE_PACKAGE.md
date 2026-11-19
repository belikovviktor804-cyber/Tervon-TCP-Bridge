
> **ВНИМАНИЕ — НОВЫЙ КАНДИДАТ 2026-11-19:** прежняя процедура ниже упаковала M4 SHA `5035217a...` до решения оператора убрать фиксированные квоты. **НЕ публиковать этот прежний бинарник как финальную v1.0.1.** Исправленный M5 SOURCE TESTS + STAGING BUILD PASS, SHA256 `4f997b60bd777516339512ac71ddbb89b96d5bf9af979286ff3bd63690ea525b`, исходник `96c136702d10b6fb9a16aa2893f6b978e8b14e86`, staging `/opt/tervon-tcp-bridge/m5/unbounded/stage-qSxWhELI/dist`. Новый кандидат **НЕ проверен в live**, release uploader и прежний `install/install-bridge.sh` всё ещё привязаны к старому SHA, их нельзя применять для нового релиза без обновления и подтверждения правильного артефакта. Два production сервера без изменений; M5.2+M5.3 продолжает работу.

# M5 — подготовка и публикация Linux v1.0.1 в GitHub Releases

**Ветка:** `work/m5-linux-release`  
**M4:** `CLOSED / OPERATOR PASS`  
**M5.2+M5.3:** offline PASS, live passive PASS, адаптивный Jitter policy PASS; **полная 180-секундная сводка, samples=19, получена**: transfer/PAD/Jitter/error continuity PASS, но Homeserver отверг 313 новых подключений; capacity gate OPEN, RTT/fairness/wall-clock jitter max NOT_MEASURED  
**M5.5 package files:** OLD M4 release assets obsolete for new no-quota build; operator preparation PASS (`RELEASE_PREP=PASS_VERIFIED_EXISTING_M4`, exit 0); **M5.4/M5.5/M5.6 gate:** CI and approval pending; GitHub Release **NOT RELEASED**.  
**M6:** HOLD.

## Обязательные входные данные

1. Точный уже работающий M4 бинарник Linux amd64 SHA256 `5035217a89476117894efe1a44db5dcdd1242773d7fd9461170f2cf6ccb98a90`.
2. Исходник проверенного M4: `8d0aa3777c209d2825a1c11dfbae9487893cb7f5`. Не меняем его и не пересобираем релиз из следующего коммита. Тег Release может быть создан на итоговом Git-коммите документации M5 — **это не тот же идентификатор, что build SOURCE_COMMIT M4**. Обе ссылки указываем явно.
3. Финальная сводка `M5_LOAD` с YouTube + двумя Speedtest; проверить ошибки, динамику трафика, реальный Padding/Jitter, поведение обоих серверов, отчёт о ресурсах. Не подменять фактические сетевые RTT/cross-stream fairness величинами loopback.
4. Решение о распространении и соответствующая лицензия/условия использования. В корне репозитория на момент подготовки не найден файл `LICENSE`; наличие исходников в публичном GitHub само по себе **не предоставляет** третьим лицам права копирования и модификации.
5. Утверждённые результаты M5.4/M5.5 (сверка rollback/reliability без необоснованного вмешательства в текущий сервер, целостность пакета, CI тесты installer). М5.6 допускается только после этих проверок.

## Подготовка assets (только на Homeserver, без изменения служб)

После `git pull --ff-only origin work/m5-linux-release` из SSH checkout:

```bash
cd /opt/tervon-tcp-bridge/m5/source || exit 1
bash install/prepare-release-assets.sh
```

Скрипт проверяет SHA **существующей подготовленной копии M4** в `/opt/tervon-tcp-bridge/m4/cutover-homeserver/m4-bridge`, копирует бинарник в отдельный каталог `/opt/tervon-tcp-bridge/m5/release/`, создаёт:

```text
tervon-bridge-linux-amd64
SHA256SUMS
BUILD_INFO
```

Никаких `/etc/`, ключей, private configs или бэкапов в пакет не попадает. Скрипт **откажется перезаписывать** уже созданные бинарник и манифест. Он не публикует релиз и не затрагивает запущенные Bridge, Stats, Xray или nginx.

Проверка:

```bash
cd /opt/tervon-tcp-bridge/m5/release || exit 1
sha256sum -c SHA256SUMS
cat BUILD_INFO
```

**Подпись пакета** (GPG/cosign) возможна только при утверждённом способе управления signing key; не создавайте и не выкладывайте секретные ключи в CI ради удобства.

## Публикация GitHub Release — только после gate PASS

Инструмент `gh` использует GitHub API с правами публикации Releases. Git clone/pull остаётся строго SSH:

```bash
git -C /opt/tervon-tcp-bridge/m5/source remote get-url origin
# ожидается git@github.com:tervon-software/Tervon-TCP-Bridge.git
```

После подтверждённого решения о выпуске, наличия правильных прав и сохранённого итогового коммита (пример использует переменную `RELEASE_COMMIT`, которую необходимо установить **в фактический утверждённый SHA**):

```bash
RELEASE_COMMIT='<УТВЕРЖДЁННЫЙ_40_СИМВОЛЬНЫЙ_SHA>'
cd /opt/tervon-tcp-bridge/m5/source || exit 1
gh auth status
gh release create v1.0.1 \
  --repo tervon-software/Tervon-TCP-Bridge \
  --target "$RELEASE_COMMIT" \
  --title 'Tervon TCP Bridge v1.0.1 — Linux amd64' \
  --notes-file docs/RELEASE_NOTES_v1.0.1.ru.md \
  /opt/tervon-tcp-bridge/m5/release/tervon-bridge-linux-amd64 \
  /opt/tervon-tcp-bridge/m5/release/SHA256SUMS \
  /opt/tervon-tcp-bridge/m5/release/BUILD_INFO
```

**Не выполнять это сейчас**: пример намеренно требует точного SHA и PASS всех gate. Не публиковать пустой/непроверенный тег и не загружать новый бинарник с отличающимся хешем. Не выполнять `gh auth login` с публикацией токена в истории команд или чате.

## После публикации

- Проверить ровно один GitHub Release `v1.0.1`, его исходный tag commit, доступность всех трёх assets и `sha256sum -c SHA256SUMS` на **скачанном** из Release файле.
- Проверить установщик на чистом тестовом сервере (`install client` и `install server` — отдельные роли), без замены работающей инфраструктуры.
- Завершить и зафиксировать M5.5/M5.6 `RELEASED` только по подтверждённой ссылке и результатам проверки.
- Инструкция установки: [INSTALL-v1.0.1.ru.md](INSTALL-v1.0.1.ru.md); release notes: [RELEASE_NOTES_v1.0.1.ru.md](RELEASE_NOTES_v1.0.1.ru.md).

## Важная граница

Пользователь подтвердил работоспособность VPN. Исходные и Jitter-policy тесты также пройдены. Промежуточный стресс-тест показал живой Padding/Jitter; **полный трёхминутный отчёт пока отсутствует**. Статус `M5 IN_PROGRESS` нужно держать честным до его сверки и завершения указанных release blockers.

## Операторский результат package preparation (2026-11-19)

`RELEASE_PREP=PASS_VERIFIED_EXISTING_M4` / `RELEASE_PREP_EXIT_CODE=0`; версия `v1.0.1`; origin M4 source `8d0aa3777c209d2825a1c11dfbae9487893cb7f5`; SHA256 `5035217a89476117894efe1a44db5dcdd1242773d7fd9461170f2cf6ccb98a90`; артефакты `/opt/tervon-tcp-bridge/m5/release`. Отдельно подтверждено `PUBLICATION_DONE=NO`. **Повторная пересборка или повторный prepare не требуются**. Следующий шаг — финальная сводка `M5_LOAD`, CI и release gate; upload/tag не выполнены.

## Причина неполного стресс-отчёта установлена

Оператор самостоятельно остановил нагрузочный сбор после 9/18 интервалов: оценил наблюдавшиеся нагрузки и работу приложения достаточными. Отсутствие summary не является сбоем Bridge/установщика. Не требуется запускать повторный стресс-тест только ради 180 секунд. Опубликованные release notes должны описывать этот факт, не утверждая полный 180s PASS; количественные RTT и per-stream fairness остаются без прямых измерений.

## Получена полная 180-секундная сводка — ранее описанный недостающий отчёт больше не блокер

По переданному оператором JSON: `samples=19`, `duration_seconds_target=180`, `PASS_PAD_AND_JITTER_ACTIVATED`, нулевые transport errors, 424 стартовавших сессии на обеих нодах, 204 670 Padding frames и 18 Jitter frames. **Однако** `homeserver.sessions_rejected=313`; влияние на клиентов и текущий configured admission cap не установлены. **Не объявлять M5.2+M5.3 полностью завершённым**, пока не рассмотрены отказы и выбранное release acceptance. Прежний текст о частичном 9-сэмпловом прогоне относится к отдельно предъявленному журналу; это не причина игнорировать новую полную сводку. Бинарный пакет уже подготовлен PASS; повторный prepare не нужен.
