package main

import (
	"container/list"
	"flag"
	"fmt"
	"math/rand"
	"myoss/api"
	"myoss/mylog"
	ss "myoss/shadowsocks"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"myoss/service"
	"myoss/service/metrics"

	"github.com/op/go-logging"
	"github.com/oschwald/geoip2-golang"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/crypto/ssh/terminal"
)

var logger *logging.Logger

// Set by goreleaser default ldflags. See https://goreleaser.com/customization/build/
var version = "dev"

// 59 seconds is most common timeout for servers that do not respond to invalid requests
const tcpReadTimeout time.Duration = 59 * time.Second

// A UDP NAT timeout of at least 5 minutes is recommended in RFC 4787 Section 4.3.
const defaultNatTimeout time.Duration = 5 * time.Minute

func init() {
	var prefix = "%{level:.1s}%{time:2006-01-02T15:04:05.000Z07:00} %{pid} %{shortfile}]"
	if terminal.IsTerminal(int(os.Stderr.Fd())) {
		// Add color only if the output is the terminal
		prefix = strings.Join([]string{"%{color}", prefix, "%{color:reset}"}, "")
	}
	logging.SetFormatter(logging.MustStringFormatter(strings.Join([]string{prefix, " %{message}"}, "")))
	logging.SetBackend(logging.NewLogBackend(os.Stderr, "", 0))
	logger = logging.MustGetLogger("")
}

type ssPort struct {
	tcpService service.TCPService
	udpService service.UDPService
	cipherList service.CipherList
}

type SSServer struct {
	natTimeout  time.Duration
	m           metrics.ShadowsocksMetrics
	replayCache service.ReplayCache
	ports       map[int]*ssPort
	api         *api.APIClient
}

func (s *SSServer) startPort(portNum int) error {
	listener, err := net.ListenTCP("tcp", &net.TCPAddr{Port: portNum})
	if err != nil {
		return fmt.Errorf("Failed to start TCP on port %v: %v", portNum, err)
	}
	packetConn, err := net.ListenUDP("udp", &net.UDPAddr{Port: portNum})
	if err != nil {
		return fmt.Errorf("Failed to start UDP on port %v: %v", portNum, err)
	}
	logger.Infof("Listening TCP and UDP on port %v", portNum)
	port := &ssPort{cipherList: service.NewCipherList()}
	// TODO: Register initial data metrics at zero.
	port.tcpService = service.NewTCPService(port.cipherList, &s.replayCache, s.m, tcpReadTimeout, s.api)
	port.udpService = service.NewUDPService(s.natTimeout, port.cipherList, s.m, s.api)
	s.ports[portNum] = port
	go port.tcpService.Serve(listener)
	go port.udpService.Serve(packetConn)
	return nil
}

func (s *SSServer) removePort(portNum int) error {
	port, ok := s.ports[portNum]
	if !ok {
		return fmt.Errorf("Port %v doesn't exist", portNum)
	}
	tcpErr := port.tcpService.Stop()
	udpErr := port.udpService.Stop()
	delete(s.ports, portNum)
	if tcpErr != nil {
		return fmt.Errorf("Failed to close listener on %v: %v", portNum, tcpErr)
	}
	if udpErr != nil {
		return fmt.Errorf("Failed to close packetConn on %v: %v", portNum, udpErr)
	}
	logger.Infof("Stopped TCP and UDP on port %v", portNum)
	return nil
}

