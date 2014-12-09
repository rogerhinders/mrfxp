(function(__G, $) {
	var socket = null;
	var host = "ws://localhost:8888/mrfxp/ws";
	var fetchUrl = "/mrfxp/fetch";

	function sendSocket(obj) {
		if(socket == null)
			alert('error sending, socket is null');

		try {
			socket.send(JSON.stringify(obj))
		} catch(e) {
			alert('ERR: '+e);
		}
	}

	function sendAjax(url, obj, success) {
		$.ajax({
			type: "POST",
			contentType: 'application/json; charset=utf-8',
			url: url,
			data: JSON.stringify(obj),
			success: function(d) {
				success($.parseJSON(d))
			}
		});
	}

	$(function() {
		if(!("WebSocket" in window)) {
			alert('Sorry, your browser is unsupported.');
		} else {
			socket = new WebSocket(host);
			try {
				socket.onopen = function() {
					//sendSocket('hellloo')
				}

				socket.onmessage = function(msg) {
					var obj = $.parseJSON(msg.data);
					switch(obj.Event) {
					case 'SetSites':
						switch(obj.Reference) {
						case "InitSitesPage":
							setSites(obj.Data);
							break;
						case "InitTransfersPage":
							setSiteTransferDropdowns(obj.Data);
							break;
						}
						break;
					case 'SetSections':
						switch(obj.Reference) {
						case "InitSettingsPage":
							setSections(obj.Data);
							break;
						case "InitEditSitePage":
							setSectionsEditGuest(obj.Data);
							break;
						}
						break;
							
					}
				}

				socket.onclose = function() {
					//alert('socket closed.');
				}
			} catch(e) {
				alert('Err: '+e);
			}
		}

		function setSites(sites) {
			$('#site-table').html('');
			$.each(sites, function(k,v) {
				$('#site-table').append(
					'<tr style="cursor: pointer;" data-id="'+v.Id+'" class="site-list-row">'+
					'	<td>'+v.Name+'</td>'+
					'	<td>'+v.Hostname+':'+v.Port+'</td>'+
					'	<td>'+v.Tls+'</td>'+
					'	<td>'+v.Username+'</td>'+
					'</tr>'
				);
			});
		}

		function setSiteTransferDropdowns(sites) {
			alert('');
		}
		
		function setSections(sections) {
			$('#section-container').html('');
			var i = 0;
			var color = ['primary','success','info','warning','danger'];
			$.each(sections, function(k,v) {
				var style = color[i%5];
				$('#section-container').append(
					'<button class="btn btn-'+style+'" type="button" style="margin: 5px;">'+
					'	'+v.Name+' <span class="badge" style="color:red;">X</span>'+
					'</button>'
				);
				i++;
			})
		}

		function setSectionsEditGuest(sections) {
			$('#section-path-container').html('');

			$.each(sections, function(k,v) {
				$('#section-path-container').append(
					'<label for="div-set-section-'+v.Id+'">'+v.Name+'</label>'+
					'<div id="div-set-section-'+v.Id+'" class="input-group">'+
					'	<input type="text" class="form-control" placeholder="Section path">'+
					'	<span class="input-group-btn">'+
					'		<button class="btn btn-success" type="button">Save</button>'+
					'	</span>'+
					'</div>'
				);
			});
		}

		function showPage(name) {
			$('.menu-ul').find('li').removeClass('active');
			$(this).parent().addClass('active');

			$('.site-page').css('display', 'none');
			$('.'+name).css('display', 'block');
		}

		$(document).on('click', '.menu-item', function() {
			showPage($(this).attr('data-page'));

			switch($(this).attr('data-page')) {
			case 'page-settings':
				sendSocket({Event: "GetSections", Reference: "InitSettingsPage", Data: {}});
				break;
			case 'page-sites':
				sendSocket({Event: "GetSites", Reference: "InitSitesPage", Data: {}});
				break;
			case 'page-sections':
				sendSocket({Event: "GetSites", Reference: "InitTransfersPage", Data: {}});
				break;
			}
			return false;
		});

		$(document).on('click', '#btn-add-section', function() {
			var sec = $('#input-add-section').val();
			
			if(sec.length == 0)
				return false;

			sendSocket({Event: "AddSection", Data: {Name: sec}});
			$('#input-add-section').val('');
			return false;
		});

		$(document).on('click', '.site-list-row', function() {
			showPage('page-editsite');
			sendSocket({Event: "GetSections", Reference: "InitEditSitePage", Data: {}});
		});
	});
})(this, jQuery);
