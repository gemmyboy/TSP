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
            console.log(b);
            Log("Message: " + String(b.size) + " " + String(b.command) + " " + String(b.data));
        };
        
        SS.socket.onclose = function(event) {
            Log("Close")
        };
    },
    //BoxData -: Wrap up data into a box to make it usable
    BoxData: function(data){
        var enc = new TextEncoder("utf-8");
        d = enc.encode(data)

        tSize = new Uint8Array([d[3], d[2], d[1], d[0]]);
        tCommand = new Uint8Array([d[7], d[6], d[5], d[4]]);
        tDestination = new Uint8Array([d[11], d[10], d[9], d[8]]);
        tSource = new Uint8Array([d[15], d[14], d[13], d[12]]);

        b = new Box();
        b.size = (new DataView(tSize.buffer)).getUint32();
        b.command = (new DataView(tCommand.buffer)).getUint32();
        b.destination = (new DataView(tDestination.buffer)).getUint32();
        b.source = (new DataView(tSource.buffer)).getUint32();
        b.data = data.slice(16);

        return b;
    },
    //UnBoxData -: Break is all down into a byte array and send it off
    UnBoxData: function(box){
        var dec = new TextDecoder("utf-8");

        tSize = new Uint8Array(16 + box.data.length);
        tCommand = new Uint8Array(box.command);
        tDestination = new Uint8Array(box.destination);
        tSource = new Uint8Array(box.source);

        data = new Uint8Array([
            tSize[3], tSize[2], tSize[1], tSize[0],
            tCommand[3], tCommand[2], tCommand[1], tCommand[0],
            tDestination[3], tDestination[2], tDestination[1], tDestination[0],
            tSource[3], tSource[2], tSource[1], tSource[0]]);
        return data;
    }
};

//Box Class
function Box() {
    this.command = null;
    this.destination = null;
    this.source = null;
    this.data = null;
    this.size = null;
};