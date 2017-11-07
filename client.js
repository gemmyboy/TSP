/*
    client.js
        by Marcus Shannon

    A JavaScript client side application for connecting and receiving data from the SS.
*/
var SS = {
    socket: null,
    groups: [],
    receiveboxqueue: [],
    //Connect -: create a connection to the SS at ipaddress
    Connect: function(ipaddress) {
        SS.socket = (new WebSocket("ws://" + ipaddress));
        //Socket on Open
        SS.socket.onopen = function(event) {};
        //Socket on Receive Message
        SS.socket.onmessage = function(event) {
            b = SS.BoxData(event.data);

            switch(b.command) {
                case SS.Constants.cList: {
                    tlist = b.data.split(";");
                    for (i = 0; i < tlist.length; i++) {
                        trow = rlist[i].split(",");
                        SS.groups[trow[1]] = trow[0];
                    }
                    break;
                }
                case SS.Constants.cSend: {
                    receiveboxqueue.push(b);
                    break;
                }
                case SS.Constants.cSendInd: {
                    receiveboxqueue.push(b);
                    break;
                }
                case SS.Constants.cDisconnect: {
                    SS.socket.close();
                    break;
                }
                case SS.Constants.cPing: {
                    if(b.data == "0") {
                        SS.Ping(1);
                    }
                    break;
                }
            }

        };
        //Socket Close
        SS.socket.onclose = function(event) {};
        //Socket Error
        SS.socket.onerror = function(event) {};
    },
    //Disconnect -: kill the connection with the SS
    Disconnect: function() {
        b = new Box();
        b.command = SS.Constants.cDisconnect;
        SS.Send(b)
        SS.socket.close();
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
        b.size = UInt8toUInt32(tSize);
        b.command = UInt8toUInt32(tCommand);
        b.destination = UInt8toUInt32(tDestination);
        b.source = UInt8toUInt32(tSource);
        b.data = data.slice(16);

        return b;
    },
    //UnBoxData -: Break is all down into a byte array and send it off
    UnBoxData: function(box){
        var dec = new TextEncoder("utf-8");

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
        return data;
    },
    //Constants -: Return uint32 code of Command
    Constants: {
        cDisconnect: 0,
        cPing: 1,
        cList: 2,
        cCreate: 3,
        cDelete: 4,
        cJoin: 5,
        cLeave: 6,
        cSend: 7,
        cSendInd: 8
    },
    //Send - : shoot a Box over to the SS
    Send: function(boxdata) {
        SS.socket.send(SS.UnBoxData(boxdata));
    },
    Receive: function() {
        return SS.receiveboxqueue.shift();
    },
    //Ping -: ping the SS to make sure it's there
    Ping: function(v) {
        if (v == undefined) { v = 0; }
        b = new Box();
        b.command = SS.Constants.cPing;
        b.data = v;
        SS.Send(b);
    },
    /*
        Group Commands
        | Create
        | Delete
        | Join
        | Leave
        | List
    */
    //GroupCreate -: create a group on the SS
    GroupCreate: function(name, pw, mpw, capacity) {
        b = new Box();
        b.command = SS.Constants.cCreate;
        b.data = name + "," + pw + "," + mpw + "," + capacity;
        SS.Send(b)
        SS.GroupList();
    },
    //GroupDelete -: delete a group on the SS
    GroupDelete: function(name) {
        if (SS.groups[name] != undefined) {
            b = new Box();
            b.command = SS.Constants.cDelete;
            b.destination = SS.groups[name];
            SS.Send(b);
            SS.GroupList();
        } else {
            SS.GroupList();
        }
    },
    //GroupJoin -: subscribe to a group to receive data from them
    GroupJoin: function(name, pw, optmpw) {
        if (SS.groups[name] != undefined) {
            b = new Box();
            b.command = SS.Constants.cJoin;
            b.destination = SS.groups[name];
            b.data = pw + "," + mpw;
            SS.Send(b);
        } else {
            SS.GroupList();
        }
    },
    //GroupLeave -: unsubscribe from a group to stopm receiving data from them
    GroupLeave: function(name) {
        if (SS.groups[name] != undefined) {
            b = new Box();
            b.command = SS.Constants.cLeave;
            b.destination = SS.groups[name];
            SS.Send(b);
        } else {
            SS.GroupList();
        }
    },
    //GroupList -: gather the current state of the SS
    GroupList: function() {
        b = new Box();
        b.command = SS.Constants.cList;
        SS.Send(b);
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

//Function to cast UInt32 to a UInt8[]
function UInt32toUInt8(value) {
    console.log(value);
    t =  [
        value >> 24,
        (value << 8) >> 24,
        (value << 16) >> 24,
        (value << 24) >> 24
    ];
    console.log(t);
    return t;
};

//Function to cast UInt8[] to UInt32
function UInt8toUInt32(value) {
    return (new DataView(value.buffer)).getUint32();
};


