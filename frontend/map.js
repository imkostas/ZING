var mapObj;
var httpRequest;
var userLocation = {
	lat: 40,
	lng: -95
};
var otherLocations = [
	{
		lat: 42,
		lng: -71,
		udid: "Marker 1"
	},
	{
		lat: 41,
		lng: -72,
		udid: "Marker 2"
	},
	{
		lat: 41.123,
		lng: 121.644,
		udid: "Hello World!"
	}
];

// Get current location
function getLocation() {
	if (navigator.geolocation) {
		navigator.geolocation.getCurrentPosition(setLocation);
	}
}

function setLocation(position) {
	userLocation = {
		lat: position.coords.latitude,
		lng: position.coords.longitude
	};
}
// Get locations from database
// Set map to center on current location
// Draw markers on the map at the locations from database
// Repeat 2 mins

function initMap() {
	getLocation();

	mapObj = new google.maps.Map(document.getElementById('map'), {
		center: new google.maps.LatLng(userLocation.lat, userLocation.lng),
		zoom: 5
	});

	for (var i = 0; i < otherLocations.length; i++) {
		var marker = new google.maps.Marker({
			position: new google.maps.LatLng(otherLocations[i].lat, otherLocations[i].lng),
			title: otherLocations[i].udid,
			map: mapObj
		});
	}
}

function UpdateMapFocus() {
	getLocation();
	console.log("Updated");
	mapObj.panTo(new google.maps.LatLng(userLocation.lat, userLocation.lng));
	mapObj.setZoom(8);
}

function getLocationsFromServer() {

}

function makeRequest(url) {
	httpRequest = new XMLHttpRequest();

	if (!httpRequest) {
		alert('Giving up :( Cannot create an XMLHTTP instance');
		return false;
	}
	httpRequest.onreadystatechange = alertContents;
	httpRequest.open('GET', url);
	httpRequest.send();
}

function alertContents() {
	if (httpRequest.readyState === XMLHttpRequest.DONE) {
		if (httpRequest.status === 200) {
			alert(httpRequest.responseText);
		}
		else {
			alert('There was a problem with the request.');
		}
	}
}
