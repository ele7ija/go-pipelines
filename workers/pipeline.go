package workers

import (
	pipe "github.com/ele7ija/pipeline"
	"time"
)

func MakeGetImagePipeline(service ImageService) *pipe.Pipeline {
	getMetadataWorker := GetMetadataWorker{service}
	getMetadataFilter := pipe.NewParallelFilter(&getMetadataWorker)

	loadThumbnailWorker := LoadThumbnailWorker{service}
	loadThumbnailFilter := pipe.NewParallelFilter(&loadThumbnailWorker)

	loadFullWorker := LoadFullWorker{service}
	loadFullFilter := pipe.NewParallelFilter(&loadFullWorker)

	base64Encoder := Base64EncodeWorker{}
	base64EncoderFilter := pipe.NewParallelFilter(&base64Encoder)

	pipeline := pipe.NewPipeline("GetImagePipeline", getMetadataFilter, loadThumbnailFilter, loadFullFilter, base64EncoderFilter)
	pipeline.StartExtracting(5 * time.Second)
	return pipeline
}

func MakeGetAllImagesPipeline(service ImageService) *pipe.Pipeline {

	loadThumbnailWorker := LoadThumbnailWorker{service}
	loadThumbnailFilter := pipe.NewParallelFilter(&loadThumbnailWorker)

	base64Encoder := Base64EncodeWorker{}
	base64EncoderFilter := pipe.NewParallelFilter(&base64Encoder)

	pipeline := pipe.NewPipeline("GetAllImagesPipeline", loadThumbnailFilter, base64EncoderFilter)
	pipeline.StartExtracting(5 * time.Second)
	return pipeline
}

func MakeCreateImagesPipeline(service ImageService) *pipe.Pipeline {

	transformFHWorker := TransformFileHeaderWorker{}
	transformFHFilter := pipe.NewParallelFilter(&transformFHWorker)

	createThumbnailWorker := CreateThumbnailWorker{service}
	createThumbnailFilter := pipe.NewBoundedParallelFilter(60, &createThumbnailWorker)

	persistWorker := PersistWorker{service}
	persistFilter := pipe.NewBoundedParallelFilter(30, &persistWorker)

	saveMetadataWorker := SaveMetadataWorker{service}
	saveMetadataFilter := pipe.NewBoundedParallelFilter(45, &saveMetadataWorker)

	base64Encoder := Base64EncodeWorker{}
	base64EncoderFilter := pipe.NewParallelFilter(&base64Encoder)

	pipeline := pipe.NewPipeline("CreateImagesPipeline", transformFHFilter, createThumbnailFilter, persistFilter, saveMetadataFilter, base64EncoderFilter)
	pipeline.StartExtracting(5 * time.Second)
	return pipeline
}
