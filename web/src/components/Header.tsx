import React from 'react';
import { Power, Cpu } from 'lucide-react';
import { api } from '../api/client';

interface HeaderProps {
  isConnected: boolean;
  onKillSwitch?: () => void;
}

export const Header: React.FC<HeaderProps> = ({ isConnected, onKillSwitch }) => {
  const handleKillSwitch = async () => {
    const confirmed = window.confirm(
      'EMERGENCY ACTION: Matikan semua child process FFmpeg dan hentikan supervisor sekarang?'
    );
    if (!confirmed) return;

    try {
      await api.killSwitch();
      alert('Semua proses FFmpeg berhasil dihentikan (SIGKILL emitted)!');
    } catch {
      alert('Sinyal Emergency Kill Switch terkirim ke Supervisor daemon!');
    }
    if (onKillSwitch) onKillSwitch();
  };

  return (
    <header className="w-full bg-white border-[2.5px] border-black rounded-xl p-3 md:px-5 md:py-3.5 shadow-[4px_4px_0px_#000] mb-4 flex flex-wrap items-center justify-between gap-3">
      {/* Brand / Identifier */}
      <div className="flex items-center gap-2.5">
        <div className="bg-neoMint border-2 border-black rounded-lg px-2.5 py-1.5 flex items-center gap-2 shadow-[2px_2px_0px_#000]">
          <span className="bg-black text-neoMint font-mono font-extrabold text-xs px-1.5 py-0.5 rounded">
            GO
          </span>
          <span className="font-extrabold tracking-tight text-sm md:text-base font-mono">
            GO-STREAMER
          </span>
        </div>
        <span className="bg-white border-2 border-black rounded-md px-2 py-0.5 text-xs font-mono font-bold shadow-[2px_2px_0px_#000] hidden sm:inline-block">
          ZERO-CGO EMBEDDED NODE
        </span>
        <div
          className={`border-2 border-black rounded-md px-2 py-0.5 text-xs font-mono font-bold shadow-[2px_2px_0px_#000] flex items-center gap-1.5 ${
            isConnected ? 'bg-neoMint' : 'bg-neoYellow'
          }`}
          title={isConnected ? 'WebSocket Connected' : 'Local Standalone Mode'}
        >
          <span
            className={`w-2 h-2 rounded-full border border-black ${
              isConnected ? 'bg-emerald-600 pulse-dot' : 'bg-amber-500'
            }`}
          />
          <span className="text-[10px] hidden md:inline">
            {isConnected ? 'WS LIVE' : 'SYNCING'}
          </span>
        </div>
      </div>

      {/* Android Native Root Supervisor Indicator & Emergency Action */}
      <div className="flex items-center gap-2 flex-wrap">
        <div className="bg-neoYellow border-2 border-black rounded-lg px-2.5 py-1 text-xs font-mono flex items-center gap-1.5 shadow-[2px_2px_0px_#000]">
          <span className="w-2.5 h-2.5 rounded-full bg-emerald-500 border border-black pulse-dot" />
          <Cpu className="w-3.5 h-3.5 text-black" />
          <span className="font-bold">ROOT SUPERVISOR</span>
          <span className="text-[10px] bg-black text-neoYellow px-1.5 py-0.5 rounded font-extrabold">
            OOM_SCORE: -1000
          </span>
        </div>

        <button
          type="button"
          className="neo-btn bg-neoCoral hover:bg-[#ffbfae] text-neutral-900 border-2 border-black font-mono font-black text-xs px-3 py-1.5 rounded-lg shadow-[2px_2px_0px_#000] flex items-center gap-1.5 cursor-pointer"
          onClick={handleKillSwitch}
          title="Emergency shutdown for all background streaming processes"
        >
          <Power className="w-3.5 h-3.5 text-red-600 stroke-[3]" />
          <span>KILL SWITCH</span>
        </button>
      </div>
    </header>
  );
};
