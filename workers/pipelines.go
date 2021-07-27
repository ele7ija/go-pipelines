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

	pipeline := pipeApi.NewPipeline(getMetadataFilter, loadThumbnailFilter, loadFullFilter)
	return pipeline
}

// GetAllImagesPipeline should get all image metadata (before it starts) and load their thumbnails
func GetAllImagesPipeline(service ImageService) *pipeApi.Pipeline {

	loadThumbnailWorker := LoadThumbnailWorker{service}
	loadThumbnailFilter := pipeApi.NewParallelFilter(&loadThumbnailWorker)

	loadFullWorker := LoadFullWorker{service}
	loadFullFilter := pipeApi.NewParallelFilter(&loadFullWorker)

	pipeline := pipeApi.NewPipeline(loadThumbnailFilter, loadFullFilter)
	return pipeline
}