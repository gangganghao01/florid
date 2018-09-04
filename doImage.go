package florid

import (
	"bytes"
	"fmt"
	"github.com/golang/freetype/truetype"
	"github.com/ngaut/log"
	"golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"net/http"
	"os"
	"strings"
	"time"
)

var magicTable = map[string]string{
	"\xff\xd8\xff":      "image/jpeg",
	"\x89PNG\r\n\x1a\n": "image/png",
	"GIF87a":            "image/gif",
	"GIF89a":            "image/gif",
}

type imageInfo struct {
	step  string
	url   string
	w     int
	h     int
	t     string
	image image.Image
	draw  draw.Image
}

func scale(div divInfo, sourceInfo *sourceInfo) int {

	var newSrcImageMap = make(map[string]imageInfo)
	size := len(sourceInfo.srcImageMap)
	for key, Info := range sourceInfo.srcImageMap[size-1] {

		if div.precent <= 0 {
			div.precent = 1
		}
		var srcWidth, srcHeight int

		if div.stretch == 1 {
			srcWidth = div.width
			srcHeight = div.height
		} else {
			srcWidth = Info.w
			srcHeight = Info.h
		}
		desWidth := int(float32(srcWidth) * div.precent)
		desHeight := int(float32(srcHeight) * div.precent)

		//绘制底图
		backGroundRect := image.Rect(0, 0, desWidth, desHeight)
		backGroundPic := image.NewRGBA(backGroundRect)
		white := color.RGBA{255, 255, 255, 255}
		draw.Draw(backGroundPic, backGroundRect, &image.Uniform{white}, image.ZP, draw.Src)
		//缩略
		draw.NearestNeighbor.Scale(backGroundPic, backGroundRect, Info.image, Info.image.Bounds(), draw.Src, nil)
		newImage := imageInfo{step: "scale", url: Info.url, w: desWidth, h: desHeight, t: Info.t, image: backGroundPic, draw: backGroundPic}
		newSrcImageMap[key] = newImage
	}
	sourceInfo.srcImageMap = append(sourceInfo.srcImageMap, newSrcImageMap)
	return SUCCESS
}
func clip(div divInfo, sourceInfo *sourceInfo) int {

	var newSrcImageMap = make(map[string]imageInfo)
	size := len(sourceInfo.srcImageMap)
	for key, Info := range sourceInfo.srcImageMap[size-1] {
		srcWidth := Info.w
		srcHeight := Info.h
		desWidth := div.width
		desHeight := div.height
		if div.width <= 0 || div.height <= 0 {
			log.Warn("div.width <=0 ||  div.height <= 0 ", div.width, div.height)
			return PARAMS_ERROR
		}

		var x1, y1, x2, y2, w, h int
		if desWidth-srcWidth < 0 {
			x2 = int((srcWidth - desWidth) / 2)
			w = desWidth
			if desHeight-srcHeight < 0 {
				y2 = int((srcHeight - desHeight) / 2)
				x1 = 0
				y1 = 0
				h = desHeight
			} else {
				y2 = 0
				x1 = 0
				y1 = int((desHeight - srcHeight) / 2)
				h = srcHeight
			}
		} else {
			x1 = int((desWidth - srcWidth) / 2)
			w = srcWidth
			if desHeight-srcHeight < 0 {
				y1 = 0
				x2 = 0
				y2 = int((srcHeight - desHeight) / 2)
				h = desHeight

			} else {
				y1 = int((desHeight - srcHeight) / 2)
				x2 = 0
				y2 = 0
				h = srcHeight
			}

		}
		x2y2 := image.Point{X: x2, Y: y2}
		wh := image.Point{X: w, Y: h}
		rect := image.Rectangle{x2y2, x2y2.Add(wh)}

		//绘制底图
		var startXY image.Point
		var backGroundRect image.Rectangle
		var uniform image.Image
		if div.space == "yes" {
			backGroundRect = image.Rect(0, 0, desWidth, desHeight)
			startXY = image.Point{X: x1, Y: y1}

			if div.ctype == "a" {
				color := hextorgb(div.backcolor)
				uniform = &image.Uniform{color}
			} else if div.ctype == "b" {
				autocolor := Info.image.At(0, 0)
				uniform = &image.Uniform{autocolor}
			} else {

				r, g, b, _ := Info.image.At(0, 0).RGBA()
				alpha := image.NewRGBA(image.Rect(0, 0, desWidth, desHeight))
				for x := 0; x < desWidth; x++ {
					for y := 0; y < desHeight; y++ {
						if y < desHeight/2 {
							alpha.Set(x, y, color.NRGBA{uint8(r), uint8(g), uint8(b), uint8(y % 256)}) //设定alpha图片的透明度
						} else {
							alpha.Set(x, y, color.NRGBA{uint8(r), uint8(g), uint8(b), uint8((desHeight - y) % 256)})
						}
					}
				}
				uniform = alpha
			}

		} else {
			backGroundRect = image.Rect(0, 0, w, h)
			startXY = image.ZP
			white := color.RGBA{255, 255, 255, 255}
			uniform = &image.Uniform{white}
		}
		backGroundPic := image.NewRGBA(backGroundRect)

		draw.Draw(backGroundPic, backGroundRect, uniform, image.ZP, draw.Over)
		draw.Copy(backGroundPic, startXY, Info.image, rect, draw.Over, nil)
		//draw.Draw(backGroundPic, backGroundRect.Add(point),  Info.image, point, draw.Over)
		newImage := imageInfo{step: "clip", url: Info.url, w: desWidth, h: desHeight, t: Info.t, image: backGroundPic, draw: backGroundPic}
		newSrcImageMap[key] = newImage
	}
	sourceInfo.srcImageMap = append(sourceInfo.srcImageMap, newSrcImageMap)
	return SUCCESS
}
func autoscale(div divInfo, sourceInfo *sourceInfo) int {

	size := len(sourceInfo.srcImageMap)
	for _, Info := range sourceInfo.srcImageMap[size-1] {
		if div.t == "w" {
			div.precent = float32(div.width) / float32(Info.w)
		} else if div.t == "h" {
			div.precent = float32(div.height) / float32(Info.h)
		} else if div.t == "b" {
			if float32(div.width)/float32(Info.w) < float32(div.height)/float32(Info.h) {
				div.precent = float32(div.width) / float32(Info.w)
			} else {
				div.precent = float32(div.height) / float32(Info.h)
			}
		}
	}

	return scale(div, sourceInfo)
}
func doDiv(div divInfo, sourceInfo *sourceInfo) int {

	ret := SUCCESS
	switch div.style {
	case "scale":
		ret = scale(div, sourceInfo)
		break
	case "autoscale":
		ret = autoscale(div, sourceInfo)
		break
	case "clip":
		ret = clip(div, sourceInfo)
		break
	default:
		return NOT_FIND_STYLE
	}
	return ret
}
func hextorgb(s string) color.RGBA {
	s = strings.ToLower(s)
	var r, g, b, a uint8
	fmt.Sscanf(s, "#%02x%02x%02x%02x", &r, &g, &b, &a)
	r *= 17
	g *= 17
	b *= 17
	a *= 17

	return color.RGBA{r, g, b, a}
}
func doFont(f fontInfo, sourceInfo *sourceInfo) int {

	size := len(sourceInfo.srcImageMap)
	for _, Info := range sourceInfo.srcImageMap[size-1] {
		drawer := &font.Drawer{
			Dst: Info.draw,
			Src: image.NewUniform(hextorgb(f.color)),
			//Src: image.Black,
			Face: truetype.NewFace(f.ttf, &truetype.Options{
				Size: float64(f.size),
				DPI:  72,
			}),
		}

		var stringWord string
		word := []rune(f.word)
		wordlen := len(word)
		var begin, end int
		/*
			if wordlen > f.length && f.length > 0 {
				stringWord = string(word[:f.length])
				if wordlen > f.length {
					stringWord = stringWord + "..."
				}
			} else {
				stringWord = f.word
			}
		*/

		rows := wordlen/f.length + 1
		lastnum := wordlen % f.length

		for i := 0; i < rows; i++ {
			begin = i * f.length
			if i == rows-1 {
				end = begin + lastnum
			} else {
				end = (i + 1) * f.length
			}
			drawer.Dot = fixed.Point26_6{
				X: fixed.I(f.x),
				Y: fixed.I(f.y + (i+1)*f.size + (i * 10)),
			}
			if f.line == i+1 && f.line < rows && lastnum != 0 {
				stringWord = string(word[begin:end]) + "..."
				drawer.DrawString(stringWord)
				break
			} else {
				stringWord = string(word[begin:end])
				drawer.DrawString(stringWord)
			}

		}

	}
	return SUCCESS
}
func doCombin(combin combinInfo, sourceInfo *sourceInfo) int {

	size := len(sourceInfo.srcImageMap)
	background := combin.background

	newImage := make(map[string]imageInfo)
	rect := image.Rect(0, 0, background.w, background.h)
	newpic := image.NewRGBA(rect)
	white := color.RGBA{255, 255, 255, 255}
	//绘制底板
	draw.Draw(newpic, rect, &image.Uniform{white}, image.ZP, draw.Src)
	draw.Draw(newpic, rect, background.image, image.ZP, draw.Over)

	for key, imgxy := range combin.img {
		xy := image.Point{X: imgxy["x"], Y: imgxy["y"]}
		draw.Draw(newpic, background.image.Bounds().Add(xy), sourceInfo.srcImageMap[size-1][key].image, image.ZP, draw.Over)
	}
	newImage["combin"] = imageInfo{step: "combin", url: background.url, w: background.w, h: background.h, t: background.t, image: newpic, draw: newpic}
	sourceInfo.srcImageMap = append(sourceInfo.srcImageMap, newImage)
	return SUCCESS

}
func doWater(water waterInfo, sourceInfo *sourceInfo) int {

	size := len(sourceInfo.srcImageMap)
	var newSrcImageMap = make(map[string]imageInfo)
	for key, Info := range sourceInfo.srcImageMap[size-1] {
		srcWidth := Info.w
		srcHeight := Info.h
		var waterWidth, waterHeight int
		if water.t == "w" {
			waterWidth = int(float32(srcWidth) * water.precent)
			waterHeight = waterWidth
		} else {
			waterHeight = int(float32(srcHeight) * water.precent)
			waterWidth = waterHeight
		}
		//水印缩略
		waterGroundRect := image.Rect(0, 0, waterWidth, waterHeight)
		waterGroundPic := image.NewRGBA(waterGroundRect)
		white := color.RGBA{255, 255, 255, 255}
		draw.Draw(waterGroundPic, waterGroundRect, &image.Uniform{white}, image.ZP, draw.Src)
		draw.NearestNeighbor.Scale(waterGroundPic, waterGroundRect, water.logo.image, water.logo.image.Bounds(), draw.Src, nil)

		//绘制底图
		backGroundRect := image.Rect(0, 0, srcWidth, srcHeight)
		backGroundPic := image.NewRGBA(backGroundRect)
		white1 := color.RGBA{255, 255, 255, 255}
		draw.Draw(backGroundPic, backGroundRect, &image.Uniform{white1}, image.ZP, draw.Src)
		draw.Draw(backGroundPic, backGroundRect, Info.image, image.ZP, draw.Over)

		//计算绘制位置
		x1 := int(srcWidth/2) - int(waterWidth/2)
		y1 := int(srcHeight/2) - int(waterHeight/2)
		startXY := image.Point{X: x1, Y: y1}
		draw.Copy(backGroundPic, startXY, waterGroundPic, waterGroundRect, draw.Over, nil)

		newImage := imageInfo{step: "water", url: Info.url, w: srcWidth, h: srcHeight, t: Info.t, image: backGroundPic, draw: backGroundPic}
		newSrcImageMap[key] = newImage
	}
	sourceInfo.srcImageMap = append(sourceInfo.srcImageMap, newSrcImageMap)
	return SUCCESS
}
func doImage(sourceInfo *sourceInfo) int {

	var ret int
	for _, div := range sourceInfo.divInfo {
		ret = doDiv(div, sourceInfo)
		if ret != SUCCESS {
			return ret
		}

	}
	for _, combin := range sourceInfo.combinInfo {
		ret = doCombin(combin, sourceInfo)
		if ret != SUCCESS {
			return ret
		}

	}

	for _, water := range sourceInfo.waterInfo {
		ret = doWater(water, sourceInfo)
		if ret != SUCCESS {
			return ret
		}
	}
	for _, f := range sourceInfo.fontInfo {
		ret = doFont(f, sourceInfo)
		if ret != SUCCESS {
			return ret
		}

	}
	for _, flow := range sourceInfo.flowInfo {
		ret = doDiv(flow, sourceInfo)
		if ret != SUCCESS {
			return ret
		}

	}
	return SUCCESS
}

