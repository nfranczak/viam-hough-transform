// Package hough implements an object tracker as a Viam vision service
package hough

import (
	"context"
	"fmt"
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

// ModelName is the name of the model
const (
	ModelName = "hough-transform"
)

var (
	// Here is where we define your new model's colon-delimited-triplet (viam:vision:object-tracker)
	Model                  = resource.NewModel("viam", "vision", ModelName)
	errUnimplemented       = errors.New("unimplemented")
	DefaultMinConfidence   = 0.2
	DefaultMaxFrequency    = 10.0
	DefaultTriggerCoolDown = 5.0
	DefaultBufferSize      = 30
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
	CameraName string  `json:"camera_name,omitempty"`
	Dp         float64 `json:"dp,omitempty"`
	MinDist    float64 `json:"min_dist,omitempty"`
	Param1     float64 `json:"param1,omitempty"`
	Param2     float64 `json:"param2,omitempty"`
	MinRadius  int     `json:"min_radius,omitempty"`
	MaxRadius  int     `json:"max_radius,omitempty"`
}

// Validate validates the config and returns implicit dependencies,
// this Validate checks if the camera and detector(vision svc) exist for the module's vision model.
func (cfg *Config) Validate(path string) ([]string, error) {
	// this makes them required for the model to successfully build
	if cfg.CameraName == "" {
		return nil, fmt.Errorf(`expected "camera_name" attribute for object tracker %q`, path)
	}

	// Return the resource names so that newTracker can access them as dependencies.
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
	images, _, err := h.cam.Images(ctx)
	if err != nil {
		h.logger.Info("inside DetectionsFromCamera, failed to get images though")
		// return nil, err
		return nil, fmt.Errorf("error1.0")
	}

	var depthImg image.Image
	for _, img := range images {
		if img.SourceName == "color" {
			depthImg = img.Image
		}
	}
	circles, err := vesselCircles(depthImg)
	if err != nil {
		h.logger.Info("vesselCircles returned an error though")
		return nil, fmt.Errorf("error1.1")
	}
	var detections []objdet.Detection
	for i, c := range circles {
		minX := c.center.X - (c.radius / 2)
		maxX := c.center.X + (c.radius / 2)
		minY := c.center.Y - (c.radius / 2)
		maxY := c.center.Y + (c.radius / 2)
		rect := image.Rectangle{
			Min: image.Point{X: minX, Y: minY},
			Max: image.Point{X: maxX, Y: maxY},
		}
		name := "circle-" + strconv.Itoa(i)
		detections = append(detections, objdet.NewDetection(rect, 100, name))
	}
	h.logger.Info("we have returned our detections, so things seems to be going fine?")
	return detections, nil
}

func (h *myHoughTransformer) Detections(ctx context.Context, img image.Image, extra map[string]interface{}) ([]objdet.Detection, error) {
	images, _, err := h.cam.Images(ctx)
	if err != nil {
		return nil, err
	}

	var depthImg image.Image
	for _, img := range images {
		if img.SourceName == "depth" {
			depthImg = img.Image
		}
	}
	circles, err := vesselCircles(depthImg)
	if err != nil {
		return nil, err
	}
	var detections []objdet.Detection
	for i, c := range circles {
		minX := c.center.X - (c.radius / 2)
		maxX := c.center.X + (c.radius / 2)
		minY := c.center.Y - (c.radius / 2)
		maxY := c.center.Y + (c.radius / 2)
		rect := image.Rectangle{
			Min: image.Point{X: minX, Y: minY},
			Max: image.Point{X: maxX, Y: maxY},
		}
		name := "circle-" + strconv.Itoa(i)
		detections = append(detections, objdet.NewDetection(rect, 100, name))
	}
	return detections, nil
}

func (h *myHoughTransformer) ClassificationsFromCamera(
	ctx context.Context,
	cameraName string,
	n int,
	extra map[string]interface{},
) (classification.Classifications, error) {
	return nil, fmt.Errorf("error1")
}

func (h *myHoughTransformer) Classifications(ctx context.Context, img image.Image,
	n int, extra map[string]interface{},
) (classification.Classifications, error) {
	return nil, fmt.Errorf("error2")
}

func (h *myHoughTransformer) GetProperties(ctx context.Context, extra map[string]interface{}) (*vision.Properties, error) {
	return nil, errUnimplemented
}
func (h *myHoughTransformer) GetObjectPointClouds(
	ctx context.Context,
	cameraName string,
	extra map[string]interface{},
) ([]*vis.Object, error) {
	return nil, fmt.Errorf("error3")
}

func (h *myHoughTransformer) CaptureAllFromCamera(
	ctx context.Context,
	cameraName string,
	opt viscapture.CaptureOptions,
	extra map[string]interface{},
) (viscapture.VisCapture, error) {
	h.logger.Info("HELLO THERE WE ARE INSIDE CAPTURE ALL FROM CAMERA")
	h.logger.Info("sanity hello")
	dets, err := h.DetectionsFromCamera(ctx, cameraName, extra)
	if err != nil {
		h.logger.Info("returning error here 1")
		return viscapture.VisCapture{}, fmt.Errorf("error4.00000")
	}
	images, _, err := h.cam.Images(ctx)
	if err != nil {
		h.logger.Info("returning error here 2")
		// return nil, err
		return viscapture.VisCapture{}, fmt.Errorf("error1.0.1111")
	}
	var colorImg image.Image
	for _, img := range images {
		if img.SourceName == "color" {
			colorImg = img.Image
		}
	}
	for _, d := range dets {
		h.logger.Infof("THIS IS THE DETECTION: %v", d)
	}
	h.logger.Info("we got our image, will return now")
	return viscapture.VisCapture{
		Image:           colorImg,
		Detections:      dets,
		Classifications: classification.Classifications{},
	}, nil
}

func (h *myHoughTransformer) Close(ctx context.Context) error {
	return nil
}

// DoCommand will return the slowest, fastest, and average time of the tracking module
func (h *myHoughTransformer) DoCommand(ctx context.Context, cmd map[string]interface{}) (map[string]interface{}, error) {
	return nil, errors.New("hello there")
}
