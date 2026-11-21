# M6 — фактическая граница восстановления Android Bridge (21.11.2026)

**Источник:** присланный оператором `tervonvpndpi-m6-source-snapshot-20261121.tar.gz` (67 файлов), текущая ветка `work/m6-android-ingress` и `integrations/android-ingress/ENROLLMENT_CONTRACT.ru.md`. Снимок неполон: отсутствуют установленный APK, ARM64 ELF и полная среда сборки. Это не отчёт об испытании VPN.

## Подтверждено по Android-исходникам

1. `VpnForegroundService.saveSubscription(...)` формирует Xray JSON через `VlessConfigParser.parseSubscriptionToConfig` и сохраняет его прямо в `filesDir/xray/config.json`. Вызова Bridge нет.
2. `VpnForegroundService.startVpnInternal()` читает этот JSON и поднимает существующие Xray/HEV/TUN. `Builder.addDisallowedApplication(packageName)` исключает собственный UID из VPN; сохранение этого поведения критично, а реальную маршрутизацию нужно проверить на устройстве.
3. `BridgeRuntimeControl.kt` отдельно умеет записать конфигурацию, запустить/остановить существующий ELF, но `vpnRouteIntegrated=false`. Запущенный ELF сам по себе не переключает VLESS.
4. `BridgeDeviceIdentity.kt` и `BridgeDeviceCsr.kt` из Stage 4 существуют. `BridgeServerCertificateStore.kt` проверяет CA, `clientAuth` и ключ устройства, но **не выполняет регистрацию**.
5. Снимок содержит устаревшую политику `REQUEST_USER_REGISTRATION`, противоречащую ранее утверждённой автоматической регистрации. В GitHub подготовлено её минимальное исправление `REQUEST_AUTO_ENROLLMENT`; **не применено к живому Android-проекту**.
6. Подготовленные в GitHub `BridgeVlessRoutePlan.kt`, `BridgeXrayConfigRoute.kt`, `BridgeClientConfigBuilder.kt`, `BridgeClientPemHandoff.kt` — **вспомогательные компоненты, не подключённые к жизненному циклу Xray/VPN**. Их автономные проверки нельзя считать E2E.

## Подтверждено по серверной части в GitHub

1. `integrations/android-ingress/xui/cmd/m6-first-device/main.go` — **локальная операторская CLI**, а не HTTP API для Android: требует `--csr`, `--device-cn`, `--output`, локальные сертификаты CA и запуск root.
2. CLI вызывает `xui.SignDeviceCSR`, `xui.AddDeviceToACL`, создаёт `android-client.crt` и `server.staged.json` в новом приватном каталоге, **не активирует ACL**; это намеренно `STAGED_ONLY`.
3. Документ `ENROLLMENT_CONTRACT.ru.md` прямо определяет интеграцию с существующим источником авторизации VLESS-подписок как **ещё не изученную**. На основании доступных исходников нельзя подтвердить конечную точку регистрации, её аутентификацию, схему ответа, доверенный discovery или безопасное применение нового сертификата в ACL.
4. Bridge v1.0.1 читает ACL при старте. Механизм разрешённого добавления нового клиента без перебоя уже работающего ingress **не подтверждён**. Запрещено менять live-сервис или ослаблять mTLS ради теста.

## Блокирующий шаг по утверждённому плану

**Автоматическая выдача клиентского mTLS-сертификата** по существующей VLESS-подписке: необходимы проверенные контракт API, действующая авторизация пользователя, привязка CSR, атомарное разрешение отпечатка клиентского сертификата и безопасная активация ingress. После этого — сохранение сертификата, запуск подготовленного Bridge и транзакционная маршрутизация Xray с возвратом к прямому VLESS по выключению обхода.

**Нельзя:** вводить новую регистрацию/секрет, доверять произвольному SNI/URL, выключать mTLS, использовать операторский CLI как публичный API, перестраивать старый Bridge, менять production-службы, подписывать или публиковать APK до проверенной интеграции.

## Решение и статус

- Восстанавливать Stage 3 и необходимый Stage 4, **не начинать всё приложение заново**.
- Отдельные локальные исходники и тесты есть; **подключение к реальному Android не подтверждено**.
- **M6: BLOCKED_ON_EXISTING_SUBSCRIPTION_AUTH_AND_SERVER_ENROLLMENT_CONTRACT.**
- **ANDROID_E2E=NOT_RUN; CERT_ENROLLMENT=NOT_IMPLEMENTED; LIVE_ACL=UNCHANGED; APK=NOT_BUILT.**
- Следующее действие — получить фактическую реализацию источника VLESS-подписок и утверждённого серверного endpoint из Homeserver/профильного репозитория; если их нет, согласовать конкретный минимальный серверный интерфейс без параллельной системы аккаунтов, затем реализовать и протестировать этот обязательный участок.
