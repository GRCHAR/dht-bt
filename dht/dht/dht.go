package dht

import (
	"bytes"
	"encoding/binary"
	"log"
	"net"
	"strconv"
	"time"

	"github.com/ghr-dht/store"
	"github.com/ghr-dht/tool"
	"github.com/marksamman/bencode"
)

var seeds = []string{
	"router.utorrent.com:6881",
	"router.bittorrent.com:6881",
	"dht.transmissionbt.com:6881",
}

type FindNodeReq struct {
	Addr string
	Req  map[string]interface{}
}

type Response struct {
	Addr *net.UDPAddr
	T    string
	R    map[string]interface{}
}

type DHT struct {
	Host        string
	Conn        *net.UDPConn
	Id          string
	RequestList chan *FindNodeReq
	ReponseList chan *Response
	DataList    chan map[string]interface{}
}

func (dht *DHT) Start() error {
	addr, err := net.ResolveUDPAddr("udp", dht.Host)
	if err != nil {
		return err
	}
	dht.Conn, err = net.ListenUDP("udp", addr)
	if err != nil {
		return err
	}
	go dht.seedLoop()
	go dht.sendRequest()
	go dht.sendResponse()
	go dht.getResponse()
	go dht.handlerDataList()
	return nil
}

func (d *DHT) seedLoop() {

	d.addRequset()
	timer := time.NewTicker(15 * time.Second)
	for {
		select {
		case <-timer.C:
			if len(d.RequestList) == 0 {
				d.addRequset()
			}
		}
	}
}

func (dht *DHT) addRequset() {
	for _, seed := range seeds {
		req := new(FindNodeReq)
		req.Addr = seed
		req.Req = tool.MakeRequest("find_node", dht.Id, "")
		dht.RequestList <- req
	}

}

func (dht *DHT) sendRequest() {
	for {

		select {
		case req := <-dht.RequestList:
			udpAddr, err := net.ResolveUDPAddr("udp", req.Addr)
			if err != nil {

				continue
			}
			_, err = dht.Conn.WriteToUDP(bencode.Encode(req.Req), udpAddr)
			if err != nil {

				continue
			}

		}

	}
}

func (dht *DHT) sendResponse() {
	for {
		select {
		case res := <-dht.ReponseList:
			// log.Println("response")
			response := tool.MakeResponse(res.T, res.R)
			_, err := dht.Conn.WriteToUDP(bencode.Encode(response), res.Addr)
			if err != nil {

				continue
			}
		}
	}
}

func (dht *DHT) getResponse() {
	b := make([]byte, 8192)
	for {
		n, addr, err := dht.Conn.ReadFromUDP(b)

		if err != nil {

			continue
		}
		msg, err := bencode.Decode(bytes.NewBuffer(b[:n]))
		if err != nil {

			continue
		}
		msg["remote_addr"] = addr
		// fmt.Println("r:", msg["r"].(map[string]interface{})["nodes"].(string), "addr", addr)
		dht.DataList <- msg
	}
}

func (dht *DHT) ping(remote_addr *net.UDPAddr, t string) {
	res := new(Response)
	res.Addr = remote_addr
	res.R = map[string]interface{}{"id": dht.Id}
	res.T = t
	dht.ReponseList <- res
}

func (dht *DHT) findNode(remote_addr *net.UDPAddr, t string) {
	res := new(Response)
	res.Addr = remote_addr
	res.R = map[string]interface{}{"id": dht.Id, "node": ""}
	res.T = t
	dht.ReponseList <- res
}

func (dht *DHT) getPeer(remote_addr *net.UDPAddr, t string, arg map[string]interface{}) {
	hash_info, ok := arg["info_hash"]
	if !ok {
		log.Println("don't have info_hash")
		return
	}
	res := new(Response)
	res.Addr = remote_addr
	res.R = map[string]interface{}{"id": tool.NeighborId(dht.Id, hash_info.(string)), "token": tool.MakeToken(remote_addr.String()), "nodes": ""}
	res.T = t
	dht.ReponseList <- res
}

func (dht *DHT) announce_peers(remote_addr *net.UDPAddr, t string, arg map[string]interface{}) {
	hash_info, ok := arg["info_hash"]
	if !ok {
		log.Println("don't have info_hash")
		return
	}
	port, ok := arg["port"]
	if !ok {
		port = int64(remote_addr.Port)
	}
	if impliedPort, ok := arg["implied_port"].(int64); ok && impliedPort != 0 {
		port = int64(remote_addr.Port)
	}

	if port.(int64) <= 0 || port.(int64) >= 65535 {
		return
	}
	peer := &net.TCPAddr{IP: remote_addr.IP, Port: int(port.(int64))}
	hashpair := new(store.HashPair)
	hashpair.Hash = []byte(hash_info.(string))
	hashpair.Addr = peer.String()
	// log.Println(remote_addr, port, hash_info.(string))
	store.HashChan <- hashpair
}

func (dht *DHT) handlerDataList() {
	for {
		select {
		case msg := <-dht.DataList:
			y, ok := msg["y"].(string)
			if !ok {
				// log.Println("y don't find")
				continue
			}
			t, ok := msg["t"].(string)
			if !ok {
				// log.Println("t don't find")
				continue
			}
			remoteAddr, _ := msg["remote_addr"].(*net.UDPAddr)
			switch y {
			case "q":

				q, ok := msg["q"].(string)
				// log.Println(q)
				if !ok {
					// log.Println("q don't find")
					continue
				}
				if q == "ping" {
					dht.ping(remoteAddr, t)
				}
				if q == "find_node" {
					dht.findNode(remoteAddr, t)
				}
				if q == "get_peers" {
					dht.getPeer(remoteAddr, t, msg["a"].(map[string]interface{}))
				}
				if q == "announce_peer" {
					dht.announce_peers(remoteAddr, t, msg["a"].(map[string]interface{}))
				}
			case "r":
				r, ok := msg["r"].(map[string]interface{})
				if !ok {
					// log.Println("r don't find")
					continue
				}
				dht.decodeNodes(r)
			}
		}
	}
}

func (d *DHT) decodeNodes(r map[string]interface{}) {
	nodes, ok := r["nodes"].(string)
	if !ok {
		return
	}

	length := len(nodes)
	if length%26 != 0 {
		return
	}

	for i := 0; i < length; i += 26 {
		id := nodes[i : i+20]
		ip := net.IP(nodes[i+20 : i+24]).String()
		port := binary.BigEndian.Uint16([]byte(nodes[i+24 : i+26]))
		if port <= 0 || port >= 65535 {
			continue
		}
		addr := ip + ":" + strconv.Itoa(int(port))
		r := tool.MakeRequest("find_node", d.Id, id)
		req := &FindNodeReq{Addr: addr, Req: r}

		d.RequestList <- req
	}

	return
}
