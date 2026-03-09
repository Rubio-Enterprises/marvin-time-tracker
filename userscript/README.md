# Marvin Relay Tracker Userscript

A Tampermonkey/Greasemonkey userscript that overlays tracking controls on the [Amazing Marvin](https://app.amazingmarvin.com) web UI, synced with the relay server via Server-Sent Events (SSE).

## Why?

When tracking is stopped from the iOS app, the Marvin web UI still shows tracking as active due to localStorage desync. This userscript eliminates that problem by communicating directly with the relay server and receiving real-time state updates.

## Features

- **Real-time sync** — SSE connection to the relay server for instant state updates
- **Floating panel** — Shows current tracking status, task title, and elapsed time
- **Start buttons** — Injected into task elements for one-click tracking
- **Stop control** — Stop tracking from the overlay panel
- **Optimistic UI** — Immediate visual feedback, confirmed by server events
- **Fallback polling** — Polls `/status` every 5s when SSE is unavailable
- **Native button hiding** — Optionally hide Marvin's built-in tracking UI
- **Shadow DOM** — Fully isolated styles, no conflicts with Marvin's UI

## Installation

1. Install [Tampermonkey](https://www.tampermonkey.net/) (Chrome/Firefox/Safari/Edge)
2. Open the userscript file `marvin-relay-tracker.user.js`
3. Tampermonkey should detect it and offer to install — click **Install**
4. Navigate to `https://app.amazingmarvin.com`
5. The "Relay Tracker" panel appears in the bottom-right corner

## Configuration

On first run, the settings panel opens automatically.

### Relay Server URL

Set this to your relay server address (e.g., `http://192.168.1.100:8080`).

Default: `http://localhost:8080`

### Hide Native Buttons

Toggle to hide Marvin's built-in time tracking buttons, reducing confusion when using the relay tracker.

### Security: `@connect`

The userscript uses `@connect *` to allow connections to any host, since the relay server address varies. To restrict this:

1. Edit the metadata block in the script
2. Replace `@connect *` with your specific server, e.g., `@connect 192.168.1.100`

## How It Works

```
Marvin Web UI
  ├── Userscript injects ▶ buttons into task elements
  ├── Floating panel shows tracking state
  └── EventSource connects to relay server /events endpoint
        ├── Receives: state, tracking_started, tracking_stopped events
        └── Fallback: polls GET /status every 5s if SSE disconnects
```

### SSE Events

| Event | Description |
|-------|-------------|
| `state` | Full state snapshot (sent on initial connection) |
| `tracking_started` | Task tracking began (includes taskId, taskTitle, startedAt) |
| `tracking_stopped` | Task tracking ended (includes taskId) |

### API Calls (via GM.xmlHttpRequest)

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/start` | Start tracking a task |
| `POST` | `/stop` | Stop tracking |
| `GET` | `/status` | Get current status (fallback polling) |

## Server Requirements

The relay server must have the SSE endpoint enabled (added in the same release as this userscript). The `/events` endpoint:

- Streams Server-Sent Events
- Sends an initial `state` event with the current tracking state
- Broadcasts `tracking_started` and `tracking_stopped` events in real time
- Sends keepalive comments every 30 seconds
- CORS is already configured to allow `*` origins

### Testing the SSE endpoint

```bash
curl -N http://localhost:8080/events
# Should receive: event: state, then keepalive comments every 30s
```

## Troubleshooting

- **Panel shows "Disconnected"** — Check that the relay server is running and the URL is correct in settings
- **No ▶ buttons on tasks** — The script waits for Marvin's DOM to load; try refreshing. Buttons appear on `div[data-item-id][data-item-type="task"]` elements.
- **SSE not connecting** — Verify CORS is enabled on the server. Check browser console for errors.
- **Settings not persisting** — Ensure Tampermonkey has storage permissions for the script
