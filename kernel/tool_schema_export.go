package kernel

// BuildDecisionToolParameters exposes the decision tool JSON-schema parameters for external callers.
func BuildDecisionToolParameters(exitPlanID string) map[string]any {
	return buildDecisionToolParameters(exitPlanID)
}

// ParseDecisionToolArguments parses the tool-call arguments JSON into reasoning + decisions.
func ParseDecisionToolArguments(toolArgsJSON string) (string, []Decision, error) {
	return parseDecisionToolArguments(toolArgsJSON)
}
