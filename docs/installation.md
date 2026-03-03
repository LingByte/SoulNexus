# Installation Guide

## Requirements

- **Go** >= 1.25.1
- **Node.js** >= 18.0.0
- **npm** >= 8.0.0
- **Git**
- **Python** >= 3.10 (for optional services)

## Installation Steps

### 1. Clone the Project

```bash
git clone https://github.com/your-username/LingEcho.git
cd LingEcho
```

### 2. Backend Configuration

```bash
# Enter project directory
cd LingEcho

# Install Go dependencies
go mod tidy

# Configure environment variables
cp env.example .env.dev
# Edit .env file to configure database and API keys
```

### 3. Frontend Configuration

```bash
# Enter frontend directory
cd ui

# Install dependencies
npm install

# Start development server
npm run dev
```

### 4. Start Backend Service

```bash
# Return to project root directory
cd ..

# Start backend service
go run ./cmd/server/. -mode=dev
```

### 5. Start Optional Services (VAD and Voiceprint)

**VAD Service** (Optional):
```bash
cd services/vad-service
python3 -m venv venv
source venv/bin/activate
pip install -r requirements.txt
python vad_service.py
# Service will start at http://localhost:7073
```

**Voiceprint Recognition Service** (Optional):
```bash
cd services/voiceprint-api
python3.10 -m venv venv  # Python 3.10 recommended
source venv/bin/activate
pip install --upgrade pip setuptools wheel
pip install torch torchaudio --index-url https://download.pytorch.org/whl/cpu
pip install -r requirements.txt
python -m app.main
# Service will start at http://localhost:7074
```

### 6. Access the Application

- **Frontend Interface**: http://localhost:3000
- **Backend API**: http://localhost:7072
- **API Documentation**: http://localhost:7072/api/docs
- **VAD Service**: http://localhost:7073 (if started)
- **Voiceprint Service**: http://localhost:7074/voiceprint/docs (if started)

## Production Environment Configuration

```bash
# Start in production mode
go run ./cmd/server/. -mode=production

# Or use systemd service
sudo systemctl start lingecho
sudo systemctl enable lingecho
```

## Configuration

For detailed environment variable configuration, please refer to [`env.example`](../server/env.example).

The configuration file includes settings for:
- Basic configuration (ports, environment)
- Database configuration (SQLite, PostgreSQL, MySQL)
- API configuration
- LLM configuration
- ASR/TTS provider configuration (Qiniu, Tencent Cloud, etc.)
- Logging, email, search, backup, monitoring
- SIP configuration
- Cache configuration (local, Redis)
- Alibaba Cloud Bailian (knowledge base service)
- Workflow trigger configuration
- Device management configuration
- Alert system configuration
- Billing system configuration

