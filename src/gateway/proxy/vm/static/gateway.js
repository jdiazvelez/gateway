/**
 * AP is the root namespace for all Gateway-provided functionality.
 *
 * @namespace
 */
var AP = AP || {};

AP.prepareRequests = function() {
  var requests = [];
  var numCalls = arguments.length;
  for (var i = 0; i < numCalls; i++) {
    var call = arguments[i];
    if (!call.request) {
      requests.push(request);
    } else {
      requests.push(call.request);
    }
  }
  return JSON.stringify(requests);
}

AP.insertResponses = function(calls, responses) {
  var numCalls = calls.length;
  for (var i = 0; i < numCalls; i++) {
    var call = calls[i];
    call.response = responses[i];
    if (call.response.type == "mongodb") {
      results = call.response.data;
      for (var i in results) {
        var idObject = results[i]._id
        if (idObject !== null && typeof idObject === 'object' && idObject._id !== null) {
          results[i]._id = ObjectId(idObject._id);
        }
      }
    }
    if (numCalls == 1) {
      response = call.response;
    }
  }
}
