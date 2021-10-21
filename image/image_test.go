package image

import (
	"context"
	"fmt"
	"github.com/DATA-DOG/go-sqlmock"
	"image"
	"image/jpeg"
	"io/ioutil"
	"math"
	"os"
	"testing"
)

var TestImagePath = "/home/bp/go/src/github.com/ele7ija/go-pipelines/workers/test.jpg"

func TestImageService_CreateThumbnail(t *testing.T) {

	testImage := OpenTestImage(t)
	testImage.Thumbnail = nil

	t.Run("create and open", func(t *testing.T) {
		// Create a thumbnail, save it to file and check whether the resolution is as wanted

		imageService := NewImageService(nil)
		err := imageService.CreateThumbnail(context.Background(), testImage)
		if err != nil {
			t.Errorf("error creating thumbnail: %s", err)
		}

		// create a temp file that will be deleted
		file, err := ioutil.TempFile(os.TempDir(), "testimg*.jpg")
		if err != nil {
			t.Errorf("error creating temp file: %s", err)
		}
		defer os.Remove(file.Name())

		// save thumbnail to file
		if err := jpeg.Encode(file, testImage.Thumbnail, nil); err != nil {
			t.Errorf("error while saving thumbnail %s", err)
		}
		file.Close()
		file, _ = os.Open(file.Name())

		// read the resolution from file
		savedImage, format, _ := image.Decode(file)
		g := savedImage.Bounds()
		if uint(g.Dx()) != ThumbnailWidth || uint(g.Dy()) != ThumbnailHeight || format != "jpeg" {
			t.Errorf("image incorrect")
		}
	})
}

func TestImageService_Persist(t *testing.T) {

	testImage := OpenTestImage(t)

	t.Run("persist images", func(t *testing.T) {
		// Persist images to FS, check whether the files exist and can be deleted

		imageService := NewImageService(nil)
		err := imageService.Persist(context.Background(), testImage)
		if err != nil {
			t.Errorf("error while persisting images: %s", err)
		}

		if testImage.FullPath == "" || testImage.ThumbnailPath == "" {
			t.Errorf("paths are empty")
		}
		if _, err := os.Stat(testImage.FullPath); os.IsNotExist(err) {
			t.Errorf("full image wasn't saved: %s", err)
		}
		if _, err := os.Stat(testImage.ThumbnailPath); os.IsNotExist(err) {
			t.Errorf("thumbnail image wasn't saved: %s", err)
		}

		if err := os.Remove(testImage.FullPath); err != nil {
			t.Errorf("couldn't remove full image: %s", err)
		}
		if err := os.Remove(testImage.ThumbnailPath); err != nil {
			t.Errorf("couldn't remove thumbnail: %s", err)
		}
	})

}

func TestImageService_SaveMetadata(t *testing.T) {

	testName := "testName"
	testFullPath := "testFullPath"
	testThumbnailPath := "testThumbnailPath"
	testImage := &Image{
		Name:          testName,
		FullPath:      testFullPath,
		ThumbnailPath: testThumbnailPath,
		Resolution:    image.Point{X: 0, Y: 0},
	}
	userId := int64(1)
	imageId := int64(10)
	testResolutionX, testResolutionY := 0, 0

	t.Run("success", func(t *testing.T) {

		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
		}
		defer db.Close()

		mock.ExpectBegin()
		mock.ExpectQuery("INSERT INTO image").WithArgs(testName, testFullPath, testThumbnailPath, testResolutionX, testResolutionY).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(imageId))
		mock.ExpectExec("INSERT INTO user_images").WithArgs(userId, imageId).WillReturnResult(sqlmock.NewResult(imageId, 1))
		mock.ExpectCommit()

		service := NewImageService(db)
		ctx := context.WithValue(context.Background(), "userId", userId)
		if err = service.SaveMetadata(ctx, testImage); err != nil {
			t.Errorf("failed while persisting: %s", err)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})

	t.Run("error while executing first query", func(t *testing.T) {

		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
		}
		defer db.Close()

		mock.ExpectBegin()
		mock.ExpectQuery("INSERT INTO image").WithArgs(testName, testFullPath, testThumbnailPath, testResolutionX, testResolutionY).WillReturnError(fmt.Errorf("some err"))
		mock.ExpectRollback()

		service := NewImageService(db)
		if err = service.SaveMetadata(context.Background(), testImage); err == nil {
			t.Errorf("persisting should've failed")
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})

	t.Run("error while executing second query", func(t *testing.T) {

		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
		}
		defer db.Close()

		mock.ExpectBegin()
		mock.ExpectQuery("INSERT INTO image").WithArgs(testName, testFullPath, testThumbnailPath, testResolutionX, testResolutionY).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(imageId))
		mock.ExpectExec("INSERT INTO user_images").WithArgs(userId, imageId).WillReturnError(fmt.Errorf("some error"))
		mock.ExpectRollback()

		service := NewImageService(db)
		ctx := context.WithValue(context.Background(), "userId", 1)
		if err = service.SaveMetadata(ctx, testImage); err == nil {
			t.Errorf("should've failed while doing second query")
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})

	t.Run("error while committing", func(t *testing.T) {

		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
		}
		defer db.Close()

		mock.ExpectBegin()
		mock.ExpectQuery("INSERT INTO image").WithArgs(testName, testFullPath, testThumbnailPath, testResolutionX, testResolutionY).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(imageId))
		mock.ExpectExec("INSERT INTO user_images").WithArgs(userId, imageId).WillReturnResult(sqlmock.NewResult(imageId, 1))
		mock.ExpectCommit().WillReturnError(fmt.Errorf("error while committing"))

		service := NewImageService(db)
		ctx := context.WithValue(context.Background(), "userId", 1)
		if err = service.SaveMetadata(ctx, testImage); err == nil {
			t.Errorf("should've failed while committing")
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})
}

