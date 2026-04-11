<script>
  import { onMount, onDestroy } from 'svelte';
  import { hdrs } from '$lib/stores/auth.js';
  import AppShell from '$lib/components/AppShell.svelte';
  import Card from '$lib/ui/Card.svelte';
  import Badge from '$lib/ui/Badge.svelte';
  import SectionLabel from '$lib/ui/SectionLabel.svelte';
  import Button from '$lib/ui/Button.svelte';

  let active = 'resumen';
  let loading = true;
  let pools = [];

  // ── App icon: duotone database ──
  const appIcon = [
    { tag: 'path', attrs: { d: 'M3 5c0 1.66 4 3 9 3s9-1.34 9-3v14c0 1.66-4 3-9 3s-9-1.34-9-3z', fill: 'currentColor', opacity: '0.12', stroke: 'none' }},
    { tag: 'ellipse', attrs: { cx: 12, cy: 5, rx: 9, ry: 3 }},
    { tag: 'path', attrs: { d: 'M3 5v14c0 1.66 4 3 9 3s9-1.34 9-3V5', fill: 'none' }},
    { tag: 'path', attrs: { d: 'M3 12c0 1.66 4 3 9 3s9-1.34 9-3', fill: 'none' }},
  ];

  // ── Sidebar navigation ──
  const sections = [
    {
      label: 'Almacenamiento',
      items: [
        { id: 'resumen', label: 'Resumen', paths: [
          { tag: 'rect', attrs: { x:3, y:3, width:7, height:7, rx:2, 'data-fill': '' }},
          { tag: 'rect', attrs: { x:14, y:3, width:7, height:7, rx:2, 'data-fill': '' }},
          { tag: 'rect', attrs: { x:3, y:14, width:7, height:7, rx:2, 'data-fill': '' }},
          { tag: 'rect', attrs: { x:14, y:14, width:7, height:7, rx:2, 'data-fill': '' }},
          { tag: 'rect', attrs: { x:3, y:3, width:7, height:7, rx:2 }},
          { tag: 'rect', attrs: { x:14, y:3, width:7, height:7, rx:2 }},
          { tag: 'rect', attrs: { x:3, y:14, width:7, height:7, rx:2 }},
          { tag: 'rect', attrs: { x:14, y:14, width:7, height:7, rx:2 }},
        ]},
        { id: 'disks', label: 'Discos', paths: [
          { tag: 'circle', attrs: { cx:12, cy:12, r:9, 'data-fill': '' }},
          { tag: 'circle', attrs: { cx:12, cy:12, r:9 }},
          { tag: 'circle', attrs: { cx:12, cy:12, r:2.5, 'data-dot': '' }},
        ]},
        { id: 'snapshots', label: 'Puntos de restauración', paths: [
          { tag: 'circle', attrs: { cx:13, cy:14, r:8, 'data-fill': '' }},
          { tag: 'polyline', attrs: { points: '1 4 1 10 7 10' }},
          { tag: 'path', attrs: { d: 'M3.51 15a9 9 0 1 0 2.13-9.36L1 10' }},
        ]},
      ]
    },
    {
      label: 'Mantenimiento',
      items: [
        { id: 'health', label: 'Salud', paths: [
          { tag: 'path', attrs: { d: 'M9 3l-3 9H2l4 9h0l3-9h4l3-9', 'data-fill': '', style: 'opacity:0.08' }},
          { tag: 'path', attrs: { d: 'M22 12h-4l-3 9L9 3l-3 9H2' }},
        ]},
        { id: 'restore', label: 'Restaurar volumen', paths: [
          { tag: 'rect', attrs: { x:3, y:15, width:18, height:6, rx:2, 'data-fill': '' }},
          { tag: 'path', attrs: { d: 'M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4' }},
          { tag: 'polyline', attrs: { points: '17 8 12 3 7 8' }},
          { tag: 'line', attrs: { x1:12, y1:3, x2:12, y2:15 }},
        ]},
      ]
    }
  ];

  // ── Data loading ──
  async function load() {
    loading = true;
    try {
      const res = await fetch('/api/storage/status', { headers: hdrs() });
      const data = await res.json();
      pools = data.pools || [];
    } catch (e) {
      console.error('[Storage] load failed', e);
    }
    loading = false;
  }

  // ── Helpers ──
  function poolStatus(pool) {
    if (pool.poolHealth === 'critical' || pool.health === 'FAULTED') return 'crit';
    if (pool.poolHealth === 'warning' || pool.health === 'DEGRADED') return 'warn';
    return 'ok';
  }
  function poolStatusLabel(pool) {
    const s = poolStatus(pool);
    return s === 'crit' ? 'En riesgo' : s === 'warn' ? 'Atención' : 'Sano';
  }
  function diskStatus(disk) {
    if (disk.smartStatus === 'critical') return 'crit';
    if (disk.smartStatus === 'warning') return 'warn';
    return 'ok';
  }
  function diskStatusLabel(disk) {
    const s = diskStatus(disk);
    return s === 'crit' ? 'Crítico' : s === 'warn' ? 'Atención' : 'Sano';
  }
  function formatSize(v) {
    if (!v && v !== 0) return '—';
    if (typeof v === 'string') return v;
    const tb = v / (1024 ** 4);
    if (tb >= 1) return tb.toFixed(1) + ' TB';
    const gb = v / (1024 ** 3);
    if (gb >= 1) return gb.toFixed(1) + ' GB';
    return (v / (1024 ** 2)).toFixed(0) + ' MB';
  }
  function formatHours(h) {
    if (!h) return '—';
    return h >= 1000 ? (h / 1000).toFixed(1) + 'k h' : h + ' h';
  }
  function vdevLabel(t) {
    return { raidz1:'RAIDZ1', raidz2:'RAIDZ2', raidz3:'RAIDZ3', mirror:'Espejo', single:'Simple', stripe:'Stripe' }[t] || t || '—';
  }

  let refreshInterval;
  onMount(() => { load(); refreshInterval = setInterval(load, 30000); });
  onDestroy(() => { if (refreshInterval) clearInterval(refreshInterval); });
