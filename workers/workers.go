package workers

import (
	"fmt"
	"github.com/nfnt/resize"
	"image"
	"image/jpeg"
	"io/ioutil"
	"log"
	"os"
)

var (
	ThumbnailWidth  = uint(200)
	ThumbnailHeigth = uint(200)
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

	CreateThumbnail(image *Image) error
	Persist(image *Image) error
	PersistMetadata(image *Image) error
}

type ImageService struct {
}

func NewImageService() *ImageService {

	return &ImageService{}
}

func (i *ImageService) CreateThumbnail(image *Image) error {

	image.Thumbnail = resize.Resize(ThumbnailWidth, ThumbnailHeigth, image.Full, resize.Lanczos3)
	return nil
}

func (i *ImageService) Persist(image *Image) error {

	// Save full image
	fullImageFile, err := ioutil.TempFile(os.TempDir(), "pipelineImg*.jpg")
	if err != nil {
		return fmt.Errorf("error creating temp file: %s", err)
	}
	if err := jpeg.Encode(fullImageFile, image.Full, nil); err != nil {
		_ = os.Remove(fullImageFile.Name())
		return fmt.Errorf("error while saving full image: %s", err)
	}

	// Save thumbnail
	thumbnailImageFile, err := ioutil.TempFile(os.TempDir(), "pipelineImgThumb*.jpg")
	if err != nil {
		_ = os.Remove(fullImageFile.Name())
		return fmt.Errorf("error creating temp file: %s", err)
	}
	if err := jpeg.Encode(thumbnailImageFile, image.Thumbnail, nil); err != nil {
		_ = os.Remove(fullImageFile.Name())
		_ = os.Remove(thumbnailImageFile.Name())
		return fmt.Errorf("error while saving thumbnail: %s", err)
	}

	image.FullPath = fullImageFile.Name()
	log.Printf("saved an image to: %s\n", image.FullPath)
	image.ThumbnailPath = thumbnailImageFile.Name()
	log.Printf("saved a thumbnail to: %s\n", image.ThumbnailPath)
	return nil
}

func (i *ImageService) PersistMetadata(image Image) (Image, error) {
	panic("implement me")
}
