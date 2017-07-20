package tsp

import (
	"encoding/binary"
	"io"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

/*
	client.go
		by Marcus Shannon

	Essentially a wrapper interface to simplify connecting and sending data to and from the TSP SyncServer.
*/

//Client -: Data structure
type Client struct {
	conn        net.Conn          //Used to send/receive data
	connStr     string            //Where to connect too
	gName       map[string]uint32 //Map Name to group ID
	muxSend     *sync.Mutex       //Mutex to control Send
	muxReceive  *sync.Mutex       //Mutex to control Receive
	receiveChan chan *Box         //Queue of Boxes client needs to process
} //End Client

//---------------Basic Functionality-----------------------------

//NewClient -: Creates a new Client Connection
func NewClient(connectString string) *Client {
	c := new(Client)
	c.connStr = connectString
	c.gName = make(map[string]uint32)
	c.muxSend = &sync.Mutex{}
	c.muxReceive = &sync.Mutex{}
	return c
} //End NewClient()

//Connect -: Connect to the SS
func (c *Client) Connect() {
	conn, err := net.Dial("tcp", c.connStr)
	if err != nil {
		log.Fatalln("Failed to connect to SS", err)
	}
	c.conn = conn

	go c.process()
	time.Sleep(time.Millisecond * 500)

	c.GroupList()
} //End Connect()

//Disconnect -: Close the connection
func (c *Client) Disconnect() {
	//Tell SS we're disconnecting
	c.SendBox(&Box{command: cDisconnect})

	//Now disconnect
	c.muxReceive.Lock()
	err := c.conn.Close()
	Check(err)
	c.muxReceive.Unlock()
} //End Disconnect()

//process -: continuously receive and process incoming data
func (c *Client) process() {
	for {
		//Connection nulls out of hardware can't keep up
		b := &Box{}
		if c.conn == nil {
			b = &Box{command: cDisconnect}
		} else {
			//Wait to receive data, once done, send confirmation bit, and generate box
			b = c.receive()
		}

		switch b.command {
		case cList:
			{
				buf := make([]byte, 8192)
				n, err := c.conn.Read(buf)
				Check(err)

				b := BoxData(buf[:n])

				//Grab list
				str := string(b.data)
				strA := strings.Split(str, ";")
				strA = strA[:len(strA)-1] //Slice adjustment since Split() adds empty memory at the end.

				for _, v := range strA {
					strB := strings.Split(v, ",")
					t, _ := strconv.Atoi(strB[0])
					c.gName[strB[1]] = uint32(t) //Map Name to ID
				}
			}
		case cSend:
			{
				c.receiveChan <- b
			}
		case cSendInd:
			{
				c.receiveChan <- b
			}
		case cDisconnect:
			{
				c.Disconnect()
				return
			}
		} //End switch
		time.Sleep(time.Millisecond * 1)
	} //End for
} //End process()

//ReceiveBox -: DeQueue data box to process
func (c *Client) ReceiveBox() *Box {
	return <-c.receiveChan
} //End ReceiveBox()

//Send -: Send data to SS | Easy Send
func (c *Client) Send(name string, data []byte) {
	c.SendBox(&Box{command: cSend, destination: c.gName[name], data: data})
} //End Send()

//SendBox -: Send data box to SS | Advanced Send
func (c *Client) SendBox(b *Box) {
	c.send(b)
} //End SendBox()

//---------------Group Functionality-----------------------------

//GroupCreate -: Create a group on the SS
func (c *Client) GroupCreate(name, password, mpassword string, capacity int) {
	c.send(&Box{command: cCreate, data: []byte(name + "," + password + "," + mpassword + "," + string(capacity))})
	c.GroupList()
} //End GroupCreate()

//GroupJoin -: Join an existing group on the SS
func (c *Client) GroupJoin(name, password, mpassword string) {
	if v, ok := c.gName[name]; ok {
		c.send(&Box{command: cJoin, data: []byte(string(v) + "," + password + "," + mpassword)})
	} else {
		c.GroupList()
	}
} //End GroupJoin()

//GroupList -: Retrieve a list of existing groups on the SS
func (c *Client) GroupList() {
	c.send(&Box{command: cList, data: nil})
} //End GroupList()

//---------------Helper Functionality-----------------------------

//send -: send Data over the connection
func (c *Client) send(b *Box) {
	ub := UnboxData(b)

	//Send box over connection
	c.muxSend.Lock()
	num, err := c.conn.Write(ub)
	c.muxSend.Unlock()
	if err != nil {
		log.Println("Box failed to send of size:", num, len(ub))
		return
	}

	//log.Println("Wrote:", num, "- Size:", len(ub)) //DEBUG
	time.Sleep(time.Millisecond * 1)
	return
} //End send()

//receive -: receive data over the connection
func (c *Client) receive() *Box {
	var data []byte
	total := 4
	size := uint32(0)
	errCount := 0

	//To handle disconnect error
	c.muxReceive.Lock()

	//Grab the size after the first read to determine how much more data to read
	tbuf := make([]byte, 4)
	num, err := c.conn.Read(tbuf)
	if err == io.EOF {
		return nil
	} else if err != nil {
		log.Println("Failed to receive:", err)
		return nil
	} else if num != 4 {
		log.Println("Didn't receive size:", num)
		return nil
	}

	//Otherwise, strip out data and go
	size = binary.LittleEndian.Uint32(tbuf)
	data = make([]byte, int(size))

	//Strangely this is efficient technically
	data[0] = tbuf[0]
	data[1] = tbuf[1]
	data[2] = tbuf[2]
	data[3] = tbuf[3]

	for total != int(size) {
		//Read Data from Connection
		num, err := c.conn.Read(data[total:int(size)])

		//Error Checking incase it's temporary
		if err != nil {
			errCount++
		} else if errCount >= 3 {
			break
		} else {
			errCount = 0
		}

		//Accumulate current data size and buffered data
		total += num

		//log.Println("Total:", total, "- Size:", int(size), "- Buffer Size:", len(data)) //DEBUG
	}
	//log.Println("Total:", total, "- Size:", int(size), "- Buffer Size:", len(data)) //DEBUG

	c.muxReceive.Unlock()

	//Create Box to view data
	b := BoxData(data)

	return b
} //End receive()
