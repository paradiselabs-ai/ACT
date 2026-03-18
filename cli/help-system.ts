/**
 * Help System - Comprehensive command reference for ACT REPL
 *
 * Provides detailed help for all commands and usage examples.
 */

export class HelpSystem {

  showHelp(command?: string): void {
    if (command) {
      this.showSpecificHelp(command);
    } else {
      this.showGeneralHelp();
    }
  }

  private showGeneralHelp(): void {
    console.log('\nACT Commands Reference:');
    console.log('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n');

    console.log('Configuration:');
    console.log('  list agents              List connected agents');
    console.log('  default agent <id>       Set default agent (Planner) for planning');
    console.log('  default agent -r         Randomly pick an Planner from registered agents');
    console.log('  remove agent <id>        Deregister an agent from ACT');
    console.log('  show default            Show current default agent\n');

    console.log('Ask Agents:');
    console.log('  ask agents <prompt>      Broadcast a question/instruction to all agents');
    console.log('  ask <prompt>             Shorthand for ask agents\n');

    console.log('Projects:');
    console.log('  create project <name> <path>       Create new project');
    console.log('  create project <name>              Create project (prompts for path)');
    console.log('  continue project <name>            Resume existing project');
    console.log('  list projects                      Show all projects');
    console.log('  show project <name>                Show project details');
    console.log('  stop project <name>                Pause project execution');
    console.log('  delete project <name>              Remove project\n');
    console.log('  nestty [--roles list] [--mock]     Launch NesTTY for current project\n');

    console.log('Sessions:');
    console.log('  brainstorm <topic> [--agents list]         Creative ideation');
    console.log('  experiment <name> [--agents list]          Comparative testing');
    console.log('  experiment -analyze <name>                 Analyze experiment');
    console.log('  roundtable <topic>                         Multi-agent discussion');
    console.log('  roundtable <topic> --interactive           HITL controls enabled\n');

    console.log('Interactive Controls (during --interactive sessions):');
    console.log('  pause                   Pause discussion');
    console.log('  resume                  Resume discussion');
    console.log('  select <agent>          Highlight agent\'s contribution');
    console.log('  edit <msg_id>           Edit message');
    console.log('  delete <msg_id>         Remove message');
    console.log('  send "<message>"        User contributes');
    console.log('  stop                    End session');
    console.log('  clean_up                Finalize and save');
    console.log('  wipe                    Remove from PVM (destructive)\n');

    console.log('Improvement:');
    console.log('  improve <scope> [options]          Run improvement analysis\n');

    console.log('  Scopes:');
    console.log('    communication, tools, assignments, conflicts,');
    console.log('    collaboration, performance, knowledge\n');

    console.log('  Session Filters:');
    console.log('    -project <name>');
    console.log('    -brainstorm <name>');
    console.log('    -roundtable <name>');
    console.log('    -experiment <name>\n');

    console.log('  Options:');
    console.log('    --agents <list>         Focus on specific agents');
    console.log('    --session <id>          Specific session');
    console.log('    --filter <good|bad|all> Quality filter');
    console.log('    --output <format>       Output format');
    console.log('    --focus <list>          Only analyze listed agents');
    console.log('    --print <path>          Save to file (markdown)\n');

    console.log('PVM (Advanced):');
    console.log('  pvm stats                   Show PVM statistics');
    console.log('  pvm search <query>          Search coordination history');
    console.log('  pvm profile <agent_id>      Show agent profile');
    console.log('  pvm export <path>           Export PVM database');
    console.log('  pvm import <path>           Import PVM database\n');

    console.log('System:');
    console.log('  status                      Show ACT server status');
    console.log('  help                        Show this help');
    console.log('  help <command>              Show detailed help for command');
    console.log('  exit                        Exit ACT REPL\n');

    console.log('Full documentation coming soon.\n');
  }

  private showSpecificHelp(command: string): void {
    switch (command.toLowerCase()) {
      case 'list':
      case 'list agents':
        this.showListHelp();
        break;
      case 'create':
      case 'create project':
        this.showCreateProjectHelp();
        break;
      case 'improve':
        this.showImproveHelp();
        break;
      case 'brainstorm':
        this.showBrainstormHelp();
        break;
      case 'experiment':
        this.showExperimentHelp();
        break;
      case 'roundtable':
        this.showRoundtableHelp();
        break;
      case 'pvm':
        this.showPVMHelp();
        break;
      default:
        console.log(`No detailed help available for: ${command}`);
        console.log('Type "help" for general command reference.');
    }
  }

