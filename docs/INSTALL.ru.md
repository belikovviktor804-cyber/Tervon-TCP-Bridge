# Установка Tervon TCP Bridge

Это руководство описывает новую установку Tervon TCP Bridge на двух Linux-серверах. Один сервер принимает локальные TCP-подключения (**Encapsulator**), второй доставляет их к заранее разрешённому TCP-сервису (**Decapsulator**).

В примерах используются вымышленные адреса и отдельные порты. Они не относятся к действующей инфраструктуре Tervon VPN.

## 1. Что понадобится

- Два Linux-сервера архитектуры x86-64.
- Подготовленный бинарный файл `tervon-bridge-linux-amd64`, полученный из авторизованного релиза проекта.
- TCP-сервис на принимающем сервере, например Xray на `127.0.0.1:19000`.
- Сетевое соединение от Encapsulator до HTTPS-порта Decapsulator.
- Собственный центр сертификации (CA) и отдельные сертификаты для клиента и сервера.
- Доступ администратора для первоначальной настройки systemd и каталогов.

Для обычной установки **Go и исходный код не нужны**: программа поставляется готовым бинарным файлом.

## 2. Схема примера

```text
Сервер A (Encapsulator)                   Сервер B (Decapsulator)
127.0.0.1:19001                           bridge.example.org:18443
        │                                            │
        └──────────── HTTPS/2 + mTLS ────────────────┘
                                                     │
                                                     ▼
                                            127.0.0.1:19000
                                              TCP backend
```

В примере имя сервиса — `vpn`. Это логическое имя маршрута, одинаковое в клиентской и серверной конфигурациях.

## 3. Установите бинарный файл

На **каждом** сервере получите бинарник из доверенного источника и проверьте опубликованную контрольную сумму SHA-256:

```bash
sha256sum ./tervon-bridge-linux-amd64
```

Сравните результат с контрольной суммой именно того релиза, который устанавливаете. После проверки:

```bash
sudo install -d -m 755 /opt/tervon-tcp-bridge
sudo install -m 755 ./tervon-bridge-linux-amd64 /opt/tervon-tcp-bridge/bridge
sudo install -d -m 750 /etc/tervon-tcp-bridge/pki
```

Команды предназначены для **новой установки**. Не выполняйте их поверх уже действующего бинарника без отдельного плана обновления.

## 4. Подготовьте сертификаты

Для межсерверного канала применяется **mTLS**: сервер проверяет сертификат клиента, а клиент — сертификат сервера. Сертификаты должны быть выпущены доверенным CA, а серверный сертификат — содержать DNS-имя `bridge.example.org` в SAN.

Расположите файлы на сервере A:

```text
/etc/tervon-tcp-bridge/pki/client.crt
/etc/tervon-tcp-bridge/pki/client.key
/etc/tervon-tcp-bridge/pki/server-ca.crt
```

На сервере B:

```text
/etc/tervon-tcp-bridge/pki/server.crt
/etc/tervon-tcp-bridge/pki/server.key
/etc/tervon-tcp-bridge/pki/client-ca.crt
```

Приватные ключи должны читаться только учётной записью соответствующей службы. Не помещайте их в репозиторий и не отправляйте вместе с диагностическими журналами.

Для авторизации клиента серверу нужен SHA-256 отпечаток **DER-представления клиентского сертификата**:

```bash
openssl x509 -in client.crt -outform DER | sha256sum
```

Полученные 64 шестнадцатеричных символа используются как ключ в объекте `authorized`.

## 5. Настройте Decapsulator на сервере B

Создайте `/etc/tervon-tcp-bridge/server.json`:

```json
{
  "mode": "server",
  "listen": "0.0.0.0:18443",
  "cert_file": "/etc/tervon-tcp-bridge/pki/server.crt",
  "key_file": "/etc/tervon-tcp-bridge/pki/server.key",
  "ca_file": "/etc/tervon-tcp-bridge/pki/client-ca.crt",
  "max_streams": 32,
  "routes": {
    "vpn": [
      { "Name": "local-vpn-backend", "Address": "127.0.0.1:19000" }
    ]
  },
  "authorized": {
    "REPLACE_WITH_CLIENT_CERT_DER_SHA256_64_HEX": ["vpn"]
  }
}
```

Замените отпечаток клиента действительным значением. Убедитесь, что backend действительно слушает `127.0.0.1:19000`. Откройте порт `18443/tcp` только для нужных источников в сетевом экране. Если используется обратный прокси, адрес `listen` можно оставить локальным и настроить прокси отдельно; пример выше рассчитан на прямое HTTPS/2-соединение с Bridge.

## 6. Настройте Encapsulator на сервере A

Создайте `/etc/tervon-tcp-bridge/client.json`:

