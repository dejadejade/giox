// +build !windows,!ios,!android

package player

// #cgo LDFLAGS: -lavcodec -lavformat -lavutil -lswscale -lswresample
// #include <libavcodec/avcodec.h>
// #include <libavformat/avformat.h>
// #include <libavutil/time.h>
// #include <libavutil/imgutils.h>
// #include <libswscale/swscale.h>
// #include <libswresample/swresample.h>
// #include <libavutil/opt.h>
//
// #define AVERROR_EAGAIN AVERROR(EAGAIN)
// typedef enum AVPixelFormat AVPixelFormat;
// typedef struct SwsContext SwsContext;
import "C"

import (
	"fmt"
	"image"
	"log"
	"unsafe"
)

func AVGetTime() int64 {
	return int64(C.av_gettime())
}

func (format *AVFormatContext) ReadFrame() (*AVPacket, error) {
	pkt := C.av_packet_alloc()
	ret := C.av_read_frame((*C.AVFormatContext)(unsafe.Pointer(format)), pkt)
	if ret < 0 {
		/*
			if format.Error() == 0 {
				return nil, nil
			}
		*/
		return nil, fmt.Errorf("failed to read frame")
	}

	return (*AVPacket)(unsafe.Pointer(pkt)), nil
}

func (format *AVFormatContext) FindInfo() error {
	ret := C.avformat_find_stream_info((*C.AVFormatContext)(unsafe.Pointer(format)), nil)
	if ret < 0 {
		return fmt.Errorf("failed to find stream info")
	}

	C.av_dump_format((*C.AVFormatContext)(unsafe.Pointer(format)), 0, nil, 0)
	return nil
}

func (format *AVFormatContext) Release() {
	f := (*C.AVFormatContext)(unsafe.Pointer(format))
	C.avformat_close_input(&f)
	C.avformat_free_context(f)
}

func (pkt *AVPacket) Free() {
	p := (*C.AVPacket)(unsafe.Pointer(pkt))
	C.av_packet_free(&p)
}

func (codec *AVCodecContext) SendPacket(pkt *AVPacket) error {
	ret := C.avcodec_send_packet((*C.AVCodecContext)(unsafe.Pointer(codec)), (*C.AVPacket)(unsafe.Pointer(pkt)))
	if ret < 0 {
		return fmt.Errorf("failed to send packet")
	}
	return nil
}

func (codec *AVCodecContext) ReceiveFrame(frame *AVFrame) (*AVFrame, error) {
	ret := C.avcodec_receive_frame((*C.AVCodecContext)(unsafe.Pointer(codec)), (*C.AVFrame)(unsafe.Pointer(frame)))
	if ret == C.AVERROR_EAGAIN || ret == C.AVERROR_EOF {
		return nil, nil
	}
	if ret < 0 {
		return nil, fmt.Errorf("failed to receive frame")
	}

	return frame, nil
}

func (codec *AVCodecContext) Release() {
	c := (*C.AVCodecContext)(unsafe.Pointer(codec))
	C.avcodec_free_context(&c)
}

func (stream *AVStream) Open() (*AVCodecContext, error) {
	codec := C.avcodec_find_decoder(uint32(stream.CodecPar().codec_id))
	if codec == nil {
		return nil, fmt.Errorf("no codec")
	}
	codecCtx := C.avcodec_alloc_context3(codec)
	ret := C.avcodec_parameters_to_context(codecCtx, (*C.AVCodecParameters)(unsafe.Pointer(stream.CodecPar())))
	if ret != 0 {
		return nil, fmt.Errorf("failed to create codec context")
	}

	ret = C.avcodec_open2(codecCtx, codec, nil)
	if ret < 0 {
		return nil, fmt.Errorf("failed to open codec")
	}

	return (*AVCodecContext)(unsafe.Pointer(codecCtx)), nil
}

func OpenInput(file string) (*AVFormatContext, error) {
	var format *C.AVFormatContext
	ret := C.avformat_open_input(&format, C.CString(file), nil, nil)
	if ret < 0 {
		return nil, fmt.Errorf("Failed to open stream")
	}
	return (*AVFormatContext)(unsafe.Pointer(format)), nil
}

func NewFrame() *AVFrame {
	return (*AVFrame)(unsafe.Pointer(C.av_frame_alloc()))
}

func (frame *AVFrame) Release() {
	aframe := (*C.AVFrame)(unsafe.Pointer(frame))
	C.av_frame_free(&aframe)
}

func (frame *AVFrame) ToRGBA(sws uintptr) (*image.RGBA, error) {
	img := image.NewRGBA(image.Rect(0, 0, int(frame.width), int(frame.height)))

	linesizes := [1]int32{frame.width * 4}
	ret := C.sws_scale((*C.SwsContext)(unsafe.Pointer(sws)), (**C.uchar)(unsafe.Pointer(&frame.data[0])), (*C.int)(unsafe.Pointer(&frame.linesize[0])), 0,
		C.int(frame.height), (**C.uchar)(unsafe.Pointer(&img.Pix[0])), (*C.int)(unsafe.Pointer(&linesizes[0])))
	if ret < 0 {
		log.Printf("Failed to copy_image\n")
		return nil, fmt.Errorf("failed to sws_scale")
	}
	return img, nil
}

func (frame *AVFrame) ToSamples(swr uintptr) (data []byte, nr int, err error) {
	nb_channels := C.av_get_channel_layout_nb_channels(C.AV_CH_LAYOUT_STEREO)

	var buffer *C.uint8_t
	C.av_samples_alloc(&buffer, nil, nb_channels, C.int(frame.nb_samples), C.AV_SAMPLE_FMT_S16, 1)
	n := C.swr_convert((*C.SwrContext)(unsafe.Pointer(swr)), &buffer, C.int(frame.nb_samples), (**C.uint8_t)(unsafe.Pointer(&frame.data[0])), C.int(frame.nb_samples))
	data = C.GoBytes(unsafe.Pointer(buffer), C.int(n)*2*2)
	C.av_freep(unsafe.Pointer(&buffer))

	return data, int(n), nil
}

func GoBytes(buf unsafe.Pointer, len int) []byte {
	return C.GoBytes(buf, C.int(len))
}

func GetSWSContext(width, height int32, format int32) uintptr {
	w, h := C.int(width), C.int(height)
	sws := C.sws_getContext(w, h, format, w, h, C.AV_PIX_FMT_RGBA, C.SWS_BILINEAR, nil, nil, nil)
	return uintptr(unsafe.Pointer(sws))
}

func FreeSWSContext(sws uintptr) {
	C.sws_freeContext((*C.SwsContext)(unsafe.Pointer(sws)))
}

func GetSWRContext(sampleRate int32, channelLayout uint64, format int32) uintptr {
	swr := C.swr_alloc_set_opts(nil,
		C.AV_CH_LAYOUT_STEREO,
		C.AV_SAMPLE_FMT_S16,
		C.int(sampleRate),
		C.int64_t(channelLayout),
		format,
		C.int(sampleRate),
		0, nil)
	C.swr_init(swr)
	return uintptr(unsafe.Pointer(swr))
}

func FreeSWRContext(swr uintptr) {
	r := (*C.SwrContext)(unsafe.Pointer(swr))
	C.swr_free(&r)
}
