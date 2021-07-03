package workers

import (
	"image"
	"image/jpeg"
	"io/ioutil"
	"os"
	"testing"
)

func TestImageService_CreateThumbnail(t *testing.T) {

	testImage := openTestImage(t)
	testImage.Thumbnail = nil

	t.Run("create and open", func(t *testing.T) {

		imageService := NewImageService()
		err := imageService.CreateThumbnail(testImage)
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
		if uint(g.Dx()) != ThumbnailWidth || uint(g.Dy()) != ThumbnailHeigth || format != "jpeg" {
			t.Errorf("image incorrect")
		}
	})
}

func TestImageService_Persist(t *testing.T) {

	testImage := openTestImage(t)

	t.Run("persist images", func(t *testing.T) {

		imageService := NewImageService()
		err := imageService.Persist(testImage)
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
			t.Errorf("full image wasn't saved: %s", err)
		}

		if err := os.Remove(testImage.FullPath); err != nil {
			t.Errorf("couldn't remove full image: %s", err)
		}
		if err := os.Remove(testImage.ThumbnailPath); err != nil {
			t.Errorf("couldn't remove thumbnail: %s", err)
		}
	})

}

func openTestImage(t *testing.T) *Image {

	existingImageFile, err := os.Open("test.jpg")
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