func (s *SSServer) doRun() error {
	logger.Infof("doRun")

	//config, err := readConfig(filename)
	//if err != nil {
	//	return fmt.Errorf("Failed to read config file %v: %v", filename, err)
	//}
	users, err := s.api.GetUsers()
	if err != nil {
		return err
	}
	//keys := []Key{}
	//keys = append(keys, Key{
	//	ID:     "user-0",
	//	Port:   9000,
	//	Cipher: "chacha20-ietf-poly1305",
	//	Secret: "121212",
	//})
	//keys = append(keys, Key{
	//	ID:     "user-1",
	//	Port:   9000,
	//	Cipher: "aes-128-gcm",
	//	Secret: "232323",
	//})
	//config := &Config{
	//	Keys: keys,
	//}
	portChanges := make(map[int]int)
	portCiphers := make(map[int]*list.List) // Values are *List of *CipherEntry.
	for _, keyConfig := range users.Data {
		portChanges[keyConfig.Port] = 1
		cipherList, ok := portCiphers[keyConfig.Port]
		if !ok {
			cipherList = list.New()
			portCiphers[keyConfig.Port] = cipherList
		}
		cipher, err := ss.NewCipher(keyConfig.Cipher, keyConfig.Secret)
		if err != nil {
			return fmt.Errorf("Failed to create cipher for key %v: %v", keyConfig.ID, err)
		}
		entry := service.MakeCipherEntry(keyConfig.ID, cipher, keyConfig.Secret)
		cipherList.PushBack(&entry)
	}
	for port := range s.ports {
		portChanges[port] = portChanges[port] - 1
	}
	for portNum, count := range portChanges {
		if count == -1 {
			if err := s.removePort(portNum); err != nil {
				return fmt.Errorf("Failed to remove port %v: %v", portNum, err)
			}
		} else if count == +1 {
			if err := s.startPort(portNum); err != nil {
				return fmt.Errorf("Failed to start port %v: %v", portNum, err)
			}
		}
	}
	for portNum, cipherList := range portCiphers {
		s.ports[portNum].cipherList.Update(cipherList)
	}
	logger.Infof("Loaded %v access keys", len(users.Data))
	s.m.SetNumAccessKeys(len(users.Data), len(portCiphers))
	return nil
}

func (s *SSServer) CheckWwwRepo() {
	ticker := time.NewTicker(30 * time.Second)
	for {
		<-ticker.C
		if len(*s.api.Wtask.RepoList) == 0 {
			logger.Infof("empty www repo")
			continue
		}
		now := time.Now().UTC()
		if (now.Unix() - s.api.Wtask.StartTime.Unix()) > 30 {
			s.api.ReportWwwTraffic(s.api.Wtask.RepoList)
			s.api.Wtask.RepoList = &[]api.WwwTraffic{}
			s.api.Wtask.StartTime = &now
		} else {
			logger.Infof("www repo no time %v ---%v", s.api.Wtask.StartTime.String(), now.String())
		}
	}
}
func (s *SSServer) CheckRepo() {
	randomNumber := rand.Intn(300) + 300
	interval := time.Duration(randomNumber) * time.Second
	ticker := time.NewTicker(interval)
	for {
		<-ticker.C
		if len(*s.api.Rtask.RepoList) == 0 {
			logger.Infof("empty repo")
			continue
		}
		now := time.Now().UTC()
		if (now.Unix() - s.api.Rtask.StartTime.Unix()) > 9 {
			s.api.ReportUserTraffic(s.api.Rtask.RepoList)
			s.api.Rtask.RepoList = &[]api.UserTraffic{}
			s.api.Rtask.StartTime = &now
		} else {
			logger.Infof("repo no time %v ---%v", s.api.Rtask.StartTime.String(), now.String())
		}
	}
}
func (s *SSServer) RepoSys() {
	ticker := time.NewTicker(300 * time.Second)
	for {
		<-ticker.C
		err := s.api.ReportSys()
		if err != nil {
			mylog.Logf("RepoSys err:%v", err)
		}
	}
}
func (s *SSServer) CheckUser(sigHup chan os.Signal) {
	rand.Seed(time.Now().UnixNano())
	// 生成30到60之间的随机整数
	randomNumber := rand.Intn(10) + 30
	interval := time.Duration(randomNumber) * time.Second
	fmt.Println("随机生成的数字为:", interval)
	ticker := time.NewTicker(interval)
	hash := map[api.Key]uint32{}
	// 在无限循环中接收信号并打印当前时间
	for {
		<-ticker.C
		doNew := false

		users, err := s.api.GetUsers()
		if err != nil {
			println(err)
			continue
		}

		if len(users.Data) != len(hash) {
			signal.Notify(sigHup, syscall.SIGHUP)
			doNew = true
			hash = make(map[api.Key]uint32)
			logger.Infof("len(users.Data) != len(hash)")
		}
		for _, ok := range users.Data {
			hashkey := ok.Hash()
			vk, vok := hash[ok]
			if !vok {
				signal.Notify(sigHup, syscall.SIGHUP)
				logger.Infof("!vok")
				doNew = true
				//break
			}
			if vk != hashkey {
				signal.Notify(sigHup, syscall.SIGHUP)
				logger.Infof("vk != hashkey")
				doNew = true

				//break
			}
			hash[ok] = hashkey
		}
		if doNew {
			s.doRun()
		}

	}

}

