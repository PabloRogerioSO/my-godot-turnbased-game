extends Control

var roster = [
	{"id": "h1", "name": "Sofa Cigs", "hp": 250},
	{"id": "h2", "name": "Assemelhado Verde", "hp": 100},
	{"id": "h3", "name": "Bolsista Ceva", "hp": 150},
	{"id": "h4", "name": "Sinuca Truco", "hp": 300},
	{"id": "h5", "name": "Lojinha Cafeina", "hp": 200}
]

@onready var roster_container = find_child("RosterContainer", true, false)
@onready var selected_container = find_child("SelectedContainer", true, false)
@onready var start_button = find_child("StartButton", true, false)

func _ready() -> void:
	if start_button != null:
		start_button.pressed.connect(_on_start_pressed)
	refresh_ui()

func refresh_ui() -> void:
	for child in roster_container.get_children():
		child.queue_free()
	for child in selected_container.get_children():
		child.queue_free()

	for hero in roster:
		var btn = Button.new()
		btn.text = hero["name"] + "\n" + str(hero["hp"]) + " HP"
		btn.custom_minimum_size = Vector2(110, 100)
		
		if GameState.selected_hero_ids.has(hero["id"]):
			btn.disabled = true
		else:
			btn.pressed.connect(_on_hero_clicked.bind(hero["id"]))
			
		roster_container.add_child(btn)

	for hero_id in GameState.selected_hero_ids:
		for hero in roster:
			if hero["id"] == hero_id:
				var btn = Button.new()
				btn.text = hero["name"] + "\n(Remove)"
				btn.custom_minimum_size = Vector2(110, 100)
				btn.pressed.connect(_on_selected_clicked.bind(hero["id"]))
				selected_container.add_child(btn)

	if start_button != null:
		start_button.disabled = (GameState.selected_hero_ids.size() != 3)

func _on_hero_clicked(hero_id: String) -> void:
	if GameState.selected_hero_ids.size() < 3:
		GameState.selected_hero_ids.append(hero_id)
		refresh_ui()

func _on_selected_clicked(hero_id: String) -> void:
	GameState.selected_hero_ids.erase(hero_id)
	refresh_ui()

func _on_start_pressed() -> void:
	start_button.disabled = true
	start_button.text = "Loading..."
	
	var http = HTTPRequest.new()
	add_child(http)
	
	var body = JSON.stringify({"hero_ids": GameState.selected_hero_ids})
	var headers = ["Content-Type: application/json"]
	
	http.request("http://localhost:8080/api/start", headers, HTTPClient.METHOD_POST, body)
	var response = await http.request_completed
	
	if response[0] == HTTPRequest.RESULT_SUCCESS and response[1] == 200:
		get_tree().change_scene_to_file("res://scenes/battle.tscn")
	else:
		start_button.text = "Error connecting to server"
		start_button.disabled = false