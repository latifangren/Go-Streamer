export interface PlatformPreset {
  id: string;
  name: string;
  defaultRtmpUrl: string;
  streamKeyPlaceholder: string;
  badgeClass: string;
  description?: string;
}

export const STREAM_PLATFORMS: PlatformPreset[] = [
  {
    id: 'youtube',
    name: 'YouTube Live',
    defaultRtmpUrl: 'rtmp://a.rtmp.youtube.com/live2',
    streamKeyPlaceholder: 'xxxx-xxxx-xxxx-xxxx-xxxx',
    badgeClass: 'bg-red-500 text-white',
  },
  {
    id: 'facebook',
    name: 'Facebook Live',
    defaultRtmpUrl: 'rtmps://live-api-s.facebook.com:443/rtmp/',
    streamKeyPlaceholder: 'FB-xxxxxxxxxxxxxxxx',
    badgeClass: 'bg-blue-600 text-white',
  },
  {
    id: 'twitch',
    name: 'Twitch',
    defaultRtmpUrl: 'rtmp://live.twitch.tv/app/',
    streamKeyPlaceholder: 'live_xxxxxxxx_xxxxxxxxxxxxxxxx',
    badgeClass: 'bg-purple-600 text-white',
  },
  {
    id: 'tiktok',
    name: 'TikTok Live',
    defaultRtmpUrl: 'rtmp://live-push.tiktok.com/live/',
    streamKeyPlaceholder: 'stream-xxxxxxxx',
    badgeClass: 'bg-neutral-900 text-cyan-400',
  },
  {
    id: 'kick',
    name: 'Kick.com',
    defaultRtmpUrl: 'rtmps://fa723fc1b171.global-contribute.live-video.net:443/app/',
    streamKeyPlaceholder: 'sk_live_xxxxxxxx',
    badgeClass: 'bg-emerald-500 text-black',
  },
  {
    id: 'custom',
    name: 'Custom RTMP',
    defaultRtmpUrl: 'rtmp://',
    streamKeyPlaceholder: 'Stream key kustom atau biarkan kosong',
    badgeClass: 'bg-yellow-400 text-black',
  },
];

export function getPlatformPreset(platformId?: string): PlatformPreset {
  const found = STREAM_PLATFORMS.find((p) => p.id.toLowerCase() === (platformId || '').toLowerCase());
  return (
    found || {
      id: platformId || 'custom',
      name: platformId ? platformId.toUpperCase() : 'Custom RTMP',
      defaultRtmpUrl: 'rtmp://',
      streamKeyPlaceholder: 'Stream key',
      badgeClass: 'bg-neutral-200 text-black',
    }
  );
}
