import React, { useRef, useEffect, useState } from 'react';
import {
  Send, Loader2, Bot, User, Check, X, HelpCircle,
  Sparkles, ChevronDown, ChevronUp
} from 'lucide-react';
import { ChatMessage, TrackedChange } from '../../types';

interface ChatPanelProps {
  messages: ChatMessage[];
  loading: boolean;
  trackedChanges: TrackedChange[];
  onSend: (text: string) => void;
  onAccept: (id: string) => void;
  onReject: (id: string) => void;
  onAnswerGap: (skill: string, answer: 'yes' | 'no' | string) => void;
  jobTitle?: string;
}

const ChatPanel: React.FC<ChatPanelProps> = ({
  messages,
  loading,
  trackedChanges,
  onSend,
  onAccept,
  onReject,
  onAnswerGap,
  jobTitle,
}) => {
  const [input, setInput] = useState('');
  const bottomRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLTextAreaElement>(null);
  const [expandedEdits, setExpandedEdits] = useState<Record<string, boolean>>({});

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages, loading]);

  const handleSend = () => {
    if (!input.trim() || loading) return;
    onSend(input.trim());
    setInput('');
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  };

  const getChangeForEdit = (editId: string) =>
    trackedChanges.find(c => c.id === editId);

  const toggleExpand = (id: string) =>
    setExpandedEdits(prev => ({ ...prev, [id]: !prev[id] }));

  return (
    <div className="flex flex-col h-full" id="chat-panel">
      {/* Header */}
      <div className="flex-shrink-0 px-4 py-3 border-b border-white/5 flex items-center gap-2">
        <div className="w-7 h-7 rounded-lg bg-brand-500/20 border border-brand-500/30 flex items-center justify-center">
          <Bot className="w-4 h-4 text-brand-400" />
        </div>
        <div>
          <p className="text-sm font-semibold text-white">AI Resume Coach</p>
          {jobTitle && (
            <p className="text-[10px] text-gray-500 truncate max-w-[180px]">
              Tailoring for: {jobTitle}
            </p>
          )}
        </div>
        <span className="ml-auto text-[10px] bg-emerald-500/20 text-emerald-300 border border-emerald-500/30 px-1.5 py-0.5 rounded-full">
          Gemma 4
        </span>
      </div>

      {/* Messages */}
      <div className="flex-1 overflow-y-auto px-4 py-3 space-y-4">
        {messages.length === 0 && !loading && (
          <div className="flex flex-col items-center justify-center h-full text-center py-10 gap-3">
            <div className="w-12 h-12 rounded-2xl bg-brand-500/10 border border-brand-500/20 flex items-center justify-center">
              <Sparkles className="w-6 h-6 text-brand-400" />
            </div>
            <p className="text-sm font-medium text-white">Your AI Resume Coach</p>
            <p className="text-xs text-gray-500 max-w-[220px]">
              Ask me to improve bullet points, add skills, fix formatting, or tailor this resume for the job.
            </p>
            <div className="flex flex-col gap-2 w-full mt-2">
              {[
                'Improve my experience bullets',
                'What skills am I missing?',
                'Rewrite my summary',
              ].map(s => (
                <button
                  key={s}
                  onClick={() => onSend(s)}
                  className="text-xs text-left px-3 py-2 rounded-lg bg-surface-200 hover:bg-surface-300 text-gray-300 hover:text-white border border-white/5 transition-all"
                >
                  {s}
                </button>
              ))}
            </div>
          </div>
        )}

        {messages.map(msg => (
          <div key={msg.id} className={`flex gap-2 ${msg.role === 'user' ? 'flex-row-reverse' : 'flex-row'}`}>
            {/* Avatar */}
            <div className={`w-6 h-6 rounded-lg flex-shrink-0 flex items-center justify-center mt-1 ${
              msg.role === 'user'
                ? 'bg-brand-500/30 border border-brand-500/40'
                : 'bg-purple-500/20 border border-purple-500/30'
            }`}>
              {msg.role === 'user'
                ? <User className="w-3.5 h-3.5 text-brand-300" />
                : <Bot className="w-3.5 h-3.5 text-purple-300" />
              }
            </div>

            {/* Bubble */}
            <div className={`flex-1 max-w-[85%] space-y-2 ${msg.role === 'user' ? 'items-end' : 'items-start'} flex flex-col`}>
              <div className={`rounded-xl px-3 py-2 text-sm leading-relaxed ${
                msg.role === 'user'
                  ? 'bg-brand-600/30 border border-brand-500/30 text-white'
                  : 'bg-surface-200 border border-white/5 text-gray-200'
              }`}>
                {msg.content}
              </div>

              {/* Proposed Edits */}
              {msg.proposedEdits && msg.proposedEdits.length > 0 && (
                <div className="w-full space-y-2">
                  {msg.proposedEdits.map(edit => {
                    const change = getChangeForEdit(edit.id);
                    const status = change?.status || 'pending';
                    const isExpanded = expandedEdits[edit.id] ?? true;

                    return (
                      <div key={edit.id} className={`rounded-xl border text-xs overflow-hidden ${
                        status === 'accepted' ? 'border-emerald-500/30 bg-emerald-500/5' :
                        status === 'rejected' ? 'border-red-500/20 bg-red-500/5 opacity-60' :
                        'border-amber-500/30 bg-amber-500/5'
                      }`}>
                        <div
                          className="flex items-center justify-between px-3 py-2 cursor-pointer"
                          onClick={() => toggleExpand(edit.id)}
                        >
                          <span className="font-semibold text-amber-300 flex items-center gap-1.5">
                            <Sparkles className="w-3 h-3" />
                            {status === 'accepted' ? '✓ Applied' : status === 'rejected' ? '✗ Rejected' : 'Proposed Edit'}
                          </span>
                          {isExpanded ? <ChevronUp className="w-3.5 h-3.5 text-gray-500" /> : <ChevronDown className="w-3.5 h-3.5 text-gray-500" />}
                        </div>

                        {isExpanded && (
                          <div className="px-3 pb-3 space-y-2">
                            <div className="bg-red-500/10 rounded-lg p-2 border border-red-500/20">
                              <p className="text-[10px] text-red-400 font-semibold mb-1">REMOVE</p>
                              <p className="text-gray-400 line-through">{edit.original}</p>
                            </div>
                            <div className="bg-emerald-500/10 rounded-lg p-2 border border-emerald-500/20">
                              <p className="text-[10px] text-emerald-400 font-semibold mb-1">REPLACE WITH</p>
                              <p className="text-gray-200">{edit.replacement}</p>
                            </div>
                            {edit.reason && (
                              <p className="text-gray-500 italic">{edit.reason}</p>
                            )}
                            {status === 'pending' && (
                              <div className="flex gap-2 mt-2">
                                <button
                                  id={`btn-accept-change-${edit.id}`}
                                  onClick={() => onAccept(edit.id)}
                                  className="flex-1 flex items-center justify-center gap-1 py-1.5 rounded-lg bg-emerald-500/20 hover:bg-emerald-500/30 text-emerald-300 border border-emerald-500/30 transition-all text-xs font-medium"
                                >
                                  <Check className="w-3.5 h-3.5" /> Accept
                                </button>
                                <button
                                  id={`btn-reject-change-${edit.id}`}
                                  onClick={() => onReject(edit.id)}
                                  className="flex-1 flex items-center justify-center gap-1 py-1.5 rounded-lg bg-red-500/10 hover:bg-red-500/20 text-red-400 border border-red-500/20 transition-all text-xs font-medium"
                                >
                                  <X className="w-3.5 h-3.5" /> Reject
                                </button>
                              </div>
                            )}
                          </div>
                        )}
                      </div>
                    );
                  })}
                </div>
              )}

              {/* Gap Questions */}
              {msg.gapPrompts && msg.gapPrompts.length > 0 && (
                <div className="w-full space-y-2">
                  {msg.gapPrompts.map((gap, i) => (
                    <div key={i} className="rounded-xl border border-brand-500/30 bg-brand-500/5 p-3 text-xs">
                      <div className="flex items-start gap-2 mb-2">
                        <HelpCircle className="w-3.5 h-3.5 text-brand-400 flex-shrink-0 mt-0.5" />
                        <div>
                          <span className="skill-pill-blue mb-1 inline-flex">{gap.skill}</span>
                          <p className="text-gray-300 mt-1">{gap.question}</p>
                        </div>
                      </div>
                      <div className="flex gap-2">
                        <button
                          id={`btn-gap-yes-${gap.skill}`}
                          onClick={() => onAnswerGap(gap.skill, 'yes')}
                          className="flex-1 py-1.5 rounded-lg bg-emerald-500/20 hover:bg-emerald-500/30 text-emerald-300 border border-emerald-500/30 transition-all font-medium"
                        >
                          Yes, I have it
                        </button>
                        <button
                          id={`btn-gap-no-${gap.skill}`}
                          onClick={() => onAnswerGap(gap.skill, 'no')}
                          className="flex-1 py-1.5 rounded-lg bg-surface-200 hover:bg-surface-300 text-gray-400 border border-white/5 transition-all"
                        >
                          No, I don't
                        </button>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>
        ))}

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

      {/* Input */}
      <div className="flex-shrink-0 p-3 border-t border-white/5">
        <div className="flex gap-2 items-end">
          <textarea
            ref={inputRef}
            id="chat-input"
            value={input}
            onChange={e => setInput(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder="Ask AI to improve your resume..."
            rows={2}
            disabled={loading}
            className="flex-1 input-field resize-none text-sm leading-relaxed min-h-[44px] max-h-32"
          />
          <button
            id="btn-send-chat"
            onClick={handleSend}
            disabled={!input.trim() || loading}
            className="btn-primary p-2.5 flex-shrink-0 disabled:opacity-40"
          >
            {loading
              ? <Loader2 className="w-4 h-4 animate-spin" />
              : <Send className="w-4 h-4" />
            }
          </button>
        </div>
        <p className="text-[10px] text-gray-600 mt-1.5 text-center">
          Press Enter to send · Shift+Enter for new line
        </p>
      </div>
    </div>
  );
};

export default ChatPanel;
