package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/beevik/etree"
)

// struct for xml
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

// struct for weather info
type weatherInfo struct {
	Status   string `json:"status"`
	Count    string `json:"count"`
	Info     string `json:"info"`
	Infocode string `json:"infocode"`
	Lives    []live `json:"lives"`
}

type live struct {
	Province      string `json:"province"`
	City          string `json:"city"`
	Adcode        string `json:"adcode"`
	Weather       string `json:"weather"`
	Temperature   string `json:"temperature"`
	Winddirection string `json:"winddirection"`
	Windpower     string `json:"windpower"`
	Humidity      string `json:"humidity"`
	Reporttime    string `json:"reporttime"`
}

func main() {
	//注册回调函数
	http.HandleFunc("/weather", handleGetWeather)
	log.Println("weather server start...")
	//绑定tcp监听地址，并开始接受请求，然后调用服务端处理程序来处理传入的连接请求
	//参数1为addr即监听地址；参数2表示服务端处理程序，通常为nil
	//当参数2为nil时，服务端调用http.DefaultServeMux进行处理
	err := http.ListenAndServe(":8081", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

// getCityWeather from gaode API 通过高德地图API获取天气
func getCityWeather(CityName string) (GetWeatherResponse, error) {

	//get weather info
	resp, err := http.Get("https://restapi.amap.com/v3/weather/weatherInfo?city=610100&key=48630e70f3afd36708389c5dd21c60ba")
	if err != nil {
		log.Fatal(err)
	}

	//decode json
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	wInf := weatherInfo{}
	err = json.Unmarshal(body, &wInf)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	return GetWeatherResponse{Weather: wInf.Lives[0].Weather, Temperature: wInf.Lives[0].Temperature}, nil
}

func handleGetWeather(w http.ResponseWriter, r *http.Request) {
	// 读取请求体中的XML数据
	xmlData, err := ioutil.ReadAll(r.Body)
	if err != nil {
		fmt.Println("Error reading XML data:", err)
		return
	}

	fmt.Println("Received xml:\n", string(xmlData))

	// 解析XML数据
	doc := etree.NewDocument()
	if err := doc.ReadFromBytes(xmlData); err != nil {
		panic(err)
	}

	// 获取城市名称
	cityName := doc.FindElement("//CityName")
	if cityName == nil {
		panic("CityName element not found")
	}
	city := cityName.Text()

	// 获取城市天气
	wresp, err := getCityWeather(city)
	if err != nil {
		fmt.Println("Error get city weather:", err)
		return
	}

	// 构造响应体
	response := Envelope{
		Soap:          "http://www.w3.org/2001/12/soap-envelope",
		EncodingStyle: "http://www.w3.org/2001/12/soap-encoding",
		Body: Body{
			N:        "http://www.nwpu.edu.cn/soa/xml/test",
			Response: &wresp,
		},
	}

	// 返回响应
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(xml.Header))
	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	if err := enc.Encode(response); err != nil {
		fmt.Printf("error: %v\n", err)
	}
}
