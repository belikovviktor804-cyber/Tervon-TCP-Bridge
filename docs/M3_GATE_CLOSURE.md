# M3 — Закрытие оставшихся критериев (2026-11-19)

## Источник истины и границы

Основной документ: `docs/DEVELOPMENT_PLAN.md`; рабочая ветка `work/m3-adaptive-jitter`. Linux v1.0.1 выпускается **после M5**, Android M6 — после Linux v1.0.1. Нельзя переходить к M4 до доказанного полного PASS M3.

**Уже подтверждено оператором:** M3 Bridge запущен на Homeserver и Finserver, счётчики PAD/Jitter растут; защищённый SSH-туннель метрик запускался успешно и установлен как active/enabled; публичный HTTPS API двух узлов PASS с Finserver через `87.103.198.219:443`; обе секции панели подтверждены скриншотами, UI PASS. Эти этапы повторять не требуется.

**Актуальный статус:** (A) восстановление managed SSH-туннеля **LIVE PASS 2026-11-19** (PID 1474240 → 1672169 за 6 секунд; внешний HTTPS двухузловой API PASS через 8 секунд); повторять не требуется. (B) длительная реальная нагрузка с доказательствами RTT/fairness/отсутствия зависших потоков **PENDING**. Unit / synthetic Go-тесты не подменяют (B).

## A. Отказ и восстановление SSH-туннеля метрик

Тестовый скрипт `tests/homeserver/m3-managed-tunnel-recovery.sh`:

- Выполняется только на Homeserver через **detached** `systemd-run`: закрытие консоли не прерывает проверку.
- До отказа убеждается, что это именно активный и включённый `tervon-bridge-fin-metrics-tunnel.service` с `Restart=always`, MainPID указывает на `/usr/bin/ssh`, checkout расположен в `/opt/tervon-tcp-bridge/m3/source`, Git remote задан только через SSH.
- Посылает `SIGTERM` **только MainPID этого конкретного юнита**, никогда не делает `restart` или `kill` для Bridge, Stats, nginx и VPN.
- Не дольше 45 секунд наблюдает новый SSH MainPID в active-юните. Если он не вернулся, пытается поднять **только этот туннель** через `systemctl start`; причину и результат оставляет в журнале.
- После восстановления процесса самостоятельно подключается по SSH к Finserver и выполняет HTTPS-запрос через **внешний IPv4 87.103.198.219** с корректными TLS SNI и проверкой сертификата. До 3 попыток проверяет реальный JSON двух узлов. Не подменяет внешний адрес локальным.

### Операторский запуск (один раз после подтверждённого GitHub CI PASS)

На Homeserver, где находится действующий SSH checkout:

```bash
git -C /opt/tervon-tcp-bridge/m3/source pull --ff-only origin work/m3-adaptive-jitter
systemd-run --unit=tervon-m3-tunnel-recovery --collect \
  /bin/bash /opt/tervon-tcp-bridge/m3/source/tests/homeserver/m3-managed-tunnel-recovery.sh
```

После выполнения (запускать можно в новой SSH-консоли):

```bash
journalctl -u tervon-m3-tunnel-recovery -n 45 --no-pager
cat /opt/tervon-tcp-bridge/m3/evidence/m3-tunnel-recovery-result.txt
```

Критерии **полного tunnel recovery PASS**: `M3_TUNNEL_RESTART=PASS`, другой `TUNNEL_PID_AFTER`, `M3_TUNNEL_RECOVERY_PUBLIC_API=PASS`, успешное завершение unit; нет перезапусков data plane. Любой иной исход — BLOCKED, исследовать журналы и не продолжать M4.

### Дополнительная независимая проверка извне (необязательная)

Основной сценарий **уже включает** внешний end-to-end API через Finserver. Если нужен отдельный независимый контроль, на Finserver и без обращения к внутреннему адресу публичного сайта:

