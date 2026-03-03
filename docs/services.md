# Services Documentation

## Core Services

### Main Service

**Location**: `cmd/server/`

**Description**: Core backend service providing RESTful API and WebSocket support

**Port**: 7072

**Features**:
- RESTful API endpoints
- WebSocket connections
- Database management
- Authentication and authorization
- File upload and storage

### Voice/SIP Service

**Description**: Integrated into main server, providing SIP softphone and voice processing capabilities

**Features**:
- SIP protocol implementation
- Real-time audio processing
- Call management
- Call recording
- ACD (Automatic Call Distribution)

### MCP Service

**Location**: `cmd/mcp/`

**Description**: Model Context Protocol service

## Standalone Services

### VAD Service

**Location**: `services/vad-service/`

**Port**: 7073

**Technology**: Python + FastAPI + SileroVAD

**Features**:
- Voice activity detection
- Supports PCM and OPUS formats
- HTTP RESTful API
- Real-time voice activity detection
- Silence detection and session management
- Dual threshold mechanism with sliding window smoothing

**Quick Start**:
```bash
cd services/vad-service
python3 -m venv venv
source venv/bin/activate
pip install -r requirements.txt
python vad_service.py
```

**API Endpoints**:
- `POST /vad/detect` - Detect voice activity
- `POST /vad/session/create` - Create a new session
- `POST /vad/session/update` - Update session state
- `DELETE /vad/session/{session_id}` - Delete a session

### Voiceprint Recognition Service

**Location**: `services/voiceprint-api/`

**Port**: 7074

**Technology**: Python + FastAPI + ModelScope

**Features**:
- Speaker identification
- Voiceprint registration and management
- Multi-speaker identification
- Similarity calculation
- MySQL database storage
- RESTful API interface

**Quick Start**:
```bash
cd services/voiceprint-api
python3.10 -m venv venv  # Python 3.10 recommended
source venv/bin/activate
pip install --upgrade pip setuptools wheel
pip install torch torchaudio --index-url https://download.pytorch.org/whl/cpu
pip install -r requirements.txt
python -m app.main
```

**API Endpoints**:
- `POST /voiceprint/register` - Register a voiceprint
- `POST /voiceprint/identify` - Identify a speaker
- `GET /voiceprint/list` - List all voiceprints
- `DELETE /voiceprint/{voiceprint_id}` - Delete a voiceprint

**API Documentation**: http://localhost:7074/voiceprint/docs

