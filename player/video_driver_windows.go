// +build windows

package player

import (
	"fmt"
	"image"
	"log"
	"syscall"
	"unsafe"
)

var (
	avcodec    = syscall.NewLazyDLL("avcodec-58.dll")
	avformat   = syscall.NewLazyDLL("avformat-58.dll")
	swscale    = syscall.NewLazyDLL("swscale-5.dll")
	swresample = syscall.NewLazyDLL("swresample-3.dll")
	avutil     = syscall.NewLazyDLL("avutil-56.dll")

	av_packet_alloc               = avcodec.NewProc("av_packet_alloc")
	av_packet_free                = avcodec.NewProc("av_packet_free")
	avcodec_send_packet           = avcodec.NewProc("avcodec_send_packet")
	avcodec_receive_frame         = avcodec.NewProc("avcodec_receive_frame")
	avcodec_find_decoder          = avcodec.NewProc("avcodec_find_decoder")
	avcodec_alloc_context3        = avcodec.NewProc("avcodec_alloc_context3")
	avcodec_parameters_to_context = avcodec.NewProc("avcodec_parameters_to_context")
	avcodec_open2                 = avcodec.NewProc("avcodec_open2")
	avcodec_free_context          = avcodec.NewProc("avcodec_free_context")

	av_read_frame             = avformat.NewProc("av_read_frame")
	av_dump_format            = avformat.NewProc("av_dump_format")
	avformat_find_stream_info = avformat.NewProc("avformat_find_stream_info")
	avformat_open_input       = avformat.NewProc("avformat_open_input")
	avformat_close_input      = avformat.NewProc("avformat_close_input")
	avformat_free_context     = avformat.NewProc("avformat_free_context")

	av_gettime                        = avutil.NewProc("av_gettime")
	av_get_channel_layout_nb_channels = avutil.NewProc("av_get_channel_layout_nb_channels")
	av_samples_alloc                  = avutil.NewProc("av_samples_alloc")
	av_frame_alloc                    = avutil.NewProc("av_frame_alloc")
	av_frame_free                     = avutil.NewProc("av_frame_free")
	av_freep                          = avutil.NewProc("av_freep")

	sws_scale          = swscale.NewProc("sws_scale")
	sws_getContext     = swscale.NewProc("sws_getContext")
	sws_freeContext    = swscale.NewProc("sws_freeContext")
	swr_alloc_set_opts = swresample.NewProc("swr_alloc_set_opts")
	swr_convert        = swresample.NewProc("swr_convert")
	swr_init           = swresample.NewProc("swr_init")
	swr_free           = swresample.NewProc("swr_free")
)

func AVGetTime() int64 {
	ret, _, _ := av_gettime.Call()
	return int64(ret)
}

