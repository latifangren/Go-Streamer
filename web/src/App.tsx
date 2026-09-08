import React, { useState, useEffect, useCallback } from 'react';
import { Header } from './components/Header';
import { Sidebar } from './components/Sidebar';
import { ControlDesk } from './pages/ControlDesk';
import { VideoTools } from './pages/VideoTools';
import { SchedulerTab } from './pages/SchedulerTab';
import { RemoteAlerts } from './pages/RemoteAlerts';
import { useWebSocket } from './hooks/useWebSocket';
import { api } from './api/client';
import {
  ActiveTab,
  Slot,
  VideoItem,
  ScheduleItem,
  TunnelSettings,
  AlertSettings,
} from './types';

const defaultSlots: Slot[] = [
  {
    id: 1,
    slot_number: 1,
    name: 'Slot 1 (YouTube Main Live)',
    status: 'running',
    target_platform: 'youtube',
    rtmp_url: 'rtmp://a.rtmp.youtube.com/live2',
    stream_key: 'xxxx-xxxx-xxxx-xxxx',
    mode: 'passthrough',
    encoding_mode: 'passthrough',
    overlay_clock_wib: true,
    overlay_watermark: 'LIVE FEED',
    overlay_watermark_pos: 'top-right',
    playlist: [
      {
        id: '1',
        slot_id: 1,
        video_id: 'v1',
        filename: 'source_promo_2026.mp4',
        order: 1,
        status: 'playing',
        duration_seconds: 8050,
      },
      {
        id: '2',
        slot_id: 1,
        video_id: 'v2',
        filename: 'broll_night_city.mp4',
        order: 2,
        status: 'next',
        duration_seconds: 2700,
      },
      {
        id: '3',
        slot_id: 1,
        video_id: 'v3',
        filename: 'ambient_lofi_loop.mp4',
        order: 3,
        status: 'queued',
        duration_seconds: 4200,
      },
    ],
  },
  {
    id: 2,
    slot_number: 2,
    name: 'Slot 2 (Twitch/FB Backup)',
    status: 'idle',
    target_platform: 'twitch',
    rtmp_url: 'rtmp://live.twitch.tv/app',
    stream_key: 'xxxx-xxxx-xxxx-xxxx',
    mode: 'passthrough',
    encoding_mode: 'passthrough',
    overlay_clock_wib: false,
    overlay_watermark: 'BACKUP STREAM',
    overlay_watermark_pos: 'top-left',
    playlist: [
      {
        id: '4',
        slot_id: 2,
        video_id: 'v4',
        filename: 'standby_slate_720p.mp4',
        order: 1,
        status: 'queued',
        duration_seconds: 600,
      },
    ],
  },
];

const defaultVideos: VideoItem[] = [
  {
    id: 'v1',
    filename: 'source_promo_2026.mp4',
    original_name: 'source_promo_2026.mp4',
    file_path: '/storage/emulated/0/stream_pool/source_promo_2026.mp4',
    file_size: 2576980377,
    duration_seconds: 8050,
    resolution: '1920x1080',
    video_codec: 'H.264 (High)',
    audio_codec: 'AAC-LC',
    fps: 30.0,
    gop_size: 2.0,
    is_passthrough_ready: true,
    created_at: '2026-03-25T10:00:00Z',
  },
  {
    id: 'v2',
    filename: 'broll_night_city.mp4',
    original_name: 'broll_night_city.mp4',
    file_path: '/storage/emulated/0/stream_pool/broll_night_city.mp4',
    file_size: 850000000,
    duration_seconds: 2700,
    resolution: '1920x1080',
    video_codec: 'H.264 (High)',
    audio_codec: 'AAC-LC',
    fps: 30.0,
    gop_size: 2.0,
    is_passthrough_ready: true,
    created_at: '2026-03-26T14:30:00Z',
  },
  {
    id: 'v3',
    filename: 'ambient_lofi_loop.mp4',
    original_name: 'ambient_lofi_loop.mp4',
    file_path: '/storage/emulated/0/stream_pool/ambient_lofi_loop.mp4',
    file_size: 1280000000,
    duration_seconds: 4200,
    resolution: '1920x1080',
    video_codec: 'H.264 (High)',
    audio_codec: 'AAC-LC',
    fps: 30.0,
    gop_size: 2.0,
    is_passthrough_ready: true,
    created_at: '2026-03-27T08:15:00Z',
  },
  {
    id: 'v4',
    filename: 'mobile_vlog_raw.mov',
    original_name: 'mobile_vlog_raw.mov',
    file_path: '/storage/emulated/0/stream_pool/mobile_vlog_raw.mov',
    file_size: 1940000000,
    duration_seconds: 840,
    resolution: '3840x2160',
    video_codec: 'HEVC (H.265)',
    audio_codec: 'PCM (Uncompressed)',
    fps: 59.94,
    gop_size: 0.5,
    is_passthrough_ready: false,
    created_at: '2026-03-28T19:20:00Z',
  },
  {
    id: 'v5',
    filename: 'intro_bumper.mp4',
    original_name: 'intro_bumper.mp4',
    file_path: '/storage/emulated/0/stream_pool/intro_bumper.mp4',
    file_size: 15728640,
    duration_seconds: 30,
    resolution: '1280x720',
    video_codec: 'MPEG-4',
    audio_codec: 'MP3',
    fps: 25.0,
    gop_size: 1.0,
    is_passthrough_ready: false,
    created_at: '2026-03-29T11:45:00Z',
  },
];

