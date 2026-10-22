#!/usr/bin/env python3
"""Development-only PTY/CLI integration with an explicit local HTTP fixture, not GGUF inference."""
import fcntl, http.server, json, os, pathlib, pty, select, struct, subprocess, tempfile, termios, threading, time
ROOT=pathlib.Path(__file__).resolve().parents[1]
class Handler(http.server.BaseHTTPRequestHandler):
    def log_message(self,*args): pass
    def do_GET(self):
        self.send_response(200); self.end_headers()
        self.wfile.write(json.dumps({'status':'ok'} if self.path=='/health' else {'data':[{'id':'integration-fixture'}]}).encode())
    def do_POST(self):
        p=json.loads(self.rfile.read(int(self.headers['Content-Length'])))
        self.send_response(200); self.end_headers()
        if p.get('stream'):
            self.wfile.write(('data: '+json.dumps({'choices':[{'delta':{'content':'أمان Linux fixture'}}]},ensure_ascii=False)+'\n\ndata: [DONE]\n\n').encode())
        else:
            self.wfile.write(json.dumps({'choices':[{'message':{'role':'assistant','content':'أمان Linux fixture'}}], 'usage':{'prompt_tokens':10,'completion_tokens':4}},ensure_ascii=False).encode())
def main():
    server=http.server.ThreadingHTTPServer(('127.0.0.1',0),Handler)
    threading.Thread(target=server.serve_forever,daemon=True).start()
    try:
        with tempfile.TemporaryDirectory() as td:
            cfg=pathlib.Path(td)/'config.toml'; cfg.write_text(f'endpoint="http://127.0.0.1:{server.server_port}"\ndata_dir="{td}/data"\nroot="{td}"\n')
            base=[str(ROOT/'bin/dotamanai-codex'),'-config',str(cfg)]
            subprocess.run(base+['doctor'],check=True,capture_output=True)
            o=json.loads(subprocess.run(base+['-json','chat','اختبار Linux'],check=True,capture_output=True).stdout)
            assert o['text']=='أمان Linux fixture' and o['usage']['completion_tokens']==4
            messages=json.loads(subprocess.run(base+['export-session',o['session']],check=True,capture_output=True).stdout)
            assert len(messages)==2 and messages[0]['content']=='اختبار Linux'
            doc=pathlib.Path(td)/'fixture.md';doc.write_text('أَمَان Linux: least privilege and prepared statements.')
            subprocess.run(base+['import',str(doc)],check=True,capture_output=True)
            hits=json.loads(subprocess.run(base+['search','امان'],check=True,capture_output=True).stdout)
            assert hits and hits[0]['sha256']
            master,slave=pty.openpty();fcntl.ioctl(slave,termios.TIOCSWINSZ,struct.pack('HHHH',24,90,0,0))
            proc=subprocess.Popen(base,stdin=slave,stdout=slave,stderr=slave,env={**os.environ,'TERM':'xterm-256color','NO_COLOR':'1'});os.close(slave)
            captured=bytearray()
            def until(needle,timeout=12):
                deadline=time.monotonic()+timeout
                while time.monotonic()<deadline:
                    if needle in captured:return
                    ready,_,_=select.select([master],[],[],0.1)
                    if ready:
                        try:captured.extend(os.read(master,65536))
                        except OSError:break
                raise AssertionError(f'PTY missing {needle!r}: {bytes(captured[-1000:])!r}')
            try:
                until(b'Dot Aman AI');os.write(master,'مرحبا\r'.encode());until(b'fixture')
                os.write(master,b'/stats\r');until(b'GPU N/A')
                os.write(master,b'\x11');proc.wait(timeout=5);assert proc.returncode==0
            finally:
                if proc.poll() is None:proc.kill();proc.wait()
                os.close(master)
            result={'fixture':'local HTTP, not real model','cli_chat':'passed','session_restart_export':'passed','offline_import_arabic_search':'passed','pty_arabic_stream_resize_default':'passed','pty_stats_quit':'passed','real_wayland_clipboard':'unrun'}
            print(json.dumps(result,indent=2))
    finally:server.shutdown();server.server_close()
if __name__=='__main__':main()
