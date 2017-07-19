package tsp

import (
	"log"
	"net"
	"strconv"
	"strings"
)

/*
	client.go
		by Marcus Shannon

	Essentially a wrapper interface to simplify connecting and sending data to and from the TSP SyncServer.
*/

//Client -: Data structure
type Client struct {
	conn    net.Conn       //Used to send/receive data
	connStr string         //Where to connect too
	gName   map[string]int //Map Name to group ID
} //End Client

//---------------Basic Functionality-----------------------------

//NewClient -: Creates a new Client Connection
func NewClient(connectString string) *Client {
	c := new(Client)
	c.connStr = connectString
	c.gName = make(map[string]int)
	return c
} //End NewClient()

//Connect -: Connect to the SS
func (c *Client) Connect() {
	conn, err := net.Dial("tcp", c.connStr)
	if err != nil {
		log.Fatalln("Failed to connect to SS", err)
	}
	c.conn = conn
} //End Connect()

//Disconnect -: Close the connection
func (c *Client) Disconnect() {
	_ = c.conn.Close()
} //End Disconnect()

//---------------Group Functionality-----------------------------

//GroupCreate -: Create a group on the SS
func (c *Client) GroupCreate(name, password, mpassword string, capacity int) {
	c.conn.Write(UnboxData(&Box{command: cCreate, data: []byte(name + "," + password + "," + mpassword + "," + string(capacity))}))
} //End GroupCreate()

//GroupJoin -: Join an existing group on the SS
func (c *Client) GroupJoin(name, password, mpassword string) {
	if v, ok := c.gName[name]; ok {
		c.conn.Write(UnboxData(&Box{command: cJoin, data: []byte(string(v) + "," + password + "," + mpassword)}))
	} else {
		c.GroupList()
	}
} //End GroupJoin()

//GroupList -: Retrieve a list of existing groups on the SS
func (c *Client) GroupList() {
	c.conn.Write(UnboxData(&Box{command: cList, data: nil}))

	buf := make([]byte, 8192)
	n, err := c.conn.Read(buf)
	Check(err)

	b := BoxData(buf[:n])

	//Grab list
	str := string(b.data)
	strA := strings.Split(str, ";")

	for _, v := range strA {
		strB := strings.Split(v, ",")
		t, _ := strconv.Atoi(strB[0])
		c.gName[strB[1]] = t //Map Name to ID
	}
} //End GroupList()