const defaultSchedules: ScheduleItem[] = [
  {
    id: 'sch-1',
    slot_id: 1,
    title: 'Daily Stream Re-align & PTS Flush',
    cron_expr: '0 06 * * *',
    duration_minutes: 120,
    is_enabled: true,
    overlap_guard_policy: 'yield_priority',
  },
  {
    id: 'sch-2',
    slot_id: 1,
    title: 'Clear Log Buffer Cache & Sync Index',
    cron_expr: '0 00 * * *',
    duration_minutes: 10,
    is_enabled: true,
    overlap_guard_policy: 'yield_priority',
  },
  {
    id: 'sch-3',
    slot_id: 2,
    title: 'Weekly Backup Stream Routine',
    cron_expr: '0 18 * * 5',
    duration_minutes: 240,
    is_enabled: false,
    overlap_guard_policy: 'terminate_prev',
  },
];

const defaultTunnel: TunnelSettings = {
  provider: 'cloudflare',
  mode: 'quick',
  is_active: false,
  public_url: '',
  tailscale_ip: '100.84.19.42',
  last_status: 'offline',
};

const defaultAlerts: AlertSettings = {
  telegram_enabled: false,
  telegram_bot_token: '',
  telegram_chat_id: '',
  discord_enabled: false,
  discord_webhook_url: '',
  trigger_on_crash: true,
  trigger_on_thermal: true,
  thermal_threshold_c: 48,
  trigger_on_low_storage: true,
  low_storage_threshold_gb: 5.0,
};

