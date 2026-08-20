import React, { useState } from 'react';
import {
  Save, Upload, Clock, CheckCircle2, X, Briefcase,
  AlertTriangle, FileUp
} from 'lucide-react';
import { Job, StructuredResume } from '../../types';
import { uploadResume, saveResumeVersion } from '../../services/api';

interface AppliedDialogProps {
  isOpen: boolean;
  onClose: () => void;
  canvasStructured: StructuredResume | null;
  selectedJob?: Job | null;
  onSaveVersion: (params: { jobId?: string; label: string; source: 'editor' | 'upload' }) => Promise<void>;
  onResumeUploaded?: () => void;
}

const AppliedDialog: React.FC<AppliedDialogProps> = ({
  isOpen,
  onClose,
  canvasStructured,
  selectedJob,
  onSaveVersion,
  onResumeUploaded,
}) => {
  const [step, setStep] = useState<'ask' | 'confirm' | 'upload' | 'done'>('ask');
  const [label, setLabel] = useState(
    selectedJob ? `Applied to ${selectedJob.title} @ ${selectedJob.company}` : 'Applied resume'
  );
  const [loading, setLoading] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [dragOver, setDragOver] = useState(false);

  if (!isOpen) return null;

  const handleSaveEditor = async () => {
    setLoading(true);
    try {
      await onSaveVersion({
        jobId: selectedJob?.id,
        label: label || 'Applied resume',
        source: 'editor',
      });
      setStep('done');
    } catch (e) {
      console.error(e);
    } finally {
      setLoading(false);
    }
  };

  const handleUploadApplied = async (file: File) => {
    if (!canvasStructured) return;
    setUploading(true);
    try {
      // Upload as new active resume
      await uploadResume(file);
      // Then save a version snapshot for that file
      await saveResumeVersion({
        snapshot_structured: canvasStructured,
        label: `${file.name} – applied to ${selectedJob?.title || 'job'}`,
        source: 'upload',
      });
      onResumeUploaded?.();
      setStep('done');
    } catch (e) {
      console.error(e);
    } finally {
      setUploading(false);
    }
  };

  const handleFileDrop = (e: React.DragEvent) => {
    e.preventDefault();
    setDragOver(false);
    const file = e.dataTransfer.files[0];
    if (file) handleUploadApplied(file);
  };

  const handleFileInput = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (file) handleUploadApplied(file);
    e.target.value = '';
  };

  return (
    <div
      className="fixed inset-0 z-[100] flex items-center justify-center bg-inverse-surface/40 backdrop-blur-sm"
      onClick={e => e.target === e.currentTarget && onClose()}
    >
      <div className="w-full max-w-md mx-4 bg-surface-container-lowest rounded-3xl border border-surface-variant shadow-elevation-3 overflow-hidden animate-fade-in">
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-surface-variant bg-surface-container-low/50">
          <div className="flex items-center gap-2.5">
            <div className="w-8 h-8 rounded-full bg-primary-fixed text-primary flex items-center justify-center">
              <Save className="w-4 h-4" />
            </div>
            <p className="text-sm font-bold font-headline text-on-surface">Save Resume Snapshot</p>
          </div>
          <button onClick={onClose} className="p-1.5 text-on-surface-variant hover:text-primary rounded-full hover:bg-surface-container transition-colors">
            <X className="w-4 h-4" />
          </button>
        </div>

        <div className="p-6 space-y-4">
          {/* Step: Ask */}
          {step === 'ask' && (
            <>
              <div className="flex items-start gap-3 p-3.5 rounded-2xl bg-secondary-fixed/40 border border-secondary-fixed-dim/40">
                <AlertTriangle className="w-4 h-4 text-secondary flex-shrink-0 mt-0.5" />
                <div>
                  <p className="text-sm font-bold text-on-surface">Did you apply with this resume?</p>
                  {selectedJob && (
                    <p className="text-xs text-on-surface-variant mt-1">
                      For: <span className="text-primary font-semibold">{selectedJob.title}</span> at {selectedJob.company}
                    </p>
                  )}
                </div>
              </div>

              <div className="space-y-2.5">
                <button
                  id="btn-applied-yes-editor"
                  onClick={() => setStep('confirm')}
                  className="w-full flex items-center gap-3.5 p-4 rounded-2xl bg-surface-container-low border border-surface-variant hover:border-primary/40 hover:bg-surface-container transition-all text-left group"
                >
                  <CheckCircle2 className="w-5 h-5 text-emerald-500 flex-shrink-0" />
                  <div>
                    <p className="text-sm font-bold text-on-surface group-hover:text-primary transition-colors">Yes, save this edited version</p>
                    <p className="text-xs text-on-surface-variant">Archive the resume you typed here</p>
                  </div>
                </button>

                <button
                  id="btn-applied-upload"
                  onClick={() => setStep('upload')}
                  className="w-full flex items-center gap-3.5 p-4 rounded-2xl bg-surface-container-low border border-surface-variant hover:border-primary/40 hover:bg-surface-container transition-all text-left group"
                >
                  <FileUp className="w-5 h-5 text-primary flex-shrink-0" />
                  <div>
                    <p className="text-sm font-bold text-on-surface group-hover:text-primary transition-colors">Applied from external source</p>
                    <p className="text-xs text-on-surface-variant">Upload the actual PDF/TXT you submitted</p>
                  </div>
                </button>

                <button
                  id="btn-applied-no"
                  onClick={onClose}
                  className="w-full flex items-center gap-3.5 p-3 rounded-2xl hover:bg-surface-container transition-all text-left"
                >
                  <Clock className="w-5 h-5 text-outline flex-shrink-0" />
                  <div>
                    <p className="text-xs font-semibold text-on-surface-variant">Not yet, just saving progress</p>
                  </div>
                </button>
              </div>
            </>
          )}

          {/* Step: Confirm editor version */}
          {step === 'confirm' && (
            <>
              <p className="text-xs font-semibold text-on-surface-variant">Label this snapshot so you can find it later:</p>
              <input
                id="input-version-label"
                type="text"
                value={label}
                onChange={e => setLabel(e.target.value)}
                className="input-field w-full"
                placeholder="e.g. Applied to Google SRE role"
              />
              {selectedJob && (
                <div className="flex items-center gap-2.5 p-3 rounded-2xl bg-surface-container-low border border-surface-variant">
                  <Briefcase className="w-4 h-4 text-on-surface-variant" />
                  <div>
                    <p className="text-xs font-bold text-on-surface">{selectedJob.title}</p>
                    <p className="text-[10px] text-on-surface-variant">{selectedJob.company} · {selectedJob.source_board}</p>
                  </div>
                </div>
              )}
              <div className="flex gap-2.5 pt-1">
                <button onClick={() => setStep('ask')} className="btn-outline flex-1 text-xs">
                  Back
                </button>
                <button
                  id="btn-confirm-save-version"
                  onClick={handleSaveEditor}
                  disabled={loading}
                  className="btn-primary flex-1 text-xs flex items-center justify-center gap-2"
                >
                  {loading ? (
                    <><span className="animate-spin border-2 border-white/30 border-t-white rounded-full w-3.5 h-3.5" /> Saving...</>
                  ) : (
                    <><Save className="w-3.5 h-3.5" /> Save Snapshot</>
                  )}
                </button>
              </div>
            </>
          )}

          {/* Step: Upload applied resume */}
          {step === 'upload' && (
            <>
              <p className="text-xs font-semibold text-on-surface-variant">Upload the resume you actually submitted:</p>
              <label
                htmlFor="applied-resume-upload"
                className={`block border-2 border-dashed rounded-2xl p-6 text-center cursor-pointer transition-all ${
                  dragOver ? 'border-primary bg-primary-fixed/20' : 'border-surface-variant bg-surface-container-low hover:border-primary/40'
                }`}
                onDragOver={e => { e.preventDefault(); setDragOver(true); }}
                onDragLeave={() => setDragOver(false)}
                onDrop={handleFileDrop}
              >
                <input
                  id="applied-resume-upload"
                  type="file"
                  accept=".pdf,.txt"
                  className="hidden"
                  onChange={handleFileInput}
                />
                {uploading ? (
                  <div className="flex flex-col items-center gap-2">
                    <span className="animate-spin border-2 border-primary/30 border-t-primary rounded-full w-8 h-8 inline-block" />
                    <p className="text-xs text-on-surface-variant">Uploading and saving...</p>
                  </div>
                ) : (
                  <div className="flex flex-col items-center gap-2">
                    <div className="w-10 h-10 rounded-full bg-surface-container flex items-center justify-center text-on-surface-variant">
                      <Upload className="w-5 h-5" />
                    </div>
                    <p className="text-xs font-semibold text-on-surface">Drop your applied resume here</p>
                    <p className="text-[10px] text-on-surface-variant">Supports .PDF and .TXT</p>
                  </div>
                )}
              </label>
              <button onClick={() => setStep('ask')} className="btn-outline w-full text-xs">
                Back
              </button>
            </>
          )}

          {/* Step: Done */}
          {step === 'done' && (
            <div className="flex flex-col items-center gap-4 py-4">
              <div className="w-14 h-14 rounded-full bg-emerald-500/10 text-emerald-500 flex items-center justify-center">
                <CheckCircle2 className="w-8 h-8" />
              </div>
              <div className="text-center">
                <p className="text-base font-bold font-headline text-on-surface">Snapshot Saved!</p>
                <p className="text-xs text-on-surface-variant mt-1">
                  Find it in your version history. Good luck! 🚀
                </p>
              </div>
              <button id="btn-close-applied-dialog" onClick={onClose} className="btn-primary text-xs px-8">
                Done
              </button>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

export default AppliedDialog;

