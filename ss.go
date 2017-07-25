package tsp

import (
	"bytes"
	"encoding/binary"
	"io"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/streamrail/concurrent-map"
)

/*
	ss.go
		by Marcus Shannon

   	Simplifying the first version of SS into some simple intelligent routing software.
	All data is encapsulated into an entity called Box with the following format:
		uint32 - Size
		uint32 - Command
		uint32 - Destination (group-id)
		uint32 - Source (connection-id)
		[]byte - data

   	Core Features:
   	- x Route data from Client to Master-Client and Master-Client to Client(s)
   	- x Connect to SS
   	- x Disconnect to SS
   	- x Connection Capacity for SS

   	Speciality Feature - Group Subscription:
   	- Creates theoretical "groups" in the data which connections subscribe too to receive data from.
   	- x Create Group - Box Data Format -: <name>,<password>,<masterpassword>,<capacity>
   	- x Delete Group - Box Data Format -: <group-id>
   	- x Join Group - Box Data Format -: <group-id>,<password>,<opt-masterpassword>
   	- x Leave Group - Box Data Format -: <group-id>
	- x Send Group - Box Data Format -: <anything>
	- x Send Group Individual - Box Data Format -: <anything>
		-NOTE: destination references group being sent too; source references specific client.
		- this works because only a master client can/will use this.
	- x List Groups in SS when queried
   	- x Set Group Capacity

   	Comes with a complete Test Suite to fully test out functionality! <3
	Use at command line: go test -v
*/

//SyncServer -: Struct for containing all Data specific to the SyncServer
type SyncServer struct {
	address  string           //IP Address : Port to where this is hosted or URL
	capacity uint64           //Max number of group connections this can handle
	current  uint64           //Current number of connections con to SS
	ider     uint64           //Used to create a unique id's for groups and connections
	listener *net.TCPListener //Listener used to accept connections

	isRunning      bool               //Flag for if SS is running
	connections    cmap.ConcurrentMap //Concurrent map used to handle connections
	groups         cmap.ConcurrentMap //Concurrent map used to handle groups
	groupsNameToID cmap.ConcurrentMap //Concurrent map used to convert group name to ID
	muxSend        *sync.Mutex        //Mutex for Sending Data
	muxIder        *sync.Mutex        //Mutex for Ider var
	muxCurrent     *sync.Mutex        //Mutex for Current var
} //End SyncServer struct

//Group -: Struct for containing all Data specific to the Group
type Group struct {
	id             string             //Identifier for Group
	name           string             //name can't be any longer than 32 bytes or 32 chars
	capacity       uint64             //Max number of group connections this can handle (default 1000)
	current        uint64             //Current size of group
	password       string             //Password used to subscribe to a group
	passwordMaster string             //Password to make a client Master of group
	membership     cmap.ConcurrentMap //Map containing membership of group [id]struct{}{}
	masterid       string             //Id to Connection which is master of this group

	muxJoin  *sync.Mutex //Mutex for Joining Group
	muxLeave *sync.Mutex //Mutex for Leaving Group
} //End Group struct

//Box -: Struct to used to split and hold incoming & outgoing data
type Box struct {
	size        uint32 //Size of box when last unpackaged
	command     uint32 //Action Box wants to take
	destination uint32 //Group-ID this Box is heading towards
	source      uint32 //Connection-ID of connection this came from
	data        []byte //Data Box is transporting
} //End Box struct

//Data -: Return the data from the Box
func (b *Box) Data() []byte { return b.data }

//List of Commands
const (
	cDisconnect uint32 = 0 //Client is disconnecting
	cPing       uint32 = 1 //Check to see if SS is alive and receiving

	cList    uint32 = 2 //List out groups on SS and send back to connection
	cCreate  uint32 = 3 //Create a group
	cDelete  uint32 = 4 //Delete a group
	cJoin    uint32 = 5 //Join a group
	cLeave   uint32 = 6 //Leave a group
	cSend    uint32 = 7 //Send data to a group
	cSendInd uint32 = 8 //Send data to an individual in a group
)

//NewSyncServer -: Creates new SyncServer
func NewSyncServer(address string) *SyncServer {
	ss := new(SyncServer)
	ss.address = address
	ss.capacity = uint64(1000) //Default
	ss.ider = uint64(1)
	ss.isRunning = false
	ss.connections = cmap.New()
	ss.groups = cmap.New()
	ss.groupsNameToID = cmap.New()
	ss.muxSend = new(sync.Mutex)
	ss.muxIder = new(sync.Mutex)
	ss.muxCurrent = new(sync.Mutex)
	return ss
} //End NewSyncServer()

