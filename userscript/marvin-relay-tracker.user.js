// ==UserScript==
// @name         Marvin Relay Tracker
// @namespace    com.strubio.marvin-relay-tracker
// @version      0.1.0
// @description  Overlay tracking controls on Amazing Marvin, synced with relay server via SSE
// @match        https://app.amazingmarvin.com/*
// @grant        GM.xmlHttpRequest
// @grant        GM_addStyle
// @grant        GM_getValue
// @grant        GM_setValue
// @connect      *
// @run-at       document-idle
// ==/UserScript==

(function () {
  'use strict';

  // === Config ===

  const Config = {
    async getRelayUrl() {
      return (await GM_getValue('relayUrl', 'http://localhost:8080')).replace(/\/+$/, '');
    },
    async setRelayUrl(url) {
      await GM_setValue('relayUrl', url.replace(/\/+$/, ''));
    },
    async getHideNative() {
      return await GM_getValue('hideNative', false);
    },
    async setHideNative(val) {
      await GM_setValue('hideNative', val);
    },
    async getCollapsed() {
      return await GM_getValue('collapsed', false);
    },
    async setCollapsed(val) {
      await GM_setValue('collapsed', val);
    },
    async isFirstRun() {
      const url = await GM_getValue('relayUrl', null);
      return url === null;
    },
  };

  // === API ===

  const API = (() => {
    let pending = false;

    function gmRequest(method, url, data) {
      return new Promise((resolve, reject) => {
        GM.xmlHttpRequest({
          method,
          url,
          headers: data ? { 'Content-Type': 'application/json' } : undefined,
          data: data ? JSON.stringify(data) : undefined,
          timeout: 15000,
          onload(resp) {
            if (resp.status >= 200 && resp.status < 300) {
              try {
                resolve(JSON.parse(resp.responseText));
              } catch {
                resolve(resp.responseText);
              }
            } else {
              reject(new Error(`HTTP ${resp.status}: ${resp.responseText}`));
            }
          },
          onerror(err) {
            reject(new Error(err.error || 'Network error'));
          },
          ontimeout() {
            reject(new Error('Request timeout'));
          },
        });
      });
    }

    return {
      async startTracking(taskId, title) {
        if (pending) return;
        pending = true;
        try {
          const url = await Config.getRelayUrl();
          return await gmRequest('POST', `${url}/start`, { taskId, title });
        } finally {
          pending = false;
        }
      },

      async stopTracking(taskId) {
        if (pending) return;
        pending = true;
        try {
          const url = await Config.getRelayUrl();
          const body = taskId ? { taskId } : {};
          return await gmRequest('POST', `${url}/stop`, body);
        } finally {
          pending = false;
        }
      },

      async getStatus() {
        const url = await Config.getRelayUrl();
        return await gmRequest('GET', `${url}/status`);
      },
    };
  })();

  // === State ===

  const State = (() => {
    const listeners = [];
    const state = {
      tracking: false,
      taskId: null,
      taskTitle: null,
      startedAt: null,
      connected: false,
    };

    return {
      get() {
        return { ...state };
      },

      update(data) {
        if (data.tracking !== undefined) state.tracking = data.tracking;
        if (data.tracking) {
          if (data.taskId !== undefined) state.taskId = data.taskId;
          if (data.taskTitle !== undefined) state.taskTitle = data.taskTitle;
          if (data.startedAt !== undefined) state.startedAt = data.startedAt;
        } else if (data.tracking === false) {
          state.taskId = null;
          state.taskTitle = null;
          state.startedAt = null;
        }
        listeners.forEach((fn) => fn(state));
      },

      setConnected(val) {
        state.connected = val;
        listeners.forEach((fn) => fn(state));
      },

      onChange(fn) {
        listeners.push(fn);
      },
    };
  })();

  // === SSE ===

  const SSE = (() => {
    let eventSource = null;
    let pollTimer = null;

    function startPolling() {
      if (pollTimer) return;
      pollTimer = setInterval(async () => {
        try {
          const status = await API.getStatus();
          State.update({
            tracking: status.tracking,
            taskId: status.taskId || null,
            taskTitle: status.taskTitle || null,
            startedAt: status.startedAt || null,
          });
          State.setConnected(true);
        } catch {
          State.setConnected(false);
        }
      }, 5000);
    }

    function stopPolling() {
      if (pollTimer) {
        clearInterval(pollTimer);
        pollTimer = null;
      }
    }

    return {
      async connect() {
        this.disconnect();
        const url = await Config.getRelayUrl();

        try {
          eventSource = new EventSource(`${url}/events`);
        } catch {
          startPolling();
          return;
        }

        eventSource.addEventListener('state', (e) => {
          try {
            const data = JSON.parse(e.data);
            State.update({
              tracking: data.tracking,
              taskId: data.taskId || null,
              taskTitle: data.taskTitle || null,
              startedAt: data.startedAt || null,
            });
          } catch { /* ignore parse errors */ }
        });

        eventSource.addEventListener('tracking_started', (e) => {
          try {
            const data = JSON.parse(e.data);
            State.update({
              tracking: true,
              taskId: data.taskId || null,
              taskTitle: data.taskTitle || null,
              startedAt: data.startedAt || null,
            });
          } catch { /* ignore */ }
        });

        eventSource.addEventListener('tracking_stopped', () => {
          State.update({ tracking: false });
        });

        eventSource.onopen = () => {
          State.setConnected(true);
          stopPolling();
        };

        eventSource.onerror = () => {
          State.setConnected(false);
          startPolling();
        };
      },

      disconnect() {
        if (eventSource) {
          eventSource.close();
          eventSource = null;
        }
        stopPolling();
      },
    };
  })();

  // === UI ===

  const UI = (() => {
    let root = null;
    let shadow = null;
    let timerInterval = null;
    let elements = {};

    function formatElapsed(startedAtMs) {
      if (!startedAtMs) return '00:00:00';
      const diff = Math.max(0, Math.floor((Date.now() - startedAtMs) / 1000));
      const h = String(Math.floor(diff / 3600)).padStart(2, '0');
      const m = String(Math.floor((diff % 3600) / 60)).padStart(2, '0');
      const s = String(diff % 60).padStart(2, '0');
      return `${h}:${m}:${s}`;
    }

    function createPanel() {
      root = document.createElement('div');
      root.id = 'marvin-relay-root';
      shadow = root.attachShadow({ mode: 'closed' });

      const style = document.createElement('style');
      style.textContent = `
        :host {
          all: initial;
          font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
          font-size: 13px;
        }
        .panel {
          position: fixed;
          bottom: 16px;
          right: 16px;
          z-index: 999999;
          background: #1e1e2e;
          color: #cdd6f4;
          border-radius: 10px;
          box-shadow: 0 4px 20px rgba(0,0,0,0.4);
          min-width: 220px;
          overflow: hidden;
          transition: all 0.2s ease;
        }
        .header {
          display: flex;
          justify-content: space-between;
          align-items: center;
          padding: 8px 12px;
          background: #313244;
          cursor: pointer;
          user-select: none;
        }
        .header-title {
          font-weight: 600;
          font-size: 12px;
          text-transform: uppercase;
          letter-spacing: 0.5px;
        }
        .header-icons {
          display: flex;
          gap: 6px;
        }
        .icon-btn {
          background: none;
          border: none;
          color: #cdd6f4;
          cursor: pointer;
          font-size: 14px;
          padding: 2px;
          opacity: 0.7;
          transition: opacity 0.15s;
        }
        .icon-btn:hover { opacity: 1; }
        .body { padding: 12px; }
        .body.hidden { display: none; }
        .status {
          margin-bottom: 8px;
          padding: 8px;
          border-radius: 6px;
          text-align: center;
        }
        .status.idle { background: #45475a; }
        .status.tracking { background: #1e4620; }
        .status.disconnected { background: #5c2020; }
        .task-title {
          font-weight: 600;
          margin-bottom: 4px;
          word-break: break-word;
        }
        .elapsed {
          font-size: 20px;
          font-weight: 700;
          font-variant-numeric: tabular-nums;
          letter-spacing: 1px;
        }
        .controls { margin-top: 8px; }
        .btn {
          width: 100%;
          padding: 8px;
          border: none;
          border-radius: 6px;
          cursor: pointer;
          font-size: 13px;
          font-weight: 600;
          transition: background 0.15s;
        }
        .btn-stop { background: #f38ba8; color: #1e1e2e; }
        .btn-stop:hover { background: #eba0ac; }
        .btn:disabled { opacity: 0.5; cursor: not-allowed; }
        .settings { margin-top: 10px; border-top: 1px solid #45475a; padding-top: 10px; }
        .settings.hidden { display: none; }
        .settings label {
          display: block;
          font-size: 11px;
          margin-bottom: 4px;
          color: #a6adc8;
        }
        .settings input[type="text"] {
          width: 100%;
          box-sizing: border-box;
          padding: 6px 8px;
          background: #313244;
          border: 1px solid #45475a;
          border-radius: 4px;
          color: #cdd6f4;
          font-size: 12px;
          margin-bottom: 8px;
        }
        .settings input[type="text"]:focus {
          outline: none;
          border-color: #89b4fa;
        }
        .checkbox-row {
          display: flex;
          align-items: center;
          gap: 6px;
          font-size: 12px;
        }
        .error {
          color: #f38ba8;
          font-size: 11px;
          margin-top: 6px;
          text-align: center;
        }
        .dot {
          display: inline-block;
          width: 6px;
          height: 6px;
          border-radius: 50%;
          margin-right: 6px;
        }
        .dot.green { background: #a6e3a1; }
        .dot.red { background: #f38ba8; }
        .dot.gray { background: #6c7086; }
      `;

      const panel = document.createElement('div');
      panel.className = 'panel';
      panel.innerHTML = `
        <div class="header">
          <span class="header-title">Relay Tracker</span>
          <div class="header-icons">
            <button class="icon-btn settings-toggle" title="Settings">&#9881;</button>
            <button class="icon-btn collapse-toggle" title="Collapse">&#9660;</button>
          </div>
        </div>
        <div class="body">
          <div class="status idle">
            <div class="status-text">No active task</div>
          </div>
          <div class="controls hidden">
            <button class="btn btn-stop">Stop</button>
          </div>
          <div class="error hidden"></div>
          <div class="settings hidden">
            <label>Relay Server URL</label>
            <input type="text" class="relay-url" placeholder="http://localhost:8080">
            <div class="checkbox-row">
              <input type="checkbox" class="hide-native" id="mrt-hide-native">
              <label for="mrt-hide-native">Hide native tracking buttons</label>
            </div>
          </div>
        </div>
      `;

      shadow.appendChild(style);
      shadow.appendChild(panel);

      elements = {
        panel,
        body: panel.querySelector('.body'),
        status: panel.querySelector('.status'),
        statusText: panel.querySelector('.status-text'),
        controls: panel.querySelector('.controls'),
        stopBtn: panel.querySelector('.btn-stop'),
        error: panel.querySelector('.error'),
        settings: panel.querySelector('.settings'),
        settingsToggle: panel.querySelector('.settings-toggle'),
        collapseToggle: panel.querySelector('.collapse-toggle'),
        relayUrl: panel.querySelector('.relay-url'),
        hideNative: panel.querySelector('.hide-native'),
      };

      // Event handlers
      elements.collapseToggle.addEventListener('click', async (e) => {
        e.stopPropagation();
        const collapsed = elements.body.classList.toggle('hidden');
        elements.collapseToggle.innerHTML = collapsed ? '&#9650;' : '&#9660;';
        await Config.setCollapsed(collapsed);
      });

      elements.settingsToggle.addEventListener('click', (e) => {
        e.stopPropagation();
        elements.settings.classList.toggle('hidden');
      });

      elements.stopBtn.addEventListener('click', async () => {
        const current = State.get();
        // Optimistic: show idle immediately
        State.update({ tracking: false });
        try {
          await API.stopTracking(current.taskId);
        } catch (err) {
          // Revert on error
          State.update({
            tracking: true,
            taskId: current.taskId,
            taskTitle: current.taskTitle,
            startedAt: current.startedAt,
          });
          showError(err.message);
        }
      });

      let urlTimeout = null;
      elements.relayUrl.addEventListener('input', () => {
        clearTimeout(urlTimeout);
        urlTimeout = setTimeout(async () => {
          const url = elements.relayUrl.value.trim();
          if (url) {
            await Config.setRelayUrl(url);
            SSE.disconnect();
            SSE.connect();
          }
        }, 800);
      });

      elements.hideNative.addEventListener('change', async () => {
        const hide = elements.hideNative.checked;
        await Config.setHideNative(hide);
        DOM.applyNativeHiding(hide);
      });

      document.body.appendChild(root);
    }

    function showError(msg) {
      elements.error.textContent = msg;
      elements.error.classList.remove('hidden');
      setTimeout(() => elements.error.classList.add('hidden'), 5000);
    }

    function render(state) {
      // Update timer
      if (timerInterval) {
        clearInterval(timerInterval);
        timerInterval = null;
      }

      if (!state.connected) {
        elements.status.className = 'status disconnected';
        elements.status.innerHTML = '<span class="dot red"></span>Disconnected';
        elements.controls.classList.add('hidden');
      } else if (state.tracking) {
        elements.status.className = 'status tracking';
        elements.status.innerHTML = `
          <span class="dot green"></span>
          <div class="task-title">${escapeHtml(state.taskTitle || 'Unknown task')}</div>
          <div class="elapsed">${formatElapsed(state.startedAt)}</div>
        `;
        elements.controls.classList.remove('hidden');
        timerInterval = setInterval(() => {
          const el = shadow.querySelector('.elapsed');
          if (el) el.textContent = formatElapsed(state.startedAt);
        }, 1000);
      } else {
        elements.status.className = 'status idle';
        elements.status.innerHTML = '<span class="dot gray"></span>No active task';
        elements.controls.classList.add('hidden');
      }
    }

    function escapeHtml(str) {
      const div = document.createElement('div');
      div.textContent = str;
      return div.innerHTML;
    }

    return {
      async init() {
        createPanel();

        // Restore collapsed state
        const collapsed = await Config.getCollapsed();
        if (collapsed) {
          elements.body.classList.add('hidden');
          elements.collapseToggle.innerHTML = '&#9650;';
        }

        // Load settings
        elements.relayUrl.value = await Config.getRelayUrl();
        elements.hideNative.checked = await Config.getHideNative();

        // Show settings if first run
        if (await Config.isFirstRun()) {
          elements.settings.classList.remove('hidden');
        }

        return { render };
      },

      showError,
    };
  })();

  // === DOM ===

  const DOM = (() => {
    let observer = null;
    let onStartClick = null;
    let nativeStyleEl = null;

    const TASK_SELECTOR = 'div[data-item-id][data-item-type="task"]';
    const TITLE_SELECTOR = '.TitlePart';
    const BUTTON_CLASS = 'mrt-start-btn';

    const NATIVE_HIDE_CSS = `
      .trackingIcon,
      .timerButton,
      .timer-button,
      [class*="TrackingControls"],
      .time-tracking-button {
        display: none !important;
      }
    `;

    function injectButton(taskEl) {
      if (taskEl.querySelector(`.${BUTTON_CLASS}`)) return;

      const btn = document.createElement('button');
      btn.className = BUTTON_CLASS;
      btn.textContent = '\u25B6';
      btn.title = 'Start tracking (Relay)';
      Object.assign(btn.style, {
        background: 'none',
        border: '1px solid rgba(166,227,161,0.5)',
        borderRadius: '4px',
        color: '#a6e3a1',
        cursor: 'pointer',
        fontSize: '10px',
        padding: '2px 5px',
        marginLeft: '4px',
        opacity: '0.6',
        transition: 'opacity 0.15s',
        verticalAlign: 'middle',
      });
      btn.addEventListener('mouseenter', () => { btn.style.opacity = '1'; });
      btn.addEventListener('mouseleave', () => { btn.style.opacity = '0.6'; });

      btn.addEventListener('click', async (e) => {
        e.stopPropagation();
        e.preventDefault();

        const itemId = taskEl.getAttribute('data-item-id');
        const titleEl = taskEl.querySelector(TITLE_SELECTOR);
        const title = titleEl ? titleEl.textContent.trim() : 'Unknown';

        if (onStartClick) {
          await onStartClick(itemId, title);
        }
      });

      const titleEl = taskEl.querySelector(TITLE_SELECTOR);
      if (titleEl) {
        titleEl.parentElement.appendChild(btn);
      }
    }

    function scanAndInject(container) {
      const tasks = container.querySelectorAll(TASK_SELECTOR);
      tasks.forEach(injectButton);
    }

    return {
      init(startClickHandler) {
        onStartClick = startClickHandler;

        // Initial scan
        scanAndInject(document.body);

        // Observe for new task elements
        const target = document.querySelector('.List2Wrapper') || document.body;
        observer = new MutationObserver((mutations) => {
          for (const mutation of mutations) {
            for (const node of mutation.addedNodes) {
              if (node.nodeType !== Node.ELEMENT_NODE) continue;
              if (node.matches && node.matches(TASK_SELECTOR)) {
                injectButton(node);
              }
              if (node.querySelectorAll) {
                scanAndInject(node);
              }
            }
          }
        });
        observer.observe(target, { childList: true, subtree: true });
      },

      applyNativeHiding(hide) {
        if (hide && !nativeStyleEl) {
          nativeStyleEl = GM_addStyle(NATIVE_HIDE_CSS);
        } else if (!hide && nativeStyleEl) {
          nativeStyleEl.remove();
          nativeStyleEl = null;
        }
      },
    };
  })();

  // === Init ===

  async function init() {
    // 1. Create UI
    const { render } = await UI.init();

    // 2. Wire state changes to UI
    State.onChange(render);

    // 3. Wire DOM start clicks
    const handleStartClick = async (taskId, title) => {
      // Optimistic update
      State.update({
        tracking: true,
        taskId,
        taskTitle: title,
        startedAt: Date.now(),
      });

      try {
        const resp = await API.startTracking(taskId, title);
        // Use server startedAt if available
        if (resp && resp.startedAt) {
          State.update({
            tracking: true,
            taskId,
            taskTitle: title,
            startedAt: resp.startedAt,
          });
        }
      } catch (err) {
        // Revert on error
        State.update({ tracking: false });
        UI.showError(err.message);
      }
    };

    // 4. Connect SSE
    await SSE.connect();

    // 5. Wait for Marvin DOM, then start DOM observer
    const waitForMarvin = () => {
      return new Promise((resolve) => {
        const check = () => {
          if (document.querySelector('.List2Wrapper') || document.querySelector('[data-item-id]')) {
            resolve();
          } else {
            setTimeout(check, 500);
          }
        };
        check();
        // Give up after 30s and init anyway
        setTimeout(resolve, 30000);
      });
    };

    await waitForMarvin();
    DOM.init(handleStartClick);

    // 6. Apply native hiding if configured
    const hideNative = await Config.getHideNative();
    DOM.applyNativeHiding(hideNative);
  }

  init();
})();
