package tsp

import (
	"sync"
	"testing"
	"time"
	//"log"
)

/*
	ss_test.go
		by Marcus Shannon

	This test file test all features for the Synchronous Server.
	To test fully: go test -v
	To test individually: go test -v -run <NameOfTest>
*/


//Test Creating SS
func TestCreate(t *testing.T) {
	ss := NewSyncServer("localhost:4444")
	if ss == nil {
		t.Error("Failed to create SS Entity")
	}
} //End TestCreate()

//Test Start & Stop
func TestStartStop(t *testing.T) {
	ss := NewSyncServer("localhost:4444")

	ss.Start()
	time.Sleep(time.Millisecond * 500)
	ss.Stop()
} //End TestStartStop()

//Test making a connection and disconnecting
func TestConnectDisconnect(t *testing.T) {
	ss := NewSyncServer("localhost:4444")

	ss.Start()
	wait := true

	go func() {
		c := NewClient("localhost:4444")
		c.Connect()
		c.Disconnect()

		wait = false
	}()

	//Wait until otherwise
	for wait {
		time.Sleep(time.Millisecond * 1)
	}

	ss.Stop()
} //End TestConnectDisconnect()

//Test making a Sending and Receiving Large Amounts of Data from single connection
func TestMassiveDataLoad(t *testing.T) {
	ss := NewSyncServer("localhost:4444")

	ss.Start()
	wait := true

	go func() {
		c := NewClient("localhost:4444")
		c.Connect()

		buf := make([]byte, 99999999)
		c.SendBox(&Box{command: cDisconnect, destination: 0, data: buf}) //1 Gigabyte
		c.Disconnect()

		wait = false
	}()

	//Wait until otherwise
	for wait {
		time.Sleep(time.Millisecond * 1)
	}

	time.Sleep(time.Millisecond * 500) //Wait on Writing
	ss.Stop()
} //End TestMassiveDataLoad()

//xTestConnections -: Used to replicate accepting an extremely large number of connections, send some data, and then disconnect
func xTestConnections(x int, address string, t *testing.T) {
	ss := NewSyncServer(address)

	ss.SetCapacity(x)
	ss.Start()
	total := 0
	m := new(sync.Mutex)

	for i := 0; i < x; i++ {
		go func() {
			c := NewClient(address)
			c.Connect()

			c.SendBox(&Box{command: cDisconnect, destination: 0, data: make([]byte, 1024)})
			//c.Disconnect()

			m.Lock()
			total++
			m.Unlock()

		}()
		time.Sleep(time.Millisecond * 1)
	}

	//Wait until otherwise
	for total < x {
		time.Sleep(time.Millisecond * 100)
	}
	ss.Stop()
} //xTestConnections()

//Test a 100 connections being made to the SS at one time.
func TestHundredConnections(t *testing.T) {
	xTestConnections(100, "localhost:4445", t)
} //End TestHundredConnections()

/*
//Test a 500 connections being made to the SS at one time.
func TestFiveHundredConnections(t *testing.T) {
	xTestConnections(500, "localhost:4446", t)
} //End TestFiveHundredConnections()


//Test a 1000 connections being made to the SS at one time.
func TestThousandConnections(t *testing.T) {
	xTestConnections(1000, "localhost:4445", t)
} //End TestThousandConnections()


//Test a 2000 connections being made to the SS at one time.
func TestTwoThousandConnections(t *testing.T) {
	xTestConnections(2000, "localhost:4446", t)
} //End TestTwoThousandConnections()
*/

//Test pinging the SS and having it Send data back
func TestPing(t *testing.T) {
	ss := NewSyncServer("localhost:4447")
	ss.Start()

	//Connect to SS
	c := NewClient("localhost:4447")
	c.Connect()

	//Send Ping Box
	c.Ping()

	//Wait and Receive the Ping
	b := c.ReceiveBox()

	if string(b.data) != "1" && b.command != cPing {
		t.Error("Bad Box returned from SS\n")
	}	

	c.Disconnect()
	ss.Stop()
} //End TestPing()

//Test Creating a Group on the SS and then get the List
func TestGroupCreate(t *testing.T) {
	ss := NewSyncServer("localhost:4449")
	ss.Start()

	//Connect to SS
	c := NewClient("localhost:4449")
	c.Connect()

	//Create a Group
	c.GroupCreate("Test1", "Password", "MPassword", 20)
	c.GroupList()

	//Check local group listing
	if !c.GroupWaitCheck("Test1") {
		t.Error("Group Failed to be created and/or listed.")
	}
	
	ss.Stop()
} //End TestGroupCreate()


