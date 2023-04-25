package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/beevik/etree"
)

type Envelope struct {
	XMLName       xml.Name `xml:"soap:Envelope"`
	Soap          string   `xml:"xmlns:soap,attr"`
	EncodingStyle string   `xml:"soap:encodingStyle,attr"`
	Body          Body     `xml:"soap:Body"`
}

type Body struct {
	XMLName  xml.Name            `xml:"soap:Body"`
	N        string              `xml:"xmlns:n,attr"`
	Response *GetWeatherResponse `xml:"m:GetWeatherResponse,omitempty"`
	Request  *GetWeather         `xml:"n:GetWeather,omitempty"`
}

type GetWeather struct {
	CityName string `xml:"CityName,omitempty"`
}

type GetWeatherResponse struct {
	Temperature string `xml:"m:Temperature,omitempty"`
	Weather     string `xml:"m:Weather,omitempty"`
}

func main() {

	request := GetWeather{"西安"}

	body := Body{XMLName: xml.Name{Space: "", Local: "soap:Body"}, N: "http://www.nwpu.edu.cn/soa/xml/test", Request: &request}

	env := Envelope{xml.Name{Space: "", Local: "Envelope"},
		"http://www.w3.org/2001/12/soap-envelope",
		"http://www.w3.org/2001/12/soap-encoding",
		body}

	//生成xml头
	reqxml := []byte(xml.Header)
	//生成xml
	data, err := xml.MarshalIndent(env, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	reqxml = append(reqxml, data...)

	//post xml to server
	resp, err := http.Post("http://localhost:8081/weather",
		"application/xml",
		ioutil.NopCloser(bytes.NewBuffer(reqxml)))

	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	//读取body内的内容
	content, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("response xml:")
	fmt.Println(string(content))

	doc := etree.NewDocument()
	if err := doc.ReadFromBytes(content); err != nil {
		panic(err)
	}

	// 获取城市天气
	weather := doc.FindElement("//Weather")
	if weather == nil {
		panic("Weather element not found")
	}
	// 获取城市温度
	temperature := doc.FindElement("//Temperature")
	if temperature == nil {
		panic("Temperature element not found")
	}

	fmt.Printf("\n西安的天气为%s, 温度为%s\n", weather.Text(), temperature.Text())
}
