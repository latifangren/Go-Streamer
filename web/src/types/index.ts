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
  target_platform: 'youtube' | 'facebook' | 'twitch' | 'custom' | string;
  rtmp_url: string;
  stream_key?: string;
  mode?: string;
  encoding_mode?: 'passthrough' | 'transcode' | 'transcode_720p';
  source_type?: string;
  video_id?: string;
  quality?: string;
  preset?: string;
  loop_playback?: boolean;
  max_duration_minutes?: number;
  auto_restart?: boolean;
  enable_overlay?: boolean;
  overlay_clock_wib?: boolean;
  overlay_watermark?: string;
  overlay_watermark_pos?: string;
  last_pts_sync_ms?: number;
  last_keyframe_gop_s?: number;
  last_net_latency_ms?: number;
  last_snapshot_path?: string;
  last_snapshot_at?: string;
  created_at?: string;
  updated_at?: string;
  playlist: PlaylistItem[];
  video?: VideoItem;
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
  created_at?: string;
  slot?: Slot;
}

export interface TunnelSettings {
  id?: number;
  provider: 'cloudflare' | 'tailscale';
  mode: 'quick' | 'named';
  is_active: boolean;
  public_url: string;
  tailscale_ip?: string;
  last_status: string;
  updated_at?: string;
}

export interface AlertSettings {
  id?: number;
  telegram_enabled: boolean;
  telegram_bot_token?: string;
  telegram_chat_id: string;
  has_telegram_token?: boolean;
  discord_enabled: boolean;
  discord_webhook_url?: string;
  has_discord_webhook?: boolean;
  trigger_on_crash: boolean;
  trigger_on_thermal: boolean;
  thermal_threshold_c: number;
  trigger_on_low_storage: boolean;
  low_storage_threshold_gb: number;
  last_alert_sent_at?: string;
  updated_at?: string;
}

export interface LogEntry {
  id: string;
  tag: 'SYS' | 'CONCAT' | 'OVERLAY' | 'RTMP' | 'ALERT' | 'ERR';
  timestamp: string;
  message: string;
}

export interface TranscodeJob {
  id: string;
  source_video_id: string;
  target_file_path: string;
  status: 'queued' | 'processing' | 'completed' | 'failed';
  progress_percent: number;
  error_message?: string;
  created_at: string;
  updated_at: string;
  source_video?: VideoItem;
}

export interface PlatformInfo {
  os: string;
  arch: string;
  is_root: boolean;
  is_termux: boolean;
  num_cpu: number;
  ffmpeg_path: string;
  ffprobe_path: string;
  version: string;
}

export interface SystemInfo {
  platform: PlatformInfo;
  config: {
    port: number;
    data_dir: string;
    video_dir: string;
    cache_dir: string;
    debug: boolean;
  };
}

export type ActiveTab = 'desk' | 'video' | 'scheduler' | 'remote';
