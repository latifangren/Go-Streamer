import React, { useState } from 'react';
import { Terminal } from 'lucide-react';
import { MetricsCards } from '../components/MetricsCards';
import { SlotCard } from '../components/SlotCard';
import { SlotConfigModal } from '../components/SlotConfigModal';
import { SystemMetrics, Slot, StreamTelemetry, LogEntry } from '../types';

interface ControlDeskProps {
  metrics: SystemMetrics;
  slots: Slot[];
  telemetries: Record<number, StreamTelemetry>;
  logs: LogEntry[];
  onStartSlot: (id: number) => void;
  onStopSlot: (id: number) => void;
  onUpdateSlot: (id: number, config: Partial<Slot>) => void;
  onRefreshSnapshot: (id: number) => void;
  onClearLogs: () => void;
}

export const ControlDesk: React.FC<ControlDeskProps> = ({
  metrics,
  slots,
  telemetries,
  logs,
  onStartSlot,
  onStopSlot,
  onUpdateSlot,
  onRefreshSnapshot,
  onClearLogs,
}) => {
  const [selectedSlot, setSelectedSlot] = useState<Slot | null>(null);
  const [isModalOpen, setIsModalOpen] = useState(false);

  const handleOpenConfig = (slot: Slot) => {
    setSelectedSlot(slot);
    setIsModalOpen(true);
  };

  return (
    <div className="space-y-4">
      {/* Top Status Hero Banner */}
      <section className="bg-white border-[2.5px] border-black rounded-xl p-4 md:p-5 shadow-[5px_5px_0px_#000] flex flex-wrap items-center justify-between gap-4">
        <div>
          <div className="flex items-center gap-2 mb-1 flex-wrap">
            <span className="bg-neoYellow border border-black font-mono font-extrabold text-[10px] px-2 py-0.5 rounded shadow-[1px_1px_0px_#000]">
              DAEMON STATUS • MARCH 2026
            </span>
            <span className="bg-neoBlue border border-black font-mono font-bold text-[10px] px-2 py-0.5 rounded shadow-[1px_1px_0px_#000]">
              TERMUX HOST
            </span>
            <span className="bg-neoPink border border-black font-mono font-bold text-[10px] px-2 py-0.5 rounded shadow-[1px_1px_0px_#000]">
              OVERLAP GUARD ON
            </span>
          </div>
          <h1 className="text-xl md:text-2xl font-black uppercase tracking-tight text-neutral-900">
            Control Desk & RTMP Telemetry
          </h1>
          <p className="text-xs md:text-sm font-medium text-neutral-600 mt-0.5">
            Zero-transcode hardware stream pipeline with Concat Playlist loops, Quick Tunnels &
            Health Checks.
          </p>
        </div>

        <div className="flex items-center gap-2">
          <div className="bg-neoMint border-2 border-black rounded-lg px-3.5 py-2 font-mono shadow-[3px_3px_0px_#000] flex items-center gap-2">
            <span className="w-3 h-3 rounded-full bg-emerald-600 border border-black pulse-dot" />
            <div>
              <div className="text-[10px] font-bold text-neutral-600 uppercase">SUPERVISOR</div>
              <div className="text-xs font-black text-black">DUAL-SLOT READY</div>
            </div>
          </div>
        </div>
      </section>

      {/* 4 Core Metrics Cards */}
      <MetricsCards metrics={metrics} />

      {/* Dual Streaming Slots Section */}
      <section className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        {slots.map((slot) => (
          <SlotCard
            key={slot.id}
            slot={slot}
            telemetry={telemetries[slot.slot_number]}
            onStart={onStartSlot}
            onStop={onStopSlot}
            onOpenConfig={handleOpenConfig}
            onRefreshSnapshot={onRefreshSnapshot}
          />
        ))}
      </section>

      {/* Retro Live Daemon Terminal & Audit Log Console */}
      <section className="bg-white border-[2.5px] border-black rounded-xl p-4 shadow-[5px_5px_0px_#000] space-y-3">
        <div className="flex items-center justify-between border-b-2 border-black pb-2">
          <div className="flex items-center gap-2">
            <Terminal className="w-4 h-4 text-black stroke-[2.5]" />
            <h3 className="font-mono font-black text-xs md:text-sm uppercase text-neutral-900">
              Live Supervisor Engine Terminal & Concat Pipeline Logs
            </h3>
          </div>
          <div className="flex items-center gap-2">
            <span className="bg-neoMint border border-black px-2 py-0.5 rounded font-mono text-[10px] font-bold shadow-[1px_1px_0px_#000]">
              AUTO-SCROLL: ON
            </span>
            <button
              type="button"
              onClick={onClearLogs}
              className="neo-btn bg-white hover:bg-neutral-100 border border-black px-2 py-0.5 rounded font-mono text-[10px] font-bold shadow-[1px_1px_0px_#000] cursor-pointer"
            >
              CLEAR
            </button>
          </div>
        </div>

        <div className="bg-neoDark border-2 border-black rounded-lg p-3.5 font-mono text-xs text-neutral-200 space-y-1.5 overflow-x-auto max-h-64 leading-relaxed">
          {logs.length === 0 ? (
            <p className="text-neutral-500 italic">[No logs captured in buffer]</p>
          ) : (
            logs.map((log) => {
              let tagColor = 'bg-neoMint text-black';
              if (log.tag === 'CONCAT') tagColor = 'bg-neoBlue text-black';
              if (log.tag === 'OVERLAY') tagColor = 'bg-neoLavender text-black';
              if (log.tag === 'RTMP') tagColor = 'bg-neoCoral text-black';
              if (log.tag === 'ALERT') tagColor = 'bg-neoYellow text-black';
              if (log.tag === 'ERR') tagColor = 'bg-red-500 text-white';

              return (
                <div key={log.id} className="flex items-start gap-2">
                  <span className={`${tagColor} font-extrabold px-1 rounded text-[10px] shrink-0`}>
                    [{log.tag}]
                  </span>
                  <span className="text-neutral-400 shrink-0">[{log.timestamp}]</span>
                  <span className="break-all">{log.message}</span>
                </div>
              );
            })
          )}
        </div>
      </section>

      {/* Slot Configuration Modal */}
      <SlotConfigModal
        slot={selectedSlot}
        isOpen={isModalOpen}
        onClose={() => setIsModalOpen(false)}
        onSave={onUpdateSlot}
      />
    </div>
  );
};
