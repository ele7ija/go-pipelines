package workers

import (
	"context"
	"database/sql"
	"fmt"
	pipeApi "github.com/ele7ija/go-pipelines/internal"
	"github.com/nfnt/resize"
	"image"
	"image/jpeg"
	"io/ioutil"
	"log"
	"os"
	"sync"
)

var (
	ThumbnailWidth  = uint(200)
	ThumbnailHeight = uint(200)
)

type Image struct {

	Name string
	Full image.Image 		`json:"omitempty"`
	FullPath string
	Thumbnail image.Image
	ThumbnailPath string
}

func NewImage(name string, fullImage image.Image) *Image {
	return &Image{
		Name: name,
		Full: fullImage,
	}
}

type ImageService interface {

	CreateThumbnail(ctx context.Context, image *Image) error
	Persist(ctx context.Context, image *Image) error
	SaveMetadata(ctx context.Context, image *Image) error
	GetAllMetadata(ctx context.Context) (<-chan *Image, <-chan error, error)
	GetMetadata(ctx context.Context, imageId int) (*Image, error)
	LoadThumbnail(ctx context.Context, img *Image) error
	LoadFull(ctx context.Context, img *Image) error
}

type ImageServiceImpl struct {

	db  *sql.DB
}

func NewImageService(db *sql.DB) *ImageServiceImpl {

	return &ImageServiceImpl{db}
}

func (i *ImageServiceImpl) CreateThumbnail(ctx context.Context, image *Image) error {

	image.Thumbnail = resize.Resize(ThumbnailWidth, ThumbnailHeight, image.Full, resize.Lanczos3)
	return nil
}

func (i *ImageServiceImpl) Persist(ctx context.Context, image *Image) error {

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

func (i *ImageServiceImpl) SaveMetadata(ctx context.Context, image *Image) (err error) {

	tx, err := i.db.Begin()
	if err != nil {
		return
	}

	defer func() {
		switch err {
		case nil:
			err = tx.Commit()
		default:
			tx.Rollback()
		}
	}()

	res, err := tx.Exec("INSERT INTO image(name, fullpath, thumbnailpath) VALUES( ?, ?, ? )", image.Name, image.FullPath, image.ThumbnailPath)
	if err != nil {
		return
	}
	imageId, err := res.LastInsertId()
	if err != nil {
		return
	}

	if _, err = tx.Exec("INSERT INTO user_images(user_id, image_id) VALUES (?, ?)", ctx.Value("userId"), imageId); err != nil {
		return
	}

	log.Printf("saved metadata for image: %s", image.Name)
	return
}

func (i *ImageServiceImpl) GetAllMetadata(ctx context.Context) (<-chan *Image, <-chan error, error) {

	userId := ctx.Value("userId").(int)
	rows, err := i.db.QueryContext(ctx, "SELECT image_id FROM user_images WHERE user_id = ?", userId)
	if err != nil {
		return nil, nil, err
	}

	var imageIds []int
	for rows.Next() {

		var imageId int
		err := rows.Scan(&imageId)
		if err != nil {
			return nil, nil, err
		}
		imageIds = append(imageIds, imageId)
	}
	rows.Close()

	images := make(chan *Image, len(imageIds))
	errors := make(chan error, len(imageIds))
	wg := sync.WaitGroup{}
	wg.Add(len(imageIds))
	for imageId := range imageIds {

		go func(imageIdTemp int) {
			img, err := i.GetMetadata(ctx, imageIdTemp)
			if err != nil {
				errors <- err
			} else {
				images <- img
			}
			wg.Done()
		}(imageId)
	}

	go func() {
		wg.Wait()
		close(images)
		close(errors)
	}()

	return images, errors, nil
}

func (i *ImageServiceImpl) GetMetadata(ctx context.Context, imageId int) (*Image, error) {

	row := i.db.QueryRowContext(ctx, "SELECT name, fullpath, thumbnailpath FROM image WHERE image_id = ?", imageId)
	if err := row.Err(); err != nil {
		return nil, err
	}

	img := Image{}
	err := row.Scan(&img.Name, &img.FullPath, &img.ThumbnailPath)
	if err != nil {
		return nil, err
	}

	return &img, nil
}

func (i *ImageServiceImpl) LoadThumbnail(ctx context.Context, img *Image)  error {

	if img.ThumbnailPath == "" {
		return fmt.Errorf("thumbnail path does not exist")
	}

	select {
	case <-ctx.Done():
		return fmt.Errorf("context done")
	default:
	}

 	f, err := os.Open(img.ThumbnailPath)
 	if err != nil {
 		return err
	}
	thumbnail, _, err := image.Decode(f)
	if err != nil {
		return err
	}
	img.Thumbnail = thumbnail

	return nil
}

func (i *ImageServiceImpl) LoadFull(ctx context.Context, img *Image)  error {

	if img.FullPath == "" {
		return fmt.Errorf("full image path does not exist")
	}

	select {
	case <-ctx.Done():
		return fmt.Errorf("context done")
	default:
	}

	f, err := os.Open(img.FullPath)
	if err != nil {
		return err
	}
	full, _, err := image.Decode(f)
	if err != nil {
		return err
	}
	img.Full = full

	return nil
}

// WORKERS

type GetMetadataWorker struct {
	ImageService
}

// Work expects the input item to have one part - the id of the image
func (worker *GetMetadataWorker) Work(ctx context.Context, in pipeApi.Item) (pipeApi.Item, error)  {

	if in.NumberOfParts() != 1 {
		return nil, fmt.Errorf("incorrect input parameters length: %d", in.NumberOfParts())
	}

	var imageId int
	var ok bool
	if imageId, ok = in.GetPart(0).(int); ok == false {
		return nil, fmt.Errorf("incorrect input parameter")
	}

	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("context done")
	default:
	}

	img, err := worker.GetMetadata(ctx, imageId)
	return pipeApi.NewGenericItem(img), err
}

type LoadThumbnailWorker struct {
	ImageService
}

func (worker *LoadThumbnailWorker) Work(ctx context.Context, in pipeApi.Item) (out pipeApi.Item, err error)  {

	if in.NumberOfParts() != 1 {
		return nil, fmt.Errorf("incorrect input parameters length: %d", in.NumberOfParts())
	}

	var img *Image
	var ok bool
	if img, ok = in.GetPart(0).(*Image); ok == false {
		return nil, fmt.Errorf("incorrect input parameter")
	}

	err = worker.LoadThumbnail(ctx, img)
	return pipeApi.NewGenericItem(img), err
}

type LoadFullWorker struct {
	ImageService
}

func (worker *LoadFullWorker) Work(ctx context.Context, in pipeApi.Item) (out pipeApi.Item, err error)  {

	if in.NumberOfParts() != 1 {
		return nil, fmt.Errorf("incorrect input parameters length: %d", in.NumberOfParts())
	}

	var img *Image
	var ok bool
	if img, ok = in.GetPart(0).(*Image); ok == false {
		return nil, fmt.Errorf("incorrect input parameter")
	}

	err = worker.LoadFull(ctx, img)
	return pipeApi.NewGenericItem(img), err
}