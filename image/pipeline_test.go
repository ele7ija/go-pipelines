package image

import (
	"bytes"
	"context"
	"fmt"
	"github.com/DATA-DOG/go-sqlmock"
	pipeline2 "github.com/ele7ija/pipeline"
	"image"
	"image/color"
	"image/draw"
	"io"
	"math"
	"mime/multipart"
	"net/http"
	"os"
	"testing"
)

func TestGetImagePipeline(t *testing.T) {

	testName := "testName"
	testFullPath := TestImagePath
	testThumbnailPath := TestImagePath
	imageId := int64(1)
	testResolutionX, testResolutionY := 0, 0

	t.Run("default", func(t *testing.T) {

		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
		}
		defer db.Close()

		rows := sqlmock.NewRows([]string{"id", "name", "fullpath", "thumbnailpath", "resolution_x", "resolution_y"}).
			AddRow(imageId, testName, testFullPath, testThumbnailPath, testResolutionX, testResolutionY)
		mock.ExpectQuery("SELECT").WillReturnRows(rows)

		service := NewImageService(db)
		pipeline := MakeGetImagePipeline(service)

		inputChan := make(chan pipeline2.Item, 1)
		errors := make(chan error, 1)
		inputChan <- int(imageId)
		close(inputChan)
		items := pipeline.Filter(context.Background(), inputChan, errors)
		go func() {
			for err := range errors {
				t.Errorf("Error: %v", err)
			}
		}()
		for item := range items {
			img, ok := item.(*ImageBase64)
			if !ok {
				t.Errorf("item at the end of the pipeline is not an image")
			}
			if img.Name != testName {
				t.Errorf("name not equal")
			}
			if img.FullBase64 != NewImageBase64(OpenTestImage(t)).FullBase64 {
				t.Errorf("image not loaded well")
			}
		}
		close(errors)
	})
}

func TestGetAllImagesPipeline(t *testing.T) {
	testFullPath := TestImagePath
	testThumbnailPath := TestImagePath

	t.Run("default", func(t *testing.T) {

		service := NewImageService(nil)
		pipeline := MakeGetAllImagesPipeline(service)

		noItems := 5
		inputChan := make(chan pipeline2.Item, noItems)
		errors := make(chan error, noItems)
		for i := 0; i < noItems; i++ {
			img := Image{
				Name:          fmt.Sprintf("%d", i),
				FullPath:      testFullPath,
				ThumbnailPath: testThumbnailPath,
			}
			inputChan <- &img
		}
		close(inputChan)

		items := pipeline.Filter(context.Background(), inputChan, errors)
		go func() {
			for err := range errors {
				t.Errorf("Error: %v", err)
			}
		}()
		for item := range items {
			img, ok := item.(*ImageBase64)
			if !ok {
				t.Errorf("item at the end of the pipeline is not an image")
			}
			if img.ThumbnailBase64 != NewImageBase64(OpenTestImage(t)).ThumbnailBase64 {
				t.Errorf("image not loaded well")
			}
		}
	})
}

func TestMakeCreateImagesPipeline(t *testing.T) {

	userId := int64(1)

	t.Run("default", func(t *testing.T) {

		ctx := context.WithValue(context.Background(), "userId", userId)
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
		}
		defer db.Close()

		service := NewImageService(db)
		pipeline := MakeCreateImagesPipelineBoundedFilters(service)

		noItems := 2
		items := make(chan pipeline2.Item, noItems)
		errors := make(chan error, noItems)
		//Prepare
		fhs := prepareFileHeaders(noItems, t)
		for i := 0; i < noItems; i++ {
			mock.ExpectBegin()
			mock.ExpectQuery("INSERT INTO image").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(i))
			mock.ExpectExec("INSERT INTO user_images").WithArgs(userId, i).WillReturnResult(sqlmock.NewResult(int64(i), 1))
			mock.ExpectCommit()
		}
		for _, fh := range fhs {
			items <- fh
		}
		go func() {
			for err := range errors {
				t.Errorf("%s", err)
			}
		}()
		close(items)
		filteredItems := pipeline.Filter(ctx, items, errors)
		for item := range filteredItems {
			img, ok := item.(*ImageBase64)
			if !ok {
				t.Errorf("not an image")
			}
			if img.FullBase64 != NewImageBase64(OpenTestImage(t)).FullBase64 {
				t.Errorf("not good image")
			}
			if img.Name != "test.jpg" {
				t.Errorf("not good name")
			}
		}
		close(errors)
	})
}

func prepareFileHeaders(items int, t *testing.T) []*multipart.FileHeader {
	var b bytes.Buffer
	w := multipart.NewWriter(&b)

	for i := 0; i < items; i++ {
		r, err := os.Open(TestImagePath)
		fw, err := w.CreateFormFile("images", TestImagePath)
		_, err = io.Copy(fw, r)
		if err != nil {
			t.Errorf("err %s", err)
		}
	}

	w.Close()

	// Now that you have a form, you can submit it to your handler.
	req, err := http.NewRequest("POST", "", &b)
	if err != nil {
		t.Errorf("err")
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	if err := req.ParseMultipartForm(2 << 30); err != nil {
		t.Errorf("err")
	}
	return req.MultipartForm.File["images"]
}

func ImgCompare(img1, img2 image.Image) (int64, image.Image, error) {
	bounds1 := img1.Bounds()
	bounds2 := img2.Bounds()
	if bounds1 != bounds2 {
		return math.MaxInt64, nil, fmt.Errorf("image bounds not equal: %+v, %+v", img1.Bounds(), img2.Bounds())
	}

	accumError := int64(0)
	resultImg := image.NewRGBA(image.Rect(
		bounds1.Min.X,
		bounds1.Min.Y,
		bounds1.Max.X,
		bounds1.Max.Y,
	))
	draw.Draw(resultImg, resultImg.Bounds(), img1, image.Point{0, 0}, draw.Src)

	for x := bounds1.Min.X; x < bounds1.Max.X; x++ {
		for y := bounds1.Min.Y; y < bounds1.Max.Y; y++ {
			r1, g1, b1, a1 := img1.At(x, y).RGBA()
			r2, g2, b2, a2 := img2.At(x, y).RGBA()

			diff := int64(sqDiffUInt32(r1, r2))
			diff += int64(sqDiffUInt32(g1, g2))
			diff += int64(sqDiffUInt32(b1, b2))
			diff += int64(sqDiffUInt32(a1, a2))

			if diff > 0 {
				accumError += diff
				resultImg.Set(
					bounds1.Min.X+x,
					bounds1.Min.Y+y,
					color.RGBA{R: 255, A: 255})
			}
		}
	}

	return int64(math.Sqrt(float64(accumError))), resultImg, nil
}

func sqDiffUInt32(x, y uint32) uint64 {
	d := uint64(x) - uint64(y)
	return d * d
}

func ImagesEqual(img1, img2 image.Image) (bool, error) {

	diff, _, err := ImgCompare(img1, img2)
	if err != nil {
		return false, err
	}
	if diff == 0 {
		return true, nil
	}
	return false, nil
}