// Stop serving on all ports.
func (s *SSServer) Stop() error {
	for portNum := range s.ports {
		if err := s.removePort(portNum); err != nil {
			return err
		}
	}
	return nil
}

// RunSSServer starts a shadowsocks server running, and returns the server or an error.
func RunSSServer(natTimeout time.Duration, sm metrics.ShadowsocksMetrics, replayHistory int, api2 *api.APIClient) (*SSServer, error) {
	server := &SSServer{
		natTimeout:  natTimeout,
		m:           sm,
		replayCache: service.NewReplayCache(replayHistory),
		ports:       make(map[int]*ssPort),
		api:         api2,
	}
	//err := server.loadConfig(filename)
	server.api.Init()
	err := server.doRun()
	if err != nil {
		return nil, fmt.Errorf("Failed to dorun: %v", err)
	}
	sigHup := make(chan os.Signal, 1)
	go server.CheckUser(sigHup)
	go server.RepoSys()
	go server.CheckRepo()
	go server.CheckWwwRepo()
	//signal.Notify(sigHup, syscall.SIGHUP)
	//go func() {
	//	for range sigHup {
	//		logger.Info("Updating config")
	//		if err := server.doRun(); err != nil {
	//			logger.Errorf("Could not reload config: %v", err)
	//		}
	//	}
	//}()
	return server, nil
}

func main() {
	var youhua string

	flag.StringVar(&youhua, "y", "n", "init")
	flag.Parse()
	if youhua != "n" {
		YouhuaRun()
		return
	}
	//flag.Parse()

	//if flags.Verbose {
	//	logging.SetLevel(logging.DEBUG, "")
	//} else {
	//
	//}
	logging.SetLevel(logging.INFO, "")
	//if flags.Version {
	//	fmt.Println(version)
	//	return
	//}

	//if flags.MetricsAddr != "" {
	//	http.Handle("/metrics", promhttp.Handler())
	//	go func() {
	//		logger.Fatal(http.ListenAndServe(flags.MetricsAddr, nil))
	//	}()
	//	logger.Infof("Metrics on http://%v/metrics", flags.MetricsAddr)
	//}

	var ipCountryDB *geoip2.Reader
	var err error

	api2 := api.New(&api.Config{APIHost: "https://aerodrome.onemelody.cn/", LogHost: "http://vice.mobileairport.net/", Key: "fe6fcd397f783b5548c918e6a026bb2d"})

	//if flags.IPCountryDB != "" {
	//	logger.Infof("Using IP-Country database at %v", flags.IPCountryDB)
	//	ipCountryDB, err = geoip2.Open(flags.IPCountryDB)
	//	if err != nil {
	//		log.Fatalf("Could not open geoip database at %v: %v", flags.IPCountryDB, err)
	//	}
	//	defer ipCountryDB.Close()
	//}
	m := metrics.NewPrometheusShadowsocksMetrics(ipCountryDB, prometheus.DefaultRegisterer)
	m.SetBuildInfo(version)
	_, err = RunSSServer(defaultNatTimeout, m, 0, api2)
	if err != nil {
		logger.Fatal(err)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
}
