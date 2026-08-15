import React, { useRef, useEffect, useState, useCallback } from 'react';
import {
  Send, Loader2, Bot, User, Check, X, HelpCircle,
  Sparkles, ChevronDown, ChevronUp, RefreshCw, AlertCircle, Cpu
} from 'lucide-react';
import { ChatMessage, TrackedChange } from '../../types';
import { getAISettings, NvidiaModel } from '../../services/api';

interface ChatPanelProps {
  messages: ChatMessage[];
  loading: boolean;
  trackedChanges: TrackedChange[];
  onSend: (text: string) => void;
  onAccept: (id: string) => void;
  onReject: (id: string) => void;
  onAnswerGap: (skill: string, answer: 'yes' | 'no' | string) => void;
  onSelectEdit?: (id: string) => void;
  jobTitle?: string;
  activeModel?: string;
  onModelChange?: (model: string) => void;
  onApplyFullResume?: (text: string) => void;
}

const ChatPanel: React.FC<ChatPanelProps> = ({
  messages,
  loading,
  trackedChanges,
  onSend,
  onAccept,
  onReject,
  onSelectEdit,
  jobTitle,
  activeModel: activeModelProp,
  onModelChange,
  onApplyFullResume,
}) => {
  const bottomRef = useRef<HTMLDivElement>(null);
  const [expandedEdits, setExpandedEdits] = useState<Record<string, boolean>>({});
  
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
    // Show just the last segment after / and drop the tag if it's 'latest'
    const base = name.split('/').pop() || name;
    const [id, tag] = base.split(':');
    return tag && tag !== 'latest' ? `${id}:${tag}` : id;
  };

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages, loading]);

  const toggleExpand = (id: string) =>
    setExpandedEdits(prev => ({ ...prev, [id]: !prev[id] }));

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
    onSend(responseText);
    setAnswers({});
  };

  const handleRestart = () => {
    setAnswers({});
    onSend("Improve ATS score");
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
            {jobTitle && (
              <p className="text-[10px] text-gray-500 truncate">
                Tailoring for: {jobTitle}
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

      {/* Messages / Wizard Container */}
      <div className="flex-1 overflow-y-auto px-4 py-3 space-y-4">
        {/* Empty State / Initial Trigger */}
        {messages.length === 0 && !loading && (
          <div className="flex flex-col items-center justify-center h-full text-center py-10 gap-4">
            <div className="w-12 h-12 rounded-2xl bg-brand-500/10 border border-brand-500/20 flex items-center justify-center">
              <Sparkles className="w-6 h-6 text-brand-400" />
            </div>
            <p className="text-sm font-medium text-white">Tailor Your Resume</p>
            <p className="text-xs text-gray-500 max-w-[240px]">
              Let AI scan your resume against the target job description and suggest improvements to boost your ATS match.
            </p>
            <button
              onClick={() => onSend("Improve ATS score")}
              className="mt-2 w-full py-2.5 px-4 rounded-lg bg-brand-600 hover:bg-brand-500 text-white text-xs font-semibold shadow-lg hover:shadow-brand-500/20 transition-all flex items-center justify-center gap-2"
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

        {/* Full Resume Replacement Card */}
        {latestMessage?.fullResumeReplacement && !loading && (
          <div className="rounded-xl border border-brand-500/30 bg-brand-500/5 p-4 text-xs space-y-3">
            <div className="flex items-start gap-2">
              <Sparkles className="w-4 h-4 text-brand-400 flex-shrink-0 mt-0.5" />
              <div>
                <p className="font-semibold text-white">Full Resume Redesign Available</p>
                <p className="text-gray-400 text-[11px] mt-0.5 leading-relaxed">
                  The AI has generated a complete, clean, and fully tailored resume that addresses all skill gaps and resolves any layout corruption.
                </p>
              </div>
            </div>

            <div className="flex gap-2">
              <button
                onClick={() => onApplyFullResume && onApplyFullResume(latestMessage.fullResumeReplacement || '')}
                className="flex-1 flex items-center justify-center gap-1.5 py-2 px-3 rounded-lg bg-brand-600 hover:bg-brand-500 text-white font-semibold shadow-lg hover:shadow-brand-500/20 transition-all"
              >
                <Check className="w-3.5 h-3.5" /> Apply Complete Resume
              </button>
            </div>
            
            {/* Simple Collapsible Preview */}
            <details className="text-[11px] text-gray-500 bg-white/5 rounded-lg border border-white/5">
              <summary className="cursor-pointer p-2 hover:text-white select-none font-medium">
                Preview New Resume
              </summary>
              <pre className="p-3 overflow-y-auto font-mono whitespace-pre-wrap max-h-48 text-[9px] border-t border-white/5 bg-black/20 text-gray-400 leading-normal">
                {latestMessage.fullResumeReplacement}
              </pre>
            </details>
          </div>
        )}

        {/* Proposed Edits Section */}
        {trackedChanges.filter(c => c.status === 'pending').length > 0 && !loading && (
          <div className="space-y-2">
            <p className="text-[10px] font-semibold text-gray-500 uppercase tracking-wider">Proposed Edits</p>
            {trackedChanges.map(change => {
              const status = change.status;
              const isExpanded = expandedEdits[change.id] ?? true;

              if (status !== 'pending') return null;

              return (
                <div key={change.id} className="rounded-xl border border-amber-500/30 bg-amber-500/5 text-xs overflow-hidden">
                  <div
                    className="flex items-center justify-between px-3 py-2 cursor-pointer"
                    onClick={() => {
                      toggleExpand(change.id);
                      if (onSelectEdit) onSelectEdit(change.id);
                    }}
                  >
                    <span className="font-semibold text-amber-300 flex items-center gap-1.5">
                      <Sparkles className="w-3 h-3" />
                      Proposed Edit
                    </span>
                    {isExpanded ? <ChevronUp className="w-3.5 h-3.5 text-gray-500" /> : <ChevronDown className="w-3.5 h-3.5 text-gray-500" />}
                  </div>

                  {isExpanded && (
                    <div className="px-3 pb-3 space-y-2">
                      <div className="bg-red-500/10 rounded-lg p-2 border border-red-500/20">
                        <p className="text-[10px] text-red-400 font-semibold mb-1">REMOVE</p>
                        <p className="text-gray-400 line-through">{change.original}</p>
                      </div>
                      <div className="bg-emerald-500/10 rounded-lg p-2 border border-emerald-500/20">
                        <p className="text-[10px] text-emerald-400 font-semibold mb-1">REPLACE WITH</p>
                        <p className="text-gray-200">{change.replacement}</p>
                      </div>
                      {change.reason && (
                        <p className="text-gray-500 italic">{change.reason}</p>
                      )}
                      <div className="flex gap-2 mt-2">
                        <button
                          onClick={() => onAccept(change.id)}
                          className="flex-1 flex items-center justify-center gap-1 py-1.5 rounded-lg bg-emerald-500/20 hover:bg-emerald-500/30 text-emerald-300 border border-emerald-500/30 transition-all text-xs font-medium"
                        >
                          <Check className="w-3.5 h-3.5" /> Accept
                        </button>
                        <button
                          onClick={() => onReject(change.id)}
                          className="flex-1 flex items-center justify-center gap-1 py-1.5 rounded-lg bg-red-500/10 hover:bg-red-500/20 text-red-400 border border-red-500/20 transition-all text-xs font-medium"
                        >
                          <X className="w-3.5 h-3.5" /> Reject
                        </button>
                      </div>
                    </div>
                  )}
                </div>
              );
            })}
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
    </div>
  );
};

export default ChatPanel;
