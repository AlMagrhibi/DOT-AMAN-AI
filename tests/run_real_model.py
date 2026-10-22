#!/usr/bin/env python3
"""Cloud-only real GGUF evaluation. Requires MODEL_PATH, LLAMA_SERVER and Go.
No automatic download or mock fallback. All resource measurements describe Cloud.
"""
import argparse,hashlib,json,os,pathlib,socket,subprocess,tempfile,threading,time,urllib.request
ROOT=pathlib.Path(__file__).resolve().parents[1]
def counters(pid):
    base=pathlib.Path('/proc')/str(pid);raw=(base/'stat').read_text();f=raw[raw.rfind(')')+2:].split()
    cpu=(int(f[11])+int(f[12]))/os.sysconf('SC_CLK_TCK');rss=0
    for line in (base/'status').read_text().splitlines():
        if line.startswith('VmRSS:'):rss=int(line.split()[1])*1024
    return cpu,rss

def main():
    parser=argparse.ArgumentParser();parser.add_argument('--model-lock',type=pathlib.Path,default=ROOT/'configs/model-lock.json');parser.add_argument('--runs',type=int,default=3);parser.add_argument('--output',type=pathlib.Path,required=True);args=parser.parse_args()
    if not 1<=args.runs<=20:parser.error('runs must be 1..20')
    model=pathlib.Path(os.environ['MODEL_PATH']).resolve();server=pathlib.Path(os.environ['LLAMA_SERVER']).resolve()
    lock=json.loads(args.model_lock.read_text());h=hashlib.sha256()
    with model.open('rb') as f:
        for b in iter(lambda:f.read(1024*1024),b''):h.update(b)
    if h.hexdigest()!=lock['sha256'] or model.stat().st_size!=lock['size_bytes']:raise RuntimeError('GGUF does not match official pinned metadata')
    probe=socket.socket();probe.settimeout(1)
    try:
        if probe.connect_ex(('127.0.0.1',8081))==0:raise RuntimeError('port 8081 in use; refusing to stop existing service')
    finally:probe.close()
    binary=ROOT/'bin/dotaman-eval';binary.parent.mkdir(exist_ok=True)
    subprocess.run(['go','build','-tags','sqlite_fts5','-trimpath','-o',str(binary),'./cmd/dotaman-eval'],cwd=ROOT,check=True)
    stop=threading.Event();samples=[];started=time.monotonic();version=subprocess.run([str(server),'--version'],capture_output=True,text=True,check=True)
    with tempfile.TemporaryDirectory(prefix='dotaman-real-server-') as td:
        log=pathlib.Path(td)/'server.log'
        with log.open('w') as out:
            env={**os.environ,'MODEL_PATH':str(model),'LLAMA_SERVER':str(server),'MODEL_CONTEXT':'4096','MODEL_THREADS':'4'}
            proc=subprocess.Popen([str(ROOT/'scripts/start-model.sh')],cwd=ROOT,env=env,stdout=out,stderr=subprocess.STDOUT)
        def sample():
            while not stop.wait(.1):
                try:cpu,rss=counters(proc.pid);samples.append((time.monotonic(),cpu,rss))
                except (OSError,ValueError,IndexError):return
        worker=threading.Thread(target=sample,daemon=True);worker.start();opener=urllib.request.build_opener(urllib.request.ProxyHandler({}))
        try:
            deadline=time.monotonic()+120
            while time.monotonic()<deadline:
                if proc.poll() is not None:raise RuntimeError('server exited during load: '+log.read_text()[-2000:])
                try:
                    with opener.open('http://127.0.0.1:8081/health',timeout=1) as r:
                        if r.status==200:break
                except OSError:time.sleep(.2)
            else:raise RuntimeError('server health timed out')
            ready=time.monotonic();run=subprocess.run([str(binary),'-runs',str(args.runs)],cwd=ROOT,capture_output=True,text=True,timeout=900)
            if not run.stdout.strip():raise RuntimeError('evaluator returned no report: '+run.stderr[-2000:])
            result=json.loads(run.stdout);result['evaluation_exit_code']=run.returncode;result['model_lock']=lock;result['llama_version']=(version.stdout+version.stderr).strip()
            result['cloud_runtime']={'server_context':4096,'server_threads':4,'server_parallel':1,'load_seconds':ready-started,'resource_sample_interval_seconds':.1,'gpu':'N/A; CPU-only build','hardware_scope':'Cloud, not target CachyOS hardware'}
            stop.set();worker.join(timeout=2)
            if samples:
                elapsed=samples[-1][0]-started;cpu=samples[-1][1]
                result['cloud_runtime'].update({'sampled_peak_server_rss_bytes':max(s[2] for s in samples),'server_cpu_seconds':cpu,'server_average_cpu_percent_one_core':100*cpu/elapsed if elapsed else None,'resource_scope':'llama-server during load + evaluation; sampled RSS may miss short peaks','samples':len(samples)})
            args.output.parent.mkdir(parents=True,exist_ok=True);args.output.write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n')
            print(json.dumps({k:v for k,v in result.items() if k not in ['samples','checks']},ensure_ascii=False,indent=2))
            if run.returncode:raise RuntimeError('real model has failed checks; inspect report '+str(args.output))
        finally:
            stop.set();worker.join(timeout=2)
            if proc.poll() is None:
                proc.terminate()
                try:proc.wait(timeout=10)
                except subprocess.TimeoutExpired:proc.kill();proc.wait()
if __name__=='__main__':main()
