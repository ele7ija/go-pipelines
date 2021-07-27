package workers

import (
	"context"
	"fmt"
	"github.com/DATA-DOG/go-sqlmock"
	"image"
	"image/jpeg"
	"io/ioutil"
	"os"
	"testing"
)

var TestImagePath = "test.jpg"

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
	}
	userId := int64(1)
	imageId := int64(1)

	t.Run("success", func(t *testing.T) {

		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
		}
		defer db.Close()

		mock.ExpectBegin()
		mock.ExpectExec("INSERT INTO image").WithArgs(testName, testFullPath, testThumbnailPath).WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectExec("INSERT INTO user_images").WithArgs(userId, imageId).WillReturnResult(sqlmock.NewResult(1, 1))
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
		mock.ExpectExec("INSERT INTO image").WithArgs(testName, testFullPath, testThumbnailPath).WillReturnError(fmt.Errorf("some error"))
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
		mock.ExpectExec("INSERT INTO image").WithArgs(testName, testFullPath, testThumbnailPath).WillReturnResult(sqlmock.NewResult(imageId, 1))
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
		mock.ExpectExec("INSERT INTO image").WithArgs(testName, testFullPath, testThumbnailPath).WillReturnResult(sqlmock.NewResult(imageId, 1))
		mock.ExpectExec("INSERT INTO user_images").WithArgs(userId, imageId).WillReturnResult(sqlmock.NewResult(1, 1))
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

	t.Run("success", func(t *testing.T) {

		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
		}
		defer db.Close()

		rows := sqlmock.NewRows([]string{"name", "fullpath", "thumbnailpath"}).
			AddRow(testName, testFullPath, testThumbnailPath)
		mock.ExpectQuery("SELECT").WillReturnRows(rows)

		service := NewImageService(db)
		img, err := service.GetMetadata(context.Background(), int(imageId))
		if err != nil {
			t.Errorf("unexpected error: %s", err)
		}

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
		rows := sqlmock.NewRows([]string{"name", "fullpath", "thumbnailpath"})
		mock.ExpectQuery("SELECT").WillReturnRows(rows)

		service := NewImageService(db)
		_, err = service.GetMetadata(context.Background(), int(imageId))
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
		_, err = service.GetMetadata(context.Background(), int(imageId))
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
			fmt.Sprintf("%s%d", testName, i),
			nil,
			fmt.Sprintf("%s%d", testFullPath, i),
			nil,
			fmt.Sprintf("%s%d", testThumbnailPath, i),
		})
	}

	t.Run("success", func(t *testing.T) {

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

		for i := 0; i < noImages; i++ {
			imageRows := sqlmock.NewRows([]string{"name", "fullpath", "thumbnailpath"})
			imageRows.AddRow(images[i].Name, images[i].FullPath, images[i].ThumbnailPath)
			mock.ExpectQuery("SELECT name, fullpath, thumbnailpath").WillReturnRows(imageRows)
		}

		service := NewImageService(db)
		images, errors, err := service.GetAllMetadata(context.Background())
		if err != nil {
			t.Errorf("unexpected error: %s", err)
		}

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

		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
		}
		defer db.Close()

		mock.ExpectQuery("SELECT image_id").WillReturnError(fmt.Errorf("some error"))

		service := NewImageService(db)
		_, _, err = service.GetAllMetadata(context.Background())
		if err == nil {
			t.Errorf("should have failed on get ids")
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})

	t.Run("get N images fails", func(t *testing.T) {

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
		for i := 0; i < noImages; i++ {
			if i < noFails {
				mock.ExpectQuery("SELECT name, fullpath, thumbnailpath").WillReturnError(fmt.Errorf("some error"))
			} else {
				imageRows := sqlmock.NewRows([]string{"name", "fullpath", "thumbnailpath"})
				imageRows.AddRow(images[i].Name, images[i].FullPath, images[i].ThumbnailPath)
				mock.ExpectQuery("SELECT name, fullpath, thumbnailpath").WillReturnRows(imageRows)
			}
		}

		service := NewImageService(db)
		images, errors, err := service.GetAllMetadata(context.Background())
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
		if counterImg != noImages - noFails || counterErr != noFails {
			t.Errorf("incorrect number of images and errors")
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
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
		Full:          imageData,
		Thumbnail:     imageData,
	}
}
