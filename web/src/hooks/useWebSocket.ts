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

const initialTelemetries: Record<number, StreamTelemetry> = {};

const initialLogs: LogEntry[] = [
  {
    id: 'boot-1',
    tag: 'SYS',
    timestamp: new Date().toLocaleTimeString('id-ID', { hour12: false }),
    message: 'Live node UI ready. Awaiting telemetry feed...',
  },
];

export function useWebSocket() {
  const [metrics, setMetrics] = useState<SystemMetrics>(initialMetrics);
  const [telemetries, setTelemetries] = useState<Record<number, StreamTelemetry>>(initialTelemetries);
  const [logs, setLogs] = useState<LogEntry[]>(initialLogs);
  const [isConnected, setIsConnected] = useState<boolean>(false);
  const isConnectedRef = useRef<boolean>(false);
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
        isConnectedRef.current = true;
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

            case 'slot_status_changed': {
              const sNum = Number(payload?.slot_number || payload?.slot_id);
              if (sNum) {
                const newStatus = payload.status || 'idle';
                setTelemetries((prev) => ({
                  ...prev,
                  [sNum]: {
                    ...(prev[sNum] || {
                      slot_number: sNum,
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
                    status: newStatus,
                    ...(newStatus === 'idle' || newStatus === 'error'
                      ? { fps: 0, bitrate_kbps: 0, speed: '0.00x' }
                      : {}),
                  },
                }));
                const logTag: LogEntry['tag'] =
                  newStatus === 'running' ? 'RTMP' : newStatus === 'error' ? 'ERR' : 'SYS';
                const timeNow = new Date().toLocaleTimeString('id-ID', { hour12: false });
                setLogs((prev) => [
                  {
                    id: Math.random().toString(),
                    tag: logTag,
                    timestamp: timeNow,
                    message: `Slot ${sNum} status changed to ${newStatus}`,
                  },
                  ...prev.slice(0, 99),
                ]);
              }
              break;
            }

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
                  message: payload?.message || 'EMERGENCY KILL SWITCH: all processes terminated.',
                },
                ...prev.slice(0, 99),
              ]);
              break;

            case 'alert_settings_updated': {
              const timeNow = new Date().toLocaleTimeString('id-ID', { hour12: false });
              setLogs((prev) => [
                {
                  id: Math.random().toString(),
                  tag: 'ALERT',
                  timestamp: timeNow,
                  message: 'Alert configuration updated successfully.',
                },
                ...prev.slice(0, 99),
              ]);
              break;
            }

            case 'tunnel_status_changed': {
              const timeNow = new Date().toLocaleTimeString('id-ID', { hour12: false });
              const isAct = Boolean(payload?.is_active);
              const prov = payload?.provider || 'Cloudflare';
              const pubUrl = payload?.public_url ? ` (${payload.public_url})` : '';
              setLogs((prev) => [
                {
                  id: Math.random().toString(),
                  tag: 'SYS',
                  timestamp: timeNow,
                  message: `Tunnel ${prov}: ${isAct ? 'CONNECTED' + pubUrl : 'DISCONNECTED'}`,
                },
                ...prev.slice(0, 99),
              ]);
              break;
            }

            case 'video_uploaded': {
              const timeNow = new Date().toLocaleTimeString('id-ID', { hour12: false });
              const vidName = payload?.original_name || payload?.filename || 'video file';
              setLogs((prev) => [
                {
                  id: Math.random().toString(),
                  tag: 'SYS',
                  timestamp: timeNow,
                  message: `Video uploaded: ${vidName}`,
                },
                ...prev.slice(0, 99),
              ]);
              break;
            }

            case 'codec_job_updated':
            case 'codec_job_created': {
              const timeNow = new Date().toLocaleTimeString('id-ID', { hour12: false });
              const jobStatus = payload?.status || 'processing';
              const progressStr =
                payload?.progress_percent !== undefined ? ` [${payload.progress_percent}%]` : '';
              setLogs((prev) => [
                {
                  id: Math.random().toString(),
                  tag: 'CONCAT',
                  timestamp: timeNow,
                  message: `Transcode job ${jobStatus}${progressStr}`,
                },
                ...prev.slice(0, 99),
              ]);
              break;
            }

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
        isConnectedRef.current = false;
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
      if (!isConnectedRef.current) {
        setMetrics((prev) => ({
          ...prev,
          cpu_percent: Math.max(12, Math.min(38, +(prev.cpu_percent + (Math.random() * 2 - 1)).toFixed(1))),
          temperature_c: Math.max(39, Math.min(46, +(prev.temperature_c + (Math.random() * 0.4 - 0.2)).toFixed(1))),
          timestamp: new Date().toISOString(),
        }));
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
  }, [connect]);

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
