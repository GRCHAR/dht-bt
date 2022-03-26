package store

import (
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"net"

	"github.com/ghr-dht/tool"
	"github.com/marksamman/bencode"
)

const (
	perBlock        = 16384
	maxMetadataSize = perBlock * 1024
	extended        = 20
	extHandshake    = 0
)

type Meta struct {
	addr         string
	infoHash     []byte
	conn         net.Conn
	peerId       string
	preHeader    []byte
	metadataSize int64
	utMetadata   int64
	pieceCount   int64
	pieces       [][]byte
}

type Tfile struct {
	Name   string `json:"file_name"`
	Length int64  `json:"file_len"`
}

type Torrent struct {
	HashHex string   `json:"hash"`
	Name    string   `json:"name"`
	Length  int64    `json:"len"`
	Files   []*Tfile `json:"files"`
}

type HashPair struct {
	Addr string
	Hash []byte
}

type HandshakeMesage struct {
	m map[string]interface{}
}

var HashChan chan *HashPair

func init() {
	HashChan = make(chan *HashPair, 10000)
}

func NewMeta(addr string, hash []byte) *Meta {
	return &Meta{
		addr:      addr,
		infoHash:  hash,
		peerId:    tool.RandString(20),
		preHeader: tool.MakePreHeader(),
	}
}

func GetMeta() {
	for {
		select {
		case hashPair := <-HashChan:
			go getTorrentMsg(hashPair)
		}
	}
}

func getTorrentMsg(hashPair *HashPair) {
	m := NewMeta(hashPair.Addr, hashPair.Hash)
	err := m.handshake()

	if err != nil {
		return
	}
	err = m.extHandShake()
	if err != nil {
		log.Println("extHandShake err", err)
		return
	}

	metadata, err := m.Begin()
	if err != nil {
		log.Println("begin err", err)
		return
	}
	defer m.conn.Close()
	parseMetaData(metadata, hex.EncodeToString(hashPair.Hash))

}

func (m *Meta) Begin() ([]byte, error) {

	for {
		data, err := m.ReadN()
		if err != nil {
			return nil, err
		}

		if len(data) < 2 {
			continue
		}

		if data[0] != extended {
			continue
		}

		if data[1] != 1 {
			continue
		}

		err = m.readOnePiece(data[2:])

		if err != nil {
			return nil, err
		}

		if !m.checkDone() {
			continue
		}

		pie := bytes.Join(m.pieces, []byte(""))
		sum := sha1.Sum(pie)
		if bytes.Equal(sum[:], m.infoHash) {
			return pie, nil
		}

		return nil, errors.New("metadata checksum mismatch")
	}
}

func (m *Meta) checkDone() bool {
	for _, value := range m.pieces {
		if value == nil {
			return false
		}
	}
	return true
}
func (m *Meta) readOnePiece(payload []byte) error {
	trailerIndex := bytes.Index(payload, []byte("ee")) + 2
	if trailerIndex == 1 {
		return errors.New("ee == 1")
	}

	dict, err := bencode.Decode(bytes.NewBuffer(payload[:trailerIndex]))
	if err != nil {
		return err
	}

	pieceIndex, ok := dict["piece"].(int64)
	if !ok || pieceIndex >= m.pieceCount {
		return errors.New("piece num error")
	}

	msgType, ok := dict["msg_type"].(int64)
	if !ok || msgType != 1 {
		return errors.New("piece type error")
	}
	m.pieces[pieceIndex] = payload[trailerIndex:]
	return nil
}

func (m *Meta) handshake() error {

	err := *new(error)
	m.conn, err = net.Dial("tcp", m.addr)
	if err != nil {

		return err
	}
	if err != nil {
		return err
	}
	buf := bytes.NewBuffer(nil)
	buf.Write(m.preHeader)
	buf.Write(m.infoHash)
	buf.WriteString(m.peerId)
	_, err = m.conn.Write(buf.Bytes())
	if err != nil {
		log.Println(err)
		return err
	}

	res := make([]byte, 68)
	n, err := io.ReadFull(m.conn, res)
	if n != 68 {
		return errors.New("hand read len err")
	}

	if !bytes.Equal(res[:20], m.preHeader[:20]) {
		return errors.New("remote peer not supporting bittorrent protocol")
	}

	if res[25]&0x10 != 0x10 {
		return errors.New("remote peer not supporting extension protocol")
	}

	if !bytes.Equal(res[28:48], m.infoHash) {
		return errors.New("invalid bittorrent header response")
	}

	// log.Printf("handshake read:%v  %v\n", string(res), n)
	return nil
}

