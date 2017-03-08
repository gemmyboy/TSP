The TransportationSynchronizationProtocols or TSP is being created so that regardless of the project
being created.  A Developer can use the TSP and start a generic server which will 
then synchronize data across all connected clients. Whether the client will accept
the data or use it is deemed irrelevent.

Core Implementation Features:
	-Synchronization(Sync):  Essentially consistently keep all available clients updated on the relevent data.
		-Sync Grouping: will simplify synchronization by allowing data to be grouped.  With this grouping
			in place, we can allow for better usage of synchronization. This will also allow for testing on
			certain groups when it comes to dealing with networking related issues.  An example of grouping
			might be positional data like: pos-x, pos-y, pos-z, rot-x, rot-y, rot-z, velocity, acceleration. etc..
				Algorithm: Will consist of event-based synchronization which will mean, synchronization will only occur
					when data actually changes or some new kind of data comes in.
			
				-NOTE: Should zoning be implemented: Each zone will have it's own set of groupings.
	-Admin Controls:  Admin controls allow a Developer or Admin to go in and activate/deactivate certain
		groups or zones.  Therefore able to dynamically enable/disable synchronization in real-time. This
		would be a web-interface dynamically generated via the webserver.  Here, we can also adjust settings
		specific to the server itself like server-load limiting, connected IP addresses, and the such.  This can include
		force-disconnecting certain clients.
		
		
Additional Implementation Features:
	-Regional Synchronization:  We can sync based on what zone a client is in. There are 2 ways to zone the data:
		-First way to zone data is based essentially on geographical data.  This is where boundaries are established and
			when a client enters that boundary, they are added onto that specific zone.
		-Second way to zone data is based on range of the client.  This can scale quickly in memory based on the number of
			clients in a specific zone. But the range is based on a set amount of distance from the individual client.

	-Remote Synchronization:  Should this need to be run over a distributed system, we will implement an intelligent 
		algorithm to handle auto-loading.  Meaning that either groupings or zones will be distributed over multiple systems.
		Eahc handling a different aspect of the synchronization.  
			-An example of this would be to have a server-stack containing 3 servers. 
				-Server A: Chat system
				-Server B: Physics system
				-Server C: User cache system
				-Server D: AI Physics system