func getImageInfo(path string, si *imageInfo, t string) int {

	var buf bytes.Buffer
	if t == "net" {

		client := &http.Client{
			Timeout: time.Duration(config.ctime) * time.Millisecond,
		}
		req, err := http.NewRequest("GET", path, nil)
		req.Header.Add("Referer", "http://pic.baidu.com")
		i := 0
		var res *http.Response
		for i < 2 {
			res, err = client.Do(req)

			if err != nil || res == nil {
				i++
				log.Warn("cliet get fatal:", err, "retry=", i)
				if i < 2 {
					time.Sleep(4 * time.Millisecond)
					continue
				}
				return IMAGE_GET_ERROR
			}

			break
		}
		defer res.Body.Close()
		if res != nil && res.StatusCode != http.StatusOK {
			log.Warn("cliet status:", res.StatusCode)
			return IMAGE_GET_ERROR
		}

		//读取图片内容
		n, err := buf.ReadFrom(res.Body)
		if n <= 0 || err != nil || res.ContentLength <= 0 {
			log.Warn("client read buf n < 0", err, res)
			return IMAGE_CONTENT_EMPTY
		}

	} else {
		f, err := os.Open(path)
		if err != nil {
			log.Warn("fail open path ", path)
			return IMAGE_GET_ERROR
		}
		defer f.Close()
		n, err := buf.ReadFrom(f)
		if n <= 0 || err != nil {
			log.Warn("buf read from fatal. ", err)
			return IMAGE_CONTENT_EMPTY
		}
	}
	//通过头文件判断图片类型
	//判断图片类型
	imageMime := mimeFromIncipit(buf.Bytes())
	if imageMime == "" {
		log.Warn("imageMime not find.")
		return IMAGE_MIME_NOT_FOUND
	}
	si.t = imageMime
	//获取图片尺寸
	var ret error
	switch imageMime {
	case "image/jpeg":
		si.image, ret = jpeg.Decode(&buf)
		if ret != nil {
			log.Warn("jpeg.Decode fatal.", ret)
			return IMAGE_DECODE_ERROR
		}
		si.w = si.image.Bounds().Dx()
		si.h = si.image.Bounds().Dy()
		break
	case "image/png":
		si.image, ret = png.Decode(&buf)
		if ret != nil {
			log.Warn("png.Decode fatal.", ret)
			return IMAGE_DECODE_ERROR
		}
		si.w = si.image.Bounds().Dx()
		si.h = si.image.Bounds().Dy()
		break
	case "image/gif":
		si.image, ret = gif.Decode(&buf)
		if ret != nil {
			log.Warn("gif.Decode fatal.", ret)
			return IMAGE_DECODE_ERROR
		}
		si.w = si.image.Bounds().Dx()
		si.h = si.image.Bounds().Dy()
		break
	}

	return SUCCESS
}
func checkImage(url string) (imageInfo, int) {
	//判断图片地址是否为空
	var si *imageInfo = &imageInfo{}
	if url == "" {
		return *si, IMAGE_URL_EMPYT
	}
	//获取图片信息
	si.url = url
	ret := getImageInfo(url, si, "net")
	return *si, ret
}
func mimeFromIncipit(incipit []byte) string {
	incipitStr := string(incipit[:])
	for magic, mime := range magicTable {
		if strings.HasPrefix(incipitStr, magic) {
			return mime
		}
	}
	return ""
}
