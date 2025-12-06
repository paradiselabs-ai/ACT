# ACT Development Setup

## Environment Status ✅

### Python Environment
- **Virtual Environment**: `/Users/user/Documents/Developer/dev/AI/act/venv`
- **Status**: ✅ Activated and configured
- **Dependencies**: All installed (see `requirements.txt`)
- **Global Environment**: ✅ Clean (no ACT-specific packages globally installed)

### Node.js Services
All services have dependencies installed:
- ✅ `server/` - ACT Phase 5 server (TypeScript)
- ✅ `mcp-servers/act-coordination-mcp/` - MCP coordination server
- ✅ `mcp-servers/act-mcp-bridge/` - ACT MCP bridge
- ✅ `sdk/python/server/` - ACT coordination server (TypeScript/Socket.IO)
- ✅ `dashboard/` - React dashboard
- ✅ `examples/` - Example agents

## Quick Start

### Python (Examples & Agents)
```bash
# Activate venv
source venv/bin/activate

# Verify dependencies
pip list | grep -E "socketio|aiohttp"

# Run example
python examples/real_ai_agents.py
```

### TypeScript Services

**Phase 5 Server (ChronologicalLog + VectorMemoryStore):**
```bash
cd server
npm test           # Run tests (29/29 passing)
npm run build      # Compile TypeScript
npm run dev        # Development mode
```

**ACT Coordination Server:**
```bash
cd sdk/python/server
npm run build      # Compile TypeScript
npm run dev        # Start server on localhost:8080
```

**MCP Servers:**
```bash
# Coordination MCP
cd mcp-servers/act-coordination-mcp
npm run build

# ACT Bridge MCP
cd mcp-servers/act-mcp-bridge
npm run build
```

## Dependencies

### Python (`requirements.txt`)
- `python-socketio==5.15.0` - WebSocket client
- `aiohttp==3.13.2` - Async HTTP client
- `websocket-client==1.8.0` - WebSocket utilities
- `requests==2.32.5` - HTTP requests

### TypeScript (various `package.json`)
- `@qdrant/js-client-rest` - Vector database (server/)
- `jest`, `ts-jest` - Testing framework
- `typescript` - TypeScript compiler
- `socket.io` - WebSocket server (sdk/python/server/)

## Verification

**Check Python venv:**
```bash
which python
# Should output: /Users/user/Documents/Developer/dev/AI/act/venv/bin/python
```

**Check global Python (should be clean):**
```bash
deactivate  # Exit venv if active
pip list | grep -E "socketio|aiohttp"
# Should output: nothing
```

**Check Node.js installations:**
```bash
cd server && npm test  # Should show 29/29 tests passing
```

## Notes

- ✅ **Global Python is clean** - No ACT dependencies installed globally
- ✅ **Venv is isolated** - All Python dependencies in venv only
- ✅ **All Node.js services have dependencies** - Ready to run
- ⚠️ **Always activate venv** before running Python examples
- ⚠️ **Each Node service needs build** before first run (`npm run build`)
