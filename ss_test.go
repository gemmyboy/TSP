package tsp

import (
	"strconv"
	"sync"
	"testing"
	"time"
)

/*
	ss_test.go
		by Marcus Shannon

	This test file test all features for the Synchronous Server.
	To test fully: go test -v
	To test individually: go test -v -run <NameOfTest>
*/

func WaitChannel(c chan struct{}, n int) {
	for i := 0; i < n; i++ {
		select {
		case <-c:
			continue
		}
	}
}

func SendChannel(c chan struct{}, n int) {
	for i := 0; i < n; i++ {
		c <- struct{}{}
	}
}

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
	chan1 := make(chan struct{}, x)

	ss.SetCapacity(x)
	ss.Start()
	total := 0
	m := new(sync.Mutex)

	for i := 0; i < x; i++ {
		go func() {
			c := NewClient(address)
			c.Connect()
			c.Disconnect()

			m.Lock()
			total++
			m.Unlock()
			chan1 <- struct{}{}
		}()
		time.Sleep(time.Millisecond * 1)
	}

	//Wait until otherwise
	WaitChannel(chan1, x)
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
	if !c.GroupCheck("Test1") {
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
	if !c.GroupCheck("Test1") {
		t.Error("Group Failed to be created and/or listed.")
	}

	//Delete a Group
	c.GroupMasterJoin("Test1", "Password", "MPassword")
	c.GroupDelete("Test1")
	c.GroupList()

	//Check local group listing
	if c.GroupCheck("Test1") {
		t.Error("Group Failed to be deleted and/or listed.")
	}

	ss.Stop()
} //End TestGroupDelete()

