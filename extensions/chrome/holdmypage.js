//var baseURL = "http://localhost:8080";
var baseURL = "http://holdmypage.appspot.com";
var bookmarked = {};

chrome.tabs.onActivated.addListener(function(activeInfo) {
	//activeInfo.tabId
	//activeInfo.windowId
});

$(document).ready(function() {
	$("#bookmarkCurrentPage").click(function() {
		if(bookmarked[tab.url] !== undefined) return;

		$.post(baseURL+'/add', {
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

	$("#bookmarkAllTabs").click(function() {
		var batchUrls = "";
		chrome.windows.getCurrent({populate:true}, function(win) {
			$.each(win.tabs, function(index, tab){
				if(bookmarked[tab.url] !== undefined) return;
				bookmarked[tab.url] = {
					url : tab.url,
					title : tab.title
				};
				batchUrls = batchUrls + tab.url + ';';
			});
			if(batchUrls.length > 0) {
				$.post(baseURL+'/batchadd', { urls : batchUrls }, function(data){

				}).fail(function(data) {
					alert(data);
				});

				chrome.browserAction.setBadgeText({ text : "HELD" });
				window.setTimeout(function() {
					chrome.browserAction.setBadgeText({ text : "" });
				}, 1000);
			}
		});
	});

	$("#openAllSavedTabs").click(function() {
		$.get(baseURL+'/getlinks', function(data){
			var urls = data.split(";");
			$.each(urls, function(index, url) {
				if(url !== "")
					chrome.tabs.create({ url: url });
			});
		}).fail(function(data) {
			
		});
	});
});