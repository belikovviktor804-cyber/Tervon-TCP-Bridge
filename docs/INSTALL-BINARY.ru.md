# Tervon TCP Bridge — установка готового бинарника

Tervon TCP Bridge передаёт двунаправленные TCP-соединения между двумя Linux-серверами через HTTP/2 поверх TLS 1.3. На входе работает **Encapsulator**, на удалённом сервере — **Decapsulator**. Внутренний протокол (например, VLESS/Trojan/Shadowsocks по TCP) остаётся непрозрачным для моста: Bridge передаёт байты, а не подменяет VPN-сервер.

**Исходники приватные.** Для установки нужен опубликованный бинарный файл, два JSON-конфига и сертификаты. Go и Git на целевых серверах не требуются.

## 1. Схема

```text
Клиентское приложение
    │ TCP
    ▼
Локальный VPN/Xray (необязательно)
    │ TCP на 127.0.0.1
    ▼
Encapsulator (mode=client)
    │ HTTPS / HTTP/2 / TLS 1.3 / клиентский сертификат
    ▼
Decapsulator (mode=server)
    │ TCP к заранее разрешённому backend
    ▼
Удалённый VPN/Xray или другой TCP-сервис
```

Клиенту моста не предоставляется возможность указать произвольный адрес назначения: сервисы и их backend задаются администратором Decapsulator.

## 2. Получение программы

