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
            Log("Sending Ping")
            
            b = new Box();
            b.command = 1;
            b.data = "You can do the thing!";
            SS.socket.send(SS.UnBoxData(b));
        };
        
        SS.socket.onmessage = function(event) {
            b = SS.BoxData(event.data);

            //DEBUG
            console.log(b);
            Log("Message: " + String(b.size) + " " + String(b.command) + " " + String(b.data));

            b.command = 0;
            b.data = "Disconnect";
            SS.socket.send(SS.UnBoxData(b));
            //DEBUG
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
        var dec = new TextEncoder("utf-8");
        console.log(box);

        tSize = UInt32toUInt8(16 + box.data.length);
        tCommand = UInt32toUInt8(box.command);
        tDestination = UInt32toUInt8(box.destination);
        tSource = UInt32toUInt8(box.source);
        tData = dec.encode(box.data)

        data = new Uint8Array(16 + tData.length);
        data[0] = tSize[3];
        data[1] = tSize[2];
        data[2] = tSize[1];
        data[3] = tSize[0];
        data[4] = tCommand[3];
        data[5] = tCommand[2];
        data[6] = tCommand[1];
        data[7] = tCommand[0];
        data[8] = tDestination[3];
        data[9] = tDestination[2];
        data[10] = tDestination[1];
        data[11] = tDestination[0];
        data[12] = tSource[3];
        data[13] = tSource[2];
        data[14] = tSource[1];
        data[15] = tSource[0];
        for(i = 0; i < tData.length; i++){
            data[16 + i] = tData[i];
        }

        console.log(data);
        return data;
    }
};

//Box Class
function Box() {
    this.command = 0;
    this.destination = 0;
    this.source = 0;
    this.data = "";
    this.size = 0;
};

//UInt8 casting function
function UInt32toUInt8(value) {
    return [
        value >> 24,
        (value >> 24) << 8,
        (value >> 24) << 16,
        (value << 24) >> 24
    ];
};
