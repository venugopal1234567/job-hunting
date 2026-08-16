import React, { useCallback, useState } from 'react';
import { Upload, FileText, CheckCircle2, AlertCircle, X, Loader2 } from 'lucide-react';
import { useResume } from '../../hooks/useResume';

interface ResumeUploaderProps {
  onResumeUploaded?: () => void;
}

const ResumeUploader: React.FC<ResumeUploaderProps> = ({ onResumeUploaded }) => {
  const { resume, loading, uploading, error, upload, refresh } = useResume();
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
        <div className="glass rounded-xl p-4 border border-emerald-500/20 animate-fade-in">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-xl bg-emerald-500/10 border border-emerald-500/20 flex items-center justify-center">
              <FileText className="w-5 h-5 text-emerald-400" />
            </div>
            <div className="flex-1">
              <p className="text-sm font-semibold text-white">{resume.filename}</p>
              <p className="text-xs text-gray-400 mt-0.5">
                {resume.raw_text_length.toLocaleString()} chars · {resume.extracted_skills?.length || 0} skills detected
              </p>
            </div>
            <CheckCircle2 className="w-5 h-5 text-emerald-400 flex-shrink-0" />
          </div>

          {/* Extracted Skills */}
          {resume.extracted_skills && resume.extracted_skills.length > 0 && (
            <div className="mt-4 pt-4 border-t border-white/5">
              <p className="section-header">Extracted Skills</p>
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
        <p className="section-header">{resume ? 'Replace Resume' : 'Upload Resume'}</p>
        <label
          id="resume-upload-dropzone"
          htmlFor="resume-file-input"
          className={`block border-2 border-dashed rounded-xl p-8 text-center cursor-pointer transition-all duration-200 ${
            dragOver
              ? 'border-brand-500 bg-brand-500/10'
              : 'border-surface-400 hover:border-surface-300 hover:bg-surface-200/50'
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
              <Loader2 className="w-8 h-8 text-brand-400 animate-spin" />
              <p className="text-sm text-gray-300">Parsing resume & extracting skills...</p>
            </div>
          ) : uploadSuccess ? (
            <div className="flex flex-col items-center gap-2">
              <CheckCircle2 className="w-8 h-8 text-emerald-400" />
              <p className="text-sm text-emerald-300 font-medium">Resume uploaded successfully!</p>
            </div>
          ) : (
            <div className="flex flex-col items-center gap-3">
              <Upload className="w-8 h-8 text-gray-500" />
              <div>
                <p className="text-sm text-gray-300 font-medium">
                  {dragOver ? 'Drop your resume here' : 'Drag & drop or click to upload'}
                </p>
                <p className="text-xs text-gray-600 mt-1">Supports .PDF and .TXT files</p>
              </div>
            </div>
          )}
        </label>
      </div>

      {error && (
        <div className="flex items-center gap-2 p-3 bg-red-500/10 border border-red-500/20 rounded-lg">
          <AlertCircle className="w-4 h-4 text-red-400 flex-shrink-0" />
          <p className="text-xs text-red-300">{error}</p>
        </div>
      )}

      {loading && !resume && (
        <div className="flex items-center gap-2 py-4">
          <Loader2 className="w-4 h-4 text-brand-400 animate-spin" />
          <span className="text-sm text-gray-400">Loading resume...</span>
        </div>
      )}
    </div>
  );
};

export default ResumeUploader;
