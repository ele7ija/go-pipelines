package workers

import (
	"context"
	"fmt"
	pipeApi "github.com/ele7ija/go-pipelines/internal"
	"image/jpeg"
	"mime/multipart"
)

type GetMetadataWorker struct {
	ImageService
}

// Work expects the input item to have one part - the id of the image
func (worker *GetMetadataWorker) Work(ctx context.Context, in pipeApi.Item) (pipeApi.Item, error)  {

	var imageId int
	var ok bool
	if imageId, ok = in.(int); ok == false {
		return nil, fmt.Errorf("incorrect input parameter")
	}

	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("context done")
	default:
	}

	imgs, err := worker.GetMetadata(ctx, []int{imageId})
	if err != nil {
		return nil, err
	}
	return imgs[0], nil
}

type LoadThumbnailWorker struct {
	ImageService
}

func (worker *LoadThumbnailWorker) Work(ctx context.Context, in pipeApi.Item) (out pipeApi.Item, err error)  {

	var img *Image
	var ok bool
	if img, ok = in.(*Image); ok == false {
		return nil, fmt.Errorf("incorrect input parameter")
	}

	err = worker.LoadThumbnail(ctx, img)
	return img, err
}

type LoadFullWorker struct {
	ImageService
}

func (worker *LoadFullWorker) Work(ctx context.Context, in pipeApi.Item) (out pipeApi.Item, err error)  {

	var img *Image
	var ok bool
	if img, ok = in.(*Image); ok == false {
		return nil, fmt.Errorf("incorrect input parameter")
	}

	err = worker.LoadFull(ctx, img)
	return img, err
}

type Base64EncodeWorker struct {
}

func (worker *Base64EncodeWorker) Work(ctx context.Context, in pipeApi.Item) (out pipeApi.Item, err error) {

	var img *Image
	var ok bool
	if img, ok = in.(*Image); ok == false {
		return nil, fmt.Errorf("incorrect input parameter")
	}

	imgBase64 := NewImageBase64(img)
	return imgBase64, err
}

type CreateThumbnailWorker struct {
	ImageService
}

func (worker *CreateThumbnailWorker) Work(ctx context.Context, in pipeApi.Item) (out pipeApi.Item, err error)  {

	var img *Image
	var ok bool
	if img, ok = in.(*Image); ok == false {
		return nil, fmt.Errorf("incorrect input parameter")
	}

	err = worker.CreateThumbnail(ctx, img)
	return img, err
}

type PersistWorker struct {
	ImageService
}

func (worker *PersistWorker) Work(ctx context.Context, in pipeApi.Item) (out pipeApi.Item, err error)  {

	var img *Image
	var ok bool
	if img, ok = in.(*Image); ok == false {
		return nil, fmt.Errorf("incorrect input parameter")
	}

	err = worker.Persist(ctx, img)
	return img, err
}

type SaveMetadataWorker struct {
	ImageService
}

func (worker *SaveMetadataWorker) Work(ctx context.Context, in pipeApi.Item) (out pipeApi.Item, err error)  {

	var img *Image
	var ok bool
	if img, ok = in.(*Image); ok == false {
		return nil, fmt.Errorf("incorrect input parameter")
	}

	err = worker.SaveMetadata(ctx, img)
	return img, err
}

type RemoveFullImageWorker struct {
}

func (worker *RemoveFullImageWorker) Work(ctx context.Context, in pipeApi.Item) (out pipeApi.Item, err error)  {

	var img *Image
	var ok bool
	if img, ok = in.(*Image); ok == false {
		return nil, fmt.Errorf("incorrect input parameter")
	}

	img.Full = nil
	return img, err
}

type TransformFileHeaderWorker struct {
}

func (worker *TransformFileHeaderWorker) Work(ctx context.Context, in pipeApi.Item) (out pipeApi.Item, err error)  {

	var fh *multipart.FileHeader
	var ok bool
	if fh, ok = in.(*multipart.FileHeader); ok == false {
		return nil, fmt.Errorf("incorrect input parameter")
	}

	f, err := fh.Open()
	if err != nil {
		return nil, err
	}
	rawimg, err := jpeg.Decode(f)
	if err != nil {
		return nil, err
	}
	img := NewImage(fh.Filename, rawimg)
	return img, err
}