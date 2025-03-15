window.onload = function() {
    let el = document.getElementById('fileInput')
    el.addEventListener('change', function() {
        const file = el.files[0];
        if (file) {
            const reader = new FileReader();
            
            reader.onload = function(e) {
                const fileContent = e.target.result;

                updateState(parseContent(fileContent));
                reloadState();
                // document.getElementById('fileContent').textContent = fileContent;
            };

            reader.readAsText(file);
        }
    });
}


var roomLogs = new Map();

function parseContent(content) { // returns []logs
    return content.split('\n')
    .map(line => line.trim())
    .filter(line => line.trim() !== '')
    .map(JSON.parse)
}

function updateState(logs) {
    console.log(logs);

    roomLogs = new Map();

    for (let log of logs) {
        let room = log.RoomID;

        if (!roomLogs.has(room)) {
            roomLogs.set(room, []);
        }

        roomLogs.get(room).push(log);
    }
}

function reloadState() {
    var el = document.getElementById('log-viewer')
    for (var room of roomLogs.keys()) {
        var roomEl = document.createElement('h2');
        roomEl.textContent = "Room id is " + room + ". List of Events: ";
        el.appendChild(roomEl);

        var logs = roomLogs.get(room);
        logs.sort((a, b) => new Date(a.Time) - new Date(b.Time)) 

        var logsEl = document.createElement('ul')
        
        logs.forEach(log => {
            let el = document.createElement("li")
            
            const time = new Date(log.Time);
            const formatedTime = 
                String(time.getUTCHours()).padStart(2, "0") + 
                ":" + 
                String(time.getUTCMinutes()).padStart(2, "0") + 
                ":" + 
                String(time.getUTCSeconds()).padStart(2, "0") + 
                "." + 
                String(time.getUTCMilliseconds()).padStart(3, '0');

            el.innerHTML = formatedTime + " " + log.DisplayName + ": " + (log.Speech ? log.Speech : log.Message);
            // el.innerHTML = "event: " + log.Message.bold() + ", happened at: " + new Date(log.Time) + (log.AudioFile ? " AudioFile: " + log.AudioFile : "")
            logsEl.appendChild(el)
        });

        roomEl.after(logsEl)

        // for (var participant of roomLogs.get(room).keys()) {
        //     let logs = roomLogs.get(room).get(participant)
        //     if (logs.length == 0) {
        //         continue;
        //     }
            
        //     let participantEl = document.createElement('h4');
        //     participantEl.textContent = logs[0].DisplayName + "(id: " + participant + ")";
        //     roomEl.after(participantEl);

        //     var logsEl = document.createElement('ul')
            
        //     logs.forEach(log => {
        //         let el = document.createElement("li")
        //         el.innerHTML = "event: " + log.Message.bold() + ", happened at: " + new Date(log.Time) + (log.AudioFile ? " AudioFile: " + log.AudioFile : "")
        //         logsEl.appendChild(el)
        //     });

        //     participantEl.after(logsEl)
        // }
    }
}
