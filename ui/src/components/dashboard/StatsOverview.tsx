import React from 'react';
import { TrendingUp, Briefcase, Star, Clock } from 'lucide-react';

interface StatsOverviewProps {
  total: number;
  highAts: number;
  skillCount: number;
  lastScrape?: string;
}

const StatsOverview: React.FC<StatsOverviewProps> = ({ total, highAts, skillCount, lastScrape }) => {
  const stats = [
    {
      label: 'Total Jobs',
      value: total,
      icon: Briefcase,
      color: 'text-brand-400',
      bg: 'bg-brand-500/10',
      border: 'border-brand-500/20',
    },
    {
      label: 'High ATS Match (>80%)',
      value: highAts,
      icon: Star,
      color: 'text-amber-400',
      bg: 'bg-amber-500/10',
      border: 'border-amber-500/20',
    },
    {
      label: 'Active Skills',
      value: skillCount,
      icon: TrendingUp,
      color: 'text-emerald-400',
      bg: 'bg-emerald-500/10',
      border: 'border-emerald-500/20',
    },
    {
      label: 'Last Scrape',
      value: lastScrape ? formatRelative(lastScrape) : 'Never',
      icon: Clock,
      color: 'text-purple-400',
      bg: 'bg-purple-500/10',
      border: 'border-purple-500/20',
    },
  ];

  return (
    <div className="grid grid-cols-2 lg:grid-cols-4 gap-4 mb-6">
      {stats.map((stat) => {
        const Icon = stat.icon;
        return (
          <div key={stat.label} className={`stat-card border ${stat.border} animate-fade-in`}>
            <div className="flex items-center justify-between">
              <span className="section-header mb-0">{stat.label}</span>
              <div className={`${stat.bg} ${stat.border} border rounded-lg p-1.5`}>
                <Icon className={`w-3.5 h-3.5 ${stat.color}`} />
              </div>
            </div>
            <div className={`text-2xl font-bold ${stat.color}`}>
              {typeof stat.value === 'number' ? stat.value.toLocaleString() : stat.value}
            </div>
          </div>
        );
      })}
    </div>
  );
};

function formatRelative(dateStr: string): string {
  const d = new Date(dateStr);
  const now = new Date();
  const diff = Math.floor((now.getTime() - d.getTime()) / 1000);
  if (diff < 60) return 'Just now';
  if (diff < 3600) return `${Math.floor(diff / 60)}m ago`;
  if (diff < 86400) return `${Math.floor(diff / 3600)}h ago`;
  return `${Math.floor(diff / 86400)}d ago`;
}

export default StatsOverview;
