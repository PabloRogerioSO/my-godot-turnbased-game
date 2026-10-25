extends Control

# --- PACING CONSTANTS ---
const TURN_DELAY := 0.5    
const ACTION_DELAY := 0.8  

@onready var hero1 = find_child("Char1", true, false)
@onready var hero2 = find_child("Char2", true, false)
@onready var hero3 = find_child("Char3", true, false)

@onready var mob1 = find_child("Mon1", true, false)
@onready var mob2 = find_child("Mon2", true, false)
@onready var mob3 = find_child("Mon3", true, false)

@onready var timeline_container = find_child("TimelineContainer", true, false)
@onready var action_button = find_child("ActionButton", true, false)
@onready var skill2_button = find_child("Skill2Button", true, false)
@onready var skill3_button = find_child("Skill3Button", true, false)

@onready var turn_label = find_child("TurnLabel", true, false)
@onready var auto_mode_check = find_child("AutoModeCheckBox", true, false)

@onready var action_log_label = find_child("ActionLogLabel", true, false)
@onready var reset_button = find_child("ResetButton", true, false)

@onready var game_over_overlay = find_child("GameOverOverlay", true, false)
@onready var game_over_label = find_child("GameOverLabel", true, false)

var http_request: HTTPRequest
var selected_target_id: String = ""
var is_player_turn: bool = false
var current_active_id: String = ""
var active_team: int = 0

var latest_entities: Array = []
var pending_skill_index: int = -1 
var pending_target_type: String = ""

func _ready() -> void:
	http_request = HTTPRequest.new()
	add_child(http_request)
	http_request.request_completed.connect(_on_request_completed)
	
	if action_button != null:
		action_button.pressed.connect(_on_action_button_pressed.bind(0))
	if skill2_button != null:
		skill2_button.pressed.connect(_on_action_button_pressed.bind(1))
	if skill3_button != null:
		skill3_button.pressed.connect(_on_action_button_pressed.bind(2))
		
	if reset_button != null:
		reset_button.pressed.connect(_on_reset_button_pressed)
	
	if game_over_overlay != null:
		game_over_overlay.gui_input.connect(_on_game_over_overlay_input)
	
	var all_cards = [hero1, hero2, hero3, mob1, mob2, mob3]
	for card in all_cards:
		if card != null:
			card.card_clicked.connect(_on_card_clicked)
	
	fetch_state("http://localhost:8080/api/battle", HTTPClient.METHOD_GET, 0)

func _on_game_over_overlay_input(event: InputEvent) -> void:
	if game_over_overlay.visible and event is InputEventMouseButton and event.pressed and event.button_index == MOUSE_BUTTON_LEFT:
		_on_reset_button_pressed()

func _on_reset_button_pressed() -> void:
	game_over_overlay.visible = false
	action_log_label.text = "Resetting game..."
	selected_target_id = ""
	pending_skill_index = -1
	pending_target_type = ""
	fetch_state("http://localhost:8080/api/reset", HTTPClient.METHOD_POST, 0)

func _on_card_clicked(entity_id: String) -> void:
	if not is_player_turn:
		return 
		
	if pending_skill_index != -1:
		var is_enemy = false
		var is_ally = false
		
		for entity in latest_entities:
			if str(entity["id"]) == entity_id:
				if int(entity["team"]) == active_team:
					is_ally = true
				else:
					is_enemy = true
				break
		
		if pending_target_type == "SINGLE_ENEMY" and not is_enemy:
			return
		if pending_target_type == "SINGLE_ALLY" and not is_ally:
			return

	selected_target_id = entity_id
	refresh_highlights()
	
	if pending_skill_index != -1:
		disable_buttons()
		fetch_state("http://localhost:8080/api/action", HTTPClient.METHOD_POST, pending_skill_index)
		pending_skill_index = -1
		pending_target_type = ""
		turn_label.text = "Executing..."

func _on_action_button_pressed(skill_index: int) -> void:
	var active_entity = null
	for entity in latest_entities:
		if str(entity["id"]) == current_active_id:
			active_entity = entity
			break
			
	if active_entity != null and skill_index < active_entity["skills"].size():
		var skill = active_entity["skills"][skill_index]
		var target_type = skill["target_type"]
		
		if (target_type == "SINGLE_ENEMY" or target_type == "SINGLE_ALLY") and is_player_turn and not auto_mode_check.button_pressed:
			pending_skill_index = skill_index
			pending_target_type = target_type
			selected_target_id = ""
			
			if target_type == "SINGLE_ENEMY":
				turn_label.text = "Select an ENEMY for " + skill["name"] + "..."
			else:
				turn_label.text = "Select an ALLY for " + skill["name"] + "..."
				
			refresh_highlights()
			return 

	disable_buttons()
	
	if not is_player_turn or auto_mode_check.button_pressed:
		selected_target_id = ""
		
	fetch_state("http://localhost:8080/api/action", HTTPClient.METHOD_POST, skill_index)
	pending_skill_index = -1
	pending_target_type = ""