  private showListHelp(): void {
    console.log('\nLIST AGENTS - Show connected agents\n');
    console.log('Usage:');
    console.log('  list agents\n');
    console.log('Description:');
    console.log('  Displays a table of all agents currently connected to the ACT server,');
    console.log('  including their status, current workload, and capabilities.\n');
    console.log('Example:');
    console.log('  >>: list agents');
    console.log('  ┌─────────────────┬──────────────────────┬──────────┬─────────────┐');
    console.log('  │ Agent ID        │ Name                 │ Status   │ Workload    │');
    console.log('  ├─────────────────┼──────────────────────┼──────────┼─────────────┤');
    console.log('  │ claude_code_1   │ Claude Code #1       │ Online   │ 0 tasks     │');
    console.log('  └─────────────────┴──────────────────────┴──────────┴─────────────┘\n');
  }

  private showCreateProjectHelp(): void {
    console.log('\nCREATE PROJECT - Start new coordinated project\n');
    console.log('Usage:');
    console.log('  create project <name> <path>');
    console.log('  create project <name>              (prompts for path)\n');
    console.log('Name rules:');
    console.log('  Single-word names: no quotes needed');
    console.log('  Multi-word names:  wrap in single or double quotes\n');
    console.log('Path rules:');
    console.log('  .   Use the current directory (recommended: cd into your project first)');
    console.log('  ~/path/to/dir   Absolute or home-relative path');
    console.log('  The directory must already exist — ACT never creates directories.\n');
    console.log('Examples:');
    console.log("  >>: create project myapp .");
    console.log("  >>: create project 'cool project' .");
    console.log('  >>: create project myapp ~/projects/myapp');
    console.log('  >>: create project "My App" ~/projects/myapp\n');
    console.log('Best practice:');
    console.log('  Create your project directory first, cd into it, then run `act`.');
    console.log("  Then: create project myapp .\n");
    console.log('Notes:');
    console.log('  - Requires a default agent to be set for project decomposition');
    console.log('  - Use "continue project <name>" to resume paused projects\n');
  }

  private showImproveHelp(): void {
    console.log('\nIMPROVE - Surgical precision coordination analysis\n');
    console.log('Usage:');
    console.log('  improve <scope> [filters] [options]\n');
    console.log('Scopes:');
    console.log('  communication    Agent-to-agent communication effectiveness');
    console.log('  tools           Tool usage patterns and effectiveness');
    console.log('  assignments     Task assignment suitability');
    console.log('  conflicts       Conflict resolution participation');
    console.log('  collaboration   Team synergy analysis');
    console.log('  performance     Overall task execution effectiveness');
    console.log('  knowledge       Knowledge gaps and learning opportunities\n');
    console.log('Session Filters:');
    console.log('  -project <name>        Analyze specific project');
    console.log('  -brainstorm <name>     Analyze brainstorm session');
    console.log('  -roundtable <name>     Analyze roundtable discussion');
    console.log('  -experiment <name>     Analyze experiment session\n');
    console.log('Options:');
    console.log('  --agents <list>         Focus on specific agents (comma-separated)');
    console.log('  --session <id>          Specific coordination session ID');
    console.log('  --filter <good|bad|all> Quality filter (default: all)');
    console.log('  --output <format>       Output format (summary, detailed-report, recommendations, json, metrics)');
    console.log('  --focus <list>          Only analyze messages from listed agents');
    console.log('  --print <path>          Save analysis to markdown file\n');
    console.log('Examples:');
    console.log('  >>: improve communication -project todo-app');
    console.log('  >>: improve performance --agents claude_code_1,windsurf_main --filter bad');
    console.log('  >>: improve knowledge -roundtable architecture-review --print ./reports/knowledge-gaps.md\n');
    console.log('Notes:');
    console.log('  - Analysis is stored in PVM for future reference');
    console.log('  - Results help identify patterns and improvement opportunities');
    console.log('  - Can generate actionable recommendations for better coordination\n');
  }

  private showBrainstormHelp(): void {
    console.log('\nBRAINSTORM - Creative ideation session\n');
    console.log('Usage:');
    console.log('  brainstorm <topic> [--agents <list>]\n');
    console.log('Description:');
    console.log('  Starts an open-ended creative discussion between agents on a topic.');
    console.log('  Unlike structured sessions, this allows free-flowing idea generation');
    console.log('  without specific goals or task execution.\n');
    console.log('Parameters:');
    console.log('  <topic>          Topic for brainstorming (use quotes for multi-word)');
    console.log('  --agents <list>  Comma-separated list of participating agents (optional)\n');
    console.log('Examples:');
    console.log('  >>: brainstorm api-design');
    console.log('  >>: brainstorm "UI/UX improvements" --agents claude_code_1,windsurf_main\n');
    console.log('Features:');
    console.log('  ✓ No code execution - pure discussion');
    console.log('  ✓ Captured in PVM for improvement analysis');
    console.log('  ✓ Can be analyzed with: improve communication -brainstorm <name>\n');
    console.log('Output:');
    console.log('  Starting brainstorm session: "api-design"');
    console.log('  Participants: 3 agents');
    console.log('  Mode: Open discussion, no task execution\n');
  }

