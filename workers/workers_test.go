package workers

import (
	"fmt"
	"image"
	_ "image/jpeg"
	"os"
	"testing"
)

func TestNewImage(t *testing.T) {

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
