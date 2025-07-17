package main

import (
	"fmt"
	"gocv.io/x/gocv"
)

func testBackend(backend gocv.NetBackendType, target gocv.NetTargetType, label string) {
	net := gocv.ReadNet("frozen_east_text_detection.pb", "") // або будь-яка інша модель
	defer net.Close()

	err := net.SetPreferableBackend(backend)
	if err != nil {
		fmt.Printf("[FAIL] Backend %s: %v\n", label, err)
		return
	}

	err = net.SetPreferableTarget(target)
	if err != nil {
		fmt.Printf("[FAIL] Target %s: %v\n", label, err)
		return
	}

	fmt.Printf("[OK] %s: працює\n", label)
}

func main() {
	fmt.Println("OpenCV version:", gocv.Version())

	testBackend(gocv.NetBackendDefault, gocv.NetTargetCPU, "Default / CPU")
	testBackend(gocv.NetBackendCUDA, gocv.NetTargetCUDA, "CUDA / GPU")
	//testBackend(gocv.NetBackendOpenCL, gocv.NetTargetOpenCL, "OpenCL / GPU")
	//testBackend(gocv.NetBackendInferenceEngine, gocv.NetTargetCPU, "OpenVINO / CPU")
	//testBackend(gocv.NetBackendInferenceEngine, gocv.NetTargetMyriad, "OpenVINO / NCS2")
}
