#include <stdio.h>
#include <libavformat/avformat.h>
#include <libavcodec/avcodec.h>
#include <libswscale/swscale.h>

int main(int argc, const char* argv[])
{
	AVStream s;
	AVIOContext ic;
	AVFrame f;
	int off1 = (size_t)(&(s.codecpar)) - (size_t)(&s);
	int off2 = (size_t)(&(ic.error)) - (size_t)(&ic);
	int off3 = (size_t)(&(f.best_effort_timestamp)) - (size_t)(&f);
	printf("var V%d = &Config{AVStream_CodecPar_Offset: %d, \n\tAVIOContext_Error_Offset: %d, \n AVFrame_BestEffortTimestamp_Offset: %d\n}\n", LIBAVFORMAT_VERSION_MAJOR, off1, off2, off3);

	printf(" AVERROR_EAGAIN = %d\n AVERROR_EOF    = %d\n AV_CH_LAYOUT_STEREO = %d\n AV_SAMPLE_FMT_S16   = %d\n"
	"AV_PIX_FMT_RGBA     = %d\n SWS_BILINEAR        = %d\n", AVERROR(EAGAIN), AVERROR_EOF, AV_CH_LAYOUT_STEREO, AV_SAMPLE_FMT_S16, AV_PIX_FMT_RGBA, SWS_BILINEAR);

	printf("AV_PIX_FMT_YUV420P=%d\nAV_PIX_FMT_YUV422P=%d\nAV_PIX_FMT_YUV444P=%d\nAV_PIX_FMT_YUV440P=%d\nAV_PIX_FMT_YUV411P=%d\nAV_PIX_FMT_YUV410P=%d\n", AV_PIX_FMT_YUV420P, AV_PIX_FMT_YUV422P, AV_PIX_FMT_YUV444P, AV_PIX_FMT_YUV440P, AV_PIX_FMT_YUV411P, AV_PIX_FMT_YUV410P);

	return 0;
}

