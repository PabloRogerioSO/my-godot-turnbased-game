package entity

type Team uint8

const (
	TeamPlayers Team = iota
	TeamMonsters
)

type Stats struct {
	MaxHP, HP        int
	Attack, Defense  int
	BaseSpeed        float64
	SpeedModifier    float64
	Effectiveness    int
	EffectResistance int
}

type StatusEffect struct {
	Type        string `json:"type"`
	Value       int    `json:"value"`
	Duration    int    `json:"duration"`
	SourceID    string `json:"source_id"`
	IsRemovable bool   `json:"is_removable"`
	IsDebuff    bool   `json:"is_debuff"`
}

type SkillEffect struct {
	Type       string  `json:"type"`
	Multiplier float64 `json:"multiplier"`
	StatusType string  `json:"status_type"`
	Value      int     `json:"value"`
	Duration   int     `json:"duration"`
	IsDebuff   bool    `json:"is_debuff"`
}

type Skill struct {
	ID              string        `json:"id"`
	Name            string        `json:"name"`
	TargetType      string        `json:"target_type"`
	Cooldown        int           `json:"cooldown"`
	CurrentCooldown int           `json:"current_cooldown"`
	Effects         []SkillEffect `json:"effects"`
}

type Entity struct {
	ID            string         `json:"id"`
	Name          string         `json:"name"`
	Team          Team           `json:"team"`
	Stats         Stats          `json:"stats"`
	Alive         bool           `json:"alive"`
	IsUnrevivable bool           `json:"is_unrevivable"`
	Skills        []Skill        `json:"skills"`
	Passives      []string       `json:"passives"`
	StatusEffects []StatusEffect `json:"status_effects"`

	CombatReadiness float64 `json:"combat_readiness"`
	ReachedCRAtTick int64   `json:"reached_cr_at_tick"`
}

func (e *Entity) EffectiveSpeed() float64 {
	speed := e.Stats.BaseSpeed + e.Stats.SpeedModifier

	for _, status := range e.StatusEffects {
		if status.Type == "HASTE" {
			speed += float64(status.Value)
		} else if status.Type == "SLOW" {
			speed -= float64(status.Value)
		}
	}

	if speed < 1 {
		return 1
	}
	return speed
}

func (e *Entity) HasPassive(passiveName string) bool {
	for _, p := range e.Passives {
		if p == passiveName {
			return true
		}
	}
	return false
}

func (e *Entity) HasStatus(statusType string) bool {
	for _, s := range e.StatusEffects {
		if s.Type == statusType {
			return true
		}
	}
	return false
}

func (e *Entity) GetEffectiveDefense() int {
	def := e.Stats.Defense

	for _, status := range e.StatusEffects {
		if status.Type == "DEF_DOWN" {
			reduction := (def * status.Value) / 100
			def -= reduction
		} else if status.Type == "DEF_UP" {
			increase := (def * status.Value) / 100
			def += increase
		}
	}

	if def < 0 {
		return 0
	}
	return def
}

func (e *Entity) TakeDamage(amount int) (shieldAbsorbed int, hpLost int) {
	remainingDamage := amount
	newStatuses := make([]StatusEffect, 0)

	for _, status := range e.StatusEffects {
		if status.Type == "SHIELD" && remainingDamage > 0 {
			if status.Value >= remainingDamage {
				shieldAbsorbed += remainingDamage
				status.Value -= remainingDamage
				remainingDamage = 0
				if status.Value > 0 {
					newStatuses = append(newStatuses, status)
				}
			} else {
				shieldAbsorbed += status.Value
				remainingDamage -= status.Value
			}
		} else {
			newStatuses = append(newStatuses, status)
		}
	}

	e.StatusEffects = newStatuses

	if remainingDamage > 0 {
		e.Stats.HP -= remainingDamage
		hpLost = remainingDamage
		if e.Stats.HP < 0 {
			e.Stats.HP = 0
		}
	}

	return shieldAbsorbed, hpLost
}

func (e *Entity) Heal(amount int) int {
	if !e.Alive {
		return 0
	}

	e.Stats.HP += amount
	if e.Stats.HP > e.Stats.MaxHP {
		actualHeal := amount - (e.Stats.HP - e.Stats.MaxHP)
		e.Stats.HP = e.Stats.MaxHP
		return actualHeal
	}
	return amount
}

func (e *Entity) TickStatusEffects() {
	active := make([]StatusEffect, 0)
	for _, status := range e.StatusEffects {
		if e.HasStatus("BANISHED") {
			active = append(active, status)
			continue
		}

		status.Duration--
		if status.Duration > 0 {
			active = append(active, status)
		}
	}
	e.StatusEffects = active
}

func (e *Entity) TickCooldowns() {
	for i := range e.Skills {
		if e.Skills[i].CurrentCooldown > 0 {
			e.Skills[i].CurrentCooldown--
		}
	}
}
