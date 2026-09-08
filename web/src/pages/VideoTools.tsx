import React, { useState, useRef } from 'react';
import {
  Upload,
  Sparkles,
  CheckCircle,
  AlertTriangle,
  Trash2,
  FileVideo,
  RefreshCw,
  Search,
} from 'lucide-react';
import { VideoItem } from '../types';

interface VideoToolsProps {
  videos: VideoItem[];
  onUpload: (file: File) => void;
  onFixCodec: (id: string) => void;
  onDelete: (id: string) => void;
}

export const VideoTools: React.FC<VideoToolsProps> = ({
  videos,
  onUpload,
  onFixCodec,
  onDelete,
}) => {
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [fixingIds, setFixingIds] = useState<Record<string, number>>({});
  const [searchQuery, setSearchQuery] = useState('');

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (file) {
      onUpload(file);
    }
  };

  const handleFixCodecClick = (id: string) => {
    setFixingIds((prev) => ({ ...prev, [id]: 10 }));
    onFixCodec(id);

    // Simulate transcode progress
    const timer = setInterval(() => {
      setFixingIds((prev) => {
        const current = prev[id] || 0;
        if (current >= 100) {
          clearInterval(timer);
          const next = { ...prev };
          delete next[id];
          return next;
        }
        return { ...prev, [id]: current + 20 };
      });
    }, 600);
  };

  const filteredVideos = videos.filter((v) =>
    v.filename.toLowerCase().includes(searchQuery.toLowerCase())
  );

  return (
    <div className="space-y-4">
      {/* Hidden file input */}
      <input
        type="file"
        ref={fileInputRef}
        onChange={handleFileChange}
        accept="video/mp4,video/mkv,video/mov,video/*"
        className="hidden"
      />

      {/* Main Container */}
      <div className="bg-white border-[2.5px] border-black rounded-xl p-4 md:p-5 shadow-[5px_5px_0px_#000] space-y-4">
        {/* Header toolbar */}
        <div className="flex flex-wrap items-center justify-between pb-3 border-b-2 border-black gap-3">
          <div>
            <h2 className="text-lg md:text-xl font-black uppercase font-mono text-neutral-900">
              Video Library & Codec Optimizer
            </h2>
            <p className="text-xs md:text-sm text-neutral-600 font-medium">
              Verifikasi kepatuhan media untuk salin RTMP zero-transcode (-c:v copy -c:a copy).
            </p>
          </div>

          <div className="flex items-center gap-2 flex-wrap">
            <div className="relative">
              <input
                type="text"
                placeholder="Cari video..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="bg-neutral-50 border-2 border-black rounded-lg py-1.5 pl-8 pr-2 font-mono text-xs shadow-[2px_2px_0px_#000] focus:ring-0 focus:outline-none"
              />
              <Search className="w-3.5 h-3.5 text-neutral-500 absolute left-2.5 top-2.5" />
            </div>

            <button
              type="button"
              onClick={() => fileInputRef.current?.click()}
              className="neo-btn bg-neoMint hover:bg-[#a6f0b3] border-2 border-black font-mono font-black text-xs px-3.5 py-2 rounded-lg shadow-[2px_2px_0px_#000] flex items-center gap-1.5 cursor-pointer"
            >
              <Upload className="w-3.5 h-3.5 stroke-[2.5]" />
              <span>Unggah Video</span>
            </button>
          </div>
        </div>

        {/* Offline Codec Fixer Banner */}
        <div className="bg-neoLavender border-2 border-black rounded-lg p-3.5 flex flex-wrap items-center justify-between gap-3 shadow-[2px_2px_0px_#000]">
          <div className="flex items-center gap-3">
            <div className="bg-white border-2 border-black p-2 rounded-lg shadow-[2px_2px_0px_#000]">
              <Sparkles className="w-5 h-5 text-purple-700" />
            </div>
            <div>
              <div className="flex items-center gap-2">
                <h4 className="font-mono font-black text-sm uppercase text-neutral-900">
                  Offline Codec Fixer & Transcoder
                </h4>
                <span className="bg-neoYellow text-black border border-black text-[9px] font-black px-1.5 py-0.2 rounded">
                  ARM NEON ACCEL
                </span>
              </div>
              <p className="text-xs text-neutral-700 mt-0.5">
                Konversi video yang bermasalah (Variable Frame Rate / codec non-H264) sebelum live agar
                proses streaming tetap berjalan 0% CPU overhead.
              </p>
            </div>
          </div>

          <div className="bg-white border-2 border-black px-3 py-1.5 rounded-lg text-xs font-mono font-bold shadow-[2px_2px_0px_#000]">
            Target Spec: H.264 High @ 30fps CFR • AAC-LC 128k
          </div>
        </div>

        {/* Video Table */}
        <div className="border-2 border-black rounded-lg overflow-hidden shadow-[2px_2px_0px_#000]">
          <div className="overflow-x-auto">
            <table className="w-full text-left font-mono text-xs border-collapse">
              <thead className="bg-neoCanvas border-b-2 border-black text-black">
                <tr>
                  <th className="p-2.5 font-black uppercase text-[11px]">Nama File Media</th>
                  <th className="p-2.5 font-black uppercase text-[11px]">Resolusi & FPS</th>
                  <th className="p-2.5 font-black uppercase text-[11px]">Video / Audio Codec</th>
                  <th className="p-2.5 font-black uppercase text-[11px]">Durasi & Size</th>
                  <th className="p-2.5 font-black uppercase text-[11px]">Status Kepatuhan</th>
                  <th className="p-2.5 font-black uppercase text-[11px] text-right">Aksi</th>
                </tr>
              </thead>
              <tbody className="divide-y-2 divide-neutral-200 bg-white">
                {filteredVideos.map((vid) => {
                  const isFixing = fixingIds[vid.id] !== undefined;
                  const progress = fixingIds[vid.id] || 0;

                  return (
                    <tr key={vid.id} className="hover:bg-neutral-50 transition-colors">
                      <td className="p-3 font-bold text-neutral-900">
                        <div className="flex items-center gap-2">
                          <FileVideo className="w-4 h-4 text-neutral-700 shrink-0" />
                          <span className="truncate max-w-[180px] md:max-w-xs">{vid.filename}</span>
                        </div>
                        {isFixing && (
                          <div className="mt-1.5 w-full bg-neutral-200 border border-black rounded-full h-2 overflow-hidden">
                            <div
                              className="bg-purple-600 h-full transition-all duration-300"
                              style={{ width: `${progress}%` }}
                            />
                          </div>
                        )}
                      </td>
                      <td className="p-3">
                        <span className="bg-neutral-100 border border-black px-1.5 py-0.5 rounded font-bold">
                          {vid.resolution} @ {vid.fps}fps
                        </span>
                      </td>
                      <td className="p-3">
                        <div className="text-[11px]">
                          <span className="text-black font-bold">{vid.video_codec}</span>
                          <span className="text-neutral-400 mx-1">/</span>
                          <span className="text-neutral-600">{vid.audio_codec}</span>
                        </div>
                      </td>
                      <td className="p-3 text-neutral-700">
                        <div>
                          {Math.floor(vid.duration_seconds / 60)}m{' '}
                          {Math.floor(vid.duration_seconds % 60)}s
                        </div>
                        <div className="text-[10px] text-neutral-500">
                          {(vid.file_size / (1024 * 1024)).toFixed(1)} MB
                        </div>
                      </td>
                      <td className="p-3">
                        {vid.is_passthrough_ready ? (
                          <span className="bg-neoMint border border-black px-2 py-1 rounded text-[10px] font-black shadow-[1px_1px_0px_#000] inline-flex items-center gap-1 text-black">
                            <CheckCircle className="w-3 h-3 text-emerald-800" />
                            PASSTHROUGH READY
                          </span>
                        ) : (
                          <span className="bg-neoYellow border border-black px-2 py-1 rounded text-[10px] font-black shadow-[1px_1px_0px_#000] inline-flex items-center gap-1 text-black">
                            <AlertTriangle className="w-3 h-3 text-amber-800" />
                            NEEDS CONVERSION
                          </span>
                        )}
                      </td>
                      <td className="p-3 text-right">
                        <div className="flex items-center justify-end gap-1.5">
                          {!vid.is_passthrough_ready && (
                            <button
                              type="button"
                              disabled={isFixing}
                              onClick={() => handleFixCodecClick(vid.id)}
                              className="neo-btn bg-neoLavender hover:bg-[#d0c2f5] border border-black px-2 py-1 rounded text-[11px] font-black shadow-[1px_1px_0px_#000] flex items-center gap-1 cursor-pointer"
                              title="Re-encode offline ke H.264 CFR"
                            >
                              <RefreshCw
                                className={`w-3 h-3 ${isFixing ? 'animate-spin' : ''}`}
                              />
                              <span>{isFixing ? `${progress}%` : 'Fix Codec'}</span>
                            </button>
                          )}
                          <button
                            type="button"
                            onClick={() => {
                              if (window.confirm(`Hapus video ${vid.filename}?`)) {
                                onDelete(vid.id);
                              }
                            }}
                            className="neo-btn bg-white hover:bg-red-50 text-red-600 border border-black p-1 rounded shadow-[1px_1px_0px_#000] cursor-pointer"
                            title="Hapus file"
                          >
                            <Trash2 className="w-3.5 h-3.5" />
                          </button>
                        </div>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </div>
  );
};
