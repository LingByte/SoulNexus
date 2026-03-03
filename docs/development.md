# Development Guide

## 🤝 Contributing

We welcome all forms of contributions! Please check our contributing guidelines:

### Development Process

1. **Fork the Project** - Click the Fork button in the top right corner
2. **Create a Branch** - `git checkout -b feature/your-feature`
3. **Commit Changes** - `git commit -m 'Add some feature'`
4. **Push Branch** - `git push origin feature/your-feature`
5. **Create PR** - Create a Pull Request on GitHub

### Code Standards

- **Go Code** - Follow Go official code standards
- **TypeScript** - Use ESLint and Prettier
- **Commit Messages** - Use conventional commit format
- **Test Coverage** - New features need to include test cases

## Development Environment Setup

### Backend Development

```bash
# Install dependencies
go mod tidy

# Run in development mode
go run ./cmd/server/. -mode=dev

# Run tests
go test ./...
```

### Frontend Development

```bash
cd ui

# Install dependencies
npm install

# Start development server
npm run dev

# Build for production
npm run build
```

## Project Structure

```
LingEcho/
├── cmd/                    # Application entry points
│   ├── server/             # Main server
│   └── mcp/                # MCP service
├── internal/               # Internal packages
│   ├── models/             # Data models
│   ├── handlers/           # HTTP handlers
│   └── services/           # Business logic
├── pkg/                    # Public packages
│   ├── sip/                # SIP protocol implementation
│   └── hardware/          # Hardware device support
├── ui/                     # Frontend React application
│   ├── src/
│   │   ├── components/     # React components
│   │   ├── pages/          # Page components
│   │   └── api/            # API clients
│   └── public/             # Static assets
├── services/               # Standalone services
│   ├── vad-service/        # VAD service
│   └── voiceprint-api/     # Voiceprint service
└── docs/                   # Documentation
```

## API Development

### Adding New API Endpoints

1. Define the route in the appropriate router file
2. Create handler function in `internal/handlers/`
3. Add business logic in `internal/services/`
4. Update API documentation

### Testing APIs

```bash
# Use curl or Postman to test endpoints
curl -X GET http://localhost:7072/api/endpoint
```

## Database Migrations

```bash
# Run migrations
go run ./cmd/server/. -migrate

# Create new migration
# Edit migration files in internal/database/migrations/
```

## Debugging

### Backend Debugging

- Use Go's built-in debugger (delve)
- Check logs in `logs/` directory
- Enable debug mode: `-mode=dev`

### Frontend Debugging

- Use React DevTools
- Check browser console
- Enable source maps in development mode

