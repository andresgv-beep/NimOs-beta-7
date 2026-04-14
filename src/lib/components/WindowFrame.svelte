<script>
  import { onMount, tick } from 'svelte';
  import { closeWindow, focusWindow, minimizeWindow, maximizeWindow, updateWindowPos, getWindowPos } from '$lib/stores/windows.js';
  import { APP_META } from '$lib/apps.js';

  export let win;

  $: meta = APP_META[win.appId] || { name: win.appId, icon: '📦' };

  // Reactive position state for the style binding
  let x = 0, y = 0, w = 800, h = 520;

  onMount(async () => {
    await tick();
    const p = getWindowPos(win.id);
    x = p.x; y = p.y; w = p.width; h = p.height;
  });

  // Drag
  let dragging = false;
  let dragOffset = { x: 0, y: 0 };

  function getZoom() {
    return parseFloat(document.documentElement.style.zoom) || 1;
  }

  function onTitleMouseDown(e) {
    if (e.target.closest('.wf-btn')) return;
    if (win.maximized) return;
    focusWindow(win.id);
    dragging = true;
    const z = getZoom();
    dragOffset = { x: e.clientX / z - x, y: e.clientY / z - y };
    window.addEventListener('mousemove', onDrag);
    window.addEventListener('mouseup', onDragEnd);
  }

  function onDrag(e) {
    if (!dragging) return;
    const z = getZoom();
    x = e.clientX / z - dragOffset.x;
    y = Math.max(0, e.clientY / z - dragOffset.y);
    updateWindowPos(win.id, { x, y });
  }

  function onDragEnd() {
    dragging = false;
    window.removeEventListener('mousemove', onDrag);
    window.removeEventListener('mouseup', onDragEnd);
  }

  // Resize
  let resizing = false;
  let resizeStart = { mx: 0, my: 0, w: 0, h: 0 };

  function onResizeMouseDown(e) {
    if (win.maximized) return;
    e.stopPropagation();
    resizing = true;
    const z = getZoom();
    resizeStart = { mx: e.clientX / z, my: e.clientY / z, w, h };
    window.addEventListener('mousemove', onResize);
    window.addEventListener('mouseup', onResizeEnd);
  }

  function onResize(e) {
    if (!resizing) return;
    const z = getZoom();
    w = Math.max(400, resizeStart.w + (e.clientX / z - resizeStart.mx));
    h = Math.max(300, resizeStart.h + (e.clientY / z - resizeStart.my));
    updateWindowPos(win.id, { width: w, height: h });
  }

  function onResizeEnd() {
    resizing = false;
    window.removeEventListener('mousemove', onResize);
    window.removeEventListener('mouseup', onResizeEnd);
  }

  // Maximize
  function doMaximize() {
    maximizeWindow(win.id);
    tick().then(() => {
      const p = getWindowPos(win.id);
      x = p.x; y = p.y; w = p.width; h = p.height;
    });
  }
</script>

<!-- svelte-ignore a11y_no_static_element_interactions -->
<div
  class="window"
  class:maximized={win.maximized}
  class:dragging
  style="z-index:{win.zIndex}; left:{x}px; top:{y}px; width:{w}px; height:{h}px;"
  on:mousedown={() => focusWindow(win.id)}
