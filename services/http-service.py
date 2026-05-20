#!/usr/bin/env python3
"""
HTTP Service for Downtime Game
Each CT runs this service with its own color.
Serves: root (HTML UI) + /health (JSON) + /status (JSON detail)
"""

import os
import sys
import json
import time
import signal
from http.server import HTTPServer, BaseHTTPRequestHandler
from datetime import datetime, timezone

# ──────────────────────────────────────────────
# COLOR CONFIG — set via env or argument
# ──────────────────────────────────────────────
COLOR_CONFIG = {
    "rojo": {
        "css": "#DC143C",
        "css_bg": "#1a0a0f",
        "css_grad_from": "#2d0a14",
        "css_grad_to": "#1a0a0f",
        "name": "Rojo",
        "emoji": "🔴",
    },
    "azul": {
        "css": "#1E90FF",
        "css_bg": "#0a0f1a",
        "css_grad_from": "#0a142d",
        "css_grad_to": "#0a0f1a",
        "name": "Azul",
        "emoji": "🔵",
    },
    "verde": {
        "css": "#00FF7F",
        "css_bg": "#0a1a0f",
        "css_grad_from": "#0a2d14",
        "css_grad_to": "#0a1a0f",
        "name": "Verde",
        "emoji": "🟢",
    },
    "amarillo": {
        "css": "#FFD700",
        "css_bg": "#1a1a0a",
        "css_grad_from": "#2d2d0a",
        "css_grad_to": "#1a1a0a",
        "name": "Amarillo",
        "emoji": "🟡",
    },
}

SERVICE_NAME = os.environ.get("SERVICE_COLOR", "rojo").lower()
if SERVICE_NAME not in COLOR_CONFIG:
    print(f"Invalid color: {SERVICE_NAME}. Using rojo.")
    SERVICE_NAME = "rojo"

COLOR = COLOR_CONFIG[SERVICE_NAME]
START_TIME = time.time()
SERVICE_ID = os.environ.get("SERVICE_ID", f"sg-{SERVICE_NAME}")
PORT = int(os.environ.get("SERVICE_PORT", "8080"))


def uptime_str():
    secs = int(time.time() - START_TIME)
    days, secs = divmod(secs, 86400)
    hours, secs = divmod(secs, 3600)
    mins, secs = divmod(secs, 60)
    parts = []
    if days:
        parts.append(f"{days}d")
    parts.append(f"{hours:02d}:{mins:02d}:{secs:02d}")
    return " ".join(parts)


