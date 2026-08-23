package main

// sDashHTML es la página de dashboard completa.
// Equivale a s_dash_html de dashboard.c.
//
// Se usa una raw string de Go. Para poder incluir el HTML sin conflictos,
// el JavaScript original que usaba template literals (backticks) se ha
// reescrito con concatenación de strings equivalente.
const sDashHTML = `<!DOCTYPE html>
<html lang='es'>
<head>
<meta charset='UTF-8'>
<meta name='viewport' content='width=device-width,initial-scale=1'>
<title>Streamline — Dashboard</title>
<link rel='preconnect' href='https://fonts.googleapis.com'>
<link href='https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600;700&display=swap' rel='stylesheet'>
<style>
:root{
  --bg:#000;--s1:#111115;--b:rgba(255,255,255,.08);
  --t:#f5f5f7;--ts:#86868b;--a:#2997ff;--g:#30d158;--r:#ff453a;
}
*{box-sizing:border-box;margin:0;padding:0;font-family:'Inter',system-ui,sans-serif}
body{background:var(--bg);color:var(--t);min-height:100vh;-webkit-font-smoothing:antialiased;padding-bottom:60px}
nav{position:fixed;top:0;width:100%;height:48px;background:rgba(0,0,0,.72);backdrop-filter:blur(20px);border-bottom:1px solid var(--b);display:flex;align-items:center;justify-content:space-between;padding:0 28px;z-index:999}
.logo{font-size:16px;font-weight:600;letter-spacing:-.02em}
.nav-links{display:flex;gap:18px;font-size:12px;font-weight:600}
.nav-links a{color:var(--ts);text-decoration:none;transition:color .2s}
.nav-links a.on{color:var(--t)}.nav-links a:hover{color:var(--a)}
.wrap{max-width:1080px;margin:0 auto;padding:72px 24px 0}
.stats{display:grid;grid-template-columns:repeat(4,1fr);gap:14px;margin-bottom:24px}
.stat{background:var(--s1);border:1px solid var(--b);border-radius:16px;padding:20px}
.stat-v{font-size:26px;font-weight:700;letter-spacing:-.03em}
.stat-l{font-size:12px;color:var(--ts);margin-top:4px}
@media(max-width:768px){.stats{grid-template-columns:repeat(2,1fr)}}
.dev-grid{display:grid;grid-template-columns:repeat(auto-fill,minmax(240px,1fr));gap:14px;margin-bottom:24px}
.dev{background:var(--s1);border:1px solid var(--b);border-radius:16px;padding:18px;position:relative;overflow:hidden}
.dev-top{display:flex;align-items:center;gap:10px;margin-bottom:12px}
.dev-ico{width:38px;height:38px;border-radius:12px;background:rgba(41,151,255,.12);display:flex;align-items:center;justify-content:center;font-size:17px}
.dev-name{font-size:14px;font-weight:600;white-space:nowrap;overflow:hidden;text-overflow:ellipsis}
.dev-id{font-size:11px;color:var(--ts);font-family:monospace}
.dot{width:8px;height:8px;border-radius:50%;flex-shrink:0}
.dot.on{background:var(--g);box-shadow:0 0 8px var(--g)}
.dot.off{background:#48484a}
.dev-grid2{display:grid;grid-template-columns:repeat(3,1fr);gap:8px}
.dev-cell{background:rgba(255,255,255,.03);border:1px solid var(--b);border-radius:10px;padding:10px;text-align:center}
.dev-cell b{display:block;font-size:16px}
.dev-cell span{font-size:10px;color:var(--ts)}
.card{background:var(--s1);border:1px solid var(--b);border-radius:20px;padding:24px;margin-bottom:20px}
.card-title{font-size:18px;font-weight:600;margin-bottom:16px;display:flex;justify-content:space-between;align-items:center}
.tabs{display:flex;border-bottom:1px solid var(--b);margin-bottom:16px;gap:26px}
.tab{background:none;border:none;border-bottom:2px solid transparent;padding-bottom:10px;font-size:13px;font-weight:500;color:var(--ts);cursor:pointer;transition:all .2s}
.tab.on{color:var(--t);border-bottom-color:var(--a)}
table{width:100%;border-collapse:collapse}
th{font-size:11px;text-transform:uppercase;letter-spacing:.06em;color:var(--ts);text-align:left;padding:10px 8px;border-bottom:1px solid var(--b)}
td{font-size:13px;padding:12px 8px;border-bottom:1px solid rgba(255,255,255,.04);vertical-align:middle}
tr:hover td{background:rgba(255,255,255,.02)}
.fname{font-weight:500;max-width:340px;white-space:nowrap;overflow:hidden;text-overflow:ellipsis}
.pill{font-size:10px;font-weight:600;padding:3px 10px;border-radius:980px;white-space:nowrap}
.pill-proc{background:rgba(41,151,255,.14);color:var(--a)}
.pill-ready{background:rgba(48,209,88,.14);color:var(--g)}
.pill-done{background:rgba(255,255,255,.06);color:var(--ts)}
td small{color:var(--ts);font-size:11px}
.btn-delete{background:transparent;color:var(--r);border:1px solid rgba(255,69,58,.35);font-size:11px;font-weight:600;padding:5px 13px;border-radius:980px;cursor:pointer;transition:all .2s;white-space:nowrap}
.btn-delete:hover{background:rgba(255,69,58,.08)}
.btn-dl{background:var(--g);color:#000;font-size:11px;font-weight:600;padding:5px 13px;border-radius:980px;border:none;cursor:pointer;white-space:nowrap;margin-right:6px}
.empty{text-align:center;padding:34px;font-size:13px;color:var(--ts)}
@media(max-width:640px){.wrap{padding-top:64px}.dev-name{max-width:130px}.fname{max-width:140px}}
</style>
</head>
<body>
<nav>
  <div class='logo'>Streamline</div>
  <div class='nav-links'>
    <a href='/'>Panel</a>
    <a class='on' href='/dashboard'>Dashboard</a>
  </div>
</nav>
<div class='wrap'>
  <div class='stats' id='stats'>
    <div class='stat'><div class='stat-v' id='s-dev'>–</div><div class='stat-l'>Dispositivos</div></div>
    <div class='stat'><div class='stat-v' id='s-vid'>–</div><div class='stat-l'>Vídeos en el servidor</div></div>
    <div class='stat'><div class='stat-v' id='s-proc'>–</div><div class='stat-l'>En proceso</div></div>
    <div class='stat'><div class='stat-v' id='s-bytes'>–</div><div class='stat-l'>Almacenamiento</div></div>
  </div>

  <div class='card'>
    <div class='card-title'>Dispositivos Conectados</div>
    <div class='dev-grid' id='devs'></div>
  </div>

  <div class='card'>
    <div class='card-title'>Vídeos y Historial</div>
    <div class='tabs'>
      <button class='tab on' id='t-ready' onclick='setTab("ready",this)'>Listos (<span id='c-ready'>0</span>)</button>
      <button class='tab' id='t-proc' onclick='setTab("processing",this)'>En Proceso (<span id='c-proc'>0</span>)</button>
      <button class='tab' id='t-hist' onclick='setTab("downloaded",this)'>Historial (<span id='c-hist'>0</span>)</button>
    </div>
    <div id='vlist'></div>
  </div>
</div>
<script>
let tab='ready';
let items=[];
let did=localStorage.getItem('sl_did')||'dash';
let dname=localStorage.getItem('sl_dname')||'Dashboard';

function fmt(n){
  if(!n&&n!==0)return'–';
  if(n>=1073741824)return(n/1073741824).toFixed(2)+' GB';
  if(n>=1048576)return(n/1048576).toFixed(1)+' MB';
  if(n>=1024)return(n/1024).toFixed(0)+' KB';
  return n+' B';
}
function esc(s){return String(s).replace(/[&<>"']/g,function(m){return {'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[m];});}

function fetchStats(){
  fetch('/api/stats').then(r=>r.json()).then(d=>{
    document.getElementById('s-vid').textContent=d.videos;
    document.getElementById('s-proc').textContent=d.processing;
    document.getElementById('s-bytes').textContent=fmt(d.bytes);
  }).catch(()=>{});
}

function fetchDevs(){
  fetch('/api/devices').then(r=>r.json()).then(list=>{
    document.getElementById('s-dev').textContent=list.length;
    let el=document.getElementById('devs');
    if(!list.length){el.innerHTML='<div class="empty">Ningún dispositivo ha subido vídeos todavía.</div>';return;}
    el.innerHTML=list.map(d=>{
      let ico=d.deviceInfo==='iPhone'?'📱':d.deviceInfo==='Android'?'🤖':'💻';
      return '<div class="dev"><div class="dev-top"><div class="dev-ico">' + ico + '</div><div style="flex:1;min-width:0"><div class="dev-name">' + esc(d.deviceInfo) + '</div><div class="dev-id">' + esc(d.deviceId) + '</div></div><div class="dot ' + (d.online?'on':'off') + '"></div></div><div class="dev-grid2"><div class="dev-cell"><b>' + d.videos + '</b><span>vídeos</span></div><div class="dev-cell"><b>' + fmt(d.totalSize) + '</b><span>total</span></div><div class="dev-cell"><b>' + d.active + '</b><span>en servidor</span></div></div></div>';
    }).join('');
  }).catch(()=>{});
}

function setTab(name,btn){
  tab=name;
  document.querySelectorAll('.tab').forEach(b=>b.classList.remove('on'));
  if(btn)btn.classList.add('on');
  render();
}

function fetchHist(){
  fetch('/api/history?deviceId=').then(r=>r.json()).then(d=>{
    items=d;render();
  }).catch(()=>{});
}

function render(){
  let ready=items.filter(x=>x.status==='ready');
  let proc=items.filter(x=>x.status==='processing'||x.status==='uploading');
  let hist=items.filter(x=>x.status==='downloaded');
  document.getElementById('c-ready').textContent=ready.length;
  document.getElementById('c-proc').textContent=proc.length;
  document.getElementById('c-hist').textContent=hist.length;
  let cur=tab==='ready'?ready:tab==='processing'?proc:hist;
  let el=document.getElementById('vlist');
  if(!cur.length){el.innerHTML='<div class="empty">Sin archivos en esta categoría.</div>';return;}
  let rows=cur.map(it=>{
    let mb=fmt(it.originalSize);
    let d=new Date(it.uploadTime*1000);
    let fe=d.toLocaleDateString('es')+' '+d.toLocaleTimeString('es',{hour:'2-digit',minute:'2-digit'});
    let st=it.status==='ready'?'<span class="pill pill-ready">Listo</span>':it.status==='processing'?'<span class="pill pill-proc">Procesando</span>':'<span class="pill pill-done">Descargado</span>';
    let act;
    if(it.status==='ready'){
      act='<button class="btn-dl" onclick="dlOne(\'' + esc(it.filename) + '\')">Descargar</button> <button class="btn-delete" onclick="delEntry(\'' + esc(it.filename) + '\',0)">Eliminar</button>';
    } else if(it.status==='processing'){
      act='<button class="btn-delete" onclick="delEntry(\'' + esc(it.filename) + '\',1)">Eliminar</button>';
    } else {
      act='<button class="btn-delete" onclick="delEntry(\'' + esc(it.filename) + '\',1)">Quitar</button>';
    }
    return '<tr><td><div class="fname">' + esc(it.filename) + '</div><small>' + esc(it.deviceInfo) + ' · ' + esc(it.deviceId) + '</small></td><td>' + mb + '</td><td>' + st + '</td><td><small>' + fe + '</small></td><td style="text-align:right;white-space:nowrap">' + act + '</td></tr>';
  }).join('');
  el.innerHTML='<table><thead><tr><th>Archivo</th><th>Tamaño</th><th>Estado</th><th>Subido</th><th></th></tr></thead><tbody>' + rows + '</tbody></table>';
}

function dlOne(name){
  let a=document.createElement('a');
  a.href='/download?file='+encodeURIComponent(name)+'&deviceId='+did;
  a.download=name;
  document.body.appendChild(a);a.click();a.remove();
  setTimeout(fetchHist,3000);
}
function delEntry(name,purge){
  fetch('/api/cleanup?file='+encodeURIComponent(name)+'&deviceId='+did+(purge?'&purge=1':'')).then(r=>r.json()).then(()=>{fetchHist();fetchStats();fetchDevs();});
}

fetchStats();fetchDevs();fetchHist();
setInterval(()=>{fetchStats();fetchDevs();fetchHist();},3000);
</script>
</body></html>`