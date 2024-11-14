// Package hough implements an object tracker as a Viam vision service
package hough

import (
	"context"
	"fmt"
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
	ModelName        = "hough-transform"
	defaultDP        = 1
	defaultMinDist   = 8
	defaultParam1    = 60
	defaultParam2    = 25
	defaultMinRadius = 35
	defaultMaxRadius = 50
)

var (
	// Here is where we define your new model's colon-delimited-triplet (viam:vision:hough-transform)
	Model            = resource.NewModel("viam", "vision", ModelName)
	errUnimplemented = errors.New("unimplemented")
)

func init() {
	resource.RegisterService(vision.API, Model, resource.Registration[vision.Service, *Config]{
		Constructor: newHoughTransformer,
	})
}

type myHoughTransformer struct {
	resource.Named
	logger    logging.Logger
	cam       camera.Camera
	dp        float64
	minDist   float64
	param1    float64
	param2    float64
	minRadius int
	maxRadius int
}

func newHoughTransformer(ctx context.Context, deps resource.Dependencies, conf resource.Config, logger logging.Logger) (vision.Service, error) {
	h := &myHoughTransformer{
		logger: logger,
	}
	if err := h.Reconfigure(ctx, deps, conf); err != nil {
		return nil, err
	}
	return h, nil
}

// Config contains names for necessary resources (camera and vision service)
type Config struct {
	CameraName string  `json:"camera_name"`
	Dp         float64 `json:"dp,omitempty"`
	MinDist    float64 `json:"min_dist,omitempty"`
	Param1     float64 `json:"param1,omitempty"`
	Param2     float64 `json:"param2,omitempty"`
	MinRadius  int     `json:"min_radius,omitempty"`
	MaxRadius  int     `json:"max_radius,omitempty"`
}

// Validate validates the config and returns implicit dependencies,
func (cfg *Config) Validate(path string) ([]string, error) {
	if cfg.CameraName == "" {
		return nil, fmt.Errorf(`expected "camera_name" attribute for object tracker %q`, path)
	}

	return []string{cfg.CameraName}, nil
}

// Reconfigure reconfigures with new settings.
func (h *myHoughTransformer) Reconfigure(ctx context.Context, deps resource.Dependencies, conf resource.Config) error {
	houghConfig, err := resource.NativeConfig[*Config](conf)
	if err != nil {
		return errors.Errorf("Could not assert proper config for %s", ModelName)
	}

	h.dp = houghConfig.Dp
	h.minDist = houghConfig.MinDist
	h.param1 = houghConfig.Param1
	h.param2 = houghConfig.Param2
	h.minRadius = houghConfig.MinRadius
	h.maxRadius = houghConfig.MaxRadius

	cam, err := camera.FromDependencies(deps, houghConfig.CameraName)
	if err != nil {
		return err
	}
	h.cam = cam

	return nil
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

	circles, err := vesselCircles(img, addOffset)
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

	detections, err := h.Detections(ctx, colorImg, map[string]interface{}{"addOffset": false})
	if err != nil {
		return viscapture.VisCapture{}, err
	}

	croppedColorImg, err := openImage()
	if err != nil {
		return viscapture.VisCapture{}, err
	}

	return viscapture.VisCapture{
		Image:      croppedColorImg,
		Detections: detections,
	}, nil
}

func (h *myHoughTransformer) Close(ctx context.Context) error {
	return nil
}

func (h *myHoughTransformer) DoCommand(ctx context.Context, cmd map[string]interface{}) (map[string]interface{}, error) {
	if _, ok := cmd["determineAmountOfCups"]; ok {
		// code here
	}
	if numOfCupsToDetect, ok := cmd["getTheDetections"].(int); ok {
		_ = numOfCupsToDetect
		// code here
	}
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

func openImage() (image.Image, error) {
	// Open the JPEG file
	file, err := os.Open("output.jpg")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Decode the image
	img, _, err := image.Decode(file)
	if err != nil {
		return nil, err
	}

	return img, nil
}
