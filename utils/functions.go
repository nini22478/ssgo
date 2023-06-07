package utils

import (
	"bytes"
	"fmt"
	"github.com/commander-cli/cmd"
	gzip "github.com/klauspost/pgzip"
	"github.com/shirou/gopsutil/v3/host"
	"io/ioutil"
	"myoss/mylog"
	"os"
	"strconv"
	"strings"
	"time"
	"unsafe"
)

var (
	netInTransfer, netOutTransfer, lastUpdateNetStats uint64
	cachedBootTime                                    time.Time
	expectDiskFsTypes                                 = []string{
		"apfs", "ext4", "ext3", "ext2", "f2fs", "reiserfs", "jfs", "btrfs",
		"fuseblk", "zfs", "simfs", "ntfs", "fat32", "exfat", "xfs", "fuse.rclone",
	}
	excludeNetInterfaces = []string{
		"lo", "tun", "docker", "veth", "br-", "vmbr", "vnet", "kube",
	}
)

func LocalRunCmd(cm string) (string, error) {
	c := cmd.NewCommand(cm)

	err := c.Execute()
	if err != nil {
		return "", err
	}
	if c.Stderr() != "" {
		return "", fmt.Errorf("LocalRunCmd-error:%v", c.Stderr())
	}
	return c.Stdout(), nil
}
func ReadFile(filename string) (string, error) {
	file, err := os.ReadFile(filename)
	if err != nil {
		return "", err
	}
	return string(file), nil
}

func byteSliceToString(bytes []byte) string {

	return *(*string)(unsafe.Pointer(&bytes))

}
func Str2Int(pre string) int {
	val, err := strconv.ParseInt(pre, 0, 64)
	if err == nil {
		return int(val)
	}
	return 0
}
func stringTobyteSlice(s string) []byte {

	tmp1 := (*[2]uintptr)(unsafe.Pointer(&s))

	tmp2 := [3]uintptr{tmp1[0], tmp1[1], tmp1[1]}

	return *(*[]byte)(unsafe.Pointer(&tmp2))

}

func WriteFile(filename, context string, mode os.FileMode) error {
	return ioutil.WriteFile(filename, stringTobyteSlice(context), mode)
}
func Gencode(byte []byte) []byte {
	var b bytes.Buffer
	gz, err := gzip.NewWriterLevel(&b, 9)
	if err != nil {

		return nil
	}
	if _, err := gz.Write(byte); err != nil {
		//log.Fatal(err)
		return nil
	}
	if err := gz.Close(); err != nil {
		//log.Fatal(err)
		return nil
	}
	return b.Bytes()
}
func GenDecode(byt []byte) []byte {
	var b bytes.Buffer
	b.Write(byt)
	r, _ := gzip.NewReader(&b)
	var b2 bytes.Buffer
	_, e := r.WriteTo(&b2)
	if e != nil {
		mylog.Logf("GenDecode err:%v", e)
	}
	//fmt.Println(b2.String())
	return b2.Bytes()
}
func FormatFileSize(fileSize int64) (size string) {
	if fileSize < 1024 {
		//return strconv.FormatInt(fileSize, 10) + "B"
		return fmt.Sprintf("%.2fB", float64(fileSize)/float64(1))
	} else if fileSize < (1024 * 1024) {
		return fmt.Sprintf("%.2fKB", float64(fileSize)/float64(1024))
	} else if fileSize < (1024 * 1024 * 1024) {
		return fmt.Sprintf("%.2fMB", float64(fileSize)/float64(1024*1024))
	} else if fileSize < (1024 * 1024 * 1024 * 1024) {
		return fmt.Sprintf("%.2fGB", float64(fileSize)/float64(1024*1024*1024))
	} else if fileSize < (1024 * 1024 * 1024 * 1024 * 1024) {
		return fmt.Sprintf("%.2fTB", float64(fileSize)/float64(1024*1024*1024*1024))
	} else { //if fileSize < (1024 * 1024 * 1024 * 1024 * 1024 * 1024)
		return fmt.Sprintf("%.2fPB", float64(fileSize)/float64(1024*1024*1024*1024*1024))
	}
}

func isListContainsStr(list []string, str string) bool {
	for i := 0; i < len(list); i++ {
		if strings.Contains(str, list[i]) {
			return true
		}
	}
	return false
}

func GetStatrtTime() {
	timestamp, _ := host.BootTime()
	t := time.Unix(int64(timestamp), 0)
	fmt.Println(t.Local().Format("2006-01-02 15:04:05"))
}
