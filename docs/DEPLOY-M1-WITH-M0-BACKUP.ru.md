# M1 — контролируемая установка с резервной копией M0

**Статус:** подготовлен сценарий развёртывания, **не выполнялся** на рабочих серверах. M1 code/isolated Homeserver tests PASS на `5ae795f0a6f342724f6dc894f0e82b7f4402f867`, включая реальный установленный M0. Сценарий развёртывания требует отдельной проверки на двух узлах. Это не релиз v1.0.1.

**Порядок:** сначала Finserver (Decapsulator), затем Homeserver (Encapsulator). При замене бинарника выполняется `systemctl restart`, что **прерывает активные соединения** соответствующего сервиса. Не выполнять в период, когда разрыв недопустим.

## 1. Только чтение: проверить оба узла

На Homeserver:

```bash
cd /opt/tervon/bridge-m0-check/build/repo || exit 1
git fetch origin work/m1-framing
git switch work/m1-framing
git merge --ff-only FETCH_HEAD
git status --short
git rev-parse HEAD
sudo bash tests/homeserver/deploy-m1-with-m0-backup.sh inspect tervon-bridge-client.service
```

На Finserver нужен тот же скрипт (передать по SSH с Homeserver или получить репозиторий через **SSH Git remote**). Выполнить:

```bash
sudo bash /opt/tervon-tcp-bridge/deploy-m1-with-m0-backup.sh inspect tervon-bridge-server.service
```

**До установки** сверить имя сервиса, путь к исполняемому файлу, активность, хеши дискового и запущенного M0. Если `inspect` не прошёл, **не устанавливать**, выяснить фактическую конфигурацию. Не раскрывать содержимое JSON и приватных ключей.

## 2. Собрать кандидат M1 на Homeserver, не трогая рабочий бинарник

```bash
cd /opt/tervon/bridge-m0-check/build/repo || exit 1
sha=$(git rev-parse HEAD)
test "$(git branch --show-current)" = work/m1-framing || exit 1
test -z "$(git status --porcelain)" || exit 1
(
  cd prototype || exit 1
  go build -trimpath -ldflags="-X main.buildVersion=M1 -X main.buildCommit=$sha" \
    -o /opt/tervon-tcp-bridge/bridge-m1.candidate ./cmd/bridge
)
sha256sum /opt/tervon-tcp-bridge/bridge-m1.candidate
```

Этот файл — только кандидат; действующий `/opt/tervon-tcp-bridge/bridge` не изменяется. Скопировать кандидат и скрипт на Finserver по SSH (подставить реальный SSH host/alias, **не** публиковать IP/ключи):

```bash
scp /opt/tervon-tcp-bridge/bridge-m1.candidate \
  FIN_SSH_HOST:/opt/tervon-tcp-bridge/bridge-m1.candidate
scp tests/homeserver/deploy-m1-with-m0-backup.sh \
  FIN_SSH_HOST:/opt/tervon-tcp-bridge/deploy-m1-with-m0-backup.sh
```

Сравнить `sha256sum` кандидата на обоих узлах. При несовпадении остановиться.

## 3. Установить серверную часть первой

На Finserver под root, **только после PASS preflight**:

```bash
sudo bash /opt/tervon-tcp-bridge/deploy-m1-with-m0-backup.sh \
  install tervon-bridge-server.service \
  /opt/tervon-tcp-bridge/bridge-m1.candidate \
  EXPECTED_FIN_M0_SHA256
```

Скрипт создаёт `/opt/tervon-tcp-bridge/backups/m0-XXXXXXXX/` (0700), сохраняет старый бинарник, unit и конфигурацию (включая секреты, доступные только root), проверяет SHA, атомарно заменяет бинарник, перезапускает **только выбранный bridge unit** и сверяет SHA запущенного процесса. При неуспехе автоматически возвращает M0 и пытается восстановить сервис. При SIGKILL/power loss возможен ручной rollback.

Проверить состояние сервиса и доступность старого `/v1/stream` с Homeserver до следующего шага. Если появились ошибки — откатить Finserver.

## 4. Установить клиентскую часть второй

На Homeserver:

```bash
sudo bash tests/homeserver/deploy-m1-with-m0-backup.sh \
  install tervon-bridge-client.service \
  /opt/tervon-tcp-bridge/bridge-m1.candidate \
  EXPECTED_HOME_M0_SHA256
```

Проверить `systemctl is-active tervon-bridge-client.service`, `systemctl status`, реальный TCP-трафик и журнал. **Конфигурация `remote_url` остаётся `/v1/stream`**: это осознанная промежуточная стадия совместимого развёртывания, а не включение TBF1.

## 5. Активация TBF1 /v2/stream — отдельная контролируемая операция

Только после подтверждённого PASS на обоих узлах и отдельного E2E через действующий маршрут: резервировать `client.json`, атомарно заменить **только** `remote_url` с `/v1/stream` на `/v2/stream`, перезапустить **только** client unit, проверить трафик. При неуспехе вернуть сохранённый `client.json` и перезапустить client. Не менять серверные маршруты, сертификаты, firewall или другие службы. Пока этот шаг не выполнен, **M1 binary deployed, TBF1 inactive**.

## 6. Ручной откат

Сначала Homeserver, затем Finserver; использовать конкретные пути `BACKUP_DIRECTORY=` из выводов установки:

```bash
sudo bash tests/homeserver/deploy-m1-with-m0-backup.sh \
  rollback tervon-bridge-client.service \
  /opt/tervon-tcp-bridge/backups/m0-XXXXXXXX
```

На Finserver:

```bash
sudo bash /opt/tervon-tcp-bridge/deploy-m1-with-m0-backup.sh \
  rollback tervon-bridge-server.service \
  /opt/tervon-tcp-bridge/backups/m0-YYYYYYYY
```

Если включён /v2, сначала вернуть `remote_url` в /v1 на клиенте (с сохранённого JSON). Откат бинарника не изменяет конфигурацию автоматически. Скрипт rollback проверяет манифест, SHA резервного бинарника, запускает старый процесс и сверяет его SHA.

**Критерии установки:** backup M0 на каждом узле; SHA активного M1 совпадает с SHA кандидата; оба bridge units active; `/v1/stream` работает; подтверждён реальный TCP-трафик; процедура отката известна. **Не объявлять эксплуатационный PASS по одним лишь `systemctl active`.**
