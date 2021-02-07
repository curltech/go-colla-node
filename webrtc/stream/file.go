package stream

import (
	"bytes"
	"github.com/at-wat/ebml-go/webm"
	"github.com/curltech/go-colla-core/logger"
	"github.com/pion/rtp"
	"github.com/pion/rtp/codecs"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
	"github.com/pion/webrtc/v3/pkg/media/h264reader"
	"github.com/pion/webrtc/v3/pkg/media/ivfreader"
	"github.com/pion/webrtc/v3/pkg/media/ivfwriter"
	"github.com/pion/webrtc/v3/pkg/media/oggreader"
	"github.com/pion/webrtc/v3/pkg/media/oggwriter"
	"github.com/pion/webrtc/v3/pkg/media/samplebuilder"
	"golang.org/x/image/vp8"
	"image/jpeg"
	"io"
	"os"
	"time"
)

/**
读取ivf视频文件到输出流
videoTrack, err := webrtc.NewTrackLocalStaticSample(webrtc.RTPCodecCapability{MimeType: "video/h264"}, "video", "pion")
videoTrack, err := webrtc.NewTrackLocalStaticSample(webrtc.RTPCodecCapability{MimeType: "video/vp8"}, "video", "pion")
*/
func readVideo(videoFileName string, videoTrack *webrtc.TrackLocalStaticSample) {
	// Assert that we have an audio or video file
	_, err := os.Stat(videoFileName)
	haveVideoFile := !os.IsNotExist(err)
	if !haveVideoFile {
		logger.Sugar.Errorf("Could not find `" + videoFileName + "`")
		return
	}

	if haveVideoFile {
		go func() {
			// Open a IVF file and start reading using our IVFReader
			file, ivfErr := os.Open(videoFileName)
			if ivfErr != nil {
				logger.Sugar.Errorf(ivfErr.Error())
				return
			}

			ivf, header, ivfErr := ivfreader.NewWith(file)
			if ivfErr != nil {
				logger.Sugar.Errorf(ivfErr.Error())
				return
			}
			// Send our video file frame at a time. Pace our sending so we send it at the same speed it should be played back as.
			// This isn't required since the video is timestamped, but we will such much higher loss if we send all at once.
			sleepTime := time.Millisecond * time.Duration((float32(header.TimebaseNumerator)/float32(header.TimebaseDenominator))*1000)
			for {
				frame, _, ivfErr := ivf.ParseNextFrame()
				if ivfErr == io.EOF {
					logger.Sugar.Infof("All video frames parsed and sent")
					return
				}

				if ivfErr != nil {
					logger.Sugar.Errorf(ivfErr.Error())
					return
				}
				frame = encrypt(frame)

				time.Sleep(sleepTime)
				if ivfErr = videoTrack.WriteSample(media.Sample{Data: frame, Duration: time.Second}); ivfErr != nil {
					logger.Sugar.Errorf(ivfErr.Error())
					return
				}
			}
		}()
	}
}

func readH264Video(videoFileName string, videoTrack *webrtc.TrackLocalStaticSample) {
	// Assert that we have an audio or video file
	_, err := os.Stat(videoFileName)
	haveVideoFile := !os.IsNotExist(err)
	if !haveVideoFile {
		logger.Sugar.Errorf("Could not find `" + videoFileName + "`")
		return
	}

	if haveVideoFile {
		go func() {
			reader, readerError := h264reader.NewReader(os.Stdin)
			if readerError != nil {
				logger.Sugar.Errorf(readerError.Error())
				return
			}

			// h264reader doesn't pass back a header, but we know the framerate so I'm going to put it here
			sleepTime := time.Millisecond * time.Duration(16)
			for {
				nal, readerError := reader.NextNAL()
				if readerError == io.EOF {
					logger.Sugar.Infof("All video frames parsed and sent")
					return
				}

				if readerError != nil {
					logger.Sugar.Errorf(readerError.Error())
					return
				}

				if nal.UnitType != h264reader.NalUnitTypeSPS && nal.UnitType != h264reader.NalUnitTypePPS {
					time.Sleep(sleepTime)
				}
				if writeErr := videoTrack.WriteSample(media.Sample{Data: nal.Data, Duration: time.Second}); writeErr != nil {
					logger.Sugar.Errorf(writeErr.Error())
					return
				}
			}
		}()
	}
}

/**
加密的模拟方法
*/
func encrypt(frame []byte) []byte {
	// Encrypt video using XOR Cipher
	for i := range frame {
		frame[i] ^= 0
	}

	return frame
}

