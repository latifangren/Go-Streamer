import React, { useState } from 'react';
import {
  Globe,
  Send,
  Bell,
  Copy,
  Check,
  ShieldAlert,
} from 'lucide-react';
import { TunnelSettings, AlertSettings } from '../types';

interface RemoteAlertsProps {
  tunnel: TunnelSettings;
  alerts: AlertSettings;
  onUpdateTunnel: (tunnel: Partial<TunnelSettings>) => void;
  onUpdateAlerts: (alerts: Partial<AlertSettings>) => void;
  onTestAlert: () => void;
}

export const RemoteAlerts: React.FC<RemoteAlertsProps> = ({
  tunnel,
  alerts,
  onUpdateTunnel,
  onUpdateAlerts,
  onTestAlert,
}) => {
  const [copied, setCopied] = useState(false);
  const [isTestingTelegram, setIsTestingTelegram] = useState(false);
  const [isTestingDiscord, setIsTestingDiscord] = useState(false);

  const [tgToken, setTgToken] = useState(alerts.telegram_bot_token || '');
  const [tgChatId, setTgChatId] = useState(alerts.telegram_chat_id || '');
  const [discordUrl, setDiscordUrl] = useState(alerts.discord_webhook_url || '');

  const handleCopyUrl = () => {
    navigator.clipboard.writeText(tunnel.public_url);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const handleTest = (type: 'telegram' | 'discord') => {
    if (type === 'telegram') {
      setIsTestingTelegram(true);
      onTestAlert();
      setTimeout(() => setIsTestingTelegram(false), 1200);
    } else {
      setIsTestingDiscord(true);
      onTestAlert();
      setTimeout(() => setIsTestingDiscord(false), 1200);
    }
  };

  const handleSaveAlerts = (e: React.FormEvent) => {
    e.preventDefault();
    onUpdateAlerts({
      telegram_bot_token: tgToken,
      telegram_chat_id: tgChatId,
      discord_webhook_url: discordUrl,
    });
    alert('Pengaturan Telegram & Discord berhasil disimpan!');
  };

  return (
    <div className="space-y-4">
      {/* Container Box */}
      <div className="bg-white border-[2.5px] border-black rounded-xl p-4 md:p-5 shadow-[5px_5px_0px_#000] space-y-4">
        {/* Header */}
        <div className="border-b-2 border-black pb-3">
          <div className="flex items-center gap-2 mb-0.5">
            <span className="bg-neoBlue border border-black p-1 rounded shadow-[1px_1px_0px_#000]">
              <Globe className="w-4 h-4 text-black stroke-[2.5]" />
            </span>
            <h2 className="text-lg font-black uppercase font-mono text-neutral-900">
              CGNAT Remote Access & Proactive Alerts
            </h2>
          </div>
          <p className="text-xs md:text-sm text-neutral-600 font-medium">
            Akses dashboard Termux/Magisk Anda dari jaringan publik secara aman tanpa port forwarding,
            serta terima notifikasi instan bila stream macet atau suhu SoC panas.
          </p>
        </div>

        {/* Cloudflare & Tailscale Grid */}
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          {/* Cloudflare Quick Tunnel */}
          <div className="p-3.5 bg-[#f8f9fb] border-2 border-black rounded-xl space-y-2.5 shadow-[2px_2px_0px_#000]">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <span className="bg-neoYellow border border-black font-black text-[10px] px-1.5 py-0.5 rounded">
                  CLOUDFLARE
                </span>
                <span className="font-extrabold text-sm text-neutral-900">Quick Tunnel Engine</span>
              </div>
              <button
                type="button"
                onClick={() => onUpdateTunnel({ is_active: !tunnel.is_active })}
                className={`neo-btn px-2.5 py-1 rounded-md border border-black font-mono text-xs font-black shadow-[1px_1px_0px_#000] cursor-pointer ${
                  tunnel.is_active ? 'bg-neoMint' : 'bg-neutral-200 text-neutral-600'
                }`}
              >
                {tunnel.is_active ? 'ENABLED' : 'DISABLED'}
              </button>
            </div>

            <p className="text-[11px] text-neutral-600 font-mono">
              Tunnel otomatis via Cloudflare Edge Worker (tidak perlu IP Publik / Port Forward).
            </p>

            <div className="p-2 bg-white border border-black rounded-lg flex items-center justify-between gap-2 font-mono text-xs">
              <span className="truncate font-bold text-neutral-800 select-all">
                {tunnel.public_url || 'https://go-streamer-demo.trycloudflare.com'}
              </span>
              <button
                type="button"
                onClick={handleCopyUrl}
                className="neo-btn bg-neutral-100 hover:bg-neutral-200 border border-black p-1 rounded text-black cursor-pointer"
                title="Salin Public URL"
              >
                {copied ? (
                  <Check className="w-3.5 h-3.5 text-emerald-600" />
                ) : (
                  <Copy className="w-3.5 h-3.5" />
                )}
              </button>
            </div>
          </div>

          {/* Tailscale Mesh VPN */}
          <div className="p-3.5 bg-[#f8f9fb] border-2 border-black rounded-xl space-y-2.5 shadow-[2px_2px_0px_#000]">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <span className="bg-neoBlue border border-black font-black text-[10px] px-1.5 py-0.5 rounded">
                  VPN MESH
                </span>
                <span className="font-extrabold text-sm text-neutral-900">Tailscale Daemon</span>
              </div>
              <span className="bg-neoMint border border-black px-2 py-0.5 rounded font-mono font-bold text-[10px] shadow-[1px_1px_0px_#000]">
                CONNECTED
              </span>
            </div>

            <p className="text-[11px] text-neutral-600 font-mono">
              IP terenkripsi WireGuard peer-to-peer antar perangkat terpercaya.
            </p>

            <div className="p-2 bg-white border border-black rounded-lg flex items-center justify-between font-mono text-xs">
              <span className="text-neutral-600">Tailscale IP:</span>
              <strong className="text-neutral-900 font-black">
                {tunnel.tailscale_ip || '100.84.19.42'}
              </strong>
            </div>
          </div>
        </div>

        {/* QR Code Quick Connect Simulation */}
        <div className="p-3 bg-neoCanvas border-2 border-black rounded-xl flex items-center gap-3 shadow-[2px_2px_0px_#000]">
          {/* Neobrutalist QR Box */}
          <div className="w-14 h-14 bg-white border-2 border-black rounded-lg p-1.5 shrink-0 flex flex-col justify-between shadow-[2px_2px_0px_#000]">
            <div className="flex justify-between">
              <div className="w-3.5 h-3.5 bg-black border border-white" />
              <div className="w-3.5 h-3.5 bg-black border border-white" />
            </div>
            <div className="text-center font-mono font-black text-[7px] leading-tight">SCAN</div>
            <div className="flex justify-between">
              <div className="w-3.5 h-3.5 bg-black border border-white" />
              <div className="w-3.5 h-3.5 bg-black border border-white" />
            </div>
          </div>

          <div className="text-xs font-mono">
            <span className="font-black text-black uppercase">Remote Quick Access:</span>
            <p className="text-neutral-700 text-[11px]">
              Buka link publik tunnel di smartphone atau laptop untuk memonitor transmisi RTMP di
              luar rumah.
            </p>
          </div>
        </div>

        {/* Proactive Webhook Alerts Configuration Form */}
        <div className="border-t-2 border-black pt-3">
          <div className="flex items-center justify-between mb-3">
            <h3 className="font-mono font-black text-sm uppercase flex items-center gap-1.5 text-neutral-900">
              <Bell className="w-4 h-4 text-black stroke-[2.5]" />
              Proactive Alert Channels (Telegram & Discord)
            </h3>
            <span className="bg-neoPink border border-black px-2 py-0.2 rounded font-mono font-bold text-[10px]">
              CRASH & THERMAL GUARD
            </span>
          </div>

          <form onSubmit={handleSaveAlerts} className="space-y-3 font-mono text-xs">
            {/* Telegram Bot */}
            <div className="p-3 bg-white border-2 border-black rounded-lg space-y-2 shadow-[2px_2px_0px_#000]">
              <div className="flex items-center justify-between">
                <span className="font-black text-neutral-900 text-xs flex items-center gap-1.5">
                  <Send className="w-3.5 h-3.5 text-sky-600" />
                  TELEGRAM BOT NOTIFIER
                </span>
                <button
                  type="button"
                  onClick={() => handleTest('telegram')}
                  disabled={isTestingTelegram}
                  className="neo-btn bg-neoBlue hover:bg-[#a6dcf9] border border-black font-bold px-2 py-0.5 rounded text-[10px] shadow-[1px_1px_0px_#000] cursor-pointer"
                >
                  {isTestingTelegram ? 'Sending...' : 'Test Alert'}
                </button>
              </div>
              <div className="grid grid-cols-1 md:grid-cols-2 gap-2">
                <div>
                  <label className="block text-neutral-600 text-[10px] uppercase font-bold mb-0.5">
                    Bot Token:
                  </label>
                  <input
                    type="password"
                    placeholder="123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11"
                    value={tgToken}
                    onChange={(e) => setTgToken(e.target.value)}
                    className="w-full border border-black rounded p-1.5 text-xs bg-neutral-50"
                  />
                </div>
                <div>
                  <label className="block text-neutral-600 text-[10px] uppercase font-bold mb-0.5">
                    Target Chat ID:
                  </label>
                  <input
                    type="text"
                    placeholder="-100123456789 or @channel"
                    value={tgChatId}
                    onChange={(e) => setTgChatId(e.target.value)}
                    className="w-full border border-black rounded p-1.5 text-xs bg-neutral-50"
                  />
                </div>
              </div>
            </div>

            {/* Discord Webhook */}
            <div className="p-3 bg-white border-2 border-black rounded-lg space-y-2 shadow-[2px_2px_0px_#000]">
              <div className="flex items-center justify-between">
                <span className="font-black text-neutral-900 text-xs flex items-center gap-1.5">
                  <Bell className="w-3.5 h-3.5 text-indigo-600" />
                  DISCORD EMBED WEBHOOK
                </span>
                <button
                  type="button"
                  onClick={() => handleTest('discord')}
                  disabled={isTestingDiscord}
                  className="neo-btn bg-neoLavender hover:bg-[#d0c2f5] border border-black font-bold px-2 py-0.5 rounded text-[10px] shadow-[1px_1px_0px_#000] cursor-pointer"
                >
                  {isTestingDiscord ? 'Sending...' : 'Test Alert'}
                </button>
              </div>
              <div>
                <label className="block text-neutral-600 text-[10px] uppercase font-bold mb-0.5">
                  Discord Webhook URL:
                </label>
                <input
                  type="text"
                  placeholder="https://discord.com/api/webhooks/..."
                  value={discordUrl}
                  onChange={(e) => setDiscordUrl(e.target.value)}
                  className="w-full border border-black rounded p-1.5 text-xs bg-neutral-50"
                />
              </div>
            </div>

            {/* Trigger Thresholds */}
            <div className="p-3 bg-neoYellow border-2 border-black rounded-lg flex flex-wrap items-center justify-between gap-2 text-[11px] shadow-[2px_2px_0px_#000]">
              <div className="flex items-center gap-2">
                <ShieldAlert className="w-4 h-4 text-black" />
                <span className="font-bold text-black">
                  Auto-alert saat: FFmpeg Crash (Non-zero exit), Suhu SoC &gt;48°C, atau Buffer Disk &lt;2GB.
                </span>
              </div>
              <button
                type="submit"
                className="neo-btn bg-black text-white border-2 border-black px-3 py-1 rounded-md font-black text-xs shadow-[2px_2px_0px_#000] cursor-pointer"
              >
                Simpan Konfigurasi Alert
              </button>
            </div>
          </form>
        </div>
      </div>
    </div>
  );
};
