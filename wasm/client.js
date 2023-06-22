const {Input, image} = require('./defs.proto3_pb.js');
const {StreamClient} = require('./defs.proto3_grpc_web_pb.js');

console.log("dialing localhost")
var streamClient = new StreamClient('127.0.0.1:2000');
var input = new Input;

var stream = streamClient.getImage(input);
stream.on('data', function(response) {
	console.log(response.getMessage());
});
stream.on('status', function(status) {
	console.log(status.code);
	console.log(status.details);
	console.log(status.metadata);
});
stream.on('end', function(end) {
	// stream end signal
});

// to close the stream
stream.cancel()
