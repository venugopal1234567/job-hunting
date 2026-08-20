import React, { useCallback, useState } from 'react';
import { Upload, FileText, CheckCircle2, AlertCircle, Loader2 } from 'lucide-react';
import { useResume } from '../../hooks/useResume';

interface ResumeUploaderProps {
  onResumeUploaded?: () => void;
}

const ResumeUploader: React.FC<ResumeUploaderProps> = ({ onResumeUploaded }) => {
  const { resume, loading, uploading, error, upload } = useResume();
  const [dragOver, setDragOver] = useState(false);
  const [uploadSuccess, setUploadSuccess] = useState(false);

  const handleFile = async (file: File) => {
    setUploadSuccess(false);
    try {
      await upload(file);
      setUploadSuccess(true);
      if (onResumeUploaded) {
        onResumeUploaded();
      }
      setTimeout(() => setUploadSuccess(false), 3000);
    } catch {}
  };

  const onDrop = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    setDragOver(false);
    const file = e.dataTransfer.files[0];
    if (file) handleFile(file);
  }, []);

  const onInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (file) handleFile(file);
    e.target.value = '';
  };

  return (
    <div className="space-y-5">
      {/* Current resume */}
      {!loading && resume && (
        <div className="bg-surface-container-lowest rounded-2xl p-4 border border-surface-variant shadow-elevation-1 animate-fade-in">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-full bg-primary-fixed text-primary flex items-center justify-center">
              <FileText className="w-5 h-5" />
            </div>
            <div className="flex-1">
              <p className="text-sm font-bold font-headline text-on-surface">{resume.filename}</p>
              <p className="text-xs text-on-surface-variant mt-0.5 font-medium">
                {resume.raw_text_length.toLocaleString()} chars · {resume.extracted_skills?.length || 0} skills detected
              </p>
            </div>
            <CheckCircle2 className="w-5 h-5 text-emerald-500 flex-shrink-0" />
          </div>

          {/* Extracted Skills */}
          {resume.extracted_skills && resume.extracted_skills.length > 0 && (
            <div className="mt-4 pt-4 border-t border-surface-variant">
              <p className="text-[11px] font-bold uppercase tracking-wider text-on-surface-variant mb-2">Extracted Skills</p>
              <div className="flex flex-wrap gap-1.5">
                {resume.extracted_skills.map(skill => (
                  <span key={skill} className="skill-pill-green">{skill}</span>
                ))}
              </div>
            </div>
          )}
        </div>
      )}

      {/* Upload area */}
      <div>
        <p className="text-[11px] font-bold uppercase tracking-wider text-on-surface-variant mb-2">{resume ? 'Replace Resume' : 'Upload Resume'}</p>
        <label
          id="resume-upload-dropzone"
          htmlFor="resume-file-input"
          className={`block border-2 border-dashed rounded-2xl p-8 text-center cursor-pointer transition-all duration-200 ${
            dragOver
              ? 'border-primary bg-primary-fixed/20'
              : 'border-surface-variant bg-surface-container-low hover:border-primary/40 hover:bg-surface-container'
          }`}
          onDragOver={e => { e.preventDefault(); setDragOver(true); }}
          onDragLeave={() => setDragOver(false)}
          onDrop={onDrop}
        >
          <input
            id="resume-file-input"
            type="file"
            accept=".pdf,.txt"
            onChange={onInputChange}
            className="hidden"
          />
          {uploading ? (
            <div className="flex flex-col items-center gap-3">
              <Loader2 className="w-8 h-8 text-primary animate-spin" />
              <p className="text-sm font-medium text-on-surface">Parsing resume & extracting skills...</p>
            </div>
          ) : uploadSuccess ? (
            <div className="flex flex-col items-center gap-2">
              <CheckCircle2 className="w-8 h-8 text-emerald-500" />
              <p className="text-sm text-emerald-600 font-semibold">Resume uploaded successfully!</p>
            </div>
          ) : (
            <div className="flex flex-col items-center gap-3">
              <div className="w-12 h-12 rounded-full bg-surface-container flex items-center justify-center text-on-surface-variant">
                <Upload className="w-6 h-6" />
              </div>
              <div>
                <p className="text-sm font-semibold text-on-surface">
                  {dragOver ? 'Drop your resume here' : 'Drag & drop or click to upload'}
                </p>
                <p className="text-xs text-on-surface-variant mt-1">Supports .PDF and .TXT files</p>
              </div>
            </div>
          )}
        </label>
      </div>

      {error && (
        <div className="flex items-center gap-2 p-3 bg-error-container text-on-error-container border border-error/20 rounded-2xl">
          <AlertCircle className="w-4 h-4 flex-shrink-0" />
          <p className="text-xs">{error}</p>
        </div>
      )}

      {loading && !resume && (
        <div className="flex items-center gap-2 py-4 justify-center">
          <Loader2 className="w-4 h-4 text-primary animate-spin" />
          <span className="text-sm text-on-surface-variant font-medium">Loading resume...</span>
        </div>
      )}
    </div>
  );
};

export default ResumeUploader;

