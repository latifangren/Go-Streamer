export interface SystemMetrics {
  cpu_percent: number;
  ram_used_mb: number;
  ram_total_mb: number;
  ram_free_mb: number;
  disk_free_gb: number;
  disk_total_gb: number;
  temperature_c: number;
  is_thermal_throttled: boolean;
  net_rx_kbps: number;
  net_tx_kbps: number;
  uptime_seconds: number;
  timestamp: string;
}

export interface StreamTelemetry {
  slot_number: number;
  status: 'idle' | 'starting' | 'running' | 'stopping' | 'error';
  frame: number;
  fps: number;
  bitrate_kbps: number;
  duration: string;
  speed: string;
  dropped_frames: number;
  pts_sync_ms: number;
  keyframe_cadence: string;
  net_latency_ms: number;
  timestamp: string;
}

export interface PlaylistItem {
  id: string;
  slot_id: number;
  video_id: string;
  filename: string;
  order: number;
  status: 'playing' | 'next' | 'queued';
  duration_seconds: number;
}

export interface Slot {
  id: number;
  slot_number: number;
  name: string;
  status: 'idle' | 'starting' | 'running' | 'stopping' | 'error';
  target_platform: 'youtube' | 'facebook' | 'twitch' | 'custom';
  rtmp_url: string;
  stream_key?: string;
  mode?: string;
  encoding_mode?: 'passthrough' | 'transcode' | 'transcode_720p';
  overlay_clock_wib?: boolean;
  overlay_watermark?: string;
  overlay_watermark_pos?: string;
  last_snapshot_path?: string;
  last_snapshot_at?: string;
  playlist: PlaylistItem[];
  telemetry?: StreamTelemetry;
}

export interface VideoItem {
  id: string;
  filename: string;
  original_name: string;
  file_path: string;
  file_size: number;
  duration_seconds: number;
  resolution: string;
  video_codec: string;
  audio_codec: string;
  fps: number;
  gop_size: number;
  is_passthrough_ready: boolean;
  created_at: string;
}

export interface ScheduleItem {
  id: string;
  slot_id: number;
  title: string;
  cron_expr: string;
  duration_minutes: number;
  is_enabled: boolean;
  overlap_guard_policy: 'yield_priority' | 'terminate_prev' | 'deny_new';
  last_run_at?: string;
  next_run_at?: string;
}

export interface TunnelSettings {
  provider: 'cloudflare' | 'tailscale';
  mode: 'quick' | 'named';
  is_active: boolean;
  public_url: string;
  tailscale_ip?: string;
  last_status: string;
}

export interface AlertSettings {
  telegram_enabled: boolean;
  telegram_bot_token?: string;
  telegram_chat_id: string;
  discord_enabled: boolean;
  discord_webhook_url?: string;
  trigger_on_crash: boolean;
  trigger_on_thermal: boolean;
  thermal_threshold_c: number;
  trigger_on_low_storage: boolean;
  low_storage_threshold_gb: number;
}

export interface LogEntry {
  id: string;
  tag: 'SYS' | 'CONCAT' | 'OVERLAY' | 'RTMP' | 'ALERT' | 'ERR';
  timestamp: string;
  message: string;
}

export type ActiveTab = 'desk' | 'video' | 'scheduler' | 'remote';
