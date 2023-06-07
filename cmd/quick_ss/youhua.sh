apt-get update -y
apt-get install tar wget curl lrszs supervisor -y > /dev/null
apt-get update -y
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
file=/tmp/supervisor.sock
supervisord]
logfile=/tmp/supervisord.log ; main log file; default $CWD/supervisord.log
logfile_maxbytes=50MB        ; max main logfile bytes b4 rotation; default 50MB
logfile_backups=10           ; # of main logfile backups; 0 means none, default 10
loglevel=info                ; log level; default info; others: debug,warn,trace
pidfile=/tmp/supervisord.pid ; supervisord pidfile; default supervisord.pid
nodaemon=false               ; start in foreground if true; default false
silent=false                 ; no logs to stdout if true; default false
minfds=1024                  ; min. avail startup file descriptors; default 1024
minprocs=200
[supervisorctl]
serverurl=unix:///tmp/supervisor.sock ; use a unix:// URL  for a unix socket
[program:agss]
user=root
command=/root/quick_ss
autostart=true
autorestart=true
startsecs=10
EOF
    ) > /etc/supervisord.conf