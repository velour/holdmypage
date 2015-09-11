var bookmarked = {};

chrome.browserAction.onClicked.addListener(function(tab) {
	if(bookmarked[tab.url] !== undefined) return;

	$.post('http://holdmypage.appspot.com/add', {
		url : tab.url
	}, function(data){

	}).fail(function(data) {

	});

	chrome.browserAction.setBadgeText({ text : "HELD" });
	window.setTimeout(function() {
		chrome.browserAction.setBadgeText({ text : "" });
	}, 1000);

	bookmarked[tab.url] = {
		url : tab.url,
		title : tab.title
	};
});