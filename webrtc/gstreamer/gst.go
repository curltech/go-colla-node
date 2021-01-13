package gstreamer

import (
	"github.com/curltech/go-colla-node/webrtc/gstreamer/sink"
	"github.com/pion/webrtc/v3"
)

var pipeline *sink.Pipeline

func createSinkPipeline(codecName string) {
	pipeline = sink.CreatePipeline(codecName)
	pipeline.Start()
}

func send(data []byte) {
	pipeline.Push(data)
}

func createReceivePipeline(codecName string, audioTrack *webrtc.TrackLocalStaticSample) {
	// Start pushing buffers on these tracks
	//receive.CreatePipeline("opus", []*webrtc.TrackLocalStaticSample{audioTrack}, *audioSrc).Start()
	//receive.CreatePipeline("vp8", []*webrtc.TrackLocalStaticSample{firstVideoTrack, secondVideoTrack}, *videoSrc).Start()
}
