package utils

import (
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
	"myoss/mylog"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
)

func GetNetStat() (in, out, inSpeed, outSpeed uint64, err error) {
	s, err := net.IOCounters(true)
	if err != nil {
		return
	}
	for _, s2 := range s {
		if isListContainsStr(excludeNetInterfaces, s2.Name) {
			continue
		}
		in += s2.BytesRecv
		out += s2.BytesSent
	}
	now := uint64(time.Now().Unix())
	diff := now - lastUpdateNetStats
	if diff > 0 {
		inSpeed = (in - netInTransfer) / diff
		outSpeed = (out - netOutTransfer) / diff
	}
	netInTransfer = in
	netOutTransfer = out
	lastUpdateNetStats = now
	return
}
func getDiskTotalAndUsed() (total uint64, used uint64) {
	devices := make(map[string]string)

	diskList, _ := disk.Partitions(false)
	for _, d := range diskList {
		fsType := strings.ToLower(d.Fstype)
		// 不统计 K8s 的虚拟挂载点：https://github.com/shirou/gopsutil/issues/1007
		if devices[d.Device] == "" && isListContainsStr(expectDiskFsTypes, fsType) && !strings.Contains(d.Mountpoint, "/var/lib/kubelet") {
			devices[d.Device] = d.Mountpoint
		}
	}

	for _, mountPath := range devices {
		diskUsageOf, err := disk.Usage(mountPath)
		if err == nil {
			total += diskUsageOf.Total
			used += diskUsageOf.Used
		}
	}

	// Fallback 到这个方法,仅统计根路径,适用于OpenVZ之类的.
	if runtime.GOOS == "linux" && total == 0 && used == 0 {
		cmd := exec.Command("df")
		out, err := cmd.CombinedOutput()
		if err == nil {
			s := strings.Split(string(out), "\n")
			for _, c := range s {
				info := strings.Fields(c)
				if len(info) == 6 {
					if info[5] == "/" {
						total, _ = strconv.ParseUint(info[1], 0, 64)
						used, _ = strconv.ParseUint(info[2], 0, 64)
						// 默认获取的是1K块为单位的.
						total = total * 1024
						used = used * 1024
					}
				}
			}
		}
	}

	return
}

type CpuInfo struct {
	Name  string
	Cores int64
	Used  float64
}
type MemInfo struct {
	Total       int64
	UsedPercent float64
}
type DiskInfo struct {
	Total uint64
	Used  uint64
}
type DeviceInfo struct {
	Cpu       []CpuInfo
	Mem       MemInfo
	Disk      DiskInfo
	BootTime  uint64
	TotalDown uint64
	TotalUp   uint64
	LastDown  uint64
	LastUp    uint64
}

func GetDeviceInfo() (ret DeviceInfo) {
	ret = DeviceInfo{}
	cp, err := cpu.Percent(0, true)
	if err != nil {
		println()
		mylog.Logf("cpu.Percent error:%v", err)
	}
	info, err := cpu.Info()
	if err != nil {
		println()
		mylog.Logf("cpu.Info error:%v", err)
	}
	men, err := mem.VirtualMemory()
	if err != nil {
		println()
		mylog.Logf("mem.VirtualMemory error:%v", err)
	}
	hi, err := host.Info()
	if err != nil {
		println()
		mylog.Logf("host.Info error:%v", err)
	}
	D, U, ND, NU, err := GetNetStat()
	if err != nil {
		println()
		mylog.Logf("GetNetStat error:%v", err)
	}
	for i, infoi := range info {
		//z,ok := cp[i]
		c := CpuInfo{
			Name:  infoi.ModelName,
			Cores: int64(infoi.Cores),
			Used:  cp[i],
		}
		ret.Cpu = append(ret.Cpu, c)
	}
	ret.Mem = MemInfo{
		Total:       int64(men.Total),
		UsedPercent: men.UsedPercent,
	}
	disT, DiskU := getDiskTotalAndUsed()
	ret.Disk = DiskInfo{
		Total: disT,
		Used:  DiskU,
	}
	ret.TotalUp = U
	ret.TotalDown = D
	ret.LastUp = NU
	ret.LastDown = ND
	ret.BootTime = hi.BootTime
	return
}