//Start -: Starts SS and generating hosting connection
func (ss *SyncServer) Start() {
	ss.isRunning = true
	go ss.processConnections()
	time.Sleep(time.Millisecond * 500)
} //End Start()

//Stop -: Stops SS and kills program
func (ss *SyncServer) Stop() {
	ss.isRunning = false
	time.Sleep(time.Millisecond * 500)
} //End Stop()

//SetCapacity -: Sets Capacity of the SS
func (ss *SyncServer) SetCapacity(x int) {
	ss.capacity = uint64(x)
} //End SetCapacity()

//IsRunning -: Whether the SS is running or not
func (ss *SyncServer) IsRunning() bool {
	return ss.isRunning
} //End IsRunning()

//processConnections -: Processes connections and forks go-routines for each connection
func (ss *SyncServer) processConnections() {
	tListener, err := net.Listen("tcp", ss.address)
	CheckKill(err)
	ss.listener = tListener.(*net.TCPListener)

	//Main Loop for accepting connections
	for ss.isRunning {
		ss.listener.SetDeadline(time.Now().Add(time.Millisecond * 500)) //Every 0.5 seconds
		con, err := ss.listener.Accept()
		if err != nil || con == nil {
			continue
		} else if ss.capacity == ss.current {
			con.Close()
		} else if err == nil {
			//Create an ID for connection and fork
			id := strconv.Itoa(int(ss.ider))
			ss.connections.Set(id, con)
			go ss.process(con, id)

			//Increment ider & current
			ss.muxIder.Lock()
			ss.ider++
			ss.muxIder.Unlock()
			ss.muxCurrent.Lock()
			ss.current++
			ss.muxCurrent.Unlock()
		}
	} //End for

	ss.listener.Close()
} //End processConnections()

//BoxData -: generates usable Box from received data
func BoxData(data []byte) *Box {
	if len(data) >= 16 { //Box requires a minimum of command and destination
		b := new(Box)

		b.command = binary.LittleEndian.Uint32(data[4:8])
		b.destination = binary.LittleEndian.Uint32(data[8:12])
		b.source = binary.LittleEndian.Uint32(data[12:16])
		b.data = data[16:]
		b.size = uint32(16 + len(b.data))
		return b
	}
	return nil
} //End box()

//UnboxData -: reduces usable Box into a byte array for sending
func UnboxData(b *Box) []byte {
	buf := bytes.NewBuffer(nil)

	tb0 := make([]byte, 4)
	tb1 := make([]byte, 4)
	tb2 := make([]byte, 4)
	tb3 := make([]byte, 4)

	binary.LittleEndian.PutUint32(tb0, uint32(16+len(b.data)))
	binary.LittleEndian.PutUint32(tb1, b.command)
	binary.LittleEndian.PutUint32(tb2, b.destination)
	binary.LittleEndian.PutUint32(tb3, b.source)

	buf.Write(tb0)
	buf.Write(tb1)
	buf.Write(tb2)
	buf.Write(tb3)
	buf.Write(b.data)

	return buf.Bytes()
} //End unbox()

//process -: performs general routing logic for given connection
func (ss *SyncServer) process(conn net.Conn, id string) {
	defer conn.Close()

	for ss.isRunning {

		//Connection nulls out of hardware can't keep up
		b := &Box{}
		if conn == nil {
			b = &Box{command: cDisconnect}
		} else {
			//Wait to receive data, once done, send confirmation bit, and generate box
			b = ss.receive(conn)
		}

		//log.Println(string(b.data)) //DEBUG

		//Error Checking Box
		if b == nil {
			b = &Box{command: cDisconnect}
		}

		//Switch Over Command Options
		switch b.command {
		case cSend: //Send Data to a Group
			ss.groupSend(b, id)
			break
		case cDisconnect: //Proper Disconnect
			ss.disconnect(conn, id)
			return
		case cPing: //Ping the SS
			ss.send(&Box{command: cPing, destination: uint32(0), data: []byte("1")}, conn)
			break
		case cList: //List out Groups
			ss.groupList(conn)
			break
		case cCreate: //Create Group
			ss.groupCreate(b)
			break
		case cDelete: //Delete Group
			ss.groupDelete(b)
			break
		case cJoin: //Join Group
			ss.groupJoin(b, id)
			break
		case cLeave: //Leave Group
			ss.groupLeave(b, id)
			break
		case cSendInd: //Send Data to individual in a Group
			ss.groupSendIndividual(b, id)
			break
		default: //Improper Disconnect
			ss.disconnect(conn, id)
			return
		}
		time.Sleep(time.Millisecond * 1)
	} //End for
} //End process()