```bash
curl --noproxy '*' -fsS --max-time 10 \
  --resolve stats.xrayvless.online:443:87.103.198.219 \
  https://stats.xrayvless.online/bridge/api/status |
python3 -c 'import json,sys; d=json.load(sys.stdin); assert d["schema"]=="tervon.bridge.dashboard.v1"; assert all(d["nodes"][k]["status"]=="ok" for k in ("homeserver","finserver")); print("M3_TUNNEL_RECOVERY_PUBLIC_API=PASS")'
```

Встроенный в сценарий внешний PASS **действителен для восстановления**, только если запрос выполнен *после* управляемого SIGTERM и подтверждения смены PID. Сам факт подготовки этого скрипта не является фактическим PASS.

## B. Сохранение границ задержки и испытания под реальной нагрузкой

**CI регрессии:** `prototype/jitter/fairness_test.go` содержит новую 30-секундную проверку восьми независимых потоков с синтетическими часами. Должны проходить `go test -race`, проверки верхней границы **4 мс/кадр** и **20 мс на фиксированное 1-секундное окно на поток**. Это **не sliding-window тест** и **не измерение реального RTT**.

**Для live PASS** требуется длительный контролируемый поток через действующий Tervon Bridge с согласованной нагрузкой, плюс параллельные короткие запросы. До/после сохранить обе JSON-метрики и доказать: throughput, latency/RTT, успешное завершение потоков, отсутствие роста `sessions_active` после закрытия нагрузки, отсутствие роста ошибок, реальную работу PAD/Jitter и отсутствие starvation у коротких потоков. Зафиксировать полную методику, продолжительность, объём, пределы и сырые артефакты. Нельзя делать вывод об RTT на основании только `connect_latency_p50_ms` — это время TCP dial к backend на Finserver.

### Новый безопасный сбор наблюдений во время настоящей эксплуатации

`tests/homeserver/m3_passive_live_evidence.py` считывает **только реальный публичный HTTPS** через Finserver, с фиксированным внешним IPv4 `87.103.198.219`, строгой TLS-проверкой и двумя `status=ok` узлами. По умолчанию 180 секунд, каждые 10 секунд, всего 19 срезов. Источник — реально работающие службы, не изолированный стенд.

Инструмент работает без установки библиотек, не изменяет VPN-подключения, nginx, Bridge, Stats, SSH-туннель или DNS. В журнал идут только разрешённые агрегированные счётчики и дельты; адреса клиентов и секреты не сохраняются. Скрипт отказывается выдавать PASS при пропаже узла, сбросе счётчиков или ошибке чтения.

Из чистого Homeserver checkout на последнем подтверждённом коммите можно запустить его отдельно от SSH-терминала:

```bash
git -C /opt/tervon-tcp-bridge/m3/source pull --ff-only origin work/m3-adaptive-jitter
systemd-run --unit=tervon-m3-passive-live --collect \
  /usr/bin/python3 /opt/tervon-tcp-bridge/m3/source/tests/homeserver/m3_passive_live_evidence.py \
  --duration 180 --interval 10
```

Результат не зависит от закрытия консоли:

```bash
journalctl -u tervon-m3-passive-live -n 50 --no-pager
```

**Граница доказательств:** `PASS_PASSIVE_TWO_NODE_COUNTER_CONTINUITY` означает только, что публичные счётчики непрерывны и оба узла отвечали. Он **не означает** RTT PASS, fairness PASS, доказательство 4 мс/кадр и 20 мс/сек для каждого потока или завершение M3. Из агрегатов панели эти параметры восстановить невозможно; нужен отдельный измеренный поток с контролируемым endpoint. Нельзя подставлять `connect_latency_p50_ms` Finserver в качестве RTT — это время backend dial.

**Текущий статус: M3 IN_PROGRESS.** Туннельный live gate закрыт; RTT/fairness испытания ещё не выполнены. В M4 переходить только после их подтверждения.
