# Диагностика Tervon TCP Bridge

Проверяйте маршрут по слоям, не смешивая проблемы Xray, Bridge и внешнего Hysteria2.

| Симптом | Что проверить |
|---|---|
| Локальный Xray не соединяется | Адрес `127.0.0.1:55635`, состояние Encapsulator, соответствие VLESS outbound |
| TLS handshake failure | CA, SAN/`server_name`, клиентский сертификат, срок действия, TLS 1.3, ALPN `h2` |
| HTTP 403 | Отпечаток SHA-256 DER клиентского leaf-сертификата и разрешённый `service` |
| HTTP 502 | Доступность разрешённых TCP backend на finserver |
| HTTP 503 | Достигнут `max_streams` |
| HTTP 404 | Неверный путь `/v1/stream`, метод или другой HTTP endpoint |
| TCP-соединение обрывается | Логи обеих сторон, backend, промежуточная сеть; действующий сеанс автоматически не восстанавливается |
| Интернет не работает, но Bridge доступен | Xray routing, DNS, выход Hysteria2, adblock и правила прямого выхода |

Команды чтения на Homeserver:

```bash
systemctl show tervon-bridge-client.service -p ActiveState -p MainPID -p ExecStart
journalctl -u tervon-bridge-client.service -n 100 --no-pager
ss -ltn | grep -F ':55635' || true
sha256sum /opt/tervon-tcp-bridge/bridge
```

Для Xray `run -test -config ...` нужно дождаться завершения команды и проверить её exit code. Прерывание `Ctrl+C` не является успешной проверкой. Не запускайте параллельный экземпляр на действующем порту и не публикуйте полные конфигурации с секретами.