```json
{
  "mode": "client",
  "listen": "127.0.0.1:19001",
  "cert_file": "/etc/tervon-tcp-bridge/pki/client.crt",
  "key_file": "/etc/tervon-tcp-bridge/pki/client.key",
  "ca_file": "/etc/tervon-tcp-bridge/pki/server-ca.crt",
  "server_name": "bridge.example.org",
  "remote_url": "https://bridge.example.org:18443/v1/stream",
  "service": "vpn",
  "max_local_connections": 32
}
```

`server_name` должен совпадать с DNS-именем в сертификате сервера. `remote_url` указывает на удалённый Decapsulator, а `service` — на один из маршрутов, разрешённых клиенту на сервере B.

Локальный `listen` привязан к `127.0.0.1`: соединяться с ним сможет приложение на том же сервере, например Xray.

## 7. Запустите службы через systemd

Для новой установки можно создать два отдельных unit-файла. Сначала подготовьте права на конфигурацию и ключи для выбранной учётной записи службы. Пример ниже использует выделенного системного пользователя `tervon-bridge` и предполагает, что он уже создан и имеет доступ на чтение необходимых файлов.

**Сервер B — `/etc/systemd/system/tervon-bridge-server.service`:**

```ini
[Unit]
Description=Tervon TCP Bridge Decapsulator
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=tervon-bridge
Group=tervon-bridge
ExecStart=/opt/tervon-tcp-bridge/bridge -config /etc/tervon-tcp-bridge/server.json
Restart=on-failure
RestartSec=3
NoNewPrivileges=true

[Install]
WantedBy=multi-user.target
```

**Сервер A — `/etc/systemd/system/tervon-bridge-client.service`:**

```ini
[Unit]
Description=Tervon TCP Bridge Encapsulator
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=tervon-bridge
Group=tervon-bridge
ExecStart=/opt/tervon-tcp-bridge/bridge -config /etc/tervon-tcp-bridge/client.json
Restart=on-failure
RestartSec=3
NoNewPrivileges=true

[Install]
WantedBy=multi-user.target
```

На новой машине создайте системную учётную запись (если её ещё нет):

```bash
sudo useradd --system --no-create-home --shell /usr/sbin/nologin tervon-bridge
```

Перед запуском назначьте владельца и права на конфигурации и ключи так, чтобы они были доступны `tervon-bridge`, но не другим пользователям. Например, для подготовленного каталога с сертификатами:

```bash
sudo chown -R root:tervon-bridge /etc/tervon-tcp-bridge
sudo find /etc/tervon-tcp-bridge -type d -exec chmod 750 {} +
sudo find /etc/tervon-tcp-bridge -type f -exec chmod 640 {} +
```

На сервере B запустите сначала backend, затем Decapsulator; на сервере A — Encapsulator:

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now tervon-bridge-server.service
```

На сервере A:

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now tervon-bridge-client.service
```

## 8. Подключите TCP-приложение

Укажите вашему локальному TCP-сервису на сервере A адрес `127.0.0.1:19001`. Для Xray это может быть адресом outbound-подключения. На сервере B данные поступят к backend `127.0.0.1:19000`.

Bridge не изменяет настройки внутреннего протокола: сертификаты и параметры VLESS/Trojan/другого TCP-сервиса остаются в ведении самого приложения.

## 9. Проверьте состояние

На соответствующих серверах:

```bash
systemctl is-active tervon-bridge-server.service
systemctl is-active tervon-bridge-client.service
journalctl -u tervon-bridge-server.service -n 30 --no-pager
journalctl -u tervon-bridge-client.service -n 30 --no-pager
```

Для проверки TLS-имени и доверия к серверу с сервера A:

```bash
openssl s_client \
  -connect bridge.example.org:18443 \
  -servername bridge.example.org \
  -verify_hostname bridge.example.org \
  -verify_return_error \
  -CAfile /etc/tervon-tcp-bridge/pki/server-ca.crt \
  -alpn h2 -brief </dev/null
```

Эта команда проверяет серверный сертификат. Полноценное прикладное подключение через Bridge требует клиентский сертификат и разрешённый сервис; его удобнее проверить через подключённое TCP-приложение.

## 10. Обслуживание

Храните конфигурации и ключи отдельно от бинарника. Перед обновлением сверяйте SHA-256 нового файла и готовьте отдельный путь для новой версии. Рабочую службу переключайте только в согласованное окно обслуживания. Журналы и состояние systemd обычно позволяют быстро определить, на какой стороне возникла проблема.

Подробности параметров: [Конфигурация](CONFIGURATION.ru.md). Порядок работы с релизами: [Обслуживание и релизы](RELEASE.ru.md).

---

**Примечание:** примеры рассчитаны на новую инфраструктуру. Существующие Homeserver и finserver работают со своими проверенными конфигурациями; установка по этому руководству не требует менять их.
