var hostname = window.location.host || "example-janus-log.ru";

var server = "https://" + hostname + "/janus";;
if(window.location.protocol === 'http:')
	 server = "http://" + hostname + ":8088/janus";

var iceServers = null;
