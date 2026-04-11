<script>
  import { onMount, onDestroy } from 'svelte';
  import { hdrs } from '$lib/stores/auth.js';
  import AppShell from '$lib/components/AppShell.svelte';
  import Card from '$lib/ui/Card.svelte';
  import Badge from '$lib/ui/Badge.svelte';
  import SectionLabel from '$lib/ui/SectionLabel.svelte';
  import Button from '$lib/ui/Button.svelte';
  import StatusHero from '$lib/ui/StatusHero.svelte';

  let active = 'resumen';
  let loading = true;
  let pools = [];
  let shares = [];
  let poolFileStats = {}; // poolName → { video, image, audio, document, other }
  let detailPool = null; // when set, shows the "Gestionar" view for this pool

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

  // ── Data ──
  async function load() {
    loading = true;
    try {
      const [statusRes, sharesRes] = await Promise.all([
        fetch('/api/storage/status', { headers: hdrs() }),
        fetch('/api/shares', { headers: hdrs() }),
      ]);
      const data = await statusRes.json();
      pools = data.pools || [];
      shares = await sharesRes.json();

      // Aggregate fileStats per pool
      const agg = {};
      for (const s of shares) {
        const pn = s.pool || s.volume;
        if (!pn) continue;
        if (!agg[pn]) agg[pn] = { video: 0, image: 0, audio: 0, document: 0, other: 0 };
        if (s.fileStats) {
          agg[pn].video += s.fileStats.video || 0;
          agg[pn].image += s.fileStats.image || 0;
          agg[pn].audio += s.fileStats.audio || 0;
          agg[pn].document += s.fileStats.document || 0;
          agg[pn].other += s.fileStats.other || 0;
        }
      }
      poolFileStats = agg;
    } catch (e) {
      console.error('[Storage] load failed', e);
    }
    loading = false;
  }

  // ── Helpers ──
  function poolStatus(pool) {
    const ph = pool.poolHealth;
    if (ph?.status === 'critical' || pool.health === 'FAULTED') return 'crit';
    if (ph?.status === 'degraded' || pool.health === 'DEGRADED') return 'warn';
    // Check if any disk has SMART issues
    if (pool.disks?.some(d => d.smartStatus === 'critical')) return 'warn';
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
  function formatHours(h) {
    if (!h) return '—';
    return h >= 1000 ? (h / 1000).toFixed(1) + 'k h' : h + ' h';
  }
  function vdevLabel(t) {
    return { raidz1:'RAIDZ1', raidz2:'RAIDZ2', raidz3:'RAIDZ3', mirror:'Espejo', single:'Simple', stripe:'Stripe' }[t] || t || '—';
  }
  function formatBytes(b) {
    if (!b) return '0 B';
    if (b >= 1099511627776) return (b / 1099511627776).toFixed(1) + ' TB';
    if (b >= 1073741824) return (b / 1073741824).toFixed(1) + ' GB';
    if (b >= 1048576) return (b / 1048576).toFixed(1) + ' MB';
    if (b >= 1024) return (b / 1024).toFixed(1) + ' KB';
    return b + ' B';
  }

  const categories = [
    { key: 'video',    label: 'Vídeo',      color: '#3b82f6' },
    { key: 'audio',    label: 'Audio',       color: '#f59e0b' },
    { key: 'image',    label: 'Imágenes',    color: '#10b981' },
    { key: 'document', label: 'Documentos',  color: '#8b5cf6' },
    { key: 'other',    label: 'Otros',       color: '#64748b' },
  ];

  function getDonutSegments(stats, pool) {
    if (!stats) return [];
    // Total used across all categories
    const totalUsed = Object.values(stats).reduce((a, b) => a + b, 0);
    if (totalUsed <= 0) return [];
    const circumference = 2 * Math.PI * 48; // r=48
    let offset = 0;
    const segs = [];
    for (const cat of categories) {
      const val = stats[cat.key] || 0;
      if (val <= 0) continue;
      const pct = val / totalUsed; // relative to used, not total — donut always full
      const len = pct * circumference;
      segs.push({ ...cat, value: val, dasharray: `${len} ${circumference}`, offset: -offset });
      offset += len;
    }
    return segs;
  }

  function getBarSegments(stats, total) {
    if (!stats) return [];
    const totalUsed = Object.values(stats).reduce((a, b) => a + b, 0);
    if (totalUsed <= 0) return [];
    // Bar width = usagePercent of total disk, segments = proportional within used
    return categories.map(c => ({
      ...c,
      value: stats[c.key] || 0,
      pct: ((stats[c.key] || 0) / totalUsed) * 100
    })).filter(s => s.pct > 0);
  }

  // ── System-level status (across all pools) ──
  $: totalDisks = pools.reduce((n, p) => n + (p.disks?.length || 0), 0);
  $: problemDisks = pools.flatMap(p => (p.disks || []).filter(d => d.smartStatus !== 'ok'));
  $: systemStatus = pools.some(p => poolStatus(p) === 'crit') ? 'crit'
                   : pools.some(p => poolStatus(p) === 'warn') ? 'warn' : 'ok';
  $: systemTitle = systemStatus === 'crit' ? 'Grupo de almacenamiento en riesgo'
                 : systemStatus === 'warn' ? 'Un volumen necesita tu atención'
                 : 'El sistema funciona correctamente';
  $: systemSub = `${pools.length} volumen${pools.length !== 1 ? 'es' : ''} · ${totalDisks} discos · ${
    problemDisks.length === 0 ? 'sin incidencias' : problemDisks.length + ' disco' + (problemDisks.length > 1 ? 's' : '') + ' con avisos'
  }`;

  let refreshInterval;
  onMount(() => { load(); refreshInterval = setInterval(load, 30000); });
  onDestroy(() => { if (refreshInterval) clearInterval(refreshInterval); });

  // Keep detailPool in sync after refresh
  $: if (detailPool && pools.length) {
    detailPool = pools.find(p => p.name === detailPool.name) || null;
  }

  function redundancyLabel(pool) {
    const r = pool.poolHealth?.redundancy;
    if (!r) return '—';
    const typeLabel = { raidz1:'Protección simple', raidz2:'Protección doble', raidz3:'Protección triple', mirror:'Espejo', single:'Sin protección', stripe:'Sin protección' };
    const tl = typeLabel[r.type] || r.type;
    return tl;
  }
  function redundancySub(pool) {
    const r = pool.poolHealth?.redundancy;
    if (!r) return '';
    return `${r.current}/${r.expected} discos · puede perder ${r.canLose}`;
  }

  // ── Pool actions ──
  async function startScrub(poolName) {
    try {
      const res = await fetch('/api/storage/scrub', {
        method: 'POST',
        headers: { ...hdrs(), 'Content-Type': 'application/json' },
        body: JSON.stringify({ pool: poolName }),
      });
      const data = await res.json();
      if (data.ok) alert('Verificación de integridad iniciada');
      else alert(data.error || 'Error');
    } catch (e) { alert('Error: ' + e.message); }
  }

  async function createSnapshot(poolName) {
    try {
      const res = await fetch('/api/storage/snapshot', {
        method: 'POST',
        headers: { ...hdrs(), 'Content-Type': 'application/json' },
        body: JSON.stringify({ pool: poolName }),
      });
      const data = await res.json();
      if (data.ok) alert('Punto de restauración creado');
      else alert(data.error || 'Error');
    } catch (e) { alert('Error: ' + e.message); }
  }

  async function destroyPool(poolName) {
    const confirm1 = prompt(`Para destruir "${poolName}", escribe ELIMINAR:`);
    if (confirm1 !== 'ELIMINAR') return;
    try {
      const res = await fetch('/api/storage/pool/destroy', {
        method: 'POST',
        headers: { ...hdrs(), 'Content-Type': 'application/json' },
        body: JSON.stringify({ name: poolName }),
      });
      const data = await res.json();
      if (data.ok) { detailPool = null; await load(); }
      else alert(data.error || 'Error al destruir');
    } catch (e) { alert('Error: ' + e.message); }
  }
</script>

<AppShell title="Almacenamiento" {appIcon} {sections} bind:active showSearch>

  {#if loading && pools.length === 0}
    <div class="state-msg">Cargando...</div>

  {:else if active === 'resumen'}
    {#if detailPool}
      <!-- ═══ GESTIONAR POOL ═══ -->
      <div class="page-header">
        <!-- svelte-ignore a11y_click_events_have_key_events -->
        <!-- svelte-ignore a11y_no_static_element_interactions -->
        <div class="back-btn" on:click={() => detailPool = null}>
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" width="14" height="14"><polyline points="15 18 9 12 15 6"/></svg>
          Volver
        </div>
        <div class="header-title">
          <span class="pool-name">{detailPool.name}</span>
          <Badge status={poolStatus(detailPool)}>{poolStatusLabel(detailPool)}</Badge>
        </div>
      </div>

      <!-- Info + Actions row -->
      <div class="row-top">
        <Card>
          <SectionLabel>Información</SectionLabel>
          <div class="info-grid">
            <div class="info-k">Nombre</div><div class="info-v">{detailPool.name}</div>
            <div class="info-k">Sistema de archivos</div><div class="info-v">{detailPool.type?.toUpperCase()}</div>
            <div class="info-k">Protección</div><div class="info-v">{redundancyLabel(detailPool)}<span class="info-sub">{redundancySub(detailPool)}</span></div>
            <div class="info-k">Punto de montaje</div><div class="info-v mono">{detailPool.mountPoint}</div>
            <div class="info-k">Estado</div><div class="info-v">{detailPool.health}</div>
            <div class="info-k">Compresión</div><div class="info-v">lz4</div>
            <div class="info-k">Creado</div><div class="info-v mono">{detailPool.createdAt?.split('T')[0] || '—'}</div>
          </div>
        </Card>

        <Card>
          <SectionLabel>Acciones</SectionLabel>
          <div class="actions-col">
            <Button variant="primary" on:click={() => startScrub(detailPool.name)}>
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" width="15" height="15"><path d="M3 12a9 9 0 1 0 3-6.7"/><polyline points="3 4 3 10 9 10"/></svg>
              Verificar integridad
            </Button>
            <Button on:click={() => createSnapshot(detailPool.name)}>
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" width="15" height="15"><path d="M12 3v12"/><polyline points="7 8 12 3 17 8"/><path d="M5 21h14"/></svg>
              Punto de restauración
            </Button>
            <div class="btn-divider"></div>
            <Button variant="danger" on:click={() => destroyPool(detailPool.name)}>
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" width="15" height="15"><polyline points="3 6 5 6 21 6"/><path d="M19 6l-1 14a2 2 0 0 1-2 2H8a2 2 0 0 1-2-2L5 6"/><path d="M10 11v6"/><path d="M14 11v6"/></svg>
              Destruir volumen
            </Button>
          </div>
        </Card>
      </div>

      <!-- Disks table full width -->
      {#if detailPool.disks?.length > 0}
        <Card>
          <SectionLabel>Discos del volumen</SectionLabel>
          <div class="dtable-head dtable-8col">
            <div></div><div>Modelo</div><div>Dispositivo</div><div>Capacidad</div><div>Temp</div><div>Horas</div><div>Rol</div><div>Estado</div>
          </div>
          {#each detailPool.disks as disk}
            <div class="dtable-row dtable-8col">
              <div class="disk-icon">
                <svg viewBox="0 0 24 24" fill="currentColor">
                  <path d="M18.84 13.38c1.13 0 2.14.45 2.9 1.18L19.37 5.18C18.84 3.54 17.9 3 16.74 3H7.26C6.1 3 5.16 3.54 4.63 5.18L2.27 14.56c.75-.73 1.76-1.18 2.89-1.18z"/>
                  <path d="M5.16 14.4C4 14.4 2.96 15.07 2.41 16.08c-.26.48-.41 1.03-.41 1.62C2 19.55 3.44 21 5.16 21h13.68c1.72 0 3.16-1.45 3.16-3.3 0-.59-.15-1.14-.41-1.62-.55-1.01-1.58-1.68-2.75-1.68z"/>
                </svg>
              </div>
              <div class="d-name">{disk.model || disk.name}</div>
              <div class="d-cell">/dev/{disk.name}</div>
              <div class="d-cell">{disk.size || '—'}</div>
              <div class="d-cell" class:temp-hot={disk.smart?.temperature > 50}>{disk.smart?.temperature ? disk.smart.temperature + '°C' : '—'}</div>
              <div class="d-cell">{formatHours(disk.smart?.powerOnHours)}</div>
              <div><span class="role-pill">data</span></div>
              <div><Badge status={diskStatus(disk)}>{diskStatusLabel(disk)}</Badge></div>
            </div>
          {/each}
        </Card>
      {/if}

    {:else}
      <!-- ═══ RESUMEN (pool list) ═══ -->
      <!-- System status hero -->
    {#if pools.length > 0}
      <Card>
        <StatusHero
          status={systemStatus}
          title={systemTitle}
          subtitle={systemSub}
          stats={[
            { label: 'Volúmenes', value: String(pools.length) },
            { label: 'Discos', value: String(totalDisks) },
          ]}
        />
        {#if problemDisks.length > 0}
          <div class="status-disks">
            <SectionLabel>Discos con incidencias</SectionLabel>
            {#each problemDisks as disk}
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
                <div class="d-cell" class:temp-hot={disk.smart?.temperature > 50}>
                  {disk.smart?.temperature ? disk.smart.temperature + '°C' : '—'}
                </div>
                <div class="d-cell">{formatHours(disk.smart?.powerOnHours)}</div>
                <div><Badge status={diskStatus(disk)}>{diskStatusLabel(disk)}</Badge></div>
              </div>
            {/each}
          </div>
        {/if}
      </Card>

      <div style="height:14px"></div>
    {/if}

    <!-- Pool cards -->
    {#each pools as pool}
      <Card>
        <!-- Header -->
        <div class="pool-header">
          <div>
            <div class="pool-name">{pool.name}</div>
            <div class="pool-sub">{vdevLabel(pool.vdevType)} · {pool.type?.toUpperCase()} · {pool.disks?.length || 0} discos</div>
          </div>
          <Badge status={poolStatus(pool)}>{poolStatusLabel(pool)}</Badge>
        </div>

        <!-- Capacity with segmented bar -->
        <div class="capacity">
          <div class="cap-row">
            <div class="bar-track">
              <div class="bar-used" style="width:{Math.max(pool.usagePercent || 0, 1)}%;display:flex">
                {#each getBarSegments(poolFileStats[pool.name], pool.total) as seg}
                  <div class="bar-seg" style="width:{seg.pct}%;background:{seg.color}" title="{seg.label}: {formatBytes(seg.value)}"></div>
                {/each}
                {#if !poolFileStats[pool.name] || getBarSegments(poolFileStats[pool.name], pool.total).length === 0}
                  <div class="bar-seg" style="width:100%;background:var(--accent)"></div>
                {/if}
              </div>
            </div>
            <div class="cap-pct" class:warn={pool.usagePercent > 80} class:crit={pool.usagePercent > 95}>
              {pool.usagePercent || 0}<span class="sym">%</span>
            </div>
          </div>
          <div class="cap-info">
            <span class="mono">{pool.usedFormatted || '0 B'} usados</span>
            <span class="mono muted">{pool.totalFormatted || '—'}</span>
          </div>
        </div>

        <!-- Actions + Donut -->
        <div class="pool-bottom">
          <div class="pool-actions">
            <Button on:click={() => { detailPool = pool; }}>Gestionar</Button>
            <Button variant="primary" on:click={() => createSnapshot(pool.name)}>+ Punto de restauración</Button>
          </div>

          {#if poolFileStats[pool.name]}
            <div class="donut-wrap">
              <div class="legend">
                {#each categories as cat}
                  {#if (poolFileStats[pool.name]?.[cat.key] || 0) > 0}
                    <div class="legend-row">
                      <span class="legend-dot" style="background:{cat.color}"></span>
                      <span>{cat.label}</span>
                      <span class="legend-size">{formatBytes(poolFileStats[pool.name][cat.key])}</span>
                    </div>
                  {/if}
                {/each}
              </div>
              <div class="donut">
                <svg width="120" height="120" viewBox="0 0 120 120">
                  <circle cx="60" cy="60" r="48" fill="none" stroke="var(--bg-elev-2)" stroke-width="14"/>
                  {#each getDonutSegments(poolFileStats[pool.name], pool) as seg}
                    <circle cx="60" cy="60" r="48" fill="none" stroke="{seg.color}" stroke-width="14"
                      stroke-dasharray="{seg.dasharray}" stroke-dashoffset="{seg.offset}"
                      style="transform:rotate(-90deg);transform-origin:50% 50%"/>
                  {/each}
                </svg>
                <div class="donut-center">
                  <div class="donut-val">{pool.totalFormatted || '—'}</div>
                  <div class="donut-lbl">TOTAL</div>
                </div>
              </div>
            </div>
          {/if}
        </div>

        <!-- SMART disk table -->
        {#if pool.disks?.length > 0}
          <div class="smart-section">
            <SectionLabel>Estado SMART</SectionLabel>
            <div class="dtable-head">
              <div></div><div>Modelo</div><div>Dispositivo</div><div>Capacidad</div><div>Temp</div><div>Horas</div><div>Estado</div>
            </div>
            {#each pool.disks as disk}
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
                <div class="d-cell" class:temp-hot={disk.smart?.temperature > 50}>
                  {disk.smart?.temperature ? disk.smart.temperature + '°C' : '—'}
                </div>
                <div class="d-cell">{formatHours(disk.smart?.powerOnHours)}</div>
                <div><Badge status={diskStatus(disk)}>{diskStatusLabel(disk)}</Badge></div>
              </div>
            {/each}
          </div>
        {/if}

        <!-- Diagnostics -->
        {#if pool.poolHealth?.diagnostics?.length > 0}
          <div class="diag-section">
            <SectionLabel>Diagnóstico</SectionLabel>
            {#each pool.poolHealth.diagnostics as diag}
              <div class="diag-row">
                <div class="diag-dot" class:crit={diag.severity >= 4} class:warn={diag.severity >= 2 && diag.severity < 4}></div>
                <span>{diag.detail}</span>
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
  .bar-used { height:100%; display:flex; overflow:hidden; border-radius:8px; }
  .bar-seg { height:100%; transition:filter 0.2s; min-width:2px; }
  .bar-seg:hover { filter:brightness(1.3); }
  .cap-pct {
    font-size:42px; font-weight:700; letter-spacing:-1.5px; line-height:1;
    color:var(--c-ok); white-space:nowrap; font-family:var(--font-mono);
  }
  .cap-pct.warn { color:var(--c-warn); }
  .cap-pct.crit { color:var(--c-crit); }
  .cap-pct .sym { font-size:28px; font-weight:600; margin-left:2px; }
  .cap-info { display:flex; justify-content:space-between; margin-top:12px; font-size:13px; max-width:62%; }
  .mono { font-family:var(--font-mono); color:var(--text-primary); }
  .muted { color:var(--text-muted); }

  .pool-bottom {
    display:flex; justify-content:space-between; align-items:flex-end;
    border-top:1px solid var(--glass-border); padding-top:20px; margin-bottom:22px;
  }
  .pool-actions { display:flex; gap:10px; }

  /* Donut + legend */
  .donut-wrap { display:flex; align-items:center; gap:18px; }
  .legend { display:flex; flex-direction:column; gap:8px; }
  .legend-row {
    display:flex; align-items:center; gap:8px;
    font-size:12px; color:var(--text-primary);
  }
  .legend-dot { width:9px; height:9px; border-radius:50%; flex-shrink:0; }
  .legend-size { font-family:var(--font-mono); font-size:11px; color:var(--text-muted); margin-left:4px; }

  .donut { position:relative; width:120px; height:120px; }
  .donut svg { display:block; }
  .donut-center {
    position:absolute; inset:0;
    display:flex; flex-direction:column;
    align-items:center; justify-content:center;
    pointer-events:none;
  }
  .donut-val { font-family:var(--font-mono); font-size:18px; font-weight:600; color:var(--text-primary); }
  .donut-lbl { font-size:10px; color:var(--text-muted); margin-top:2px; text-transform:uppercase; letter-spacing:0.5px; }

  .smart-section { border-top:1px solid var(--glass-border); padding-top:18px; }

  .dtable-head, .dtable-row {
    display:grid;
    grid-template-columns:28px 1.6fr 1.2fr 0.7fr 0.6fr 0.7fr auto;
    gap:12px; padding:0;
    width:100%;
  }
  .dtable-head {
    padding-bottom:8px;
    font-size:10px; color:var(--text-muted); text-transform:uppercase; letter-spacing:0.8px;
    align-items:center;
  }
  .dtable-row {
    align-items:center; padding-top:9px; padding-bottom:9px;
    border-radius:8px; font-size:12px; cursor:pointer; transition:background 0.15s;
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

  .diag-section { border-top:1px solid var(--glass-border); padding-top:18px; margin-top:18px; }
  .diag-row {
    display:flex; align-items:center; gap:10px;
    padding:8px 12px; border-radius:6px; font-size:12px;
    color:var(--text-secondary);
  }
  .diag-dot {
    width:8px; height:8px; border-radius:50%; flex-shrink:0;
    background:var(--c-info);
  }
  .diag-dot.warn { background:var(--c-warn); }
  .diag-dot.crit { background:var(--c-crit); }

  .state-msg { font-size:13px; color:var(--text-muted); padding:20px 0; text-align:center; }

  .status-disks {
    margin-top:20px; padding-top:18px;
    border-top:1px solid var(--glass-border);
  }

  /* ── Gestionar view ── */
  .page-header {
    display:flex; align-items:center; gap:20px; margin-bottom:24px;
  }
  .back-btn {
    display:inline-flex; align-items:center; gap:6px;
    font-size:13px; color:var(--text-secondary);
    cursor:pointer; padding:6px 10px; border-radius:6px;
    transition:all 0.15s; background:transparent;
  }
  .back-btn:hover { color:var(--text-primary); background:var(--bg-elev-2); }
  .header-title { display:flex; align-items:center; gap:14px; }

  .row-top {
    display:grid; grid-template-columns:1.4fr 1fr;
    gap:18px; margin-bottom:18px;
  }

  .info-grid {
    display:grid; grid-template-columns:140px 1fr;
    gap:13px 16px; font-size:13px;
  }
  .info-k { color:var(--text-secondary); }
  .info-v { color:var(--text-primary); font-weight:500; text-align:right; }
  .info-v.mono { font-family:var(--font-mono); font-weight:400; }
  .info-sub {
    display:block; font-size:11px; color:var(--text-muted);
    font-weight:400; margin-top:2px;
  }

  .actions-col { display:flex; flex-direction:column; gap:10px; }
  .btn-divider { height:1px; background:var(--glass-border); margin:6px 0; }

  .role-pill {
    display:inline-block; font-size:10px;
    padding:3px 8px; border-radius:5px;
    background:var(--bg-elev-2); color:var(--text-secondary);
    font-family:var(--font-mono); text-transform:uppercase; letter-spacing:0.5px;
  }

  /* 8-column disk table for gestionar */
  .dtable-8col {
    grid-template-columns:28px 1.5fr 0.9fr 0.6fr 0.5fr 0.55fr 0.5fr auto;
  }
</style>
