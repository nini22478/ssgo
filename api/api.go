package api

import (
	"database/sql"
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
	_ "github.com/go-sql-driver/mysql"
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
type Configs struct {
	Database DatabaseConfig `toml:"database"`
}

// DatabaseConfig 结构体用于存储数据库配置信息
type DatabaseConfig struct {
	Username string `toml:"username"`
	Password string `toml:"password"`
	Host     string `toml:"host"`
	Port     int    `toml:"port"`
	DBName   string `toml:"dbname"`
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

// 定义 Key 结构体，表示每个查询结果的键值对
type Keys struct {
	Port   int
	Cipher string
	// 添加其他字段根据你的表结构
}
type Key2 struct {
	Nid   string
	WgKey string
	// 添加其他字段根据你的表结构
}

// 合并 Keys 和 Key2 结构体的方法
func mergeToUserEntry(keys Keys, key2 Key2) Key {
	return Key{
		ID:     key2.Nid,
		Port:   keys.Port,   // 这里使用了 Keys 结构体的 ID 字段
		Cipher: keys.Cipher, // 这里使用了 Key2 结构体的 Secret 字段
		Secret: key2.WgKey,  // 这里使用了 Keys 结构体的 Name 字段
	}
}

// 定义一个结构体，表示数据库连接
type Database struct {
	db *sql.DB
}

// 数据库初始化函数
func NewDatabase(username, password, dbname string) (*Database, error) {
	// 读取配置文件
	// var config Configs
	// if _, err := toml.DecodeFile("config.toml", &config); err != nil {
	// 	log.Fatal(err)
	// }
	// 构建数据库连接字符串
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", username, password,
		"slave1.cgjczblgjaws.ap-east-1.rds.amazonaws.com", 3306, dbname)

	// 打开数据库连接
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	// 检查数据库连接是否正常
	err = db.Ping()
	if err != nil {
		return nil, err
	}

	return &Database{db}, nil
}

// 查询函数
func (db *Database) Query(query string) (*sql.Rows, error) {
	rows, err := db.db.Query(query)
	if err != nil {
		return nil, err
	}
	return rows, nil
}

// 关闭数据库连接函数
func (db *Database) Close() {
	db.db.Close()
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
	path := "/api/SsInit"
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

	// // path := "/api/SsGetUsers"
	// path := "http://vice.mobileairport.net/api/tool/GetUsers"
	retc = &UserRets{}
	// // c.client.SetQueryParam("n", strconv.Itoa(*c.NodeID))
	// client := resty.New()
	// ret, err := client.R().SetQueryParam("n", strconv.Itoa(*c.NodeID)).
	// 	Get(path)
	// if ret.StatusCode() != 200 {
	// 	return
	// }
	// retstr := utils.GenDecode(ret.Body())
	// err = json.Unmarshal(retstr, retc)
	// 初始化数据库连接
	db, err := NewDatabase("admin", "jzhMzB69OZaAHzNGyNYU", "vpnplan")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// 执行第一个查询
	query1 := "SELECT group_id,port,cipher FROM server_shadowsocks where id =" + strconv.Itoa(*c.NodeID)
	rows1, err := db.Query(query1)
	if err != nil {
		log.Fatal(err)
	}
	defer rows1.Close()

	// 处理第一个查询结果
	for rows1.Next() {
		var groupid int
		var key Keys
		// 根据你的表结构定义相应的变量

		err := rows1.Scan(&groupid, &key.Port, &key.Cipher)
		if err != nil {
			log.Fatal(err)
		}
		// 执行第二个查询
		query2 := "SELECT nid,wg_key FROM m_user where sup_id != 0 and sup_id >=" + strconv.Itoa(groupid)
		rows2, err := db.Query(query2)
		if err != nil {
			log.Fatal(err)
		}
		defer rows2.Close()

		// 处理第二个查询结果
		for rows2.Next() {
			// 处理第二个查询结果的逻辑，类似上面的处理方式
			var key2 Key2
			err := rows2.Scan(&key2.Nid, &key2.WgKey)
			if err != nil {
				log.Fatal(err)
			}

			// 将第一个查询的数据和第二个查询的数据合并到 Data 切片中
			retc.Data = append(retc.Data, mergeToUserEntry(key, key2))
		}
	}
	// 检查是否有错误导致迭代结束
	if err := rows1.Err(); err != nil {
		log.Fatal(err)
	}

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
	url := "http://vice.mobileairport.net/api/tool/SsRepoWww"
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
