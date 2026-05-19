# google/gemma-3-12b — produce-01 (produce)

**score:** 100/100
**reasons:** (none)
**elapsedMs:** 54929  **tok/s:** 12.03
**usage:** {"prompt_tokens":395,"completion_tokens":661,"total_tokens":1056,"completion_tokens_details":{"reasoning_tokens":0}}

## Prompt
```
[
  {
    "role": "system",
    "content": "You are the Planner agent in ACT. You produce SPIL (Structured Progressive Instruction Language) task specifications.\n\nSPIL rules:\n- @keyword \"value\"  — inline metadata\n- @keyword:         — section header, items follow as `- item` lines\n- > \"text\"          — natural language directive, placed where most relevant (NOT grouped at top/bottom)\n\nCTD (Conceptually Top-Down) ordering — each section must depend only on what's above:\n  1. Core metadata (@task, @project, @priority)\n  2. Core directive — one `>` establishing scope\n  3. @data — data structures\n  4. @context — what already exists / files to reference\n  5. @success_criteria — the validation checklist (Assurance scores each item)\n  6. Nuanced `>` directives elaborating constraints\n  7. @error_handling — failure conditions and responses\n  8. Preferential `>` directives — style, conventions (last)\n\nRequired sections: @task, @data, @context, @success_criteria, @error_handling.\n\n@success_criteria items must be concrete, testable outcomes (verbs like 'returns', 'creates', 'rejects'). Not vague ('works correctly').\n\nOutput ONLY the SPIL document. No prose, no markdown fences, no commentary."
  },
  {
    "role": "user",
    "content": "Write a SPIL task spec for this work:\n\nProject: pomodoro-cli\nTask: Add `pause` and `resume` subcommands to the CLI. State persists across invocations in a JSON file at ~/.pomodoro/state.json. A paused timer must remain paused even if the user closes the terminal. Existing `start` and `stop` subcommands already work and store timer state in the same file. Use Node.js + commander.js. Test with vitest."
  }
]
```

## Raw response
```
@task "Add pause and resume subcommands to pomodoro-cli"
@project "pomodoro-cli"
@priority "medium"
> Implement pause and resume functionality for the pomodoro CLI, persisting state in a JSON file.

@data
- `stateFile`: Path to the state persistence file (e.g., "~/.pomodoro/state.json")
- `timerState`: JSON object representing the timer's state, including:
    - `isRunning`: boolean (true if timer is running, false otherwise)
    - `pauseTime`: number (timestamp representing when the timer was paused, null if not paused)
    - `startTime`: number (timestamp representing when the timer started, null if not running)
    - `remainingTime`: number (remaining time in seconds)

@context
- Existing CLI with `start` and `stop` subcommands using Node.js and commander.js
- State persistence implemented in JSON file at `~/.pomodoro/state.json`
- Vitest for testing

@success_criteria
- `pause` subcommand:
    - Returns 0 on success.
    - Pauses the timer if it is running, updating `pauseTime` in `timerState`.
    - Saves the updated `timerState` to `stateFile`.
    - Prints a confirmation message "Timer paused." to the console.
- `resume` subcommand:
    - Returns 0 on success.
    - Resumes the timer if it is paused, calculating `remainingTime` based on `pauseTime` and `startTime`.
    - Updates `isRunning` to true in `timerState`.
    - Saves the updated `timerState` to `stateFile`.
    - Prints a confirmation message "Timer resumed." to the console.
- Timer state persistence:
    - Paused timer remains paused even after terminal closure and subsequent re-opening.
    - Resumed timer continues from the point of pausing after terminal closure and subsequent re-opening.
- Vitest tests:
    - Tests for `pause` subcommand pass, covering scenarios with and without a running timer.
    - Tests for `resume` subcommand pass, covering scenarios with and without a paused timer.
    - State persistence tests pass, verifying that pausing and resuming works across invocations.

@error_handling
- `pause` subcommand:
    - If the timer is not running, prints an error message "Timer is not running." to the console and returns 1.
- `resume` subcommand:
    - If the timer is not paused, prints an error message "Timer is not paused." to the console and returns 1.
- File access errors:
    - If there is an error reading or writing to `stateFile`, prints a generic error message "Error accessing state file." and returns 1.

> Ensure the pause and resume commands integrate seamlessly with existing start and stop functionality, sharing the same state persistence mechanism.

> Adhere to established CLI command structure using commander.js.

> Write Vitest tests that thoroughly validate pause and resume functionality, including persistence across terminal sessions.

```

## Stripped (thinking removed)
```
@task "Add pause and resume subcommands to pomodoro-cli"
@project "pomodoro-cli"
@priority "medium"
> Implement pause and resume functionality for the pomodoro CLI, persisting state in a JSON file.

@data
- `stateFile`: Path to the state persistence file (e.g., "~/.pomodoro/state.json")
- `timerState`: JSON object representing the timer's state, including:
    - `isRunning`: boolean (true if timer is running, false otherwise)
    - `pauseTime`: number (timestamp representing when the timer was paused, null if not paused)
    - `startTime`: number (timestamp representing when the timer started, null if not running)
    - `remainingTime`: number (remaining time in seconds)

@context
- Existing CLI with `start` and `stop` subcommands using Node.js and commander.js
- State persistence implemented in JSON file at `~/.pomodoro/state.json`
- Vitest for testing

@success_criteria
- `pause` subcommand:
    - Returns 0 on success.
    - Pauses the timer if it is running, updating `pauseTime` in `timerState`.
    - Saves the updated `timerState` to `stateFile`.
    - Prints a confirmation message "Timer paused." to the console.
- `resume` subcommand:
    - Returns 0 on success.
    - Resumes the timer if it is paused, calculating `remainingTime` based on `pauseTime` and `startTime`.
    - Updates `isRunning` to true in `timerState`.
    - Saves the updated `timerState` to `stateFile`.
    - Prints a confirmation message "Timer resumed." to the console.
- Timer state persistence:
    - Paused timer remains paused even after terminal closure and subsequent re-opening.
    - Resumed timer continues from the point of pausing after terminal closure and subsequent re-opening.
- Vitest tests:
    - Tests for `pause` subcommand pass, covering scenarios with and without a running timer.
    - Tests for `resume` subcommand pass, covering scenarios with and without a paused timer.
    - State persistence tests pass, verifying that pausing and resuming works across invocations.

@error_handling
- `pause` subcommand:
    - If the timer is not running, prints an error message "Timer is not running." to the console and returns 1.
- `resume` subcommand:
    - If the timer is not paused, prints an error message "Timer is not paused." to the console and returns 1.
- File access errors:
    - If there is an error reading or writing to `stateFile`, prints a generic error message "Error accessing state file." and returns 1.

> Ensure the pause and resume commands integrate seamlessly with existing start and stop functionality, sharing the same state persistence mechanism.

> Adhere to established CLI command structure using commander.js.

> Write Vitest tests that thoroughly validate pause and resume functionality, including persistence across terminal sessions.
```