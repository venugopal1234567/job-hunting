import React from 'react';
import { RotateCcw, Download, PanelRightClose, PanelRight, CheckCircle2, Loader2, Sparkles } from 'lucide-react';
import { Job } from '../../../types';
import ATSScoreBar from '../ATSScoreBar';

interface ResumeEditorHeaderProps {
  selectedJob?: Job | null;
  atsScore: number | null;
  prevAtsScore: number | null;
  atsLoading: boolean;
  onReanalyzeATS?: () => void;
  saveMessage: string;
  activeSubTab: 'editor' | 'pdf';
  onSubTabChange: (tab: 'editor' | 'pdf') => void;
  fitToSinglePage: boolean;
  onToggleFitToSinglePage: () => void;
  isSaving: boolean;
  isReverting: boolean;
  onRevert: () => void;
  exportingPDF: boolean;
  onExportPDF: () => void;
  chatVisible: boolean;
  onToggleChat: () => void;
  onUploadClick: () => void;
  onConvertToNewLayout?: () => void;
  isConvertingLayout?: boolean;
}

export const ResumeEditorHeader: React.FC<ResumeEditorHeaderProps> = ({
  selectedJob,
  atsScore,
  prevAtsScore,
  atsLoading,
  onReanalyzeATS,
  saveMessage,
  activeSubTab,
  onSubTabChange,
  isSaving,
  isReverting,
  onRevert,
  exportingPDF,
  onExportPDF,
  chatVisible,
  onToggleChat,
  isConvertingLayout = false,
}) => {
  return (
    <div className="bg-surface-container-lowest border border-surface-variant rounded-2xl p-4 mb-4 no-print shadow-elevation-1">
      <div className="flex flex-wrap items-center justify-between gap-4">
        {/* ATS Score Bar */}
        <div className="flex-1 min-w-[260px]">
          <ATSScoreBar
            score={atsScore}
            previousScore={prevAtsScore}
            loading={atsLoading}
            jobTitle={selectedJob?.title}
            onReanalyze={selectedJob ? onReanalyzeATS : undefined}
          />
        </div>

        <div className="flex items-center gap-2.5 flex-wrap">
          {/* Status indicator */}
          {saveMessage && (
            <span className="flex items-center gap-1 text-xs text-primary font-bold animate-fade-in mr-1">
              <CheckCircle2 className="w-4 h-4 text-emerald-500" /> {saveMessage}
            </span>
          )}

          {/* Sub-tabs */}
          <div className="flex items-center bg-surface-container p-1 rounded-full border border-surface-variant">
            <button
              onClick={() => onSubTabChange('editor')}
              className={`px-3.5 py-1.5 rounded-full text-xs font-semibold flex items-center gap-1.5 transition-all ${
                activeSubTab === 'editor'
                  ? 'bg-primary text-on-primary shadow-sm'
                  : 'text-on-surface-variant hover:text-primary'
              }`}
            >
              <span>✏️ Visual Canvas</span>
            </button>
            <button
              onClick={() => onSubTabChange('pdf')}
              className={`px-3.5 py-1.5 rounded-full text-xs font-semibold flex items-center gap-1.5 transition-all ${
                activeSubTab === 'pdf'
                  ? 'bg-primary text-on-primary shadow-sm'
                  : 'text-on-surface-variant hover:text-primary'
              }`}
            >
              <span>📄 Original PDF</span>
            </button>
          </div>

          {/* Revert button */}
          <button
            id="btn-revert-resume"
            onClick={onRevert}
            disabled={isSaving || isReverting || isConvertingLayout}
            className="px-3.5 py-1.5 rounded-full bg-secondary-fixed text-on-secondary-fixed hover:bg-secondary-fixed-dim border border-secondary-fixed-dim/40 text-xs font-semibold flex items-center gap-1.5 disabled:opacity-40 transition-all active:scale-95"
            title="Revert all edits back to original resume text"
          >
            {isReverting || isConvertingLayout ? (
              <Loader2 className="w-3.5 h-3.5 animate-spin text-secondary" />
            ) : (
              <RotateCcw className="w-3.5 h-3.5" />
            )}
            {isReverting ? 'Reverting...' : isConvertingLayout ? 'Converting...' : 'Revert Original'}
          </button>

          {/* Export to PDF */}
          <button
            id="btn-export-pdf"
            onClick={onExportPDF}
            disabled={exportingPDF}
            className="btn-primary text-xs py-1.5 px-4 flex items-center gap-1.5"
            title="Export clean PDF file"
          >
            {exportingPDF ? (
              <Loader2 className="w-3.5 h-3.5 animate-spin text-white" />
            ) : (
              <Download className="w-3.5 h-3.5 text-white" />
            )}
            Export to PDF
          </button>

          {/* AI Resume Coach Toggle */}
          <button
            id="btn-toggle-chat"
            onClick={onToggleChat}
            className={`py-1.5 px-3.5 rounded-full text-xs font-semibold border flex items-center gap-1.5 transition-all ${
              chatVisible
                ? 'bg-primary-fixed text-on-primary-fixed border-primary-fixed-dim'
                : 'bg-surface-container text-on-surface-variant border-surface-variant hover:bg-surface-container-high'
            }`}
            title={chatVisible ? 'Hide AI Resume Coach' : 'Show AI Resume Coach'}
          >
            {chatVisible ? <PanelRightClose className="w-4 h-4" /> : <PanelRight className="w-4 h-4" />}
            <span>AI Coach</span>
          </button>
        </div>
      </div>
    </div>
  );
};

