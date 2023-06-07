package api

import (
	"hash/fnv"
)

type Key struct {
	ID     string
	Port   int
	Cipher string
	Secret string
}

func (p Key) Hash() uint32 {
	// 创建一个fnv哈希器
	h := fnv.New32a()
	// 写入结构体的字段值
	h.Write([]byte(p.ID))
	h.Write([]byte(p.Secret))
	// 返回哈希值
	return h.Sum32()
}

type UserRets struct {
	Data []Key
}

type UserTraffic struct {
	UID string
	U   int64
	D   int64
}
type WwwTraffic struct {
	UNID   string
	Host   string
	Uip    string
	Date   int64
	IsUdp  int8
	Status int8
}
