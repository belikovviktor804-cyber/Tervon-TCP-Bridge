import 'dart:async';
import 'dart:math' as math;

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

/// Read-only dashboard for the existing localhost Bridge /metrics/v1.
class BridgeMonitorScreen extends StatefulWidget {
  const BridgeMonitorScreen({super.key});
  @override
  State<BridgeMonitorScreen> createState() => _BridgeMonitorScreenState();
}

class _BridgeMonitorScreenState extends State<BridgeMonitorScreen> {
  static const _channel = MethodChannel('com.tervonvpndpi/vpn_service');
  static const _bg = Color(0xFF0B0E17);
  static const _panel = Color(0xFF141825);
  static const _border = Color(0xFF262C3C);
  static const _gold = Color(0xFFE3B665);
  static const _blue = Color(0xFF70A6FF);
  static const _green = Color(0xFF49D5AC);
  static const _red = Color(0xFFFF7886);
  static const _fg = Color(0xFFF2F3F7);
  static const _muted = Color(0xFF9DA6BB);

  Timer? _timer;
  bool _busy = false;
  bool _available = false;
  bool _binary = false;
  bool _environment = false;
  bool _connected = false;
  bool _rawExpanded = false;
  bool _loading = true;
  String _status = 'Получение данных…';
  Map<String, String> _data = const {};
  Map<String, String> _details = const {};
  final _upSamples = <double>[];
  final _downSamples = <double>[];
  DateTime? _sampledAt;

  @override
  void initState() {
    super.initState();
    _read();
    _timer = Timer.periodic(const Duration(seconds: 2), (_) => _read());
  }

  @override
  void dispose() {
    _timer?.cancel();
    super.dispose();
  }

  static double? _parse(Map<String, String> data, String name) {
    final n = double.tryParse(data[name] ?? '');
    return n != null && n.isFinite ? n : null;
  }

  double? metric(String name) => _available ? _parse(_data, name) : null;

  Future<void> _read() async {
    if (_busy) return;
    _busy = true;
    try {
      final result = await _channel.invokeMapMethod<String, dynamic>('getBridgeMonitoring');
      if (!mounted) return;
      final metrics = <String, String>{};
      if (result?['metrics'] is Map) {
        for (final e in (result!['metrics'] as Map).entries) {
          if (e.key is String && e.value != null) metrics[e.key as String] = e.value.toString();
        }
      }
      final details = <String, String>{};
      if (result?['details'] is Map) {
        for (final e in (result!['details'] as Map).entries) {
          if (e.key is String && e.value != null) details[e.key as String] = e.value.toString();
        }
      }
      final ok = result?['metricsAvailable'] == true;
      setState(() {
        _loading = false;
        _available = ok;
        _binary = result?['binaryPresent'] == true;
        _environment = result?['privateEnvironmentReady'] == true;
        _connected = result?['transportConnected'] == true;
        _status = result?['status']?.toString() ?? 'Нет данных';
        _data = ok ? metrics : const {};
        _details = ok ? details : const {};
        _sampledAt = ok ? DateTime.now() : null;
        if (!ok) {
          _upSamples.clear();
          _downSamples.clear();
        } else {
          final up = _parse(metrics, 'upload_bytes_per_second');
          final down = _parse(metrics, 'download_bytes_per_second');
          if (up != null && down != null) {
            _upSamples.add(math.max(0.0, up));
            _downSamples.add(math.max(0.0, down));
            if (_upSamples.length > 60) _upSamples.removeAt(0);
            if (_downSamples.length > 60) _downSamples.removeAt(0);
          }
        }
      });
    } catch (_) {
      if (mounted) setState(() {
        _loading = false;
        _available = false;
        _connected = false;
        _status = 'Локальная телеметрия недоступна';
        _data = const {};
        _details = const {};
        _sampledAt = null;
        _upSamples.clear();
        _downSamples.clear();
      });
    } finally {
      _busy = false;
    }
  }

  static String readable(double? n, {bool perSecond = false}) {
    if (n == null || !n.isFinite || n < 0) return '—';
    const units = ['Б', 'КБ', 'МБ', 'ГБ', 'ТБ'];
    var i = 0;
    while (n >= 1024 && i < units.length - 1) {
      n = n / 1024;
      i++;
    }
    final decimals = i == 0 || n >= 100 ? 0 : n >= 10 ? 1 : 2;
    return n.toStringAsFixed(decimals) + ' ' + units[i] + (perSecond ? '/с' : '');
  }

  String count(String name) => metric(name)?.toStringAsFixed(0) ?? '—';

  String latency(String name) {
    if ((metric('connect_latency_samples') ?? 0) == 0) return '—';
    final n = metric(name);
    return n == null ? '—' : n.toStringAsFixed(n < 10 ? 1 : 0) + ' мс';
  }