>
  <!-- Drag zone — invisible bar at top for dragging -->
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div class="drag-zone" on:mousedown={onTitleMouseDown}></div>

  <!-- Window controls — right side icons -->
  {#if win.appId === 'transfermanager'}
    <div class="wf-controls">
      <button class="wf-btn" on:click={() => closeWindow(win.id)} title="Cerrar">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
      </button>
    </div>
  {:else}
    <div class="wf-controls">
      <button class="wf-btn" on:click={() => minimizeWindow(win.id)} title="Minimizar">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><line x1="5" y1="12" x2="19" y2="12"/></svg>
      </button>
      <button class="wf-btn" on:click={doMaximize} title="Maximizar">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><rect x="6" y="6" width="12" height="12" rx="2"/></svg>
      </button>
      <button class="wf-btn wf-close" on:click={() => closeWindow(win.id)} title="Cerrar">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
      </button>
    </div>
  {/if}

  <!-- App content — fills entire window -->
  <div class="content">
    {#if win.isWebApp && win.webAppPort}
      {#await import('$lib/apps/WebApp.svelte') then module}
        <svelte:component this={module.default} appId={win.appId} port={win.webAppPort} name={win.webAppName} />
      {/await}
    {:else if win.appId === 'files'}
      {#await import('$lib/apps/FileManager.svelte') then module}
        <svelte:component this={module.default} />
      {/await}
    {:else if win.appId === 'nimsettings'}
      {#await import('$lib/apps/Settings.svelte') then module}
        <svelte:component this={module.default} />
      {/await}
    {:else if win.appId === 'storage'}
      {#await import('$lib/apps/StorageApp.svelte') then module}
        <svelte:component this={module.default} />
      {/await}
    {:else if win.appId === 'network'}
      {#await import('$lib/apps/NetworkApp.svelte') then module}
        <svelte:component this={module.default} />
      {/await}
    {:else if win.appId === 'nimtorrent'}
      {#await import('$lib/apps/NimTorrent.svelte') then module}
        <svelte:component this={module.default} />
      {/await}
    {:else if win.appId === 'appstore'}
      {#await import('$lib/apps/AppStore.svelte') then module}
        <svelte:component this={module.default} />
      {/await}
    {:else if win.appId === 'mediaplayer'}
      {#await import('$lib/apps/MediaPlayer.svelte') then module}
        <svelte:component this={module.default} />
      {/await}
    {:else if win.appId === 'nimbackup'}
      {#await import('$lib/apps/NimBackup.svelte') then module}
        <svelte:component this={module.default} />
      {/await}
    {:else if win.appId === 'texteditor'}
      {#await import('$lib/apps/Notes.svelte') then module}
        <svelte:component this={module.default} />
      {/await}
    {:else if win.appId === 'nimhealth'}
      {#await import('$lib/apps/NimHealth.svelte') then module}
        <svelte:component this={module.default} />
      {/await}
    {:else if win.appId === 'transfermanager'}
      {#await import('$lib/apps/TransferManager.svelte') then module}
        <svelte:component this={module.default} />
      {/await}
    {:else}
      <div class="placeholder">
        <span style="font-size:48px">{meta.icon}</span>
        <p>{meta.name}</p>
        <small>Coming soon</small>
      </div>
    {/if}
  </div>

  {#if !win.maximized}
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div class="resize-handle" on:mousedown={onResizeMouseDown}></div>
  {/if}
</div>

<style>
  .window {
    position: fixed;
    border-radius: 12px;
    overflow: hidden;
    border: 1px solid var(--window-border);
    box-shadow: var(--window-shadow);
    display: flex; flex-direction: column;
    background: var(--bg-elev-1, #151823);
    animation: winIn 0.42s cubic-bezier(0.16,1,0.3,1) both;
  }
  .window.dragging { user-select: none; }
  .window.maximized {
    border-radius: 0 !important; border: none !important;
    box-shadow: none !important;
    left: 0 !important; top: 0 !important;
    width: 100vw !important;
    height: calc(100vh - var(--taskbar-height, 48px)) !important;
  }
  @keyframes winIn {
    from { opacity: 0; transform: scale(0.96) translateY(10px); }
    to { opacity: 1; transform: scale(1) translateY(0); }
  }

  .drag-zone {
    position: absolute; top: 0; left: 0; right: 0;
    height: 34px; z-index: 9998;
    cursor: default; user-select: none;
  }

  .wf-controls {
    position: absolute; top: 0; right: 0;
    display: flex; z-index: 9999;
    height: var(--titlebar-height, 40px);
    align-items: center;
    padding-right: 4px;
  }
  .wf-btn {
    width: 36px; height: 100%;
    border: none; background: transparent; padding: 0;
    color: var(--text-muted, rgba(255,255,255,0.35));
    cursor: pointer; display: flex; align-items: center; justify-content: center;
    transition: background 0.12s, color 0.12s;
  }
  .wf-btn svg { width: 14px; height: 14px; }
  .wf-btn:hover { background: rgba(255,255,255,0.06); color: var(--text-primary, rgba(255,255,255,0.9)); }
  .wf-close:hover { background: rgba(239,68,68,0.8); color: #fff; }

  .content { flex: 1; overflow: hidden; }

  .placeholder {
    width: 100%; height: 100%;
    display: flex; flex-direction: column;
    align-items: center; justify-content: center;
    gap: 8px; color: rgba(255,255,255,0.3);
    background: var(--bg-app, #0c0e14);
  }
  .placeholder p { font-size: 14px; font-weight: 500; }
  .placeholder small { font-size: 11px; }

  .resize-handle {
    position: absolute; bottom: 0; right: 0;
    width: 16px; height: 16px;
    cursor: nwse-resize; z-index: 10;
  }
</style>
