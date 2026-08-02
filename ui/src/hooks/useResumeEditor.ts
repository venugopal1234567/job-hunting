import { useState, useCallback, useRef } from 'react';
import { ChatMessage, TrackedChange, ProposedEdit, GapQuestionPrompt } from '../types';
import { chatWithResume, saveResumeText, saveResumeVersion } from '../services/api';

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
  const sendMessage = useCallback(async (userText: string) => {
    if (!userText.trim() || isChatLoading) return;

    const userMsg: ChatMessage = {
      id: mkId(),
      role: 'user',
      content: userText,
      timestamp: Date.now(),
    };
    setChatMessages(prev => [...prev, userMsg]);
    setIsChatLoading(true);

    try {
      const response = await chatWithResume(userText, editorContent, jobId);

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

  // Accept a tracked change: apply it to editor content
  const acceptChange = useCallback((changeId: string) => {
    setTrackedChanges(prev => prev.map(c => {
      if (c.id !== changeId) return c;
      
      const originalClean = c.original.replace(/[\u200b\u200c\u200d\ufeff]/g, '');
      const replacementClean = c.replacement.replace(/[\u200b\u200c\u200d\ufeff]/g, '');

      // Apply the replacement in the editor content with robust match options
      setEditorContent(current => {
        const cleanCurrent = current.replace(/[\u200b\u200c\u200d\ufeff]/g, '');

        // Try exact match first
        let idx = cleanCurrent.indexOf(originalClean);
        if (idx !== -1) {
          return cleanCurrent.slice(0, idx) + replacementClean + cleanCurrent.slice(idx + originalClean.length);
        }

        // Try stripping trailing punctuation/spacing
        const originalNoPunct = originalClean.trim().replace(/[;,.:\s]+$/, '');
        idx = cleanCurrent.indexOf(originalNoPunct);
        if (idx !== -1) {
          let matchLength = originalNoPunct.length;
          const nextChar = cleanCurrent[idx + matchLength];
          if (nextChar && /[;,.:\s]/.test(nextChar)) {
            matchLength++;
          }
          return cleanCurrent.slice(0, idx) + replacementClean + cleanCurrent.slice(idx + matchLength);
        }

        console.warn(`[AI Editor] Could not apply change. Original text not found: "${c.original}"`);
        return current;
      });
      setIsDirty(true);
      return { ...c, status: 'accepted' };
    }));
  }, []);

  // Reject a tracked change
  const rejectChange = useCallback((changeId: string) => {
    setTrackedChanges(prev => prev.map(c =>
      c.id === changeId ? { ...c, status: 'rejected' } : c
    ));
  }, []);

  // Auto-save editor content to backend
  const saveContent = useCallback(async () => {
    if (!editorContent.trim() || isSaving) return null;
    setIsSaving(true);
    try {
      const result = await saveResumeText(editorContent);
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
    setChatMessages,
  };
};