func TestImageService_GetMetadata(t *testing.T) {
	testName := "testName"
	testFullPath := "testFullPath"
	testThumbnailPath := "testThumbnailPath"
	imageId := int64(1)
	testResolutionX, testResolutionY := 0, 0

	t.Run("success", func(t *testing.T) {

		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
		}
		defer db.Close()

		rows := sqlmock.NewRows([]string{"id", "name", "fullpath", "thumbnailpath", "resolution_x", "resolution_y"}).
			AddRow(imageId, testName, testFullPath, testThumbnailPath, testResolutionX, testResolutionY)
		mock.ExpectQuery("SELECT").WillReturnRows(rows)

		service := NewImageService(db)
		imgs, err := service.GetMetadata(context.Background(), []int{int(imageId)})
		if err != nil {
			t.Errorf("unexpected error: %s", err)
		}
		if len(imgs) != 1 {
			t.Errorf("Got more than 1 image metadata")
		}
		img := imgs[0]
		if img.Name != testName || img.FullPath != testFullPath || img.ThumbnailPath != testThumbnailPath {
			t.Errorf("image was not instantiated well")
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})

	t.Run("image not found by id", func(t *testing.T) {

		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
		}
		defer db.Close()

		//query := fmt.Sprintf("SELECT name, fullpath, thumbnailpath FROM image WHERE image_id = %d", imageId)
		rows := sqlmock.NewRows([]string{"id", "name", "fullpath", "thumbnailpath", "resolution_x", "resolution_y"})
		mock.ExpectQuery("SELECT").WillReturnRows(rows)

		service := NewImageService(db)
		_, err = service.GetMetadata(context.Background(), []int{int(imageId)})
		if err == nil {
			t.Errorf("expected error of no rows found")
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})

	t.Run("query error", func(t *testing.T) {

		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
		}
		defer db.Close()

		mock.ExpectQuery("SELECT").WillReturnError(fmt.Errorf("some error"))

		service := NewImageService(db)
		_, err = service.GetMetadata(context.Background(), []int{int(imageId)})
		if err == nil {
			t.Errorf("expected query error")
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})
}

