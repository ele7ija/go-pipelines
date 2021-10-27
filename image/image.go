package image

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"fmt"
	"github.com/nfnt/resize"
	log "github.com/sirupsen/logrus"
	"image"
	"image/jpeg"
	"io/ioutil"
	"math"
	"os"
	"sync"
)

const (
	ThumbnailWidth  = uint(200)
	ThumbnailHeight = uint(200)
)

type Image struct {
	Id            int         `json:"id,omitempty"`
	Name          string      `json:"name"`
	Full          image.Image `json:"full,omitempty"`
	FullPath      string      `json:"fullPath"`
	Resolution    image.Point `json:"resolution,omitempty"`
	Thumbnail     image.Image `json:"thumbnail,omitempty"`
	ThumbnailPath string      `json:"thumbnailPath"`
}

type ImageBase64 struct {
	Id              int         `json:"id,omitempty"`
	Name            string      `json:"name"`
	FullBase64      string      `json:"fullBase64,omitempty"`
	FullPath        string      `json:"fullPath"`
	Resolution      image.Point `json:"resolution,omitempty"`
	ThumbnailBase64 string      `json:"thumbnailBase64,omitempty"`
	ThumbnailPath   string      `json:"thumbnailPath"`
}

func NewImage(name string, fullImage image.Image) *Image {
	return &Image{
		Name:       name,
		Full:       fullImage,
		Resolution: fullImage.Bounds().Max,
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
		Id:              img.Id,
		Name:            img.Name,
		FullBase64:      fullBase64Encoding,
		FullPath:        img.FullPath,
		Resolution:      img.Resolution,
		ThumbnailBase64: thumbBase64Encoding,
		ThumbnailPath:   img.ThumbnailPath,
	}
}

type ImageService interface {
	CreateThumbnail(ctx context.Context, image *Image) error
	Persist(ctx context.Context, image *Image) error
	SaveMetadata(ctx context.Context, image *Image) error
	GetAllMetadata(ctx context.Context) (<-chan *Image, <-chan error, error)
	GetMetadata(ctx context.Context, imageIds []int) ([]*Image, error)
	LoadThumbnail(ctx context.Context, img *Image) error
	LoadFull(ctx context.Context, img *Image) error
}

func NewImageService(db *sql.DB) *imageService {

	return &imageService{db}
}

type imageService struct {
	db *sql.DB
}

func (i *imageService) CreateThumbnail(ctx context.Context, image *Image) error {

	image.Thumbnail = resize.Resize(ThumbnailWidth, ThumbnailHeight, image.Full, resize.Lanczos3)
	return nil
}

func (i *imageService) Persist(ctx context.Context, image *Image) error {

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

func (i *imageService) SaveMetadata(ctx context.Context, image *Image) (err error) {

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
	err = tx.QueryRowContext(ctx, "INSERT INTO image (name, fullpath, thumbnailpath, resolution_x, resolution_y) VALUES( $1, $2, $3, $4, $5 ) RETURNING id", image.Name, image.FullPath, image.ThumbnailPath, image.Resolution.X, image.Resolution.Y).Scan(&imageId)
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

func (i *imageService) GetAllMetadata(ctx context.Context) (<-chan *Image, <-chan error, error) {

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
	if len(imageIds) == 0 {
		close(images)
		close(errors)
		return images, errors, nil
	}
	wg := sync.WaitGroup{}

	idGroups := divide(imageIds, 100)
	wg.Add(len(idGroups))
	for _, ids := range idGroups {
		go func(ids []int) {
			imgs, err := i.GetMetadata(ctx, ids)
			if err != nil {
				errors <- err
				log.Errorf("Error processing images with ids: %v", ids)
			} else {
				for _, img := range imgs {
					images <- img
				}
				log.Debugf("Processed successfully image with id: %v", ids)
			}
			wg.Done()
		}(ids)
	}

	go func() {
		wg.Wait()
		close(images)
		close(errors)
		log.Printf("Done processing all %d images", len(imageIds))
	}()

	return images, errors, nil
}

func divide(ids []int, groupSize int) [][]int {
	if len(ids) <= groupSize {
		return [][]int{ids}
	}
	numGroups := int(math.Ceil(float64(len(ids)) / float64(groupSize)))
	var retval [][]int
	for index := 0; index < numGroups; index++ {
		start := index * groupSize
		var end int
		if index == numGroups-1 {
			end = len(ids)
		} else {
			end = (index + 1) * groupSize
		}
		retval = append(retval, ids[start:end])
	}
	return retval
}

func (i *imageService) GetMetadata(ctx context.Context, imageIds []int) ([]*Image, error) {

	var str string
	for i, imgId := range imageIds {
		str += fmt.Sprintf("%d", imgId)
		if i != len(imageIds)-1 {
			str += ","
		}
	}
	rows, err := i.db.QueryContext(ctx, fmt.Sprintf("SELECT id, name, fullpath, thumbnailpath, resolution_x, resolution_y FROM image WHERE id IN (%s)", str))
	if err != nil {
		return nil, err
	}
	var imgs []*Image
	counter := 0
	for rows.Next() {
		counter++
		var img Image
		err := rows.Scan(&img.Id, &img.Name, &img.FullPath, &img.ThumbnailPath, &img.Resolution.X, &img.Resolution.Y)
		if err != nil {
			return nil, err
		}
		imgs = append(imgs, &img)
	}
	rows.Close()
	if counter != len(imageIds) {
		return nil, fmt.Errorf("some images weren't found")
	}

	return imgs, nil
}

func (i *imageService) LoadThumbnail(ctx context.Context, img *Image) error {

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

func (i *imageService) LoadFull(ctx context.Context, img *Image) error {

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