//Test Sending Data from Client-to-Client using SS as medium.
func TestGroupSend(t *testing.T) {
	ss := NewSyncServer("localhost:4460")

	ss.Start()
	chanMaster := make(chan struct{})
	chanClient := make(chan struct{})

	//Kick off Master
	go func() {
		//Connect to SS
		c := NewClient("localhost:4460")
		c.Connect()

		//Create a Group
		c.GroupCreate("Test1", "Password", "MPassword", 20)
		c.GroupList()
		c.GroupCheck("Test1")

		//Join as Master
		c.GroupMasterJoin("Test1", "Password", "MPassword")

		b := c.ReceiveBox()
		if string(b.data) == "Hello There!" {
			SendChannel(chanMaster, 1)
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

		c.GroupCheck("Test1")

		//Join as Client
		c.GroupJoin("Test1", "Password")

		//Send Random Data
		c.Send("Test1", []byte("Hello There!"))

		b := c.ReceiveBox()
		if string(b.data) == "Hello Back!" {
			SendChannel(chanClient, 1)
		} else {
			t.Error("Received a Bad Box on Client", string(b.data))
		}

		c.Disconnect()
	}() //End Client

	//Wait until otherwise
	WaitChannel(chanMaster, 1)
	WaitChannel(chanClient, 1)

	ss.Stop()
} //End TestGroupSend()

//Test all membership functionality for Groups including Joining and Leaving
func TestGroupAll(t *testing.T) {
	ss := NewSyncServer("localhost:4451")
	ss.Start()

	//Connect to SS
	c := NewClient("localhost:4451")
	c.Connect()

	//Create a Group
	c.GroupCreate("Test1", "Password", "MPassword", 20)
	c.GroupList()
	if !c.GroupCheck("Test1") {
		t.Fatal("Failed to Create Group")
	}

	//Join as Master
	c.GroupMasterJoin("Test1", "Password", "MPassword")
	c.GroupLeave("Test1")
	c.GroupMasterJoin("Test1", "Password", "MPassword")
	c.GroupLeave("Test1")
	c.GroupMasterJoin("Test1", "Password", "MPassword")
	c.GroupLeave("Test1")
	c.GroupJoin("Test1", "Password")
	c.GroupLeave("Test1")

	c.GroupDelete("Test1")
	c.GroupList()
	if !c.GroupCheck("Test1") {
		t.Fatal("False Delete Group Succeeded")
	}

	c.GroupMasterJoin("Test1", "Password", "MPassword")
	c.GroupDelete("Test1")
	c.GroupList()
	if c.GroupCheck("Test1") {
		t.Fatal("Failed to Delete Group")
	}

	ss.Stop()
} //End TestGroupAll()

//Test Massive Amount of Clients connecting to SS
func xMassConnectPing(t *testing.T, num int, address string) {
	n := num
	chan1 := make(chan struct{}, n)
	chan2 := make(chan struct{}, n)
	chan3 := make(chan struct{}, n)
	ss := NewSyncServer(address)
	ss.Start()

	//Kick off Master
	go func() {
		//Connect to SS
		c := NewClient(address)
		c.Connect()

		SendChannel(chan2, n)
		WaitChannel(chan1, n)

		c.Disconnect()
	}() //End Master

	//Kick off N Client(s)
	for i := 0; i < n; i++ {
		go func(l int, chan1 chan struct{}, chan2 chan struct{}, chan3 chan struct{}, address string) {
			<-chan2

			//Connect to SS
			c := NewClient(address)
			c.Connect()
			c.Ping()
			_, _ = c.Receive()
			chan1 <- struct{}{}

			c.Disconnect()
			chan3 <- struct{}{}
		}(i, chan1, chan2, chan3, address) //End Client
	}
	WaitChannel(chan3, n)
	ss.Stop()
}

//Test 10 Connections using Ping
func Test10MassConnectPing(t *testing.T) {
	xMassConnectPing(t, 10, "localhost:4463")
}

//Test 50 Connections using Ping
func Test50MassConnectPing(t *testing.T) {
	xMassConnectPing(t, 50, "localhost:4464")
}

//Test 100 Connections using Ping
func Test100MassConnectPing(t *testing.T) {
	xMassConnectPing(t, 100, "localhost:4465")
}

/*
//Test 1000 Connections using Ping
func Test1000MassConnectPing(t *testing.T) {
	xMassConnectPing(t, 1000, "localhost:4466")
}

//Test 2000 Connections using Ping
func Test2000MassConnectPing(t *testing.T) {
	xMassConnectPing(t, 2000, "localhost:4467")
}
*/

//Test Master-Client(s) Grouping model by Counting to N
func TestCounting(t *testing.T) {
	n := 10
	chan1 := make(chan struct{}, n)
	chan2 := make(chan struct{}, n)
	chan3 := make(chan struct{}, n)
	boolMaster := 0
	boolClient := 0

	ss := NewSyncServer("localhost:4468")
	ss.Start()

	//Kick off Master
	go func() {
		//Connect to SS
		c := NewClient("localhost:4468")
		c.Connect()

		//Create a Group
		c.GroupCreate("Test1", "Password", "MPassword", 20)
		c.GroupList()
		c.GroupCheck("Test1")

		//Join as Master
		c.GroupMasterJoin("Test1", "Password", "MPassword")

		SendChannel(chan2, n)
		WaitChannel(chan1, n)

		for i := 0; i < n; i++ {
			c.Send("Test1", []byte(strconv.Itoa(i+1)))
		}

		count := 0
		for count < n {
			b := c.ReceiveBox()
			value, _ := strconv.Atoi(string(b.Data()))
			if value == n {
				count++
				boolMaster++
			}
		}

		c.Disconnect()
	}() //End Master

	//Kick off N Client(s)
	for i := 0; i < n; i++ {

		go func(l int) {
			<-chan2

			//Connect to SS
			c := NewClient("localhost:4468")
			c.Connect()
			c.GroupList()

			c.GroupCheck("Test1")

			//Join as Client
			c.GroupJoin("Test1", "Password")
			chan1 <- struct{}{}

			for {
				b := c.ReceiveBox()
				value, _ := strconv.Atoi(string(b.Data()))
				if value == n {
					boolClient++
					break
				}
				c.Send("Test1", []byte(strconv.Itoa(value+1)))
			}
			c.Disconnect()
			chan3 <- struct{}{}
		}(i) //End Client
	}

	//Wait until otherwise
	WaitChannel(chan3, n)

	c := NewClient("localhost:4468")
	c.Connect()
	c.GroupList()
	c.GroupMasterJoin("Test1", "Password", "MPassword")
	c.GroupDelete("Test1")
	c.GroupList()
	if c.GroupCheck("Test1") {
		t.Fatal("Failed to Delete Group")
	}

	ss.Stop()
}
