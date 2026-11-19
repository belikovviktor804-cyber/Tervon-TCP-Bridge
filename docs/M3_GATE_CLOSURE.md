# M3 — Закрытие оставшихся критериев (2026-11-19)

## Источник истины и границы

Основной документ: `docs/DEVELOPMENT_PLAN.md`; рабочая ветка `work/m3-adaptive-jitter`. Linux v1.0.1 выпускается **после M5**, Android M6 — после Linux v1.0.1. Нельзя переходить к M4 до доказанного полного PASS M3.

**Уже подтверждено оператором:** M3 Bridge запущен на Homeserver и Finserver, счётчики PAD/Jitter растут; защищённый SSH-туннель метрик запускался успешно и установлен как active/enabled; публичный HTTPS API двух узлов PASS с Finserver через `87.103.198.219:443`; обе секции панели подтверждены скриншотами, UI PASS. Эти этапы повторять не требуется.

**Остаются два обязательных критерия:** (A) независимое восстановление *только* managed SSH-туннеля после принудительного завершения основного процесса; (B) реальная продолжительная нагрузка на действующий data plane, количественные доказательства задержек / fairness / отсутствия зависших потоков. Unit / synthetic Go-тесты не подменяют (B).

## A. Отказ и восстановление SSH-туннеля метрик

Тестовый скрипт `tests/homeserver/m3-managed-tunnel-recovery.sh`:

- Выполняется только на Homeserver через **detached** `systemd-run`: закрытие консоли не прерывает проверку.
- До отказа убеждается, что это именно активный и включённый `tervon-bridge-fin-metrics-tunnel.service` с `Restart=always`, MainPID указывает на `/usr/bin/ssh`, checkout расположен в `/opt/tervon-tcp-bridge/m3/source`, Git remote задан только через SSH.
- Посылает `SIGTERM` **только MainPID этого конкретного юнита**, никогда не делает `restart` или `kill` для Bridge, Stats, nginx и VPN.
- Не дольше 45 секунд наблюдает новый SSH MainPID в active-юните. Если он не вернулся, пытается поднять **только этот туннель** через `systemctl start`; причину и результат оставляет в журнале.
- Проверяет лишь **восстановление процесса**, не выдаёт это за восстановление передачи метрик. Публичное end-to-end доказательство делается отдельно с Finserver через внешний IPv4.

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

Критерии частичного PASS: `M3_TUNNEL_RESTART=PASS`, другой `TUNNEL_PID_AFTER`, успешное завершение unit; нет перезапусков data plane. Любой иной исход — BLOCKED, исследовать журналы и не продолжать M4.

### End-to-end после восстановления

На **Finserver**, независимо от Homeserver и без обращения к внутреннему адресу публичного сайта:

```bash
curl --noproxy '*' -fsS --max-time 10 \
  --resolve stats.xrayvless.online:443:87.103.198.219 \
  https://stats.xrayvless.online/bridge/api/status |
python3 -c 'import json,sys; d=json.load(sys.stdin); assert d["schema"]=="tervon.bridge.dashboard.v1"; assert all(d["nodes"][k]["status"]=="ok" for k in ("homeserver","finserver")); print("M3_TUNNEL_RECOVERY_PUBLIC_API=PASS")'
```

Этот PASS **действителен для восстановления**, только если запрос выполнен *после* управляемого SIGTERM и подтверждения смены PID.

## B. Сохранение границ задержки и испытания под реальной нагрузкой

**CI регрессии:** `prototype/jitter/fairness_test.go` содержит новую 30-секундную проверку восьми независимых потоков с синтетическими часами. Должны проходить `go test -race`, проверки верхней границы **4 мс/кадр** и **20 мс на фиксированное 1-секундное окно на поток**. Это **не sliding-window тест** и **не измерение реального RTT**.

**Для live PASS** требуется длительный контролируемый поток через действующий Tervon Bridge с согласованной нагрузкой, плюс параллельные короткие запросы. До/после сохранить обе JSON-метрики и доказать: throughput, latency/RTT, успешное завершение потоков, отсутствие роста `sessions_active` после закрытия нагрузки, отсутствие роста ошибок, реальную работу PAD/Jitter и отсутствие starvation у коротких потоков. Зафиксировать полную методику, продолжительность, объём, пределы и сырые артефакты. Нельзя делать вывод об RTT на основании только `connect_latency_p50_ms` — это время TCP dial к backend на Finserver.

**Текущий статус: M3 IN_PROGRESS.** В этой записи нет показаний, что fault injection или live long-run тест выполнены. Эти пункты закрываются только после операторских фактических логов и PASS.