/**
读取ogg格式的音频文件到输出流
*/
func readAudio(audioFileName string, audioTrack *webrtc.TrackLocalStaticSample) {
	_, err := os.Stat(audioFileName)
	haveAudioFile := !os.IsNotExist(err)
	if !haveAudioFile {
		logger.Sugar.Errorf("Could not find `" + audioFileName)
		return
	}
	if haveAudioFile {
		go func() {
			// Open a IVF file and start reading using our IVFReader
			file, oggErr := os.Open(audioFileName)
			if oggErr != nil {
				logger.Sugar.Errorf(oggErr.Error())
				return
			}

			// Open on oggfile in non-checksum mode.
			ogg, _, oggErr := oggreader.NewWith(file)
			if oggErr != nil {
				logger.Sugar.Errorf(oggErr.Error())
				return
			}

			// Keep track of last granule, the difference is the amount of samples in the buffer
			var lastGranule uint64
			for {
				pageData, pageHeader, oggErr := ogg.ParseNextPage()
				if oggErr == io.EOF {
					logger.Sugar.Errorf("All audio pages parsed and sent")
					return
				}

				if oggErr != nil {
					logger.Sugar.Errorf(oggErr.Error())
					return
				}

				// The amount of samples is the difference between the last and current timestamp
				sampleCount := float64(pageHeader.GranulePosition - lastGranule)
				lastGranule = pageHeader.GranulePosition
				sampleDuration := time.Duration((sampleCount/48000)*1000) * time.Millisecond

				if oggErr = audioTrack.WriteSample(media.Sample{Data: pageData, Duration: sampleDuration}); oggErr != nil {
					panic(oggErr)
				}

				time.Sleep(sampleDuration)
			}
		}()
	}
}

/**
写ogg音频流到文件
*/
func writeAudio(audioFileName string, track *webrtc.TrackRemote) {
	oggFile, err := oggwriter.New(audioFileName, 48000, 2)
	if err != nil {
		return
	}
	codec := track.Codec()
	if codec.MimeType == "audio/opus" {
		logger.Sugar.Infof("Got Opus track, saving to disk as output.opus (48 kHz, 2 channels)")
		saveToDisk(oggFile, track)
	}
}

/**
写ivf视频流到文件
*/
func writeVideo(videoFileName string, track *webrtc.TrackRemote) {
	ivfFile, err := ivfwriter.New(videoFileName)
	if err != nil {
		logger.Sugar.Errorf(err.Error())
		return
	}
	codec := track.Codec()
	if codec.MimeType == "video/VP8" {
		logger.Sugar.Infof("Got VP8 track, saving to disk as output.ivf")
		saveToDisk(ivfFile, track)
	}
}

/**
写入磁盘文件
*/
func saveToDisk(i media.Writer, track *webrtc.TrackRemote) {
	defer func() {
		if err := i.Close(); err != nil {
			logger.Sugar.Errorf(err.Error())
			return
		}
	}()

	for {
		rtpPacket, _, err := track.ReadRTP()
		if err != nil {
			panic(err)
		}
		if err := i.WriteRTP(rtpPacket); err != nil {
			logger.Sugar.Errorf(err.Error())
			return
		}
	}
}

/**
将远程的媒体webm格式保存到磁盘
*/
type webmSaver struct {
	audioWriter, videoWriter       webm.BlockWriteCloser
	audioBuilder, videoBuilder     *samplebuilder.SampleBuilder
	audioTimestamp, videoTimestamp time.Duration
	filename                       string
}

func newWebmSaver(filename string) *webmSaver {
	saver := &webmSaver{
		audioBuilder: samplebuilder.New(10, &codecs.OpusPacket{}, 48000),
		videoBuilder: samplebuilder.New(10, &codecs.VP8Packet{}, 90000),
	}
	saver.filename = filename

	return saver
}

func (s *webmSaver) Close() error {
	logger.Sugar.Infof("Finalizing webm...\n")
	if s.audioWriter != nil {
		if err := s.audioWriter.Close(); err != nil {
			logger.Sugar.Errorf(err.Error())

			return err
		}
	}
	if s.videoWriter != nil {
		if err := s.videoWriter.Close(); err != nil {
			logger.Sugar.Errorf(err.Error())

			return err
		}
	}

	return nil
}

func (s *webmSaver) PushOpus(rtpPacket *rtp.Packet) error {
	s.audioBuilder.Push(rtpPacket)

	for {
		sample := s.audioBuilder.Pop()
		if sample == nil {
			return nil
		}
		if s.audioWriter != nil {
			s.audioTimestamp += sample.Duration
			if _, err := s.audioWriter.Write(true, int64(s.audioTimestamp/time.Millisecond), sample.Data); err != nil {
				logger.Sugar.Errorf(err.Error())

				return err
			}
		}
	}

	return nil
}

