# ACT Configuration Schema Generator

Generates a JSON Schema for the ACT configuration file (`~/.act.json`). Editors
with JSON-Schema support use this for validation and autocomplete.

## Usage

```bash
go run cmd/schema/main.go > act-schema.json
```

## Schema Features

The generated schema includes:

- All configuration options with descriptions
- Default values where applicable
- Validation for enum values (e.g., model IDs, provider types)
- Required fields
- Type checking

## Using the Schema

1. **Editor integration** — point your editor at `act-schema.json` for `~/.act.json`.
2. **Validation tools** — feed both files to any JSON-Schema validator.
3. **Documentation** — the schema enumerates every legal role and field.

## Example Configuration

```json
{
  "data": {
    "directory": ".act"
  },
  "debug": false,
  "providers": {
    "anthropic": {
      "apiKey": "your-api-key"
    }
  },
  "agents": {
    "planner": {
      "model": "claude-opus-4-20250514",
      "maxTokens": 8000
    },
    "observer": {
      "model": "claude-sonnet-4-20250514",
      "maxTokens": 2000
    },
    "assurance": {
      "model": "claude-sonnet-4-20250514",
      "maxTokens": 5000
    },
    "qa_synthesizer": {
      "model": "claude-sonnet-4-20250514",
      "maxTokens": 5000
    },
    "developer": {
      "model": "claude-sonnet-4-20250514",
      "maxTokens": 5000,
      "backend": "act-agent"
    }
  }
}
```
