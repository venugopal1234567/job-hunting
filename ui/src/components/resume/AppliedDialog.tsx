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
      className="fixed inset-0 z-[100] flex items-center justify-center bg-black/70 backdrop-blur-sm"
      onClick={e => e.target === e.currentTarget && onClose()}
    >
      <div className="w-full max-w-md mx-4 glass rounded-2xl border border-white/10 shadow-2xl overflow-hidden animate-fade-in">
        {/* Header */}
        <div className="flex items-center justify-between px-5 py-4 border-b border-white/5">
          <div className="flex items-center gap-2">
            <div className="w-8 h-8 rounded-lg bg-brand-500/20 border border-brand-500/30 flex items-center justify-center">
              <Save className="w-4 h-4 text-brand-400" />
            </div>
            <p className="text-sm font-semibold text-white">Save Resume Snapshot</p>
          </div>
          <button onClick={onClose} className="btn-ghost p-1.5">
            <X className="w-4 h-4" />
          </button>
        </div>

        <div className="p-5 space-y-4">
          {/* Step: Ask */}
          {step === 'ask' && (
            <>
              <div className="flex items-start gap-3 p-3 rounded-xl bg-amber-500/10 border border-amber-500/20">
                <AlertTriangle className="w-4 h-4 text-amber-400 flex-shrink-0 mt-0.5" />
                <div>
                  <p className="text-sm font-medium text-white">Did you apply with this resume?</p>
                  {selectedJob && (
                    <p className="text-xs text-gray-400 mt-1">
                      For: <span className="text-brand-300">{selectedJob.title}</span> at {selectedJob.company}
                    </p>
                  )}
                </div>
              </div>

              <div className="space-y-2">
                <button
                  id="btn-applied-yes-editor"
                  onClick={() => setStep('confirm')}
                  className="w-full flex items-center gap-3 p-4 rounded-xl bg-emerald-500/10 border border-emerald-500/20 hover:bg-emerald-500/15 transition-all text-left"
                >
                  <CheckCircle2 className="w-5 h-5 text-emerald-400 flex-shrink-0" />
                  <div>
                    <p className="text-sm font-medium text-white">Yes, save this edited version</p>
                    <p className="text-xs text-gray-500">Archive the resume you typed here</p>
                  </div>
                </button>

                <button
                  id="btn-applied-upload"
                  onClick={() => setStep('upload')}
                  className="w-full flex items-center gap-3 p-4 rounded-xl bg-brand-500/10 border border-brand-500/20 hover:bg-brand-500/15 transition-all text-left"
                >
                  <FileUp className="w-5 h-5 text-brand-400 flex-shrink-0" />
                  <div>
                    <p className="text-sm font-medium text-white">Applied from external source</p>
                    <p className="text-xs text-gray-500">Upload the actual PDF/TXT you submitted</p>
                  </div>
                </button>

                <button
                  id="btn-applied-no"
                  onClick={onClose}
                  className="w-full flex items-center gap-3 p-3 rounded-xl hover:bg-surface-200 transition-all text-left"
                >
                  <Clock className="w-5 h-5 text-gray-500 flex-shrink-0" />
                  <div>
                    <p className="text-sm text-gray-400">Not yet, just saving progress</p>
                  </div>
                </button>
              </div>
            </>
          )}

          {/* Step: Confirm editor version */}
          {step === 'confirm' && (
            <>
              <p className="text-sm text-gray-300">Label this snapshot so you can find it later:</p>
              <input
                id="input-version-label"
                type="text"
                value={label}
                onChange={e => setLabel(e.target.value)}
                className="input-field w-full"
                placeholder="e.g. Applied to Google SRE role"
              />
              {selectedJob && (
                <div className="flex items-center gap-2 p-3 rounded-xl bg-surface-200 border border-white/5">
                  <Briefcase className="w-4 h-4 text-gray-500" />
                  <div>
                    <p className="text-xs text-gray-300">{selectedJob.title}</p>
                    <p className="text-[10px] text-gray-500">{selectedJob.company} · {selectedJob.source_board}</p>
                  </div>
                </div>
              )}
              <div className="flex gap-2">
                <button onClick={() => setStep('ask')} className="btn-ghost flex-1 text-sm">
                  Back
                </button>
                <button
                  id="btn-confirm-save-version"
                  onClick={handleSaveEditor}
                  disabled={loading}
                  className="btn-primary flex-1 text-sm flex items-center justify-center gap-2"
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
              <p className="text-sm text-gray-300">Upload the resume you actually submitted:</p>
              <label
                htmlFor="applied-resume-upload"
                className={`block border-2 border-dashed rounded-xl p-6 text-center cursor-pointer transition-all ${
                  dragOver ? 'border-brand-500 bg-brand-500/10' : 'border-surface-400 hover:border-surface-300'
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
                    <span className="animate-spin border-2 border-brand-400/30 border-t-brand-400 rounded-full w-8 h-8 inline-block" />
                    <p className="text-sm text-gray-400">Uploading and saving...</p>
                  </div>
                ) : (
                  <div className="flex flex-col items-center gap-2">
                    <Upload className="w-8 h-8 text-gray-500" />
                    <p className="text-sm text-gray-300 font-medium">Drop your applied resume here</p>
                    <p className="text-xs text-gray-600">Supports .PDF and .TXT</p>
                  </div>
                )}
              </label>
              <button onClick={() => setStep('ask')} className="btn-ghost w-full text-sm">
                Back
              </button>
            </>
          )}

          {/* Step: Done */}
          {step === 'done' && (
            <div className="flex flex-col items-center gap-4 py-4">
              <div className="w-14 h-14 rounded-2xl bg-emerald-500/10 border border-emerald-500/20 flex items-center justify-center">
                <CheckCircle2 className="w-7 h-7 text-emerald-400" />
              </div>
              <div className="text-center">
                <p className="text-sm font-semibold text-white">Snapshot Saved!</p>
                <p className="text-xs text-gray-500 mt-1">
                  Find it in your version history. Good luck! 🚀
                </p>
              </div>
              <button id="btn-close-applied-dialog" onClick={onClose} className="btn-primary text-sm px-6">
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