  Widget surface(Widget child, {EdgeInsetsGeometry padding = const EdgeInsets.all(17)}) =>
      Container(
        padding: padding,
        decoration: BoxDecoration(
          color: _panel,
          border: Border.all(color: _border),
          borderRadius: BorderRadius.circular(20),
        ),
        child: child,
      );

  Widget heading(String name, String subtitle) => Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(name, style: const TextStyle(
              color: _fg, fontSize: 17, fontWeight: FontWeight.w700)),
          const SizedBox(height: 4),
          Text(subtitle, style: const TextStyle(color: _muted, fontSize: 12)),
          const SizedBox(height: 15),
        ],
      );

  Widget line(String label, String value, {Color? valueColor}) => Padding(
        padding: const EdgeInsets.symmetric(vertical: 8),
        child: Row(children: [
          Expanded(child: Text(label, style: const TextStyle(color: _muted, fontSize: 12))),
          const SizedBox(width: 12),
          Flexible(child: Text(value, textAlign: TextAlign.end,
              style: TextStyle(color: valueColor ?? _fg, fontSize: 12,
                  fontWeight: FontWeight.w600))),
        ]),
      );

  Widget endpoint(IconData icon, String name, String subtitle, Color accent) =>
      Expanded(child: Column(children: [
        Container(
          width: 48, height: 48,
          decoration: BoxDecoration(
            color: accent.withOpacity(.12),
            borderRadius: BorderRadius.circular(15)),
          child: Icon(icon, color: accent),
        ),
        const SizedBox(height: 8),
        Text(name, style: const TextStyle(color: _fg, fontSize: 13,
            fontWeight: FontWeight.bold)),
        const SizedBox(height: 3),
        Text(subtitle, textAlign: TextAlign.center,
            style: const TextStyle(color: _muted, fontSize: 10)),
      ]));

  Widget topology() {
    final confirmed = _connected && _available;
    return surface(Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
      Row(children: [
        const Expanded(child: Text('Android → Server',
          style: TextStyle(color: _fg, fontSize: 21, fontWeight: FontWeight.w800))),
        Container(
          padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
          decoration: BoxDecoration(
            borderRadius: BorderRadius.circular(20),
            color: (confirmed ? _green : _muted).withOpacity(.12)),
          child: Row(mainAxisSize: MainAxisSize.min, children: [
            Icon(confirmed ? Icons.check_circle_outline : Icons.info_outline,
                color: confirmed ? _green : _muted, size: 13),
            const SizedBox(width: 5),
            Text(confirmed ? 'Подключено' : 'Не подтверждено',
              style: TextStyle(color: confirmed ? _green : _muted, fontSize: 11)),
          ]),
        ),
      ]),
      const SizedBox(height: 21),
      Row(children: [
        endpoint(Icons.smartphone_rounded, 'ANDROID',
            _binary ? 'Ядро найдено' : 'Нет ядра', _gold),
        Expanded(child: Column(children: [
          Icon(Icons.arrow_forward_rounded, color: confirmed ? _green : _muted),
          const SizedBox(height: 6),
          Text('HTTPS/2 · mTLS', textAlign: TextAlign.center,
            style: TextStyle(fontSize: 10, color: confirmed ? _green : _muted)),
          const SizedBox(height: 4),
          Text(confirmed ? 'Подтверждено' : 'Канал не измерен',
              textAlign: TextAlign.center,
              style: const TextStyle(fontSize: 9, color: _muted)),
        ])),
        endpoint(Icons.dns_rounded, 'SERVER', 'Удалённый Bridge', _blue),
      ]),
      const SizedBox(height: 18),
      const Divider(color: _border),
      Text(_status, style: const TextStyle(color: _muted, fontSize: 12)),
      const SizedBox(height: 7),
      const Text('Локальная телеметрия не подтверждает доставку трафика до сервера.',
          style: TextStyle(color: _muted, fontSize: 11)),
    ]));
  }

  Widget statCard(String label, String value, String hint, IconData icon, Color accent) =>
      surface(Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
        Row(children: [
          Icon(icon, color: accent, size: 19),
          const Spacer(),
          Text(hint, style: const TextStyle(color: _muted, fontSize: 10)),
        ]),
        const SizedBox(height: 14),
        Text(value, style: const TextStyle(color: _fg,
            fontSize: 23, fontWeight: FontWeight.w800)),
        const SizedBox(height: 6),
        Text(label, style: const TextStyle(color: _muted, fontSize: 12)),
      ]), padding: const EdgeInsets.all(14));

  Widget overview() => LayoutBuilder(builder: (context, constraints) {
    final width = (constraints.maxWidth - 12) / 2;
    return Wrap(spacing: 12, runSpacing: 12, children: [
      SizedBox(width: width, child: statCard('Отправка',
          readable(metric('upload_bytes_per_second'), perSecond: true),
          'Android →', Icons.north_east, _gold)),
      SizedBox(width: width, child: statCard('Получение',
          readable(metric('download_bytes_per_second'), perSecond: true),
          '← Server', Icons.south_west, _blue)),
      SizedBox(width: width, child: statCard('TCP-сессии',
          count('sessions_active'), 'Локально', Icons.hub_rounded, _green)),
      SizedBox(width: width, child: statCard('Задержка P95',
          latency('connect_latency_p95_ms'), 'Connect', Icons.speed_rounded, _gold)),
    ]);
  });

  Widget traffic() => surface(Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
    heading('Трафик в реальном времени', 'Последние измерения • интервал 2 секунды'),
    Row(children: [
      const Icon(Icons.circle, color: _gold, size: 9),
      const SizedBox(width: 6),
      const Text('Отправка', style: TextStyle(color: _muted, fontSize: 11)),
      const SizedBox(width: 16),
      const Icon(Icons.circle, color: _blue, size: 9),
      const SizedBox(width: 6),
      const Text('Получение', style: TextStyle(color: _muted, fontSize: 11)),
    ]),
    const SizedBox(height: 14),
    SizedBox(height: 155, width: double.infinity,
      child: _upSamples.length < 2
        ? const Center(child: Text('Ожидание реальных измерений',
            style: TextStyle(color: _muted, fontSize: 12)))
        : CustomPaint(painter: _TrafficGraph(_upSamples, _downSamples))),
    const SizedBox(height: 12),
    Row(children: [
      Expanded(child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
        const Text('Отправлено всего', style: TextStyle(color: _muted, fontSize: 11)),
        const SizedBox(height: 5),
        Text(readable(metric('upload_bytes')),
            style: const TextStyle(color: _fg, fontWeight: FontWeight.w700)),
      ])),
      Expanded(child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
        const Text('Получено всего', style: TextStyle(color: _muted, fontSize: 11)),
        const SizedBox(height: 5),
        Text(readable(metric('download_bytes')),
            style: const TextStyle(color: _fg, fontWeight: FontWeight.w700)),
      ])),
    ]),
  ]));

  Widget sessions() => surface(Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
    heading('TCP-сессии', 'Счётчики работающего ядра Bridge'),
    line('Создано', count('sessions_started')),
    line('Завершено', count('sessions_completed')),
    line('Отклонено', count('sessions_rejected')),
    line('Активных сейчас', count('sessions_active'), valueColor: _green),
    line('Подключений к backend: успех', count('backend_dial_success')),
    line('Подключений к backend: ошибка', count('backend_dial_failure')),
    line('Задержка P50', latency('connect_latency_p50_ms')),
    line('Задержка P99', latency('connect_latency_p99_ms')),
  ]));

  String feature(String key) {
    if (!_available) return '—';
    final value = _details[key];
    if (value == null) return 'Нет данных';
    if (value == 'NOT_IMPLEMENTED') return 'Не реализовано ядром';
    return value.replaceAll('_', ' ');
  }

  Widget security() => surface(Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
    heading('Протокол и маскировка', 'Только показатели, подтверждённые ядром'),
    line('TLS 1.3 / HTTP/2 / mTLS',
      metric('tls13_h2_mtls_requests') == null ? '—' :
      count('tls13_h2_mtls_requests') + ' запросов'),
    line('Adaptive Padding', feature('padding_state')),
    line('Jitter', feature('jitter_state')),
    line('Готовность локального ядра',
      !_available ? 'Нет данных' :
      _data['ready'] == 'true' ? 'READY' :
      _data['ready'] == 'false' ? 'NOT READY' : 'Нет данных'),
    line('Версия Bridge', _details['build_version'] ?? '—'),
    line('Время работы', metric('uptime_seconds') == null ? '—'
        : (metric('uptime_seconds')! ~/ 60).toString() + ' мин'),
  ]));

  Widget errors() {
    const errorKinds = <String, String>{
      'authorization': 'Авторизация',
      'tls_handshake': 'TLS handshake',
      'backend_dial': 'Backend dial',
      'http_status': 'HTTP статус',
      'timeout': 'Тайм-аут',
      'cancellation': 'Отмена',
      'shutdown': 'Завершение',
    };
    final keys = errorKinds.entries.where(
        (e) => metric('errors.' + e.key) != null).toList();
    return surface(Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
      heading('Ошибки и диагностика', 'Агрегированные показатели без данных пользователей'),
      if (keys.isEmpty) const Text('Нет доступных счётчиков ошибок',
          style: TextStyle(color: _muted, fontSize: 12))
      else ...keys.map((e) => line(e.value, count('errors.' + e.key),
          valueColor: (metric('errors.' + e.key) ?? 0) > 0 ? _red : _fg)),
      const Divider(color: _border),
      line('Ядро ARM64', _binary ? 'Найдено' : 'Не найдено'),
      line('Файловое окружение', _environment ? 'Готово' : 'Недоступно'),
      line('Локальные метрики', _available ? 'Доступны' : 'Нет данных'),
    ]));
  }

  Widget rawMetrics() => surface(Column(children: [
    InkWell(
      onTap: () => setState(() => _rawExpanded = !_rawExpanded),
      child: Row(children: [
        const Expanded(child: Text('Технические показатели',
          style: TextStyle(color: _fg, fontSize: 14, fontWeight: FontWeight.w700))),
        Icon(_rawExpanded ? Icons.expand_less : Icons.expand_more, color: _muted),
      ]),
    ),
    if (_rawExpanded) ...[
      const SizedBox(height: 12),
      if (_data.isEmpty) const Text('Метрики недоступны',
          style: TextStyle(color: _muted))
      else ..._data.entries.map((e) => line(e.key, e.value)),
    ],
  ]));

  @override
  Widget build(BuildContext context) {
    final t = _sampledAt;
    final last = t == null ? 'Нет свежих измерений' :
      'Обновлено ' + t.hour.toString().padLeft(2, '0') + ':' +
      t.minute.toString().padLeft(2, '0') + ':' +
      t.second.toString().padLeft(2, '0');
    return Scaffold(
      backgroundColor: _bg,
      appBar: AppBar(
        title: const Text('Bridge Monitoring'),
        foregroundColor: _fg, backgroundColor: _bg,
        actions: [IconButton(tooltip: 'Обновить',
          onPressed: _busy ? null : _read,
          icon: const Icon(Icons.refresh_rounded))],
      ),
      body: RefreshIndicator(
        onRefresh: _read, color: _gold, backgroundColor: _panel,
        child: ListView(
          physics: const AlwaysScrollableScrollPhysics(),
          padding: const EdgeInsets.fromLTRB(16, 10, 16, 30),
          children: [
            topology(),
            const SizedBox(height: 12),
            Row(children: [
              Icon(Icons.circle, color: _available ? _green : _muted, size: 7),
              const SizedBox(width: 7),
              Expanded(child: Text(_loading ? 'Загрузка…' : last,
                  style: const TextStyle(color: _muted, fontSize: 11))),
              const Text('Каждые 2 сек', style: TextStyle(color: _muted, fontSize: 11)),
            ]),
            const SizedBox(height: 14),
            overview(),
            const SizedBox(height: 14),
            traffic(),
            const SizedBox(height: 14),
            sessions(),
            const SizedBox(height: 14),
            security(),
            const SizedBox(height: 14),
            errors(),
            const SizedBox(height: 14),
            rawMetrics(),
            const SizedBox(height: 20),
            const Center(child: Text(
              'Мониторинг локальный • данные не отправляются с этой страницы',
              textAlign: TextAlign.center, style: TextStyle(color: _muted, fontSize: 11))),
          ],
        ),
      ),
    );
  }
}