HTML_PAGE = f"""<!DOCTYPE html>
<html lang="es">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Servicio {COLOR['name']} — Downtime Game</title>
<style>
  *, *::before, *::after {{ margin: 0; padding: 0; box-sizing: border-box; }}

  body {{
    background: {COLOR['css_bg']};
    background: linear-gradient(135deg, {COLOR['css_grad_from']} 0%, {COLOR['css_grad_to']} 50%, #000000 100%);
    color: #ffffff;
    font-family: 'Courier New', Courier, 'Liberation Mono', monospace;
    min-height: 100vh;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    overflow: hidden;
    position: relative;
  }}

  /* ── Grid background ── */
  .grid {{
    position: fixed;
    top: 0; left: 0; right: 0; bottom: 0;
    background-image:
      linear-gradient(rgba(255,255,255,0.02) 1px, transparent 1px),
      linear-gradient(90deg, rgba(255,255,255,0.02) 1px, transparent 1px);
    background-size: 40px 40px;
    pointer-events: none;
    z-index: 0;
  }}

  /* ── Scanline overlay ── */
  .scanlines {{
    position: fixed;
    top: 0; left: 0; right: 0; bottom: 0;
    background: repeating-linear-gradient(
      0deg,
      transparent,
      transparent 2px,
      rgba(0,0,0,0.08) 2px,
      rgba(0,0,0,0.08) 4px
    );
    pointer-events: none;
    z-index: 1;
  }}

  /* ── Glow orbs ── */
  .orb {{
    position: fixed;
    border-radius: 50%;
    filter: blur(80px);
    opacity: 0.15;
    pointer-events: none;
    z-index: 0;
  }}
  .orb-1 {{
    width: 400px; height: 400px;
    background: {COLOR['css']};
    top: -100px; right: -100px;
    animation: orbFloat 8s ease-in-out infinite alternate;
  }}
  .orb-2 {{
    width: 300px; height: 300px;
    background: {COLOR['css']};
    bottom: -80px; left: -80px;
    animation: orbFloat 10s ease-in-out infinite alternate-reverse;
  }}

  @keyframes orbFloat {{
    0% {{ transform: translate(0, 0) scale(1); }}
    100% {{ transform: translate(30px, -30px) scale(1.1); }}
  }}

  /* ── Content ── */
  .content {{
    position: relative;
    z-index: 2;
    text-align: center;
    padding: 2rem;
  }}

  .emoji {{
    font-size: 4rem;
    display: block;
    margin-bottom: 0.5rem;
    animation: pulse 2s ease-in-out infinite;
  }}

  @keyframes pulse {{
    0%, 100% {{ transform: scale(1); opacity: 1; }}
    50% {{ transform: scale(1.05); opacity: 0.8; }}
  }}

  .service-name {{
    font-size: 4.5rem;
    font-weight: 700;
    color: {COLOR['css']};
    text-shadow:
      0 0 20px {COLOR['css']}44,
      0 0 40px {COLOR['css']}33,
      0 0 80px {COLOR['css']}22;
    letter-spacing: 0.5rem;
    margin-bottom: 0.5rem;
    text-transform: uppercase;
  }}

  .service-id {{
    font-size: 1rem;
    color: rgba(255,255,255,0.4);
    letter-spacing: 0.3rem;
    margin-bottom: 2rem;
  }}

  /* ── Status indicator ── */
  .status-row {{
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 0.75rem;
    margin-bottom: 2.5rem;
  }}

  .status-dot {{
    width: 14px;
    height: 14px;
    border-radius: 50%;
    background: {COLOR['css']};
    box-shadow: 0 0 15px {COLOR['css']};
    animation: dotPulse 1.5s ease-in-out infinite;
  }}

  @keyframes dotPulse {{
    0%, 100% {{ opacity: 1; transform: scale(1); }}
    50% {{ opacity: 0.5; transform: scale(0.85); }}
  }}

  .status-text {{
    font-size: 1.3rem;
    color: {COLOR['css']};
    letter-spacing: 0.4rem;
    text-transform: uppercase;
    font-weight: 700;
  }}

  /* ── Info grid ── */
  .info-grid {{
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 1px;
    background: rgba(255,255,255,0.06);
    border: 1px solid rgba(255,255,255,0.08);
    margin-bottom: 2rem;
    min-width: 340px;
  }}

  .info-item {{
    background: rgba(0,0,0,0.4);
    padding: 0.8rem 1.2rem;
    text-align: left;
  }}

  .info-label {{
    font-size: 0.65rem;
    color: rgba(255,255,255,0.35);
    text-transform: uppercase;
    letter-spacing: 0.15rem;
    margin-bottom: 0.25rem;
  }}

  .info-value {{
    font-size: 1.1rem;
    color: #ffffff;
  }}

  .info-value.highlight {{
    color: {COLOR['css']};
  }}

  /* ── Endpoints ── */
  .endpoints {{
    display: flex;
    gap: 1rem;
    justify-content: center;
    flex-wrap: wrap;
  }}

  .endpoint {{
    padding: 0.5rem 1rem;
    border: 1px solid rgba(255,255,255,0.1);
    border-radius: 4px;
    font-size: 0.75rem;
    color: rgba(255,255,255,0.4);
    text-decoration: none;
    transition: all 0.2s;
    background: rgba(255,255,255,0.03);
  }}

  .endpoint:hover {{
    border-color: {COLOR['css']}66;
    color: {COLOR['css']};
    background: rgba(255,255,255,0.06);
  }}

  .endpoint code {{
    color: rgba(255,255,255,0.6);
    font-family: inherit;
  }}

  /* ── Responsive ── */
  @media (max-width: 600px) {{
    .service-name {{ font-size: 2.5rem; letter-spacing: 0.25rem; }}
    .emoji {{ font-size: 2.5rem; }}
    .info-grid {{ grid-template-columns: 1fr; min-width: auto; }}
  }}
</style>
</head>
<body>
  <div class="grid"></div>
  <div class="scanlines"></div>
  <div class="orb orb-1"></div>
  <div class="orb orb-2"></div>

  <div class="content">
    <span class="emoji">{COLOR['emoji']}</span>
    <div class="service-name">{COLOR['name']}</div>
    <div class="service-id">{SERVICE_ID}</div>

    <div class="status-row">
      <div class="status-dot"></div>
      <span class="status-text">ONLINE</span>
    </div>

    <div class="info-grid">
      <div class="info-item">
        <div class="info-label">Uptime</div>
        <div class="info-value highlight" id="uptime">{uptime_str()}</div>
      </div>
      <div class="info-item">
        <div class="info-label">Puerto</div>
        <div class="info-value">{PORT}</div>
      </div>
      <div class="info-item">
        <div class="info-label">Timestamp</div>
        <div class="info-value" id="timestamp">{datetime.now(timezone.utc).strftime('%Y-%m-%d %H:%M:%S UTC')}</div>
      </div>
      <div class="info-item">
        <div class="info-label">Estado</div>
        <div class="info-value highlight">Saludable</div>
      </div>
    </div>

    <div class="endpoints">
      <a class="endpoint" href="/" target="_blank"><code>/</code> — Status page</a>
      <a class="endpoint" href="/health" target="_blank"><code>/health</code> — Health check</a>
    </div>
  </div>

  <script>
    function updateTime() {{
      fetch('/health')
        .then(r => r.json())
        .then(d => {{
          document.getElementById('uptime').textContent = d.uptime;
          document.getElementById('timestamp').textContent = d.timestamp.replace('T', ' ').replace('Z', ' UTC');
        }})
        .catch(() => {{}});
    }}
    setInterval(updateTime, 2000);
  </script>
</body>
</html>"""


