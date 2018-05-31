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
	receiveChan chan *Box         //Queue of Boxes client has received
	sendChan    chan *Box         //Queue of Boxes client needs to send
	closeFlag   bool              //Disconnect was called
	closeGroup  *sync.WaitGroup   //Wait for internal goroutines to exit
} //End Client

//---------------Basic Functionality-----------------------------

//NewClient -: Creates a new Client Connection
func NewClient(connectString string) *Client {
	c := new(Client)
	c.connStr = connectString
	c.gName = make(map[string]uint32)
	c.muxSend = &sync.Mutex{}
	c.muxReceive = &sync.Mutex{}
	c.receiveChan = make(chan *Box, 1000)
	c.sendChan = make(chan *Box, 1000)
	c.closeFlag = false
	c.closeGroup = &sync.WaitGroup{}
	c.closeGroup.Add(2)
	return c
} //End NewClient()

//Connect -: Connect to the SS
func (c *Client) Connect() {
	conn, err := net.Dial("tcp", c.connStr)
	if err != nil {
		log.Fatalln("Failed to connect to SS", err)
	}
	c.conn = conn

	go c.processReceive() //Goroutine for Receiving Data
	go c.processSend()    //Goroutine for Sending Data

	time.Sleep(time.Millisecond * 1000)
} //End Connect()

//Disconnect -: Close the connection
func (c *Client) Disconnect() {

	//Make function call Atomic for each Connect call
	if c.closeFlag {
		return
	}
	c.closeFlag = true
	c.conn.SetReadDeadline(time.Now())

	//Tell SS that it's time to die quietly
	c.SendBox(&Box{command: cDisconnect})

	//Wait until processReceive & processSend exit
	c.closeGroup.Wait()

	//Close the channels
	close(c.receiveChan)
	close(c.sendChan)

	//Destroy/Disconnect because this is never graceful
	err := c.conn.Close()
	Check(err)
} //End Disconnect()

//Ping -: Ping the SS to make sure it's still alive.
func (c *Client) Ping() {
	c.SendBox(&Box{command: cPing, data: []byte("0")})
} //End Ping()

//process -: continuously receive and process incoming data
func (c *Client) processReceive() {
	defer func() { c.closeGroup.Done() }()

	for !c.closeFlag {
		//Connection nulls out if hardware can't keep up
		b := &Box{}

		//Wait to receive data, once done, send confirmation bit, and generate box
		b = c.receive()

		//Error Checking Box
		if b == nil {
			b = &Box{command: cDisconnect}
		}

		switch b.command {
		case cList:
			{
				// buf := make([]byte, 8192)
				// n, err := c.conn.Read(buf)
				// Check(err)

				// b := BoxData(buf[:n])

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
		case cPing:
			{
				c.receiveChan <- b
			}
		default:
			{
				return /*Error State*/
			}
		} //End switch
		time.Sleep(time.Millisecond * 1)
	} //End for
} //End processReceive()

//processSend -: continuously process the queue of boxes to send.
func (c *Client) processSend() {
	defer func() { c.closeGroup.Done() }()
	for {
		select {
		case b := <-c.sendChan:
			{
				if b != nil {
					c.send(b)
				}
			} //End case
		default:
			{
				if c.closeFlag {
					return
				}
			} //End case
		} //End select
	} //End for
} //End processSend()

//Receive -: Receive data from SS | Easy Receive
func (c *Client) Receive() (int, []byte) {
	b := c.ReceiveBox()
	return int(b.source), b.data
}

//ReceiveBox -: DeQueue data box to process
func (c *Client) ReceiveBox() *Box {
	b := <-c.receiveChan
	if b == nil {
		return &Box{command: cDisconnect}
	}
	return b
} //End ReceiveBox()

//ReceiveChan -: Return the channel to process
func (c *Client) ReceiveChan() chan *Box {
	return c.receiveChan
} //End ReceiveChan()

//Send -: Send data to SS | Easy Send
func (c *Client) Send(name string, data []byte) {
	v, ok := c.gName[name]
	if !ok {
		log.Println("Group doesn't exist, dropping box")
		return
	}
	c.SendBox(&Box{command: cSend, destination: v, data: data})
} //End Send()

//SendBox -: Send data box to SS | Advanced Send
func (c *Client) SendBox(b *Box) {
	c.sendChan <- b
} //End SendBox()

//---------------Group Functionality-----------------------------

//GroupCreate -: Create a group on the SS
func (c *Client) GroupCreate(name, password, mpassword string, capacity int) {
	c.SendBox(&Box{command: cCreate, data: []byte(name + "," + password + "," + mpassword + "," + string(capacity))})
} //End GroupCreate()

//GroupDelete -: Delete a group on the SS
func (c *Client) GroupDelete(name string) {
	if v, ok := c.gName[name]; ok {
		delete(c.gName, name)
		c.SendBox(&Box{command: cDelete, destination: uint32(v)})
	}
} //End GroupDelete()

//GroupJoin -: Join an existing group on the SS
func (c *Client) GroupJoin(name, password string) {
	if v, ok := c.gName[name]; ok {
		c.SendBox(&Box{command: cJoin, destination: uint32(v), data: []byte(password)})
	}
} //End GroupJoin()

//GroupMasterJoin -: Join an existing group on the SS
func (c *Client) GroupMasterJoin(name, password, mpassword string) {
	if v, ok := c.gName[name]; ok {
		c.SendBox(&Box{command: cJoin, destination: uint32(v), data: []byte(password + "," + mpassword)})
	}
} //End GroupJoin()

//GroupLeave -: Leave an existing group on the SS
func (c *Client) GroupLeave(name string) {
	if v, ok := c.gName[name]; ok {
		c.SendBox(&Box{command: cLeave, destination: uint32(v)})
	}
} //End GroupLeave()

//GroupList -: Retrieve a list of existing groups on the SS
func (c *Client) GroupList() {
	c.SendBox(&Box{command: cList, data: nil})
} //End GroupList()

//GroupCheck -: Check if group exists yet
func (c *Client) GroupCheck(gpName string) bool {
	_, ok := c.gName[gpName]; return ok
} //End GroupCheck()

//GroupWaitCheck -: Pauses client until GroupCheck returns true but only retries some # of times.
func (c *Client) GroupWaitCheck(gpName string) bool {
	for i := 0; i < 20; i++ {
		if c.GroupCheck(gpName) {
			return true
		}
		time.Sleep(time.Millisecond * 100)
	}
	return false
} //End GroupWaitCheck() 

//---------------Helper Functionality-----------------------------

//send -: send Data over the connection
func (c *Client) send(b *Box) {
	ub := UnboxData(b)

	//Send box over connection
	c.muxSend.Lock()
	_, err := c.conn.Write(ub)
	c.muxSend.Unlock()
	if err != nil {
		//log.Println("Client-Box failed to send of size:", num, len(ub))
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
	if c.closeFlag {
		return nil
	} else if err == io.EOF {
		return nil
	} else if err != nil {
		c.closeFlag = true
		//log.Println("Client-Failed to receive:", err)
		return nil
	} else if num != 4 {
		//log.Println("Client-Didn't receive size:", num)
		c.closeFlag = true
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
