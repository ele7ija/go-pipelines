package workers

import (
	"context"
	"fmt"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/ele7ija/go-pipelines/internal"
	"image"
	"image/color"
	"image/draw"
	"math"
	"sync"
	"testing"
)

func TestGetImagePipeline(t *testing.T) {

	testName := "testName"
	testFullPath := TestImagePath
	testThumbnailPath := TestImagePath
	imageId := int64(1)

	t.Run("default", func(t *testing.T) {

		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
		}
		defer db.Close()

		rows := sqlmock.NewRows([]string{"name", "fullpath", "thumbnailpath"}).
			AddRow(testName, testFullPath, testThumbnailPath)
		mock.ExpectQuery("SELECT").WillReturnRows(rows)

		service := NewImageService(db)
		pipeline := GetImagePipeline(service)

		inputChan := make(chan internal.Item, 1)
		inputChan <- internal.NewGenericItem(int(imageId))
		close(inputChan)
		items, errors := pipeline.Filter(context.Background(), inputChan)
		wg := sync.WaitGroup{}
		wg.Add(1)
		go func() {
			for err := range errors {
				t.Errorf("Error: %v", err)
			}
			wg.Done()
		}()
		for item := range items {
			img, ok := item.GetPart(0).(*Image)
			if !ok {
				t.Errorf("item at the end of the pipeline is not an image")
			}
			if img.Name != testName {
				t.Errorf("name not equal")
			}
			if val, err := ImagesEqual(img.Full, OpenTestImage(t).Full); err != nil || !val {
				t.Errorf("image not loaded well")
			}
		}
		wg.Wait()
	})
}

func TestGetAllImagesPipeline(t *testing.T) {
	testFullPath := TestImagePath
	testThumbnailPath := TestImagePath

	t.Run("default", func(t *testing.T) {

		service := NewImageService(nil)
		pipeline := GetAllImagesPipeline(service)

		noItems := 5
		inputChan := make(chan internal.Item, noItems)
		for i := 0; i < noItems; i++ {
			img := Image{
				Name:          fmt.Sprintf("%d", i),
				FullPath:      testFullPath,
				ThumbnailPath: testThumbnailPath,
			}
			inputChan <- internal.NewGenericItem(&img)
		}
		close(inputChan)

		items, errors := pipeline.Filter(context.Background(), inputChan)
		wg := sync.WaitGroup{}
		wg.Add(1)
		go func() {
			for err := range errors {
				t.Errorf("Error: %v", err)
			}
			wg.Done()
		}()
		for item := range items {
			img, ok := item.GetPart(0).(*Image)
			if !ok {
				t.Errorf("item at the end of the pipeline is not an image")
			}
			if val, err := ImagesEqual(img.Full, OpenTestImage(t).Full); err != nil || !val {
				t.Errorf("image not loaded well")
			}
			if val, err := ImagesEqual(img.Thumbnail, OpenTestImage(t).Thumbnail); err != nil || !val {
				t.Errorf("image not loaded well")
			}
		}
		wg.Wait()
	})
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
