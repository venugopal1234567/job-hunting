import { useState, useCallback, useRef } from 'react';
import { ChatMessage, TrackedChange, ProposedEdit, GapQuestionPrompt } from '../types';
import { chatWithResume, saveResumeText, saveResumeVersion, revertResumeText } from '../services/api';

let msgCounter = 0;
const mkId = () => `msg_${++msgCounter}_${Date.now()}`;

export const useResumeEditor = (jobId?: string) => {
  const [editorContent, setEditorContent] = useState('');
  const [trackedChanges, setTrackedChanges] = useState<TrackedChange[]>([]);
  const [chatMessages, setChatMessages] = useState<ChatMessage[]>([]);
  const [isChatLoading, setIsChatLoading] = useState(false);
  const [isSaving, setIsSaving] = useState(false);
  const [isDirty, setIsDirty] = useState(false);
  const [lastSavedSkills, setLastSavedSkills] = useState<string[]>([]);
  const editorRef = useRef<HTMLTextAreaElement>(null);

  const initContent = useCallback((text: string, skills: string[]) => {
    const cleanText = text.replace(/[\u200b\u200c\u200d\ufeff]/g, '');
    setEditorContent(cleanText);
    setLastSavedSkills(skills);
    setIsDirty(false);
  }, []);

  const updateContent = useCallback((text: string) => {
    setEditorContent(text);
    setIsDirty(true);
  }, []);

  // Send a chat message to the AI
  const sendMessage = useCallback(async (userText: string, model?: string) => {
    if (!userText.trim() || isChatLoading) return;

    const userMsg: ChatMessage = {
      id: mkId(),
      role: 'user',
      content: userText,
      timestamp: Date.now(),
    };
    // Reset chat session and clear previous suggestions to start fresh
    setChatMessages([userMsg]);
    setTrackedChanges([]);
    setIsChatLoading(true);

    try {
      const response = await chatWithResume(userText, editorContent, jobId, model);

      // Map proposed edits → tracked changes
      const newChanges: TrackedChange[] = (response.proposed_edits || []).map((edit: ProposedEdit) => ({
        id: edit.id || `change_${Date.now()}_${Math.random()}`,
        original: edit.original,
        replacement: edit.replacement,
        reason: edit.reason,
        status: 'pending' as const,
        messageId: userMsg.id,
      }));

      if (newChanges.length > 0) {
        setTrackedChanges(prev => [...prev, ...newChanges]);
      }

      const aiMsg: ChatMessage = {
        id: mkId(),
        role: 'assistant',
        content: response.message,
        proposedEdits: response.proposed_edits?.map((e: ProposedEdit) => ({ ...e })),
        gapPrompts: response.gap_prompts?.map((g: GapQuestionPrompt) => ({ ...g })),
        fullResumeReplacement: response.full_resume_replacement,
        timestamp: Date.now(),
      };
      setChatMessages(prev => [...prev, aiMsg]);
    } catch (err: any) {
      const errMsg: ChatMessage = {
        id: mkId(),
        role: 'assistant',
        content: `⚠️ Error: ${err.response?.data?.error || err.message || 'Chat failed'}`,
        timestamp: Date.now(),
      };
      setChatMessages(prev => [...prev, errMsg]);
    } finally {
      setIsChatLoading(false);
    }
  }, [editorContent, isChatLoading, jobId]);

  // Answer a gap question — feed the answer back as a message
  const answerGapQuestion = useCallback((skill: string, answer: 'yes' | 'no' | string) => {
    const text = answer === 'yes'
      ? `Yes, I have experience with ${skill}. Please add it to my resume.`
      : answer === 'no'
      ? `No, I don't have experience with ${skill}.`
      : answer;
    sendMessage(text);
  }, [sendMessage]);

interface MatchResult {
  start: number;
  end: number;
}

const expandToWordBoundaries = (text: string, start: number, end: number): MatchResult => {
  let newStart = start;
  let newEnd = end;
  const isWordChar = (char: string) => /[a-zA-Z0-9_]/.test(char);

  if (start > 0 && isWordChar(text[start]) && isWordChar(text[start - 1])) {
    while (newStart > 0 && isWordChar(text[newStart - 1])) {
      newStart--;
    }
  }

  if (end < text.length && isWordChar(text[end - 1]) && isWordChar(text[end])) {
    while (newEnd < text.length && isWordChar(text[newEnd])) {
      newEnd++;
    }
  }

  return { start: newStart, end: newEnd };
};

const findFlexibleMatch = (text: string, pattern: string): MatchResult | null => {
  const normalize = (str: string) => str.replace(/[\s\u200b\u200c\u200d\ufeff]+/g, ' ').trim().toLowerCase();
  
  const cleanText = text.replace(/[\u200b\u200c\u200d\ufeff]/g, '');
  const normPattern = normalize(pattern);
  if (!normPattern) return null;

  const stripPrefix = (str: string) => str.replace(/^[•\-\*\s]+/, '').trim();
  const normPatternClean = stripPrefix(normPattern);
  if (!normPatternClean) return null;

  const words = normPatternClean.split(' ').filter(w => w !== '').map(w => w.replace(/[-\/\\^$*+?.()|[\]{}]/g, '\\$&'));
  if (words.length === 0) return null;

  try {
    const regexStr = '[•\\-\\*\\s]*' + words.join('[\\s\\r\\n\\W]*');
    const regex = new RegExp(regexStr, 'i');
    const match = cleanText.match(regex);
    if (match && match.index !== undefined) {
      return expandToWordBoundaries(cleanText, match.index, match.index + match[0].length);
    }
  } catch (e) {
    // Ignore regex errors
  }

  const cleanPattern = pattern.replace(/[\u200b\u200c\u200d\ufeff]/g, '');
  const idx = cleanText.indexOf(cleanPattern);
  if (idx !== -1) {
    return expandToWordBoundaries(cleanText, idx, idx + cleanPattern.length);
  }

  const cleanPatternNoPunct = cleanPattern.trim().replace(/[;,.:\s]+$/, '');
  const idx2 = cleanText.indexOf(cleanPatternNoPunct);
  if (idx2 !== -1) {
    return expandToWordBoundaries(cleanText, idx2, idx2 + cleanPatternNoPunct.length);
  }

  return null;
};

  // Accept a tracked change: apply it to editor content
  const acceptChange = useCallback((changeId: string) => {
    setTrackedChanges(prev => {
      const targetChange = prev.find(c => c.id === changeId);
      if (!targetChange || targetChange.status !== 'pending') return prev;

      const targetMatch = findFlexibleMatch(editorContent, targetChange.original);
      if (!targetMatch) {
        console.warn(`[AI Editor] Could not apply change. Original text not found: "${targetChange.original}"`);
        return prev;
      }

      // Apply the replacement in the editor content
      setEditorContent(current => {
        const replacementClean = targetChange.replacement.replace(/[\u200b\u200c\u200d\ufeff]/g, '');
        return current.slice(0, targetMatch.start) + replacementClean + current.slice(targetMatch.end);
      });

      setIsDirty(true);

      // Return updated trackedChanges: mark target as accepted, and reject any overlapping edits in the OLD content
      return prev.map(c => {
        if (c.id === changeId) {
          return { ...c, status: 'accepted' as const };
        }
        if (c.status === 'pending') {
          const otherMatch = findFlexibleMatch(editorContent, c.original);
          if (otherMatch) {
            const overlap = targetMatch.start < otherMatch.end && otherMatch.start < targetMatch.end;
            if (overlap) {
              console.warn(`[AI Editor] Discarding overlapping pending change: "${c.original}"`);
              return { ...c, status: 'rejected' as const };
            }
          }
        }
        return c;
      });
    });
  }, [editorContent]);

  // Reject a tracked change
  const rejectChange = useCallback((changeId: string) => {
    setTrackedChanges(prev => prev.map(c =>
      c.id === changeId ? { ...c, status: 'rejected' } : c
    ));
  }, []);

  // Auto-save editor content to backend
  const saveContent = useCallback(async (textOverride?: string) => {
    const textToSave = textOverride !== undefined ? textOverride : editorContent;
    if (!textToSave.trim() || isSaving) return null;
    setIsSaving(true);
    try {
      const result = await saveResumeText(textToSave);
      setLastSavedSkills(result.skills);
      setIsDirty(false);
      return result;
    } catch (err) {
      console.error('Save failed:', err);
      return null;
    } finally {
      setIsSaving(false);
    }
  }, [editorContent, isSaving]);

  // Save an "applied" version snapshot
  const saveAsApplied = useCallback(async (params: {
    jobId?: string;
    label: string;
    source: 'editor' | 'upload';
  }) => {
    // First save current edits
    await saveContent();
    return saveResumeVersion({
      snapshot_text: editorContent,
      job_id: params.jobId,
      label: params.label,
      source: params.source,
    });
  }, [editorContent, saveContent]);

  // Revert edited text back to original
  const revertContent = useCallback(async () => {
    setIsSaving(true);
    try {
      const result = await revertResumeText();
      setEditorContent(result.text);
      setLastSavedSkills(result.skills);
      setTrackedChanges([]); // Clear pending changes on revert
      setIsDirty(false);
      return result;
    } catch (err) {
      console.error('Revert failed:', err);
      return null;
    } finally {
      setIsSaving(false);
    }
  }, []);

  // Apply complete resume replacement
  const applyFullResume = useCallback((text: string) => {
    if (!text) return;
    const cleanText = text.replace(/[\u200b\u200c\u200d\ufeff]/g, '');
    setEditorContent(cleanText);
    setTrackedChanges([]); // Clear any line-by-line diffs since it's a full replacement
    setIsDirty(true);
  }, []);

  const pendingChanges = trackedChanges.filter(c => c.status === 'pending');

  return {
    editorContent,
    editorRef,
    trackedChanges,
    pendingChanges,
    chatMessages,
    isChatLoading,
    isSaving,
    isDirty,
    lastSavedSkills,
    initContent,
    updateContent,
    sendMessage,
    answerGapQuestion,
    acceptChange,
    rejectChange,
    saveContent,
    saveAsApplied,
    revertContent,
    setChatMessages,
    applyFullResume,
  };
};
