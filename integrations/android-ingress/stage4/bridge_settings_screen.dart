import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'bridge_runtime_screen.dart';
import '../../vpn/xray_service.dart';

/// Automatic Bridge enrollment for the current HTTPS subscription.
/// No manual destination fields or credential entry.
class BridgeSettingsScreen extends StatefulWidget {
  const BridgeSettingsScreen({super.key});

  @override
  State<BridgeSettingsScreen> createState() => _BridgeSettingsScreenState();
}

class _BridgeSettingsScreenState extends State<BridgeSettingsScreen> {
  static const _channel = MethodChannel('com.tervonvpndpi/bridge_control');
  static const _background = Color(0xFF0B0E17);
  static const _card = Color(0xFF141825);
  static const _gold = Color(0xFFD4A853);
  bool _enabled = false;
  bool _busy = false;
  bool _stage4Busy = false;
  String _stage4Result = '';
  String _message = 'Проверка состояния Bridge…';
  String _cliHelp = '';

  @override
  void initState() { super.initState(); _load(); }

  Future<void> _load() async {
    try {
      final data = await _channel.invokeMapMethod<String, dynamic>('getSettings');
      if (!mounted) return;
      setState(() {
        _enabled = data?['enabled'] == true;
        _message = data?['runtimeIntegrated'] == true
            ? 'Bridge: локальная TCP-маршрутизация активна'
            : 'Bridge: маршрутизация пока не активна';
      });
    } catch (_) {
      if (mounted) setState(() => _message = 'Не удалось прочитать состояние Bridge');
    }
  }

