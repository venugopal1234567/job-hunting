# 🎯 RemoteHunter

> **AI-Powered Job Aggregator & Intelligent ATS Resume Tailoring Engine**

RemoteHunter is an open-source, full-stack job intelligence platform designed to simplify remote job discovery, track applications, and optimize resumes for ATS (Applicant Tracking Systems) using customizable AI models.

---

## ✨ Features

- **🤖 Automated Multi-Source Job Scraping**: Periodically scrapes and normalizes job listings across diverse remote job boards using a flexible cron scheduler.
- **⚡ Interactive Visual Resume Canvas**: Real-time 1-page fitted visual canvas editor with live print preview, PDF export via backend, and instant score feedback.
- **🎯 AI Resume Coach & ATS Optimizer**: Tailor resumes specifically to selected job postings with AI-guided bullet optimizations, missing keyword analysis, and score tracking.
- **⚙️ Multi-Model AI Support**: Switch dynamically between OpenAI-compatible LLMs, NVIDIA API endpoints, and local models (e.g. via Ollama).
- **📊 Real-time Dashboard**: Filter jobs by remote region, job title, source board, and publication age with instant metrics.
- **📄 Native PDF Handling**: Direct PDF uploading, text extraction, and original document preview.

---

## 🏗 Tech Stack

| Layer | Technology |
| :--- | :--- |
| **Frontend** | React 18, TypeScript, Vite, Tailwind CSS, Lucide Icons |
| **Backend** | Go 1.22+, Chi Router, PostgreSQL, Headless Chrome API |
| **Containerization** | Docker, Docker Compose, Nginx |
| **AI Integration** | OpenAI-compatible API client, NVIDIA NIM / Ollama support |

---

## 🚀 Quick Start with Docker

### Prerequisites
- [Docker](https://docs.docker.com/get-docker/) & [Docker Compose](https://docs.docker.com/compose/install/)

### 1. Clone the Repository
```bash
git clone https://github.com/your-username/remotehunter.git
cd remotehunter
```

### 2. Configure Environment Variables
Copy the example environment file:
```bash
cp .env.example .env
```
Edit `.env` to configure your API keys (e.g., `NVIDIA_API_KEY`, `SERPAPI_API_KEY`).

### 3. Launch Services
Start all containers in detached mode:
```bash
docker compose up -d --build
```

### 4. Access Application
Open your browser and navigate to:
- **Web UI**: `http://localhost:3000`
- **Backend API**: `http://localhost:8080/api/v1`

---

## 🛠 Manual Development Setup

### Backend (Go)
1. Ensure PostgreSQL is running on port `5432` with a database named `remotehunter`.
2. Navigate to the `backend` directory:
   ```bash
   cd backend
   go run cmd/server/main.go
   ```

### Frontend (React + Vite)
1. Navigate to the `ui` directory:
   ```bash
   cd ui
   npm install
   npm run dev
   ```
2. The UI will run at `http://localhost:5173`.

---

## 📁 Repository Structure

```
├── backend/                  # Go REST API backend & scraper engine
│   ├── cmd/server/           # Application entry point
│   ├── internal/             # Core internal packages
│   │   ├── db/               # PostgreSQL migrations & DB handlers
│   │   ├── handler/          # HTTP request handlers & routing
│   │   ├── models/           # Go struct schemas & database entities
│   │   ├── parser/           # PDF & text parser utilities
│   │   └── scraper/          # Scraper interfaces & board implementations
│   └── Dockerfile
├── ui/                       # React TypeScript frontend
│   ├── src/
│   │   ├── components/       # Dashboard, Resume Editor, Settings components
│   │   ├── hooks/            # Custom React hooks
│   │   ├── services/         # API client & services
│   │   ├── types/            # TypeScript interfaces
│   │   └── utils/            # Resume parsers & layout renderers
│   ├── AGENT.md              # Frontend architecture & developer guide
│   └── Dockerfile
├── AGENT.md                  # Guide for adding new job scrapers
├── docker-compose.yml        # Docker composition manifest
└── README.md
```

---

## 🤝 Contributing

Contributions are welcome! If you would like to add a new job scraper or improve the resume editor:
1. Refer to [AGENT.md](file:///home/venu/Documents/projects/ai/job-hunting/AGENT.md) for step-by-step instructions on writing and registering new scraper modules.
2. Refer to [`ui/src/AGENT.md`](file:///home/venu/Documents/projects/ai/job-hunting/ui/src/AGENT.md) for frontend contribution guidelines and code standards.
3. Open a Pull Request with clean, tested changes.

---

## 📄 License

[MIT License](LICENSE)