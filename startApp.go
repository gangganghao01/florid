package florid

import (
	"errors"
	"github.com/bitly/go-simplejson"
	"github.com/golang/freetype"
	"github.com/golang/freetype/truetype"
	"github.com/ngaut/log"
	"io/ioutil"
)


type Template struct {
	tokenMap   map[string]interface{}
	ttf        map[string][]*truetype.Font
	background map[string]*imageInfo
	logo      map[string]*imageInfo
}
type Config struct {
	port   int
	rtime  int
	wtime  int
	ctime  int
	template string
	md5 bool
	pwd string
}
var template *Template
var config *Config

func init() {
	log.SetOutputByName("./florid.log")
	log.SetHighlighting(false)
	template = &Template{
	    tokenMap: make(map[string]interface{}), 
	    ttf: make(map[string][]*truetype.Font), 
	    background: make(map[string]*imageInfo), 
	    logo:make(map[string]*imageInfo),
	}
	config = &Config{}
}

func initTemplate() error {
	data, err := ioutil.ReadFile(config.template)
	if err != nil {
		log.Warn(err)
		return err
	}
	conf, err := simplejson.NewJson(data)
	if err != nil {
		log.Warn(err)
		return err
	}
	root, err := conf.Array()
	if err != nil {
		log.Warn(err)
		return err
	}
	for i := 0; i < len(root); i++ {
		child := conf.GetIndex(i)
		token, _ := child.Get("token").String()
		template.tokenMap[token] = child

		//check font
		fontArr, _ := child.Get("font").Array()
		for _, font := range fontArr {
			fontMap := font.(map[string]interface{})

			if fontMap["ttf"] == nil {
				return errors.New("error: ttf =nil not find.")
			}
			ttfPath := fontMap["ttf"].(string)
			if ttfPath == "" {
				return errors.New("error: ttf not find.")
			}
			fontfile, err := ioutil.ReadFile(ttfPath)
			if err != nil {
				log.Warn(err)
				return err
			}
			fontbyte, err := freetype.ParseFont(fontfile)
			if err != nil {
				log.Warn(err)
				return err
			}
			template.ttf[token] = append(template.ttf[token], fontbyte)
		}

		//check font
		combinArr, _ := child.Get("combin").Array()
		for _, combin := range combinArr {
			combinMap := combin.(map[string]interface{})

			if combinMap["background"] == nil {
				return errors.New("error: background  nil not find.")
			}
			backgroundPath := combinMap["background"].(string)
			if backgroundPath == "" {
				return errors.New("error: backgroundPath not find.")
			}
			//ext := path.Ext(backgroundPath)
			//	template.ttf[token] = append(template.ttf[token],fontbyte)
			var backgroundInfo *imageInfo = &imageInfo{}
			//获取图片信息
			backgroundInfo.url = backgroundPath
			ret := getImageInfo(backgroundPath, backgroundInfo, "disk")
			if ret != SUCCESS {
				return errors.New("get Image info error.")
			}
			template.background[token] = backgroundInfo
		}
		waterArr, _ := child.Get("water").Array()
		for _, water := range waterArr {
			waterMap := water.(map[string]interface{})

			if waterMap["logo"] == nil {
				return errors.New("error: logo  nil not find.")
			}
			logoPath := waterMap["logo"].(string)
			if logoPath == "" {
				return errors.New("error: logoPath not find.")
			}
			
			var logoInfo *imageInfo = &imageInfo{}
			//获取图片信息
			logoInfo.url = logoPath
			ret := getImageInfo(logoPath, logoInfo, "disk")
			if ret != SUCCESS {
				return errors.New("get Image info error.")
			}
			template.logo[token] = logoInfo
		}
	}
	return nil
}
func initConfig(path string) error {

	data, err := ioutil.ReadFile(path)
	if err != nil {
		log.Warn(err)
		return err
	}
	conf, err := simplejson.NewJson(data)
	if err != nil {
		log.Warn(err)
		return err
	}
	port, _ := conf.Get("port").Int()
	rtime, _ := conf.Get("rtime").Int()
	wtime, _ := conf.Get("wtime").Int()
	ctime, _ := conf.Get("ctime").Int()
	config.port = port
	config.rtime = rtime
	config.wtime = wtime
	config.ctime = ctime
	tplPath, _ := conf.Get("template").String()
	config.template = tplPath
	md5, _ := conf.Get("md5").Bool()
	config.md5 = md5
	pwd, _ := conf.Get("pwd").String()
	config.pwd = pwd
	return err
}
func Start(conf string)  {
	var err error
	err = initConfig(conf)
	if err != nil {
		log.Error(err)
		return 
	}
	err = initTemplate()
	if err != nil {
		log.Error(err)
		return 
	}
	log.Info("config:", conf)
	log.Info("template:", config.template)
	log.Info("start florid App.")
	Server.Run()
}
