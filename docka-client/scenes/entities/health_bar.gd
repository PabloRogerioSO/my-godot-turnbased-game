extends Control

@onready var bar: ProgressBar = $Bar
@onready var shield_bar: ProgressBar = $ShieldBar
@onready var hp_label: Label = $HPLabel

var current_hp: int = 100
var max_hp: int = 100
var current_shield: int = 0

func setup(new_max_hp: int, new_current_hp: int = -1, new_shield: int = 0) -> void:
	max_hp = new_max_hp
	if new_current_hp < 0:
		current_hp = max_hp
	else:
		current_hp = new_current_hp
		
	current_shield = new_shield
	
	# Update HP Bar
	bar.max_value = max_hp
	bar.value = current_hp
	
	# Update Shield Bar
	# We use a visual ratio so the shield bar doesn't get out of bounds if shield > max_hp
	shield_bar.max_value = max_hp
	shield_bar.value = current_shield
	
	if current_shield > 0:
		shield_bar.visible = true
		hp_label.text = str(current_hp) + " / " + str(max_hp) + " (+" + str(current_shield) + ")"
	else:
		shield_bar.visible = false
		hp_label.text = str(current_hp) + " / " + str(max_hp)
