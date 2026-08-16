import React, { useEffect, useState, useRef, useCallback } from 'react';
import { Loader2, FileText } from 'lucide-react';
import { Job } from '../../types';
import { getResumeFullText, getActiveResume, analyzeJob, uploadResume } from '../../services/api';
import { useResumeEditor } from '../../hooks/useResumeEditor';
import ChatPanel from './ChatPanel';
import AppliedDialog from './AppliedDialog';
import ResumeUploader from './ResumeUploader';
import { formatResumeTextToHTML } from '../../utils/resumeRenderer';
import { ResumeEditorHeader } from './resume-editor/ResumeEditorHeader';
import { ResumeCanvasPane } from './resume-editor/ResumeCanvasPane';

interface ResumeEditorProps {
  selectedJob?: Job | null;
}

const ResumeEditor: React.FC<ResumeEditorProps> = ({ selectedJob }) => {
  const {
    editorContent,
    chatMessages,
    isChatLoading,
    isSaving,
    isDirty,
    initContent,
    updateContent,
    sendMessage,
    answerGapQuestion,
    saveContent,
    saveAsApplied,
    revertContent,
    setChatMessages,
    applyFullResume,
  } = useResumeEditor(selectedJob?.id);

  const [loadingResume, setLoadingResume] = useState(true);
  const [noResume, setNoResume] = useState(false);
  const [atsScore, setAtsScore] = useState<number | null>(null);
  const [prevAtsScore, setPrevAtsScore] = useState<number | null>(null);
  const [atsLoading, setAtsLoading] = useState(false);
  const [chatVisible, setChatVisible] = useState(true);
  const [showAppliedDialog, setShowAppliedDialog] = useState(false);
  const [saveMessage, setSaveMessage] = useState('');
  const [activeSubTab, setActiveSubTab] = useState<'editor' | 'pdf'>('editor');
  const [hasPDF, setHasPDF] = useState(false);
  const [activeModel, setActiveModel] = useState<string>('');
  const [fitToSinglePage, setFitToSinglePage] = useState(true);
  const [exportingPDF] = useState(false);

  const saveTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const autoSaveTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const uploadInputRef = useRef<HTMLInputElement>(null);
  const canvasParentRef = useRef<HTMLDivElement>(null);
  const [canvasScale, setCanvasScale] = useState<number>(1);

  // Responsive scaling observer for the Visual Resume Canvas sheet
  useEffect(() => {
    const parentEl = canvasParentRef.current;
    if (!parentEl) return;

    const updateScale = () => {
      const pagePixelWidth = 816;
      const containerWidth = parentEl.clientWidth - 48;
      if (containerWidth > 0 && containerWidth < pagePixelWidth) {
        setCanvasScale(containerWidth / pagePixelWidth);
      } else {
        setCanvasScale(1);
      }
    };

    updateScale();
    const observer = new ResizeObserver(updateScale);
    observer.observe(parentEl);
    window.addEventListener('resize', updateScale);

    return () => {
      observer.disconnect();
      window.removeEventListener('resize', updateScale);
    };
  }, []);

  // Load resume on mount
  useEffect(() => {
    const load = async () => {
      setLoadingResume(true);
      try {
        const [textData, meta] = await Promise.all([
          getResumeFullText(),
          getActiveResume(),
        ]);
        if (!textData) {
          setNoResume(true);
          return;
        }
        initContent(textData.text, meta?.extracted_skills || []);
        setHasPDF(!!meta?.has_pdf);
        setNoResume(false);
      } catch {
        setNoResume(true);
      } finally {
        setLoadingResume(false);
      }
    };
    load();
  }, [initContent]);

  // Isolate chat messages on job change (Do NOT auto-run ATS)
  useEffect(() => {
    setChatMessages([]);
  }, [selectedJob?.id, setChatMessages]);

  const handleSave = useCallback(async (silent = false) => {
    const result = await saveContent();
    if (result) {
      if (!silent) {
        setSaveMessage('Saved!');
        if (saveTimerRef.current) clearTimeout(saveTimerRef.current);
        saveTimerRef.current = setTimeout(() => setSaveMessage(''), 2000);
      }
    }
  }, [saveContent]);

  // Debounced auto-save on content change
  useEffect(() => {
    if (!isDirty || !editorContent) return;
    if (autoSaveTimerRef.current) clearTimeout(autoSaveTimerRef.current);
    autoSaveTimerRef.current = setTimeout(() => {
      handleSave(true);
    }, 3000);
    return () => {
      if (autoSaveTimerRef.current) clearTimeout(autoSaveTimerRef.current);
    };
  }, [editorContent, isDirty, handleSave]);

  // ATS calculation is ONLY run when manually triggered by the user
  const runATS = useCallback(async (force = true) => {
    if (!selectedJob) return;
    setAtsLoading(true);
    try {
      const result = await analyzeJob(selectedJob.id, undefined, force);
      setPrevAtsScore(atsScore);
      setAtsScore(result.ats_score);
    } catch {
      // silently ignore
    } finally {
      setAtsLoading(false);
    }
  }, [selectedJob, atsScore]);

  const handleRevert = useCallback(async () => {
    if (!window.confirm('Are you sure you want to revert all changes? This will restore the original resume text.')) {
      return;
    }
    const result = await revertContent();
    if (result) {
      setChatMessages([]);
      setSaveMessage('Reverted to Original!');
      if (saveTimerRef.current) clearTimeout(saveTimerRef.current);
      saveTimerRef.current = setTimeout(() => setSaveMessage(''), 2500);
    }
  }, [revertContent, setChatMessages]);

  const handleApplyFullResume = useCallback(async (text: string) => {
    applyFullResume(text);
    await saveContent(text);
  }, [applyFullResume, saveContent]);

  const handleExportPDF = () => {
    const fullHTML = formatResumeTextToHTML(editorContent, fitToSinglePage);

    const printFrame = document.createElement('iframe');
    printFrame.style.position = 'fixed';
    printFrame.style.right = '0';
    printFrame.style.bottom = '0';
    printFrame.style.width = '0';
    printFrame.style.height = '0';
    printFrame.style.border = '0';
    document.body.appendChild(printFrame);

    const frameDoc = printFrame.contentWindow?.document;
    if (frameDoc) {
      frameDoc.open();
      frameDoc.write(fullHTML);
      frameDoc.close();
      setTimeout(() => {
        printFrame.contentWindow?.focus();
        printFrame.contentWindow?.print();
        setTimeout(() => {
          document.body.removeChild(printFrame);
        }, 1000);
      }, 250);
    }
  };

  const handleResumeUploaded = async () => {
    setLoadingResume(true);
    try {
      const [textData, meta] = await Promise.all([getResumeFullText(), getActiveResume()]);
      if (textData) {
        initContent(textData.text, meta?.extracted_skills || []);
        setHasPDF(!!meta?.has_pdf);
        setNoResume(false);
      }
    } finally {
      setLoadingResume(false);
    }
  };

  const handleDirectUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    setLoadingResume(true);
    try {
      await uploadResume(file);
      await handleResumeUploaded();
    } catch (err) {
      console.error(err);
      alert('Upload failed: ' + (err instanceof Error ? err.message : String(err)));
    } finally {
      setLoadingResume(false);
      e.target.value = '';
    }
  };

  if (loadingResume) {
    return (
      <div className="flex items-center justify-center h-full min-h-[400px] gap-3">
        <Loader2 className="w-6 h-6 text-brand-400 animate-spin" />
        <p className="text-sm text-gray-400">Loading resume...</p>
      </div>
    );
  }

  if (noResume) {
    return (
      <div className="max-w-xl mx-auto py-10">
        <div className="text-center mb-8">
          <div className="w-16 h-16 rounded-2xl bg-surface-200 border border-white/5 flex items-center justify-center mx-auto mb-4">
            <FileText className="w-8 h-8 text-gray-600" />
          </div>
          <h2 className="text-lg font-bold text-white">Upload Your Resume</h2>
          <p className="text-sm text-gray-400 mt-2">
            Upload your resume to start using the AI editor.
          </p>
        </div>
        <div className="glass rounded-xl p-6">
          <ResumeUploader />
        </div>
      </div>
    );
  }

  return (
    <div className="w-full">
      <input
        type="file"
        ref={uploadInputRef}
        style={{ display: 'none' }}
        accept=".pdf,.txt"
        onChange={handleDirectUpload}
      />

      <ResumeEditorHeader
        selectedJob={selectedJob}
        atsScore={atsScore}
        prevAtsScore={prevAtsScore}
        atsLoading={atsLoading}
        onReanalyzeATS={runATS}
        saveMessage={saveMessage}
        activeSubTab={activeSubTab}
        onSubTabChange={setActiveSubTab}
        fitToSinglePage={fitToSinglePage}
        onToggleFitToSinglePage={() => setFitToSinglePage(!fitToSinglePage)}
        isSaving={isSaving}
        onRevert={handleRevert}
        exportingPDF={exportingPDF}
        onExportPDF={handleExportPDF}
        chatVisible={chatVisible}
        onToggleChat={() => setChatVisible(!chatVisible)}
        onUploadClick={() => uploadInputRef.current?.click()}
      />

      {/* 50/50 Split Workspace Grid */}
      <div className={`grid grid-cols-1 ${chatVisible ? 'lg:grid-cols-2' : 'lg:grid-cols-1'} gap-4 h-[calc(100vh-200px)] min-h-[550px] w-full no-print`}>
        {/* AI Resume Coach Sidebar Panel */}
        {chatVisible && (
          <div className="min-w-0 h-full glass rounded-xl border border-white/10 overflow-hidden flex flex-col no-print shadow-xl">
            <ChatPanel
              messages={chatMessages}
              loading={isChatLoading}
              onSend={(text) => sendMessage(text, activeModel)}
              onAnswerGap={answerGapQuestion}
              jobTitle={selectedJob?.title}
              activeModel={activeModel}
              onModelChange={setActiveModel}
              onApplyFullResume={handleApplyFullResume}
            />
          </div>
        )}

        {/* Visual Resume Canvas / Original PDF Viewer */}
        <ResumeCanvasPane
          activeSubTab={activeSubTab}
          fitToSinglePage={fitToSinglePage}
          editorContent={editorContent}
          onUpdateContent={updateContent}
          canvasParentRef={canvasParentRef}
          canvasScale={canvasScale}
          hasPDF={hasPDF}
          onUploadClick={() => uploadInputRef.current?.click()}
        />
      </div>

      {/* Applied Dialog */}
      <AppliedDialog
        isOpen={showAppliedDialog}
        onClose={() => setShowAppliedDialog(false)}
        currentText={editorContent}
        selectedJob={selectedJob}
        onSaveVersion={saveAsApplied}
        onResumeUploaded={handleResumeUploaded}
      />
    </div>
  );
};

export default ResumeEditor;
