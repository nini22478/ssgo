package api

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"myoss/mylog"
	"myoss/utils"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/bitly/go-simplejson"
	"github.com/go-resty/resty/v2"
)

type APIClient struct {
	client  *resty.Client
	APIHost string
	LogHost string
	NodeID  *int
	Key     string
	Port    int
	Cipher  string
	Rtask   *RepoTask
	Wtask   *RepoWwwTask
}
type RepoTask struct {
	StartTime *time.Time
	RepoList  *[]UserTraffic
}
type RepoWwwTask struct {
	StartTime *time.Time
	RepoList  *[]WwwTraffic
}

func New(apiConfig *Config) *APIClient {

	client := resty.New()
	client.SetRetryCount(3)

	client.SetTimeout(5 * time.Second)

	client.OnError(func(req *resty.Request, err error) {
		if v, ok := err.(*resty.ResponseError); ok {
			// v.Response contains the last response from the server
			// v.Err contains the original error
			log.Print(v.Err)
		}
	})
	client.SetBaseURL(apiConfig.APIHost)
	// client.SetBaseURL(apiConfig.LogHost)
	// Create Key for each requests
	client.SetQueryParams(map[string]string{
		"n": strconv.Itoa(apiConfig.NodeID),
		"t": apiConfig.Key,
	})
	client.OnBeforeRequest(func(client *resty.Client, request *resty.Request) error {
		if request.URL == "/i" {
			//mylog.Logf("%v", request)

		}
		mylog.Logf("%v", request.URL)

		return nil
	})
	// Registering Response Middleware
	t := time.Now().UTC()
	task := RepoTask{
		StartTime: &t,
		RepoList:  &[]UserTraffic{},
	}
	Wtask := RepoWwwTask{
		StartTime: &t,
		RepoList:  &[]WwwTraffic{},
	}
	apiClient := &APIClient{
		client:  client,
		NodeID:  &apiConfig.NodeID,
		Key:     apiConfig.Key,
		APIHost: apiConfig.APIHost,
		LogHost: apiConfig.LogHost,
		Rtask:   &task,
		Wtask:   &Wtask,
	}
	return apiClient
}

type Config struct {
	APIHost string `mapstructure:"ApiHost"`
	LogHost string `mapstructure:"LogHost"`
	NodeID  int    `mapstructure:"NodeID"`
	Key     string `mapstructure:"ApiKey"`
}

func ERandPort() int {
	s := []int{
		11451, 14514, 26708, 36708, 10080, 28818, 21818}
	rand.Shuffle(len(s), func(i, j int) {
		s[i], s[j] = s[j], s[i]
	})
	return s[0]
}
func (c *APIClient) Init() error {
	path := "/api/SsInitFirm"
	res, err := c.client.R().Get(path)
	if err != nil {
		return err
	}
	mylog.Logf("data1:%v", res)

	res_j, err := c.parseResponse(res, path, err)
	//return nil
	if err != nil {
		return err
	}
	mylog.Logf("data:%v", res_j.Get("data"))
	port_str, err := res_j.Get("data").Get("port").String()
	nid := res_j.Get("data").Get("id").MustInt()
	c.NodeID = &nid
	port := utils.Str2Int(port_str)
	mylog.Logf("port:%v\n%v", port, err)
	if port == 0 {
		port = ERandPort()
		c.client.SetQueryParam("n", strconv.Itoa(nid))

		_, err := c.client.R().SetQueryParam("p", strconv.Itoa(port)).Post(path)
		if err != nil {
			return err
		}
		//println(r.String())
	}
	c.Cipher = res_j.Get("data").Get("cipher").MustString()
	c.Port = port
	//print(fmt.Sprintf("%v", res_j))
	return nil
}
func (c *APIClient) GetUsers() (retc *UserRets, err error) {

	// path := "/api/SsGetUsers"
	path := "http://47.92.220.167/api/SsGetUsers"
	retc = &UserRets{}
	// c.client.SetQueryParam("n", strconv.Itoa(*c.NodeID))
	client := resty.New()
	ret, err := client.R().SetQueryParam("n", strconv.Itoa(*c.NodeID)).
		Get(path)
	if ret.StatusCode() != 200 {
		return
	}
	retstr := utils.GenDecode(ret.Body())
	mylog.Logf("retstr:retstr:%v", retstr)
	err = json.Unmarshal(retstr, retc)
	return
}
func (c *APIClient) AddWwwRepo(traffic WwwTraffic) {
	if traffic.UNID != "" && (traffic.Host != "") {
		newlist := append(*c.Wtask.RepoList, traffic)
		c.Wtask.RepoList = &newlist
	}
}
func (c *APIClient) ReportSys() error {
	path := "/api/SsRepoSys"

	data := utils.GetDeviceInfo()
	dat, _ := json.Marshal(data)

	m := map[string]string{}
	m["q"] = base64.StdEncoding.EncodeToString(utils.Gencode(dat))
	dat, _ = json.Marshal(m)
	mylog.Logf("%v", m)
	res, err := c.client.R().
		SetQueryParam("n", strconv.Itoa(*c.NodeID)).
		SetBody(m).
		Post(path)
	_, err = c.parseResponse(res, path, err)
	if err != nil {
		mylog.Logf("ReportSys:err:%v", err)
		return err
	}
	return nil
}
func (c *APIClient) ReportWwwTraffic(traffic *[]WwwTraffic) error {
	// path := "/api/tool/SsRepoWww"
	//
	data := []WwwTraffic{}

	for _, tc := range *traffic {
		data = append(data, tc)
	}
	dat, _ := json.Marshal(data)

	// m := map[string]string{}
	// m["q"] = base64.StdEncoding.EncodeToString(utils.Gencode(dat))
	// dat, _ = json.Marshal(m)
	// mylog.Logf("%v", dat)
	// res, err := c.client.R().
	// 	SetQueryParam("n", strconv.Itoa(*c.NodeID)).
	// 	SetBody(dat).
	// 	Post(path)
	// _, err = c.parseLogResponse(res, path, err)
	// if err != nil {
	// 	mylog.Logf("ReportWwwTraffic:err:%v", err)
	// 	return err
	// }
	url := "http://vice.mobileairport.net/api/tool/SsRepoWwwFirm"
	res, err := http.Post(url, "application/json;charset=utf-8", strings.NewReader(string(dat)))
	if err != nil {
		return err
	}
	defer res.Body.Close()
	// content, err := ioutil.ReadAll(res.Body)
	// if err != nil {
	// 	return err
	// }
	return nil
}
func (c *APIClient) AddRepo(utice *UserTraffic) {

	fmt.Printf("repo add %v", utice)

	if utice.UID != "" && (utice.D != 0 || utice.U != 0) {
		newlist := append(*c.Rtask.RepoList, *utice)
		c.Rtask.RepoList = &newlist
	}
}

