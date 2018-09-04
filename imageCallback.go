package florid

import (
	"encoding/json"
	"github.com/bitly/go-simplejson"
	"github.com/golang/freetype/truetype"
	"github.com/ngaut/log"
	"image/gif"
	"image/jpeg"
	"image/png"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
	//"runtime"
	//"flag"
	//"runtime/pprof"
	//"os"
)

//var cpuprofile = flag.String("cpuprofile", "", "write cpu profile `file`")
//var memprofile = flag.String("memprofile", "", "write memory profile to `file`")
const (
	SUCCESS              = 0
	IMAGE_URL_EMPYT      = -1
	IMAGE_CONTENT_EMPTY  = -2
	IMAGE_MIME_NOT_FOUND = -3
	IMAGE_GET_ERROR      = -4
	TOKEN_NOT_FIND       = -5
	PARAMS_ERROR         = -6
	IMAGE_DECODE_ERROR   = -7
	NOT_FIND_STYLE       = -8
)

var retMessage = map[int]string{
	SUCCESS:              "success",
	IMAGE_URL_EMPYT:      "image url empty",
	IMAGE_CONTENT_EMPTY:  "image content empty",
	IMAGE_MIME_NOT_FOUND: "image mime not find",
	IMAGE_GET_ERROR:      "image get fatal",
	TOKEN_NOT_FIND:       "token not find",
	PARAMS_ERROR:         "params fatal",
	IMAGE_DECODE_ERROR:   "image decode fatal",
	NOT_FIND_STYLE:       "not find style",
}

type combinInfo struct {
	style      string
	background *imageInfo
	img        map[string]map[string]int
}
type divInfo struct {
	style     string
	width     int
	height    int
	precent   float32
	ctype     string
	backcolor string
	t         string
	space     string
	stretch   int
}
type fontInfo struct {
	word   string
	size   int
	length int
	line   int
	color  string
	ttf    *truetype.Font
	x      int
	y      int
}
type waterInfo struct {
	style   string
	precent float32
	t       string
	logo    *imageInfo
}
type sourceInfo struct {
	token       string
	imagetype   string
	srcImageMap []map[string]imageInfo
	divInfo     []divInfo
	flowInfo    []divInfo
	fontInfo    []fontInfo
	combinInfo  []combinInfo
	waterInfo   []waterInfo
}

