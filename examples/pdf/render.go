// +build !windows

package main

// #cgo CFLAGS: -I.
// #cgo !darwin LDFLAGS: -L. -lpdfium -lm -lpthread -lstdc++
// #cgo darwin LDFLAGS: -L. -lpdfium -lm -framework CoreFoundation -framework CoreGraphics -lc++
// #include "fpdfview.h"
// extern FPDF_BOOL FPDFPage_HasTransparency(FPDF_PAGE page);
import "C"

import (
	"errors"
	"image"
	"unsafe"
)

func InitPDFLibrary() {
	C.FPDF_InitLibrary()
}

func DestroyPDFLibrary() {
	C.FPDF_DestroyLibrary()
}

func lastError() error {
	err := C.FPDF_GetLastError()
	var errMsg string
	switch err {
	case C.FPDF_ERR_SUCCESS:
		errMsg = "Success"
	case C.FPDF_ERR_UNKNOWN:
		errMsg = "Unknown error"
	case C.FPDF_ERR_FILE:
		errMsg = "Unable to read file"
	case C.FPDF_ERR_FORMAT:
		errMsg = "Incorrect format"
	case C.FPDF_ERR_PASSWORD:
		errMsg = "Invalid password"
	case C.FPDF_ERR_SECURITY:
		errMsg = "Invalid encryption"
	case C.FPDF_ERR_PAGE:
		errMsg = "Incorrect page"
	default:
		errMsg = "Unexpected error"
	}
	return errors.New(errMsg)

}

func (pdf *PDFDocument) load() error {
	var doc C.FPDF_DOCUMENT
	if pdf.file != "" {
		doc = C.FPDF_LoadDocument(C.CString(pdf.file), nil)
	} else if len(pdf.data) > 0 {
		doc = C.FPDF_LoadMemDocument(unsafe.Pointer(&pdf.data[0]), C.int(len(pdf.data)), nil)
	}

	if doc == nil {
		return lastError()
	}

	pdf.pdoc = uintptr(unsafe.Pointer(doc))
	count := int(C.FPDF_GetPageCount(doc))
	pdf.pages = make([]*pdfPage, count)
	return nil
}

func (pdf *PDFDocument) Close() {
	if pdf.pdoc != 0 {
		doc := C.FPDF_DOCUMENT(unsafe.Pointer(pdf.pdoc))
		C.FPDF_CloseDocument(doc)
		pdf.pdoc = 0
	}
}

func (pdf *PDFDocument) renderPage(idx int, dpi float32) (image.Image, error) {
	pdf.mutex.Lock()
	defer pdf.mutex.Unlock()

	if dpi == 0 {
		dpi = 160.
	}

	doc := C.FPDF_DOCUMENT(unsafe.Pointer(pdf.pdoc))
	page := C.FPDF_LoadPage(doc, C.int(idx))
	if page == nil {
		return nil, lastError()
	}
	defer C.FPDF_ClosePage(page)

	width := C.int(C.FPDF_GetPageWidthF(page) * C.float(dpi) / 72.)
	height := C.int(C.FPDF_GetPageHeightF(page) * C.float(dpi) / 72.)
	alpha := C.FPDFPage_HasTransparency(page)
	fillColor := uint32(0xffffffff) //argb
	if int(alpha) == 1 {
		fillColor = 0
	}

	bmp := C.FPDFBitmap_Create(width, height, alpha)
	if bmp == nil {
		return nil, lastError()
	}
	defer C.FPDFBitmap_Destroy(bmp)

	C.FPDFBitmap_FillRect(bmp, 0, 0, width, height, C.ulong(fillColor))
	C.FPDF_RenderPageBitmap(bmp, page, 0, 0, width, height, 0, C.FPDF_ANNOT)
	buf := C.FPDFBitmap_GetBuffer(bmp)

	img := image.NewRGBA(image.Rect(0, 0, int(width), int(height)))
	img.Stride = int(C.FPDFBitmap_GetStride(bmp))
	src := uintptr(buf)
	for y := 0; y < int(height); y++ {
		dst := img.Pix[y*img.Stride : (y+1)*img.Stride]
		for x := 0; x < int(width); x++ {
			//BGRA to RGBA
			dpix := dst[x*4 : (x+1)*4]
			spix := *(*[4]uint8)(unsafe.Pointer(src))
			dpix[0] = spix[2]
			dpix[1] = spix[1]
			dpix[2] = spix[0]
			dpix[3] = spix[3]
			src += 4
		}
	}

	return img, nil
}
