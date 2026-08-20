import React, { useRef, useEffect, useState, useCallback } from 'react';
import {
  Send, Loader2, Bot, User, Check, X, HelpCircle,
  Sparkles, ChevronDown, ChevronUp, RefreshCw, AlertCircle, Cpu,
  FileText, MessageSquare
} from 'lucide-react';
import { ChatMessage, TrackedChange } from '../../types';
import { getAISettings, NvidiaModel } from '../../services/api';

interface ChatPanelProps {
  messages: ChatMessage[];
  loading: boolean;
  onSend: (text: string, customJd?: string, directCommand?: boolean) => void;
  onAnswerGap: (skill: string, answer: 'yes' | 'no' | string) => void;
  jobTitle?: string;
  activeModel?: string;
  onModelChange?: (model: string) => void;
  onApplyFullResume?: (text: string | object) => void;
  customJdText: string;
  customJdEnabled: boolean;
  onCustomJdTextChange: (text: string) => void;
  onCustomJdEnabledChange: (enabled: boolean) => void;
}

const ChatPanel: React.FC<ChatPanelProps> = ({
  messages,
  loading,
  onSend,
  jobTitle,
  activeModel: activeModelProp,
  onModelChange,
  onApplyFullResume,
  customJdText,
  customJdEnabled,
  onCustomJdTextChange,
  onCustomJdEnabledChange,
}) => {
  const bottomRef = useRef<HTMLDivElement>(null);
  const [expandedEdits, setExpandedEdits] = useState<Record<string, boolean>>({});
  const [appliedMessageIds, setAppliedMessageIds] = useState<Record<string, boolean>>({});
  
  // Custom command input state (no chat history maintained)
  const [customCommand, setCustomCommand] = useState('');

  // Track wizard answers
  const [answers, setAnswers] = useState<Record<string, { answered: boolean; hasSkill: boolean; details: string }>>({});
  const [availableModels, setAvailableModels] = useState<NvidiaModel[]>([]);
  const [selectedModel, setSelectedModel] = useState<string>(activeModelProp || '');
  const [modelDropdownOpen, setModelDropdownOpen] = useState(false);

  // Load models once on mount
  useEffect(() => {
    getAISettings().then(s => {
      setAvailableModels(s.available_models || []);
      if (!activeModelProp) setSelectedModel(s.active_model || '');
    }).catch(() => {});
  }, [activeModelProp]);

  // Sync if parent changes it
  useEffect(() => {
    if (activeModelProp) setSelectedModel(activeModelProp);
  }, [activeModelProp]);

  const handleModelSelect = useCallback((name: string) => {
    setSelectedModel(name);
    setModelDropdownOpen(false);
    onModelChange?.(name);
  }, [onModelChange]);

  const shortModelName = (name: string) => {
    const base = name.split('/').pop() || name;
    const [id, tag] = base.split(':');
    return tag && tag !== 'latest' ? `${id}:${tag}` : id;
  };

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages, loading]);

  // Handle submitting a custom command (feature 1 & 2)
  const handleCustomCommandSubmit = (e?: React.FormEvent) => {
    if (e) e.preventDefault();
    if (!customCommand.trim() || loading) return;

    // "Ask AI" box: direct command — skip JD context and gap analysis
    onSend(customCommand.trim(), undefined, true);
    setCustomCommand('');
  };

  // Find the latest assistant message
  const assistantMessages = messages.filter(m => m.role === 'assistant');
  const latestMessage = assistantMessages[assistantMessages.length - 1];

  const gapPrompts = latestMessage?.gapPrompts || [];
  const unansweredCount = gapPrompts.filter(p => !answers[p.skill]?.answered).length;
  const isError = latestMessage?.content?.includes('⚠️ Error:');

  const handleContinue = () => {
    if (unansweredCount > 0) return;
    
    let responseText = "Here are my details for the missing skills:\n";
    gapPrompts.forEach(p => {
      const ans = answers[p.skill];
      if (ans.hasSkill) {
        responseText += `- **${p.skill}**: Yes. Details: ${ans.details.trim() || 'I have experience in this area.'}\n`;
      } else {
        responseText += `- **${p.skill}**: No, I do not have experience with this.\n`;
      }
    });
    
    responseText += "\nPlease refine my resume and generate the improved section suggestions based on these inputs.";
    const jdContext = customJdEnabled && customJdText.trim() ? customJdText.trim() : undefined;
    onSend(responseText, jdContext);
    setAnswers({});
  };

  const handleRestart = () => {
    setAnswers({});
    const jdContext = customJdEnabled && customJdText.trim() ? customJdText.trim() : undefined;
    onSend("Improve ATS score", jdContext);
  };

  return (
    <div className="flex flex-col h-full bg-surface-100" id="chat-panel">
      {/* Header */}
      <div className="flex-shrink-0 px-4 py-3 border-b border-white/5">
        <div className="flex items-center gap-2">
          <div className="w-7 h-7 rounded-lg bg-brand-500/20 border border-brand-500/30 flex items-center justify-center">
            <Bot className="w-4 h-4 text-brand-400" />
          </div>
          <div className="flex-1 min-w-0">
            <p className="text-sm font-semibold text-white">AI Resume Coach</p>
            {jobTitle && !customJdEnabled && (
              <p className="text-[10px] text-gray-500 truncate">
                Tailoring for: {jobTitle}
              </p>
            )}
            {customJdEnabled && (
              <p className="text-[10px] text-emerald-400 font-medium truncate flex items-center gap-1">
                <span className="w-1.5 h-1.5 rounded-full bg-emerald-400"></span> Custom JD Enabled
              </p>
            )}
          </div>
          {/* Model Selector */}
          <div className="relative flex-shrink-0">
            <button
              id="btn-model-selector"
              onClick={() => setModelDropdownOpen(o => !o)}
              className="flex items-center gap-1.5 px-2 py-1 rounded-lg bg-white/5 hover:bg-white/10 border border-white/10 hover:border-white/20 transition-all text-[10px] text-gray-400 hover:text-white max-w-[130px]"
              title="Switch AI model for this session"
            >
              <Cpu className="w-3 h-3 text-violet-400 flex-shrink-0" />
              <span className="truncate font-mono">{selectedModel ? shortModelName(selectedModel) : 'Model'}</span>
              <ChevronDown className={`w-2.5 h-2.5 flex-shrink-0 transition-transform ${modelDropdownOpen ? 'rotate-180' : ''}`} />
            </button>
            {modelDropdownOpen && (
              <div className="absolute right-0 top-full mt-1 w-64 rounded-xl border border-white/10 bg-surface-100 shadow-2xl shadow-black/50 z-50 overflow-hidden animate-fade-in">
                <div className="px-3 py-2 border-b border-white/5">
                  <p className="text-[10px] text-gray-500 font-medium uppercase tracking-wide">Select Model</p>
                </div>
                <div className="max-h-64 overflow-y-auto py-1">
                  {availableModels.length === 0 ? (
                    <p className="text-[10px] text-gray-500 px-3 py-3 text-center">No models found</p>
                  ) : (
                    availableModels.map(m => (
                      <button
                        key={m.name}
                        id={`chat-model-${m.name.replace(/[^a-z0-9]/gi, '-')}`}
                        onClick={() => handleModelSelect(m.name)}
                        className={`w-full text-left px-3 py-2 flex items-center justify-between gap-2 transition-colors hover:bg-white/5 ${
                          m.name === selectedModel ? 'text-brand-300' : 'text-gray-300'
                        }`}
                      >
                        <span className="text-[11px] font-mono truncate">{shortModelName(m.name)}</span>
                        {m.name === selectedModel && (
                          <span className="w-1.5 h-1.5 rounded-full bg-brand-400 flex-shrink-0" />
                        )}
                      </button>
                    ))
                  )}
                </div>
              </div>
            )}
          </div>
          {messages.length > 0 && !loading && (
            <button
              onClick={handleRestart}
              className="text-[10px] bg-white/5 hover:bg-white/10 text-gray-400 hover:text-white border border-white/10 px-2 py-1 rounded-lg flex items-center gap-1 transition-all"
              title="Restart ATS tailoring scanner"
            >
              <RefreshCw className="w-3 h-3" /> Re-Check
            </button>
          )}
        </div>
      </div>

      {/* Widget 1: AI Resume Coach (Scanner & Wizard Feedback) */}
      <div className="flex-1 min-h-0 overflow-y-auto px-4 py-3 space-y-4">
        {/* Empty State / Initial Trigger */}
        {messages.length === 0 && !loading && (
          <div className="flex flex-col items-center justify-center min-h-[160px] text-center py-4 gap-3 my-auto">
            <div className="w-12 h-12 rounded-2xl bg-brand-500/10 border border-brand-500/20 flex items-center justify-center mb-1">
              <Sparkles className="w-6 h-6 text-brand-400" />
            </div>
            <p className="text-sm font-bold text-white">Tailor Your Resume</p>
            <p className="text-xs text-gray-400 max-w-[280px] leading-relaxed">
              Auto-scan your active resume against job requirements to calculate ATS score and fill key skill gaps.
            </p>
            <button
              onClick={() => {
                const jdContext = customJdEnabled && customJdText.trim() ? customJdText.trim() : undefined;
                onSend("Improve ATS score", jdContext);
              }}
              className="mt-2 w-full max-w-[280px] py-2.5 px-4 rounded-xl bg-brand-600 hover:bg-brand-500 text-white text-xs font-semibold shadow-lg hover:shadow-brand-500/20 transition-all flex items-center justify-center gap-2"
            >
              <Sparkles className="w-4 h-4" />
              Improve ATS Score
            </button>
          </div>
        )}

        {/* Error Alert Display */}
        {isError && !loading && (
          <div className="rounded-xl border border-red-500/20 bg-red-500/5 p-4 text-xs text-red-300 space-y-3">
            <div className="flex items-start gap-2">
              <AlertCircle className="w-4 h-4 text-red-400 flex-shrink-0 mt-0.5" />
              <p className="leading-relaxed">{latestMessage.content}</p>
            </div>
            <button
              onClick={handleRestart}
              className="w-full py-2 rounded-lg bg-red-500/10 hover:bg-red-500/20 text-red-300 border border-red-500/20 font-semibold transition-all"
            >
              Retry Re-Check
            </button>
          </div>
        )}

        {/* Conversational Explanation text card */}
        {latestMessage && !isError && latestMessage.content && !loading && (
          <div className="rounded-xl border border-white/5 bg-surface-200 p-3.5 text-xs text-gray-300 leading-relaxed space-y-1">
            <div className="flex items-center gap-1.5 font-semibold text-white mb-1.5">
              <Bot className="w-3.5 h-3.5 text-purple-400" />
              <span>Coach Feedback</span>
            </div>
            <p>{latestMessage.content}</p>
          </div>
        )}

        {/* Tailored Result Action Card */}
        {latestMessage?.structuredResume && !loading && (
          <div className="rounded-xl border border-brand-500/30 bg-brand-500/5 p-3.5 text-xs space-y-3">
            <div className="flex items-start gap-2">
              <Sparkles className="w-4 h-4 text-brand-400 flex-shrink-0 mt-0.5" />
              <div>
                <p className="font-semibold text-white">AI Suggestion Ready</p>
                <p className="text-gray-400 text-[11px] mt-0.5 leading-relaxed">
                  Review the feedback above. Click below if you wish to apply these suggested changes to your Visual Canvas.
                </p>
              </div>
            </div>

            <div className="flex gap-2">
              {onApplyFullResume && !appliedMessageIds[latestMessage.id] && (
                <button
                  onClick={() => {
                    onApplyFullResume(latestMessage.structuredResume!);
                    setAppliedMessageIds(prev => ({ ...prev, [latestMessage.id]: true }));
                  }}
                  className="flex-1 py-2 px-3 rounded-lg bg-brand-600 hover:bg-brand-500 text-white font-semibold shadow-md transition-all flex items-center justify-center gap-1.5"
                >
                  <Check className="w-3.5 h-3.5 text-emerald-300" /> Apply to Canvas
                </button>
              )}
              <button
                onClick={() => {
                  handleRestart();
                }}
                className="py-2 px-3 rounded-lg bg-surface-300 hover:bg-surface-400 text-gray-200 hover:text-white border border-white/10 font-semibold transition-all flex items-center justify-center gap-1.5"
              >
                <RefreshCw className="w-3.5 h-3.5 text-brand-400" /> Re-Check
              </button>
            </div>
          </div>
        )}

        {/* Gap Questions Wizard */}
        {gapPrompts.length > 0 && !loading && (
          <div className="space-y-3 pt-2">
            <p className="text-[10px] font-semibold text-gray-500 uppercase tracking-wider">Gap Questionnaire ({gapPrompts.length - unansweredCount}/{gapPrompts.length})</p>
            {gapPrompts.map((gap, i) => {
              const answer = answers[gap.skill];
              const isYes = answer?.answered && answer.hasSkill;
              const isNo = answer?.answered && !answer.hasSkill;

              return (
                <div key={i} className={`rounded-xl border p-3 text-xs transition-all ${
                  answer?.answered 
                    ? 'border-white/10 bg-white/5' 
                    : 'border-brand-500/30 bg-brand-500/5'
                }`}>
                  <div className="flex items-start gap-2 mb-2">
                    <HelpCircle className="w-3.5 h-3.5 text-brand-400 flex-shrink-0 mt-0.5" />
                    <div>
                      <span className="skill-pill-blue mb-1 inline-flex">{gap.skill}</span>
                      <p className="text-gray-300 mt-1">{gap.question}</p>
                    </div>
                  </div>
                  <div className="flex gap-2">
                    <button
                      onClick={() => setAnswers(prev => ({
                        ...prev,
                        [gap.skill]: { answered: true, hasSkill: true, details: prev[gap.skill]?.details || '' }
                      }))}
                      className={`flex-1 py-1.5 rounded-lg transition-all font-medium ${
                        isYes 
                          ? 'bg-emerald-500/30 text-emerald-200 border border-emerald-500/50' 
                          : 'bg-emerald-500/10 hover:bg-emerald-500/20 text-emerald-400 border border-emerald-500/20'
                      }`}
                    >
                      Yes, I have it
                    </button>
                    <button
                      onClick={() => setAnswers(prev => ({
                        ...prev,
                        [gap.skill]: { answered: true, hasSkill: false, details: '' }
                      }))}
                      className={`flex-1 py-1.5 rounded-lg transition-all ${
                        isNo 
                          ? 'bg-red-500/20 text-red-300 border border-red-500/35' 
                          : 'bg-surface-200 hover:bg-surface-300 text-gray-400 border border-white/5'
                      }`}
                    >
                      No, I don't
                    </button>
                  </div>

                  {/* Inline context input for Yes answers */}
                  {isYes && (
                    <div className="mt-3 space-y-1.5 animate-fade-in">
                      <p className="text-[10px] text-gray-500">Provide short detail/experience (optional):</p>
                      <textarea
                        value={answer.details}
                        onChange={(e) => {
                          const val = e.target.value;
                          setAnswers(prev => ({
                            ...prev,
                            [gap.skill]: { ...prev[gap.skill], details: val }
                          }));
                        }}
                        placeholder="e.g., Used this in project X for 2 years..."
                        rows={2}
                        className="w-full bg-surface-300 border border-white/10 rounded-lg p-2 text-xs text-white outline-none focus:border-brand-500/50 resize-none"
                      />
                    </div>
                  )}
                </div>
              );
            })}

            {/* Continue button */}
            <button
              onClick={handleContinue}
              disabled={unansweredCount > 0}
              className={`w-full py-2.5 px-4 rounded-lg text-xs font-semibold shadow-lg transition-all flex items-center justify-center gap-2 mt-4 ${
                unansweredCount > 0
                  ? 'bg-surface-200 text-gray-500 cursor-not-allowed border border-white/5 shadow-none'
                  : 'bg-brand-600 hover:bg-brand-500 text-white hover:shadow-brand-500/20'
              }`}
            >
              <Send className="w-3.5 h-3.5" />
              {unansweredCount > 0 
                ? `Answer all questions to continue (${gapPrompts.length - unansweredCount}/${gapPrompts.length})` 
                : 'Continue Tailoring'}
            </button>
          </div>
        )}

        {/* AI typing indicator */}
        {loading && (
          <div className="flex gap-2">
            <div className="w-6 h-6 rounded-lg bg-purple-500/20 border border-purple-500/30 flex items-center justify-center">
              <Bot className="w-3.5 h-3.5 text-purple-300" />
            </div>
            <div className="bg-surface-200 border border-white/5 rounded-xl px-4 py-3 flex items-center gap-2">
              <div className="flex gap-1">
                <span className="w-1.5 h-1.5 rounded-full bg-gray-500 animate-bounce" style={{ animationDelay: '0ms' }} />
                <span className="w-1.5 h-1.5 rounded-full bg-gray-500 animate-bounce" style={{ animationDelay: '150ms' }} />
                <span className="w-1.5 h-1.5 rounded-full bg-gray-500 animate-bounce" style={{ animationDelay: '300ms' }} />
              </div>
              <span className="text-xs text-gray-500">Analyzing...</span>
            </div>
          </div>
        )}

        <div ref={bottomRef} />
      </div>

      {/* Widget 2: Custom Job Description Context Widget */}
      <div className="flex-shrink-0 mx-3 mb-2 rounded-xl border border-white/10 bg-surface-200/90 p-3 shadow-md">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <FileText className="w-3.5 h-3.5 text-indigo-400" />
            <span className="text-xs font-semibold text-white">Custom Job Description Context</span>
          </div>
          <label className="relative inline-flex items-center cursor-pointer">
            <input
              type="checkbox"
              id="toggle-custom-jd"
              checked={customJdEnabled}
              onChange={(e) => onCustomJdEnabledChange(e.target.checked)}
              className="sr-only peer"
            />
            <div className="w-8 h-4 bg-gray-700 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-3 after:w-3 after:transition-all peer-checked:bg-indigo-600"></div>
          </label>
        </div>

        {customJdEnabled && (
          <div className="mt-2.5 animate-fade-in space-y-1.5">
            <textarea
              id="custom-jd-input"
              value={customJdText}
              onChange={(e) => onCustomJdTextChange(e.target.value)}
              placeholder="Paste custom job description or requirements context here..."
              rows={2}
              className="w-full bg-surface-300 border border-white/10 rounded-lg p-2 text-xs text-gray-200 outline-none focus:border-indigo-500/60 placeholder:text-gray-500 resize-none font-mono"
            />
            <p className="text-[10px] text-gray-400">
              💡 When enabled, custom JD context will be passed to AI for custom commands and ATS checks.
            </p>
          </div>
        )}
      </div>

      {/* Widget 3: Ask AI Custom Command Widget */}
      <div className="flex-shrink-0 mx-3 mb-3 rounded-xl border border-white/10 bg-surface-200/90 p-3 shadow-md">
        <div className="flex items-center gap-2 mb-2">
          <MessageSquare className="w-3.5 h-3.5 text-brand-400" />
          <span className="text-xs font-semibold text-white">Ask AI</span>
        </div>
        <form onSubmit={handleCustomCommandSubmit} className="flex gap-2">
          <div className="relative flex-1">
            <input
              type="text"
              id="custom-command-input"
              value={customCommand}
              onChange={(e) => setCustomCommand(e.target.value)}
              placeholder="e.g. Add TypeScript to skills, expand career summary..."
              disabled={loading}
              className="w-full bg-surface-300 border border-white/10 rounded-lg pl-3 pr-8 py-2 text-xs text-white placeholder-gray-500 outline-none focus:border-brand-500/50 transition-all disabled:opacity-50"
            />
            {customCommand && (
              <button
                type="button"
                onClick={() => setCustomCommand('')}
                className="absolute right-2 top-1/2 -translate-y-1/2 text-gray-400 hover:text-white"
              >
                <X className="w-3.5 h-3.5" />
              </button>
            )}
          </div>
          <button
            type="submit"
            id="btn-send-custom-command"
            disabled={!customCommand.trim() || loading}
            className="px-3 py-2 bg-brand-600 hover:bg-brand-500 disabled:opacity-50 text-white rounded-lg text-xs font-semibold flex items-center gap-1.5 transition-all shadow-md shadow-brand-500/20 flex-shrink-0"
          >
            {loading ? <Loader2 className="w-3.5 h-3.5 animate-spin" /> : <Send className="w-3.5 h-3.5" />}
            Ask AI
          </button>
        </form>
        <p className="text-[10px] text-gray-500 mt-1.5 px-0.5">
          Ask AI to update any section or skill. Result directly applies to canvas.
        </p>
      </div>
    </div>
  );
};

export default ChatPanel;
