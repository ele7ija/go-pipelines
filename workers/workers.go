package workers

import (
	"github.com/nfnt/resize"
	"image"
)

type Image struct {

	Full image.Image
	FullPath string
	Thumbnail image.Image
	ThumbnailPath string

}

func NewImage(fullImage image.Image) *Image {
	return &Image{
		Full: fullImage,
	}
}

type ImageServiceI interface {

	CreateThumbnail(image *Image) (*Image, error)
	Persist(image *Image) (*Image, error)
	PersistMetadata(image *Image) (*Image, error)

}

type ImageService struct {

}

func (i *ImageService) CreateThumbnail(image *Image) (*Image, error) {

	image.Thumbnail = resize.Resize(200, 200, image.Full, resize.Lanczos3)
	return image, nil
}

func (i *ImageService) Persist(image Image) (Image, error) {
	panic("implement me")
	// err = jpeg.Encode(someWriter, image.Image, nil)
}

func (i *ImageService) PersistMetadata(image Image) (Image, error) {
	panic("implement me")
}
