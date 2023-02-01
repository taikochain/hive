import '../extlib/bootstrap.module.js'
import '../extlib/dataTables.module.js'
import { $ } from '../extlib/jquery.module.js'
import { html, format, nav } from './utils.js'
import * as app from './app.js'

$(document).ready(function() {
	app.init();
	
	console.log("Loading file list...");
	$.ajax("listing.jsonl", {
		success: showFileListing,
		failure: function(status, err) {
			alert(err);
		},
	});
});

function resultStats(fails, success, total) {
	f = parseInt(fails), s = parseInt(success);
	t = parseInt(total);
	f = isNaN(f) ? "?" : f;
	s = isNaN(s) ? "?" : s;
	t = isNaN(t) ? "?" : t;
	return '<b><span class="text-danger">' + f +
		'</span>&nbsp;:&nbsp;<span class="text-success">' + s +
		'</span> &nbsp;/&nbsp;' + t + '</b>';
}

function linkToSuite(suiteID, suiteName, linkText) {
	let url = app.route.suite(suiteID, suiteName);
	return html.get_link(url, linkText);
}

function showFileListing(data, error) {
	console.log("Got file list")
	// the data is jsonlines
	/*
		{
			"fileName": "./1587325327-fa7ec3c7d09a8cfb754097f79df82118.json",
			"name": "Sync test suite",
			"start": "",
			"simLog": "1587325280-00befe48086b1ef74fbb19b9b7d43e4d-simulator.log",
			"passes": 0,
			"fails": 0,
			"size": 435,
			"clients": [],
			"description": "This suite of tests verifies that clients can sync from each...'\n",
			"ntests": 0
	}
	*/

	let table = $("#filetable");
	var suites = [];
	data.split("\n").forEach(function(elem, index) {
		if (!elem) {
			return;
		}
		suites.push(JSON.parse(elem));
	});

	filetable = $("#filetable").DataTable({
		data: suites,
		pageLength: 50,
		autoWidth: false,
		order: [[0, 'desc']],
		columns: [
			{
				title: "Start time",
				data: "start",
				type: "date",
				width: "11em",
				render: function(v, type) {
					if (type === 'display') {
						return new Date(v).toLocaleString();
					}
					return v;
				},
			},
			{
				title: "Suite",
				data: "name",
				width: "10em",
			},
			{
				title: "Clients",
				data: "clients",
				width: "30%",
				render: function(data) {
					return data.join(", ")
				},
			},
			{
				title: "Status",
				data: null,
				width: "9em",
				render: function(data) {
					if (data.fails > 0) {
						let prefix = data.timeout ? "Timeout" : "Fail";
						return "&#x2715; <b>" + prefix + " (" + data.fails + " / " + (data.fails + data.passes) + ")</b>";
					}
					return "&#x2713 (" + data.passes + ")";
				},
			},
			{
				title: "",
				data: null,
				width: "180px",
				orderable: false,
				render: function(data) {
					let loadText = "Load (" + format.units(data.size) + ")";
					let loadLink = linkToSuite(data.fileName, data.name, loadText);
					const btnclass = ["btn", "btn-sm", "btn-primary"];
					loadLink.classList.add(...btnclass);
					return loadLink.outerHTML;
				},
			},
		],
	});
}
