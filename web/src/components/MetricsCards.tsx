import React from 'react';
import { Cpu, Layers, Flame, HardDrive } from 'lucide-react';
import { SystemMetrics } from '../types';

interface MetricsCardsProps {
  metrics: SystemMetrics;
}

export const MetricsCards: React.FC<MetricsCardsProps> = ({ metrics }) => {
  const ramPercent = Math.min(
    100,
    Math.max(0, Math.round((metrics.ram_used_mb / (metrics.ram_total_mb || 1)) * 100))
  );

  const diskUsedGB = Math.max(0, metrics.disk_total_gb - metrics.disk_free_gb);
  const diskPercent = Math.min(
    100,
    Math.max(0, Math.round((diskUsedGB / (metrics.disk_total_gb || 1)) * 100))
  );

  const isOverheat = metrics.temperature_c > 48 || metrics.is_thermal_throttled;

  return (
    <section className="grid grid-cols-2 md:grid-cols-4 gap-3">
      {/* 1. CPU LOAD */}
      <div className="bg-neoBlue border-[2.5px] border-black rounded-xl p-3 shadow-[3px_3px_0px_#000] flex flex-col justify-between">
        <div className="flex items-center justify-between">
          <span className="font-mono font-black text-xs uppercase text-neutral-800 flex items-center gap-1">
            <Cpu className="w-3.5 h-3.5 stroke-[2.5]" />
            CPU LOAD
          </span>
          <span className="bg-white border border-black rounded px-1.5 py-0.2 text-[10px] font-mono font-bold">
            8 CORES
          </span>
        </div>

        <div className="my-2">
          <div className="font-mono text-2xl md:text-3xl font-black tracking-tight text-neutral-900">
            {metrics.cpu_percent.toFixed(1)}%
          </div>
          <p className="text-[10px] text-neutral-600 font-mono">
            {metrics.cpu_percent < 30 ? 'Low Overhead / Copy Mode' : 'Transcode Active'}
          </p>
        </div>

        <div>
          <div className="w-full bg-white border border-black rounded-full h-2 overflow-hidden">
            <div
              className="bg-black h-full transition-all duration-300"
              style={{ width: `${Math.min(100, Math.max(5, metrics.cpu_percent))}%` }}
            />
          </div>
        </div>
      </div>

      {/* 2. RAM MEMORY */}
      <div className="bg-neoMint border-[2.5px] border-black rounded-xl p-3 shadow-[3px_3px_0px_#000] flex flex-col justify-between">
        <div className="flex items-center justify-between">
          <span className="font-mono font-black text-xs uppercase text-neutral-800 flex items-center gap-1">
            <Layers className="w-3.5 h-3.5 stroke-[2.5]" />
            RAM MEMORY
          </span>
          <span className="bg-white border border-black rounded px-1.5 py-0.2 text-[10px] font-mono font-bold">
            {ramPercent}%
          </span>
        </div>

        <div className="my-2">
          <div className="font-mono text-2xl md:text-3xl font-black tracking-tight text-neutral-900">
            {(metrics.ram_used_mb / 1024).toFixed(1)}
            <span className="text-sm font-bold text-neutral-600 ml-1">
              / {(metrics.ram_total_mb / 1024).toFixed(1)} GB
            </span>
          </div>
          <p className="text-[10px] text-neutral-600 font-mono">
            Free: {(metrics.ram_free_mb / 1024).toFixed(1)} GB Available
          </p>
        </div>

        <div>
          <div className="w-full bg-white border border-black rounded-full h-2 overflow-hidden">
            <div
              className="bg-emerald-600 h-full transition-all duration-300"
              style={{ width: `${ramPercent}%` }}
            />
          </div>
        </div>
      </div>

      {/* 3. SOC THERMAL */}
      <div
        className={`border-[2.5px] border-black rounded-xl p-3 shadow-[3px_3px_0px_#000] flex flex-col justify-between transition-colors ${
          isOverheat ? 'bg-neoCoral' : 'bg-neoYellow'
        }`}
      >
        <div className="flex items-center justify-between">
          <span className="font-mono font-black text-xs uppercase text-neutral-800 flex items-center gap-1">
            <Flame className="w-3.5 h-3.5 stroke-[2.5] text-red-600" />
            SOC THERMAL
          </span>
          <span
            className={`border border-black rounded px-1.5 py-0.2 text-[9px] font-mono font-black ${
              isOverheat ? 'bg-red-600 text-white' : 'bg-white text-black'
            }`}
          >
            {isOverheat ? 'OVERHEAT >48°' : 'NOMINAL OK'}
          </span>
        </div>

        <div className="my-2">
          <div className="font-mono text-2xl md:text-3xl font-black tracking-tight text-neutral-900">
            {metrics.temperature_c.toFixed(1)}°C
          </div>
          <p className="text-[10px] text-neutral-600 font-mono">
            {isOverheat ? 'Thermal Throttling Guard Active' : 'Passive Android Thermal Zone'}
          </p>
        </div>

        <div>
          <div className="w-full bg-white border border-black rounded-full h-2 overflow-hidden">
            <div
              className={`h-full transition-all duration-300 ${
                isOverheat ? 'bg-red-600' : 'bg-amber-600'
              }`}
              style={{
                width: `${Math.min(100, Math.max(10, (metrics.temperature_c / 80) * 100))}%`,
              }}
            />
          </div>
        </div>
      </div>

      {/* 4. STORAGE BUFFER */}
      <div className="bg-neoLavender border-[2.5px] border-black rounded-xl p-3 shadow-[3px_3px_0px_#000] flex flex-col justify-between">
        <div className="flex items-center justify-between">
          <span className="font-mono font-black text-xs uppercase text-neutral-800 flex items-center gap-1">
            <HardDrive className="w-3.5 h-3.5 stroke-[2.5]" />
            STORAGE BUFFER
          </span>
          <span className="bg-white border border-black rounded px-1.5 py-0.2 text-[10px] font-mono font-bold">
            {diskPercent}% USED
          </span>
        </div>

        <div className="my-2">
          <div className="font-mono text-2xl md:text-3xl font-black tracking-tight text-neutral-900">
            {metrics.disk_free_gb.toFixed(1)}
            <span className="text-sm font-bold text-neutral-600 ml-1">GB FREE</span>
          </div>
          <p className="text-[10px] text-neutral-600 font-mono">
            Total: {metrics.disk_total_gb.toFixed(1)} GB (/data/media)
          </p>
        </div>

        <div>
          <div className="w-full bg-white border border-black rounded-full h-2 overflow-hidden">
            <div
              className="bg-purple-600 h-full transition-all duration-300"
              style={{ width: `${diskPercent}%` }}
            />
          </div>
        </div>
      </div>
    </section>
  );
};
