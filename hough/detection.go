package hough

import (
	"errors"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"os"
	"sort"

	"github.com/golang/geo/r3"
	"go.viam.com/rdk/rimage/transform"
	"gocv.io/x/gocv"
)

// for normalizing
const minDepth uint32 = 300 //mm
const maxDepth uint32 = 675 //mm

// for cropping (original image is size 640x480)
var crop = image.Rectangle{Min: image.Pt(115, 0), Max: image.Pt(600, 440)}

// circles with radii smaller than this will be ignored
const circleRThreshold = 18

type Circle struct {
	center image.Point
	radius int
}

func vesselCircles(img image.Image, addOffset bool) ([]Circle, error) {
	croppedImg := cropImage(img)
	_ = croppedImg

	// Save the image as a .jpg file
	outFile, err := os.Create("input.jpg")
	if err != nil {
		return nil, err
	}
	defer outFile.Close()

	if err = jpeg.Encode(outFile, croppedImg, nil); err != nil {
		panic(err)
	}

	// Reads the image "input.jpg" in color mode using gocv.IMRead.
	// The IMReadColor flag loads the image image in BGR format.
	// If the image fails to load , it panics with an error message.
	// The defer mat.CLose() ensures that the mat is released from memory when the function exists
	mat := gocv.IMRead("input.jpg", gocv.IMReadColor)
	if mat.Empty() {
		return nil, errors.New("cannot read image")
	}
	defer mat.Close()

	// Convert to grayscale
	// This converts the original color image (mat) to grayscale and stores it in gray.
	gray := gocv.NewMat()
	defer gray.Close()
	gocv.CvtColor(mat, &gray, gocv.ColorBGRToGray)

	// Blur to reduce noise
	gocv.MedianBlur(gray, &gray, 15)

	// Save the blurred picture for debugging purposes
	if ok := gocv.IMWrite("blurred.jpg", gray); !ok {
		return nil, errors.New("failed to save the output image")
	}

	// Detect circles using HoughCircles
	circles := gocv.NewMat()
	defer circles.Close()

	// READ MORE ABOUT THIS HERE:
	// https://docs.opencv.org/4.x/dd/d1a/group__imgproc__feature.html#ga47849c3be0d0406ad3ca45db65a25d2d
	gocv.HoughCirclesWithParams(
		gray,                   // src
		&circles,               // circles
		gocv.HoughGradient,     // method - only HoughGradient is supported
		1,                      // dp: inverse ratio of the accumulator resolution to the image resolution
		float64(gray.Rows()/8), // minDist: minimum distance between the centers of detected circles (Question: how is distance calculated here?)
		60,                     // param1: the higher threshold for the canny edge detector
		25,                     // param2: the accumulator threshold for circle detection
		35,                     // minRadius of bounding circle
		50,                     // maxRadius of bouding circle
	)

	// Draw the circles on the original image
	goodCircles := make([]Circle, 0)
	for i := 0; i < circles.Cols(); i++ {
		circle := circles.GetVecfAt(0, i)
		center := image.Pt(int(circle[0]), int(circle[1]))
		radius := int(circle[2])
		if radius < circleRThreshold {
			continue
		}
		gocv.Circle(&mat, center, radius, color.RGBA{255, 0, 0, 0}, 2)
		if addOffset {
			// need to add the offset back so circle is returned with respect to original image
			goodCircles = append(goodCircles, Circle{center.Add(crop.Min), radius})
		} else {
			goodCircles = append(goodCircles, Circle{center, radius})
		}
	}

	// Save the output image with circles
	output := "output.jpg"
	if ok := gocv.IMWrite(output, mat); !ok {
		return nil, errors.New("failed to save the output image")
	}

	// order the circles by radius
	sort.Slice(goodCircles, func(i, j int) bool {
		return goodCircles[i].radius > goodCircles[j].radius
	})
	return goodCircles, nil
}

// normalizeDepth converts a depth image to a high-contrast grayscale image,
// emphasizing objects like cups and bottles for Hough transform.
func normalizeDepth(img image.Image, min, max uint32) *image.Gray {
	bounds := img.Bounds()
	grayImg := image.NewGray(bounds)

	// Normalize depth window we are interested in to 0-255
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			depth, _, _, _ := img.At(x, y).RGBA()
			normalized := uint8((depth - min) * 255 / (max - min))
			grayImg.SetGray(x, y, color.Gray{Y: normalized})
		}
	}
	return grayImg
}

func circleToPt(intrinsics transform.PinholeCameraIntrinsics, circle Circle, z, xAdjustment, yAdjustment float64) r3.Vector {
	xmm := (float64(circle.center.X) - intrinsics.Ppx) * (z / intrinsics.Fx)
	ymm := (float64(circle.center.Y) - intrinsics.Ppy) * (z / intrinsics.Fy)
	xmm = xmm + xAdjustment
	ymm = ymm + yAdjustment
	return r3.Vector{X: xmm, Y: ymm, Z: z}
}

func cropImage(src image.Image) image.Image {
	// Create a new RGBA image with the size of the crop rectangle
	croppedImg := image.NewRGBA(image.Rect(0, 0, crop.Dx(), crop.Dy()))

	// Adjust the draw point to correctly position the cropped area
	draw.Draw(croppedImg, croppedImg.Bounds(), src, crop.Min, draw.Src)
	return croppedImg
}
