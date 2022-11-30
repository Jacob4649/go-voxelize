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
	Process(input I, output *PipelineStatus) O
}

// Monitorable post processing pipeline
type chainedPipeline[I any, O any] struct {
	// All the steps in the pipeline to be executed in
	callChain func(input I, output *PipelineStatus) O
}

// Chains some monitorable pipelines
func ChainPipeline[I any, C any, O any](firstPipeline PostProcessingPipeline[I, C], secondPipeline PostProcessingPipeline[C, O]) PostProcessingPipeline[I, O] {
	return &chainedPipeline[I, O]{callChain: func(input I, output *PipelineStatus) O {
		return secondPipeline.Process(firstPipeline.Process(input, output), output)
	}}
}

// Executes a monitorable pipeline
func(pipeline *chainedPipeline[I, O]) Process(input I, output *PipelineStatus) O {
	return pipeline.callChain(input, output)
}

// Processes with the specified pipeline
func ProcessWithPipeline[I any, O any](input I, pipeline PostProcessingPipeline[I, O], status *PipelineStatus) O {
	
	if status == nil {
		status = &PipelineStatus{}
	}
	
	output := pipeline.Process(input, status)

	status.Step = ""

	return output
}