func fetch_state(url: String, method: int, skill_index: int) -> void:
	var body = JSON.stringify({"skill_index": skill_index, "target_id": selected_target_id})
	var headers = ["Content-Type: application/json"]
	
	var error = http_request.request(url, headers, method, body)
	if error != OK:
		enable_buttons()

func enable_buttons() -> void:
	if is_player_turn:
		if action_button != null: action_button.disabled = false
		if skill2_button != null: skill2_button.disabled = false
		if skill3_button != null: skill3_button.disabled = false

func disable_buttons() -> void:
	if action_button != null: action_button.disabled = true
	if skill2_button != null: skill2_button.disabled = true
	if skill3_button != null: skill3_button.disabled = true

func _on_request_completed(result, response_code, _headers, body) -> void:
	if result == HTTPRequest.RESULT_SUCCESS and response_code == 200:
		var json = JSON.new()
		if json.parse(body.get_string_from_utf8()) == OK:
			var data = json.data
			latest_entities = data["entities"]
			
			update_cards(data["entities"])
			update_timeline(data["timeline"])
			
			if data.has("action_log"):
				var log_text = ""
				for entry in data["action_log"]:
					log_text += str(entry) + "\n"
				action_log_label.text = log_text
				await get_tree().process_frame
				action_log_label.scroll_to_line(action_log_label.get_line_count() - 1)
			
			if data.has("steps"):
				await play_visual_steps(data["steps"])
			
			handle_turn_logic(data)
		else:
			print("JSON Parse Error.")

func handle_turn_logic(data: Dictionary) -> void:
	if data["is_game_over"]:
		game_over_overlay.visible = true
		var winner = int(data["winning_team"])
		if winner == 0:
			game_over_label.text = "VICTORY!"
			game_over_label.add_theme_color_override("font_color", Color(0.2, 0.8, 0.2))
		else:
			game_over_label.text = "DEFEAT!"
			game_over_label.add_theme_color_override("font_color", Color(0.8, 0.2, 0.2))
		
		disable_buttons()
		turn_label.text = "Battle Over"
		current_active_id = ""
		refresh_highlights()
		return
		
	active_team = int(data["active_team"])
	is_player_turn = (active_team == 0)
	
	current_active_id = str(data["active_actor_id"])
	
	var active_actor_name = current_active_id
	var active_entity = null
	for entity in data["entities"]:
		if entity["id"] == current_active_id:
			active_actor_name = entity["name"]
			active_entity = entity
			break
			
	turn_label.text = "Waiting for: " + active_actor_name
	
	if auto_mode_check.button_pressed or not is_player_turn:
		disable_buttons()
		refresh_highlights()
		
		await get_tree().create_timer(TURN_DELAY).timeout
		_on_action_button_pressed(0)
	else:
		turn_label.text += " (Your Turn)"
		selected_target_id = "" 
		
		if active_entity != null:
			var skills = active_entity["skills"]
			
			if skills.size() > 1 and skill2_button != null:
				var s2_cd = int(skills[1]["current_cooldown"])
				if s2_cd > 0:
					skill2_button.text = skills[1]["name"] + " (CD: " + str(s2_cd) + ")"
					skill2_button.disabled = true
				else:
					skill2_button.text = skills[1]["name"]
					skill2_button.disabled = false
					
			if skills.size() > 2 and skill3_button != null:
				var s3_cd = int(skills[2]["current_cooldown"])
				if s3_cd > 0:
					skill3_button.text = skills[2]["name"] + " (CD: " + str(s3_cd) + ")"
					skill3_button.disabled = true
				else:
					skill3_button.text = skills[2]["name"]
					skill3_button.disabled = false
		
		if action_button != null: action_button.disabled = false
		refresh_highlights()

func refresh_highlights() -> void:
	var all_cards = [hero1, hero2, hero3, mob1, mob2, mob3]
	for card in all_cards:
		if card != null:
			var is_active = (card.entity_id == current_active_id and is_player_turn)
			var is_target = (card.entity_id == selected_target_id and selected_target_id != "")
			card.update_highlight(is_active, is_target)

