extends Node

var http_request: HTTPRequest

func _ready() -> void:
	# We create the HTTPRequest node entirely in code to keep things clean
	http_request = HTTPRequest.new()
	add_child(http_request)

# Fetches the battle state from the Go server
func fetch_battle_state() -> Dictionary:
	print("Fetching battle state from Go server...")
	
	var error = http_request.request("http://localhost:8080/api/battle")
	if error != OK:
		print("Failed to start HTTP request.")
		return {}
		
	# The await keyword pauses this function until the server responds
	var response = await http_request.request_completed
	
	var result = response[0]
	var response_code = response[1]
	var headers = response[2]
	var body = response[3]
	
	if result == HTTPRequest.RESULT_SUCCESS and response_code == 200:
		var json = JSON.new()
		var parse_error = json.parse(body.get_string_from_utf8())
		if parse_error == OK:
			print("Successfully parsed JSON!")
			return json.data
		else:
			print("JSON Parse Error.")
			return {}
	else:
		print("HTTP Error: ", result, " Code: ", response_code)
		return {}
