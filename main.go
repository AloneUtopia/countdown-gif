package main

import (
	"bytes"
	"fmt"
	"github.com/fogleman/gg"
	"image"
	"image/color/palette"
	"image/draw"
	"image/gif"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"
)

// http://127.0.0.1:9090/countdown-gif?endTime=1637410929

func main() {

	http.HandleFunc("/countdown-gif", func(writer http.ResponseWriter, request *http.Request) {
		params := request.URL.Query()
		endTime, err := strconv.Atoi(params.Get("endTime"))
		if err != nil {
			endTime = 0
		}
		gifPath, err := countdownGif(endTime)
		if err != nil {
			fmt.Fprintf(writer, err.Error())
		} else {
			http.ServeFile(writer, request, gifPath)
			os.Remove(gifPath)
		}
	})

	go func() {
		for {
			time.Sleep(time.Second)
			resp, err := http.Get("http://localhost:9090/countdown-gif")
			if err != nil {
				continue
			}
			resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				continue
			}
			break
		}
		log.Println("倒计时gif 服务已启动： http://localhost:9090/countdown-gif?endTime=timeStamp")
	}()

	err := http.ListenAndServe(":9090", nil)
	if err != nil {
		log.Fatalf("启动 HTTP 服务器失败: %v", err)
	}

}

//倒计时gif
func countdownGif(endTime int) (string, error) {
	dx := 760
	dy := 160
	fontFile := "msyh.ttf"
	fontSize := 80
	frame := 60
	gifPath := strconv.Itoa(endTime) + ".gif"

	//最大时间差(99 : 23 : 59 : 59)
	maxDistance := 100*24*60*60 - 1

	timeUnix := time.Now().Unix()
	timeDistance := int(int64(endTime) - timeUnix)

	//处理时间差及 gif 帧数
	if timeDistance > maxDistance {
		timeDistance = maxDistance
	} else if timeDistance > 0 && timeDistance < 60 {
		frame = timeDistance
	} else if timeDistance < 0 {
		timeDistance = 0
		frame = 1
	}
	imagePath := make([]string, frame)
	fileName := getCode(10)

	//用协程来绘制每一帧图片
	chs := make([]chan error, frame)
	for i := 0; i < frame; i++ {
		chs[i] = make(chan error)
		savePath := "./" + fileName + strconv.Itoa(i) + ".png"
		imagePath[i] = savePath
		go drawPng(savePath, timeDistance-i, chs[i], dx, dy, fontFile, fontSize)
	}
	for _, ch := range chs {
		drawPngError := <-ch
		if drawPngError != nil {
			return "", drawPngError
		}
	}
	//将绘制好的每一帧图片合成 gif
	drawGifErr := drawGif(imagePath, gifPath)
	if drawGifErr != nil {
		return "", drawGifErr
	}
	return gifPath, nil
}

//绘制图片
func drawPng(savePath string, timeDistance int, ch chan error, dx int, dy int, fontFile string, fontSize int) {

	timeStr := timeDistanceToStr(timeDistance)

	dc := gg.NewContext(dx, dy)
	//绘制背景
	dc.SetRGB255(255, 255, 255)
	dc.Clear()

	//设置文字的颜色、字体、大小
	dc.SetRGB255(0, 0, 255)
	if err := dc.LoadFontFace(fontFile, float64(fontSize)); err != nil {
		panic(err)
	}
	//获取文字的长宽
	sWidth, sHeight := dc.MeasureString(timeStr)

	//居中绘制文字
	dc.DrawString(timeStr, (float64(dx)-sWidth)/2, (float64(dy)+sHeight)/2)

	err := dc.SavePNG(savePath)
	if err != nil {
		ch <- err
	} else {
		ch <- nil
	}

}

//将图片合成gif
func drawGif(imagePath []string, gifPath string) error {
	g := gif.GIF{}

	for _, src := range imagePath {
		f, err := os.Open(src)
		if err != nil {
			fmt.Printf("Could not open file %s. Error: %s\n", src, err)
			return err
		}

		//解码图片
		img, _, _ := image.Decode(f)
		Paletted := image.NewPaletted(img.Bounds(), palette.Plan9)
		draw.FloydSteinberg.Draw(Paletted, img.Bounds(), img, image.Point{})

		//每一帧的图片
		g.Image = append(g.Image, Paletted)
		//每一帧的延迟时间 单位0.01 秒
		g.Delay = append(g.Delay, 100)
		f.Close()
		os.Remove(src)
	}

	f, _ := os.Create(gifPath)
	defer f.Close()

	err := gif.EncodeAll(f, &g)
	if err != nil {
		return err
	}
	return nil
}

//获取指定长度的随机字符串
func getCode(codeLen int) string {
	rawStr := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890"

	buf := make([]byte, 0, codeLen)
	b := bytes.NewBuffer(buf)
	// 随机从中获取
	rand.Seed(time.Now().UnixNano())
	for rawStrLen := len(rawStr); codeLen > 0; codeLen-- {
		randNum := rand.Intn(rawStrLen)
		b.WriteByte(rawStr[randNum])
	}
	return b.String()
}

//将时间差转换成格式时间字符  02 ： 15 : 03 : 04
func timeDistanceToStr(timeDistance int) string {

	d := timeDistance / (24 * 60 * 60)
	dModulo := timeDistance % (24 * 60 * 60)

	h := dModulo / (60 * 60)
	hModulo := dModulo % (60 * 60)

	i := hModulo / 60

	s := hModulo % 60

	dStr := fmt.Sprintf("%02d", d)
	hStr := fmt.Sprintf("%02d", h)
	iStr := fmt.Sprintf("%02d", i)
	sStr := fmt.Sprintf("%02d", s)

	str := dStr + " : " + hStr + " : " + iStr + " : " + sStr

	return str
}
