package lasProcessing

// Status of a PostProcessingPipeline
type PipelineStatus struct {

	// Step name
	Step string;

	// Progress in the step
	Progress float64;

}

// Pipeline for post processing from input to output
type PostProcessingPipeline[I any, O any] interface {
	Process(input I, output chan<- PipelineStatus) O
}

// Monitorable post processing pipeline
type ChainedPipeline[I any, O any] struct {
	PostProcessingPipeline[I, O]

	// All the steps in the pipeline to be executed in
	Steps []*PostProcessingPipeline[any, any]

}

// Executes a monitorable pipeline
func(pipeline *ChainedPipeline[I, O]) Process(input I, output chan<- PipelineStatus) O {

	var currentInput any = input

	for _, step := range pipeline.Steps {
		currentInput = (*step).Process(currentInput, output)
	}

	return currentInput.(O)
}
