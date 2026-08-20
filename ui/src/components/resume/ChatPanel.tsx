import React, { useRef, useEffect, useState, useCallback } from 'react';
import {
  Send, Loader2, Bot, Check, X, HelpCircle,
  Sparkles, ChevronDown, RefreshCw, AlertCircle, Cpu,
  FileText, MessageSquare
} from 'lucide-react';
import { ChatMessage } from '../../types';
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
  const [appliedMessageIds, setAppliedMessageIds] = useState<Record<string, boolean>>({});
  const [customCommand, setCustomCommand] = useState('');
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

  const handleCustomCommandSubmit = (e?: React.FormEvent) => {
    if (e) e.preventDefault();
    if (!customCommand.trim() || loading) return;

    onSend(customCommand.trim(), undefined, true);
    setCustomCommand('');
  };

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
    <div className="flex flex-col h-full bg-surface-container-lowest border border-surface-variant rounded-2xl" id="chat-panel">
      {/* Header */}
      <div className="flex-shrink-0 px-4 py-3.5 border-b border-surface-variant bg-surface-container-low/50">
        <div className="flex items-center gap-2.5">
          <div className="w-8 h-8 rounded-full bg-primary-fixed text-primary flex items-center justify-center shadow-sm">
            <Sparkles className="w-4 h-4" />
          </div>
          <div className="flex-1 min-w-0">
            <p className="text-sm font-bold font-headline text-on-surface">AI Resume Coach</p>
            {jobTitle && !customJdEnabled && (
              <p className="text-[11px] text-on-surface-variant truncate">
                Tailoring for: {jobTitle}
              </p>
            )}
            {customJdEnabled && (
              <p className="text-[11px] text-tertiary font-semibold truncate flex items-center gap-1">
                <span className="w-1.5 h-1.5 rounded-full bg-tertiary"></span> Custom JD Enabled
              </p>
            )}
          </div>
          {/* Model Selector */}
          <div className="relative flex-shrink-0">
            <button
              id="btn-model-selector"
              onClick={() => setModelDropdownOpen(o => !o)}
              className="flex items-center gap-1.5 px-3 py-1.5 rounded-full bg-surface-container hover:bg-surface-container-high border border-surface-variant transition-all text-xs font-semibold text-on-surface max-w-[140px]"
              title="Switch AI model for this session"
            >
              <Cpu className="w-3.5 h-3.5 text-primary flex-shrink-0" />
              <span className="truncate">{selectedModel ? shortModelName(selectedModel) : 'Model'}</span>
              <ChevronDown className={`w-3 h-3 flex-shrink-0 transition-transform ${modelDropdownOpen ? 'rotate-180' : ''}`} />
            </button>
            {modelDropdownOpen && (
              <div className="absolute right-0 top-full mt-1.5 w-64 rounded-2xl border border-surface-variant bg-surface-container-lowest shadow-elevation-3 z-50 overflow-hidden animate-fade-in">
                <div className="px-3.5 py-2.5 border-b border-surface-variant bg-surface-container-low">
                  <p className="text-[11px] text-on-surface-variant font-bold uppercase tracking-wider">Select Model</p>
                </div>
                <div className="max-h-64 overflow-y-auto py-1">
                  {availableModels.length === 0 ? (
                    <p className="text-xs text-on-surface-variant px-3 py-3 text-center">No models found</p>
                  ) : (
                    availableModels.map(m => (
                      <button
                        key={m.name}
                        id={`chat-model-${m.name.replace(/[^a-z0-9]/gi, '-')}`}
                        onClick={() => handleModelSelect(m.name)}
                        className={`w-full text-left px-3.5 py-2.5 flex items-center justify-between gap-2 transition-colors hover:bg-surface-container ${
                          m.name === selectedModel ? 'text-primary font-bold bg-primary-fixed/30' : 'text-on-surface text-xs'
                        }`}
                      >
                        <span className="truncate">{shortModelName(m.name)}</span>
                        {m.name === selectedModel && (
                          <span className="w-2 h-2 rounded-full bg-primary flex-shrink-0" />
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
              className="text-xs bg-surface-container hover:bg-surface-container-high text-on-surface border border-surface-variant px-3 py-1.5 rounded-full flex items-center gap-1 font-semibold transition-all"
              title="Restart ATS tailoring scanner"
            >
              <RefreshCw className="w-3 h-3 text-primary" /> Re-Check
            </button>
          )}
        </div>
      </div>

      {/* Widget 1: AI Resume Coach (Scanner & Wizard Feedback) */}
      <div className="flex-1 min-h-0 overflow-y-auto px-4 py-4 space-y-4">
        {/* Empty State / Initial Trigger */}
        {messages.length === 0 && !loading && (
          <div className="flex flex-col items-center justify-center min-h-[180px] text-center py-6 gap-3 my-auto">
            <div className="w-14 h-14 rounded-full bg-primary-fixed flex items-center justify-center mb-1 text-primary shadow-sm">
              <Sparkles className="w-7 h-7" />
            </div>
            <p className="text-base font-bold font-headline text-on-surface">Tailor Your Resume</p>
            <p className="text-xs text-on-surface-variant max-w-[280px] leading-relaxed">
              Auto-scan your active resume against job requirements to calculate ATS score and fill key skill gaps.
            </p>
            <button
              onClick={() => {
                const jdContext = customJdEnabled && customJdText.trim() ? customJdText.trim() : undefined;
                onSend("Improve ATS score", jdContext);
              }}
              className="mt-2 w-full max-w-[280px] btn-primary text-xs py-2.5 px-5 flex items-center justify-center gap-2"
            >
              <Sparkles className="w-4 h-4" />
              Improve ATS Score
            </button>
          </div>
        )}

        {/* Error Alert Display */}
        {isError && !loading && (
          <div className="rounded-2xl border border-error/20 bg-error-container text-on-error-container p-4 text-xs space-y-3">
            <div className="flex items-start gap-2">
              <AlertCircle className="w-4 h-4 flex-shrink-0 mt-0.5" />
              <p className="leading-relaxed">{latestMessage.content}</p>
            </div>
            <button
              onClick={handleRestart}
              className="w-full py-2 rounded-full bg-error text-on-error font-semibold transition-all shadow-sm"
            >
              Retry Re-Check
            </button>
          </div>
        )}

        {/* Conversational Explanation text card */}
        {latestMessage && !isError && latestMessage.content && !loading && (
          <div className="rounded-2xl border border-surface-variant bg-surface-container-low p-4 text-xs text-on-surface leading-relaxed space-y-2">
            <div className="flex items-center gap-1.5 font-bold font-headline text-primary mb-1">
              <Sparkles className="w-4 h-4" />
              <span>Coach Feedback</span>
            </div>
            <p className="text-on-surface-variant">{latestMessage.content}</p>
          </div>
        )}

        {/* Tailored Result Action Card */}
        {latestMessage?.structuredResume && !loading && (
          <div className="rounded-2xl border border-primary/30 bg-primary-fixed/30 p-4 text-xs space-y-3">
            <div className="flex items-start gap-2.5">
              <Sparkles className="w-4 h-4 text-primary flex-shrink-0 mt-0.5" />
              <div>
                <p className="font-bold font-headline text-on-surface">AI Suggestion Ready</p>
                <p className="text-on-surface-variant text-[11px] mt-0.5 leading-relaxed">
                  Review the feedback above. Click below if you wish to apply these suggested changes to your Visual Canvas.
                </p>
              </div>
            </div>

            <div className="flex gap-2 pt-1">
              {onApplyFullResume && !appliedMessageIds[latestMessage.id] && (
                <button
                  onClick={() => {
                    onApplyFullResume(latestMessage.structuredResume!);
                    setAppliedMessageIds(prev => ({ ...prev, [latestMessage.id]: true }));
                  }}
                  className="flex-1 btn-primary text-xs py-2 px-3 flex items-center justify-center gap-1.5"
                >
                  <Check className="w-3.5 h-3.5" /> Apply to Canvas
                </button>
              )}
              <button
                onClick={() => {
                  handleRestart();
                }}
                className="btn-outline py-2 px-3 text-xs font-semibold flex items-center justify-center gap-1.5"
              >
                <RefreshCw className="w-3.5 h-3.5 text-primary" /> Re-Check
              </button>
            </div>
          </div>
        )}

        {/* Gap Questions Wizard */}
        {gapPrompts.length > 0 && !loading && (
          <div className="space-y-3 pt-2">
            <p className="text-[11px] font-bold text-on-surface-variant uppercase tracking-wider">
              Gap Questionnaire ({gapPrompts.length - unansweredCount}/{gapPrompts.length})
            </p>
            {gapPrompts.map((gap, i) => {
              const answer = answers[gap.skill];
              const isYes = answer?.answered && answer.hasSkill;
              const isNo = answer?.answered && !answer.hasSkill;

              return (
                <div key={i} className={`rounded-2xl border p-4 text-xs transition-all ${
                  answer?.answered 
                    ? 'border-surface-variant bg-surface-container-low' 
                    : 'border-primary/30 bg-primary-fixed/20'
                }`}>
                  <div className="flex items-start gap-2.5 mb-3">
                    <HelpCircle className="w-4 h-4 text-primary flex-shrink-0 mt-0.5" />
                    <div>
                      <span className="skill-pill-blue mb-1 inline-flex">{gap.skill}</span>
                      <p className="text-on-surface mt-1 font-medium">{gap.question}</p>
                    </div>
                  </div>
                  <div className="flex gap-2">
                    <button
                      onClick={() => setAnswers(prev => ({
                        ...prev,
                        [gap.skill]: { answered: true, hasSkill: true, details: prev[gap.skill]?.details || '' }
                      }))}
                      className={`flex-1 py-2 rounded-full transition-all font-semibold text-xs ${
                        isYes 
                          ? 'bg-tertiary text-on-tertiary shadow-sm' 
                          : 'bg-tertiary-fixed text-on-tertiary-fixed hover:bg-tertiary-fixed-dim'
                      }`}
                    >
                      Yes, I have it
                    </button>
                    <button
                      onClick={() => setAnswers(prev => ({
                        ...prev,
                        [gap.skill]: { answered: true, hasSkill: false, details: '' }
                      }))}
                      className={`flex-1 py-2 rounded-full transition-all font-semibold text-xs ${
                        isNo 
                          ? 'bg-secondary text-on-secondary shadow-sm' 
                          : 'bg-surface-container text-on-surface hover:bg-surface-container-high'
                      }`}
                    >
                      No, I don't
                    </button>
                  </div>

                  {/* Inline context input for Yes answers */}
                  {isYes && (
                    <div className="mt-3 space-y-1.5 animate-fade-in">
                      <p className="text-[10px] font-semibold text-on-surface-variant">Provide short detail/experience (optional):</p>
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
                        className="w-full bg-surface-container-lowest border border-surface-variant rounded-xl p-2.5 text-xs text-on-surface outline-none focus:border-primary focus:ring-1 focus:ring-primary/40 resize-none"
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
              className={`w-full py-2.5 px-4 rounded-full text-xs font-semibold shadow-elevation-1 transition-all flex items-center justify-center gap-2 mt-4 ${
                unansweredCount > 0
                  ? 'bg-surface-container text-on-surface-variant cursor-not-allowed shadow-none'
                  : 'btn-primary'
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
          <div className="flex gap-2.5 items-center">
            <div className="w-7 h-7 rounded-full bg-primary-fixed text-primary flex items-center justify-center">
              <Bot className="w-4 h-4" />
            </div>
            <div className="bg-surface-container-low border border-surface-variant rounded-2xl px-4 py-2.5 flex items-center gap-2">
              <div className="flex gap-1">
                <span className="w-1.5 h-1.5 rounded-full bg-primary animate-bounce" style={{ animationDelay: '0ms' }} />
                <span className="w-1.5 h-1.5 rounded-full bg-primary animate-bounce" style={{ animationDelay: '150ms' }} />
                <span className="w-1.5 h-1.5 rounded-full bg-primary animate-bounce" style={{ animationDelay: '300ms' }} />
              </div>
              <span className="text-xs text-on-surface-variant font-medium">Analyzing...</span>
            </div>
          </div>
        )}

        <div ref={bottomRef} />
      </div>

      {/* Widget 2: Custom Job Description Context Widget */}
      <div className="flex-shrink-0 mx-3 mb-2 rounded-2xl border border-surface-variant bg-surface-container-low p-3.5 shadow-sm">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <FileText className="w-4 h-4 text-primary" />
            <span className="text-xs font-bold text-on-surface">Custom Job Description Context</span>
          </div>
          <label className="relative inline-flex items-center cursor-pointer">
            <input
              type="checkbox"
              id="toggle-custom-jd"
              checked={customJdEnabled}
              onChange={(e) => onCustomJdEnabledChange(e.target.checked)}
              className="sr-only peer"
            />
            <div className="w-9 h-5 bg-surface-container-highest peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:bg-primary"></div>
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
              className="w-full bg-surface-container-lowest border border-surface-variant rounded-xl p-2.5 text-xs text-on-surface outline-none focus:border-primary focus:ring-1 focus:ring-primary/40 placeholder:text-outline resize-none"
            />
            <p className="text-[10px] text-on-surface-variant font-medium">
              💡 When enabled, custom JD context will be passed to AI for custom commands and ATS checks.
            </p>
          </div>
        )}
      </div>

      {/* Widget 3: Ask AI Custom Command Widget */}
      <div className="flex-shrink-0 mx-3 mb-3 rounded-2xl border border-surface-variant bg-surface-container-low p-3.5 shadow-sm">
        <div className="flex items-center gap-2 mb-2">
          <MessageSquare className="w-4 h-4 text-primary" />
          <span className="text-xs font-bold text-on-surface">Ask AI</span>
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
              className="w-full input-field pl-4 pr-8 py-2 text-xs"
            />
            {customCommand && (
              <button
                type="button"
                onClick={() => setCustomCommand('')}
                className="absolute right-3 top-1/2 -translate-y-1/2 text-outline hover:text-on-surface"
              >
                <X className="w-3.5 h-3.5" />
              </button>
            )}
          </div>
          <button
            type="submit"
            id="btn-send-custom-command"
            disabled={!customCommand.trim() || loading}
            className="btn-primary text-xs py-2 px-4 flex items-center gap-1.5 flex-shrink-0"
          >
            {loading ? <Loader2 className="w-3.5 h-3.5 animate-spin" /> : <Send className="w-3.5 h-3.5" />}
            Ask AI
          </button>
        </form>
        <p className="text-[10px] text-on-surface-variant mt-1.5 px-1 font-medium">
          Ask AI to update any section or skill. Result directly applies to canvas.
        </p>
      </div>
    </div>
  );
};

export default ChatPanel;

