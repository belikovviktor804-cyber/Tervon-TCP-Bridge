# M5 — одновременное переключение двух Linux Bridge на сборку без квоты

**Решение оператора:** выполнить обновление Homeserver и Finserver в одном запуске, с параллельными рестартами **только** `tervon-bridge-client.service` и `tervon-bridge-server.service`.

**ИТОГ 2026-11-19:** оператор выполнил сценарий, `M5_DUAL_RESULT=PASS_TWO_NODE_CUTOVER`. Оба бинарника нового SHA запущены, JSON с нулевой квотой подтверждён, оба публичных `max_sessions=0`, отработали подготовка и параллельное переключение. Инструкции ниже сохранены для аудита **уже состоявшегося однократного переключения**, не запускать повторно на работающих новых экземплярах.  Выполнение неизбежно прервёт существующие сессии Bridge на короткое время; постоянного бесшовного cutover не обещаем. Другие службы (Xray, Stats, nginx, Guard и пр.) не перезапускаем.

## Входные данные

| Параметр | Значение |
|---|---|
| SSH remote GitHub | `git@github.com:tervon-software/Tervon-TCP-Bridge.git` |
| Ветка | `work/m5-linux-release` |
| Протестированный исходник кандидата | `96c136702d10b6fb9a16aa2893f6b978e8b14e86` |
| SHA действующей M4 | `5035217a89476117894efe1a44db5dcdd1242773d7fd9461170f2cf6ccb98a90` |
| SHA нового M5 | `4f997b60bd777516339512ac71ddbb89b96d5bf9af979286ff3bd63690ea525b` |
| Подготовленный новый бинарник | `/opt/tervon-tcp-bridge/m5/unbounded/stage-qSxWhELI/dist/tervon-bridge-linux-amd64` |
| Homeserver unit/binary | `tervon-bridge-client.service` / `/opt/tervon-tcp-bridge/bridge` |
| Finserver unit/binary | `tervon-bridge-server.service` / `/opt/tervon-bridge-linux-amd64` |
| SSH-доступ из Homeserver на Finserver | `ssh finserver` в root SSH config |

## Запуск с Homeserver

Первоначально синтаксис и offline-тесты должны завершиться PASS. **Если хоть одна команда не проходит, переключение не выполняется**.

```bash
set -Eeuo pipefail
cd /opt/tervon-tcp-bridge/m5/source
git pull --ff-only origin work/m5-linux-release

python3 -m py_compile \
  tests/m5/m5-dual-cutover-worker.py \
  tests/m5/m5-dual-cutover.py \
  tests/m5/test_m5_dual_cutover.py \
  tests/m5/test_m5_dual_orchestration.py

python3 -m unittest discover -s tests/m5 -p 'test_m5_dual_*.py' -v

systemd-run --unit=tervon-m5-dual-cutover --collect \
  --property=TimeoutStartSec=600 \
  /usr/bin/python3 \
  /opt/tervon-tcp-bridge/m5/source/tests/m5/m5-dual-cutover.py deploy
```

Журнал процесса:

```bash
journalctl -u tervon-m5-dual-cutover -n 100 --no-pager
```

**Обратите внимание:** успешное создание unit `systemd-run` значит только, что операция запущена, **не** что уже подтверждён PASS. Ищите окончательную строку `M5_DUAL_RESULT=PASS_TWO_NODE_CUTOVER` или сообщение об откате/блокировке. При `CRITICAL_ROLLBACK_INCOMPLETE` остановитесь и проведите ручное восстановление по приватным бэкапам.

## Что делает сценарий

1. Проверяет ветку, SSH remote, SHA готового кандидата. По SSH доставляет бинарник и скрипт на Finserver (в отдельный каталог M5).
2. Проверяет наличие **старого** работающего M4 SHA на обеих нодах, корректные `ExecStart` и JSON, значения квот **ровно 100**. Не публикует и не печатает приватные JSON или PKI. При несовпадении не начинает рестарт.
3. На каждой машине сохраняет бинарник, конфигурацию и проверяемый manifest под `/opt/tervon-tcp-bridge/m5/cutover-no-quota` с ограниченными правами. Создаёт копии конфигураций, меняя **только одно поле**: клиент `max_local_connections=0`, сервер `max_streams=0`.
4. После подготовки обоих запускает **параллельно** замену бинарника и JSON и перезапуск только двух Bridge unit. Два рестарта не атомарны по времени: возможен кратковременный разрыв существующих клиентов.
5. Для каждой машины проверяет, что новый процесс действительно исполняет SHA M5 и читает config с нулевой квотой. Затем ожидает через существующий read-only dashboard `ready=true` и `max_sessions=0` на **обоих** узлах.
6. Если переключение одного из узлов либо общая проверка не прошла, выполняется **попытка отката на обоих узлах** с сохранённых старого бинарника и JSON. При сбое самого отката нельзя заявлять успешное восстановление — нужен отчёт и ручная диагностика.

## После успешного запуска

```bash
cd /opt/tervon-tcp-bridge/m5/source
python3 tests/m5/m5_capacity_readonly.py
```

Ожидаемые строки:

```text
M5_CAPACITY_HOMESERVER_CONFIGURED_MAX_SESSIONS=0
M5_CAPACITY_FINSERVER_CONFIGURED_MAX_SESSIONS=0
M5_CAPACITY=PASS_READ_ONLY_LIMIT_OBSERVED
```

`0` означает отсутствие **программного ограничения количества TCP-сессий**, а не бесконечные ресурсы CPU, RAM, сокетов и файловых дескрипторов. Не повышайте произвольно системные пределы и не добавляйте новые клиентские квоты без отдельного решения.

## Границы безопасности и неудачные запуски

- Скрипт рассчитан только на текущие конкретные бинарные SHA, роли, unit и конфиги. Путь и содержимое private configs никогда не копируются в GitHub.
- Повторный `deploy` при существующем каталоге snapshot **намеренно отказывается работать** и не стирает бэкапы. Не удаляйте этот каталог для «повторной попытки» без осмотра состояния.
- Если Finserver недоступен по SSH или неожиданно поменялась конфигурация, остановка происходит **до** рестартов.
- Подготовленные ранее `/opt/tervon-tcp-bridge/m5/release` assets содержат **старый M4** и не должны публиковаться как обновлённая v1.0.1.
- Финальный GitHub Release будет собран **после** подтверждения двухузлового cutover и обновления привязанного SHA установщика.
