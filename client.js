var socket = require('socket.io-client')('http://localhost:5331')
var admin = require("firebase-admin");
var serviceAccount = require("./serviceAccountKey.json");

admin.initializeApp({
  credential: admin.credential.cert(serviceAccount),
  databaseURL: ""
});


// watch for requests from remote clients
var queryRef = admin.database().ref("query")
queryRef.on("child_added", function(snapshot){

  // watch for changes in child key
  queryRef.child(snapshot.key)
    .child("request")
    .on("value", function(snapshot){

      // send request string to indexer
      var q_str = snapshot.val()
      // console.log("Q.STR", q_str, q_str == null)

      if (q_str == null || q_str.length<3){ return }
      socket.emit('query:event', q_str, function(responses){
        if (responses == null) { // no response
          snapshot.ref.parent.child("response").set([]) 
          return
        }

        // parse each response into a match item
        var matches = [] 
        responses.forEach(function(response){
          var matchText = response["text"]
          var wordsByPreceedingHit = matchText.split("<mark>")
          var preceedingText = wordsByPreceedingHit[0]
          var trailingText = wordsByPreceedingHit[1]
          var preceedingWords = preceedingText.split(" ")
          var nPreecdingWords = preceedingWords.length
          if (nPreecdingWords > 3) {
            matchText = [preceedingWords[nPreecdingWords-3],
                         preceedingWords[nPreecdingWords-2],
                         preceedingWords[nPreecdingWords-1]].join(" ")
            matchText += "<mark>"+wordsByPreceedingHit.slice(1).join("<mark>")
          }

          if (trailingText != null){
            var trailingWords = trailingText.split(" ")
            var nTrailingWords = trailingWords.length
            if (nTrailingWords < 2){
              // give more context by adding more preceeding words if possible
              if (nPreecdingWords > 5) {
                matchText = [preceedingWords[nPreecdingWords-5],
                             preceedingWords[nPreecdingWords-4],
                             matchText].join(" ")
              }
            }
          }

          if (nPreecdingWords > 3){
            matchText = "..."+matchText
          }
      
          // replace highlights with fire text color
          matchText = matchText.replace(/<mark>/g, "<span style=\"color:#FC5830;\">")
                                .replace(/<\/mark>/g, "</span>")

          // give text a tan color
          matchText ="<span style=\"color:#687077\">"+matchText+"</span>"

          response["text"] = matchText
          matches.push(response)
        })

        // send response
        snapshot.ref.parent.child("response").set(matches) 
      })

    })
})


// remove listener when client removes
queryRef.on("child_removed", function(snapshot){

})



/* **** doc structure
id: "01001004"
score: 0.31715551877440973
text: "<span style=\"color:#687077\">And <span  style=\"color:#FC5830;\">God</span> saw that the </span>"
*/
