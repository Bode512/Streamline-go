<script lang="ts">
  import { onMount } from 'svelte'
  import { ServerURL } from '../wailsjs/go/main/App.js'

  type Stats = { videos: number; processing: number; ready: number; downloaded: number; bytes: number }
  type Item = { filename: string; deviceId: string; deviceInfo: string; status: string; originalSize: number; uploadTime: number }

  let server = 'http://127.0.0.1:8000'
  let connected = false
  let error = ''
  let stats: Stats = { videos: 0, processing: 0, ready: 0, downloaded: 0, bytes: 0 }
  let history: Item[] = []
  let activity: string[] = []
  let qrURL = ''
  let durations: Record<string, string> = {}

  const api = (path: string) => `${server.replace(/\/$/, '')}${path}`
  const formatBytes = (value: number) => value < 1024 * 1024 ? `${Math.round(value / 1024)} KB` : `${(value / 1024 / 1024).toFixed(1)} MB`
  const formatDate = (value: number) => new Date(value * 1000).toLocaleString()
  const formatDuration = (value: number) => `${Math.floor(value / 60)}:${Math.floor(value % 60).toString().padStart(2, '0')}`

  async function refresh() {
    error = ''
    try {
      const [statsResponse, historyResponse] = await Promise.all([fetch(api('/api/stats')), fetch(api('/api/history'))])
      if (!statsResponse.ok || !historyResponse.ok) throw new Error('Servidor no disponible')
      stats = await statsResponse.json()
      history = await historyResponse.json()
      connected = true
    } catch (reason) {
      connected = false
      error = reason instanceof Error ? reason.message : 'No se pudo conectar'
    }
  }

  function loadQR() { qrURL = api('/api/qr') }

  async function jobAction(action: string, filename: string) {
    await fetch(api(`/api/jobs/${action}?file=${encodeURIComponent(filename)}`), { method: 'POST' })
    await refresh()
  }

  function connectEvents() {
    const stream = new EventSource(api('/api/events'))
    stream.addEventListener('streamline', (event) => {
      const data = JSON.parse((event as MessageEvent).data)
      activity = [`${data.type} · ${data.filename ?? 'sistema'}`, ...activity].slice(0, 5)
      refresh()
    })
    stream.onerror = () => stream.close()
  }

  onMount(() => {
    ServerURL().then((value: string) => { server = value; refresh(); loadQR(); connectEvents() }).catch(() => { refresh(); loadQR(); connectEvents() })
    const timer = window.setInterval(refresh, 10000)
    return () => window.clearInterval(timer)
  })
</script>

<svelte:head><title>Streamline Desktop</title></svelte:head>

