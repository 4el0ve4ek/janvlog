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


var roomToParticipantToLogs = new Map();

function parseContent(content) { // returns []logs
    return content.split('\n')
    .map(line => line.trim())
    .filter(line => line.trim() !== '')
    .map(JSON.parse)
}

function updateState(logs) {
    console.log(logs);

    roomToParticipantToLogs = new Map();

    for (let log of logs) {
        let room = log.RoomID;

        let participant = log.ParticipantID;
        if (!roomToParticipantToLogs.has(room)) {
            roomToParticipantToLogs.set(room, new Map());
        }

        let participantToLogs = roomToParticipantToLogs.get(room);
        if (!participantToLogs.has(participant)) {
            participantToLogs.set(participant, []);
        }

        let participantLogs = participantToLogs.get(participant);
        participantLogs.push(log);
    }
}

function reloadState() {
    var el = document.getElementById('log-viewer')
    for (var room of roomToParticipantToLogs.keys()) {
        var roomEl = document.createElement('h2');
        roomEl.textContent = "Room id is " + room + ". List of Participants: ";
        el.after(roomEl);

        for (var participant of roomToParticipantToLogs.get(room).keys()) {
            let logs = roomToParticipantToLogs.get(room).get(participant)
            if (logs.length == 0) {
                continue;
            }
            
            let participantEl = document.createElement('h4');
            participantEl.textContent = logs[0].DisplayName + "(id: " + participant + ")";
            roomEl.after(participantEl);

            var logsEl = document.createElement('ul')
            
            logs.forEach(log => {
                let el = document.createElement("li")
                el.innerHTML = "event: " + log.Message.bold() + ", happened at: " + new Date(log.Time) + (log.AudioFile ? " AudioFile: " + log.AudioFile : "")
                logsEl.appendChild(el)
            });

            participantEl.after(logsEl)
        }
    }
}
