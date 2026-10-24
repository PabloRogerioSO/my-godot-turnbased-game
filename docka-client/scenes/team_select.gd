extends Control

var roster = [
	{"id": "h1", "name": "Knight", "hp": 250},
	{"id": "h2", "name": "Archer", "hp": 100},
	{"id": "h3", "name": "Mage", "hp": 100},
	{"id": "h4", "name": "Cleric", "hp": 150},
	{"id": "h5", "name": "Rogue", "hp": 110},
	{"id": "h6", "name": "Paladin", "hp": 300}
]

@onready var roster_container = $VBoxContainer/RosterContainer
@onready var selected_container = $VBoxContainer/SelectedContainer
@onready var start_button = $VBoxContainer/Button

func _ready() -> void:
	start_button.pressed.connect(_on_start_pressed)
	refresh_ui()

func refresh_ui() -> void:
	# Clear existing buttons
	for child in roster_container.get_children():
		child.queue_free()
	for child in selected_container.get_children():
		child.queue_free()

	# Build Roster
	for hero in roster:
		var btn = Button.new()
		btn.text = hero["name"] + "\n" + str(hero["hp"]) + " HP"
		btn.custom_minimum_size = Vector2(100, 100)
		
		if GameState.selected_hero_ids.has(hero["id"]):
			btn.disabled = true
		else:
			btn.pressed.connect(_on_hero_clicked.bind(hero["id"]))
			
		roster_container.add_child(btn)

	# Build Selected
	for hero_id in GameState.selected_hero_ids:
		for hero in roster:
			if hero["id"] == hero_id:
				var btn = Button.new()
				btn.text = hero["name"] + "\n(Click to Remove)"
				btn.custom_minimum_size = Vector2(100, 100)
				btn.pressed.connect(_on_selected_clicked.bind(hero["id"]))
				selected_container.add_child(btn)

	# Toggle Start Button
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
		get_tree().change_scene_to_file("res://scenes/Battle.tscn")
	else:
		start_button.text = "Error connecting to server"
		start_button.disabled = false
