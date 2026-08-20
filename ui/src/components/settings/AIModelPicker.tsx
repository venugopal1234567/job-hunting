import React, { useEffect, useState, useCallback } from 'react';
import {
  Cpu, RefreshCw, CheckCircle2, Loader2, AlertTriangle,
  ChevronRight, Sparkles, HardDrive, Clock
} from 'lucide-react';
import { getAISettings, updateAISettings, getAIModels, NvidiaModel, AISettings } from '../../services/api';

const formatSize = (bytes: number): string => {
  if (bytes === 0) return '—';
  const gb = bytes / 1_073_741_824;
  if (gb >= 1) return `${gb.toFixed(1)} GB`;
  const mb = bytes / 1_048_576;
  return `${mb.toFixed(0)} MB`;
};

const formatAge = (isoDate: string): string => {
  if (!isoDate) return '—';
  const diff = Date.now() - new Date(isoDate).getTime();
  const days = Math.floor(diff / 86_400_000);
  if (days === 0) return 'Today';
  if (days === 1) return 'Yesterday';
  if (days < 30) return `${days}d ago`;
  const months = Math.floor(days / 30);
  return `${months}mo ago`;
};

const familyColor: Record<string, string> = {
  nvidia:   'bg-primary-fixed text-on-primary-fixed border-primary-fixed-dim/40',
  openai:   'bg-primary-fixed text-on-primary-fixed border-primary-fixed-dim/40',
  meta:     'bg-secondary-fixed text-on-secondary-fixed border-secondary-fixed-dim/40',
  mistral:  'bg-secondary-fixed text-on-secondary-fixed border-secondary-fixed-dim/40',
  google:   'bg-tertiary-fixed text-on-tertiary-fixed border-tertiary-fixed-dim/40',
  deepseek: 'bg-tertiary-fixed text-on-tertiary-fixed border-tertiary-fixed-dim/40',
  minimax:  'bg-secondary-fixed text-on-secondary-fixed border-secondary-fixed-dim/40',
  qwen:     'bg-tertiary-fixed text-on-tertiary-fixed border-tertiary-fixed-dim/40',
  'z-ai':   'bg-primary-fixed text-on-primary-fixed border-primary-fixed-dim/40',
};

const getFamilyClass = (model: NvidiaModel): string => {
  const family = (model.family || model.name.split(':')[0].split('/')[0] || '').toLowerCase();
  for (const [key, cls] of Object.entries(familyColor)) {
    if (family.includes(key)) return cls;
  }
  return 'bg-surface-container text-on-surface-variant border-surface-variant';
};

const getFamilyLabel = (model: NvidiaModel): string => {
  return (model.family || model.name.split('/')[0] || 'nvidia').toLowerCase();
};

// ── Component ──────────────────────────────────────────────────────────────────

