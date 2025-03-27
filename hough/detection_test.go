package hough

import (
	"image"
	"os"
	"testing"

	"go.viam.com/test"
)

func read(fn string) (image.Image, error) {
	file, err := os.Open(fn)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	return img, err
}

func TestHough1(t *testing.T) {
	img, err := read("data/a1.jpg")
	test.That(t, err, test.ShouldBeNil)

	c := &HoughConfig{
		Crop:     &image.Rectangle{Min: image.Pt(115, 0), Max: image.Pt(600, 440)},
		SkipBlur: true,
	}
	c.setDefaults()
	c.MinDist = float64(c.MinRadius)

	circles, err := vesselCircles(img, c, false, false, true)
	test.That(t, err, test.ShouldBeNil)
	test.That(t, len(circles), test.ShouldEqual, 3)
}