/// Custom graph uses only observed speed samples. No chart dependency.
class _TrafficGraph extends CustomPainter {
  const _TrafficGraph(this.up, this.down);
  final List<double> up;
  final List<double> down;

  @override
  void paint(Canvas canvas, Size size) {
    final grid = Paint()..color = const Color(0xFF262C3C)..strokeWidth = 1;
    for (var n = 1; n <= 3; n++) {
      final y = size.height * n / 4;
      canvas.drawLine(Offset(0, y), Offset(size.width, y), grid);
    }
    final maximum = math.max(1.0, [...up, ...down].reduce(math.max));
    void draw(List<double> data, Color color) {
      if (data.length < 2) return;
      final path = Path();
      for (var i = 0; i < data.length; i++) {
        final x = i * size.width / (data.length - 1);
        final y = size.height - (data[i] / maximum) * (size.height - 6) - 3;
        if (i == 0) { path.moveTo(x, y); } else { path.lineTo(x, y); }
      }
      canvas.drawPath(path, Paint()..color = color..strokeWidth = 2.5
        ..strokeCap = StrokeCap.round..strokeJoin = StrokeJoin.round
        ..style = PaintingStyle.stroke);
    }
    draw(up, const Color(0xFFE3B665));
    draw(down, const Color(0xFF70A6FF));
  }

  @override
  bool shouldRepaint(covariant _TrafficGraph old) => true;
}
