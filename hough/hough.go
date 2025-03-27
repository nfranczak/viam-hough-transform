// Package hough implements an object tracker as a Viam vision service
package hough

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"strconv"

	"image"

	"github.com/pkg/errors"
	"go.viam.com/rdk/components/camera"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/resource"
	"go.viam.com/rdk/services/vision"
	vis "go.viam.com/rdk/vision"
	"go.viam.com/rdk/vision/classification"
	objdet "go.viam.com/rdk/vision/objectdetection"
	"go.viam.com/rdk/vision/viscapture"
)

const (
	ModelName = "hough-transform"
)

var (
	// Here is where we define your new model's colon-delimited-triplet (viam:vision:hough-transform)
	Model            = resource.NewModel("viam", "circle-detector", ModelName)
	errUnimplemented = errors.New("unimplemented")
)

func init() {
	resource.RegisterService(vision.API, Model, resource.Registration[vision.Service, *HoughConfig]{
		Constructor: newHoughTransformer,
	})
}

type myHoughTransformer struct {
	resource.Named
	resource.AlwaysRebuild

	logger logging.Logger
	cam    camera.Camera
	conf   *HoughConfig
}

func newHoughTransformer(ctx context.Context, deps resource.Dependencies, conf resource.Config, logger logging.Logger) (vision.Service, error) {

	newConf, err := resource.NativeConfig[*HoughConfig](conf)
	if err != nil {
		return nil, errors.Errorf("Could not assert proper config for %s", ModelName)
	}

	h := &myHoughTransformer{
		logger: logger,
		conf:   newConf,
	}

	h.cam, err = camera.FromDependencies(deps, newConf.CameraName)
	if err != nil {
		return nil, err
	}

	return h, nil
}

func (h *myHoughTransformer) DetectionsFromCamera(
	ctx context.Context,
	cameraName string,
	extra map[string]interface{},
) ([]objdet.Detection, error) {
	colorImg, err := h.getImage(ctx)
	if err != nil {
		return nil, err
	}

	detections, err := h.Detections(ctx, colorImg, map[string]interface{}{"addOffset": true})
	if err != nil {
		return nil, err
	}

	return detections, nil
}

func (h *myHoughTransformer) Detections(ctx context.Context, img image.Image, extra map[string]interface{}) ([]objdet.Detection, error) {
	addOffset, ok := extra["addOffset"].(bool)
	if !ok {
		return nil, errors.New("we do not know if we should add an offset to the detections, please specify")
	}

	circles, err := vesselCircles(img, h.conf, addOffset, false, "")
	if err != nil {
		return nil, err
	}

	return formatDetections(circles), nil
}

func (h *myHoughTransformer) ClassificationsFromCamera(
	ctx context.Context,
	cameraName string,
	n int,
	extra map[string]interface{},
) (classification.Classifications, error) {
	return nil, errUnimplemented
}

func (h *myHoughTransformer) Classifications(ctx context.Context, img image.Image,
	n int, extra map[string]interface{},
) (classification.Classifications, error) {
	return nil, errUnimplemented
}

func (h *myHoughTransformer) GetProperties(ctx context.Context, extra map[string]interface{}) (*vision.Properties, error) {
	return &vision.Properties{
		DetectionSupported:      true,
		ClassificationSupported: false,
		ObjectPCDsSupported:     false,
	}, nil
}
func (h *myHoughTransformer) GetObjectPointClouds(
	ctx context.Context,
	cameraName string,
	extra map[string]interface{},
) ([]*vis.Object, error) {
	return nil, errUnimplemented
}

func (h *myHoughTransformer) CaptureAllFromCamera(
	ctx context.Context,
	cameraName string,
	opt viscapture.CaptureOptions,
	extra map[string]interface{},
) (viscapture.VisCapture, error) {

	colorImg, err := h.getImage(ctx)
	if err != nil {
		return viscapture.VisCapture{}, err
	}

	output := fmt.Sprintf("output-%d.jpg", rand.Int()%1000)

	circles, err := vesselCircles(colorImg, h.conf, false, false, output)
	if err != nil {
		return viscapture.VisCapture{}, err
	}

	detections := formatDetections(circles)

	croppedColorImg, err := openImage(output)
	if err != nil {
		return viscapture.VisCapture{}, err
	}

	os.Remove(output)

	return viscapture.VisCapture{
		Image:      croppedColorImg,
		Detections: detections,
	}, nil
}

func (h *myHoughTransformer) Close(ctx context.Context) error {
	return nil
}

func (h *myHoughTransformer) DoCommand(ctx context.Context, cmd map[string]interface{}) (map[string]interface{}, error) {
	return nil, errors.New("called DoCommand but nothing was executed")
}

func (h *myHoughTransformer) getImage(ctx context.Context) (image.Image, error) {
	images, _, err := h.cam.Images(ctx)
	if err != nil {
		return nil, err
	}

	var colorImg image.Image
	for _, img := range images {
		if img.SourceName == "color" {
			colorImg = img.Image
		}
	}
	return colorImg, nil
}

func formatDetections(circles []Circle) []objdet.Detection {
	var detections []objdet.Detection
	for i, c := range circles {
		minX := c.center.X - (c.radius)
		maxX := c.center.X + (c.radius)
		minY := c.center.Y - (c.radius)
		maxY := c.center.Y + (c.radius)
		rect := image.Rectangle{
			Min: image.Point{X: minX, Y: minY},
			Max: image.Point{X: maxX, Y: maxY},
		}
		name := "circle-" + strconv.Itoa(i)
		detections = append(detections, objdet.NewDetection(rect, 1, name))
	}
	return detections
}

func openImage(fn string) (image.Image, error) {
	file, err := os.Open("output.jpg")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	return img, err
}
