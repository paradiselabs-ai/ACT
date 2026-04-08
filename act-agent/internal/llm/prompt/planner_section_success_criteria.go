package prompt

// PlannerSectionSuccessCriteria returns the expandable Planner section
// describing how to write strong @success_criteria. Loaded on-demand
// via expand_prompt_section("success_criteria") when the Planner needs
// to write or repair task acceptance criteria.
func PlannerSectionSuccessCriteria() string {
	return `# Writing @success_criteria

@success_criteria is the contract Assurance scores against. It is the
single most important field in a CREATE_TASK because Assurance's 95%
gate is computed against it line by line. Weak criteria → weak validation
→ broken downstream work.

## Rules

1. **Each line is independently testable.** "The button works" is not
   a criterion. "Clicking the submit button POSTs to /api/orders and
   shows a confirmation toast" is.
2. **Prefer observable outcomes over implementation details.** "Uses
   useState" is brittle. "State persists across re-renders" is not.
3. **Include at least one failure case.** "Returns 401 on invalid
   token", "Throws ValidationError on empty input". Otherwise the
   agent will only test the happy path.
4. **3-7 lines is the right size.** Fewer than 3 → too vague to
   validate. More than 7 → you're describing implementation, not
   acceptance.
5. **No "etc." or "as appropriate" or open-ended language.** Assurance
   can't score those.

## Good example

` + "```" + `
@success_criteria
- 15-min access token expiry enforced server-side
- Refresh rotation invalidates the previous refresh token
- 401 returned on invalid or expired token
- Token payload includes user ID and role claim
- Tests cover happy path, expiry, and rotation
` + "```" + `

## Bad example (don't do this)

` + "```" + `
@success_criteria
- Auth works correctly
- Tests pass
- Code is clean
` + "```" + `

## Repairing weak criteria

If Assurance keeps failing tasks because criteria are unclear, the
problem is probably you, not the agent. Rewrite the criteria with
sharper observable outcomes and re-issue the task.`
}
