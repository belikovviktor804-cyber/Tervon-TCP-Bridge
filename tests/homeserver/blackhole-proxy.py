#!/usr/bin/env python3
"""Loopback-only fault injector for isolated bridge tests; no host firewall changes."""
import asyncio
import pathlib
import sys

state = pathlib.Path(sys.argv[1])

async def relay(reader, writer):
    try:
        while True:
            data = await reader.read(65536)
            if not data:
                break
            if state.exists():
                continue  # Discard both directions without closing TCP sockets.
            writer.write(data)
            await writer.drain()
    except (OSError, ConnectionError):
        pass
    finally:
        try:
            writer.write_eof()
        except (OSError, AttributeError):
            pass

async def handle(reader, writer):
    try:
        upstream_reader, upstream_writer = await asyncio.open_connection("127.0.0.1", 18443)
    except OSError:
        writer.close()
        return
    await asyncio.gather(relay(reader, upstream_writer), relay(upstream_reader, writer))
    upstream_writer.close()
    writer.close()

async def main():
    server = await asyncio.start_server(handle, "127.0.0.1", 18444)
    async with server:
        await server.serve_forever()

asyncio.run(main())