//二进制大端发送
func (m *Meta) WriteTo(data []byte) error {

	length := uint32(len(data))

	buf := bytes.NewBuffer(nil)
	binary.Write(buf, binary.BigEndian, length)

	sendMsg := append(buf.Bytes(), data...)
	_, err := m.conn.Write(sendMsg)
	if err != nil {
		return fmt.Errorf("write message failed: %v", err)
	}
	return nil
}

func (m *Meta) ReadN() ([]byte, error) {
	length := make([]byte, 4)
	_, err := io.ReadFull(m.conn, length)
	if err != nil {
		return nil, err
	}

	size := binary.BigEndian.Uint32(length)

	data := make([]byte, size)
	_, err = io.ReadFull(m.conn, data)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (m *Meta) extHandShake() error {
	//etxHandShark
	data := append([]byte{extended, extHandshake}, bencode.Encode(map[string]interface{}{
		"m": map[string]interface{}{
			"ut_metadata": 1,
		},
	})...)

	if err := m.WriteTo(data); err != nil {
		return err
	}

	data, err := m.ReadN()
	if err != nil {
		return err
	}

	if data[0] != extended {
		return errors.New("data 0 err")
	}
	if data[1] != 0 {
		return errors.New("data 1 err")
	}
	return m.onExtHandshake(data[2:])
}

func (this *Meta) onExtHandshake(payload []byte) error {

	dict, err := bencode.Decode(bytes.NewBuffer(payload))
	if err != nil {
		return err
	}

	metadataSize, ok := dict["metadata_size"].(int64)
	if !ok {
		return errors.New("invalid extension header response")
	}

	if metadataSize > maxMetadataSize {
		return errors.New("metadata_size too long")
	}

	if metadataSize < 0 {
		return errors.New("negative metadata_size")
	}

	m, ok := dict["m"].(map[string]interface{})
	if !ok {
		return errors.New("negative metadata m")
	}

	utMetadata, ok := m["ut_metadata"].(int64)
	if !ok {
		return errors.New("negative metadata ut_metadata")
	}
	this.metadataSize = metadataSize
	this.utMetadata = utMetadata
	this.pieceCount = metadataSize / perBlock
	if this.metadataSize%perBlock != 0 {
		this.pieceCount++
	}
	this.pieces = make([][]byte, this.pieceCount)
	this.sendRequestPiece()
	return nil
}

func (m *Meta) sendRequestPiece() {
	for i := 0; i < int(m.pieceCount); i++ {
		m.requestPiece(i)
	}
}

func (mw *Meta) requestPiece(i int) {
	buf := bytes.NewBuffer(nil)
	buf.WriteByte(extended)
	buf.WriteByte(byte(mw.utMetadata))
	buf.Write(bencode.Encode(map[string]interface{}{
		"msg_type": 0,
		"piece":    i,
	}))
	err := mw.WriteTo(buf.Bytes())
	if err != nil {
		fmt.Println("write err :", err.Error())
	}
}

func parseMetaData(metaData []byte, hashinfo string) {
	var insertname string
	insertlength := 0
	meta, err := bencode.Decode(bytes.NewBuffer(metaData))
	if err != nil {
		return
	}
	if name, ok := meta["name.utf-8"].(string); ok {
		log.Println("name", name)
		insertname = name
	} else if name, ok := meta["name"].(string); ok {
		log.Println("name", name)
		insertname = name
	}
	if length, ok := meta["length"].(int64); ok {
		log.Println("length", length)
		insertlength = int(length)
	}
	if files, ok := meta["files"].([]interface{}); ok {
		for _, file := range files {
			if f, ok := file.(map[string]interface{}); ok {
				log.Println(f)
			}
		}
	}
	log.Println("hashinfo:", hashinfo)
	insertBt(hashinfo, insertname, int64(insertlength))

}
