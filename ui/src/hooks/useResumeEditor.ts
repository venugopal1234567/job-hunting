import { useState, useCallback, useRef } from 'react';
import { ChatMessage, ProposedEdit, GapQuestionPrompt } from '../types';
import { chatWithResume, saveResumeText, saveResumeVersion, revertResumeText } from '../services/api';

let msgCounter = 0;
const mkId = () => `msg_${++msgCounter}_${Date.now()}`;

export const useResumeEditor = (jobId?: string) => {
  const [editorContent, setEditorContent] = useState('');
  const [chatMessages, setChatMessages] = useState<ChatMessage[]>([]);
  const [isChatLoading, setIsChatLoading] = useState(false);
  const [isSaving, setIsSaving] = useState(false);
  const [isReverting, setIsReverting] = useState(false);
  const [isDirty, setIsDirty] = useState(false);
  const [lastSavedSkills, setLastSavedSkills] = useState<string[]>([]);
  const editorRef = useRef<HTMLTextAreaElement>(null);
  const lastAIHTMLRef = useRef<string>('');

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
    setChatMessages([userMsg]);
    setIsChatLoading(true);

    try {
      const response = await chatWithResume(userText, editorContent, jobId, model);

      if (response.html) {
        lastAIHTMLRef.current = response.html;
        setEditorContent(response.html);
        setIsDirty(true);
      } else if (response.full_resume_replacement) {
        const cleanText = response.full_resume_replacement.replace(/[\u200b\u200c\u200d\ufeff]/g, '');
        setEditorContent(cleanText);
        setIsDirty(true);
      }

      const aiMsg: ChatMessage = {
        id: mkId(),
        role: 'assistant',
        content: response.message,
        proposedEdits: response.proposed_edits?.map((e: ProposedEdit) => ({ ...e })),
        gapPrompts: response.gap_prompts?.map((g: GapQuestionPrompt) => ({ ...g })),
        fullResumeReplacement: response.html || response.full_resume_replacement,
        structuredResume: response.structured_resume,
        html: response.html,
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

  // Auto-save editor content to backend
  const saveContent = useCallback(async (textOverride?: string) => {
    const textToSave = textOverride !== undefined ? textOverride : editorContent;
    if (!textToSave.trim() || isSaving) return null;
    setIsSaving(true);
    const htmlToSave = lastAIHTMLRef.current || undefined;
    lastAIHTMLRef.current = '';
    try {
      const result = await saveResumeText(textToSave, htmlToSave);
      setLastSavedSkills(result.skills);
      setIsDirty(false);
      return result;
    } catch (err) {
      if (htmlToSave) lastAIHTMLRef.current = htmlToSave;
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
    setIsReverting(true);
    try {
      const result = await revertResumeText();
      const textToUse = result.html || result.text;
      setEditorContent(textToUse);
      setLastSavedSkills(result.skills);
      setIsDirty(false);
      return result;
    } catch (err) {
      console.error('Revert failed:', err);
      return null;
    } finally {
      setIsReverting(false);
    }
  }, []);

  // Apply complete resume replacement
  const applyFullResume = useCallback((text: string) => {
    if (!text) return;
    const cleanText = text.replace(/[\u200b\u200c\u200d\ufeff]/g, '');
    setEditorContent(cleanText);
    setIsDirty(true);
  }, []);

  return {
    editorContent,
    editorRef,
    chatMessages,
    isChatLoading,
    isSaving,
    isReverting,
    isDirty,
    lastSavedSkills,
    initContent,
    updateContent,
    sendMessage,
    answerGapQuestion,
    saveContent,
    saveAsApplied,
    revertContent,
    applyFullResume,
    setChatMessages,
  };
};