// ReportUserTraffic reports the user traffic
func (c *APIClient) ReportUserTraffic(userTraffic *[]UserTraffic) error {
	path := "/api/SsRepoTice"
	//
	data := []UserTraffic{}
	hdata := map[string]*UserTraffic{}
	for _, traffic := range *userTraffic {
		o, ok := hdata[traffic.UID]
		if !ok || o == nil {
			hdata[traffic.UID] = &UserTraffic{
				UID: traffic.UID,
				U:   traffic.U,
				D:   traffic.D}
		} else {
			mylog.Logf("up:%v,down:%v", traffic.U, traffic.D)
			mylog.Logf("up2:%v", o)

			if traffic.D != 0 {
				o.D += traffic.D
			}
			if traffic.U != 0 {
				o.U += traffic.U
			}
			hdata[traffic.UID] = o

		}

	}
	mylog.Logf("up1:%v", hdata)

	for _, tc := range hdata {
		mylog.Logf("up3:%v", tc)

		data = append(data, *tc)
	}
	dat, err := json.Marshal(data)

	m := map[string]string{}
	m["q"] = base64.StdEncoding.EncodeToString(utils.Gencode(dat))
	fmt.Println(string(dat))
	res, err := c.client.R().
		SetQueryParam("n", strconv.Itoa(*c.NodeID)).
		SetBody(m).
		ForceContentType("application/json").
		Post(path)
	_, err = c.parseResponse(res, path, err)
	if err != nil {
		return err
	}
	return nil
}
func (c *APIClient) Debug() {
	c.client.SetDebug(true)
}

func (c *APIClient) assembleURL(path string) string {
	return c.APIHost + path
}
func (c *APIClient) assembleLogURL(path string) string {
	return c.LogHost + path
}
func (c *APIClient) parseResponse(res *resty.Response, path string, err error) (*simplejson.Json, error) {
	if err != nil {
		return nil, fmt.Errorf("request %s failed: %s", c.assembleURL(path), err)
	}

	if res.StatusCode() > 400 {
		body := res.Body()
		return nil, fmt.Errorf("request %s failed: %s, %s", c.assembleURL(path), string(body), err)
	}

	//mylog.Logf("%v",utils.GenDecode(res.Body()))
	rtn, err := simplejson.NewJson(utils.GenDecode(res.Body()))
	if err != nil {
		return nil, fmt.Errorf("ret %s invalid", res.String())
	}
	return rtn, nil
}
func (c *APIClient) parseLogResponse(res *resty.Response, path string, err error) (*simplejson.Json, error) {
	if err != nil {
		return nil, fmt.Errorf("request %s failed: %s", c.assembleLogURL(path), err)
	}

	if res.StatusCode() > 400 {
		body := res.Body()
		return nil, fmt.Errorf("request %s failed: %s, %s", c.assembleLogURL(path), string(body), err)
	}

	//mylog.Logf("%v",utils.GenDecode(res.Body()))
	rtn, err := simplejson.NewJson(utils.GenDecode(res.Body()))
	if err != nil {
		return nil, fmt.Errorf("ret %s invalid", res.String())
	}
	return rtn, nil
}
