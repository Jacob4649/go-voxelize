package main

import (
	"github.com/jblindsay/lidario"
)

// Chunk of a LAS file
type LASChunk struct {
	
	// Start of the LAS chunk
	start int

	// End of the LAS chunk
	end int
}

// Divides a LAS file into chunks for processing
func ChunkFile(file *lidario.LasFile, numChunks int) []*LASChunk {
	chunks := make([]*LASChunk, 0)

	chunkSize := file.Header.NumberPoints / numChunks

	for i := 1; i < numChunks; i++ { // iterate for one less than numChunks
		start := (i-1) * chunkSize
		chunks = append(chunks, &LASChunk{start: start, end: start + chunkSize})
	}

	// final chunk
	chunks = append(chunks, &LASChunk{start: (numChunks - 1) * chunkSize, end: file.Header.NumberPoints})

	return chunks
}

// Object that can voxelize a LAS file into an output format
type LASProcessor[T any] interface {

	// Processes a chunk of the given LAS file
	Process(inputFile *lidario.LasFile, chunk *LASChunk, voxelSize float64, output chan<- *T)

	// Gets the default state of the output object
	EmptyOutput() *T

	// Combines output objects
	CombineOutput(base *T, incoming *T) *T
}

// Distributes the provided chunks over the provided channel, then sends nil
func distributeChunks(chunks []*LASChunk, output chan<- *LASChunk, concurrency int) {
	for _, chunk := range chunks {
		output <- chunk
	}
	for i := 0; i < concurrency; i++ {
		output <- nil
	}
}

// Concurrently processes some chunks
func handleConcurrentProcess[T any](inputFile *lidario.LasFile, inputChunk <-chan *LASChunk, processor LASProcessor[T], voxelSize float64, output chan<- *T) {
	// for non nil input chunks
	for chunk := <- inputChunk; chunk != nil; chunk = <- inputChunk {
		processor.Process(inputFile, chunk, voxelSize, output)
	}
}

// Concurrently processes a LAS file into voxels in the specified output format
func ConcurrentProcess[T any](inputFile *lidario.LasFile, chunks []*LASChunk, processor LASProcessor[T], concurrency int, voxelSize float64) *T {
	output := processor.EmptyOutput()

	outputChannel := make(chan *T)

	chunkChannel := make(chan *LASChunk)

	// distribute chunks over chunk channel
	go distributeChunks(chunks, chunkChannel, concurrency)

	// start processing goroutines
	for i := 0; i < concurrency; i++ {
		go handleConcurrentProcess(inputFile, chunkChannel, processor, voxelSize, outputChannel)
	}

	// collect outputs, stop when all have been read
	for _, _ = range chunks {
		readOutput := <- outputChannel
		output = processor.CombineOutput(output, readOutput)
	}

	return output
}

// Processes a LAS file sequentially
func SequentialProcess[T any](inputFile *lidario.LasFile, chunks []*LASChunk, processor LASProcessor[T], voxelSize float64) *T {
	output := processor.EmptyOutput()

	for _, chunk := range chunks {
		channel := make(chan *T)
		processor.Process(inputFile, chunk, voxelSize, channel)
		sequentialOutput := <- channel
		output = processor.CombineOutput(output, sequentialOutput)
	}

	return output
}
