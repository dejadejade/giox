package player

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"io"
	"log"
	"runtime"
	"sync"
	"time"
	"unsafe"

	"github.com/hajimehoshi/oto"
	"golang.org/x/sync/errgroup"
)

var once sync.Once
var audioCtx *oto.Context

func InitDriver() {
}

type driver struct {
	format         *AVFormatContext
	videoStream    *AVStream
	audioStream    *AVStream
	videoCodecCtx  *AVCodecContext
	audioCodecCtx  *AVCodecContext
	videoStreamIdx int
	audioStreamIdx int

	sws uintptr // *C.SwsContext
	swr uintptr // *C.SwrContext

	videoPacketQ chan *AVPacket
	audioPacketQ chan *AVPacket

	duration int64

	videoCurrentPTSTime int64
	videoCurrentPTS     float64
	frameLastPTS        float64
	frameLastDelay      float64
	frameTimer          float64

	videoClock float64
	audioClock float64

	pictureQ chan Picture

	onPicture   func(Picture)
	onStartStop func(bool)

	//	audioCtx    *oto.Context
	audioPlayer *oto.Player
	audioQ      chan AudioSample

	file  string
	done  chan command
	timer *time.Timer

	audioStopChan chan bool

	mutex   sync.Mutex
	started bool
	ctx     context.Context
	cancel  context.CancelFunc
	cgroup  *errgroup.Group
}

func newDriver(file string, onPicture func(Picture), onStartStop func(bool)) *driver {
	return &driver{file: file, videoStreamIdx: -1, audioStreamIdx: -1, done: make(chan command), onPicture: onPicture, onStartStop: onStartStop}
}

func (d *driver) getVideoClock() float64 {
	delta := float64(AVGetTime()-d.videoCurrentPTSTime) / 1000000.
	return d.videoCurrentPTS + delta
}

func (d *driver) syncVideo(frame *AVFrame, pts float64) float64 {
	if pts != 0 {
		d.videoClock = pts
	} else {
		pts = d.videoClock
	}
	delay := frame.Delay(d.videoStream.TimeBase())
	d.videoClock += delay

	return pts
}

func (d *driver) decodeThread() error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	var err error
loop:
	for {
		select {
		case <-d.ctx.Done():
			break loop
		default:
		}

		var pkt *AVPacket
		pkt, err = d.format.ReadFrame()
		if err != nil {
			break
		}
		if pkt == nil {
			continue
		}
		idx := pkt.StreamIndex()
		if idx == d.videoStreamIdx {
			select {
			case d.videoPacketQ <- pkt:
			case <-d.ctx.Done():
				break loop
			}
		} else if idx == d.audioStreamIdx {
			select {
			case d.audioPacketQ <- pkt:
			case <-d.ctx.Done():
				break loop
			}
		} else {
			log.Printf("dropping pket")
			pkt.Free()
		}
	}

	close(d.videoPacketQ)
	close(d.audioPacketQ)
	go d.stop()
	return err
}

func (d *driver) videoThread() error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	frame := NewFrame()
	defer frame.Release()

	var err error
loop:
	for {
		select {
		case <-d.ctx.Done():
			break loop
		default:
		}

		pkt := <-d.videoPacketQ
		err := d.videoCodecCtx.SendPacket(pkt)
		pkt.Free()

		if err != nil {
			log.Printf("Failed to avcodec_send_packet\n")
			break
		}

		for err == nil {
			var vframe *AVFrame
			vframe, err = d.videoCodecCtx.ReceiveFrame(frame)
			if err != nil {
				break loop
			}
			if vframe == nil {
				break
			}

			pts := frame.BestEffortTimestamp()
			dpts := float64(pts) * d.videoStream.TimeBase() //d.videoTimeBase
			dpts = d.syncVideo(frame, dpts)

			width, height := d.videoStream.CodecPar().width, d.videoStream.CodecPar().height
			hsub := int32(1)

			var ximg image.Image
			var r image.YCbCrSubsampleRatio = -1
			switch d.videoStream.CodecPar().format {
			case AV_PIX_FMT_YUV420P:
				r = image.YCbCrSubsampleRatio420
				hsub = 2
			case AV_PIX_FMT_YUV422P:
				r = image.YCbCrSubsampleRatio422
			case AV_PIX_FMT_YUV444P:
				r = image.YCbCrSubsampleRatio444
			case AV_PIX_FMT_YUV440P:
				r = image.YCbCrSubsampleRatio440
				hsub = 2
			case AV_PIX_FMT_YUV411P:
				r = image.YCbCrSubsampleRatio411
			case AV_PIX_FMT_YUV410P:
				r = image.YCbCrSubsampleRatio410
				hsub = 2
			}

			//		log.Printf("%p %p %p %d %d %d\n", ybuf, ubuf, vbuf, ylen, ulen, vlen)
			if r >= 0 {
				if vframe.data[0] == 0 || vframe.data[1] == 0 || vframe.data[2] == 0 {
					log.Printf("Invalid data ptr: %v", vframe.data)
					break
				}

				img := image.NewYCbCr(image.Rect(0, 0, int(width), int(height)), r)
				img.YStride = int(vframe.linesize[0])
				img.CStride = int(vframe.linesize[1])
				img.Y = GoBytes(unsafe.Pointer(vframe.data[0]), int(height*vframe.linesize[0]))
				img.Cb = GoBytes(unsafe.Pointer(vframe.data[1]), int(height/hsub*vframe.linesize[1]))
				img.Cr = GoBytes(unsafe.Pointer(vframe.data[2]), int(height/hsub*vframe.linesize[2]))
				ximg = img
			}

			if ximg == nil {
				img, err := frame.ToRGBA(d.sws)
				if err != nil {
					log.Printf("Failed to copy_image\n")
					break
				}
				ximg = img
			}

			select {
			case d.pictureQ <- Picture{img: ximg, pts: float64(dpts)}:
			case <-d.ctx.Done():
				break loop
			}
		}
	}

	close(d.pictureQ)
	return err
}