const AIModelPicker: React.FC = () => {
  const [settings, setSettings] = useState<AISettings | null>(null);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [saving, setSaving] = useState<string | null>(null);
  const [saved, setSaved] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    setError(null);
    try {
      const s = await getAISettings();
      setSettings(s);
    } catch {
      setError('Could not reach backend — make sure the Go server is running.');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { load(); }, [load]);

  const handleRefreshModels = async () => {
    setRefreshing(true);
    setError(null);
    try {
      const models = await getAIModels();
      setSettings(prev => prev ? { ...prev, available_models: models } : prev);
    } catch {
      setError('Failed to refresh model list from NVIDIA API.');
    } finally {
      setRefreshing(false);
    }
  };

  const handleSelectModel = async (modelName: string) => {
    if (saving || modelName === settings?.active_model) return;
    setSaving(modelName);
    setError(null);
    try {
      await updateAISettings(modelName);
      setSettings(prev => prev ? { ...prev, active_model: modelName } : prev);
      setSaved(modelName);
      setTimeout(() => setSaved(null), 2500);
    } catch {
      setError('Failed to save model selection. Please try again.');
    } finally {
      setSaving(null);
    }
  };

  if (loading) {
    return (
      <div className="flex items-center gap-2 py-6 justify-center">
        <Loader2 className="w-5 h-5 text-primary animate-spin" />
        <span className="text-xs text-on-surface-variant font-medium">Loading AI settings...</span>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2.5">
          <div className="w-8 h-8 rounded-full bg-primary-fixed text-primary flex items-center justify-center">
            <Cpu className="w-4 h-4" />
          </div>
          <div>
            <h3 className="text-base font-bold font-headline text-on-surface">AI Model</h3>
            <p className="text-xs text-on-surface-variant mt-0.5">Select which NVIDIA NIM cloud model powers the resume coach</p>
          </div>
        </div>
        <button
          onClick={handleRefreshModels}
          disabled={refreshing}
          className="btn-ghost text-xs flex items-center gap-1.5 px-3 py-1.5"
          title="Refresh model list from NVIDIA API"
          id="btn-refresh-models"
        >
          <RefreshCw className={`w-3.5 h-3.5 ${refreshing ? 'animate-spin' : ''}`} />
          Refresh
        </button>
      </div>

      {/* Active Model Badge */}
      {settings?.active_model && (
        <div className="flex items-center gap-2 px-4 py-2.5 rounded-2xl bg-primary-fixed/40 border border-primary-fixed-dim/60">
          <span className="w-2 h-2 rounded-full bg-primary animate-pulse-slow flex-shrink-0" />
          <span className="text-xs font-semibold text-on-surface-variant flex-shrink-0">Active:</span>
          <span className="text-xs font-bold text-primary truncate">{settings.active_model}</span>
        </div>
      )}

      {/* Error */}
      {error && (
        <div className="flex items-start gap-2 px-4 py-3 rounded-2xl bg-error-container text-on-error-container border border-error/20">
          <AlertTriangle className="w-4 h-4 flex-shrink-0 mt-0.5" />
          <p className="text-xs leading-relaxed">{error}</p>
        </div>
      )}

      {/* Model List */}
      <div className="space-y-2.5">
        {!settings?.available_models?.length ? (
          <div className="text-center py-8 space-y-2">
            <p className="text-xs text-on-surface-variant">No models found.</p>
          </div>
        ) : (
          settings.available_models.map(model => {
            const isActive = model.name === settings.active_model;
            const isSaving = saving === model.name;
            const isSaved  = saved  === model.name;

            return (
              <button
                key={model.name}
                id={`model-${model.name.replace(/[^a-z0-9]/gi, '-')}`}
                onClick={() => handleSelectModel(model.name)}
                disabled={!!saving}
                className={`
                  w-full text-left px-4 py-3.5 rounded-2xl border transition-all duration-200 group
                  ${isActive
                    ? 'bg-primary-fixed/30 border-primary shadow-elevation-1'
                    : 'bg-surface-container-low border-surface-variant hover:border-outline-variant hover:bg-surface-container'
                  }
                `}
              >
                <div className="flex items-center justify-between gap-3">
                  {/* Left: name + tags */}
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2 flex-wrap">
                      <span className={`text-xs font-semibold truncate ${isActive ? 'text-primary' : 'text-on-surface'}`}>
                        {model.name}
                      </span>
                      <span className={`text-[10px] px-2 py-0.5 rounded-full border font-semibold uppercase tracking-wider ${getFamilyClass(model)}`}>
                        {getFamilyLabel(model)}
                      </span>
                    </div>
                    <div className="flex items-center gap-3 mt-1.5">
                      <span className="flex items-center gap-1 text-[11px] text-on-surface-variant">
                        <HardDrive className="w-3 h-3" />
                        {formatSize(model.size)}
                      </span>
                      <span className="flex items-center gap-1 text-[11px] text-on-surface-variant">
                        <Clock className="w-3 h-3" />
                        {formatAge(model.modified_at)}
                      </span>
                    </div>
                  </div>

                  {/* Right: status icon */}
                  <div className="flex-shrink-0">
                    {isSaving ? (
                      <Loader2 className="w-4 h-4 text-primary animate-spin" />
                    ) : isSaved ? (
                      <CheckCircle2 className="w-4 h-4 text-emerald-500" />
                    ) : isActive ? (
                      <div className="flex items-center gap-1.5 bg-primary text-on-primary px-3 py-1 rounded-full text-xs font-semibold shadow-sm">
                        <span>Active</span>
                        <Sparkles className="w-3 h-3 text-white" />
                      </div>
                    ) : (
                      <ChevronRight className="w-4 h-4 text-outline group-hover:text-on-surface transition-colors" />
                    )}
                  </div>
                </div>
              </button>
            );
          })
        )}
      </div>
    </div>
  );
};

export default AIModelPicker;