//---------------------------------------------------
//-- Group Functions

//disconnect -: clear a connection out of the SS
func (ss *SyncServer) disconnect(conn net.Conn, id string) {
	for _, id := range ss.groups.Keys() {
		tg, _ := ss.groups.Get(id)
		g := tg.(*Group)

		//Leave the Group
		g.muxLeave.Lock()
		if g.membership.Has(id) {
			g.membership.Remove(id)
			g.current--
		} else if g.masterid == id {
			g.masterid = ""
		}
		g.muxLeave.Unlock()
	}
	ss.connections.Remove(id)

	//Inform SS they are leaving
	ss.muxCurrent.Lock()
	ss.current--
	ss.muxCurrent.Unlock()
} //End disconnect()

//groupSend -: if current connection is master, sends to whole group,
//	otherwise only sends to master.
func (ss *SyncServer) groupSend(b *Box, id string) {

	num := strconv.Itoa(int(b.destination))
	tSource, _ := strconv.Atoi(id)
	b.source = uint32(tSource)

	tGroup, e := ss.groups.Get(num)
	if !e {
		log.Println("Dropped Box:", b.destination, b.command)
		return
	}
	group := tGroup.(*Group)

	//Check Master
	if group.masterid == id { //Send to all members
		for _, k := range group.membership.Keys() {
			tConn, _ := ss.connections.Get(k)
			conn := tConn.(net.Conn)
			ss.send(b, conn)
		} //End for
	} else if group.masterid != "" { //Send to master
		tConn, _ := ss.connections.Get(group.masterid)
		conn := tConn.(net.Conn)
		ss.send(b, conn)
	}
} //End groupSend()

//groupList -: Send a list of existing
func (ss *SyncServer) groupList(conn net.Conn) {
	data := bytes.NewBufferString("")

	//Generate the List of Groups currently on SS
	for _, id := range ss.groups.Keys() {
		v, _ := ss.groups.Get(id)
		data.WriteString(id + "," + v.(*Group).name + ";")
	} //End for

	ss.send(&Box{command: cList, destination: uint32(0), data: data.Bytes()}, conn)
} //End groupList()

//groupCreate -: Create a group on SS
func (ss *SyncServer) groupCreate(b *Box) {
	//Data must contain:
	//	<name>,<password>,<masterpassword>,<capacity>
	data := strings.Split(string(b.data), ",")

	//Parameter Check
	if len(data) < 4 {
		return
	}

	//Parse Capacity for a room
	cap, err := strconv.ParseUint(data[3], 10, 64)
	if err != nil {
		cap = 10
	}

	//Parse and convert ID into something usable
	id := strconv.FormatUint(ss.ider, 10)
	ss.muxIder.Lock()
	ss.ider++
	ss.muxIder.Unlock()

	//Checking if room already exists
	tID, tB := ss.groupsNameToID.Get(data[0])
	if tB {
		if ss.groups.Has(tID.(string)) {
			log.Println("Group: " + data[0] + " already exists!")
			return
		}
	}

	//Create name to ID mapping
	ss.groupsNameToID.Set(data[0], id)

	//Create Room
	ss.groups.Set(
		id,
		&Group{
			id:             id,
			name:           data[0],
			password:       data[1],
			passwordMaster: data[2],
			membership:     cmap.New(),
			capacity:       cap,
			muxJoin:        new(sync.Mutex),
			muxLeave:       new(sync.Mutex)})
} //End groupCreate()

//groupDelete -: Delete a group on SS
func (ss *SyncServer) groupDelete(b *Box) {
	//Data must contain:
	//	<group-id>
	data := string(b.data)

	//Nil check
	if data == "" {
		log.Println("Bad data returned")
		return
	}

	//Check group Exists
	if ss.groups.Has(data) {
		ss.groups.Remove(data)
	}
} //End groupDelete()

