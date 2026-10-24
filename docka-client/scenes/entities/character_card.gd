extends Control

signal card_clicked(entity_id: String)

@export var entity_id: String = ""
@export var character_name: String = "Unit"
@export var card_color: Color = Color(0.25, 0.25, 0.30)
@export var max_hp: int = 100

@onready var card_rect: ColorRect = $CardRect
@onready var name_label: Label = $NameLabel
@onready var status_icon_container: HBoxContainer = $StatusIconContainer
@onready var health_bar = $HealthBar

var original_color: Color

func _ready() -> void:
	original_color = card_color
	mouse_filter = Control.MOUSE_FILTER_STOP 
	
	# Recursively make all children ignore the mouse so they don't block the root
	for child in get_children():
		_ignore_mouse_recursive(child)
	
	if card_rect != null:
		card_rect.color = card_color
	if name_label != null:
		name_label.text = character_name
	if health_bar != null:
		health_bar.setup(max_hp)
		
	# Clear any default icons in the container
	if status_icon_container != null:
		for child in status_icon_container.get_children():
			child.queue_free()
	
	gui_input.connect(_on_gui_input)

func _ignore_mouse_recursive(node: Node) -> void:
	if node is Control:
		node.mouse_filter = Control.MOUSE_FILTER_IGNORE
	for child in node.get_children():
		_ignore_mouse_recursive(child)

func _on_gui_input(event: InputEvent) -> void:
	if event is InputEventMouseButton and event.pressed and event.button_index == MOUSE_BUTTON_LEFT:
		card_clicked.emit(entity_id)

func update_highlight(is_active_turn: bool, is_selected_target: bool) -> void:
	if card_rect == null:
		return 
		
	if is_active_turn:
		card_rect.color = Color(1.0, 0.8, 0.2) # #FFCC33 (Gold)
	elif is_selected_target:
		card_rect.color = Color(0.9, 0.2, 0.2) # #E63333 (Red)
	else:
		card_rect.color = original_color

func update_statuses(statuses: Array) -> void:
	if status_icon_container == null:
		return
		
	# Clear existing icons
	for child in status_icon_container.get_children():
		child.queue_free()
		
	# Create a new icon for each status
	for s in statuses:
		var status_type = str(s["type"])
		var icon = ColorRect.new()
		icon.custom_minimum_size = Vector2(16, 16)
		
		# Determine the color based on the status type
		var is_debuff = false
		if s.has("is_debuff"):
			is_debuff = s["is_debuff"]
			
		match status_type:
			"HASTE":
				icon.color = Color(0.2, 0.8, 1.0) # #33CCFF (Light Blue)
			"SLOW":
				icon.color = Color(0.8, 0.2, 0.8) # #CC33CC (Purple)
			"POISON":
				icon.color = Color(0.6, 0.2, 0.8) # #9933CC (Dark Purple)
			"DEF_UP":
				icon.color = Color(0.2, 0.8, 0.2) # #33CC33 (Green)
			"DEF_DOWN":
				icon.color = Color(1.0, 0.2, 0.2) # #FF3333 (Red)
			"SHIELD":
				icon.color = Color(1.0, 1.0, 1.0) # #FFFFFF (White)
			"BANISHED":
				icon.color = Color(0.3, 0.0, 0.5) # #4D0080 (Deep Purple)
			_:
				# Fallback color for unknown statuses
				if is_debuff:
					icon.color = Color(0.8, 0.2, 0.2)
				else:
					icon.color = Color(0.2, 0.8, 0.2)
		
		status_icon_container.add_child(icon)