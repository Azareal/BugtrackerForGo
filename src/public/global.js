function post_link(event)
{
	event.preventDefault();
	var form_action = $(event.target).closest('a').attr("href");
	console.log("Form Action: " + form_action);
	$.ajax({
		url: form_action,
		type: "POST",
		dataType: "json",
		data: {is_js: "1"}
	});
}

$(document).ready(function(){
	$(".open_edit").click(function(event){
		console.log("Clicked on edit");
		event.preventDefault();
		
		$(".hide_on_edit").hide();
		$(".show_on_edit").show();
	});
	
	$(".submit_edit").click(function(event){
		event.preventDefault();
		
		$(".issue_name").html($(".issue_name_input").val());
		$(".issue_content").html($(".issue_content_input").val());
		$(".issue_status_e:not(.open_edit)").html($(".issue_status_input").val());
		
		$(".hide_on_edit").show();
		$(".show_on_edit").hide();
		
		var issue_name_input = $('.issue_name_input').val();
		var issue_status_input = $('.issue_status_input').val();
		var issue_content_input = $('.issue_content_input').val();
		var form_action = $(this).closest('form').attr("action");
		console.log("New Issue Name: " + issue_name_input);
		console.log("New Issue Status: " + issue_status_input);
		console.log("Form Action: " + form_action);
		$.ajax({
			url: form_action,
			data: {
				issue_name: issue_name_input,
				issue_status: issue_status_input,
				issue_content: issue_content_input,
				issue_js: 1
			},
			type: "POST",
			dataType: "json"
		});
	});
	
	$(".post_link").click(post_link);
	$(".delete_item").click(function(event)
	{
		post_link(event);
		var block = $(this).closest('.deletable_block');
		block.remove();
	});
	
	$(".edit_item").click(function(event)
	{
		event.preventDefault();
		var block_parent = $(this).closest('.editable_parent');
		var block = block_parent.find('.editable_block').eq(0);
		//block.html("<textarea style='width: 100%;' name='edit_" +
		block.html("<textarea style='width: 100%;' name='edit_item'>" + block.html() + "</textarea><br /><a href='" + $(this).closest('a').attr("href") + "'><button class='submit_edit' type='submit'>Update</button></a>");
		
		$(".submit_edit").click(function(event)
		{
			event.preventDefault();
			var block_parent = $(this).closest('.editable_parent');
			var block = block_parent.find('.editable_block').eq(0);
			var newContent = block.find('textarea').eq(0).val();
			block.html(newContent);
			
			var form_action = $(this).closest('a').attr("href");
			console.log("Form Action: " + form_action);
			$.ajax({
				url: form_action,
				type: "POST",
				dataType: "json",
				data: {is_js: "1",edit_item: newContent}
			});
		});
	});
});