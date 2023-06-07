package main

import (
	"fmt"
	"myoss/mylog"
	"myoss/utils"
	"time"
)

func iniSh() error {

	filename := "youhua.sh"
	str, _ := utils.ReadFile(filename)
	if str == "" {
		context := `
apt update -y
apt list --upgradable
apt install tar wget curl supervisor -y
echo '环境优化'
ulimit -n 51200
echo "soft nofile 51200" >> /etc/security/limits.conf
echo "hard nofile 51200" >> /etc/security/limits.conf
(cat <<EOF
fs.file-max = 102400
net.core.somaxconn = 1048576
net.ipv4.tcp_syncookies = 1
net.ipv4.tcp_tw_reuse = 1
net.ipv4.tcp_timestamps = 1
net.ipv4.tcp_fin_timeout = 30
net.core.default_qdisc = fq
net.ipv4.tcp_congestion_control = bbr
net.ipv4.tcp_fastopen = 3
net.ipv4.tcp_max_syn_backlog = 1048576
net.ipv4.tcp_synack_retries = 1
net.ipv4.tcp_orphan_retries = 1
net.ipv4.ip_local_port_range = 32768 65535
net.ipv4.tcp_mem = 88560 118080 177120
net.ipv4.tcp_wmem = 4096 16384 8388608
EOF
    ) > /etc/sysctl.conf
sysctl -p
lsmod | grep bbr
(cat <<EOF
[unix_http_server]
file=/tmp/supervisor.sock   ; the path to the socket file

[supervisord]
logfile=/tmp/supervisord.log ; main log file; default $CWD/supervisord.log
logfile_maxbytes=50MB        ; max main logfile bytes b4 rotation; default 50MB
logfile_backups=10           ; # of main logfile backups; 0 means none, default 10
loglevel=info                ; log level; default info; others: debug,warn,trace
pidfile=/tmp/supervisord.pid ; supervisord pidfile; default supervisord.pid
nodaemon=false               ; start in foreground if true; default false
silent=false                 ; no logs to stdout if true; default false
minfds=1024                  ; min. avail startup file descriptors; default 1024
minprocs=200                 ; min. avail process descriptors;default 200

[rpcinterface:supervisor]
supervisor.rpcinterface_factory = supervisor.rpcinterface:make_main_rpcinterface


[supervisorctl]
serverurl=unix:///tmp/supervisor.sock ; use a unix:// URL  for a unix socket



[program:agss]
user=root
command=/root/supervisd
autostart=true
autorestart=true
startsecs=10
EOF
    ) > /etc/supervisord.conf
`
		err := utils.WriteFile(filename, context, 0644)
		if err != nil {
			return err
		}
	}
	return nil
}
func YouhuaRun() error {
	ishas, err := utils.LocalRunCmd("cat ./hasrun.sh")
	if ishas != "" {
		return nil
	}
	err = iniSh()
	if err != nil {
		mylog.Logf("%v", err)
		return err
	}
	ret, err := utils.LocalRunCmd("chmod +x  ./youhua.sh")
	if err != nil {
		mylog.Logf("%v,%v", err, ret)
		return err
	}
	ret, err = utils.LocalRunCmd("sh ./youhua.sh")
	if err != nil {
		mylog.Logf("%v,%v", err, ret)

		return err
	}
	mylog.Logf("youhuaret:%v", ret)
	ret, err = utils.LocalRunCmd(fmt.Sprintf("cat '%v' > ./hasrun.sh", time.Now().String()))
	if err != nil {
		mylog.Logf("%v,%v", err, ret)
		return err
	}
	return nil
}
