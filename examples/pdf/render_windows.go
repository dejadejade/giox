// +build windows

package main

import (
	"errors"
	"image"
	"log"
	"syscall"
	"unsafe"
)

var (
	pdfium = syscall.NewLazyDLL("pdfium.dll")

	pFPDF_InitLibrary         = pdfium.NewProc("FPDF_InitLibrary")
	pFPDF_DestroyLibrary      = pdfium.NewProc("FPDF_DestroyLibrary")
	pFPDF_GetLastError        = pdfium.NewProc("FPDF_GetLastError")
	pFPDF_LoadDocument        = pdfium.NewProc("FPDF_LoadDocument")
	pFPDF_LoadMemDocument     = pdfium.NewProc("FPDF_LoadMemDocument")
	pFPDF_GetPageCount        = pdfium.NewProc("FPDF_GetPageCount")
	pFPDF_CloseDocument       = pdfium.NewProc("FPDF_CloseDocument")
	pFPDF_LoadPage            = pdfium.NewProc("FPDF_LoadPage")
	pFPDF_ClosePage           = pdfium.NewProc("FPDF_ClosePage")
	pFPDF_GetPageWidthF       = pdfium.NewProc("FPDF_GetPageWidthF")
	pFPDF_GetPageHeightF      = pdfium.NewProc("FPDF_GetPageHeightF")
	pFPDF_RenderPageBitmap    = pdfium.NewProc("FPDF_RenderPageBitmap")
	pFPDFBitmap_Create        = pdfium.NewProc("FPDFBitmap_Create")
	pFPDFBitmap_Destroy       = pdfium.NewProc("FPDFBitmap_Destroy")
	pFPDFBitmap_FillRect      = pdfium.NewProc("FPDFBitmap_FillRect")
	pFPDFBitmap_GetBuffer     = pdfium.NewProc("FPDFBitmap_GetBuffer")
	pFPDFBitmap_GetStride     = pdfium.NewProc("FPDFBitmap_GetStride")
	pFPDFPage_HasTransparency = pdfium.NewProc("FPDFPage_HasTransparency")
	pFPDF_GetPageSizeByIndexF = pdfium.NewProc("FPDF_GetPageSizeByIndexF")
)

func InitPDFLibrary() {
	pFPDF_InitLibrary.Call()
}

func DestroyPDFLibrary() {
	pFPDF_DestroyLibrary.Call()
}

func lastError() error {
	err, _, _ := pFPDF_GetLastError.Call()
	_ = err
	log.Printf("error: %v\n", err)
	var errMsg string
	/*	switch err {
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
	*/
	return errors.New(errMsg)

}

func (pdf *PDFDocument) load() error {
	var doc uintptr
	if pdf.file != "" {
		file := append([]byte(pdf.file), 0)
		doc, _, _ = pFPDF_LoadDocument.Call(uintptr(unsafe.Pointer(&file[0])), 0)
	} else if len(pdf.data) > 0 {
		doc, _, _ = pFPDF_LoadMemDocument.Call(uintptr(unsafe.Pointer(&pdf.data[0])), uintptr(len(pdf.data)), 0)
	}

	if doc == 0 {
		return lastError()
	}

	pdf.pdoc = doc
	count, _, _ := pFPDF_GetPageCount.Call(doc)
	pdf.pages = make([]*pdfPage, int(count))
	return nil
}

func (pdf *PDFDocument) Close() {
	if pdf.pdoc != 0 {
		pFPDF_CloseDocument.Call(pdf.pdoc)
		pdf.pdoc = 0
	}
}

func (pdf *PDFDocument) renderPage(idx int, dpi float32) (image.Image, error) {
	pdf.mutex.Lock()
	defer pdf.mutex.Unlock()

	if dpi == 0 {
		dpi = 160.
	}

	doc := pdf.pdoc
	page, _, _ := pFPDF_LoadPage.Call(doc, uintptr(idx))
	if page == 0 {
		return nil, lastError()
	}
	defer pFPDF_ClosePage.Call(page)

	var fsize = []float32{0, 0}
	pFPDF_GetPageSizeByIndexF.Call(doc, uintptr(idx), uintptr(unsafe.Pointer(&fsize[0])))

	width := (float32(fsize[0]) * dpi / 72.)
	height := (float32(fsize[1]) * dpi / 72.)
	alpha, _, _ := pFPDFPage_HasTransparency.Call(page)
	fillColor := uint32(0xffffffff) //argb
	if int(alpha) == 1 {
		fillColor = 0
	}

	bmp, _, _ := pFPDFBitmap_Create.Call(uintptr(width), uintptr(height), uintptr(alpha))
	if bmp == 0 {
		return nil, lastError()
	}
	defer pFPDFBitmap_Destroy.Call(bmp)

	pFPDFBitmap_FillRect.Call(bmp, 0, 0, uintptr(width), uintptr(height), uintptr(fillColor))
	pFPDF_RenderPageBitmap.Call(bmp, page, 0, 0, uintptr(width), uintptr(height), 0, 0)
	buf, _, _ := pFPDFBitmap_GetBuffer.Call(bmp)
	strides, _, _ := pFPDFBitmap_GetStride.Call(bmp)

	img := image.NewRGBA(image.Rect(0, 0, int(width), int(height)))
	img.Stride = int(strides)
	src := buf
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
