package hough

import (
	"image"
	"testing"

	"go.viam.com/test"
)

func TestHough1(t *testing.T) {
	img, err := openImage("data/a1.jpg")
	test.That(t, err, test.ShouldBeNil)

	c := &HoughConfig{
		Crop:     &image.Rectangle{Min: image.Pt(115, 0), Max: image.Pt(600, 440)},
		SkipBlur: true,
	}
	c.setDefaults()
	c.MinDist = float64(c.MinRadius)

	circles, err := vesselCircles(img, c, false, "a1-output.jpg")
	test.That(t, err, test.ShouldBeNil)
	test.That(t, len(circles), test.ShouldEqual, 3)
}
