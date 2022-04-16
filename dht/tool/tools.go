package tool

import (
	"bytes"
	"crypto/rand"
	"crypto/sha1"
	"fmt"
)

var secret string = "ghr"

func RandString(num int) string {
	b := make([]byte, num)
	n, err := rand.Read(b)
	if err != nil {
		fmt.Printf("rand id err:%s\n", err.Error())
		panic(err)
	}
	if n != num {
		fmt.Printf("rand id len error :%d\n", n)
		panic(err)
	}
	return string(b)
}

func MakeRequest(method string, nodeId string, target string) map[string]interface{} {
	neighbordId := nodeId
	if len(target) != 0 {
		neighbordId = NeighborId(nodeId, target)
	}
	ret := make(map[string]interface{})
	ret["t"] = RandString(2)
	ret["y"] = "q"
	ret["q"] = method
	ret["a"] = map[string]interface{}{"id": neighbordId, "target": RandString(20)}
	return ret
}

func MakeResponse(t string, r map[string]interface{}) map[string]interface{} {
	ret := make(map[string]interface{})
	ret["t"] = t
	ret["y"] = "r"
	ret["r"] = r

	return ret
}

func MakeToken(ip string) string {
	s := sha1.New()
	s.Write([]byte(ip))
	s.Write([]byte(secret))
	return string(s.Sum(nil))
}

func ValidateToken(token string, ip string) bool {
	return token == MakeToken(ip)
}

func NeighborId(nodeId string, target string) string {
	if len(target) < 15 || len(nodeId) < 15 {
		defer recover()
	}
	return target[0:15] + nodeId[15:]
}

func MakePreHeader() []byte {
	buf := bytes.NewBuffer(nil)
	buf.WriteByte(19)
	buf.WriteString("BitTorrent protocol")
	buf.Write([]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x10, 0x00, 0x00})
	return buf.Bytes()
}