</script>

<AppShell title="Almacenamiento" {appIcon} {sections} bind:active showSearch>

  {#if loading && pools.length === 0}
    <div class="state-msg">Cargando...</div>

  {:else if active === 'resumen'}
    {#each pools as pool}
      <Card>
        <div class="pool-header">
          <div>
            <div class="pool-name">{pool.name}</div>
            <div class="pool-sub">{vdevLabel(pool.vdevType)} · {pool.type?.toUpperCase()} · {pool.disks?.length || 0} discos</div>
          </div>
          <Badge status={poolStatus(pool)}>{poolStatusLabel(pool)}</Badge>
        </div>

        <div class="capacity">
          <div class="cap-row">
            <div class="bar-track">
              <div class="bar-fill" style="width:{pool.usedPercent || 0}%"></div>
            </div>
            <div class="cap-pct" class:warn={pool.usedPercent > 80} class:crit={pool.usedPercent > 95}>
              {pool.usedPercent || 0}<span class="sym">%</span>
            </div>
          </div>
          <div class="cap-info">
            <span class="mono">{formatSize(pool.used)} usados</span>
            <span class="mono muted">{formatSize(pool.capacity)}</span>
          </div>
        </div>

        <div class="pool-actions">
          <Button>Gestionar</Button>
          <Button variant="primary">+ Punto de restauración</Button>
        </div>

        {#if pool.enrichedDisks?.length > 0}
          <div class="smart-section">
            <SectionLabel>Estado SMART</SectionLabel>
            <div class="dtable-head">
              <div></div><div>Modelo</div><div>Dispositivo</div><div>Capacidad</div><div>Temp</div><div>Horas</div><div>Estado</div>
            </div>
            {#each pool.enrichedDisks as disk}
              <div class="dtable-row">
                <div class="disk-icon">
                  <svg viewBox="0 0 24 24" fill="currentColor">
                    <path d="M18.84 13.38c1.13 0 2.14.45 2.9 1.18L19.37 5.18C18.84 3.54 17.9 3 16.74 3H7.26C6.1 3 5.16 3.54 4.63 5.18L2.27 14.56c.75-.73 1.76-1.18 2.89-1.18z"/>
                    <path d="M5.16 14.4C4 14.4 2.96 15.07 2.41 16.08c-.26.48-.41 1.03-.41 1.62C2 19.55 3.44 21 5.16 21h13.68c1.72 0 3.16-1.45 3.16-3.3 0-.59-.15-1.14-.41-1.62-.55-1.01-1.58-1.68-2.75-1.68z"/>
                  </svg>
                </div>
                <div class="d-name">{disk.model || disk.name}</div>
                <div class="d-cell">/dev/{disk.name}</div>
                <div class="d-cell">{disk.size || '—'}</div>
                <div class="d-cell" class:temp-hot={disk.temperature > 50}>{disk.temperature ? disk.temperature + '°C' : '—'}</div>
                <div class="d-cell">{formatHours(disk.powerOnHours)}</div>
                <div><Badge status={diskStatus(disk)}>{diskStatusLabel(disk)}</Badge></div>
              </div>
            {/each}
          </div>
        {/if}
      </Card>
    {/each}

    {#if pools.length === 0}
      <Card>
        <div class="state-msg">
          <div style="font-size:18px;font-weight:600;color:var(--text-primary);margin-bottom:8px">Sin volúmenes</div>
          <div>Crea un volumen para empezar a almacenar datos.</div>
        </div>
      </Card>
    {/if}

  {:else if active === 'disks'}
    <Card><SectionLabel>Discos físicos</SectionLabel><div class="state-msg">En desarrollo</div></Card>
  {:else if active === 'snapshots'}
    <Card><SectionLabel>Puntos de restauración</SectionLabel><div class="state-msg">En desarrollo</div></Card>
  {:else if active === 'health'}
    <Card><SectionLabel>Salud del sistema</SectionLabel><div class="state-msg">En desarrollo</div></Card>
  {:else if active === 'restore'}
    <Card><SectionLabel>Restaurar volumen</SectionLabel><div class="state-msg">En desarrollo</div></Card>
  {/if}
</AppShell>

<style>
  .pool-header { display:flex; justify-content:space-between; align-items:flex-start; margin-bottom:22px; }
  .pool-name { font-size:26px; font-weight:700; letter-spacing:-0.5px; color:var(--text-primary); }
  .pool-sub { font-size:13px; color:var(--text-secondary); margin-top:4px; }

  .capacity { margin-bottom:22px; }
  .cap-row { display:flex; align-items:center; gap:32px; max-width:78%; }
  .bar-track {
    flex:1; height:13px; border-radius:8px;
    background:var(--bg-elev-2); overflow:hidden;
    border:1px solid var(--glass-border);
  }
  .bar-fill { height:100%; border-radius:8px; background:var(--accent); transition:width 0.5s; }
  .cap-pct {
    font-size:42px; font-weight:700; letter-spacing:-1.5px; line-height:1;
    color:var(--c-ok); white-space:nowrap; font-family:var(--font-mono);
  }
  .cap-pct.warn { color:var(--c-warn); }
  .cap-pct.crit { color:var(--c-crit); }
  .cap-pct .sym { font-size:28px; font-weight:600; margin-left:2px; }
  .cap-info { display:flex; justify-content:space-between; margin-top:12px; font-size:13px; max-width:62%; }
  .mono { font-family:var(--font-mono); }
  .muted { color:var(--text-muted); }

  .pool-actions { display:flex; gap:10px; border-top:1px solid var(--glass-border); padding-top:20px; margin-bottom:22px; }

  .smart-section { border-top:1px solid var(--glass-border); padding-top:18px; }

  .dtable-head {
    display:grid; grid-template-columns:28px 1.4fr 1.2fr 0.7fr 0.6fr 0.7fr 110px;
    gap:14px; padding:0 12px 8px;
    font-size:10px; color:var(--text-muted); text-transform:uppercase; letter-spacing:0.8px;
  }
  .dtable-row {
    display:grid; grid-template-columns:28px 1.4fr 1.2fr 0.7fr 0.6fr 0.7fr 110px;
    align-items:center; gap:14px; padding:9px 12px; border-radius:8px;
    font-size:12px; cursor:pointer; transition:background 0.15s;
  }
  .dtable-row + .dtable-row { margin-top:2px; }
  .dtable-row:hover { background:var(--bg-elev-2); }

  .disk-icon {
    width:26px; height:26px; border-radius:6px;
    background:var(--bg-elev-2);
    display:flex; align-items:center; justify-content:center;
    color:var(--text-secondary);
  }
  .disk-icon svg { width:16px; height:16px; display:block; }
  .dtable-row:hover .disk-icon { color:var(--text-primary); }

  .d-name { font-weight:500; color:var(--text-primary); white-space:nowrap; overflow:hidden; text-overflow:ellipsis; }
  .d-cell { font-family:var(--font-mono); color:var(--text-secondary); white-space:nowrap; }
  .temp-hot { color:var(--c-warn); }

  .state-msg { font-size:13px; color:var(--text-muted); padding:20px 0; text-align:center; }
</style>