func get_card_by_id(id: String):
	match id:
		"h1": return hero1
		"h2": return hero2
		"h3": return hero3
		"m1": return mob1
		"m2": return mob2
		"m3": return mob3
	return null

func update_cards(entities: Array) -> void:
	var players = []
	var monsters = []
	
	for entity in entities:
		if int(entity["team"]) == 0:
			players.append(entity)
		else:
			monsters.append(entity)
			
	var hero_slots = [hero1, hero2, hero3]
	for i in range(min(players.size(), hero_slots.size())):
		var card = hero_slots[i]
		if card != null:
			card.entity_id = str(players[i]["id"]) 
			update_card(card, players[i])
			
	var mob_slots = [mob1, mob2, mob3]
	for i in range(min(monsters.size(), mob_slots.size())):
		var card = mob_slots[i]
		if card != null:
			card.entity_id = str(monsters[i]["id"])
			update_card(card, monsters[i])

func update_card(card, entity: Dictionary) -> void:
	var hp = int(entity["stats"]["HP"])
	var max_hp = int(entity["stats"]["MaxHP"])
	var is_alive = entity["alive"]
	var name_str = str(entity["name"])
	
	var statuses = []
	if entity.has("status_effects") and entity["status_effects"] is Array:
		statuses = entity["status_effects"]
	
	var total_shield = 0
	var is_banished = false
	for s in statuses:
		if s["type"] == "SHIELD":
			total_shield += int(s["value"])
		elif s["type"] == "BANISHED":
			is_banished = true
	
	if card.health_bar != null:
		card.health_bar.setup(max_hp, hp, total_shield)
		
	if card.name_label != null:
		card.name_label.text = name_str
		
	card.update_statuses(statuses)
	
	if not is_alive:
		card.modulate = Color(1, 1, 1, 0.3)
	elif is_banished:
		card.modulate = Color(0.5, 0.0, 1.0, 0.5)
	else:
		card.modulate = Color(1, 1, 1, 1)

func update_timeline(timeline_data: Array) -> void:
	if timeline_container == null: return
	
	for child in timeline_container.get_children():
		child.queue_free()
		
	for i in range(timeline_data.size()):
		var entry = timeline_data[i]
		var unit_name = str(entry["name"])
		var cr_val = int(entry["cr"])
		
		var display_text = unit_name + " (" + str(cr_val) + "%)"
		
		var label = Label.new()
		label.text = display_text
		
		if i == 0:
			label.add_theme_color_override("font_color", Color(1.0, 0.8, 0.2))
			label.add_theme_font_size_override("font_size", 24)
		else:
			label.add_theme_color_override("font_color", Color(0.7, 0.7, 0.7))
			label.add_theme_font_size_override("font_size", 16)
			
		label.modulate.a = 0.0 
		label.position.y = 20 
		
		timeline_container.add_child(label)
		
		var tween = create_tween()
		tween.set_parallel(true)
		tween.tween_property(label, "modulate:a", 1.0, 0.3).set_delay(i * 0.05)
		tween.tween_property(label, "position:y", 0.0, 0.3).set_delay(i * 0.05)
		
		if i < timeline_data.size() - 1:
			var arrow = Label.new()
			arrow.text = " -> "
			arrow.add_theme_color_override("font_color", Color(0.5, 0.5, 0.5))
			
			arrow.modulate.a = 0.0
			timeline_container.add_child(arrow)
			
			var tween_arrow = create_tween()
			tween_arrow.tween_property(arrow, "modulate:a", 1.0, 0.3).set_delay(i * 0.05 + 0.05)

func play_visual_steps(steps: Array) -> void:
	if steps.is_empty(): return

	var current_batch = []
	var current_group = ""

	for step in steps:
		var step_type = step["type"]
		var step_group = step_type
		
		# Added "MISS" to the DAMAGE_PHASE so it shows up at the same time as the damage number
		if step_type in ["DAMAGE", "SHARED_DAMAGE", "REFLECT_DAMAGE", "SHIELD_DAMAGE", "MISS"]:
			step_group = "DAMAGE_PHASE"
			
		if step_group != current_group and not current_batch.is_empty():
			process_visual_batch(current_batch)
			await get_tree().create_timer(ACTION_DELAY).timeout
			current_batch.clear()

		current_batch.append(step)
		current_group = step_group

	if not current_batch.is_empty():
		process_visual_batch(current_batch)
		await get_tree().create_timer(ACTION_DELAY).timeout

