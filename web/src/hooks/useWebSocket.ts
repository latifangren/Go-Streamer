import { useState, useEffect, useRef, useCallback } from 'react';
import { SystemMetrics, StreamTelemetry, LogEntry } from '../types';

interface RawWebSocketMessage {
  event?: string;
  type?: string;
  data?: any;
  timestamp?: number | string;
}

const initialMetrics: SystemMetrics = {
  cpu_percent: 18.4,
  ram_used_mb: 2940,
  ram_total_mb: 6144,
  ram_free_mb: 3204,
  disk_free_gb: 42.8,
  disk_total_gb: 64.0,
  temperature_c: 41.2,
  is_thermal_throttled: false,
  net_rx_kbps: 120.5,
  net_tx_kbps: 2450.0,
  uptime_seconds: 384920,
  timestamp: new Date().toISOString(),
};

const initialTelemetries: Record<number, StreamTelemetry> = {
  1: {
    slot_number: 1,
    status: 'running',
    frame: 14920,
    fps: 30.0,
    bitrate_kbps: 2420.5,
    duration: '04:12:35',
    speed: '1.00x',
    dropped_frames: 0,
    pts_sync_ms: 0.0,
    keyframe_cadence: '2.00s GOP',
    net_latency_ms: 38,
    timestamp: new Date().toISOString(),
  },
  2: {
    slot_number: 2,
    status: 'idle',
    frame: 0,
    fps: 0,
    bitrate_kbps: 0,
    duration: '00:00:00',
    speed: '0.00x',
    dropped_frames: 0,
    pts_sync_ms: 0.0,
    keyframe_cadence: 'Standby',
    net_latency_ms: 0,
    timestamp: new Date().toISOString(),
  },
};

const initialLogs: LogEntry[] = [
  {
    id: '1',
    tag: 'SYS',
    timestamp: '08:00:01',
    message: 'Ingest node spawned PGID:4192, worker uid=0, renice -10',
  },
  {
    id: '2',
    tag: 'CONCAT',
    timestamp: '08:00:02',
    message: 'Loaded playlist /storage/emulated/0/stream_pool/playlist.txt [3 files, auto-loop: enabled]',
  },
  {
    id: '3',
    tag: 'CONCAT',
    timestamp: '08:00:02',
    message: 'Stream #0:0: Video: h264 (High) (avc1), yuv420p, 1920x1080 [SAR 1:1], 2400 kb/s, 30 fps',
  },
  {
    id: '4',
    tag: 'OVERLAY',
    timestamp: '08:00:02',
    message: 'Dynamic clock filter bound: %{localtime\\:%H\\:%M\\:%S} (alpha: 0.85)',
  },
  {
    id: '5',
    tag: 'RTMP',
    timestamp: '08:00:03',
    message: 'TCP Handshake OK -> rtmp://a.rtmp.youtube.com/live2 (latency: 38ms)',
  },
  {
    id: '6',
    tag: 'RTMP',
    timestamp: '08:00:04',
    message: 'Publishing stream payload: 2420 kb/s @ 30.00 fps [ZERO-FRAME-DROP]',
  },
];

