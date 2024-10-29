// Package main is a module which implements the hough transform
package main

import (
	"hough"

	"go.viam.com/rdk/module"
	"go.viam.com/rdk/resource"
	"go.viam.com/rdk/services/vision"
)

func main() {
	module.ModularMain(
		hough.ModelName,
		resource.APIModel{API: vision.API, Model: hough.Model},
	)
}