<main>
  <header class="topbar">
    <div class="brand"><span class="mark">S</span><div><strong>Streamline</strong><small>VIDEO OPERATIONS</small></div></div>
    <div class="connection"><span class:online={connected}></span>{connected ? 'Servidor conectado' : 'Sin conexión'}</div>
  </header>

  <section class="hero">
    <div><p class="eyebrow">CONTROL CENTER</p><h1>Tu biblioteca,<br /><em>en movimiento.</em></h1><p class="lede">Procesa, comparte y sigue cada conversión desde un solo lugar.</p></div>
    <div class="connect-box"><label for="server">Servidor Streamline</label><div class="server-input"><input id="server" bind:value={server} /><button title="Conectar" on:click={() => { refresh(); loadQR(); connectEvents() }}>→</button></div>{#if error}<p class="error">{error}</p>{:else}<p class="hint">Instancia local o cualquier equipo en tu red.</p>{/if}</div>
  </section>

  <section class="stats">
    <div class="stat primary"><span>EN COLA</span><strong>{stats.processing}</strong><small>Procesando ahora</small></div>
    <div class="stat"><span>LISTOS</span><strong>{stats.ready}</strong><small>Esperando descarga</small></div>
    <div class="stat"><span>DESCARGADOS</span><strong>{stats.downloaded}</strong><small>Sesiones completadas</small></div>
    <div class="stat"><span>DATOS ACTIVOS</span><strong>{formatBytes(stats.bytes)}</strong><small>En el servidor</small></div>
  </section>

  <section class="content-grid">
    <div class="panel jobs"><div class="panel-head"><div><p class="eyebrow">RECENT ACTIVITY</p><h2>Trabajos</h2></div><button class="refresh" title="Actualizar" on:click={refresh}>↻</button></div>
      {#if history.length === 0}<div class="empty">Todavía no hay vídeos en la biblioteca.</div>{:else}<div class="job-list">{#each history.slice(0, 7) as item}<div class="job"><div class="preview">{#if item.status === 'ready'}<video src={api(`/preview?file=${encodeURIComponent(item.filename)}`)} preload="metadata" muted on:loadedmetadata={(event) => durations[item.filename] = formatDuration((event.currentTarget as HTMLVideoElement).duration)}></video><span class="play">▶</span>{:else}<span class="format">{item.filename.split('.').pop()?.toUpperCase().slice(0, 3)}</span>{/if}</div><div class="file"><strong>{item.filename}</strong><small>{item.deviceInfo || item.deviceId} · {formatDate(item.uploadTime)}</small><div class="meta">{formatBytes(item.originalSize)} {#if durations[item.filename]} · {durations[item.filename]}{/if}</div></div><span class="status" class:ready={item.status === 'ready'} class:failed={item.status === 'failed'}>{item.status}</span>{#if item.status === 'processing'}<button class="action" title="Cancelar" on:click={() => jobAction('cancel', item.filename)}>×</button>{:else if item.status === 'failed'}<button class="action" title="Reintentar" on:click={() => jobAction('retry', item.filename)}>↻</button>{/if}</div>{/each}</div>{/if}
    </div>
    <aside class="panel share"><p class="eyebrow">MOBILE DROP</p><h2>Envía desde tu móvil</h2><p>Escanea el código para abrir el panel de subida en tu red local.</p>{#if qrURL}<img class="qr" src={qrURL} alt="Código QR de Streamline" />{/if}<div class="share-url">{server}/</div></aside>
  </section>

  <footer><span>STREAMLINE DESKTOP · CORE ONLINE</span><span>{activity.length ? activity[0] : 'Esperando eventos...'}</span></footer>
</main>

<style>
  :global(*) { box-sizing: border-box; }
  :global(body) { margin: 0; background: #101313; color: #f4f1e8; font-family: Georgia, 'Times New Roman', serif; }
  :global(button), :global(input) { font: inherit; }
  main { max-width: 1240px; margin: auto; padding: 28px 42px 24px; min-height: 100vh; background: radial-gradient(circle at 85% 12%, #244846 0, transparent 28%), #101313; }
  .topbar { display: flex; align-items: center; justify-content: space-between; border-bottom: 1px solid #3b4c48; padding-bottom: 18px; }
  .brand { display: flex; gap: 12px; align-items: center; } .brand strong { font-size: 21px; letter-spacing: .02em; } .brand small { display: block; color: #9bb1a7; font: 10px Arial, sans-serif; letter-spacing: .2em; margin-top: 3px; }
  .mark { display: grid; place-items: center; width: 38px; height: 38px; background: #d6a85e; color: #101313; font-size: 25px; font-weight: bold; }
  .connection { color: #9bb1a7; font: 12px Arial, sans-serif; } .connection span { display: inline-block; width: 8px; height: 8px; margin-right: 8px; background: #a65c4b; border-radius: 50%; } .connection span.online { background: #8ac28b; box-shadow: 0 0 12px #8ac28b; }
  .hero { display: flex; justify-content: space-between; gap: 40px; padding: 70px 0 48px; align-items: end; } .eyebrow { margin: 0 0 10px; color: #d6a85e; font: 10px Arial, sans-serif; letter-spacing: .2em; } h1 { margin: 0; font-size: clamp(42px, 6vw, 78px); font-weight: normal; line-height: .94; } h1 em { color: #d6a85e; font-style: italic; } .lede { color: #9bb1a7; font: 15px Arial, sans-serif; max-width: 390px; line-height: 1.6; margin: 24px 0 0; }
  .connect-box { width: 340px; padding: 20px; border: 1px solid #3b4c48; background: rgba(27, 39, 37, .75); } .connect-box label { display: block; color: #9bb1a7; font: 11px Arial, sans-serif; margin-bottom: 10px; } .server-input { display: flex; } input { width: 100%; min-width: 0; padding: 11px; border: 1px solid #53645f; background: #101313; color: #f4f1e8; outline: none; } .server-input button, .refresh, .action { border: 0; background: #d6a85e; color: #101313; cursor: pointer; width: 44px; font-size: 22px; } .hint, .error { color: #9bb1a7; font: 11px Arial, sans-serif; line-height: 1.5; margin: 12px 0 0; } .error { color: #db8e77; }
  .stats { display: grid; grid-template-columns: repeat(4, 1fr); gap: 12px; } .stat { padding: 20px; border: 1px solid #3b4c48; background: #18201f; } .stat.primary { border-color: #d6a85e; } .stat span, .stat small { display: block; color: #9bb1a7; font: 10px Arial, sans-serif; letter-spacing: .13em; } .stat strong { display: block; margin: 13px 0 8px; color: #f4f1e8; font-size: 34px; font-weight: normal; }
  .content-grid { display: grid; grid-template-columns: 1.5fr 1fr; gap: 12px; margin-top: 12px; } .panel { border: 1px solid #3b4c48; background: #18201f; padding: 24px; } .panel-head { display: flex; justify-content: space-between; align-items: start; border-bottom: 1px solid #3b4c48; padding-bottom: 18px; } h2 { margin: 0; font-size: 27px; font-weight: normal; } .refresh { width: 32px; height: 32px; } .job { display: flex; align-items: center; gap: 12px; padding: 14px 0; border-bottom: 1px solid #2c3a37; } .preview { position: relative; flex: 0 0 76px; height: 48px; overflow: hidden; background: #2b4541; } .preview video { width: 100%; height: 100%; object-fit: cover; } .play { position: absolute; inset: 0; display: grid; place-items: center; color: #f4f1e8; font-size: 13px; background: rgba(16, 19, 19, .28); } .format { display: grid; place-items: center; width: 100%; height: 100%; color: #d6a85e; font: 10px Arial, sans-serif; } .file { flex: 1; min-width: 0; } .file strong { display: block; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-size: 15px; font-weight: normal; } .file small { display: block; color: #9bb1a7; font: 10px Arial, sans-serif; margin-top: 5px; } .meta { color: #d6a85e; font: 10px Arial, sans-serif; margin-top: 5px; } .status { color: #d6a85e; font: 10px Arial, sans-serif; text-transform: uppercase; } .status.ready { color: #8ac28b; } .status.failed { color: #db8e77; } .action { width: 27px; height: 27px; font-size: 17px; } .empty { color: #9bb1a7; padding: 38px 0; font: 13px Arial, sans-serif; } .share { background: #d6a85e; color: #101313; } .share .eyebrow { color: #435c52; } .share p { max-width: 260px; font: 14px Arial, sans-serif; line-height: 1.5; } .qr { display: block; width: 170px; height: 170px; margin: 22px auto; border: 8px solid #f4f1e8; } .share-url { border-top: 1px solid #a47f45; padding-top: 12px; overflow: hidden; text-overflow: ellipsis; font: 12px Arial, sans-serif; }
  footer { display: flex; justify-content: space-between; color: #6e8780; font: 10px Arial, sans-serif; letter-spacing: .12em; padding-top: 24px; }
  @media (max-width: 760px) { main { padding: 20px; } .hero, .content-grid { display: block; } .connect-box { width: auto; margin-top: 32px; } .stats { grid-template-columns: repeat(2, 1fr); margin-top: 12px; } .share { margin-top: 12px; } footer { display: block; line-height: 2; } }
</style>
