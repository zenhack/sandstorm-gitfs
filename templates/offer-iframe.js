document.addEventListener("DOMContentLoaded", function(event) {

	window.addEventListener("message", function(event) {
		if (event.data.rpcId !== "0") {
			return;
		}
		if (event.data.error) {
			console.log("ERROR: " + event.data.error);
			return;
		}
		var elt = document.getElementById("offer-iframe");
		elt.setAttribute("src", event.data.uri);
	});

	window.parent.postMessage({
		renderTemplate: {
			rpcId: "0",
			template: window.location.protocol + "//git:$API_TOKEN@$API_HOST/r/{{ .RepoSlug }}",
			clipboardButton: 'left',
		}
	}, "*");

});