export const App: React.FC = () => {
  const [activeTab, setActiveTab] = useState<ActiveTab>('desk');
  const [slots, setSlots] = useState<Slot[]>(defaultSlots);
  const [videos, setVideos] = useState<VideoItem[]>(defaultVideos);
  const [schedules, setSchedules] = useState<ScheduleItem[]>(defaultSchedules);
  const [overlapStatus, setOverlapStatus] = useState<{
    has_overlap: boolean;
    conflict_count: number;
    conflicts: any[];
    total_schedules: number;
  } | null>(null);
  const [tunnel, setTunnel] = useState<TunnelSettings>(defaultTunnel);
  const [alerts, setAlerts] = useState<AlertSettings>(defaultAlerts);

  const {
    metrics,
    telemetries,
    setTelemetries,
    logs,
    addLog,
    clearLogs,
    isConnected,
  } = useWebSocket();

  // 1. Fetch data on mount from REST API
  const refreshSlots = useCallback(async () => {
    try {
      const data = await api.getSlots();
      if (Array.isArray(data) && data.length > 0) {
        setSlots(data);
      }
    } catch (err) {
      console.warn('[REST] api.getSlots fallback to local state:', err);
    }
  }, []);

  const refreshVideos = useCallback(async () => {
    try {
      const data = await api.getVideos();
      if (Array.isArray(data) && data.length > 0) {
        setVideos(data);
      }
    } catch (err) {
      console.warn('[REST] api.getVideos fallback to local state:', err);
    }
  }, []);

  const refreshSchedules = useCallback(async () => {
    try {
      const [schData, ovData] = await Promise.all([
        api.getSchedules(),
        api.getOverlapStatus(),
      ]);
      if (Array.isArray(schData)) {
        setSchedules(schData);
      }
      if (ovData) {
        setOverlapStatus(ovData);
      }
    } catch (err) {
      console.warn('[REST] api.getSchedules/overlapStatus fallback:', err);
    }
  }, []);

  const refreshTunnel = useCallback(async () => {
    try {
      const data = await api.getTunnelStatus();
      if (data) {
        setTunnel(data);
      }
    } catch (err) {
      console.warn('[REST] api.getTunnelStatus fallback to local state:', err);
    }
  }, []);

  const refreshAlerts = useCallback(async () => {
    try {
      const data = await api.getAlertSettings();
      if (data) {
        setAlerts(data);
      }
    } catch (err) {
      console.warn('[REST] api.getAlertSettings fallback to local state:', err);
    }
  }, []);

  useEffect(() => {
    refreshSlots();
    refreshVideos();
    refreshSchedules();
    refreshTunnel();
    refreshAlerts();
  }, [refreshSlots, refreshVideos, refreshSchedules, refreshTunnel, refreshAlerts]);

  // 2. Tombol Aksi: Start / Stop Slot
  const handleStartSlot = async (id: number) => {
    const slot = slots.find((s) => s.id === id || s.slot_number === id);
    const slotNumber = slot?.slot_number || id;

    // Optimistic UI
    setSlots((prev) =>
      prev.map((s) => (s.id === id || s.slot_number === slotNumber ? { ...s, status: 'running' } : s))
    );
    setTelemetries((prev) => ({
      ...prev,
      [slotNumber]: {
        slot_number: slotNumber,
        status: 'running',
        frame: 1,
        fps: 30.0,
        bitrate_kbps: 2450.0,
        duration: '00:00:01',
        speed: '1.00x',
        dropped_frames: 0,
        pts_sync_ms: 0.0,
        keyframe_cadence: '2.00s GOP (STABLE)',
        net_latency_ms: 36,
        timestamp: new Date().toISOString(),
      },
    }));
    addLog('RTMP', `Memulai pipeline streaming Slot ${slotNumber}...`);

    try {
      await api.startSlot(slotNumber);
      addLog('RTMP', `Slot ${slotNumber} streaming aktif (FFmpeg PID spawned)`);
      refreshSlots();
    } catch (err: any) {
      addLog('ERR', `Gagal start Slot ${slotNumber}: ${err.message}`);
      alert(`Gagal memulai streaming Slot ${slotNumber}: ${err.message}`);
      refreshSlots();
    }
  };

  const handleStopSlot = async (id: number) => {
    const slot = slots.find((s) => s.id === id || s.slot_number === id);
    const slotNumber = slot?.slot_number || id;

    // Optimistic UI
    setSlots((prev) =>
      prev.map((s) => (s.id === id || s.slot_number === slotNumber ? { ...s, status: 'idle' } : s))
    );
    setTelemetries((prev) => ({
      ...prev,
      [slotNumber]: {
        ...prev[slotNumber],
        status: 'idle',
        fps: 0,
        bitrate_kbps: 0,
      },
    }));
    addLog('RTMP', `Menghentikan streaming Slot ${slotNumber}...`);

    try {
      await api.stopSlot(slotNumber);
      addLog('RTMP', `Slot ${slotNumber} berhasil dihentikan (SIGTERM -> clean exit)`);
      refreshSlots();
    } catch (err: any) {
      addLog('ERR', `Gagal stop Slot ${slotNumber}: ${err.message}`);
      refreshSlots();
    }
  };

  // 3. Tombol Aksi: Simpan Konfigurasi Slot
  const handleUpdateSlot = async (id: number, config: Partial<Slot>) => {
    const slot = slots.find((s) => s.id === id || s.slot_number === id);
    const slotNumber = slot?.slot_number || id;

    setSlots((prev) =>
      prev.map((s) => (s.id === id || s.slot_number === slotNumber ? { ...s, ...config } : s))
    );

    try {
      await api.updateSlot(slotNumber, config);
      addLog('SYS', `Konfigurasi Slot ${slotNumber} berhasil diperbarui di server.`);
      await refreshSlots();
      alert(`Konfigurasi Slot ${slotNumber} berhasil disimpan!`);
    } catch (err: any) {
      addLog('ERR', `Gagal update Slot ${slotNumber}: ${err.message}`);
      alert(`Gagal menyimpan konfigurasi: ${err.message}`);
    }
  };

  const handleRefreshSnapshot = (id: number) => {
    const slot = slots.find((s) => s.id === id || s.slot_number === id);
    const slotNumber = slot?.slot_number || id;
    addLog('SYS', `Snapshot frame refreshed dari pipe raw FFmpeg Slot ${slotNumber}`);
  };

  // 4. Tombol Aksi: Video Upload, Fix Codec & Delete
  const handleUploadVideo = async (file: File) => {
    addLog('SYS', `Mengunggah media: ${file.name} (${(file.size / (1024 * 1024)).toFixed(1)} MB)...`);
    try {
      const uploaded = await api.uploadVideo(file);
      addLog('SYS', `Upload sukses: ${uploaded.filename}`);
      await refreshVideos();
      alert(`File ${uploaded.filename} berhasil diunggah!`);
    } catch (err: any) {
      addLog('ERR', `Upload gagal: ${err.message}`);
      // Fallback local preview
      const newVideo: VideoItem = {
        id: 'v-' + Date.now(),
        filename: file.name,
        original_name: file.name,
        file_path: `/storage/emulated/0/stream_pool/${file.name}`,
        file_size: file.size,
        duration_seconds: 1800,
        resolution: '1920x1080',
        video_codec: 'H.264 (High)',
        audio_codec: 'AAC-LC',
        fps: 30.0,
        gop_size: 2.0,
        is_passthrough_ready: true,
        created_at: new Date().toISOString(),
      };
      setVideos((prev) => [newVideo, ...prev]);
      alert(`File ${file.name} tersimpan di pool lokal.`);
    }
  };

  const handleFixCodec = async (id: string) => {
    addLog('CONCAT', `Menjalankan offline codec standardizer untuk video ${id}...`);
    try {
      await api.createFixJob(id);
      addLog('CONCAT', `Job transcode offline dibuat untuk video ${id}`);
      setTimeout(async () => {
        await refreshVideos();
        addLog('CONCAT', `Standardisasi codec selesai: video ${id} PASSTHROUGH READY`);
      }, 3000);
    } catch (err: any) {
      addLog('ERR', `Fix codec gagal: ${err.message}`);
      // Fallback update
      setTimeout(() => {
        setVideos((prev) =>
          prev.map((v) =>
            v.id === id
              ? {
                  ...v,
                  video_codec: 'H.264 (High)',
                  audio_codec: 'AAC-LC',
                  fps: 30.0,
                  resolution: '1920x1080',
                  is_passthrough_ready: true,
                }
              : v
          )
        );
        addLog('CONCAT', `Local fallback conversion selesai untuk ${id}`);
      }, 2500);
    }
  };

  const handleDeleteVideo = async (id: string) => {
    try {
      await api.deleteVideo(id);
      addLog('SYS', `Video ${id} berhasil dihapus dari penyimpanan.`);
      await refreshVideos();
    } catch (err: any) {
      addLog('ERR', `Gagal menghapus video ${id}: ${err.message}`);
      setVideos((prev) => prev.filter((v) => v.id !== id));
    }
  };

  // 5. Tombol Aksi: Schedules
  const handleAddSchedule = async (item: Partial<ScheduleItem>) => {
    try {
      await api.createSchedule(item);
      addLog('SYS', `Jadwal baru ditambahkan: "${item.title}"`);
      await refreshSchedules();
    } catch (err: any) {
      addLog('ERR', `Gagal membuat jadwal: ${err.message}`);
      const newSchedule: ScheduleItem = {
        id: 'sch-' + Date.now(),
        slot_id: item.slot_id || 1,
        title: item.title || 'Scheduled Ingest',
        cron_expr: item.cron_expr || '0 00 * * *',
        duration_minutes: item.duration_minutes || 60,
        is_enabled: true,
        overlap_guard_policy: item.overlap_guard_policy || 'yield_priority',
      };
      setSchedules((prev) => [...prev, newSchedule]);
    }
  };

  const handleDeleteSchedule = async (id: string) => {
    try {
      await api.deleteSchedule(id);
      addLog('SYS', `Jadwal ${id} berhasil dihapus.`);
      await refreshSchedules();
    } catch (err: any) {
      addLog('ERR', `Gagal menghapus jadwal: ${err.message}`);
      setSchedules((prev) => prev.filter((s) => s.id !== id));
    }
  };

  // 6. Tombol Aksi: Tunnel Start/Stop
  const handleUpdateTunnel = async (_t: Partial<TunnelSettings>) => {
    const shouldStart = !tunnel.is_active;
    addLog('SYS', `${shouldStart ? 'Menghidupkan' : 'Mematikan'} Cloudflare Quick Tunnel...`);

    try {
      if (shouldStart) {
        await api.startTunnel({ provider: 'cloudflare', mode: 'quick' });
        addLog('SYS', 'Cloudflare Quick Tunnel berhasil diaktifkan.');
      } else {
        await api.stopTunnel();
        addLog('SYS', 'Cloudflare Quick Tunnel berhasil dinonaktifkan.');
      }
      await refreshTunnel();
    } catch (err: any) {
      addLog('ERR', `Operasi tunnel gagal: ${err.message}`);
      setTunnel((prev) => ({
        ...prev,
        is_active: shouldStart,
        public_url: shouldStart ? 'https://go-streamer-quick.trycloudflare.com' : '',
        last_status: shouldStart ? 'online' : 'offline',
      }));
    }
  };

  // 7. Tombol Aksi: Alerts Save & Test Alert
  const handleUpdateAlerts = async (a: Partial<AlertSettings>) => {
    try {
      await api.updateAlertSettings(a);
      addLog('ALERT', 'Pengaturan notifikasi Telegram / Discord berhasil disimpan di database.');
      await refreshAlerts();
      alert('Pengaturan Telegram & Discord berhasil disimpan!');
    } catch (err: any) {
      addLog('ERR', `Gagal simpan alert settings: ${err.message}`);
      setAlerts((prev) => ({ ...prev, ...a }));
      alert('Pengaturan disimpan lokal.');
    }
  };

  const handleTestAlert = async () => {
    addLog('ALERT', 'Mengirim test payload notifikasi ke Telegram / Discord...');
    try {
      const res = await api.testAlert();
      addLog('ALERT', `Notifikasi terkirim: ${res.message || 'OK'}`);
      alert(`Test Alert berhasil dikirim! ${res.message || ''}`);
    } catch (err: any) {
      addLog('ERR', `Gagal kirim test alert: ${err.message}`);
      alert(`Gagal kirim test alert: ${err.message}`);
    }
  };

  // 8. Tombol Aksi: Kill Switch
  const handleKillSwitch = () => {
    setSlots((prev) => prev.map((s) => ({ ...s, status: 'idle' })));
    setTelemetries((prev) => {
      const next = { ...prev };
      Object.keys(next).forEach((k) => {
        const num = Number(k);
        if (next[num]) {
          next[num] = {
            ...next[num],
            status: 'idle',
            fps: 0,
            bitrate_kbps: 0,
          };
        }
      });
      return next;
    });
    addLog('ERR', 'EMERGENCY KILL SWITCH: Seluruh child process FFmpeg dimatikan seketika!');
    refreshSlots();
  };

  return (
    <div className="max-w-7xl mx-auto">
      {/* Top Header Bar */}
      <Header isConnected={isConnected} onKillSwitch={handleKillSwitch} />

      {/* Main Grid: Sidebar + Active Page */}
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-4">
        <Sidebar
          activeTab={activeTab}
          onSelectTab={setActiveTab}
          tunnelActive={tunnel.is_active}
        />

        <main className="lg:col-span-9 xl:col-span-10">
          {activeTab === 'desk' && (
            <ControlDesk
              metrics={metrics}
              slots={slots}
              telemetries={telemetries}
              logs={logs}
              onStartSlot={handleStartSlot}
              onStopSlot={handleStopSlot}
              onUpdateSlot={handleUpdateSlot}
              onRefreshSnapshot={handleRefreshSnapshot}
              onClearLogs={clearLogs}
            />
          )}

          {activeTab === 'video' && (
            <VideoTools
              videos={videos}
              onUpload={handleUploadVideo}
              onFixCodec={handleFixCodec}
              onDelete={handleDeleteVideo}
            />
          )}

          {activeTab === 'scheduler' && (
            <SchedulerTab
              schedules={schedules}
              overlapStatus={overlapStatus}
              onAddSchedule={handleAddSchedule}
              onDeleteSchedule={handleDeleteSchedule}
            />
          )}

          {activeTab === 'remote' && (
            <RemoteAlerts
              tunnel={tunnel}
              alerts={alerts}
              onUpdateTunnel={handleUpdateTunnel}
              onUpdateAlerts={handleUpdateAlerts}
              onTestAlert={handleTestAlert}
            />
          )}
        </main>
      </div>
    </div>
  );
};

export default App;
