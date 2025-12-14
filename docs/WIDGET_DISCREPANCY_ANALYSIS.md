# ACT Widget System Codebase Discrepancy Analysis

## 📊 **CODEBASE ANALYSIS: Current State vs Requirements**

### ✅ **What Exists and Works**

#### 1. ACT Server Core (`/sdk/python/server/`)
- **Express server**: ✅ Running on port 8080
- **Socket.IO integration**: ✅ WebSocket server operational
- **Coordination events**: ✅ EventHub broadcasts events via Socket.IO
- **REST API**: ✅ `/api/agents`, `/api/tasks`, `/api/pvm/search` endpoints work
- **Health endpoint**: ✅ `/health` returns server status

#### 2. Dashboard Assets (`/client/`)
- **HTML dashboard**: ✅ `index.html` - Complete 4-widget layout
- **Widget JavaScript**: ✅ `widgets.js` - WebSocket client with event handlers
- **Styling**: ✅ Dark theme with responsive design
- **Widget components**: ✅ Agent Registry, Task Coordinator, Project Status, Conflict Resolution

#### 3. WebSocket Events
- **Event broadcasting**: ✅ Server emits `agent_registered`, `task_assigned`, `task_progress`, etc.
- **Event structure**: ✅ Properly formatted JSON events with timestamps

### ❌ **Critical Discrepancies Identified**

#### **DISCREPANCY #1: Static File Serving Not Configured**
**Issue**: Server cannot serve HTML/CSS/JS files
**Evidence**:
```bash
$ curl -s -I http://localhost:8080/
HTTP/1.1 404 Not Found
```
**Code Location**: `/sdk/python/server/src/index.ts`
**Missing Code**:
```typescript
// Should exist but doesn't
app.use(express.static("public"));
app.get("/", (req, res) => {
  res.sendFile(path.join(process.cwd(), "public/index.html"));
});
```

#### **DISCREPANCY #2: Dashboard Files in Wrong Location**
**Issue**: Files exist in `/client/` but server expects `/public/`
**Evidence**:
```bash
$ ls /client/
index.html  widgets.js
$ ls /sdk/python/server/public/
# Empty or doesn't exist
```
**Required Action**: Copy files from `/client/` to `/sdk/python/server/public/`

#### **DISCREPANCY #3: Missing Path Import**
**Issue**: Server uses `__dirname` but doesn't import `path` module
**Code Location**: `/sdk/python/server/src/index.ts`
**Current Code**: `res.sendFile(__dirname + "/public/index.html");`
**Should Be**: `res.sendFile(path.join(process.cwd(), "public/index.html"));`

#### **DISCREPANCY #4: WebSocket Protocol Alignment**
**Issue**: Need to verify Socket.IO event names match between server and client
**Server Events**: `agent_registered`, `task_assigned`, `task_progress`, `agent_status_update`
**Client Listeners**: Need to check if `widgets.js` listens for correct event names

#### **DISCREPANCY #5: CORS Configuration**
**Issue**: WebSocket connections may fail due to CORS
**Current CORS**: Only allows `["http://localhost:3000", "http://localhost:3001", "http://localhost:5173", "http://localhost:5000"]`
**Missing**: `http://localhost:8080` for dashboard accessing itself

### 🔍 **Detailed Code Analysis**

#### Server Event Broadcasting (`EventHub.ts`):
```typescript
// ✅ Correctly broadcasts via Socket.IO
io.emit('agent_registered', { agentId, name, capabilities });
io.emit('task_assigned', { taskId, agentId, task });
io.emit('task_progress', { taskId, progress, status, message });
```

#### Client Event Listening (`widgets.js`):
```javascript
// ✅ Correctly listens for events
socket.on('agent_registered', (e) => { this.handleAgentRegistered(e); });
socket.on('task_assigned', (e) => { this.handleTaskAssigned(e); });
socket.on('task_progress', (e) => { this.handleTaskProgress(e); });
```

#### File Structure Issues:
```
/client/ (files exist here)
├── index.html ✅
└── widgets.js ✅

/sdk/python/server/public/ (files should be here)
└── (empty - needs files copied)
```

### 📋 **FIX PRIORITY ORDER**

1. **High Priority** (Blocks basic functionality):
   - Copy dashboard files to correct location
   - Add static file serving to server
   - Fix dashboard routes

2. **Medium Priority** (Improves functionality):
   - Fix path imports and CORS
   - Verify WebSocket event alignment

3. **Low Priority** (Polish):
   - Clean up any corrupted code
   - Optimize performance
   - Add error handling

### ✅ **VERIFICATION CHECKLIST**

- [ ] `http://localhost:8080/` serves HTML (not 404)
- [ ] Dashboard loads with 4 widget sections
- [ ] WebSocket connection establishes (no console errors)
- [ ] `node test-agent.js` triggers real-time widget updates
- [ ] All widgets show live data (agents, tasks, progress, activity)
- [ ] No crashes or memory leaks during 5+ minute test
