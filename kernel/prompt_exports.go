package kernel

// BuildDecisionAnalysisSystemPrompt returns the analysis-layer system prompt used in the pre-analysis stage.
func (e *StrategyEngine) BuildDecisionAnalysisSystemPrompt() string {
	return e.buildDecisionAnalysisSystemPrompt()
}

// BuildVisionSystemPrompt returns the vision (chart-reading) stage system prompt.
func (e *StrategyEngine) BuildVisionSystemPrompt() string {
	return e.buildVisionSystemPrompt()
}