func (format *AVFormatContext) ReadFrame() (*AVPacket, error) {
	pkt, _, _ := av_packet_alloc.Call()
	ret, _, _ := av_read_frame.Call(uintptr(unsafe.Pointer(format)), pkt)
	if int32(ret) < 0 {
		if format.Error() == 0 {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read frame")
	}

	return (*AVPacket)(unsafe.Pointer(pkt)), nil
}

func (format *AVFormatContext) FindInfo() error {
	ret, _, _ := avformat_find_stream_info.Call(uintptr(unsafe.Pointer(format)), 0)
	if int32(ret) < 0 {
		return fmt.Errorf("failed to find stream info")
	}

	av_dump_format.Call(uintptr(unsafe.Pointer(format)), 0, 0, 0)
	return nil
}

func (codec *AVCodecContext) Release() {
	avcodec_free_context.Call(uintptr(unsafe.Pointer(&codec)))
}

func (pkt *AVPacket) Free() {
	av_packet_free.Call(uintptr(unsafe.Pointer(&pkt)))
}

func (format *AVFormatContext) Release() {
	avformat_close_input.Call(uintptr(unsafe.Pointer(&format)))
	avformat_free_context.Call(uintptr(unsafe.Pointer(format)))
}

func (codec *AVCodecContext) SendPacket(pkt *AVPacket) error {
	ret, _, _ := avcodec_send_packet.Call(uintptr(unsafe.Pointer(codec)), uintptr(unsafe.Pointer(pkt)))
	if int32(ret) < 0 {
		return fmt.Errorf("failed to send packet")
	}
	return nil
}

func (codec *AVCodecContext) ReceiveFrame(frame *AVFrame) (*AVFrame, error) {
	ret, _, _ := avcodec_receive_frame.Call(uintptr(unsafe.Pointer(codec)), uintptr(unsafe.Pointer(frame)))
	if int32(ret) < 0 {
		if int32(ret) == AVERROR_EAGAIN || int32(ret) == AVERROR_EOF {
			return nil, nil
		}
		if ret < 0 {
			return nil, fmt.Errorf("failed to receive frame")
		}
	}

	return frame, nil
}

func (stream *AVStream) Open() (*AVCodecContext, error) {
	codec, _, _ := avcodec_find_decoder.Call(uintptr(stream.CodecPar().codec_id))
	if codec == 0 {
		return nil, fmt.Errorf("no codec")
	}
	codecCtx, _, _ := avcodec_alloc_context3.Call(codec)
	ret, _, _ := avcodec_parameters_to_context.Call(codecCtx, uintptr(unsafe.Pointer(stream.CodecPar())))
	if int32(ret) < 0 {
		return nil, fmt.Errorf("failed to create codec context")
	}

	ret, _, _ = avcodec_open2.Call(codecCtx, codec, 0)
	if int32(ret) < 0 {
		return nil, fmt.Errorf("failed to open codec")
	}

	return (*AVCodecContext)(unsafe.Pointer(codecCtx)), nil
}

func OpenInput(file string) (*AVFormatContext, error) {
	fn := append([]byte(file), 0)
	var format *AVFormatContext
	ret, _, _ := avformat_open_input.Call(uintptr(unsafe.Pointer(&format)), uintptr(unsafe.Pointer(&fn[0])), 0, 0)
	if int32(ret) < 0 {
		return nil, fmt.Errorf("Failed to open stream")
	}
	return format, nil
}

func NewFrame() *AVFrame {
	p, _, _ := av_frame_alloc.Call()
	return (*AVFrame)(unsafe.Pointer(p))
}

func (frame *AVFrame) Release() {
	av_frame_free.Call(uintptr(unsafe.Pointer(&frame)))
}

func (frame *AVFrame) ToRGBA(sws uintptr) (*image.RGBA, error) {
	img := image.NewRGBA(image.Rect(0, 0, int(frame.width), int(frame.height)))

	linesizes := [1]int32{frame.width * 4}
	ret, _, _ := sws_scale.Call(uintptr(unsafe.Pointer(sws)), uintptr(unsafe.Pointer(&frame.data[0])), uintptr(unsafe.Pointer(&frame.linesize[0])), 0,
		uintptr(frame.height), uintptr(unsafe.Pointer(&img.Pix[0])), uintptr(unsafe.Pointer(&linesizes[0])))
	if int32(ret) < 0 {
		log.Printf("Failed to copy_image\n")
		return nil, fmt.Errorf("failed to sws_scale")
	}
	return img, nil
}

const (
	AVERROR_EAGAIN      = -11
	AVERROR_EOF         = -541478725
	AV_CH_LAYOUT_STEREO = 3
	AV_SAMPLE_FMT_S16   = 1
	AV_PIX_FMT_RGBA     = 26
	SWS_BILINEAR        = 2
)

func (frame *AVFrame) ToSamples(swr uintptr) (data []byte, nr int, err error) {
	nb_channels, _, _ := av_get_channel_layout_nb_channels.Call(AV_CH_LAYOUT_STEREO)

	var buffer uintptr
	av_samples_alloc.Call(uintptr(unsafe.Pointer(&buffer)), 0, uintptr(nb_channels), uintptr(frame.nb_samples), AV_SAMPLE_FMT_S16, 1)
	n, _, _ := swr_convert.Call(swr, uintptr(unsafe.Pointer(&buffer)), uintptr(frame.nb_samples), uintptr(unsafe.Pointer(&frame.data[0])), uintptr(frame.nb_samples))
	data = GoBytes(unsafe.Pointer(buffer), int(n)*2*2)
	av_freep.Call(uintptr(unsafe.Pointer(&buffer)))

	return data, int(n), nil
}

func GoBytes(buf unsafe.Pointer, len int) []byte {
	var h = struct {
		addr uintptr
		len  int
		cap  int
	}{uintptr(buf), len, len}

	b := *(*[]byte)(unsafe.Pointer(&h))
	a := append([]byte{}, b...)
	return a
}

func GetSWSContext(width, height int32, format int32) uintptr {
	w, h := uintptr(width), uintptr(height)
	sws, _, _ := sws_getContext.Call(w, h, uintptr(format), w, h, AV_PIX_FMT_RGBA, SWS_BILINEAR, 0, 0, 0)
	return sws
}

func FreeSWSContext(sws uintptr) {
	sws_freeContext.Call(sws)
}

func GetSWRContext(sampleRate int32, channelLayout uint64, format int32) uintptr {
	swr, _, _ := swr_alloc_set_opts.Call(0,
		AV_CH_LAYOUT_STEREO,
		AV_SAMPLE_FMT_S16,
		uintptr(sampleRate),
		uintptr(channelLayout),
		uintptr(format),
		uintptr(sampleRate),
		0, 0)
	swr_init.Call(swr)
	return swr
}

func FreeSWRContext(swr uintptr) {
	swr_free.Call(uintptr(unsafe.Pointer(&swr)))
}