func process_visual_batch(batch: Array) -> void:
	for step in batch:
		var step_type = step["type"]
		var target_id = step["target"]
		var val = int(step["value"]) if step.has("value") else 0
		
		if step_type == "DAMAGE":
			spawn_floating_text(target_id, "-" + str(val), Color(1.0, 0.2, 0.2))
		elif step_type == "SHARED_DAMAGE":
			spawn_floating_text(target_id, "-" + str(val), Color(0.8, 0.8, 0.8))
		elif step_type == "REFLECT_DAMAGE":
			spawn_floating_text(target_id, "-" + str(val), Color(0.0, 0.8, 1.0))
		elif step_type == "SHIELD_DAMAGE":
			spawn_floating_text(target_id, "-" + str(val), Color(1.0, 1.0, 1.0))
		elif step_type == "EXTRA_DAMAGE":
			spawn_floating_text(target_id, "-" + str(val), Color(1.0, 0.4, 0.0)) 
		elif step_type == "DUAL_ATTACK":
			spawn_floating_text(target_id, "Dual!", Color(0.0, 0.8, 1.0))
		elif step_type == "CR_PUSH":
			spawn_floating_text(target_id, "CR UP!", Color(0.2, 0.8, 1.0))
		elif step_type == "CR_PULL":
			spawn_floating_text(target_id, "CR DOWN!", Color(0.8, 0.2, 0.8))
		elif step_type == "EXECUTE":
			spawn_floating_text(target_id, "EXECUTED!", Color(1.0, 0.0, 0.0))
		elif step_type == "BANISH":
			spawn_floating_text(target_id, "BANISHED!", Color(0.6, 0.2, 1.0))
		elif step_type == "COUNTER":
			spawn_floating_text(target_id, "-" + str(val), Color(1.0, 0.6, 0.0)) 
		elif step_type == "DOT":
			spawn_floating_text(target_id, "-" + str(val), Color(0.6, 0.2, 0.8)) 
		elif step_type == "HEAL":
			spawn_floating_text(target_id, "+" + str(val), Color(0.2, 0.8, 0.2))
		elif step_type == "REVIVE":
			spawn_floating_text(target_id, "REVIVED!", Color(0.9, 0.9, 0.9))
		elif step_type == "CLEANSE":
			spawn_floating_text(target_id, "Cleansed!", Color(0.9, 0.9, 0.9))
		elif step_type == "BUFF":
			spawn_floating_text(target_id, str(step["text"]), Color(0.2, 0.8, 0.2))
		elif step_type == "SHIELD_ADD":
			spawn_floating_text(target_id, "+" + str(val) + " SHD", Color(1.0, 1.0, 1.0))
		elif step_type == "DEBUFF":
			spawn_floating_text(target_id, str(step["text"]), Color(0.8, 0.2, 0.8))
		elif step_type == "RESIST":
			spawn_floating_text(target_id, "Resist", Color(0.5, 0.5, 0.5))
		elif step_type == "INTERCEPT":
			spawn_floating_text(target_id, str(step["text"]), Color(1.0, 0.9, 0.0))
		elif step_type == "MISS":
			spawn_floating_text(target_id, "MISS", Color(0.6, 0.6, 0.6))

func spawn_floating_text(target_id: String, text: String, color: Color) -> void:
	var target_card = get_card_by_id(target_id)
	if target_card == null:
		return
		
	var label = Label.new()
	label.text = text
	label.add_theme_color_override("font_color", color)
	label.add_theme_font_size_override("font_size", 18) 
	label.add_theme_color_override("font_outline_color", Color(0, 0, 0, 1))
	label.add_theme_constant_override("outline_size", 4)
	
	label.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	label.custom_minimum_size = Vector2(100, 0) 
	label.mouse_filter = Control.MOUSE_FILTER_IGNORE
	
	add_child(label)
	label.z_index = 100 
	
	var card_center = target_card.global_position + (target_card.size / 2)
	label.global_position = card_center - Vector2(50, 10) + Vector2(randf_range(-10, 10), randf_range(-20, 10))
	
	var tween = create_tween()
	tween.set_parallel(true)
	tween.tween_property(label, "global_position:y", label.global_position.y - 60, 1.5)
	tween.tween_property(label, "modulate:a", 0.0, 1.5)
	
	tween.chain().tween_callback(label.queue_free)