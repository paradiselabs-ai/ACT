package prompt

// PlannerSectionExamples returns the expandable Planner section
// containing fully-worked PROJECT_BRIEF and CREATE_TASK examples for
// reference. Loaded on-demand via expand_prompt_section("examples")
// when the Planner needs to confirm directive shape, especially when
// it's been a while since the last directive emission.
func PlannerSectionExamples() string {
	return `# Worked Examples

## PROJECT_BRIEF (intake completion)

After the 5-question intake conversation, on user confirmation, write
the brief on its own line in your reply text. NOT a tool call.

` + "```" + `
PROJECT_BRIEF: {"description":"Snake game with neural-network arena where two trained snakes compete on a shared board","techStack":"Python 3.11, PyTorch, Pygame","constraints":"Must run on CPU only, 30fps minimum","successCriteria":"Two snakes complete a 2-minute match without errors; trained model checkpoints save and load","agentsInvolved":["developer","qa_engineer"]}
` + "```" + `

The orchestrator parses this, POSTs to /api/projects, and switches you
to BUILD mode automatically.

## CREATE_TASK (build mode)

Each task is a single CREATE_TASK directive on its own line. Multiple
tasks = multiple lines.

### Example 1: simple

` + "```" + `
CREATE_TASK: {"title":"Implement Snake game core loop","description":"@task\n> Build the Snake class with movement, collision, and food spawning. No graphics yet — pure logic.\n@success_criteria\n- Snake class with up/down/left/right movement\n- Collision detection for walls and self\n- Food spawning avoids snake body\n- Eating food grows snake by 1 segment\n- Unit tests cover movement, collision, growth","requiredCapabilities":["python"],"priority":"high"}
` + "```" + `

### Example 2: with dependencies

` + "```" + `
CREATE_TASK: {"title":"Add Pygame rendering layer","description":"@task\n> Render the existing Snake game state with Pygame. Read state from the Snake class — do not modify it.\n@dependencies\n- Snake game core loop must be complete\n@success_criteria\n- 30fps rendering on CPU\n- Snake, food, and walls all visible\n- Score displayed in corner\n- Window closes cleanly on quit","requiredCapabilities":["python","pygame"],"priority":"medium"}
` + "```" + `

### Example 3: research/spike

` + "```" + `
CREATE_TASK: {"title":"Research PPO vs DQN for snake arena","description":"@task\n> Compare PPO and DQN as RL approaches for two-agent snake competition. Recommend one with justification.\n@success_criteria\n- Side-by-side comparison of sample efficiency, stability, and CPU cost\n- Concrete recommendation with at least 2 references\n- Implementation effort estimate for both","requiredCapabilities":["ml","research"],"priority":"high"}
` + "```" + `

## Common mistakes to avoid

- Wrapping the directive in code fences. Don't.
- Putting multiple JSON objects on one CREATE_TASK line. One task per line.
- Calling bash with the directive as the command. The directive is plain
  text in your reply — the orchestrator scans for it.
- Forgetting @success_criteria. Assurance will reject the task at intake.`
}
