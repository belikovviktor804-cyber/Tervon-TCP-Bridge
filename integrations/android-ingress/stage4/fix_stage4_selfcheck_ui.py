#!/usr/bin/env python3
"""Guarded single-file fix for Stage4 CSR button feedback."""
import argparse
import os
from pathlib import Path
import tempfile

REL = Path("lib/features/home/bridge_settings_screen.dart")
OLD_STATE = """  bool _enabled = false;
  bool _busy = false;
  String _message = 'Конфигурация ещё не загружена';
"""
NEW_STATE = """  bool _enabled = false;
  bool _busy = false;
  bool _stage4Busy = false;
  String _stage4Result = '';
  String _message = 'Конфигурация ещё не загружена';
"""
OLD_METHOD = """  Future<void> _checkStage4Identity() async {
    if (_busy) return;
    setState(() { _busy = true; _message = 'Проверка ключа устройства и CSR…'; });
    try {
      final data = await _channel.invokeMapMethod<String, dynamic>('stage4IdentitySelfCheck');
      if (mounted) setState(() {
        _message = data?['result'] == 'PASS'
            ? 'Stage 4: PASS — ключ устройства, подпись и CSR'
            : 'Stage 4: FAIL — самопроверка не пройдена';
      });
    } on PlatformException catch (e) {
      if (mounted) setState(() => _message = 'Stage 4: ${e.code}');
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }
"""
NEW_METHOD = """  Future<void> _checkStage4Identity() async {
    if (_busy || _stage4Busy) return;
    setState(() {
      _stage4Busy = true;
      _stage4Result = 'Проверка ключа устройства и CSR…';
    });
    try {
      final data =
          await _channel.invokeMapMethod<String, dynamic>('stage4IdentitySelfCheck');
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
"""
OLD_BUTTON = """          OutlinedButton(
            onPressed: _busy ? null : _checkStage4Identity,
            child: const Text('Проверить ключ и CSR (Stage 4)'),
          ),"""
NEW_BUTTON = """          OutlinedButton(
            onPressed: (_busy || _stage4Busy) ? null : _checkStage4Identity,
            child: const Text('Проверить ключ и CSR (Stage 4)'),
          ),
          if (_stage4Result.isNotEmpty) ...[
            const SizedBox(height: 8),
            SelectableText(
              _stage4Result,
              style: const TextStyle(color: Colors.white70),
            ),
          ],"""

def replace_once(s, old, new, label):
    if s.count(old) != 1:
        raise ValueError(f"{label}: original block not found exactly once")
    return s.replace(old, new, 1)

def main():
    p = argparse.ArgumentParser()
    p.add_argument("--app-root", type=Path, required=True)
    group = p.add_mutually_exclusive_group(required=True)
    group.add_argument("--check", action="store_true")
    group.add_argument("--apply", action="store_true")
    a = p.parse_args()
    target = a.app_root / REL
    if not target.is_file() or target.is_symlink():
        p.exit(2, "STAGE4_UI=BLOCKED missing or symlink target\n")
    original = target.read_bytes()
    try:
        s = original.decode("utf-8")
        s = replace_once(s, OLD_STATE, NEW_STATE, "state")
        s = replace_once(s, OLD_METHOD, NEW_METHOD, "method")
        s = replace_once(s, OLD_BUTTON, NEW_BUTTON, "button")
        if s.count("stage4IdentitySelfCheck") != 1:
            raise ValueError("unexpected method channel references")
    except (UnicodeError, ValueError) as e:
        p.exit(2, f"STAGE4_UI=BLOCKED {e}\n")
    print("STAGE4_UI_PATCH=READY")
    print("STAGE4_UI_CHANGED_FILES=1")
    if a.check:
        print("STAGE4_UI_FILES_UNCHANGED=YES")
        return
    fd, tmp = tempfile.mkstemp(prefix=".stage4-ui-", dir=target.parent)
    try:
        with os.fdopen(fd, "wb") as out:
            out.write(s.encode("utf-8"))
            out.flush()
            os.fsync(out.fileno())
        os.chmod(tmp, target.stat().st_mode & 0o777)
        if target.read_bytes() != original:
            p.exit(2, "STAGE4_UI=BLOCKED concurrent change\n")
        os.replace(tmp, target)
        print("STAGE4_UI_PATCH=APPLIED")
        print("STAGE4_UI_BUILD=NOT_RUN")
    finally:
        if os.path.exists(tmp):
            os.unlink(tmp)

if __name__ == "__main__":
    main()
