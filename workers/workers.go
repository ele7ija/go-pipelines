package workers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"fmt"
	pipeApi "github.com/ele7ija/go-pipelines/internal"
	"github.com/nfnt/resize"
	"image"
	"image/jpeg"
	"io/ioutil"
	"log"
	"mime/multipart"
	"os"
	"sync"
)

const (
	ThumbnailWidth  = uint(200)
	ThumbnailHeight = uint(200)
)

type Image struct {

	Id int					`json:"id,omitempty"`
	Name string				`json:"name"`
	Full image.Image 		`json:"full,omitempty"`
	FullPath string			`json:"fullPath"`
	Thumbnail image.Image	`json:"thumbnail,omitempty"`
	ThumbnailPath string	`json:"thumbnailPath"`
}

type ImageBase64 struct {

	Id int					`json:"id,omitempty"`
	Name string				`json:"name"`
	FullBase64 string		`json:"fullBase64,omitempty"`
	FullPath string			`json:"fullPath"`
	ThumbnailBase64 string	`json:"thumbnailBase64,omitempty"`
	ThumbnailPath string	`json:"thumbnailPath"`
}

func NewImage(name string, fullImage image.Image) *Image {
	return &Image{
		Name: name,
		Full: fullImage,
	}
}

func NewImageBase64(img *Image) *ImageBase64 {

	fullBase64Encoding := ""
	if img.Full != nil {
		fullBuff := new(bytes.Buffer)
		jpeg.Encode(fullBuff, img.Full, nil)
		fullBase64Encoding = "data:image/jpeg;base64,"
		fullBase64Encoding += base64.StdEncoding.EncodeToString(fullBuff.Bytes())
	}

	thumbBase64Encoding := ""
	if img.Thumbnail != nil {
		thumbBuff := new(bytes.Buffer)
		jpeg.Encode(thumbBuff, img.Thumbnail, nil)
		thumbBase64Encoding = "data:image/jpeg;base64,"
		thumbBase64Encoding += base64.StdEncoding.EncodeToString(thumbBuff.Bytes())
	}

	return &ImageBase64{
		Id:				 img.Id,
		Name:            img.Name,
		FullBase64:		 fullBase64Encoding,
		FullPath:        img.FullPath,
		ThumbnailBase64: thumbBase64Encoding,
		ThumbnailPath:   img.ThumbnailPath,
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

	var imageId int
	err = tx.QueryRowContext(ctx, "INSERT INTO image (name, fullpath, thumbnailpath) VALUES( $1, $2, $3 ) RETURNING id", image.Name, image.FullPath, image.ThumbnailPath).Scan(&imageId)
	if err != nil {
		return
	}
	image.Id = imageId

	if _, err = tx.Exec("INSERT INTO user_images (user_id, image_id) VALUES ($1, $2)", ctx.Value("userId"), imageId); err != nil {
		return
	}

	log.Printf("saved metadata for image: %s", image.Name)
	return
}

func (i *ImageServiceImpl) GetAllMetadata(ctx context.Context) (<-chan *Image, <-chan error, error) {

	userId := ctx.Value("userId").(int)
	rows, err := i.db.QueryContext(ctx, "SELECT image_id FROM user_images WHERE user_id = $1", userId)
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
	log.Printf("Got %d image ids to process", len(imageIds))

	images := make(chan *Image, len(imageIds))
	errors := make(chan error, len(imageIds))
	wg := sync.WaitGroup{}
	wg.Add(len(imageIds))
	for _, imageId := range imageIds {

		go func(imageIdTemp int) {
			img, err := i.GetMetadata(ctx, imageIdTemp)
			if err != nil {
				errors <- err
				log.Printf("Error processing image with id: %d", imageIdTemp)
			} else {
				images <- img
				log.Printf("Processed successfully image with id: %d", imageIdTemp)
			}
			wg.Done()
		}(imageId)
	}

	go func() {
		wg.Wait()
		close(images)
		close(errors)
		log.Printf("Done processing all %d images", len(imageIds))
	}()

	return images, errors, nil
}

func (i *ImageServiceImpl) GetMetadata(ctx context.Context, imageId int) (*Image, error) {

	row := i.db.QueryRowContext(ctx, "SELECT name, fullpath, thumbnailpath FROM image WHERE id = $1;", imageId)
	if err := row.Err(); err != nil {
		return nil, err
	}

	img := Image{Id: imageId}
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

type Base64EncodeWorker struct {
}

func (worker *Base64EncodeWorker) Work(ctx context.Context, in pipeApi.Item) (out pipeApi.Item, err error) {

	if in.NumberOfParts() != 1 {
		return nil, fmt.Errorf("incorrect input parameters length: %d", in.NumberOfParts())
	}

	var img *Image
	var ok bool
	if img, ok = in.GetPart(0).(*Image); ok == false {
		return nil, fmt.Errorf("incorrect input parameter")
	}

	imgBase64 := NewImageBase64(img)
	return pipeApi.NewGenericItem(imgBase64), err
}

type CreateThumbnailWorker struct {
	ImageService
}

func (worker *CreateThumbnailWorker) Work(ctx context.Context, in pipeApi.Item) (out pipeApi.Item, err error)  {

	if in.NumberOfParts() != 1 {
		return nil, fmt.Errorf("incorrect input parameters length: %d", in.NumberOfParts())
	}

	var img *Image
	var ok bool
	if img, ok = in.GetPart(0).(*Image); ok == false {
		return nil, fmt.Errorf("incorrect input parameter")
	}

	err = worker.CreateThumbnail(ctx, img)
	return pipeApi.NewGenericItem(img), err
}

type PersistWorker struct {
	ImageService
}

func (worker *PersistWorker) Work(ctx context.Context, in pipeApi.Item) (out pipeApi.Item, err error)  {

	if in.NumberOfParts() != 1 {
		return nil, fmt.Errorf("incorrect input parameters length: %d", in.NumberOfParts())
	}

	var img *Image
	var ok bool
	if img, ok = in.GetPart(0).(*Image); ok == false {
		return nil, fmt.Errorf("incorrect input parameter")
	}

	err = worker.Persist(ctx, img)
	return pipeApi.NewGenericItem(img), err
}

type SaveMetadataWorker struct {
	ImageService
}

func (worker *SaveMetadataWorker) Work(ctx context.Context, in pipeApi.Item) (out pipeApi.Item, err error)  {

	if in.NumberOfParts() != 1 {
		return nil, fmt.Errorf("incorrect input parameters length: %d", in.NumberOfParts())
	}

	var img *Image
	var ok bool
	if img, ok = in.GetPart(0).(*Image); ok == false {
		return nil, fmt.Errorf("incorrect input parameter")
	}

	err = worker.SaveMetadata(ctx, img)
	return pipeApi.NewGenericItem(img), err
}

type RemoveFullImageWorker struct {
}

func (worker *RemoveFullImageWorker) Work(ctx context.Context, in pipeApi.Item) (out pipeApi.Item, err error)  {

	if in.NumberOfParts() != 1 {
		return nil, fmt.Errorf("incorrect input parameters length: %d", in.NumberOfParts())
	}

	var img *Image
	var ok bool
	if img, ok = in.GetPart(0).(*Image); ok == false {
		return nil, fmt.Errorf("incorrect input parameter")
	}

	img.Full = nil
	return pipeApi.NewGenericItem(img), err
}

type TransformFileHeaderWorker struct {
}

func (worker *TransformFileHeaderWorker) Work(ctx context.Context, in pipeApi.Item) (out pipeApi.Item, err error)  {

	if in.NumberOfParts() != 1 {
		return nil, fmt.Errorf("incorrect input parameters length: %d", in.NumberOfParts())
	}

	var fh *multipart.FileHeader
	var ok bool
	if fh, ok = in.GetPart(0).(*multipart.FileHeader); ok == false {
		return nil, fmt.Errorf("incorrect input parameter")
	}

	f, err := fh.Open()
	rawimg, err := jpeg.Decode(f)
	img := NewImage(fh.Filename, rawimg)
	return pipeApi.NewGenericItem(img), err
}