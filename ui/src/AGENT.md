# Frontend Architecture & Contributor Guide (`ui/src`)

Welcome to the **RemoteHunter UI** codebase (`ui/src`). This document provides an architectural overview of the frontend application and guidelines for open-source contributors maintaining or extending the user interface.

---

## 📁 Directory Structure

```
ui/src/
├── components/          # Reusable UI components categorized by domain
│   ├── dashboard/       # Job listing cards, search/filters, and stats overview
│   ├── job-detail/      # Detailed job modal and job description breakdown
│   ├── layout/           # Application navigation bar and main shell
│   ├── resume/          # AI Resume editor, ATS score bar, chat panel, upload modal
│   │   └── resume-editor/ # Modular components for ResumeEditor (Header, Canvas)
│   └── settings/        # AI model configuration picker and scraper source manager
├── hooks/               # Custom React hooks managing local & async state
│   ├── useAtsAnalysis.ts # Job ATS scoring state and operations
│   ├── useJobs.ts        # Job filtering, search, pagination, and fetching
│   ├── useResume.ts      # Active resume metadata loading
│   └── useResumeEditor.ts# State management for resume text, chat history & saves
├── services/            # API integration modules
│   └── api.ts           # Axios client & typed backend endpoints
├── types/               # TypeScript interfaces & domain models
│   └── index.ts         # Job, Resume, ScraperConfig, and API response types
├── utils/               # Helper utilities & pure logic modules
│   ├── resumeHelpers.ts # String escaping, DOM text extraction, keyword matching
│   ├── resumeParser.ts  # Plain-text resume parser & structured document generator
│   └── resumeRenderer.ts# Visual HTML & print CSS layout generator
├── App.tsx              # Main layout, route routing, and active tab manager
├── index.css            # Tailwind CSS directives & global design tokens
└── main.tsx             # Application DOM entry point
```

---

## 🛠 Coding & Design Standards

1. **File Size Limit**:
   - No source file should exceed **500 lines**. Large features must be broken down into modular sub-components or utility functions.

2. **Component Separation**:
   - Keep stateful container components separated from presentational components.
   - Domain-specific utilities (parsers, formatters, renderers) live inside `src/utils/` rather than mixed inside `.tsx` files.

3. **Styling Guidelines**:
   - Use TailwindCSS classes for component layouts.
   - Global design system variables and custom scrollbar/glassmorphism classes reside in `src/index.css`.

4. **Type Safety**:
   - Always define interfaces in `src/types/index.ts` for all API payloads and component props. Avoid using `any`.

---

## 🚀 Verification & Building

Before submitting a pull request, run the build script to ensure TypeScript types and bundle bundling pass:

```bash
npm run build
```

This runs `tsc` followed by `vite build`. All code must compile with zero errors.