export function useWebSocket() {
  const [metrics, setMetrics] = useState<SystemMetrics>(initialMetrics);
  const [telemetries, setTelemetries] = useState<Record<number, StreamTelemetry>>(initialTelemetries);
  const [logs, setLogs] = useState<LogEntry[]>(initialLogs);
  const [isConnected, setIsConnected] = useState<boolean>(false);
  const wsRef = useRef<WebSocket | null>(null);
  const reconnectTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const isMountedRef = useRef<boolean>(true);

  const connect = useCallback(() => {
    if (!isMountedRef.current) return;

    try {
      const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
      const host = window.location.host || 'localhost:8080';

      // Path WebSocket otomatis ke /api/v1/ws dengan fallback /ws
      const primaryWsUrl = `${protocol}//${host}/api/v1/ws`;
      const fallbackWsUrl = `${protocol}//${host}/ws`;

      let ws: WebSocket;
      try {
        ws = new WebSocket(primaryWsUrl);
      } catch {
        ws = new WebSocket(fallbackWsUrl);
      }
      wsRef.current = ws;

      let primaryFailed = false;

      ws.onopen = () => {
        if (!isMountedRef.current) return;
        setIsConnected(true);
        console.log('[WebSocket] Connected to Go-Streamer backend');
      };

      ws.onmessage = (event) => {
        if (!isMountedRef.current) return;
        try {
          const parsed: RawWebSocketMessage = JSON.parse(event.data);
          // Normalisasi parser event agar kompatibel dengan event maupun type
          const eventName = parsed.event || parsed.type;
          const payload = parsed.data;

          if (!eventName) return;

          switch (eventName) {
            case 'system_metrics':
            case 'metrics':
              if (payload) {
                setMetrics((prev) => ({
                  ...prev,
                  ...payload,
                  timestamp: payload.timestamp || new Date().toISOString(),
                }));
              }
              break;

            case 'stream_telemetry':
            case 'telemetry':
              if (payload && typeof payload.slot_number === 'number') {
                setTelemetries((prev) => ({
                  ...prev,
                  [payload.slot_number]: {
                    ...prev[payload.slot_number],
                    ...payload,
                    timestamp: payload.timestamp || new Date().toISOString(),
                  },
                }));
              }
              break;

            case 'slot_status_changed':
              if (payload && typeof payload.slot_number === 'number') {
                setTelemetries((prev) => ({
                  ...prev,
                  [payload.slot_number]: {
                    ...(prev[payload.slot_number] || {
                      slot_number: payload.slot_number,
                      frame: 0,
                      fps: 0,
                      bitrate_kbps: 0,
                      duration: '00:00:00',
                      speed: '0.00x',
                      dropped_frames: 0,
                      pts_sync_ms: 0,
                      keyframe_cadence: 'Standby',
                      net_latency_ms: 0,
                      timestamp: new Date().toISOString(),
                    }),
                    status: payload.status,
                  },
                }));
                const logTag: LogEntry['tag'] = payload.status === 'running' ? 'RTMP' : 'SYS';
                const timeNow = new Date().toLocaleTimeString('id-ID', { hour12: false });
                setLogs((prev) => [
                  {
                    id: Math.random().toString(),
                    tag: logTag,
                    timestamp: timeNow,
                    message: `Slot ${payload.slot_number} status changed to ${payload.status}`,
                  },
                  ...prev.slice(0, 99),
                ]);
              }
              break;

            case 'killswitch_activated':
              setTelemetries((prev) => {
                const next = { ...prev };
                Object.keys(next).forEach((k) => {
                  const sNum = Number(k);
                  if (next[sNum]) {
                    next[sNum].status = 'idle';
                    next[sNum].fps = 0;
                    next[sNum].bitrate_kbps = 0;
                  }
                });
                return next;
              });
              setLogs((prev) => [
                {
                  id: Math.random().toString(),
                  tag: 'ERR',
                  timestamp: new Date().toLocaleTimeString('id-ID', { hour12: false }),
                  message: 'EMERGENCY KILL SWITCH: all processes terminated.',
                },
                ...prev.slice(0, 99),
              ]);
              break;

            case 'log':
              if (payload) {
                setLogs((prev) => [payload, ...prev.slice(0, 99)]);
              }
              break;

            default:
              break;
          }
        } catch (err) {
          console.error('[WebSocket] Failed to parse incoming packet:', err);
        }
      };

      ws.onerror = () => {
        if (!primaryFailed && ws.url.includes('/api/v1/ws')) {
          primaryFailed = true;
          try {
            ws.close();
            const fallbackWs = new WebSocket(fallbackWsUrl);
            wsRef.current = fallbackWs;
            fallbackWs.onopen = ws.onopen;
            fallbackWs.onmessage = ws.onmessage;
            fallbackWs.onclose = ws.onclose;
            return;
          } catch {
            // fallback error
          }
        }
      };

      ws.onclose = () => {
        if (!isMountedRef.current) return;
        setIsConnected(false);
        wsRef.current = null;
        if (reconnectTimeoutRef.current) clearTimeout(reconnectTimeoutRef.current);
        reconnectTimeoutRef.current = setTimeout(() => {
          connect();
        }, 3000);
      };
    } catch {
      if (reconnectTimeoutRef.current) clearTimeout(reconnectTimeoutRef.current);
      reconnectTimeoutRef.current = setTimeout(() => {
        connect();
      }, 5000);
    }
  }, []);

  useEffect(() => {
    isMountedRef.current = true;
    connect();

    // Fallback heartbeat animation bila backend belum tersambung
    const interval = setInterval(() => {
      if (!isConnected) {
        setMetrics((prev) => ({
          ...prev,
          cpu_percent: Math.max(12, Math.min(38, +(prev.cpu_percent + (Math.random() * 2 - 1)).toFixed(1))),
          temperature_c: Math.max(39, Math.min(46, +(prev.temperature_c + (Math.random() * 0.4 - 0.2)).toFixed(1))),
          timestamp: new Date().toISOString(),
        }));

        setTelemetries((prev) => {
          if (!prev[1] || prev[1].status !== 'running') return prev;
          return {
            ...prev,
            1: {
              ...prev[1],
              frame: prev[1].frame + 30,
              fps: +(29.9 + Math.random() * 0.2).toFixed(1),
              bitrate_kbps: +(2410 + Math.random() * 30).toFixed(1),
              timestamp: new Date().toISOString(),
            },
          };
        });
      }
    }, 1000);

    return () => {
      isMountedRef.current = false;
      clearInterval(interval);
      if (reconnectTimeoutRef.current) clearTimeout(reconnectTimeoutRef.current);
      if (wsRef.current) {
        wsRef.current.close();
        wsRef.current = null;
      }
    };
  }, [connect, isConnected]);

  const sendMessage = useCallback((data: any) => {
    if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify(data));
    }
  }, []);

  const addLog = useCallback((tag: LogEntry['tag'], message: string) => {
    const newLog: LogEntry = {
      id: Math.random().toString(),
      tag,
      timestamp: new Date().toLocaleTimeString('id-ID', { hour12: false }),
      message,
    };
    setLogs((prev) => [newLog, ...prev.slice(0, 99)]);
  }, []);

  const clearLogs = useCallback(() => {
    setLogs([]);
  }, []);

  return {
    metrics,
    telemetries,
    setTelemetries,
    logs,
    addLog,
    clearLogs,
    isConnected,
    sendMessage,
  };
}
