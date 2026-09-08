import React, { useState, useEffect } from 'react';
import { Sliders, Clapperboard, Calendar, Globe, Clock, Terminal } from 'lucide-react';
import { ActiveTab } from '../types';

interface SidebarProps {
  activeTab: ActiveTab;
  onSelectTab: (tab: ActiveTab) => void;
  tunnelActive?: boolean;
}

export const Sidebar: React.FC<SidebarProps> = ({
  activeTab,
  onSelectTab,
  tunnelActive = true,
}) => {
  const [timeStr, setTimeStr] = useState<string>('');

  useEffect(() => {
    const updateTime = () => {
      const now = new Date();
      setTimeStr(
        now.toLocaleTimeString('id-ID', {
          timeZone: 'Asia/Jakarta',
          hour12: false,
        }) + ' WIB'
      );
    };
    updateTime();
    const timer = setInterval(updateTime, 1000);
    return () => clearInterval(timer);
  }, []);

  const navItems: { id: ActiveTab; label: string; icon: React.ReactNode; badge?: string }[] = [
    {
      id: 'desk',
      label: 'CONTROL DESK',
      icon: <Sliders className="w-4 h-4 stroke-[2.5]" />,
    },
    {
      id: 'video',
      label: 'VIDEO & CODEC TOOLS',
      icon: <Clapperboard className="w-4 h-4 stroke-[2.5]" />,
      badge: 'OFFLINE',
    },
    {
      id: 'scheduler',
      label: 'SMART SCHEDULER',
      icon: <Calendar className="w-4 h-4 stroke-[2.5]" />,
      badge: 'GUARD ON',
    },
    {
      id: 'remote',
      label: 'REMOTE & ALERTS',
      icon: <Globe className="w-4 h-4 stroke-[2.5]" />,
      badge: 'TUNNEL',
    },
  ];

  return (
    <aside className="lg:col-span-3 xl:col-span-2 space-y-4">
      <div className="bg-neoMint border-[2.5px] border-black rounded-xl p-3 shadow-[5px_5px_0px_#000]">
        <div className="flex items-center justify-between border-b-2 border-black pb-2 mb-3">
          <span className="font-mono text-xs font-black uppercase tracking-wider text-neutral-800">
            Navigation Node
          </span>
          <span className="bg-white border border-black rounded px-1.5 py-0.2 text-[10px] font-mono font-bold">
            v1.0-STABLE
          </span>
        </div>

        {/* 4 Core Navigation Tabs */}
        <nav className="space-y-2">
          {navItems.map((item) => {
            const isActive = activeTab === item.id;
            return (
              <button
                key={item.id}
                type="button"
                onClick={() => onSelectTab(item.id)}
                className={`neo-btn w-full flex items-center justify-between p-2.5 rounded-lg border-2 border-black font-mono font-bold text-xs transition-all text-left cursor-pointer ${
                  isActive
                    ? 'bg-white shadow-[3px_3px_0px_#000] translate-x-1 text-black font-black'
                    : 'bg-[#f3fbf5] hover:bg-white text-neutral-800 hover:shadow-[2px_2px_0px_#000]'
                }`}
              >
                <div className="flex items-center gap-2">
                  {item.icon}
                  <span>{item.label}</span>
                </div>
                {item.badge && (
                  <span className="bg-neoYellow text-[9px] font-black px-1.5 py-0.5 rounded border border-black shadow-[1px_1px_0px_#000]">
                    {item.badge}
                  </span>
                )}
              </button>
            );
          })}
        </nav>

        {/* Realtime Clock Widget */}
        <div className="mt-4 p-2.5 bg-white border-2 border-black rounded-lg text-center shadow-[2px_2px_0px_#000]">
          <div className="flex items-center justify-center gap-1.5 text-xs text-neutral-700 font-bold mb-0.5">
            <Clock className="w-3.5 h-3.5 text-neutral-600 stroke-[2.5]" />
            <span>NODE REALTIME</span>
          </div>
          <div className="font-mono text-base font-extrabold tracking-widest text-neutral-900">
            {timeStr || '12:00:00 WIB'}
          </div>
          <div className="text-[10px] text-neutral-500 font-mono mt-0.5">
            Asia/Jakarta (UTC+7)
          </div>
        </div>

        {/* Quick Tunnel Pill Status */}
        <div className="mt-3 p-2 bg-neoBlue border-2 border-black rounded-lg flex items-center justify-between text-xs font-mono">
          <div className="flex items-center gap-1.5">
            <span
              className={`w-2 h-2 rounded-full border border-black ${
                tunnelActive ? 'bg-emerald-600 pulse-dot' : 'bg-neutral-400'
              }`}
            />
            <span className="font-bold text-[11px]">TUNNEL:</span>
          </div>
          <span
            className={`font-black px-1.5 py-0.2 rounded border border-black text-[10px] ${
              tunnelActive ? 'bg-neoMint text-black' : 'bg-neoCoral text-black'
            }`}
          >
            {tunnelActive ? 'ONLINE' : 'STANDBY'}
          </span>
        </div>

        {/* Host Daemon Specs Info */}
        <div className="mt-3 p-2 bg-white/70 border border-black rounded-lg font-mono text-[11px] text-neutral-800">
          <div className="flex items-center gap-1 font-bold text-black mb-1">
            <Terminal className="w-3 h-3" />
            <span>Magisk Root Native</span>
          </div>
          <p className="text-neutral-600 text-[10px] mb-2 leading-tight">
            Termux Android UID:0 Daemon
          </p>
          <div className="flex items-center justify-between border-t border-neutral-300 pt-1.5">
            <span className="text-[10px] text-neutral-500">Wakelock:</span>
            <span className="bg-neoMint border border-black px-1 text-[9px] font-bold">
              ACTIVE
            </span>
          </div>
        </div>
      </div>
    </aside>
  );
};
