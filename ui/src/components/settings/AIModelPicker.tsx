import React, { useEffect, useState, useCallback } from 'react';
import {
  Cpu, RefreshCw, CheckCircle2, Loader2, AlertTriangle,
  ChevronRight, Zap, HardDrive, Clock
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
  nvidia:   'bg-emerald-500/15 text-emerald-300 border-emerald-500/20',
  openai:   'bg-violet-500/15 text-violet-300 border-violet-500/20',
  meta:     'bg-blue-500/15 text-blue-300 border-blue-500/20',
  mistral:  'bg-amber-500/15 text-amber-300 border-amber-500/20',
  google:   'bg-pink-500/15 text-pink-300 border-pink-500/20',
  deepseek: 'bg-cyan-500/15 text-cyan-300 border-cyan-500/20',
  minimax:  'bg-orange-500/15 text-orange-300 border-orange-500/20',
  qwen:     'bg-teal-500/15 text-teal-300 border-teal-500/20',
  'z-ai':   'bg-purple-500/15 text-purple-300 border-purple-500/20',
};

const getFamilyClass = (model: NvidiaModel): string => {
  const family = (model.family || model.name.split(':')[0].split('/')[0] || '').toLowerCase();
  for (const [key, cls] of Object.entries(familyColor)) {
    if (family.includes(key)) return cls;
  }
  return 'bg-gray-500/15 text-gray-300 border-gray-500/20';
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
    } catch (e: any) {
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
        <Loader2 className="w-4 h-4 text-brand-400 animate-spin" />
        <span className="text-xs text-gray-400">Loading AI settings...</span>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2.5">
          <div className="w-7 h-7 rounded-lg bg-violet-500/20 border border-violet-500/30 flex items-center justify-center">
            <Cpu className="w-4 h-4 text-violet-400" />
          </div>
          <div>
            <h3 className="text-sm font-semibold text-white">AI Model</h3>
            <p className="text-[10px] text-gray-500 mt-0.5">Select which NVIDIA NIM cloud model powers the resume coach</p>
          </div>
        </div>
        <button
          onClick={handleRefreshModels}
          disabled={refreshing}
          className="btn-ghost text-[10px] flex items-center gap-1.5 px-2 py-1"
          title="Refresh model list from NVIDIA API"
          id="btn-refresh-models"
        >
          <RefreshCw className={`w-3 h-3 ${refreshing ? 'animate-spin' : ''}`} />
          Refresh
        </button>
      </div>

      {/* Active Model Badge */}
      {settings?.active_model && (
        <div className="flex items-center gap-2 px-3 py-2 rounded-lg bg-brand-500/10 border border-brand-500/20">
          <span className="w-2 h-2 rounded-full bg-brand-400 animate-pulse-slow flex-shrink-0" />
          <span className="text-[10px] text-gray-400 flex-shrink-0">Active:</span>
          <span className="text-xs font-mono text-brand-300 truncate">{settings.active_model}</span>
        </div>
      )}

      {/* Error */}
      {error && (
        <div className="flex items-start gap-2 px-3 py-2.5 rounded-lg bg-red-500/10 border border-red-500/20">
          <AlertTriangle className="w-3.5 h-3.5 text-red-400 flex-shrink-0 mt-0.5" />
          <p className="text-[10px] text-red-300 leading-relaxed">{error}</p>
        </div>
      )}

      {/* Model List */}
      <div className="space-y-2">
        {!settings?.available_models?.length ? (
          <div className="text-center py-8 space-y-2">
            <p className="text-xs text-gray-500">No local Ollama models found.</p>
            <p className="text-[10px] text-gray-600">
              Pull a model with: <code className="font-mono bg-white/5 px-1 rounded">ollama pull gemma3</code>
            </p>
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
                  w-full text-left px-4 py-3 rounded-xl border transition-all duration-200 group
                  ${isActive
                    ? 'bg-brand-500/10 border-brand-500/30 shadow-sm shadow-brand-500/10'
                    : 'glass border-white/5 hover:border-white/15 hover:bg-white/5'
                  }
                `}
              >
                <div className="flex items-center justify-between gap-3">
                  {/* Left: name + tags */}
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2 flex-wrap">
                      <span className={`text-xs font-mono font-medium truncate ${isActive ? 'text-brand-200' : 'text-white'}`}>
                        {model.name}
                      </span>
                      <span className={`text-[9px] px-1.5 py-0.5 rounded border font-medium uppercase tracking-wide ${getFamilyClass(model)}`}>
                        {getFamilyLabel(model)}
                      </span>
                    </div>
                    <div className="flex items-center gap-3 mt-1.5">
                      <span className="flex items-center gap-1 text-[10px] text-gray-500">
                        <HardDrive className="w-2.5 h-2.5" />
                        {formatSize(model.size)}
                      </span>
                      <span className="flex items-center gap-1 text-[10px] text-gray-500">
                        <Clock className="w-2.5 h-2.5" />
                        {formatAge(model.modified_at)}
                      </span>
                    </div>
                  </div>

                  {/* Right: status icon */}
                  <div className="flex-shrink-0">
                    {isSaving ? (
                      <Loader2 className="w-4 h-4 text-brand-400 animate-spin" />
                    ) : isSaved ? (
                      <CheckCircle2 className="w-4 h-4 text-emerald-400" />
                    ) : isActive ? (
                      <div className="flex items-center gap-1.5">
                        <span className="text-[10px] text-brand-400 font-medium">Active</span>
                        <Zap className="w-3.5 h-3.5 text-brand-400" />
                      </div>
                    ) : (
                      <ChevronRight className="w-4 h-4 text-gray-600 group-hover:text-gray-400 transition-colors" />
                    )}
                  </div>
                </div>
              </button>
            );
          })
        )}
      </div>

      {/* Footer hint */}
      <p className="text-[10px] text-gray-600 text-center pt-1">
        Pull new models via{' '}
        <code className="font-mono bg-white/5 px-1 rounded text-gray-400">ollama pull &lt;model&gt;</code>
        {' '}then click Refresh
      </p>
    </div>
  );
};

export default AIModelPicker;
