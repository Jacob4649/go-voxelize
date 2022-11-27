package lasProcessing

import (
	"io"

	"github.com/Jacob4649/go-voxelize/go-voxelize/lidarioMod"
)

// Chunk of a LAS file
type LASChunk struct {
	
	// Start of the LAS chunk
	Start int

	// End of the LAS chunk
	End int
}

// Gets the bytes for a LAS chunk
func(chunk *LASChunk) ReadOnFile(file *lidarioMod.LasFile) []byte {
	recordLength := file.Header.PointRecordLength

	offset := int64(file.Header.OffsetToPoints)

	pointOffset := int64(recordLength) * int64(chunk.Start)

	delta := int64(chunk.End) - int64(chunk.Start)

	chunkLength := delta * int64(file.Header.PointRecordLength)

	rawBytes := make([]byte, chunkLength)

	_, err := file.RawFile.ReadAt(rawBytes, offset + pointOffset)

	if err != nil && err != io.EOF {
		panic(err)
	}

	return rawBytes
}

// Divides a LAS file into chunks for processing
func ChunkFile(file *lidarioMod.LasFile, numChunks int) []*LASChunk {
	chunks := make([]*LASChunk, 0)

	chunkSize := file.Header.NumberPoints / numChunks

	for i := 1; i < numChunks; i++ { // iterate for one less than numChunks
		start := (i-1) * chunkSize
		chunks = append(chunks, &LASChunk{Start: start, End: start + chunkSize})
	}

	// final chunk
	chunks = append(chunks, &LASChunk{Start: (numChunks - 1) * chunkSize, End: file.Header.NumberPoints})

	return chunks
}

// Object that can voxelize a LAS file into an output format
type LASProcessor[T any] interface {

	// Processes a chunk of the given LAS file
	Process(inputFile *lidarioMod.LasFile, chunk *LASChunk, voxelSize float64, output chan<- *T, status *float64)

	// Gets the default state of the output object
	EmptyOutput(inputFile *lidarioMod.LasFile, voxelSize float64) *T

	// Combines output objects
	CombineOutput(base *T, incoming *T) *T
}

// Distributes the provided chunks over the provided channel, then sends nil
func distributeChunks(chunks []*LASChunk, output chan<- *LASChunk, concurrency int, status *ConcurrentStatus) {
	for i, chunk := range chunks {
		*(status.CurrentChunk) = i
		output <- chunk
	}
	*(status.CurrentChunk) = len(chunks)
	for i := 0; i < concurrency; i++ {
		output <- nil
	}
}

// Concurrently processes some chunks
func handleConcurrentProcess[T any](inputFile *lidarioMod.LasFile, inputChunk <-chan *LASChunk, processor LASProcessor[T], voxelSize float64, output chan<- *T, status *float64) {
	// for non nil input chunks
	for chunk := <- inputChunk; chunk != nil; chunk = <- inputChunk {
		processor.Process(inputFile, chunk, voxelSize, output, status)
	}
}

// Status of concurrent processing
type ConcurrentStatus struct {

	// Total number of chunks
	TotalChunks *int

	// Total number of concurrent processors
	Concurrency *int

	// First chunk not being processed on (or TotalChunks if all are being processed on)
	CurrentChunk *int

	// Progress on each currently processing chunk from 0% to 100% (0.0 - 1.0)
	ChunkProgress []*float64

	// Number of merges performed
	Merges *int

}

// Gets a new ConcurrentStatus
func NewConcurrentStatus() *ConcurrentStatus {

	totalChunks := 0

	concurrency := 0

	currentChunk := 0

	chunkProgress := []*float64{}

	merges := 0

	return &ConcurrentStatus{
		TotalChunks: &totalChunks, 
		Concurrency: &concurrency, 
		CurrentChunk: &currentChunk, 
		ChunkProgress: chunkProgress, 
		Merges: &merges}

}

// Concurrently processes a LAS file into voxels in the specified output format
func ConcurrentProcess[T any](inputFile *lidarioMod.LasFile, chunks []*LASChunk, processor LASProcessor[T], concurrency int, voxelSize float64, status *ConcurrentStatus) *T {
	
	if status == nil {
		status = NewConcurrentStatus()
	}

	*(status.TotalChunks) = len(chunks)

	*(status.Concurrency) = concurrency

	for i := 0; i < concurrency; i++ {
		percent := 0.0
		status.ChunkProgress = append(status.ChunkProgress, &percent)
	}


	output := processor.EmptyOutput(inputFile, voxelSize)

	outputChannel := make(chan *T)

	chunkChannel := make(chan *LASChunk)

	// distribute chunks over chunk channel
	go distributeChunks(chunks, chunkChannel, concurrency, status)

	// start processing goroutines
	for i := 0; i < concurrency; i++ {
		go handleConcurrentProcess(inputFile, chunkChannel, processor, voxelSize, outputChannel, status.ChunkProgress[i])
	}

	// collect outputs, stop when all have been read
	for _, _ = range chunks {
		readOutput := <- outputChannel
		output = processor.CombineOutput(output, readOutput)
		*(status.Merges) += 1
	}

	return output
}

// Processes a LAS file sequentially
func SequentialProcess[T any](inputFile *lidarioMod.LasFile, chunks []*LASChunk, processor LASProcessor[T], voxelSize float64, currentChunk *int, chunkProgress *float64) *T {
	
	if currentChunk == nil {
		x := 0
		currentChunk = &x
	}

	if chunkProgress == nil {
		x := 0.0
		chunkProgress = &x
	}
	
	output := processor.EmptyOutput(inputFile, voxelSize)

	*currentChunk = 0

	for _, chunk := range chunks {
		channel := make(chan *T)
		processor.Process(inputFile, chunk, voxelSize, channel, chunkProgress)
		sequentialOutput := <- channel
		output = processor.CombineOutput(output, sequentialOutput)
		*currentChunk += 1
	}

	return output
}
