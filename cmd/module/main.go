// Package main is a module which implements the hough transform
package main

import (
	"go.viam.com/rdk/module"
	"go.viam.com/rdk/resource"
	"go.viam.com/rdk/services/vision"

	"github.com/nfranczak/viam-hough-transform/hough"
)

const moduleName = "Hough Transform Go Module"

func main() {
	module.ModularMain(
		moduleName,
		resource.APIModel{API: vision.API, Model: hough.Model},
	)
}
