extends Control

signal card_clicked(entity_id: String)

@export var entity_id: String = ""
@export var character_name: String = "Unit"
@export var card_color: Color = Color(0.25, 0.25, 0.30)
@export var max_hp: int = 100

@onready var card_rect: ColorRect = $CardRect
@onready var name_label: Label = $NameLabel
@onready var status_icon_container: FlowContainer = $StatusIconContainer
@onready var health_bar = $HealthBar

var original_color: Color

func _ready() -> void:
	original_color = card_color
	mouse_filter = Control.MOUSE_FILTER_STOP 
	
	for child in get_children():
		_ignore_mouse_recursive(child)
	
	if card_rect != null:
		card_rect.color = card_color
	if name_label != null:
		name_label.text = character_name
	if health_bar != null:
		health_bar.setup(max_hp)
		
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
		card_rect.color = Color(1.0, 0.8, 0.2)
	elif is_selected_target:
		card_rect.color = Color(0.9, 0.2, 0.2)
	else:
		card_rect.color = original_color

func update_statuses(statuses: Array) -> void:
	if status_icon_container == null:
		return
		
	for child in status_icon_container.get_children():
		child.queue_free()
		
	for s in statuses:
		var status_type = str(s["type"])
		
		var icon = ColorRect.new()
		icon.custom_minimum_size = Vector2(12, 12) 
		
		var is_debuff = false
		if s.has("is_debuff"):
			is_debuff = s["is_debuff"]
			
		match status_type:
			"HASTE": icon.color = Color(0.2, 0.8, 1.0)
			"SLOW": icon.color = Color(0.8, 0.2, 0.8)
			"POISON": icon.color = Color(0.6, 0.2, 0.8)
			"DEF_UP": icon.color = Color(0.2, 0.8, 0.2)
			"DEF_DOWN": icon.color = Color(1.0, 0.2, 0.2)
			"SHIELD": icon.color = Color(1.0, 1.0, 1.0)
			"BANISHED": icon.color = Color(0.3, 0.0, 0.5)
			"MORALE": icon.color = Color(1.0, 0.8, 0.2)
			"JOY": icon.color = Color(1.0, 0.5, 0.0)
			_:
				if is_debuff: icon.color = Color(0.8, 0.2, 0.2)
				else: icon.color = Color(0.2, 0.8, 0.2)
		
		status_icon_container.add_child(icon)