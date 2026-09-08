import React, { useState } from 'react';
import { ShieldCheck, Plus, Clock, Trash2, Check } from 'lucide-react';
import { ScheduleItem } from '../types';

interface SchedulerTabProps {
  schedules: ScheduleItem[];
  overlapStatus?: {
    has_overlap: boolean;
    conflict_count: number;
    conflicts: any[];
    total_schedules: number;
  } | null;
  onAddSchedule: (item: Partial<ScheduleItem>) => void;
  onDeleteSchedule: (id: string) => void;
}

export const SchedulerTab: React.FC<SchedulerTabProps> = ({
  schedules,
  overlapStatus,
  onAddSchedule,
  onDeleteSchedule,
}) => {
  const [policy, setPolicy] = useState<ScheduleItem['overlap_guard_policy']>('yield_priority');
  const [isAdding, setIsAdding] = useState(false);
  const [newTitle, setNewTitle] = useState('');
  const [newCron, setNewCron] = useState('0 06 * * *');
  const [newDuration, setNewDuration] = useState(120);
  const [newSlotId, setNewSlotId] = useState(1);

  const handleCreate = (e: React.FormEvent) => {
    e.preventDefault();
    if (!newTitle) return;
    onAddSchedule({
      title: newTitle,
      cron_expr: newCron,
      duration_minutes: newDuration,
      slot_id: newSlotId,
      overlap_guard_policy: policy,
      is_enabled: true,
    });
    setNewTitle('');
    setIsAdding(false);
  };

  return (
    <div className="space-y-4">
      {/* Overlap Guard & Collision Header */}
      <div className="bg-white border-[2.5px] border-black rounded-xl p-4 md:p-5 shadow-[5px_5px_0px_#000] space-y-3">
        <div className="flex flex-wrap items-center justify-between gap-3 border-b-2 border-black pb-3">
          <div className="flex items-center gap-2">
            <div className="bg-neoYellow border-2 border-black p-1.5 rounded-lg shadow-[2px_2px_0px_#000]">
              <ShieldCheck className="w-5 h-5 text-neutral-900 stroke-[2.5]" />
            </div>
            <div>
              <div className="flex items-center gap-2">
                <h2 className="font-mono font-black text-base md:text-lg uppercase text-neutral-900">
                  Overlap Guard & Smart Scheduler
                </h2>
                <span className="bg-neoMint border border-black font-mono font-black text-[9px] px-1.5 py-0.2 rounded shadow-[1px_1px_0px_#000]">
                  ENABLED (ACTIVE)
                </span>
              </div>
              <p className="text-xs text-neutral-600 font-medium">
                Pencegahan bentrokan jadwal streaming otomatis antar-slot dengan kebijakan resolusi deterministik.
              </p>
            </div>
          </div>

          <button
            type="button"
            onClick={() => setIsAdding(!isAdding)}
            className="neo-btn bg-neoMint hover:bg-[#a6f0b3] border-2 border-black font-mono font-black text-xs px-3 py-1.5 rounded-lg shadow-[2px_2px_0px_#000] flex items-center gap-1 cursor-pointer"
          >
            <Plus className="w-3.5 h-3.5 stroke-[3]" />
            <span>Tambah Jadwal</span>
          </button>
        </div>

        {/* Policy & Status Grid */}
        <div className="grid grid-cols-1 md:grid-cols-2 gap-3 pt-1">
          <div className="p-3 bg-neutral-50 border-2 border-black rounded-lg space-y-1.5">
            <label className="font-mono font-black text-xs text-neutral-800 uppercase block">
              Collision Resolution Policy:
            </label>
            <select
              value={policy}
              onChange={(e) => setPolicy(e.target.value as ScheduleItem['overlap_guard_policy'])}
              className="w-full bg-white border-2 border-black rounded-lg p-2 font-mono text-xs shadow-[2px_2px_0px_#000] cursor-pointer font-bold focus:ring-0 focus:outline-none"
            >
              <option value="yield_priority">Yield to Highest Priority (Slot 1 Always Wins)</option>
              <option value="terminate_prev">Terminate Preceding Stream Gracefully</option>
              <option value="deny_new">Deny New Ingest Until Active Slot Clears</option>
            </select>
          </div>

          <div
            className={`p-3 border-2 border-black rounded-lg flex items-center gap-2.5 shadow-[2px_2px_0px_#000] ${
              overlapStatus?.has_overlap ? 'bg-neoCoral' : 'bg-neoPink'
            }`}
          >
            <ShieldCheck
              className={`w-6 h-6 shrink-0 stroke-[2.5] ${
                overlapStatus?.has_overlap ? 'text-red-700' : 'text-emerald-800'
              }`}
            />
            <div className="text-xs font-mono">
              <p className="font-black text-black">
                {overlapStatus?.has_overlap
                  ? `OVERLAP WARNING: ${overlapStatus.conflict_count} JADWAL BENTROK`
                  : 'NO TIME OVERLAPS DETECTED'}
              </p>
              <p className="text-neutral-700 text-[11px]">
                {overlapStatus?.has_overlap
                  ? `Kebijakan aktif (${policy}) akan diterapkan untuk mencegah bentrokan encoder.`
                  : 'Sistem jadwal aman dari bentrokan hardware encoder Termux.'}
              </p>
            </div>
          </div>
        </div>
      </div>

      {/* Add New Schedule Form (if open) */}
      {isAdding && (
        <form
          onSubmit={handleCreate}
          className="bg-white border-[2.5px] border-black rounded-xl p-4 shadow-[4px_4px_0px_#000] space-y-3 font-mono text-xs"
        >
          <div className="font-black uppercase text-sm border-b-2 border-black pb-1.5">
            Buat Tugas Jadwal Baru (Cron Task)
          </div>
          <div className="grid grid-cols-1 md:grid-cols-4 gap-3">
            <div>
              <label className="block font-bold text-neutral-700 uppercase mb-1">Nama Jadwal:</label>
              <input
                type="text"
                placeholder="Contoh: Morning Live Loop"
                value={newTitle}
                onChange={(e) => setNewTitle(e.target.value)}
                required
                className="w-full border-2 border-black rounded-lg p-2 font-mono text-xs shadow-[2px_2px_0px_#000] focus:ring-0 focus:outline-none"
              />
            </div>
            <div>
              <label className="block font-bold text-neutral-700 uppercase mb-1">Cron Pattern (WIB):</label>
              <input
                type="text"
                placeholder="0 06 * * *"
                value={newCron}
                onChange={(e) => setNewCron(e.target.value)}
                required
                className="w-full border-2 border-black rounded-lg p-2 font-mono text-xs shadow-[2px_2px_0px_#000] focus:ring-0 focus:outline-none"
              />
            </div>
            <div>
              <label className="block font-bold text-neutral-700 uppercase mb-1">Durasi (Menit):</label>
              <input
                type="number"
                value={newDuration}
                onChange={(e) => setNewDuration(Number(e.target.value))}
                min="1"
                required
                className="w-full border-2 border-black rounded-lg p-2 font-mono text-xs shadow-[2px_2px_0px_#000] focus:ring-0 focus:outline-none"
              />
            </div>
            <div>
              <label className="block font-bold text-neutral-700 uppercase mb-1">Target Engine:</label>
              <select
                value={newSlotId}
                onChange={(e) => setNewSlotId(Number(e.target.value))}
                className="w-full border-2 border-black rounded-lg p-2 font-mono text-xs shadow-[2px_2px_0px_#000] cursor-pointer font-bold focus:ring-0 focus:outline-none"
              >
                <option value={1}>Slot 1 (YouTube Main)</option>
                <option value={2}>Slot 2 (Twitch/FB)</option>
              </select>
            </div>
          </div>
          <div className="flex justify-end gap-2 pt-2">
            <button
              type="button"
              onClick={() => setIsAdding(false)}
              className="neo-btn bg-neutral-100 border-2 border-black px-3 py-1.5 rounded-lg font-bold shadow-[2px_2px_0px_#000] cursor-pointer"
            >
              Batal
            </button>
            <button
              type="submit"
              className="neo-btn bg-neoMint border-2 border-black px-4 py-1.5 rounded-lg font-black shadow-[2px_2px_0px_#000] flex items-center gap-1 cursor-pointer"
            >
              <Check className="w-3.5 h-3.5 stroke-[3]" />
              <span>Simpan Jadwal</span>
            </button>
          </div>
        </form>
      )}

      {/* Cron Job Automations List */}
      <div className="bg-white border-[2.5px] border-black rounded-xl p-4 md:p-5 shadow-[5px_5px_0px_#000] space-y-3">
        <div className="flex items-center justify-between border-b-2 border-black pb-2">
          <h3 className="font-mono font-black text-sm uppercase flex items-center gap-2">
            <Clock className="w-4 h-4 text-black" />
            Cron Job Automations & Active Routine
          </h3>
          <span className="font-mono text-xs text-neutral-500 font-bold">
            {schedules.length} Tasks Configured
          </span>
        </div>

        <div className="space-y-2.5 font-mono text-xs">
          {schedules.map((item) => (
            <div
              key={item.id}
              className="p-3 bg-[#f8f9fb] border-2 border-black rounded-xl flex flex-wrap justify-between items-center gap-2 shadow-[2px_2px_0px_#000]"
            >
              <div className="space-y-0.5">
                <div className="flex items-center gap-2">
                  <p className="font-black text-neutral-900 text-sm">{item.title}</p>
                  <span className="bg-neoBlue border border-black px-1.5 py-0.2 rounded text-[10px] font-bold">
                    SLOT {item.slot_id}
                  </span>
                </div>
                <p className="text-[11px] text-neutral-600">
                  Cron: <strong className="text-black">{item.cron_expr}</strong> (WIB) • Durasi:{' '}
                  {item.duration_minutes} menit • Policy: {item.overlap_guard_policy}
                </p>
              </div>

              <div className="flex items-center gap-2">
                <span
                  className={`border border-black font-black px-2 py-0.5 rounded text-[10px] shadow-[1px_1px_0px_#000] ${
                    item.is_enabled ? 'bg-neoMint text-black' : 'bg-neutral-200 text-neutral-600'
                  }`}
                >
                  {item.is_enabled ? 'ACTIVE' : 'PAUSED'}
                </span>
                <button
                  type="button"
                  onClick={() => onDeleteSchedule(item.id)}
                  className="neo-btn bg-white hover:bg-red-50 text-red-600 border border-black p-1 rounded shadow-[1px_1px_0px_#000] cursor-pointer"
                  title="Hapus task jadwal"
                >
                  <Trash2 className="w-3.5 h-3.5" />
                </button>
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
};