  Future<void> _toggleBypass(bool enable) async {
    if (_busy) return;
    setState(() { _busy = true; _message = 'Проверка Bridge…'; });
    try {
      String? url;
      if (enable) {
        url = await XrayService.getActiveSubscriptionRaw();
        final parsed = Uri.tryParse(url ?? '');
        if (parsed == null || parsed.scheme != 'https' || parsed.host.isEmpty ||
            parsed.userInfo.isNotEmpty) {
          if (mounted) setState(() =>
              _message = 'Нет выбранной HTTPS-подписки для автоматической регистрации');
          return;
        }
      }
      final data = await _channel.invokeMapMethod<String, dynamic>(
        'setBypassEnabled', {'enabled': enable, if (url != null) 'subscriptionUrl': url});
      if (!mounted) return;
      final state = data?['state']?.toString() ?? 'UNKNOWN';
      setState(() {
        _enabled = data?['enabled'] == true;
        _message = switch (state) {
          'DISABLED' => 'Обход DPI выключен',
          'ACTIVE' => 'Обход DPI: активен локальный маршрут Bridge',
          'PREPARED' => 'Сертификат готов. Bridge включится при следующем подключении VPN.',
          'DIRECT_FALLBACK' => 'Bridge недоступен: восстановлено прямое подключение VPN',
          'RESTART_FAILED' => 'Переключение не завершено: VPN остановлен для безопасности',
          'NO_BRIDGE' => 'На сервере текущей подписки Bridge отсутствует',
          'CANCELLED' => 'Регистрация отменена',
          'ENROLLMENT_FAILED' => 'Ошибка автоматической регистрации; прямой VPN не изменён',
          _ => 'Неизвестный результат регистрации Bridge',
        };
      });
    } on PlatformException catch (e) {
      if (mounted) setState(() {
        _enabled = false;
        _message = 'Автоматическая регистрация: ${e.code}';
      });
    } catch (_) {
      if (mounted) setState(() {
        _enabled = false;
        _message = 'Автоматическая регистрация Bridge не выполнена';
      });
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  Future<void> _checkStage4Identity() async {
    if (_busy || _stage4Busy) return;
    setState(() {
      _stage4Busy = true;
      _stage4Result = 'Проверка ключа устройства и CSR…';
    });
    try {
      final data = await _channel.invokeMapMethod<String, dynamic>('stage4IdentitySelfCheck');
      if (!mounted) return;
      setState(() {
        _stage4Result = data?['result'] == 'PASS'
            ? 'Stage 4: PASS — ключ устройства, подпись и CSR'
            : 'Stage 4: FAIL — самопроверка не пройдена';
      });
    } on PlatformException catch (e) {
      if (mounted) {
        setState(() {
          final detail = e.message?.trim();
          _stage4Result = (detail == null || detail.isEmpty)
              ? 'Stage 4: ${e.code}'
              : 'Stage 4: ${e.code} ($detail)';
        });
      }
    } catch (_) {
      if (mounted) setState(() => _stage4Result = 'Stage 4: UNEXPECTED_ERROR');
    } finally {
      if (mounted) setState(() => _stage4Busy = false);
    }
  }

  Future<void> _inspect() async {
    if (_busy) return;
    setState(() { _busy = true; _message = 'Чтение справки ядра…'; });
    try {
      final data = await _channel.invokeMapMethod<String, dynamic>('inspectCore');
      if (mounted) setState(() {
        _cliHelp = data?['output']?.toString() ?? '';
        _message = data?['message']?.toString() ?? 'Нет ответа';
      });
    } catch (_) {
      if (mounted) setState(() => _message = 'Ошибка запуска диагностики');
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  Widget _flowStep(IconData icon, String title, String subtitle) => Container(
    width: double.infinity,
    padding: const EdgeInsets.symmetric(vertical: 12, horizontal: 14),
    decoration: BoxDecoration(
      color: const Color(0xFF1B2030),
      borderRadius: BorderRadius.circular(12),
      border: Border.all(color: _gold.withOpacity(0.35)),
    ),
    child: Row(children: [
      Icon(icon, color: _gold, size: 26),
      const SizedBox(width: 12),
      Expanded(child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(title, style: const TextStyle(
              color: Colors.white, fontWeight: FontWeight.w600)),
          const SizedBox(height: 3),
          Text(subtitle, style: const TextStyle(
              color: Colors.white70, fontSize: 12)),
        ],
      )),
    ]),
  );

  Widget _flowArrow() => const Center(
    child: Padding(
      padding: EdgeInsets.symmetric(vertical: 4),
      child: Icon(Icons.arrow_downward, color: _gold, size: 20),
    ),
  );

  Widget _connectionDiagram() => Column(
    crossAxisAlignment: CrossAxisAlignment.start,
    children: [
      const Text('Как работает автоматический обход',
        style: TextStyle(color: _gold, fontSize: 16,
            fontWeight: FontWeight.w600)),
      const SizedBox(height: 12),
      _flowStep(Icons.smartphone, 'TervonVPN',
          'Использует текущую выбранную HTTPS-подписку'),
      _flowArrow(),
      _flowStep(Icons.verified_user_outlined, 'Регистрация на сервере подписки',
          'Обнаружение Bridge, сертификат устройства и разрешение ACL'),
      _flowArrow(),
      _flowStep(Icons.lock_outline, 'Защищённый Bridge',
          'Параметры подключения выдаются сервером автоматически; TLS 1.3 и mTLS'),
      _flowArrow(),
      _flowStep(Icons.route_outlined, 'VLESS TCP',
          'Маршрут выбранного профиля через Bridge; при ошибке — прямое подключение'),
      const SizedBox(height: 10),
      const Text('Адреса, порты и сертификаты вручную указывать не нужно.',
        style: TextStyle(color: Colors.white60, fontSize: 12)),
    ],
  );

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: _background,
      appBar: AppBar(title: const Text('Обход DPI'),
          backgroundColor: _background, foregroundColor: Colors.white),
      body: ListView(padding: const EdgeInsets.all(16), children: [
        Card(color: _card, child: Padding(
          padding: const EdgeInsets.all(16),
          child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
            const Text('Tervon TCP Bridge v1.0.1',
                style: TextStyle(color: _gold, fontSize: 17)),
            const SizedBox(height: 10),
            const Text('Bridge включается только после регистрации сертификата и проверки локального TCP-маршрута.',
                style: TextStyle(color: Colors.white70)),
            SwitchListTile(
              title: const Text('Обход DPI', style: TextStyle(color: Colors.white)),
              subtitle: const Text('Автоматическая регистрация через текущую HTTPS-подписку; при ошибке — прямой маршрут',
                  style: TextStyle(color: Colors.white60)),
              value: _enabled, onChanged: _busy ? null : _toggleBypass,
            ),
            const SizedBox(height: 12),
            _connectionDiagram(),
          ]),
        )),
        OutlinedButton.icon(
          icon: const Icon(Icons.play_circle_outline),
          onPressed: () => Navigator.push(context, MaterialPageRoute(
            builder: (_) => const BridgeRuntimeScreen(),
          )),
          label: const Text('Управление процессом Bridge'),
        ),
        const SizedBox(height: 12),
        Card(color: _card, child: Padding(
          padding: const EdgeInsets.all(16),
          child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
            const Text('Диагностика ядра',
                style: TextStyle(color: _gold, fontSize: 16)),
            const SizedBox(height: 8),
            OutlinedButton(onPressed: _busy ? null : _inspect,
              child: const Text('Показать параметры запуска ядра')),
            OutlinedButton(
              onPressed: (_busy || _stage4Busy) ? null : _checkStage4Identity,
              child: const Text('Проверить ключ и CSR (Stage 4)'),
            ),
            if (_stage4Result.isNotEmpty) ...[
              const SizedBox(height: 8),
              SelectableText(_stage4Result,
                  style: const TextStyle(color: Colors.white70)),
            ],
            const SizedBox(height: 8),
            SelectableText(_message, style: const TextStyle(color: Colors.white70)),
            if (_cliHelp.isNotEmpty) ...[
              const SizedBox(height: 8),
              SelectableText(_cliHelp, style: const TextStyle(
                  color: Colors.white70, fontSize: 12)),
              TextButton(
                onPressed: () => Clipboard.setData(ClipboardData(text: _cliHelp)),
                child: const Text('Копировать справку'),
              ),
            ],
          ]),
        )),
      ]),
    );
  }
}