func (this *webmSaver) PushVP8(rtpPacket *rtp.Packet) error {
	this.videoBuilder.Push(rtpPacket)

	for {
		sample := this.videoBuilder.Pop()
		if sample == nil {
			return nil
		}
		// Read VP8 header.
		videoKeyframe := (sample.Data[0]&0x1 == 0)
		if videoKeyframe {
			// Keyframe has frame information.
			raw := uint(sample.Data[6]) | uint(sample.Data[7])<<8 | uint(sample.Data[8])<<16 | uint(sample.Data[9])<<24
			width := int(raw & 0x3FFF)
			height := int((raw >> 16) & 0x3FFF)

			if this.videoWriter == nil || this.audioWriter == nil {
				// Initialize WebM saver using received frame size.
				err := this.InitWriter(width, height)
				if err != nil {
					logger.Sugar.Errorf(err.Error())

					return err
				}
			}
		}
		if this.videoWriter != nil {
			this.videoTimestamp += sample.Duration
			if _, err := this.videoWriter.Write(videoKeyframe, int64(this.audioTimestamp/time.Millisecond), sample.Data); err != nil {
				logger.Sugar.Errorf(err.Error())

				return err
			}
		}
	}

	return nil
}

func (this *webmSaver) InitWriter(width, height int) error {
	w, err := os.OpenFile(this.filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		logger.Sugar.Errorf(err.Error())

		return err
	}

	ws, err := webm.NewSimpleBlockWriter(w,
		[]webm.TrackEntry{
			{
				Name:            "Audio",
				TrackNumber:     1,
				TrackUID:        12345,
				CodecID:         "A_OPUS",
				TrackType:       2,
				DefaultDuration: 20000000,
				Audio: &webm.Audio{
					SamplingFrequency: 48000.0,
					Channels:          2,
				},
			}, {
				Name:            "Video",
				TrackNumber:     2,
				TrackUID:        67890,
				CodecID:         "V_VP8",
				TrackType:       1,
				DefaultDuration: 33333333,
				Video: &webm.Video{
					PixelWidth:  uint64(width),
					PixelHeight: uint64(height),
				},
			},
		})
	if err != nil {
		logger.Sugar.Errorf(err.Error())

		return err
	}
	logger.Sugar.Infof("WebM saver has started with video width=%d, height=%d\n", width, height)
	this.audioWriter = ws[0]
	this.videoWriter = ws[1]

	return nil
}

func (this *webmSaver) push(track *webrtc.TrackRemote) error {
	// Read RTP packets being sent to Pion
	rtp, _, readErr := track.ReadRTP()
	if readErr != nil {
		return readErr
	}
	switch track.Kind() {
	case webrtc.RTPCodecTypeAudio:
		this.PushOpus(rtp)
	case webrtc.RTPCodecTypeVideo:
		this.PushVP8(rtp)
	}

	return nil
}

func Save(filename string, audioTrack *webrtc.TrackRemote, videoTrack *webrtc.TrackRemote) error {
	saver := newWebmSaver(filename)
	defer saver.Close()
	for {
		err := saver.push(audioTrack)
		if err != nil {
			return err
		}
		err = saver.push(videoTrack)
		if err != nil {
			return err
		}
	}
}

/**
convert incoming video frames to jpeg
*/
func Snapshot(rtpPacket *rtp.Packet) (*bytes.Buffer, error) {
	// Initialized with 20 maxLate, my samples sometimes 10-15 packets
	sampleBuilder := samplebuilder.New(20, &codecs.VP8Packet{}, 90000)
	decoder := vp8.NewDecoder()

	for {
		// Pull RTP Packet from rtpChan
		sampleBuilder.Push(rtpPacket)

		// Use SampleBuilder to generate full picture from many RTP Packets
		sample := sampleBuilder.Pop()
		if sample == nil {
			continue
		}

		// Read VP8 header.
		videoKeyframe := (sample.Data[0]&0x1 == 0)
		if !videoKeyframe {
			continue
		}

		// Begin VP8-to-image decode: Init->DecodeFrameHeader->DecodeFrame
		decoder.Init(bytes.NewReader(sample.Data), len(sample.Data))

		// Decode header
		if _, err := decoder.DecodeFrameHeader(); err != nil {
			return nil, err
		}

		// Decode Frame
		img, err := decoder.DecodeFrame()
		if err != nil {
			return nil, err
		}

		// Encode to (RGB) jpeg
		buffer := new(bytes.Buffer)
		if err = jpeg.Encode(buffer, img, nil); err != nil {
			return nil, err
		}

		return buffer, nil
	}
	return nil, nil
}
