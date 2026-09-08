import {
  SystemMetrics,
  Slot,
  VideoItem,
  ScheduleItem,
  TunnelSettings,
  AlertSettings,
} from '../types';

const BASE_URL = '/api/v1';

async function request<T>(endpoint: string, options?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE_URL}${endpoint}`, {
    headers: {
      'Content-Type': 'application/json',
      ...options?.headers,
    },
    ...options,
  });

  if (!res.ok) {
    let errMsg = `Request failed with status ${res.status}`;
    try {
      const errJson = await res.json();
      if (errJson.error) errMsg = errJson.error;
      else if (errJson.message) errMsg = errJson.message;
    } catch {
      const errText = await res.text();
      if (errText) errMsg = errText;
    }
    throw new Error(errMsg);
  }

  return res.json();
}

export const api = {
  // System
  getHealth: () => request<{ status: string; timestamp: string }>('/health'),
  getSystemMetrics: () => request<SystemMetrics>('/system/metrics'),
  getSystemInfo: () => request<any>('/system/info'),
  killSwitch: () =>
    request<{ message: string }>('/system/killswitch', { method: 'POST' }),

  // Slots
  getSlots: () => request<Slot[]>('/slots'),
  getSlot: (idOrSlotNumber: number | string) =>
    request<Slot>(`/slots/${idOrSlotNumber}`),
  updateSlot: (idOrSlotNumber: number | string, data: Partial<Slot>) =>
    request<Slot>(`/slots/${idOrSlotNumber}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    }),
  startSlot: (idOrSlotNumber: number | string) =>
    request<{ message: string; slot_id: number; slot_number: number; status: string }>(
      `/slots/${idOrSlotNumber}/start`,
      { method: 'POST' }
    ),
  stopSlot: (idOrSlotNumber: number | string) =>
    request<{ message: string; slot_id: number; slot_number: number; status: string }>(
      `/slots/${idOrSlotNumber}/stop`,
      { method: 'POST' }
    ),
  getSnapshotUrl: (idOrSlotNumber: number | string) =>
    `${BASE_URL}/slots/${idOrSlotNumber}/snapshot?t=${Date.now()}`,

  // Videos
  getVideos: () => request<VideoItem[]>('/videos'),
  uploadVideo: async (file: File): Promise<VideoItem> => {
    const formData = new FormData();
    formData.append('video', file);
    formData.append('user_id', 'usr_admin_default');
    const res = await fetch(`${BASE_URL}/videos/upload`, {
      method: 'POST',
      body: formData,
    });
    if (!res.ok) {
      const text = await res.text();
      throw new Error(text || 'Video upload failed');
    }
    return res.json();
  },
  deleteVideo: (id: string) =>
    request<{ message: string; id: string }>(`/videos/${id}`, { method: 'DELETE' }),
  renameVideo: (id: string, newName: string) =>
    request<VideoItem>(`/videos/${id}/rename`, {
      method: 'PUT',
      body: JSON.stringify({ filename: newName }),
    }),
  getStorageStats: () =>
    request<{
      total_storage_bytes: number;
      total_videos: number;
      free_space_bytes: number;
      total_space_bytes: number;
    }>('/videos/storage'),

  // Codec / Fixer
  createFixJob: (videoId: string) =>
    request<{ id: string; source_video_id: string; status: string }>(
      '/codec/fix',
      {
        method: 'POST',
        body: JSON.stringify({ source_video_id: videoId }),
      }
    ),
  listCodecJobs: () => request<any[]>('/codec/jobs'),

  // Scheduler
  getSchedules: () => request<ScheduleItem[]>('/schedules'),
  createSchedule: (data: Partial<ScheduleItem>) =>
    request<ScheduleItem>('/schedules', {
      method: 'POST',
      body: JSON.stringify(data),
    }),
  deleteSchedule: (id: string) =>
    request<{ message: string; id: string }>(`/schedules/${id}`, { method: 'DELETE' }),
  getOverlapStatus: () =>
    request<{
      has_overlap: boolean;
      conflict_count: number;
      conflicts: Array<{ schedule_a: string; schedule_b: string }>;
      total_schedules: number;
    }>('/schedules/overlap-status'),

  // Remote Tunnel
  getTunnelStatus: async (): Promise<TunnelSettings> => {
    const res = await request<{ settings: any; tailscale_ip: string }>('/tunnel/status');
    const s = res.settings || {};
    return {
      provider: s.provider || 'cloudflare',
      mode: s.mode || 'quick',
      is_active: Boolean(s.is_active),
      public_url: s.public_url || '',
      tailscale_ip: res.tailscale_ip || '',
      last_status: s.last_status || 'offline',
    };
  },
  startTunnel: (params?: { provider?: string; mode?: string; token?: string }) =>
    request<any>('/tunnel/start', {
      method: 'POST',
      body: JSON.stringify(params || { provider: 'cloudflare', mode: 'quick' }),
    }),
  stopTunnel: () => request<any>('/tunnel/stop', { method: 'POST' }),

  // Alerts
  getAlertSettings: () => request<AlertSettings>('/alerts/settings'),
  updateAlertSettings: (settings: Partial<AlertSettings>) =>
    request<AlertSettings>('/alerts/settings', {
      method: 'PUT',
      body: JSON.stringify(settings),
    }),
  testAlert: () =>
    request<{ message: string }>('/alerts/test', {
      method: 'POST',
    }),
};