//Test Deleting a Group from the SS
func TestGroupDelete(t *testing.T) {
	ss := NewSyncServer("localhost:4450")
	ss.Start()

	//Connect to SS
	c := NewClient("localhost:4450")
	c.Connect()

	//Create a Group
	c.GroupCreate("Test1", "Password", "MPassword", 20)
	c.GroupList()

	//Check local group listing
	if !c.GroupWaitCheck("Test1") {
		t.Error("Group Failed to be created and/or listed.")
	}
	
	//Delete a Group
	c.GroupDelete("Test1")
	c.GroupList()

	//Check local group listing
	if c.GroupCheck("Test1") {
		t.Error("Group Failed to be deleted and/or listed.")
	}

	ss.Stop()
} //End TestGroupDelete()


//Test Sending Data from Client-to-Client using SS as medium.
func TestGroupSendIndividual(t *testing.T) {
	ss := NewSyncServer("localhost:4460")

	ss.Start()
	boolMaster := false
	boolClient := false

	//Kick off Master
	go func() {
		//Connect to SS
		c := NewClient("localhost:4460")
		c.Connect()

		//Create a Group
		c.GroupCreate("Test1", "Password", "MPassword", 20)
		c.GroupList()
		c.GroupWaitCheck("Test1")

		//Join as Master
		c.GroupMasterJoin("Test1", "Password", "MPassword")

		b := c.ReceiveBox()
		if string(b.data) == "Hello There!" {
			boolMaster = true
		} else {
			t.Error("Received a Bad Box on Master", string(b.data))
		}

		//Functionality we're actually testing
		c.Send("Test1", []byte("Hello Back!"))

		c.Disconnect()
	}() //End Master
	time.Sleep(time.Millisecond * 1000)

	//Kick off Client
	go func() {
		//Connect to SS
		c := NewClient("localhost:4460")
		c.Connect()
		c.GroupList()

		c.GroupWaitCheck("Test1")

		//Join as Client
		c.GroupJoin("Test1", "Password")

		//Send Random Data
		c.Send("Test1", []byte("Hello There!"))

		b := c.ReceiveBox()
		if string(b.data) == "Hello Back!" {
			boolClient = true
		} else {
			t.Error("Received a Bad Box on Client", string(b.data))
		}

		c.Disconnect()
	}() //End Client

	//Wait until otherwise
	for !boolMaster || !boolClient {
		time.Sleep(time.Millisecond * 1000)
	}

	ss.Stop()
} //End TestGroupSendIndividual()

