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
      color: 'text-primary',
      iconBg: 'bg-primary-fixed text-on-primary-fixed',
      borderColor: 'border-surface-variant',
    },
    {
      label: 'High ATS Match (>80%)',
      value: highAts,
      icon: Star,
      color: 'text-secondary',
      iconBg: 'bg-secondary-fixed text-on-secondary-fixed',
      borderColor: 'border-surface-variant',
    },
    {
      label: 'Active Skills',
      value: skillCount,
      icon: TrendingUp,
      color: 'text-tertiary',
      iconBg: 'bg-tertiary-fixed text-on-tertiary-fixed',
      borderColor: 'border-surface-variant',
    },
    {
      label: 'Last Scrape',
      value: lastScrape ? formatRelative(lastScrape) : 'Never',
      icon: Clock,
      color: 'text-on-surface-variant',
      iconBg: 'bg-surface-container-highest text-on-surface-variant',
      borderColor: 'border-surface-variant',
    },
  ];

  return (
    <div className="grid grid-cols-2 lg:grid-cols-4 gap-4 sm:gap-6 mb-6">
      {stats.map((stat) => {
        const Icon = stat.icon;
        return (
          <div
            key={stat.label}
            className={`bg-surface-container-lowest border ${stat.borderColor} rounded-2xl p-5 sm:p-6 shadow-elevation-1 flex flex-col justify-between hover:shadow-elevation-2 transition-all duration-200 animate-fade-in`}
          >
            <div className="flex items-center justify-between">
              <span className="text-[11px] font-semibold uppercase tracking-wider text-on-surface-variant">
                {stat.label}
              </span>
              <div className={`${stat.iconBg} rounded-full p-2.5 flex items-center justify-center`}>
                <Icon className="w-4 h-4" />
              </div>
            </div>
            <div className={`text-3xl sm:text-4xl font-bold font-headline mt-3 ${stat.color}`}>
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

