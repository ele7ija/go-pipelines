package workers

import (
	"fmt"
	"image"
	"image/jpeg"
	_ "image/jpeg"
	"io/ioutil"
	"os"
	"testing"
)

func TestImageService_CreateThumbnail(t *testing.T) {

	existingImageFile, err := os.Open("test.jpg")
	if err != nil {
		t.Errorf("test image doesn't exist %s", err)
	}
	if existingImageFile == nil {
		t.Errorf("image null")
	}
	defer existingImageFile.Close()
	imageData, _, err := image.Decode(existingImageFile)
	if err != nil {
		t.Errorf("image failed decoding")
	}
	testImage := NewImage(imageData)


	t.Run("create and open", func(t *testing.T) {

		imageService := NewImageService()
		testImage, err := imageService.CreateThumbnail(testImage)
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
		if g.Dx() != 200 || g.Dy() != 200 || format != "jpeg" {
			t.Errorf("image incorrect")
		}
		fmt.Println(file.Name())
	})
}

func TestImageService_Persist(t *testing.T) {

	t.Run("jpeg image", func(t *testing.T) {
		// Read image from file that already exists
		existingImageFile, err := os.Open("test.jpg")
		defer existingImageFile.Close()
		if err != nil {
			t.Errorf("image doesn't exist")
		}
		if existingImageFile == nil {
			t.Errorf("image null")
		}

		// Calling the generic image.Decode() will tell give us the data
		// and type of image it is as a string. We expect "png"
		_, imageType, err := image.Decode(existingImageFile)
		if err != nil {
			t.Errorf("image failed decoding")
		}


		fmt.Println(imageType)
	})

}