  private showExperimentHelp(): void {
    console.log('\nEXPERIMENT - Comparative testing session\n');
    console.log('Usage:');
    console.log('  experiment <name> [--agents <list>]          # Start experiment');
    console.log('  experiment -analyze <name>                   # Analyze results\n');
    console.log('Description:');
    console.log('  Runs multiple approaches to the same problem in parallel, then');
    console.log('  analyzes and compares the results to determine the best solution.\n');
    console.log('Parameters:');
    console.log('  <name>           Experiment name/identifier');
    console.log('  --agents <list>  Comma-separated list of participating agents (optional)\n');
    console.log('Examples:');
    console.log('  >>: experiment react-vs-vue');
    console.log('  >>: experiment "Database choices" --agents claude_code_1,claude_code_2\n');
    console.log('Analysis Example:');
    console.log('  >>: experiment -analyze react-vs-vue');
    console.log('  Analyzing experiment "react-vs-vue"...');
    console.log('  Default agent analyzing both implementations...');
    console.log('  Analysis:');
    console.log('  • React Implementation: Better TypeScript support (94% success)');
    console.log('  • Vue Implementation: Simpler API (87% success)');
    console.log('  Recommendation: Use React for consistency with existing codebase\n');
    console.log('Features:');
    console.log('  ✓ Parallel execution of different approaches');
    console.log('  ✓ Automated LLM-powered analysis');
    console.log('  ✓ Evidence-based recommendations');
    console.log('  ✓ Results stored in PVM for future reference\n');
  }

  private showRoundtableHelp(): void {
    console.log('\nROUND TABLE - Multi-agent structured discussion\n');
    console.log('Usage:');
    console.log('  roundtable <topic>                         # Standard roundtable');
    console.log('  roundtable <topic> --interactive           # With HITL controls\n');
    console.log('Description:');
    console.log('  Structured discussion between all connected agents on important');
    console.log('  decisions. Interactive mode allows human-in-the-loop control.\n');
    console.log('Parameters:');
    console.log('  <topic>          Discussion topic');
    console.log('  --interactive    Enable human-in-the-loop controls\n');
    console.log('Interactive Controls:');
    console.log('  pause            Pause discussion');
    console.log('  resume           Resume discussion');
    console.log('  select <agent>   Highlight agent\'s contribution');
    console.log('  edit <msg_id>    Edit a message');
    console.log('  delete <msg_id>  Remove message');
    console.log('  send "<msg>"     User contributes to discussion');
    console.log('  stop             End roundtable');
    console.log('  clean_up         Create summary and save');
    console.log('  wipe             Remove from PVM (destructive)\n');
    console.log('Examples:');
    console.log('  >>: roundtable architecture-decisions');
    console.log('  >>: roundtable "Database technology choice" --interactive\n');
    console.log('Interactive Example:');
    console.log('  >>: roundtable database-choice --interactive');
    console.log('  Starting roundtable: "database-choice"');
    console.log('  Mode: Interactive (HITL controls enabled)');
    console.log('  >>: send "What about scalability? We expect 10M+ users."');
    console.log('  >>: select claude_code_1');
    console.log('  >>: clean_up\n');
    console.log('Features:');
    console.log('  ✓ All agents participate by default');
    console.log('  ✓ Interactive mode for complex decisions');
    console.log('  ✓ Complete audit trail in PVM');
    console.log('  ✓ Summary generation and export capabilities\n');
  }

  private showPVMHelp(): void {
    console.log('\nPVM (PAIRed Vector Minutes) - Memory system commands\n');
    console.log('Description:');
    console.log('  PVM is ACT\'s intelligent memory system that stores and retrieves');
    console.log('  coordination patterns, agent performance data, and improvement insights.\n');
    console.log('Commands:');
    console.log('  pvm stats                    Show PVM statistics');
    console.log('  pvm search <query>           Search coordination history');
    console.log('  pvm profile <agent_id>       Show evidence-based agent profile');
    console.log('  pvm export <path>            Export PVM database');
    console.log('  pvm import <path>            Import PVM database\n');
    console.log('Examples:');
    console.log('  >>: pvm stats');
    console.log('  PVM Statistics:');
    console.log('    Total coordination events: 1,847');
    console.log('    Agent profiles: 5');
    console.log('    Projects: 12');
    console.log('    FLUX evaluations: 156\n');
    console.log('  >>: pvm profile claude_code_1');
    console.log('  Agent Profile: claude_code_1');
    console.log('  Performance: 94% success rate');
    console.log('  Specializations: Backend API (98%), Authentication (95%)');
    console.log('  Best Collaborators: windsurf_main (92% synergy)\n');
    console.log('Features:');
    console.log('  ✓ Evidence-based agent profiles (not self-reported)');
    console.log('  ✓ Semantic search across coordination history');
    console.log('  ✓ Automatic performance tracking');
    console.log('  ✓ Context-aware recommendations');
    console.log('  ✓ Always current (based on recent outcomes)\n');
  }
}
