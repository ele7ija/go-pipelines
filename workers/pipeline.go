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

func MakeCreateImagesPipelineBoundedFilters(service ImageService) *pipe.Pipeline {

	transformFHWorker := TransformFileHeaderWorker{}
	transformFHFilter := pipe.NewBoundedParallelFilter(30, &transformFHWorker)

	createThumbnailWorker := CreateThumbnailWorker{service}
	createThumbnailFilter := pipe.NewBoundedParallelFilter(35, &createThumbnailWorker)

	persistWorker := PersistWorker{service}
	persistFilter := pipe.NewBoundedParallelFilter(40, &persistWorker)

	saveMetadataWorker := SaveMetadataWorker{service}
	saveMetadataFilter := pipe.NewBoundedParallelFilter(10, &saveMetadataWorker)

	base64Encoder := Base64EncodeWorker{}
	base64EncoderFilter := pipe.NewBoundedParallelFilter(40, &base64Encoder)

	pipeline := pipe.NewPipeline("CreateImagesPipelineBounded3035401040", transformFHFilter, createThumbnailFilter, persistFilter, saveMetadataFilter, base64EncoderFilter)
	pipeline.StartExtracting(5 * time.Second)
	return pipeline
}

func MakeCreateImagesPipeline1Transform1Filter(service ImageService) *pipe.Pipeline {

	transformFHWorker := TransformFileHeaderWorker{}
	transformFHFilter := pipe.NewParallelFilter(&transformFHWorker)

	createThumbnailWorker := CreateThumbnailWorker{service}
	createThumbnailFilter := pipe.NewParallelFilter(&createThumbnailWorker)

	persistWorker := PersistWorker{service}
	persistFilter := pipe.NewParallelFilter(&persistWorker)

	saveMetadataWorker := SaveMetadataWorker{service}
	saveMetadataFilter := pipe.NewParallelFilter(&saveMetadataWorker)

	base64Encoder := Base64EncodeWorker{}
	base64EncoderFilter := pipe.NewParallelFilter(&base64Encoder)

	pipeline := pipe.NewPipeline("CreateImagesPipeline1Transform1Filter", transformFHFilter, createThumbnailFilter, persistFilter, saveMetadataFilter, base64EncoderFilter)
	pipeline.StartExtracting(5 * time.Second)
	return pipeline
}

func MakeCreateImagesPipelineNTransform1Filter(service ImageService) *pipe.Pipeline {

	transformFHWorker := TransformFileHeaderWorker{}
	createThumbnailWorker := CreateThumbnailWorker{service}
	persistWorker := PersistWorker{service}
	saveMetadataWorker := SaveMetadataWorker{service}
	base64Encoder := Base64EncodeWorker{}

	filter := pipe.NewParallelFilter(&transformFHWorker, &createThumbnailWorker, &persistWorker, &saveMetadataWorker, &base64Encoder)

	pipeline := pipe.NewPipeline("CreateImagesPipelineNTransform1Filter", filter)
	pipeline.StartExtracting(5 * time.Second)
	return pipeline
}