func (d *driver) audioThread() error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	frame := NewFrame()
	defer frame.Release()

	var err error
	cp := d.audioStream.CodecPar()
loop:
	for {
		select {
		case <-d.ctx.Done():
			break loop
		default:
		}

		pkt := <-d.audioPacketQ
		if pkt == nil {
			break
		}

		pts := pkt.PTS(d.audioStream.TimeBase())
		if pts != 0 {
			d.audioClock = pts
		}

		err = d.audioCodecCtx.SendPacket(pkt)
		pkt.Free()

		if err != nil {
			log.Printf("Failed to avcodec_send_packet\n")
			break
		}

		for err == nil {
			var aframe *AVFrame
			aframe, err = d.audioCodecCtx.ReceiveFrame(frame)
			if err != nil {
				break loop
			}
			if aframe == nil {
				break
			}

			// got frame
			data, nr, _ := frame.ToSamples(d.swr)

			//			pts := d.audioClock
			d.audioClock += float64(len(data)) / (float64(cp.sample_rate) * 2 * 2)
			select {
			case d.audioQ <- AudioSample{data: data, pts: float64(pts), numSamples: int(nr)}:
			case <-d.ctx.Done():
				break loop
			}
		}
	}

	close(d.audioQ)
	return err
}

func (d *driver) refreshPicture() {
	select {
	case pic := <-d.pictureQ:
		d.videoCurrentPTS = pic.pts
		d.videoCurrentPTSTime = AVGetTime()
		delay := pic.pts - d.frameLastPTS
		if delay <= 0 || delay >= 1.0 {
			delay = d.frameLastDelay
		} else {
			d.frameLastDelay = delay
		}
		d.frameLastPTS = pic.pts

		d.frameTimer += delay
		actualDelay := d.frameTimer - float64(AVGetTime())/1000000.
		if actualDelay < .01 {
			actualDelay = .01
		}

		d.onPicture(pic)
		/*
			select {
			case d.outputQ <- pic:
			case <-d.ctx.Done():
				return
			}
		*/
		d.timer = time.NewTimer(time.Millisecond * time.Duration(int(actualDelay*1000+.5)))
		// d.timer = time.NewTimer(time.Millisecond * 20)

	default:
		d.timer = time.NewTimer(time.Millisecond)
	}
}

func (d *driver) audioOutThread(stopChan chan bool) error {
loop:
	for {
		select {
		case sample := <-d.audioQ:
			io.Copy(d.audioPlayer, bytes.NewReader(sample.data))

		case <-stopChan:
			break loop

		case <-d.ctx.Done():
			break loop
		}
	}

	return nil
}

func (d *driver) videoOutThread() error {
	d.timer = time.NewTimer(time.Millisecond * 20)
loop:
	for {
		select {
		case <-d.ctx.Done():
			break loop
		default:
		}

		timer := d.timer
		select {
		case cmd := <-d.done:
			switch cmd {
			case cmdPlay:
				println("playing")
				d.refreshPicture()
				d.audioStopChan = make(chan bool)
				d.cgroup.Go(func() error {
					d.audioOutThread(d.audioStopChan)
					return nil
				})
			case cmdPause:
				println("pausing")
				timer.Stop()
				close(d.audioStopChan)
			}

		case <-d.ctx.Done():
			timer.Stop()
			break loop

		case <-timer.C:
			d.refreshPicture()
		}
	}

	return nil
}

