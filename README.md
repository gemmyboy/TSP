# TSP
The Transportation Synchronization Protocol (TSP) is used to simplify network routing in large scale systems where numerous 
applications need to transfer data to one another.

## Concept
TSP is a package used to instantiate a simple but efficient object called a Synchronous Server (SS). This is then used to automatically 
route and coordinate data between numerous different TCP-Connections connected to it.  

The concept of Group subscription is used to designate routing between the different clients.  Each 'Group' consists of a single 
Master-Client and numerous Standard-Client(s).  When a client connects via TCP to the SS, it may then query for a _List_ of groups 
currently existing on the SS and subscribe/join a group.  

When subscribing to a group as a _Master_-Client, all data coming from Standard-Client(s) will be funneled to the Master-Client. 
Vice-versa, a Master-Client may either broadcast data **or** send data individually to Standard-Client(s) subscribed to the group.

## Testing
Use: go test-v

## Code Usage
```go
//Instantiate
ss := tsp.NewSyncServer("localhost:4444")

//Set Maximum TCP Connection Limit -Optional
ss.SetCapacity(2000)  //Default is 1000

//Start listening and routing data
ss.Start()

//Stop the SS and discontinue all networking activities
ss.Stop()

```
## TODO
+ Create Client-Side API for both Master-Client(s) & Standard-Client(s)
+ Create Client-Side API documentation
+ More stress testing
+ More security options