package workers

import (
pipeApi "github.com/ele7ija/go-pipelines/internal"
	"time"
)

func MakeGetImagePipeline(service ImageService) *pipeApi.Pipeline {
	getMetadataWorker := GetMetadataWorker{service}
	getMetadataFilter := pipeApi.NewParallelFilter(&getMetadataWorker)

	loadThumbnailWorker := LoadThumbnailWorker{service}
	loadThumbnailFilter := pipeApi.NewParallelFilter(&loadThumbnailWorker)

	loadFullWorker := LoadFullWorker{service}
	loadFullFilter := pipeApi.NewParallelFilter(&loadFullWorker)

	base64Encoder := Base64EncodeWorker{}
	base64EncoderFilter := pipeApi.NewParallelFilter(&base64Encoder)

	pipeline := pipeApi.NewPipeline("GetImagePipeline", getMetadataFilter, loadThumbnailFilter, loadFullFilter, base64EncoderFilter)
	pipeline.StartExtracting(5 * time.Second)
	return pipeline
}

func MakeGetAllImagesPipeline(service ImageService) *pipeApi.Pipeline {

	loadThumbnailWorker := LoadThumbnailWorker{service}
	loadThumbnailFilter := pipeApi.NewParallelFilter(&loadThumbnailWorker)

	base64Encoder := Base64EncodeWorker{}
	base64EncoderFilter := pipeApi.NewParallelFilter(&base64Encoder)

	pipeline := pipeApi.NewPipeline("GetAllImagesPipeline", loadThumbnailFilter, base64EncoderFilter)
	pipeline.StartExtracting(5 * time.Second)
	return pipeline
}

func MakeCreateImagesPipeline(service ImageService) *pipeApi.Pipeline {

	transformFHWorker := TransformFileHeaderWorker{}
	transformFHFilter := pipeApi.NewParallelFilter(&transformFHWorker)

	createThumbnailWorker := CreateThumbnailWorker{service}
	createThumbnailFilter := pipeApi.NewBoundedParallelFilter(60, &createThumbnailWorker)

	persistWorker := PersistWorker{service}
	persistFilter := pipeApi.NewBoundedParallelFilter(30, &persistWorker)

	saveMetadataWorker := SaveMetadataWorker{service}
	saveMetadataFilter := pipeApi.NewBoundedParallelFilter(45, &saveMetadataWorker)

	base64Encoder := Base64EncodeWorker{}
	base64EncoderFilter := pipeApi.NewParallelFilter(&base64Encoder)

	pipeline := pipeApi.NewPipeline("CreateImagesPipeline", transformFHFilter, createThumbnailFilter, persistFilter, saveMetadataFilter, base64EncoderFilter)
	pipeline.StartExtracting(5 * time.Second)
	return pipeline
}