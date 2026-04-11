package prompt

// PlannerSectionExamples returns the expandable Planner section
// containing fully-worked PROJECT_BRIEF and CREATE_TASK examples for
// reference. Loaded on-demand via expand_prompt_section("examples")
// when the Planner needs to confirm directive shape, especially when
// it's been a while since the last directive emission.
func PlannerSectionExamples() string {
	return `# Worked Examples

Examples use EXAMPLE_PROJECT_BRIEF and EXAMPLE_TASK as markers so they
cannot be confused with real directives. When you emit real directives,
use PROJECT_BRIEF: and CREATE_TASK: (no EXAMPLE_ prefix).

## Intake completion

After the 5-question intake conversation, on user confirmation, write
the brief as plain text in your reply. NOT a tool call, NOT in a code fence.

EXAMPLE_PROJECT_BRIEF: {"description":"Snake game with neural-network arena where two trained snakes compete on a shared board","techStack":"Python 3.11, PyTorch, Pygame","constraints":"Must run on CPU only, 30fps minimum","successCriteria":"Two snakes complete a 2-minute match without errors; trained model checkpoints save and load","agentsInvolved":["developer","qa_engineer"]}

## Build mode — task directives

Each task is one directive on its own line. Multiple tasks = multiple lines.
Plain text in your reply — not a tool call, not in a code fence.

### Simple task

EXAMPLE_TASK: {"title":"Implement Snake game core loop","description":"@task\n> Build the Snake class with movement, collision, and food spawning. No graphics yet — pure logic.\n@success_criteria\n- Snake class with up/down/left/right movement\n- Collision detection for walls and self\n- Food spawning avoids snake body\n- Eating food grows snake by 1 segment\n- Unit tests cover movement, collision, growth","requiredCapabilities":["python"],"priority":"high"}

### Task with dependencies

EXAMPLE_TASK: {"title":"Add Pygame rendering layer","description":"@task\n> Render the existing Snake game state with Pygame. Read state from the Snake class — do not modify it.\n@dependencies\n- Snake game core loop must be complete\n@success_criteria\n- 30fps rendering on CPU\n- Snake, food, and walls all visible\n- Score displayed in corner\n- Window closes cleanly on quit","requiredCapabilities":["python","pygame"],"priority":"medium"}

### Research/spike

EXAMPLE_TASK: {"title":"Research PPO vs DQN for snake arena","description":"@task\n> Compare PPO and DQN as RL approaches for two-agent snake competition. Recommend one with justification.\n@success_criteria\n- Side-by-side comparison of sample efficiency, stability, and CPU cost\n- Concrete recommendation with at least 2 references\n- Implementation effort estimate for both","requiredCapabilities":["ml","research"],"priority":"high"}

## Common mistakes to avoid

- Wrapping the directive in a code fence. Don't — the parser won't see it.
- Putting multiple JSON objects on one CREATE_TASK line. One task per line.
- Calling bash with the directive as the command. Plain text in your reply only.
- Forgetting @success_criteria. Assurance will reject the task at validation.`
}
