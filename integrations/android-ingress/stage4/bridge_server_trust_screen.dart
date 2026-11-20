import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

/// Manual per-server public-key confirmation. No network and no VPN changes.
class BridgeServerTrustScreen extends StatefulWidget {
  const BridgeServerTrustScreen({super.key});
  @override
  State<BridgeServerTrustScreen> createState() => _BridgeServerTrustScreenState();
}

class _BridgeServerTrustScreenState extends State<BridgeServerTrustScreen> {
  static const _channel = MethodChannel('com.tervonvpndpi/bridge_control');
  final _serverId = TextEditingController();
  final _keyId = TextEditingController();
  final _publicKey = TextEditingController();
  final _fingerprint = TextEditingController();
  bool _confirmed = false, _busy = false;
  String _result = '';

  @override
  void dispose() {
    for (final c in [_serverId, _keyId, _publicKey, _fingerprint]) { c.dispose(); }
    super.dispose();
  }

  Future<void> _save() async {
    if (_busy || !_confirmed) return;
    setState(() { _busy = true; _result = 'Проверка ключа…'; });
    try {
      final data = await _channel.invokeMapMethod<String, dynamic>(
        'confirmServerCatalogKey', {
          'serverId': _serverId.text.trim(),
          'keyId': _keyId.text.trim(),
          'publicKeyBase64': _publicKey.text.trim(),
          'verifiedFingerprintSha256': _fingerprint.text.trim(),
          'explicitIndependentConfirmation': true,
        },
      );
      if (mounted) setState(() {
        _result = data?['result'] == 'PINNED'
            ? 'Публичный ключ сервера закреплён. Bridge не включён.'
            : 'Ключ не был закреплён.';
      });
    } on PlatformException catch (e) {
      if (mounted) setState(() => _result = 'Отказ: ${e.code}');
    } catch (_) {
      if (mounted) setState(() => _result = 'Ошибка подтверждения ключа.');
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  Widget _field(TextEditingController c, String label, {int lines = 1}) =>
      Padding(
        padding: const EdgeInsets.only(bottom: 12),
        child: TextField(
          controller: c,
          maxLines: lines,
          enabled: !_busy,
          autocorrect: false,
          enableSuggestions: false,
          decoration: InputDecoration(
            labelText: label,
            border: const OutlineInputBorder(),
          ),
        ),
      );

  @override
  Widget build(BuildContext context) => Scaffold(
    appBar: AppBar(title: const Text('Доверие к Bridge-серверу')),
    body: ListView(padding: const EdgeInsets.all(16), children: [
      const Text('Закрепление публичного ключа',
          style: TextStyle(fontSize: 18, fontWeight: FontWeight.bold)),
      const SizedBox(height: 12),
      const Text(
        'Получите отпечаток SHA-256 через отдельный доверенный канал '
        'администратора панели Remnawave или 3x-ui. '
        'Отпечаток из того же непроверенного ответа, что и ключ, '
        'не подтверждает подлинность сервера.',
      ),
      const SizedBox(height: 16),
      _field(_serverId, 'Подтверждённый ID сервера'),
      _field(_keyId, 'ID ключа каталога'),
      _field(_publicKey, 'Публичный Ed25519 ключ Base64', lines: 2),
      _field(_fingerprint, 'Независимо проверенный SHA-256 (64 hex)', lines: 2),
      CheckboxListTile(
        value: _confirmed,
        onChanged: _busy ? null : (v) => setState(() => _confirmed = v == true),
        title: const Text('Я независимо сверил отпечаток ключа'),
        subtitle: const Text('Не регистрирует устройство и не включает VPN Bridge.'),
        controlAffinity: ListTileControlAffinity.leading,
      ),
      FilledButton(
        onPressed: (_busy || !_confirmed) ? null : _save,
        child: const Text('Закрепить ключ сервера'),
      ),
      if (_result.isNotEmpty) ...[
        const SizedBox(height: 12),
        SelectableText(_result),
      ],
    ]),
  );
}
