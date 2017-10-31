/*
    client.js
        by Marcus Shannon

    A JavaScript client side application for connecting and receiving data from the SS.
*/
var SS = {
    socket: null,
    Connect: function(ipaddress) {
        SS.socket = (new WebSocket("ws://" + ipaddress));
        SS.socket.onopen = function(event) {
            Log("Open")
        };
        
        SS.socket.onmessage = function(event) {
            b = SS.BoxData(event.data);
            Log("Message: " + String(b.data));
            


        };
        
        SS.socket.onclose = function(event) {
            Log("Close")
        };
    },
    BoxData: function(data){
        var enc = new TextEncoder("utf-8");
        d = enc.encode(data)

        tSize = new Uint8Array([d[3], d[2], d[1], d[0]]);
        tCommand = new Uint32Array([d[7], d[6], d[5], d[4]]);
        tDestination = new Uint32Array([d[11], d[10], d[9], d[8]]);
        tSource = new Uint32Array([d[15], d[14], d[13], d[12]]);

        b = Box();
        b.size = (new DataView(tSize.buffer)).getUint32();
        b.command = (new DataView(tCommand.buffer)).getUint32();
        b.destination = (new DataView(tDestination.buffer)).getUint32();
        b.source = (new DataView(tSource.buffer)).getUint32();
        b.data = data.slice(16);

        return b;
    },
    UnBoxData: function(box){}
};

//Box Class
function Box() {
    this.command = null;
    this.destination = null;
    this.source = null;
    this.data = null;
    this.size = null;
    return this;
};