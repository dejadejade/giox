package player

import "unsafe"

const AV_NOPTS_VALUE = uint64(0x8000000000000000)

type AVRational struct {
	num int32
	den int32
}

func (r AVRational) D() float64 {
	return float64(r.num) / float64(r.den)
}

type AVFormatContext struct {
	av_class   uintptr
	iformat    uintptr
	oformat    uintptr
	priv_data  uintptr
	pb         uintptr
	ctx_flags  int32
	nb_streams uint32
	streams    uintptr
	//omitted
}

func (format *AVFormatContext) Stream(idx int) *AVStream {
	if idx < 0 || idx >= int(format.nb_streams) {
		return nil
	}
	p := format.streams + unsafe.Sizeof(uintptr(0))*uintptr(idx)
	return *(**AVStream)(unsafe.Pointer(p))
}

func (format *AVFormatContext) Error() int {
	err := *(*int32)(unsafe.Pointer(format.pb + config.AVIOContext_Error_Offset))
	return int(err)
}

const AVMEDIA_TYPE_VIDEO = 0
const AVMEDIA_TYPE_AUDIO = 1

type AVCodecContext struct {
	av_class                  uintptr
	log_level_offset          int32
	codec_type                int32
	codec                     uintptr
	codec_id                  int32
	codec_tag                 uint32
	priv_data                 uintptr
	internal                  uintptr
	opaque                    uintptr
	bit_rate                  int64
	bit_rate_tolerance        int32
	global_quality            int32
	compression_level         int32
	flags                     int32
	flags2                    int32
	extradata                 uintptr
	extradata_size            int32
	time_base                 AVRational
	ticks_per_frame           int32
	delay                     int32
	width, height             int32
	coded_width, coded_height int32
	gop_size                  int32
	pix_fmt                   int32
	//omitted
}

const AVCodecContextAudio_Offset = 0

type AVStream struct {
	index               int32
	id                  int32
	codec               uintptr // to be deprecated
	priv_data           uintptr
	time_base           AVRational
	start_time          int64
	duration            int64
	nb_frames           int64
	disposition         int32
	discard             int32
	sample_aspect_ratio AVRational
	metadata            uintptr
	avg_frame_rate      AVRational
	//omitted
}

func (stream *AVStream) TimeBase() float64 {
	return stream.time_base.D()
}

type AVColorRange = int32

type AVCodecParameters struct {
	codec_type            int32
	codec_id              int32
	codec_tag             uint32
	extradata             uintptr
	extradata_size        int32
	format                int32
	bit_rate              int64
	bits_per_coded_sample int32
	bits_per_raw_sample   int32
	profile               int32
	level                 int32
	width, height         int32
	sample_aspect_ratio   AVRational
	field_order           int32

	color_range     int32
	color_primaries int32
	color_trc       int32
	color_space     int32
	chroma_location int32

	video_delay      int32
	channel_layout   uint64
	channels         int32
	sample_rate      int32
	block_align      int32
	frame_size       int32
	initial_padding  int32
	trailing_padding int32
	seek_preroll     int32
}

type Config struct {
	AVStream_CodecPar_Offset           uintptr
	AVIOContext_Error_Offset           uintptr
	AVFrame_BestEffortTimestamp_Offset uintptr
}

func (stream *AVStream) CodecPar() *AVCodecParameters {
	p := uintptr(unsafe.Pointer(stream)) + config.AVStream_CodecPar_Offset
	return *((**AVCodecParameters)(unsafe.Pointer(p)))
}

type AVPacket struct {
	buf             uintptr
	pts, dts        uint64
	data            uintptr
	size            int32
	stream_index    int32
	flags           int32
	side_data       uintptr
	side_data_elems int32
	duration        int64
	pos             int64
	//omitted
}

func (pkt *AVPacket) PTS(timebase float64) float64 {
	if pkt.pts == AV_NOPTS_VALUE {
		return 0
	}

	return timebase * float64(pkt.pts)
}

func (pkt *AVPacket) StreamIndex() int {
	return int(pkt.stream_index)
}

const AV_NUM_DATA_POINTERS = 8

type AVFrame struct {
	data                   [AV_NUM_DATA_POINTERS]uintptr
	linesize               [AV_NUM_DATA_POINTERS]int32
	extended_data          uintptr
	width, height          int32
	nb_samples             int32
	format                 int32
	key_frame              int32
	pict_type              int32
	sample_aspect_ratio    AVRational
	pts                    int64
	pkt_dts                int64
	coded_picture_number   int32
	display_picture_number int32
	quality                int32
	opaque                 uintptr
	repeat_pict            int32
	interlaced_frame       int32
	//omitted
}

func (frame *AVFrame) Delay(timebase float64) float64 {
	delay := timebase
	delay += float64(frame.repeat_pict) * (delay * 0.5)
	return delay
}

func (frame *AVFrame) BestEffortTimestamp() int64 {
	pts := *(*int64)(unsafe.Pointer(uintptr(unsafe.Pointer(frame)) + config.AVFrame_BestEffortTimestamp_Offset))
	if uint64(pts) == AV_NOPTS_VALUE {
		return 0
	}
	return pts
}

var V58 = &Config{AVStream_CodecPar_Offset: 208,
	AVIOContext_Error_Offset:           120,
	AVFrame_BestEffortTimestamp_Offset: 408,
}

var config *Config = V58

const (
	AV_PIX_FMT_YUV420P = 0
	AV_PIX_FMT_YUV422P = 4
	AV_PIX_FMT_YUV444P = 5
	AV_PIX_FMT_YUV440P = 31
	AV_PIX_FMT_YUV411P = 7
	AV_PIX_FMT_YUV410P = 6
)