func imageCallback(res http.ResponseWriter, req *http.Request) int {
	/*
		flag.Parse()
		if *cpuprofile != "" {
	        f, err := os.Create(*cpuprofile)
	        if err != nil {
	            log.Fatal("could not create CPU profile: ", err)
	        }
	        if err := pprof.StartCPUProfile(f); err != nil {
	            log.Fatal("could not start CPU profile: ", err)
	        }
	        defer pprof.StopCPUProfile()
	        time.Sleep(500 * time.Millisecond)
	    }
	*/
	beginTime := time.Now().UnixNano() / 1e6;
	token := req.Form.Get("token")
	config, ok := template.tokenMap[token]
	if !ok {
		log.Error("token not find:", TOKEN_NOT_FIND)
		return TOKEN_NOT_FIND
	}
	//遍历所需要的token配置
	sourceInfo := &sourceInfo{}
	sourceInfo.token = token
	//获取总配置
	configMap, _ := config.(*simplejson.Json).Map()
	//获取切图type
	if configMap["type"] == nil {
		log.Warn("tpl param fatal [type]!")
		return PARAMS_ERROR
	}
	sourceInfo.imagetype = configMap["type"].(string)
	err := parseDiv(configMap, sourceInfo, req)
	if err != SUCCESS {
		log.Warn("parseDiv fatal:", err)
		return err
	}
	err = parseFlow(configMap, sourceInfo, req)
	if err != SUCCESS {
		log.Warn("parseFlow fatal:", err)
		return err
	}
	ttfs := template.ttf[token]
	err = parseFont(configMap, sourceInfo, ttfs, req)
	if err != SUCCESS {
		log.Warn("parseFont fatal:", err)
		return err
	}
	background := template.background[token]
	err = parseCombin(configMap, sourceInfo, background, req)
	if err != SUCCESS {
		log.Warn("parseCombin fatal:", err)
		return err
	}
	err = parseImage(configMap, sourceInfo, req)
	if err != SUCCESS {
		log.Warn("parseImage fatal:", err)
		return err
	}
	logo := template.logo[token]
	err = parseWater(configMap, sourceInfo, logo, req)
	if err != SUCCESS {
		log.Warn("parseImage fatal:", err)
		return err
	}
	err = doImage(sourceInfo)
	if err != SUCCESS {
		log.Warn("doImage fatal:", err)
		return err
	}
	
	len := len(sourceInfo.srcImageMap)
	for _, info := range sourceInfo.srcImageMap[len-1] {
		switch info.t {
		case "image/jpeg":
			jpeg.Encode(res, info.image, &jpeg.Options{90})
			break
		case "image/png":
			png.Encode(res, info.image)
			break
		case "image/gif":
			gif.Encode(res, info.image, nil)
			break
		}
		break
	}
	endTime := time.Now().UnixNano() / 1e6;
	_ = (endTime - beginTime)
	//log.Info("time:",(endTime - beginTime) )//, "sourceInfo:", sourceInfo)
	/*
	     if *memprofile != "" {
	       f, err := os.Create(*memprofile)
	       if err != nil {
	           log.Fatal("could not create memory profile: ", err)
	       }
	       runtime.GC() // get up-to-date statistics
	       if err := pprof.WriteHeapProfile(f); err != nil {
	           log.Fatal("could not write memory profile: ", err)
	       }
	       f.Close()
	   }
	*/
	return SUCCESS
	//获取配置
}
func getParams(params string) string {
	reg, _ := regexp.Compile(`\$\{(.*?)\}`)
	param := reg.FindStringSubmatch(params)
	if len(param) >= 2 {
		return param[1]
	}
	return ""
}
func parseFont(configMap map[string]interface{}, sourceInfo *sourceInfo, ttfs []*truetype.Font, req *http.Request) int {
	//获取div
	if configMap["font"] == nil {
		return SUCCESS
	}
	fontArr := configMap["font"].([]interface{})
	for i, font := range fontArr {
		var tempFont fontInfo
		fontMap := font.(map[string]interface{})
		if fontMap["word"] != nil {
			wordstr := fontMap["word"].(string)
			reg, _ := regexp.Compile(`\$\{(.*?)\}`)

			params := reg.FindAllStringSubmatch(wordstr, -1)
			for _, param := range params {
				paramstr := req.Form.Get(param[1])
				wordstr = strings.Replace(wordstr, param[0], paramstr, -1)
			}
			tempFont.word = wordstr
		}
		if fontMap["length"] != nil {
			lengthint64, _ := fontMap["length"].(json.Number).Int64()
			tempFont.length = int(lengthint64)
		}
		if fontMap["line"] != nil {
			lineint64, _ := fontMap["line"].(json.Number).Int64()
			tempFont.line = int(lineint64)
		} else {
			tempFont.line = 1
		}
		if fontMap["ttf"] != nil {
			tempFont.ttf = ttfs[i]
		}
		if fontMap["x"] != nil {
			xint64, _ := fontMap["x"].(json.Number).Int64()
			tempFont.x = int(xint64)
		}
		if fontMap["y"] != nil {
			yint64, _ := fontMap["y"].(json.Number).Int64()
			tempFont.y = int(yint64)
		}
		if fontMap["size"] != nil {
			sizeint64, _ := fontMap["size"].(json.Number).Int64()
			tempFont.size = int(sizeint64)
		}
		if fontMap["color"] != nil {
			tempFont.color = fontMap["color"].(string)
		}
		sourceInfo.fontInfo = append(sourceInfo.fontInfo, tempFont)
	}
	return SUCCESS
}
func parseImage(configMap map[string]interface{}, sourceInfo *sourceInfo, req *http.Request) int {
	//获取来源图片变量
	imageParamsArr := configMap["imageParams"].([]interface{})
	var srcImageMap = make(map[string]imageInfo)
	for _, params := range imageParamsArr {
		//获取图片地址
		imageUrl := req.Form.Get(getParams(params.(string)))
		//检查来源图片和获取信息
		imageInfo, ret := checkImage(imageUrl)
		if ret != SUCCESS {
			return ret
		}
		srcImageMap[params.(string)] = imageInfo
	}
	sourceInfo.srcImageMap = append(sourceInfo.srcImageMap, srcImageMap)
	return SUCCESS
}
func parseFlow(configMap map[string]interface{}, sourceInfo *sourceInfo, req *http.Request) int {
	//获取div
	if configMap["flow"] == nil {
		return SUCCESS
	}
	divArr := configMap["flow"].([]interface{})
	for _, div := range divArr {
		var tempDiv divInfo
		divMap := div.(map[string]interface{})
		tempDiv.style = divMap["style"].(string)
		if divMap["width"] != nil {
			width := req.Form.Get(getParams(divMap["width"].(string)))
			tempDiv.width, _ = strconv.Atoi(width)
		}
		if divMap["height"] != nil {
			height := req.Form.Get(getParams(divMap["height"].(string)))
			tempDiv.height, _ = strconv.Atoi(height)
		}
		if divMap["precent"] != nil {
			precent := req.Form.Get(getParams(divMap["precent"].(string)))
			precent64, _ := strconv.ParseFloat(precent, 32)
			tempDiv.precent = float32(precent64)
		}
		if divMap["ctype"] != nil {
			tempDiv.ctype = divMap["ctype"].(string)
		}
		if divMap["backcolor"] != nil {
			tempDiv.backcolor = divMap["backcolor"].(string)
		}
		if divMap["space"] != nil {
			tempDiv.space = divMap["space"].(string)
		}
		if divMap["type"] != nil {
			tempDiv.t = divMap["type"].(string)
		}
		sourceInfo.flowInfo = append(sourceInfo.flowInfo, tempDiv)
	}
	return SUCCESS
}
func parseDiv(configMap map[string]interface{}, sourceInfo *sourceInfo, req *http.Request) int {
	//获取div
	if configMap["div"] == nil {
		return SUCCESS
	}
	divArr := configMap["div"].([]interface{})
	for _, div := range divArr {
		var tempDiv divInfo
		divMap := div.(map[string]interface{})
		tempDiv.style = divMap["style"].(string)
		if divMap["width"] != nil {

			wkey := getParams(divMap["width"].(string))
			if wkey == "" {
				tempDiv.width, _ = strconv.Atoi(divMap["width"].(string))
			} else {
				width := req.Form.Get(wkey)
				tempDiv.width, _ = strconv.Atoi(width)
			}
		}
		if divMap["height"] != nil {
		    hkey := getParams(divMap["height"].(string))
		    
		    if hkey == "" {
				tempDiv.height, _ = strconv.Atoi(divMap["height"].(string))
			} else {
				height := req.Form.Get(hkey)
				tempDiv.height, _ = strconv.Atoi(height)
			}
		}
		if divMap["precent"] != nil {
			precent := req.Form.Get(getParams(divMap["precent"].(string)))
			precent64, _ := strconv.ParseFloat(precent, 32)
			tempDiv.precent = float32(precent64)
		}
		if divMap["ctype"] != nil {
			tempDiv.ctype = divMap["ctype"].(string)
		}
		if divMap["backcolor"] != nil {
			tempDiv.backcolor = divMap["backcolor"].(string)
		}
		if divMap["space"] != nil {
			tempDiv.space = divMap["space"].(string)
		}
		if divMap["type"] != nil {
			tempDiv.t = divMap["type"].(string)
		}
		if divMap["stretch"] != nil {
			stretch64, _ := divMap["stretch"].(json.Number).Int64()
			tempDiv.stretch = int(stretch64)
		}
		sourceInfo.divInfo = append(sourceInfo.divInfo, tempDiv)
	}
	return SUCCESS
}
func parseCombin(configMap map[string]interface{}, sourceInfo *sourceInfo, background *imageInfo, req *http.Request) int {
	//获取div
	if configMap["combin"] == nil {
		return SUCCESS
	}
	imageParamsArr := configMap["imageParams"].([]interface{})
	combinArr := configMap["combin"].([]interface{})
	for _, combin := range combinArr {
		var tempCombin combinInfo
		combinMap := combin.(map[string]interface{})
		tempCombin.style = combinMap["style"].(string)
		if combinMap["background"] != nil {
			tempCombin.background = background
		}
		tempCombin.img = make(map[string]map[string]int)
		
		for _, params := range imageParamsArr {
			//获取图片位置
			param := params.(string)
			if combinMap[param] != nil {
				imgMap := combinMap[param].(map[string]interface{})
				xint64, _ := imgMap["x"].(json.Number).Int64()
				yint64, _ := imgMap["y"].(json.Number).Int64()
				xymap := make(map[string]int)
				xymap["x"] = int(xint64)
				xymap["y"] = int(yint64)
				tempCombin.img[param] = xymap
			}
		}
		sourceInfo.combinInfo = append(sourceInfo.combinInfo, tempCombin)
	}
	return SUCCESS
}
func parseWater(configMap map[string]interface{}, sourceInfo *sourceInfo, logo *imageInfo, req *http.Request) int {
	//获取div
	if configMap["water"] == nil {
		return SUCCESS
	}
	waterArr := configMap["water"].([]interface{})
	for _, water := range waterArr {
		var tempWater waterInfo
		waterMap := water.(map[string]interface{})
		tempWater.style = waterMap["style"].(string)
		if waterMap["logo"] != nil {
			tempWater.logo = logo
		}
		if waterMap["type"] != nil {
			tempWater.t = waterMap["type"].(string)
		}
		if waterMap["precent"] != nil {
			precent64, _ := waterMap["precent"].(json.Number).Float64()
			tempWater.precent = float32(precent64)
		}
		sourceInfo.waterInfo = append(sourceInfo.waterInfo, tempWater)
	}
	log.Info(sourceInfo)
	return SUCCESS
}
