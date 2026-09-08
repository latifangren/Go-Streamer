import React, { useState, useEffect } from 'react';
import { X, Sliders, Eye, EyeOff, Check, AlertCircle } from 'lucide-react';
import { Slot } from '../types';

interface SlotConfigModalProps {
  slot: Slot | null;
  isOpen: boolean;
  onClose: () => void;
  onSave: (id: number, config: Partial<Slot>) => void;
}

export const SlotConfigModal: React.FC<SlotConfigModalProps> = ({
  slot,
  isOpen,
  onClose,
  onSave,
}) => {
  const [platform, setPlatform] = useState<Slot['target_platform']>('youtube');
  const [rtmpUrl, setRtmpUrl] = useState('');
  const [streamKey, setStreamKey] = useState('');
  const [showStreamKey, setShowStreamKey] = useState(false);
  const [encodingMode, setEncodingMode] = useState<Slot['encoding_mode']>('passthrough');
  const [overlayClockWIB, setOverlayClockWIB] = useState(true);
  const [watermarkText, setWatermarkText] = useState('LIVE FEED');

  useEffect(() => {
    if (slot) {
      setPlatform(slot.target_platform || 'youtube');
      setRtmpUrl(slot.rtmp_url || 'rtmp://a.rtmp.youtube.com/live2');
      setStreamKey(slot.stream_key || '');
      setEncodingMode(slot.encoding_mode || 'passthrough');
      setOverlayClockWIB(slot.overlay_clock_wib ?? true);
      setWatermarkText(slot.overlay_watermark || 'LIVE FEED');
    }
  }, [slot]);

  if (!isOpen || !slot) return null;

  const handlePlatformChange = (p: Slot['target_platform']) => {
    setPlatform(p);
    if (p === 'youtube') setRtmpUrl('rtmp://a.rtmp.youtube.com/live2');
    else if (p === 'facebook') setRtmpUrl('rtmps://live-api-s.facebook.com:443/rtmp/');
    else if (p === 'twitch') setRtmpUrl('rtmp://live.twitch.tv/app/');
    else if (p === 'custom' && !rtmpUrl) setRtmpUrl('rtmp://');
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    onSave(slot.id, {
      target_platform: platform,
      rtmp_url: rtmpUrl,
      stream_key: streamKey,
      encoding_mode: encodingMode,
      overlay_clock_wib: overlayClockWIB,
      overlay_watermark: watermarkText,
    });
    onClose();
  };

  return (
    <div className="fixed inset-0 bg-black/60 z-50 flex items-center justify-center p-3.5 backdrop-blur-xs">
      <div className="bg-white border-[3px] border-black rounded-xl p-5 max-w-lg w-full shadow-[8px_8px_0px_#000] space-y-4 max-h-[92vh] overflow-y-auto">
        {/* Modal Header */}
        <div className="flex items-center justify-between border-b-2 border-black pb-3">
          <div className="flex items-center gap-2">
            <span className="bg-neoYellow border-2 border-black p-1 rounded-md shadow-[1px_1px_0px_#000]">
              <Sliders className="w-4 h-4 stroke-[2.5]" />
            </span>
            <h3 className="font-mono font-black text-base uppercase text-neutral-900">
              CONFIG ENGINE: Slot {slot.slot_number} ({slot.name})
            </h3>
          </div>
          <button
            type="button"
            onClick={onClose}
            className="neo-btn bg-neutral-100 hover:bg-neutral-200 border border-black p-1 rounded cursor-pointer"
          >
            <X className="w-4 h-4" />
          </button>
        </div>

        <form onSubmit={handleSubmit} className="space-y-4 text-xs font-mono">
          {/* Target Platform Quick Switcher */}
          <div>
            <label className="block font-black text-neutral-800 uppercase mb-1.5">
              Target Streaming Platform:
            </label>
            <div className="grid grid-cols-4 gap-1.5">
              {(['youtube', 'facebook', 'twitch', 'custom'] as const).map((p) => {
                const isSel = platform === p;
                return (
                  <button
                    key={p}
                    type="button"
                    onClick={() => handlePlatformChange(p)}
                    className={`neo-btn py-1.5 px-2 rounded-lg border-2 border-black font-black uppercase text-[11px] transition-all cursor-pointer ${
                      isSel
                        ? 'bg-neoMint shadow-[2px_2px_0px_#000]'
                        : 'bg-neutral-100 hover:bg-white text-neutral-700'
                    }`}
                  >
                    {p}
                  </button>
                );
              })}
            </div>
          </div>

          {/* Dynamic Stream Overlay Toggle Box */}
          <div className="p-3 bg-neoYellow border-2 border-black rounded-lg space-y-2.5 shadow-[2px_2px_0px_#000]">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <span className="bg-black text-neoYellow px-1 py-0.2 rounded font-black text-[10px]">
                  OVERLAY
                </span>
                <label
                  htmlFor="overlay-clock"
                  className="font-black text-neutral-900 uppercase text-[11px] cursor-pointer"
                >
                  Dynamic WIB Clock & Watermark
                </label>
              </div>
              <input
                id="overlay-clock"
                type="checkbox"
                checked={overlayClockWIB}
                onChange={(e) => setOverlayClockWIB(e.target.checked)}
                className="w-4 h-4 rounded border-2 border-black accent-black cursor-pointer"
              />
            </div>

            <div className="grid grid-cols-2 gap-2 text-[10px]">
              <div>
                <span className="text-neutral-700 block font-bold mb-0.5">Overlay Content:</span>
                <input
                  type="text"
                  value={watermarkText}
                  onChange={(e) => setWatermarkText(e.target.value)}
                  placeholder="Watermark text"
                  className="w-full bg-white border border-black rounded p-1 text-[11px] font-bold"
                />
              </div>
              <div>
                <span className="text-neutral-700 block font-bold mb-0.5">Filter Overhead:</span>
                <span className="font-bold text-neutral-900 block mt-1">
                  +1.8% CPU (Drawtext Shader)
                </span>
              </div>
            </div>
          </div>

          {/* RTMP Server Endpoint */}
          <div>
            <label className="block font-black text-neutral-800 uppercase mb-1">
              RTMP Server Endpoint:
            </label>
            <input
              type="text"
              value={rtmpUrl}
              onChange={(e) => setRtmpUrl(e.target.value)}
              placeholder="rtmp://a.rtmp.youtube.com/live2"
              required
              className="w-full border-2 border-black rounded-lg p-2 font-mono text-xs focus:ring-0 focus:outline-none shadow-[2px_2px_0px_#000] bg-white"
            />
          </div>

          {/* Stream Key */}
          <div>
            <div className="flex items-center justify-between mb-1">
              <label className="font-black text-neutral-800 uppercase">Stream Key:</label>
              <button
                type="button"
                onClick={() => setShowStreamKey(!showStreamKey)}
                className="text-[10px] text-neutral-600 flex items-center gap-1 hover:underline cursor-pointer"
              >
                {showStreamKey ? (
                  <>
                    <EyeOff className="w-3 h-3" /> Hide
                  </>
                ) : (
                  <>
                    <Eye className="w-3 h-3" /> Show
                  </>
                )}
              </button>
            </div>
            <input
              type={showStreamKey ? 'text' : 'password'}
              value={streamKey}
              onChange={(e) => setStreamKey(e.target.value)}
              placeholder="xxxx-xxxx-xxxx-xxxx"
              className="w-full border-2 border-black rounded-lg p-2 font-mono text-xs focus:ring-0 focus:outline-none shadow-[2px_2px_0px_#000] bg-white"
            />
          </div>

          {/* Encoding Mode */}
          <div>
            <label className="block font-black text-neutral-800 uppercase mb-1">
              Encoding Strategy:
            </label>
            <select
              value={encodingMode}
              onChange={(e) => setEncodingMode(e.target.value as Slot['encoding_mode'])}
              className="w-full border-2 border-black rounded-lg p-2 font-mono text-xs focus:ring-0 focus:outline-none shadow-[2px_2px_0px_#000] bg-white cursor-pointer font-bold"
            >
              <option value="passthrough">Passthrough (-c:v copy -c:a copy) [Recommended]</option>
              <option value="transcode">
                Transcode with Overlay (-c:v libx264 -preset ultrafast)
              </option>
              <option value="transcode_720p">Transcode to 720p 30fps (Low Bitrate)</option>
            </select>
          </div>

          {/* Recommendation Info notice */}
          <div className="p-2 bg-neoBlue/60 border border-black rounded-lg flex items-start gap-2 text-[10px]">
            <AlertCircle className="w-4 h-4 text-neutral-800 shrink-0 mt-0.5" />
            <p className="text-neutral-800">
              Gunakan mode <strong>Passthrough</strong> untuk efisiensi CPU 90%+ lebih hemat pada
              perangkat Android/ARM. Hanya gunakan transcode jika memerlukan watermark dinamis.
            </p>
          </div>

          {/* Modal Actions */}
          <div className="flex justify-end gap-2 border-t-2 border-black pt-3">
            <button
              type="button"
              onClick={onClose}
              className="neo-btn bg-neutral-100 hover:bg-neutral-200 border-2 border-black font-mono font-bold text-xs px-3 py-2 rounded-lg shadow-[2px_2px_0px_#000] cursor-pointer"
            >
              Batal
            </button>
            <button
              type="submit"
              className="neo-btn bg-neoMint hover:bg-[#a6f0b3] border-2 border-black font-mono font-black text-xs px-4 py-2 rounded-lg shadow-[3px_3px_0px_#000] flex items-center gap-1.5 cursor-pointer"
            >
              <Check className="w-3.5 h-3.5 stroke-[3]" />
              <span>Simpan Konfigurasi</span>
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};
