#!/usr/bin/env python3
"""Verify the local chat contract while external proxy/network names are unusable."""
import argparse, json, os, pathlib, re, socketserver, subprocess, tempfile, threading
from http.server import BaseHTTPRequestHandler

ROOT = pathlib.Path(__file__).resolve().parents[1]

class Handler(BaseHTTPRequestHandler):
    def log_message(self, *_):
        pass
    def do_GET(self):
        payload = {"status": "ok"} if self.path == "/health" else {"data": [{"id": "offline-fixture"}]}
        raw = json.dumps(payload).encode()
        self.send_response(200); self.send_header("Content-Length", str(len(raw))); self.end_headers(); self.wfile.write(raw)
    def do_POST(self):
        length = int(self.headers.get("Content-Length", "0")); _ = self.rfile.read(length)
        raw = json.dumps({"choices": [{"message": {"role": "assistant", "content": "offline local response"}}], "usage": {"prompt_tokens": 3, "completion_tokens": 3}}).encode()
        self.send_response(200); self.send_header("Content-Length", str(len(raw))); self.end_headers(); self.wfile.write(raw)

def main():
    parser = argparse.ArgumentParser(); parser.add_argument("--output", default="docs/validation/offline-contract.json"); args = parser.parse_args()
    server = socketserver.ThreadingTCPServer(("127.0.0.1", 0), Handler)
    threading.Thread(target=server.serve_forever, daemon=True).start()
    try:
        with tempfile.TemporaryDirectory(prefix="dotaman-offline-") as td:
            cfg = pathlib.Path(td) / "config.toml"
            cfg.write_text(f'endpoint="http://127.0.0.1:{server.server_address[1]}"\ndata_dir="{td}/data"\nroot="{td}"\n')
            trace = pathlib.Path(td) / "connect.trace"
            env = {**os.environ, "HTTP_PROXY": "http://192.0.2.1:9", "HTTPS_PROXY": "http://192.0.2.1:9", "ALL_PROXY": "http://192.0.2.1:9"}
            command = ["strace", "-f", "-e", "trace=connect", "-o", str(trace), str(ROOT / "bin/dotamanai-codex"), "-config", str(cfg), "-json", "chat", "offline test"]
            result = subprocess.run(command, cwd=ROOT, env=env, capture_output=True, text=True, timeout=30)
            if result.returncode != 0:
                raise SystemExit(result.stderr or result.stdout)
            response = json.loads(result.stdout)
            lines = trace.read_text().splitlines()
            remote = [line for line in lines if "AF_INET" in line and '127.0.0.1' not in line]
            report = {"scope": "local app with unusable external proxies", "chat": response.get("text"), "connects": len(lines), "remote_ipv4_connects": remote, "passed": bool(response.get("text")) and not remote}
            pathlib.Path(args.output).write_text(json.dumps(report, ensure_ascii=False, indent=2) + "\n")
            print(json.dumps(report, ensure_ascii=False, indent=2))
            if not report["passed"]:
                raise SystemExit(1)
    finally:
        server.shutdown(); server.server_close()

if __name__ == "__main__":
    main()
