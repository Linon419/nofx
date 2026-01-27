#!/usr/bin/env python3
"""
Telegram Bullindex Server

Fetches top coins from @coinarch_bot /bullindex command and exposes them via HTTP API.
The nofx system can consume this API using source_type: "external".

Usage:
    python bullindex_server.py --port 8765

Configuration:
    Set environment variables:
    - TELEGRAM_API_ID: Your Telegram API ID (get from https://my.telegram.org)
    - TELEGRAM_API_HASH: Your Telegram API Hash
    - TELEGRAM_SESSION: Session name (default: "bullindex_session")

API Endpoints:
    GET /coins - Returns JSON: {"coins": ["BTC", "ETH", ...], "updated_at": "..."}
    GET /health - Health check

Example nofx config:
    {
        "coin_source": {
            "source_type": "external",
            "external_coins_url": "http://localhost:8765/coins"
        }
    }
"""

import asyncio
import json
import os
import re
import sys
import argparse
from datetime import datetime
from http.server import HTTPServer, BaseHTTPRequestHandler
from threading import Thread, Lock
from typing import Optional

try:
    from telethon import TelegramClient
    from telethon.tl.types import Message
except ImportError:
    print("Error: telethon not installed. Run: pip install telethon")
    sys.exit(1)

try:
    from dotenv import load_dotenv
    # Try loading .env from multiple locations
    script_dir = os.path.dirname(os.path.abspath(__file__))
    project_root = os.path.dirname(script_dir)

    env_loaded = False
    for env_path in [
        os.path.join(script_dir, ".env"),      # Same folder as script
        os.path.join(project_root, ".env"),    # Project root
        ".env",                                 # Current working directory
    ]:
        if os.path.exists(env_path):
            load_dotenv(env_path)
            print(f"Loaded .env from: {env_path}")
            env_loaded = True
            break

    if not env_loaded:
        print(f"Warning: .env not found")
except ImportError:
    print("Note: python-dotenv not installed, using system env only")


class BullindexCache:
    """Thread-safe cache for bullindex data."""

    def __init__(self):
        self._coins: list[str] = []
        self._updated_at: Optional[str] = None
        self._lock = Lock()

    def update(self, coins: list[str]):
        with self._lock:
            self._coins = coins
            self._updated_at = datetime.utcnow().isoformat() + "Z"

    def get(self) -> dict:
        with self._lock:
            return {
                "coins": self._coins.copy(),
                "updated_at": self._updated_at,
                "count": len(self._coins),
            }


# Global cache
cache = BullindexCache()


def parse_bullindex_message(text: str, limit: int = 5) -> list[str]:
    """
    Parse bullindex message to extract coin symbols.

    Example input:
    今日多头指数排行

    `1.ONG (19.61%)    70.0分
    2.ONT (6.02%)     36.2分
    ...`
    """
    coins = []
    # Match pattern: "1.ONG" or "1.ONG (19.61%)"
    pattern = r"(\d+)\.([A-Z0-9]+)\s*\("
    matches = re.findall(pattern, text.upper())

    if matches:
        # Sort by rank number and take top N
        matches.sort(key=lambda x: int(x[0]))
        coins = [m[1] for m in matches[:limit]]

    return coins


class BullindexHandler(BaseHTTPRequestHandler):
    """HTTP request handler for bullindex API."""

    def log_message(self, format, *args):
        # Suppress default logging
        pass

    def do_GET(self):
        if self.path == "/coins" or self.path == "/":
            data = cache.get()
            self.send_response(200)
            self.send_header("Content-Type", "application/json")
            self.send_header("Access-Control-Allow-Origin", "*")
            self.end_headers()
            self.wfile.write(json.dumps(data).encode())
        elif self.path == "/health":
            self.send_response(200)
            self.send_header("Content-Type", "application/json")
            self.end_headers()
            self.wfile.write(json.dumps({"status": "ok"}).encode())
        else:
            self.send_error(404)


async def fetch_bullindex(client: TelegramClient, limit: int = 5) -> list[str]:
    """Fetch bullindex data from @coinarch_bot."""
    bot_username = "coinarch_bot"

    try:
        # Send /bullindex command
        await client.send_message(bot_username, "/bullindex")
        print(f"[{datetime.now()}] Sent /bullindex to @{bot_username}")

        # Wait for response
        await asyncio.sleep(5)

        # Get recent messages from the bot
        messages = await client.get_messages(bot_username, limit=5)
        for msg in messages:
            if isinstance(msg, Message) and msg.text:
                print(f"[DEBUG] Message: {repr(msg.text[:500])}")  # Debug output
                coins = parse_bullindex_message(msg.text, limit)
                if coins:
                    print(f"[{datetime.now()}] Parsed coins: {coins}")
                    return coins

        print(f"[{datetime.now()}] No valid bullindex response found")
        return []

    except Exception as e:
        print(f"[{datetime.now()}] Error fetching bullindex: {e}")
        return []


async def update_loop(
    api_id: int, api_hash: str, session: str, limit: int, interval: int
):
    """Background loop to periodically update bullindex data."""
    client = TelegramClient(session, api_id, api_hash)

    await client.start()
    print(f"[{datetime.now()}] Telegram client connected")

    while True:
        try:
            coins = await fetch_bullindex(client, limit)
            if coins:
                cache.update(coins)
                print(f"[{datetime.now()}] Cache updated: {coins}")
        except Exception as e:
            print(f"[{datetime.now()}] Update error: {e}")

        await asyncio.sleep(interval)


def run_http_server(port: int):
    """Run HTTP server in a separate thread."""
    server = HTTPServer(("0.0.0.0", port), BullindexHandler)
    print(f"[{datetime.now()}] HTTP server started on port {port}")
    server.serve_forever()


def main():
    parser = argparse.ArgumentParser(description="Telegram Bullindex Server")
    parser.add_argument("--port", type=int, default=8765, help="HTTP server port")
    parser.add_argument(
        "--limit", type=int, default=5, help="Number of top coins to fetch"
    )
    parser.add_argument(
        "--interval",
        type=int,
        default=1800,
        help="Update interval in seconds (default: 30 min)",
    )
    args = parser.parse_args()

    # Get Telegram credentials from environment
    api_id = os.environ.get("TELEGRAM_API_ID")
    api_hash = os.environ.get("TELEGRAM_API_HASH")
    session = os.environ.get("TELEGRAM_SESSION", "bullindex_session")

    if not api_id or not api_hash:
        print("Error: TELEGRAM_API_ID and TELEGRAM_API_HASH environment variables required")
        print("Get them from https://my.telegram.org")
        sys.exit(1)

    # Start HTTP server in background thread
    http_thread = Thread(target=run_http_server, args=(args.port,), daemon=True)
    http_thread.start()

    # Run Telegram update loop
    asyncio.run(
        update_loop(int(api_id), api_hash, session, args.limit, args.interval)
    )


if __name__ == "__main__":
    main()