class Handler(BaseHTTPRequestHandler):
    def do_GET(self):
        if self.path == "/health":
            self._json(200, {
                "status": "ok",
                "service": SERVICE_ID,
                "color": SERVICE_NAME,
                "timestamp": datetime.now(timezone.utc).isoformat(),
                "uptime": uptime_str(),
                "uptime_seconds": int(time.time() - START_TIME),
            })
        elif self.path == "/status":
            self._json(200, {
                "status": "ok",
                "service": SERVICE_ID,
                "color": SERVICE_NAME,
                "color_hex": COLOR["css"],
                "port": PORT,
                "started_at": datetime.fromtimestamp(START_TIME, tz=timezone.utc).isoformat(),
                "uptime": uptime_str(),
                "uptime_seconds": int(time.time() - START_TIME),
                "healthy": True,
            })
        elif self.path == "/":
            self._html(200, HTML_PAGE)
        else:
            self._json(404, {"status": "error", "message": "not found"})

    def _html(self, code, body):
        self.send_response(code)
        self.send_header("Content-Type", "text/html; charset=utf-8")
        self.send_header("Cache-Control", "no-cache, no-store, must-revalidate")
        self.end_headers()
        self.wfile.write(body.encode("utf-8"))

    def _json(self, code, data):
        self.send_response(code)
        self.send_header("Content-Type", "application/json")
        self.send_header("Access-Control-Allow-Origin", "*")
        self.send_header("Cache-Control", "no-cache, no-store, must-revalidate")
        self.end_headers()
        self.wfile.write(json.dumps(data).encode("utf-8"))

    def log_message(self, fmt, *args):
        sys.stderr.write(f"[{datetime.now().strftime('%H:%M:%S')}] {args[0]} {args[1]} {args[2]}\n")


def main():
    server = HTTPServer(("0.0.0.0", PORT), Handler)

    def shutdown(sig, frame):
        print("\nShutting down...")
        server.shutdown()
        sys.exit(0)

    signal.signal(signal.SIGTERM, shutdown)
    signal.signal(signal.SIGINT, shutdown)

    print(f"🚀 Servicio {COLOR['name']} iniciado en puerto {PORT}")
    print(f"   Service ID: {SERVICE_ID}")
    print(f"   Health:     http://0.0.0.0:{PORT}/health")
    print(f"   Status:     http://0.0.0.0:{PORT}/status")
    print(f"   Web:        http://0.0.0.0:{PORT}/")
    print(f"   PID: {os.getpid()}")

    server.serve_forever()


if __name__ == "__main__":
    main()
