import React, { useEffect, useState } from 'react';
import { Plus, Trash2, ToggleLeft, ToggleRight, RefreshCw, Clock, Loader2, CheckCircle2, Globe } from 'lucide-react';
import { getSettings, updateSources } from '../../services/api';
import { ScraperConfig } from '../../types';

const SourceManager: React.FC = () => {
  const [sources, setSources] = useState<ScraperConfig[]>([]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [saved, setSaved] = useState(false);
  const [newSource, setNewSource] = useState({ board_name: '', target_url: '', cron_schedule: '@every 1h' });
  const [adding, setAdding] = useState(false);

  useEffect(() => {
    loadSettings();
  }, []);

  const loadSettings = async () => {
    setLoading(true);
    try {
      const settings = await getSettings();
      setSources(settings.sources || []);
    } catch (err) {
      console.error('Failed to load settings:', err);
    } finally {
      setLoading(false);
    }
  };

  const toggleSource = (id: number) => {
    setSources(prev => prev.map(s => s.id === id ? { ...s, enabled: !s.enabled } : s));
  };

  const removeSource = (id: number) => {
    setSources(prev => prev.filter(s => s.id !== id));
  };

  const handleSave = async () => {
    setSaving(true);
    try {
      await updateSources(sources);
      setSaved(true);
      setTimeout(() => setSaved(false), 2000);
    } catch (err) {
      console.error('Save failed:', err);
    } finally {
      setSaving(false);
    }
  };

  const handleAddSource = () => {
    if (!newSource.board_name || !newSource.target_url) return;
    setSources(prev => [...prev, {
      ...newSource,
      id: Date.now(),
      enabled: true,
      last_run_at: null,
    }]);
    setNewSource({ board_name: '', target_url: '', cron_schedule: '@every 1h' });
    setAdding(false);
  };

  const formatLastRun = (lastRunAt: string | null) => {
    if (!lastRunAt) return 'Never';
    const d = new Date(lastRunAt);
    const diff = Math.floor((Date.now() - d.getTime()) / 60000);
    if (diff < 60) return `${diff}m ago`;
    return `${Math.floor(diff / 60)}h ago`;
  };

  if (loading) {
    return (
      <div className="flex items-center gap-2 py-8 justify-center">
        <Loader2 className="w-5 h-5 text-primary animate-spin" />
        <span className="text-xs text-on-surface-variant font-medium">Loading settings...</span>
      </div>
    );
  }

  return (
    <div className="space-y-5">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2.5">
          <div className="w-8 h-8 rounded-full bg-tertiary-fixed text-tertiary flex items-center justify-center">
            <Globe className="w-4 h-4" />
          </div>
          <div>
            <h3 className="text-base font-bold font-headline text-on-surface">Job Board Sources</h3>
            <p className="text-xs text-on-surface-variant mt-0.5">Configure and manage remote job scraping targets</p>
          </div>
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={() => setAdding(true)}
            className="btn-ghost text-xs flex items-center gap-1.5 px-3 py-1.5"
            id="btn-add-source"
          >
            <Plus className="w-3.5 h-3.5 text-primary" /> Add Source
          </button>
          <button
            onClick={handleSave}
            disabled={saving}
            className="btn-primary text-xs py-1.5 px-4 flex items-center gap-1.5"
            id="btn-save-sources"
          >
            {saving ? <Loader2 className="w-3.5 h-3.5 animate-spin" /> : saved ? <CheckCircle2 className="w-3.5 h-3.5 text-white" /> : <RefreshCw className="w-3.5 h-3.5" />}
            {saved ? 'Saved!' : 'Save Changes'}
          </button>
        </div>
      </div>

      {/* Add new source form */}
      {adding && (
        <div className="bg-surface-container-low rounded-2xl p-5 border border-primary/20 animate-fade-in space-y-3.5 shadow-sm">
          <p className="text-xs font-bold text-primary uppercase tracking-wider">New Scraper Source</p>
          <input
            id="new-source-name"
            type="text"
            placeholder="Board name (e.g. MyJobBoard)"
            value={newSource.board_name}
            onChange={e => setNewSource(p => ({ ...p, board_name: e.target.value }))}
            className="input-field w-full"
          />
          <input
            id="new-source-url"
            type="url"
            placeholder="Target URL"
            value={newSource.target_url}
            onChange={e => setNewSource(p => ({ ...p, target_url: e.target.value }))}
            className="input-field w-full"
          />
          <input
            id="new-source-schedule"
            type="text"
            placeholder="Cron schedule (e.g. @every 1h)"
            value={newSource.cron_schedule}
            onChange={e => setNewSource(p => ({ ...p, cron_schedule: e.target.value }))}
            className="input-field w-full"
          />
          <div className="flex gap-2 pt-1">
            <button onClick={handleAddSource} className="btn-primary text-xs py-2 flex-1">Add Source</button>
            <button onClick={() => setAdding(false)} className="btn-outline py-2 text-xs flex-1">Cancel</button>
          </div>
        </div>
      )}

      {/* Sources list */}
      <div className="space-y-2.5">
        {sources.map(source => (
          <div
            key={source.id}
            className={`bg-surface-container-low rounded-2xl p-4 border border-surface-variant transition-all hover:bg-surface-container ${source.enabled ? '' : 'opacity-60'}`}
          >
            <div className="flex items-start justify-between gap-3">
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2">
                  <span className={`w-2.5 h-2.5 rounded-full flex-shrink-0 ${source.enabled ? 'bg-emerald-500' : 'bg-outline'}`} />
                  <p className="text-sm font-bold text-on-surface truncate">{source.board_name}</p>
                </div>
                <p className="text-xs text-on-surface-variant mt-0.5 truncate font-mono">{source.target_url}</p>
                <div className="flex items-center gap-3 mt-1.5">
                  <span className="flex items-center gap-1 text-[11px] text-on-surface-variant">
                    <RefreshCw className="w-3 h-3 text-primary" /> {source.cron_schedule}
                  </span>
                  <span className="flex items-center gap-1 text-[11px] text-on-surface-variant">
                    <Clock className="w-3 h-3" /> Last run: {formatLastRun(source.last_run_at)}
                  </span>
                </div>
              </div>
              <div className="flex items-center gap-1.5 flex-shrink-0">
                <button
                  onClick={() => toggleSource(source.id)}
                  className="btn-ghost p-1.5"
                  title={source.enabled ? 'Disable' : 'Enable'}
                  id={`toggle-source-${source.id}`}
                >
                  {source.enabled
                    ? <ToggleRight className="w-5 h-5 text-primary" />
                    : <ToggleLeft className="w-5 h-5 text-outline" />
                  }
                </button>
                <button
                  onClick={() => removeSource(source.id)}
                  className="btn-ghost p-1.5 hover:text-error"
                  id={`delete-source-${source.id}`}
                >
                  <Trash2 className="w-4 h-4" />
                </button>
              </div>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
};

export default SourceManager;