/*
//Test all membership functionality for Groups including Joining and Leaving
func TestGroupAll(t *testing.T) {
	ss := NewSyncServer("localhost:4451")
	ss.Start()

	//Connect to SS
	conn, err := net.Dial("tcp", "localhost:4451")
	if err != nil {
		t.Error("Failed to connect to SS", err)
	}

	//Create a Group
	ss.send(&Box{
		command:     cCreate,
		destination: uint32(0),
		data:        []byte("Test1,Password,MPassword,20")}, conn)

	//Request List of Groups
	ss.send(&Box{
		command:     cList,
		destination: uint32(0),
		data:        nil}, conn)

	//Receive actual List
	b := ss.receive(conn)
	if b == nil {
		t.Error("Failed to receive Box from SS")
	}
	if b.command != cList {
		t.Error("Bad Box was received", b.command)
	}

	data := string(b.data)
	if data != "2,Test1;" {
		t.Error("Bad data was returned in List Box: ", data)
	}

	//Join a Group
	ss.send(&Box{
		command:     cJoin,
		destination: uint32(2),
		data:        []byte("Password")}, conn)

	//Leave a Group
	ss.send(&Box{
		command:     cLeave,
		destination: uint32(2)}, conn)

	//Delete a Group
	ss.send(&Box{
		command:     cDelete,
		destination: uint32(2)}, conn)

	ss.Stop()
} //End TestGroupAll()

//Test Sending Data back and forth between a Master Client and Regular Client
func TestPingPong(t *testing.T) {
	ss := NewSyncServer("localhost:4452")

	ss.Start()
	boolMaster := false
	boolClient := false

	//Kick off Master
	go func() {
		//Connect to SS
		connM, err := net.Dial("tcp", "localhost:4452")
		if err != nil {
			t.Error("Failed to connect to SS", err)
		}

		//Create a Group
		ss.send(&Box{
			command:     cCreate,
			destination: uint32(0),
			data:        []byte("Test1,Password,MPassword,20")}, connM)

		//Join as Master
		ss.send(&Box{
			command:     cJoin,
			destination: uint32(2),
			data:        []byte("Password,MPassword")}, connM)

		b := ss.receive(connM)
		if string(b.data) == "Hello There!" {
			boolMaster = true
		} else {
			t.Error("Received a Bad Box on Master", string(b.data))
		}

		//Send Random Data
		ss.send(&Box{
			command:     cSend,
			destination: uint32(2),
			data:        []byte("Hello Back!")}, connM)
	}() //End Master
	time.Sleep(time.Millisecond * 200)

	//Kick off Client
	go func() {
		//Connect to SS
		connC, err := net.Dial("tcp", "localhost:4452")
		if err != nil {
			t.Error("Failed to connect to SS", err)
		}

		//Join as Client
		ss.send(&Box{
			command:     cJoin,
			destination: uint32(2),
			data:        []byte("Password")}, connC)

		//Send Random Data
		ss.send(&Box{
			command:     cSend,
			destination: uint32(2),
			data:        []byte("Hello There!")}, connC)

		b := ss.receive(connC)
		if string(b.data) == "Hello Back!" {
			boolClient = true
		} else {
			t.Error("Received a Bad Box on Client", string(b.data))
		}
	}() //End Client

	//Wait until otherwise
	for !boolMaster || !boolClient {
		time.Sleep(time.Millisecond * 500)
	}

	ss.Stop()
} //End TestPingPong

//Test X Client connections and 1 Master Connection.
//	All X Clients will connect and send Master data.
//	Once Master received from all X, it will broadcast
//	one data to all Clients. Then everyone will disconnect.
func xTestPingPong(x int, address string, t *testing.T) {
	ss := NewSyncServer(address)

	ss.SetCapacity(x + 1)
	ss.Start()
	boolMaster := 0
	boolClient := 0
	muxClient := new(sync.Mutex)

	//Kick off Master
	go func() {
		//Connect to SS
		connM, err := net.Dial("tcp", address)
		if err != nil {
			log.Println("Failed to connect to SS", err)
			return
		}

		//Create a Group
		ss.send(&Box{
			command:     cCreate,
			destination: uint32(0),
			source:      uint32(0),
			data:        []byte("Test1,Password,MPassword,20")}, connM)

		//Join as Master
		ss.send(&Box{
			command:     cJoin,
			destination: uint32(2),
			source:      uint32(0),
			data:        []byte("Password,MPassword")}, connM)

		for boolMaster < x {
			time.Sleep(time.Millisecond * 1)
			b := ss.receive(connM)
			if string(b.data) == "Hello There!" {
				boolMaster++
			} else {
				log.Println("Received a Bad Box on Master", string(b.data))
				return
			}
		}

		//Send Random Data
		ss.send(&Box{
			command:     cSend,
			destination: uint32(2),
			source:      uint32(0),
			data:        []byte("Hello Back!")}, connM)
		connM.Close()
	}() //End Master
	time.Sleep(time.Millisecond * 200)

	//Kick off all the Clients
	for i := 0; i < x; i++ {
		go func() {
			me := i
			//Connect to SS
			connC, err := net.Dial("tcp", address)
			if err != nil {
				//log.Println("Failed to connect to SS", err)
				return
			}

			//log.Println(me, " joined room")

			//Join as Client
			ss.send(&Box{
				command:     cJoin,
				destination: uint32(2),
				source:      uint32(me),
				data:        []byte("Password")}, connC)

			//Send Random Data
			ss.send(&Box{
				command:     cSend,
				destination: uint32(2),
				source:      uint32(me),
				data:        []byte("Hello There!")}, connC)

			b := ss.receive(connC)
			if string(b.data) == "Hello Back!" {
				muxClient.Lock()
				boolClient++
				muxClient.Unlock()
			} else {
				log.Println("Received a Bad Box on Client", string(b.data))
			}
			//log.Println(me, " received data and closing")
			time.Sleep(time.Millisecond * 10)
			connC.Close()
		}() //End Client
		time.Sleep(time.Millisecond * 1)
	} //End for

	//Wait until otherwise
	for boolMaster != x || boolClient != x {
		time.Sleep(time.Millisecond * 250)
		//log.Println(boolMaster, "-", boolClient)
	}

	ss.Stop()
} //End xTestPingPong()

//Test PingPong with 10 Clients
func TestTenPingPong(t *testing.T) {
	xTestPingPong(10, "localhost:4453", t)
} //End TestTenPingPong()

//Test PingPong with 25 Clients
func TestTwentyFivePingPong(t *testing.T) {
	xTestPingPong(25, "localhost:4454", t)
} //End TestTwentyFivePingPong()

//Test PingPong with 100 Clients
func TestOneHundredPingPong(t *testing.T) {
	xTestPingPong(100, "localhost:4455", t)
} //End TestOneHundredPingPong()

//Test PingPong with 1000 Clients
func TestOneThousandPingPong(t *testing.T) {
	xTestPingPong(1000, "localhost:4456", t)
} //End TestOneThousandPingPong()

//Test PingPong with 2000 Clients
func TestTwoThousandPingPong(t *testing.T) {
	xTestPingPong(2000, "localhost:4457", t)
} //End TestTwoThousandPingPong()

//Test PingPong with 5000 Clients
func TestFiveThousandPingPong(t *testing.T) {
	xTestPingPong(5000, "localhost:4458", t)
} //End TestFiveThousandPingPong()

//Test PingPong with 10000 Clients
// --NOTE: Kind of unstable after 10000
// --Somewhat unstable even at 10k.
//	but has more to do with machine than software.
// func TestTenThousandPingPong(t *testing.T) {
// 	xTestPingPong(10000, "localhost:4459", t)
// } //End TestTenThousandPingPong()

*/
