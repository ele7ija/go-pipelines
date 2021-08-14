package workers

import (
	pipeApi "github.com/ele7ija/go-pipelines/internal"
)

// GetImagePipeline should get image metadata and load its thumbnail and full image
func GetImagePipeline(service ImageService) *pipeApi.Pipeline {

	getMetadataWorker := GetMetadataWorker{service}
	getMetadataFilter := pipeApi.NewParallelFilter(&getMetadataWorker)

	loadThumbnailWorker := LoadThumbnailWorker{service}
	loadThumbnailFilter := pipeApi.NewParallelFilter(&loadThumbnailWorker)

	loadFullWorker := LoadFullWorker{service}
	loadFullFilter := pipeApi.NewParallelFilter(&loadFullWorker)

	base64Encoder := Base64EncodeWorker{}
	base64EncoderFilter := pipeApi.NewParallelFilter(&base64Encoder)

	pipeline := pipeApi.NewPipeline(getMetadataFilter, loadThumbnailFilter, loadFullFilter, base64EncoderFilter)
	return pipeline
}

// GetAllImagesPipeline should get all image metadata (before it starts) and load their thumbnails
func GetAllImagesPipeline(service ImageService) *pipeApi.Pipeline {

	loadThumbnailWorker := LoadThumbnailWorker{service}
	loadThumbnailFilter := pipeApi.NewParallelFilter(&loadThumbnailWorker)

	base64Encoder := Base64EncodeWorker{}
	base64EncoderFilter := pipeApi.NewParallelFilter(&base64Encoder)

	pipeline := pipeApi.NewPipeline(loadThumbnailFilter, base64EncoderFilter)
	return pipeline
}

// GetCreateImagesPipeline should create all the images
func GetCreateImagesPipeline(service ImageService) *pipeApi.Pipeline {

	transformFHWorker := TransformFileHeaderWorker{}
	transformFHFilter := pipeApi.NewParallelFilter(&transformFHWorker)

	createThumbnailWorker := CreateThumbnailWorker{service}
	createThumbnailFilter := pipeApi.NewParallelFilter(&createThumbnailWorker)

	persistWorker := PersistWorker{service}
	persistFilter := pipeApi.NewParallelFilter(&persistWorker)

	saveMetadataWorker := SaveMetadataWorker{service}
	saveMetadataFilter := pipeApi.NewParallelFilter(&saveMetadataWorker)

	base64Encoder := Base64EncodeWorker{}
	base64EncoderFilter := pipeApi.NewParallelFilter(&base64Encoder)

	pipeline := pipeApi.NewPipeline(transformFHFilter, createThumbnailFilter, persistFilter, saveMetadataFilter, base64EncoderFilter)
	return pipeline
}