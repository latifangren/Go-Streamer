import React, { useState } from 'react';
import {
  Play,
  Square,
  SlidersHorizontal,
  RefreshCw,
  Radio,
  Zap,
} from 'lucide-react';
import { Slot, StreamTelemetry } from '../types';

interface SlotCardProps {
  slot: Slot;
  telemetry?: StreamTelemetry;
  onStart: (id: number) => void;
  onStop: (id: number) => void;
  onOpenConfig: (slot: Slot) => void;
  onRefreshSnapshot?: (id: number) => void;
}

export const SlotCard: React.FC<SlotCardProps> = ({
  slot,
  telemetry,
  onStart,
  onStop,
  onOpenConfig,
  onRefreshSnapshot,
}) => {
  const [isRefreshing, setIsRefreshing] = useState(false);
  const [snapshotTime, setSnapshotTime] = useState(
    new Date().toLocaleTimeString('id-ID', { hour12: false }) + ' WIB'
  );

  const isRunning = slot.status === 'running' || telemetry?.status === 'running';
  const playlist = slot.playlist || [];

  const handleRefresh = () => {
    setIsRefreshing(true);
    setSnapshotTime(new Date().toLocaleTimeString('id-ID', { hour12: false }) + ' WIB');
    if (onRefreshSnapshot) {
      onRefreshSnapshot(slot.id);
    }
    setTimeout(() => {
      setIsRefreshing(false);
    }, 400);
  };

  const currentFps = telemetry?.fps ?? (isRunning ? 30.0 : 0);
  const currentBitrate = telemetry?.bitrate_kbps ?? (isRunning ? 2420 : 0);
  const currentDuration = telemetry?.duration || (isRunning ? '04:12:35' : '00:00:00');
  const currentSpeed = telemetry?.speed || (isRunning ? '1.00x' : '0.00x');
  const droppedFrames = telemetry?.dropped_frames ?? 0;
  const ptsSync = telemetry?.pts_sync_ms ?? 0.0;
  const keyframeGOP = telemetry?.keyframe_cadence || (isRunning ? '2.00s GOP (STABLE)' : 'Standby');
  const netLatency = telemetry?.net_latency_ms ?? (isRunning ? 38 : 0);

  return (
    <article className="bg-white border-[2.5px] border-black rounded-xl p-4 shadow-[5px_5px_0px_#000] flex flex-col justify-between space-y-3.5">
      {/* Slot Header */}
      <div className="flex flex-wrap items-center justify-between gap-2 border-b-2 border-black pb-2.5">
        <div className="flex items-center gap-2">
          <span
            className={`border-2 border-black font-mono font-black text-xs px-2.5 py-1 rounded-md shadow-[2px_2px_0px_#000] ${
              slot.slot_number === 1 ? 'bg-neoMint' : 'bg-neoBlue'
            }`}
          >
            SLOT {slot.slot_number} • {slot.slot_number === 1 ? 'PRIMARY ENGINE' : 'BACKUP ENGINE'}
          </span>
          <div>
            <h2 className="font-extrabold text-sm md:text-base tracking-tight text-neutral-900 leading-tight">
              {slot.name}
            </h2>
            <p className="font-mono text-[10px] text-neutral-500 truncate max-w-[200px] md:max-w-xs">
              {slot.rtmp_url}
            </p>
          </div>
        </div>

        <div>
          {isRunning ? (
            <div className="bg-neoMint border-2 border-black text-xs font-mono font-black px-2.5 py-1 rounded-md shadow-[2px_2px_0px_#000] flex items-center gap-1.5">
              <span className="w-2.5 h-2.5 rounded-full bg-emerald-600 border border-black pulse-dot" />
              <span>STREAMING</span>
            </div>
          ) : (
            <div className="bg-neutral-100 border-2 border-black text-xs font-mono font-black px-2.5 py-1 rounded-md shadow-[2px_2px_0px_#000] flex items-center gap-1.5 text-neutral-600">
              <span className="w-2.5 h-2.5 rounded-full bg-neutral-400 border border-black" />
              <span>IDLE / STANDBY</span>
            </div>
          )}
        </div>
      </div>

      {/* Live Ingest Snapshot Preview */}
      <div className="relative border-2 border-black rounded-lg overflow-hidden bg-neoDark text-white shadow-[2px_2px_0px_#000]">
        <div className="aspect-video w-full relative flex items-center justify-center bg-gradient-to-br from-neutral-900 via-neutral-950 to-neutral-900 overflow-hidden">
          {isRunning ? (
            <>
              {/* Retro scanlines & frame representation */}
              <div
                className="absolute inset-0 opacity-20 pointer-events-none"
                style={{
                  backgroundImage:
                    'repeating-linear-gradient(0deg, #000, #000 2px, transparent 2px, transparent 4px)',
                }}
              />
              <div className="absolute inset-0 flex flex-col justify-between p-3 z-10">
                <div className="flex justify-between items-start">
                  <div>
                    <span className="bg-black/70 border border-black text-neoMint font-mono font-black text-xs px-2 py-0.5 rounded shadow-[1px_1px_0px_#000] flex items-center gap-1.5 inline-flex">
                      <Radio className="w-3 h-3 text-red-500 animate-pulse" />
                      LIVE FEED
                    </span>
                    <p className="text-[10px] text-neutral-200 font-mono mt-1 font-bold">
                      NOW: {playlist[0]?.filename || 'source_stream.mp4'} [1/{playlist.length || 1}]
                    </p>
                  </div>
                  <span className="bg-black/70 border border-neutral-700 text-[10px] font-mono font-bold px-1.5 py-0.5 rounded text-neoYellow">
                    1080p @ 30fps
                  </span>
                </div>

                <div className="flex justify-between items-center text-[9px] font-mono text-neutral-300">
                  <span className="bg-black/60 px-1.5 py-0.5 rounded border border-neutral-700">
                    OVERLAY: {slot.overlay_clock_wib ? '[CLK+TITLE] ACTIVE' : 'PASSTHROUGH RAW'}
                  </span>
                  <span className="bg-black/60 px-1.5 py-0.5 rounded border border-neutral-700 text-neoYellow">
                    TIMESTAMP: {snapshotTime}
                  </span>
                </div>
              </div>
            </>
          ) : (
            <div className="text-center p-6 space-y-1 font-mono">
              <p className="text-sm font-bold text-neutral-400">NO ACTIVE INGEST FEED</p>
              <p className="text-[10px] text-neutral-500">
                Tekan [START STREAM] untuk memulai streaming passthrough FFmpeg
              </p>
            </div>
          )}
        </div>

        {/* Snapshot Refresh Button */}
        <div className="absolute bottom-1.5 right-1.5 z-20">
          <button
            type="button"
            className="neo-btn bg-white hover:bg-neutral-200 text-black border border-black font-mono text-[9px] font-black px-2 py-0.5 rounded shadow-[1px_1px_0px_#000] flex items-center gap-1 cursor-pointer"
            onClick={handleRefresh}
            title="Refresh snapshot dari pipe FFmpeg"
          >
            <RefreshCw className={`w-2.5 h-2.5 ${isRefreshing ? 'animate-spin' : ''}`} />
            <span>REFRESH</span>
          </button>
        </div>
      </div>

      {/* Stream Health Indicators */}
      <div className="grid grid-cols-3 gap-1.5 text-center text-[10px] font-mono">
        <div className="bg-white border border-black p-1 rounded shadow-[1px_1px_0px_#000]">
          <span className="text-neutral-500 block text-[9px]">PTS SYNC</span>
          <span className="font-black text-emerald-700">
            {ptsSync.toFixed(2)}ms {ptsSync === 0 ? '(EXCELLENT)' : '(WARN)'}
          </span>
        </div>
        <div className="bg-white border border-black p-1 rounded shadow-[1px_1px_0px_#000]">
          <span className="text-neutral-500 block text-[9px]">KEYFRAME CADENCE</span>
          <span className="font-black text-neutral-900">{keyframeGOP}</span>
        </div>
        <div className="bg-white border border-black p-1 rounded shadow-[1px_1px_0px_#000]">
          <span className="text-neutral-500 block text-[9px]">NET LATENCY</span>
          <span className="font-black text-emerald-700">{netLatency}ms (DIRECT)</span>
        </div>
      </div>

      {/* Source Strategy & Architecture Specs */}
      <div className="bg-[#f8f9fb] border-2 border-black rounded-lg p-2.5 space-y-1.5 text-xs font-mono">
        <div className="flex justify-between items-center border-b border-neutral-300 pb-1">
          <span className="text-neutral-500 font-bold uppercase text-[10px]">Source Strategy:</span>
          <span className="bg-neoPink border border-black font-black px-1.5 py-0.2 rounded text-[10px]">
            {slot.encoding_mode === 'passthrough'
              ? 'PASSTHROUGH (ZERO-TRANSCODE)'
              : 'TRANSCODE (FILTER OVERLAY)'}
          </span>
        </div>
        <div className="flex justify-between items-center text-[11px]">
          <span className="text-neutral-600">Direct Codec:</span>
          <strong className="text-black">H.264 (High) / AAC-LC Stereo</strong>
        </div>
        <div className="flex justify-between items-center text-[11px]">
          <span className="text-neutral-600">Overlap Guard:</span>
          <span className="text-emerald-700 font-black">YIELD TO PRIMARY</span>
        </div>
      </div>

      {/* Live Telemetry Bar */}
      <div className="bg-neoDark text-white border-2 border-black rounded-lg p-2 font-mono shadow-[2px_2px_0px_#000]">
        <div className="flex items-center justify-between text-[10px] text-neutral-400 mb-1 border-b border-neutral-700 pb-1">
          <span className="flex items-center gap-1 font-black text-neoMint">
            <Zap className="w-3 h-3" />
            LIVE TELEMETRY
          </span>
          <span>SPEED: {currentSpeed}</span>
        </div>
        <div className="grid grid-cols-4 gap-2 text-center">
          <div>
            <span className="text-[9px] text-neutral-400 block">FPS</span>
            <span className="font-black text-xs text-neoMint">{currentFps.toFixed(1)}</span>
          </div>
          <div>
            <span className="text-[9px] text-neutral-400 block">BITRATE</span>
            <span className="font-black text-xs text-neoBlue">{currentBitrate.toFixed(0)} kbps</span>
          </div>
          <div>
            <span className="text-[9px] text-neutral-400 block">DURATION</span>
            <span className="font-black text-xs text-white">{currentDuration}</span>
          </div>
          <div>
            <span className="text-[9px] text-neutral-400 block">PKT DROP</span>
            <span className="font-black text-xs text-neoCoral">{droppedFrames} (0.0%)</span>
          </div>
        </div>
      </div>

      {/* Playlist Queue Visualizer */}
      <div className="border-2 border-black rounded-lg p-2.5 bg-white space-y-1.5">
        <div className="flex items-center justify-between">
          <span className="font-mono font-black text-[11px] uppercase tracking-wider text-neutral-800">
            PLAYLIST QUEUE (CONCAT LOOP)
          </span>
          <span className="text-[10px] font-mono text-neutral-500 font-bold">
            {playlist.length} Files
          </span>
        </div>
        <div className="flex flex-wrap gap-1.5 font-mono text-[10px]">
          {playlist.map((item, idx) => {
            let badgeBg = 'bg-neutral-100';
            let label = '[QUEUED]';
            if (item.status === 'playing') {
              badgeBg = 'bg-neoMint';
              label = '[PLAYING]';
            } else if (item.status === 'next') {
              badgeBg = 'bg-neoBlue';
              label = '[NEXT]';
            }
            return (
              <span
                key={item.id || idx}
                className={`${badgeBg} border border-black rounded px-1.5 py-0.5 font-bold shadow-[1px_1px_0px_#000] flex items-center gap-1`}
              >
                <span className="font-black">{label}</span>
                <span className="truncate max-w-[120px]">{item.filename}</span>
              </span>
            );
          })}
        </div>
      </div>

      {/* Action Buttons */}
      <div className="flex items-center gap-2 pt-1">
        {isRunning ? (
          <button
            type="button"
            className="neo-btn flex-1 bg-neoCoral hover:bg-[#ffbfae] text-neutral-900 border-2 border-black font-mono font-black text-xs py-2 px-3 rounded-lg shadow-[3px_3px_0px_#000] flex items-center justify-center gap-1.5 cursor-pointer"
            onClick={() => onStop(slot.id)}
          >
            <Square className="w-3.5 h-3.5 fill-current" />
            <span>STOP STREAM</span>
          </button>
        ) : (
          <button
            type="button"
            className="neo-btn flex-1 bg-neoMint hover:bg-[#aef0ba] text-neutral-900 border-2 border-black font-mono font-black text-xs py-2 px-3 rounded-lg shadow-[3px_3px_0px_#000] flex items-center justify-center gap-1.5 cursor-pointer"
            onClick={() => onStart(slot.id)}
          >
            <Play className="w-3.5 h-3.5 fill-current" />
            <span>START STREAM</span>
          </button>
        )}

        <button
          type="button"
          className="neo-btn bg-white hover:bg-neutral-100 text-neutral-900 border-2 border-black font-mono font-bold text-xs py-2 px-3.5 rounded-lg shadow-[2px_2px_0px_#000] flex items-center gap-1.5 cursor-pointer"
          onClick={() => onOpenConfig(slot)}
        >
          <SlidersHorizontal className="w-3.5 h-3.5" />
          <span>CONFIG</span>
        </button>
      </div>
    </article>
  );
};