var errStarted = errors.New("already started")

func (d *driver) start() error {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	if d.started {
		return errStarted
	}

	var err error
	d.format, err = OpenInput(d.file)
	if err != nil {
		return err
	}

	d.videoStream, d.audioStream, d.videoStreamIdx, d.audioStreamIdx, d.videoCodecCtx, d.audioCodecCtx, err = d.format.OpenStreams()
	if err != nil {
		return err
	}

	log.Printf("Stream: %d, %#v, %d, %#v\n", d.videoStreamIdx, d.videoStream, d.audioStreamIdx, d.audioStream)

	once.Do(func() {
		var err error
		audioCtx, err = oto.NewContext(int(d.audioStream.CodecPar().sample_rate), 2, 2, 4096)
		if err != nil {
			log.Printf("Failed to create audio context")
		}
	})

	if audioCtx == nil {
		return fmt.Errorf("No audio")
	}

	d.audioPlayer = audioCtx.NewPlayer()

	cp := d.videoStream.CodecPar()
	d.duration = int64(float64(d.videoStream.duration)*d.videoStream.TimeBase()) * 1000000

	d.videoCurrentPTSTime = AVGetTime()
	d.frameTimer = float64(d.videoCurrentPTSTime) / 1000000
	d.frameLastDelay = 40e-3
	d.sws = GetSWSContext(cp.width, cp.height, cp.format)
	d.swr = GetSWRContext(d.audioStream.CodecPar().sample_rate, d.audioStream.CodecPar().channel_layout, d.audioStream.CodecPar().format)

	log.Printf("Opened stream %dx%d, pixfmt: %d\n", cp.width, cp.height, cp.format)

	d.videoPacketQ = make(chan *AVPacket)
	d.audioPacketQ = make(chan *AVPacket)
	d.pictureQ = make(chan Picture)
	d.audioQ = make(chan AudioSample, 1000)
	d.audioStopChan = make(chan bool)

	d.ctx, d.cancel = context.WithCancel(context.Background())
	d.cgroup, d.ctx = errgroup.WithContext(d.ctx)
	d.cgroup.Go(d.decodeThread)
	d.cgroup.Go(d.videoThread)
	d.cgroup.Go(d.audioThread)
	d.cgroup.Go(func() error {
		d.audioOutThread(d.audioStopChan)
		return nil
	})
	d.cgroup.Go(d.videoOutThread)

	log.Printf("Started\n")
	go d.onStartStop(true)

	d.started = true
	return nil
}

func (d *driver) stop() {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	if !d.started {
		log.Printf("Not started\n")
		return
	}

	if d.cancel != nil {
		d.cancel()
	}

	for pkt, ok := <-d.videoPacketQ; ok; pkt, ok = <-d.videoPacketQ {
		pkt.Free()
	}
	for pkt, ok := <-d.audioPacketQ; ok; pkt, ok = <-d.audioPacketQ {
		pkt.Free()
	}

	// wait for threads to exit
	d.cgroup.Wait()

	FreeSWSContext(d.sws)
	d.sws = 0
	FreeSWRContext(d.swr)
	d.swr = 0

	d.videoCodecCtx.Release()
	d.videoCodecCtx = nil

	d.audioCodecCtx.Release()
	d.audioCodecCtx = nil

	d.format.Release()
	d.format = nil

	if d.audioPlayer != nil {
		d.audioPlayer.Close()
	}

	//d.audioCtx.Close()
	log.Printf("stopped\n")
	d.started = false
	d.ctx = nil
	d.cancel = nil
	go d.onStartStop(false)
}

func (format *AVFormatContext) OpenStreams() (vstream, astream *AVStream, videoIdx, audioIdx int, vcodec, acodec *AVCodecContext, err error) {
	err = format.FindInfo()
	if err != nil {
		return
	}

	for i := 0; i < int(format.nb_streams); i++ {
		stream := format.Stream(i)
		if stream == nil {
			continue
		}

		codecpar := stream.CodecPar()
		if codecpar == nil {
			continue
		}

		if codecpar.codec_type == AVMEDIA_TYPE_VIDEO && vstream == nil {
			videoIdx = i
			vstream = stream
			continue
		}

		if codecpar.codec_type == AVMEDIA_TYPE_AUDIO && astream == nil {
			audioIdx = i
			astream = stream
		}
	}

	if vstream == nil || astream == nil {
		err = fmt.Errorf("failed to open streams")
		return
	}

	vcodec, err = vstream.Open()
	if err != nil {
		return
	}

	acodec, err = astream.Open()
	if err != nil {
		return
	}
	return
}