На странице [Releases](https://github.com/belikovviktor804-cyber/Tervon-TCP-Bridge/releases) выберите опубликованный выпуск для `linux/amd64`. Скачайте два файла: `tervon-bridge-linux-amd64` и `SHA256SUMS`. Сверьте контрольную сумму:

```bash
sha256sum -c SHA256SUMS
```

Для проверенного бинарника Tervon TCP Bridge SHA-256:

```text
5c813bae9cb0c68e3d5ac1e9afb0556f13cb270b369a86ae8ffc0d20adb086e2
```

Этот хеш относится именно к указанному бинарнику; при выпуске новой версии используйте сумму из соответствующего Release.

## 3. Подготовка каждой машины

Команды ниже выполняются **на новом сервере**, а не поверх действующей установки:

```bash
sudo useradd --system --home /nonexistent --shell /usr/sbin/nologin tervon-bridge
sudo install -d -m 0755 /opt/tervon-tcp-bridge
sudo install -d -m 0750 -o root -g tervon-bridge /etc/tervon-tcp-bridge
sudo install -d -m 0750 -o root -g tervon-bridge /etc/tervon-tcp-bridge/pki
sudo install -m 0755 tervon-bridge-linux-amd64 /opt/tervon-tcp-bridge/bridge
```

Если пользователь уже существует, повторно создавать его не нужно. Путь `/opt/tervon-tcp-bridge/bridge` одинаков для обоих режимов.

## 4. Сертификаты и взаимная аутентификация

В примере используется собственный CA. Создайте CA в защищённой административной среде и храните `ca.key` **вне серверов**. Для каждого участника выпускается отдельный сертификат.

Пример команд для демонстрационного CA и сертификатов (OpenSSL 3):

```bash
openssl genpkey -algorithm EC -pkeyopt ec_paramgen_curve:P-256 -out ca.key
openssl req -x509 -new -sha256 -days 3650 -key ca.key -out ca.crt -subj "/CN=Tervon Bridge CA"

openssl genpkey -algorithm EC -pkeyopt ec_paramgen_curve:P-256 -out server.key
openssl req -new -key server.key -out server.csr -subj "/CN=bridge.example.org"
printf 'subjectAltName=DNS:bridge.example.org\nextendedKeyUsage=serverAuth\n' > server.ext
openssl x509 -req -in server.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out server.crt -days 365 -sha256 -extfile server.ext

openssl genpkey -algorithm EC -pkeyopt ec_paramgen_curve:P-256 -out client.key
openssl req -new -key client.key -out client.csr -subj "/CN=bridge-client"
printf 'extendedKeyUsage=clientAuth\n' > client.ext
openssl x509 -req -in client.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out client.crt -days 365 -sha256 -extfile client.ext
```

На удалённый сервер передайте `server.crt`, `server.key`, `ca.crt`; на локальный — `client.crt`, `client.key`, `ca.crt`. Приватные ключи передавайте только по защищённому каналу и не публикуйте. Файлы ключей должны быть доступны учётной записи `tervon-bridge` на чтение, но не другим пользователям.

Отпечаток клиентского сертификата для серверного разрешения:

```bash
openssl x509 -in client.crt -outform DER | sha256sum
```

Используйте 64 символа SHA-256 без пробелов в ключе `authorized`.

## 5. Настройка Decapsulator на удалённом сервере

Скопируйте [пример server.json](../examples/server.json) в `/etc/tervon-tcp-bridge/server.json` и настройте:

```json
{
  "mode": "server",
  "listen": "0.0.0.0:18443",
  "cert_file": "/etc/tervon-tcp-bridge/pki/server.crt",
  "key_file": "/etc/tervon-tcp-bridge/pki/server.key",
  "ca_file": "/etc/tervon-tcp-bridge/pki/ca.crt",
  "max_streams": 100,
  "routes": {
    "vpn": [
      {"Name": "vpn-backend", "Address": "127.0.0.1:19000"}
    ]
  },
  "authorized": {
    "ВСТАВЬТЕ_64_СИМВОЛА_SHA256_КЛИЕНТСКОГО_СЕРТИФИКАТА": ["vpn"]
  }
}
```

Здесь `127.0.0.1:19000` — адрес уже работающего удалённого TCP-сервиса. Порт `18443/tcp` должен быть доступен только тем клиентам, которым разрешено подключение. При наличии публичного IP настройте firewall и DNS `bridge.example.org` на этот сервер.

## 6. Настройка Encapsulator на локальном сервере

Скопируйте [пример client.json](../examples/client.json) в `/etc/tervon-tcp-bridge/client.json`:

```json
{
  "mode": "client",
  "listen": "127.0.0.1:19001",
  "cert_file": "/etc/tervon-tcp-bridge/pki/client.crt",
  "key_file": "/etc/tervon-tcp-bridge/pki/client.key",
  "ca_file": "/etc/tervon-tcp-bridge/pki/ca.crt",
  "server_name": "bridge.example.org",
  "remote_url": "https://bridge.example.org:18443/v1/stream",
  "service": "vpn",
  "max_local_connections": 100
}
```

Имя `server_name` должно совпадать с DNS SAN сертификата сервера. Значение `service` должно присутствовать одновременно в `routes` и в списке разрешённых сервисов соответствующего клиентского сертификата.

Локальный TCP-клиент, включая Xray, подключается к `127.0.0.1:19001`. Данные доставляются к удалённому `127.0.0.1:19000`.

## 7. Запуск через systemd

В репозитории есть шаблоны [client.service](../systemd/tervon-bridge-client.service.example) и [server.service](../systemd/tervon-bridge-server.service.example). На новом сервере выберите только соответствующий режим:

```bash
# Удалённый сервер:
sudo cp tervon-bridge-server.service.example /etc/systemd/system/tervon-bridge-server.service

# Локальный сервер:
sudo cp tervon-bridge-client.service.example /etc/systemd/system/tervon-bridge-client.service
```

Перед запуском проверьте пути `ExecStart`, конфигурацию и права на ключи. После этого на каждой машине:

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now tervon-bridge-server.service   # только на удалённой машине
# либо
sudo systemctl enable --now tervon-bridge-client.service   # только на локальной машине
```

Проверка состояния:

```bash
systemctl status tervon-bridge-server.service --no-pager
journalctl -u tervon-bridge-server.service -n 50 --no-pager
```

Для локального сервера замените имя unit на `tervon-bridge-client.service`.

## 8. Подключение Xray или другого TCP-приложения

Bridge не требует менять внутренний протокол приложения. В Xray направьте нужный TCP-outbound на локальный адрес Encapsulator `127.0.0.1:19001`. На удалённом сервере укажите в `routes.vpn[].Address` адрес соответствующего TCP-inbound Xray. У обоих Xray-соединений должны совпадать параметры их **собственного** протокола. Bridge не заменяет эту настройку.

Можно передавать и обычный TCP-поток: для Bridge это тот же двунаправленный набор байтов.

## 9. Диагностика

- Нет соединения с локальным портом: проверьте `listen` и состояние Encapsulator.
- Ошибка TLS: проверьте CA, срок сертификатов, DNS SAN, `server_name` и наличие клиентского сертификата.
- HTTP 403: проверьте отпечаток клиентского сертификата и `authorized`.
- HTTP 502: проверьте адрес и доступность удалённого backend.
- HTTP 503: проверьте лимит `max_streams`.
- HTTP 404: проверьте путь `/v1/stream`.

Подробности — в [диагностике](TROUBLESHOOTING.ru.md) и [описании конфигурации](CONFIGURATION.ru.md).

## 10. Обновление

Для обновления скачайте новый бинарник и его SHA-256, проверьте контрольную сумму и подготовьте отдельную копию. Сохраняйте текущую версию до успешного переключения. Не заменяйте исполняемый файл действующей службы в процессе подготовки.

**Tervon TCP Bridge готов к использованию как самостоятельный TCP-транспорт.** Исходники, секреты и реальные конфигурации инфраструктуры для установки не нужны.