func TestImageService_GetAllMetadata(t *testing.T) {

	testName := "testName"
	testFullPath := "testFullPath"
	testThumbnailPath := "testThumbnailPath"
	noImages := 5
	images := make([]*Image, 0)
	for i := 0; i < noImages; i++ {
		images = append(images, &Image{
			Name:          fmt.Sprintf("%s%d", testName, i),
			FullPath:      fmt.Sprintf("%s%d", testFullPath, i),
			ThumbnailPath: fmt.Sprintf("%s%d", testThumbnailPath, i),
		})
	}
	userId := 1
	testResolutionX, testResolutionY := 0, 0

	t.Run("success", func(t *testing.T) {

		ctx := context.WithValue(context.Background(), "userId", userId)
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
		}
		defer db.Close()

		imageIdRows := sqlmock.NewRows([]string{"image_id"})
		for i := 0; i < noImages; i++ {
			imageIdRows.AddRow(i)
		}
		mock.ExpectQuery("SELECT image_id").WillReturnRows(imageIdRows)

		imageRows := sqlmock.NewRows([]string{"id", "name", "fullpath", "thumbnailpath", "resolution_x", "resolution_y"})
		for i := 0; i < noImages; i++ {
			imageRows.AddRow(i, images[i].Name, images[i].FullPath, images[i].ThumbnailPath, testResolutionX, testResolutionY)
		}
		mock.ExpectQuery("SELECT id, name, fullpath, thumbnailpath, resolution_x, resolution_y").WillReturnRows(imageRows)

		service := NewImageService(db)
		images, errors, err := service.GetAllMetadata(ctx)
		if err != nil {
			t.Errorf("unexpected error: %s", err)
		}
		go func() {
			for err := range errors {
				t.Errorf("an error happened in the pipeline: %s", err)
			}
		}()
		counter := 0
		for range images {
			counter++
		}
		if counter != noImages || len(errors) != 0 {
			t.Errorf("incorrect number of images")
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})

	t.Run("get ids fails", func(t *testing.T) {

		ctx := context.WithValue(context.Background(), "userId", userId)
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
		}
		defer db.Close()

		mock.ExpectQuery("SELECT image_id").WillReturnError(fmt.Errorf("some error"))

		service := NewImageService(db)
		_, _, err = service.GetAllMetadata(ctx)
		if err == nil {
			t.Errorf("should have failed on get ids")
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})

	t.Run("N images don't exist", func(t *testing.T) {

		ctx := context.WithValue(context.Background(), "userId", userId)
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
		}
		defer db.Close()

		imageIdRows := sqlmock.NewRows([]string{"image_id"})
		for i := 0; i < noImages; i++ {
			imageIdRows.AddRow(i)
		}
		mock.ExpectQuery("SELECT image_id").WillReturnRows(imageIdRows)

		noFails := 3
		imageRows := sqlmock.NewRows([]string{"id", "name", "fullpath", "thumbnailpath", "resolution_x", "resolution_y"})
		for i := 0; i < noImages-noFails; i++ {
			imageRows.AddRow(i, images[i].Name, images[i].FullPath, images[i].ThumbnailPath, testResolutionX, testResolutionY)
		}
		mock.ExpectQuery("SELECT id, name, fullpath, thumbnailpath, resolution_x, resolution_y").WillReturnRows(imageRows)

		service := NewImageService(db)
		images, errors, err := service.GetAllMetadata(ctx)
		if err != nil {
			t.Errorf("unexpected error: %s", err)
		}

		counterImg := 0
		for range images {
			counterImg++
		}
		counterErr := 0
		for range errors {
			counterErr++
		}
		if counterErr != 1 && counterImg != 0 {
			t.Errorf("incorrect number of images and errors")
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})
}

func TestDivide(t *testing.T) {

	t.Run("less than size", func(t *testing.T) {

		numberOfIds := 10
		groupSize := 11
		ids := make([]int, numberOfIds)
		divisions := divide(ids, groupSize)
		if len(divisions) != 1 {
			t.Errorf("err")
		}
		if len(divisions[0]) != numberOfIds {
			t.Errorf("err")
		}
	})

	t.Run("equal to size", func(t *testing.T) {

		numberOfIds := 10
		groupSize := 11
		ids := make([]int, numberOfIds)
		divisions := divide(ids, groupSize)
		if len(divisions) != 1 {
			t.Errorf("err")
		}
		if len(divisions[0]) != numberOfIds {
			t.Errorf("err")
		}
	})

	t.Run("more than size last equal to group size", func(t *testing.T) {

		numberOfIds := 15
		groupSize := 5
		ids := make([]int, numberOfIds)
		divisions := divide(ids, groupSize)
		for i, division := range divisions {
			if i != len(divisions)-1 && len(division) != groupSize {
				t.Errorf("err")
			} else if i == len(divisions)-1 {
				if int(math.Mod(float64(numberOfIds), float64(groupSize))) == 0 {
					if len(division) != groupSize {
						t.Errorf("err")
					}
				} else if len(division) != int(math.Mod(float64(numberOfIds), float64(groupSize))) {
					t.Errorf("err")
				}
			}
		}
	})

	t.Run("more than size last not equal to group size", func(t *testing.T) {

		numberOfIds := 16
		groupSize := 5
		ids := make([]int, numberOfIds)
		divisions := divide(ids, groupSize)
		for i, division := range divisions {
			if i != len(divisions)-1 && len(division) != groupSize {
				t.Errorf("err")
			} else if i == len(divisions)-1 {
				fmt.Println(int(math.Mod(float64(numberOfIds), float64(groupSize))))
				if int(math.Mod(float64(numberOfIds), float64(groupSize))) == 0 {
					if len(division) != groupSize {
						t.Errorf("err")
					}
				} else if len(division) != int(math.Mod(float64(numberOfIds), float64(groupSize))) {
					t.Errorf("err")
				}
			}
		}
	})
}

func OpenTestImage(t *testing.T) *Image {

	existingImageFile, err := os.Open(TestImagePath)
	if err != nil {
		t.Errorf("test image doesn't exist %s", err)
		return nil
	}
	if existingImageFile == nil {
		t.Errorf("image null")
		return nil
	}
	defer existingImageFile.Close()

	imageData, _, err := image.Decode(existingImageFile)
	if err != nil {
		t.Errorf("image failed decoding: %s", err)
		return nil
	}

	return &Image{
		Full:      imageData,
		Thumbnail: imageData,
	}
}