//groupJoin -: Join a group on SS
func (ss *SyncServer) groupJoin(b *Box, id string) {
	//Data must contain:
	//	<group-id>,<password>,<opt-masterpassword>
	//	if masterpassword exists here, then trying to join as Master client
	data := strings.Split(string(b.data), ",")
	if len(data) < 2 {
		return
	} else if len(data) == 2 { //Join as regular member
		tg, e := ss.groups.Get(data[0])

		//Group doesn't exist
		if !e {
			return
		}
		g := tg.(*Group)

		//Password & Capacity check
		if g.password != data[1] && g.current >= g.capacity {
			log.Println("Bad Password or over Capacity", data[1], g.current)
			return
		}

		//Actually Join Group
		g.muxJoin.Lock()
		g.current++
		g.membership.Set(id, struct{}{})
		g.muxJoin.Unlock()
	} else if len(data) == 3 { //Join as a master
		tg, e := ss.groups.Get(data[0])

		//Group doesn't exist
		if !e {
			log.Println("Group doesn't exist!")
			return
		}
		g := tg.(*Group)

		//Password & Master-Password check
		if g.password != data[1] || g.passwordMaster != data[2] {
			log.Println("Bad Password(s)", data[1], data[2])
			return
		}

		//Set Master Client
		g.muxJoin.Lock()
		g.masterid = id
		g.muxJoin.Unlock()
	}
} //End groupJoin()

//groupLeave -: Leave a group on SS
func (ss *SyncServer) groupLeave(b *Box, id string) {
	//Data must contain:
	//	<group-id>
	data := string(b.data)

	//Check nil case
	if data == "" {
		return
	}

	tg, e := ss.groups.Get(data)

	//Group doesn't exist
	if !e {
		log.Println("Group doesn't exist!")
		return
	}
	g := tg.(*Group)

	g.muxLeave.Lock()
	//Check if this is Master
	if g.masterid == id {
		g.masterid = ""
	} else if g.membership.Has(id) { //Check if member
		g.membership.Remove(id)
		g.current--
	}
	g.muxLeave.Unlock()
} //End groupLeave()

//groupSendIndividual -: Send data to an individual in group
//	--Destination is still group-id
//	--Specific to this protocol, source will be specific conn id.
func (ss *SyncServer) groupSendIndividual(b *Box, id string) {
	//Pull Group Data
	groupID := strconv.Itoa(int(b.destination))
	tg, ok1 := ss.groups.Get(groupID)
	if !ok1 {
		log.Println("Group not found: ", groupID)
		return
	}
	g := tg.(*Group)

	//Pull Connection data
	connectionID := strconv.Itoa(int(b.source))
	if !g.membership.Has(connectionID) {
		log.Println("Connection", connectionID, "not found as a member of group", groupID)
		return
	}
	tConn, _ := ss.connections.Get(connectionID)
	conn := tConn.(net.Conn)

	//Group found, membership confirmed, connection found, now send.
	ss.send(b, conn)
} //End groupSendIndividual()

//---------------------------------------------------
//-- Helper Functions

//send -: Sends data and waits for confirmation
func (ss *SyncServer) send(b *Box, conn net.Conn) {
	ub := UnboxData(b)

	//Send box over connection
	ss.muxSend.Lock()
	num, err := conn.Write(ub)
	ss.muxSend.Unlock()
	if err != nil {
		log.Println("Box failed to send of size:", num, len(ub))
		return
	}

	//log.Println("Wrote:", num, "- Size:", len(ub)) //DEBUG
	time.Sleep(time.Millisecond * 1)
	return
} //End send()

//receive -: Receives data, sends confirmation byte, and then returns box
func (ss *SyncServer) receive(conn net.Conn) *Box {
	var data []byte
	total := 4
	size := uint32(0)
	errCount := 0

	//Grab the size after the first read to determine how much more data to read
	tbuf := make([]byte, 4)
	num, err := conn.Read(tbuf)
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
		num, err := conn.Read(data[total:int(size)])

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

	//Create Box to view data
	b := BoxData(data)

	time.Sleep(time.Millisecond * 1)
	return b
} //End receive()

//Check -: just checks for error, outputs error but does not kill
func Check(err error) bool {
	if err != nil {
		log.Println(err)
		return false
	}
	return true
} //End Check()

//CheckKill -: Checks Error, if there is one, it logs it and kills program
func CheckKill(err error) {
	if err != nil {
		log.Fatalln(err)
	} //end if
} //End CheckKill()
