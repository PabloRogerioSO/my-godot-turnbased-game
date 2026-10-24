package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"sort"
	"sync"

	"docka/backend/internal/combat"
	"docka/backend/internal/entity"
)

type VisualStep struct {
	Type   string `json:"type"`
	Target string `json:"target"`
	Value  int    `json:"value"`
	Text   string `json:"text"`
}

type BattleResponse struct {
	CurrentTick   int64                  `json:"current_tick"`
	Entities      []*entity.Entity       `json:"entities"`
	Timeline      []combat.TimelineEntry `json:"timeline"`
	Steps         []VisualStep           `json:"steps"`
	ActionLog     []string               `json:"action_log"`
	ActiveActorID string                 `json:"active_actor_id"`
	ActiveTeam    int                    `json:"active_team"`
	IsGameOver    bool                   `json:"is_game_over"`
	WinningTeam   int                    `json:"winning_team"`
}

type ActionRequest struct {
	SkillIndex int    `json:"skill_index"`
	TargetID   string `json:"target_id"`
}

type StartRequest struct {
	HeroIDs []string `json:"hero_ids"`
}

var (
	state         *combat.CombatState
	stateMu       sync.Mutex
	activeHeroIDs = []string{"h1", "h2", "h3"} // Remember the player's choice for resets
)

func makePlayerSkills() []entity.Skill {
	return []entity.Skill{
		{
			ID: "s1", Name: "Basic Attack", TargetType: "SINGLE", Cooldown: 0, CurrentCooldown: 0,
			Effects: []entity.SkillEffect{{Type: "DAMAGE", Multiplier: 1.0}},
		},
		{
			ID: "s2", Name: "Coordinated Strike", TargetType: "SINGLE", Cooldown: 2, CurrentCooldown: 0,
			Effects: []entity.SkillEffect{
				{Type: "DAMAGE", Multiplier: 1.2},
				{Type: "DUAL_ATTACK"},
			},
		},
		{
			ID: "s3", Name: "Explosion", TargetType: "SINGLE", Cooldown: 3, CurrentCooldown: 0,
			Effects: []entity.SkillEffect{
				{Type: "DAMAGE", Multiplier: 0.8},
				{Type: "EXTRA_DAMAGE", Multiplier: 0.3},
				{Type: "EXTRA_DAMAGE", Multiplier: 0.4},
			},
		},
	}
}

func makeMonsterSkills() []entity.Skill {
	return []entity.Skill{
		{
			ID: "s1", Name: "Basic Attack", TargetType: "SINGLE", Cooldown: 0, CurrentCooldown: 0,
			Effects: []entity.SkillEffect{{Type: "DAMAGE", Multiplier: 1.0}},
		},
		{
			ID: "s2", Name: "Venom Strike", TargetType: "SINGLE", Cooldown: 2, CurrentCooldown: 0,
			Effects: []entity.SkillEffect{
				{Type: "DAMAGE", Multiplier: 0.8},
				{Type: "DEBUFF", StatusType: "POISON", Value: 15, Duration: 3, IsDebuff: true},
			},
		},
		{
			ID: "s3", Name: "Execute Strike", TargetType: "SINGLE", Cooldown: 3, CurrentCooldown: 0,
			Effects: []entity.SkillEffect{
				{Type: "DAMAGE", Multiplier: 0.5},
				{Type: "EXECUTE", Value: 30},
			},
		},
	}
}

// Master Roster of Heroes
func getHeroByID(id string) *entity.Entity {
	skills := makePlayerSkills()
	switch id {
	case "h1":
		return &entity.Entity{ID: "h1", Name: "Knight", Team: entity.TeamPlayers, Alive: true,
			Stats:  entity.Stats{MaxHP: 250, HP: 250, Attack: 40, Defense: 20, BaseSpeed: 95.0, Effectiveness: 30, EffectResistance: 20},
			Skills: skills, Passives: []string{"COUNTER", "ANTIDOTE"}, StatusEffects: make([]entity.StatusEffect, 0)}
	case "h2":
		return &entity.Entity{ID: "h2", Name: "Archer", Team: entity.TeamPlayers, Alive: true,
			Stats:  entity.Stats{MaxHP: 100, HP: 100, Attack: 55, Defense: 10, BaseSpeed: 120.0, Effectiveness: 50, EffectResistance: 15},
			Skills: skills, Passives: make([]string, 0), StatusEffects: make([]entity.StatusEffect, 0)}
	case "h3":
		return &entity.Entity{ID: "h3", Name: "Mage", Team: entity.TeamPlayers, Alive: true,
			Stats:  entity.Stats{MaxHP: 100, HP: 100, Attack: 60, Defense: 8, BaseSpeed: 105.0, Effectiveness: 60, EffectResistance: 10},
			Skills: skills, Passives: make([]string, 0), StatusEffects: make([]entity.StatusEffect, 0)}
	case "h4": // NEW
		return &entity.Entity{ID: "h4", Name: "Cleric", Team: entity.TeamPlayers, Alive: true,
			Stats:  entity.Stats{MaxHP: 150, HP: 150, Attack: 30, Defense: 15, BaseSpeed: 100.0, Effectiveness: 40, EffectResistance: 30},
			Skills: skills, Passives: make([]string, 0), StatusEffects: make([]entity.StatusEffect, 0)}
	case "h5": // NEW
		return &entity.Entity{ID: "h5", Name: "Rogue", Team: entity.TeamPlayers, Alive: true,
			Stats:  entity.Stats{MaxHP: 110, HP: 110, Attack: 65, Defense: 8, BaseSpeed: 130.0, Effectiveness: 40, EffectResistance: 10},
			Skills: skills, Passives: make([]string, 0), StatusEffects: make([]entity.StatusEffect, 0)}
	case "h6": // NEW
		return &entity.Entity{ID: "h6", Name: "Paladin", Team: entity.TeamPlayers, Alive: true,
			Stats:  entity.Stats{MaxHP: 300, HP: 300, Attack: 35, Defense: 25, BaseSpeed: 85.0, Effectiveness: 20, EffectResistance: 40},
			Skills: skills, Passives: make([]string, 0), StatusEffects: make([]entity.StatusEffect, 0)}
	}
	return nil
}

func createCustomState(heroIDs []string) *combat.CombatState {
	newState := &combat.CombatState{
		ActionLog: make([]string, 0),
	}

	// Add selected heroes
	for _, id := range heroIDs {
		hero := getHeroByID(id)
		if hero != nil {
			newState.Entities = append(newState.Entities, hero)
		}
	}

	// Add standard monsters
	newState.Entities = append(newState.Entities, &entity.Entity{
		ID: "m1", Name: "Goblin", Team: entity.TeamMonsters, Alive: true,
		Stats:  entity.Stats{MaxHP: 80, HP: 80, Attack: 30, Defense: 5, BaseSpeed: 130.0, Effectiveness: 100, EffectResistance: 30},
		Skills: makeMonsterSkills(), Passives: make([]string, 0), StatusEffects: make([]entity.StatusEffect, 0),
	})
	newState.Entities = append(newState.Entities, &entity.Entity{
		ID: "m2", Name: "Slime", Team: entity.TeamMonsters, Alive: true,
		Stats:  entity.Stats{MaxHP: 100, HP: 100, Attack: 20, Defense: 15, BaseSpeed: 90.0, Effectiveness: 100, EffectResistance: 50},
		Skills: makeMonsterSkills(), Passives: make([]string, 0), StatusEffects: make([]entity.StatusEffect, 0),
	})
	newState.Entities = append(newState.Entities, &entity.Entity{
		ID: "m3", Name: "Wolf", Team: entity.TeamMonsters, Alive: true,
		Stats:  entity.Stats{MaxHP: 100, HP: 100, Attack: 45, Defense: 12, BaseSpeed: 115.0, Effectiveness: 100, EffectResistance: 20},
		Skills: makeMonsterSkills(), Passives: []string{"COUNTER"}, StatusEffects: make([]entity.StatusEffect, 0),
	})

	return newState
}

func getAIBestAction(actor *entity.Entity, entities []*entity.Entity) (int, *entity.Entity) {
	var enemies []*entity.Entity
	for _, e := range entities {
		if e.Alive && e.Team != actor.Team && !e.HasStatus("BANISHED") {
			enemies = append(enemies, e)
		}
	}
	if len(enemies) == 0 {
		return 0, nil
	}
	sort.Slice(enemies, func(i, j int) bool { return enemies[i].Stats.HP < enemies[j].Stats.HP })
	bestTarget := enemies[0]
	bestSkillIndex := 0
	maxDamageMultiplier := 0.0
	isExecutable := (bestTarget.Stats.HP <= (bestTarget.Stats.MaxHP * 30 / 100))

	for i, skill := range actor.Skills {
		if skill.CurrentCooldown > 0 {
			continue
		}
		hasExecute := false
		damageMultiplier := 0.0
		for _, effect := range skill.Effects {
			if effect.Type == "EXECUTE" {
				hasExecute = true
			}
			if effect.Type == "DAMAGE" {
				damageMultiplier = effect.Multiplier
			}
		}
		if hasExecute && isExecutable {
			return i, bestTarget
		}
		if !hasExecute && damageMultiplier > maxDamageMultiplier {
			maxDamageMultiplier = damageMultiplier
			bestSkillIndex = i
		}
	}
	return bestSkillIndex, bestTarget
}

func buildResponse(state *combat.CombatState, steps []VisualStep) BattleResponse {
	nextActor := state.PeekNextActor()
	resp := BattleResponse{
		CurrentTick: state.CurrentTick, Entities: state.Entities, Timeline: state.SimulateTimeline(),
		Steps: steps, ActionLog: state.ActionLog, IsGameOver: false,
	}
	if nextActor != nil {
		resp.ActiveActorID = nextActor.ID
		resp.ActiveTeam = int(nextActor.Team)
	} else {
		resp.IsGameOver = true
	}

	playersAlive := 0
	monstersAlive := 0
	for _, e := range state.Entities {
		if e.Alive && !e.HasStatus("BANISHED") {
			if e.Team == entity.TeamPlayers {
				playersAlive++
			} else {
				monstersAlive++
			}
		}
	}
	if playersAlive == 0 {
		resp.IsGameOver = true
		resp.WinningTeam = int(entity.TeamMonsters)
	} else if monstersAlive == 0 {
		resp.IsGameOver = true
		resp.WinningTeam = int(entity.TeamPlayers)
	}
	return resp
}

func main() {
	state = createCustomState(activeHeroIDs)

	http.HandleFunc("/api/start", func(w http.ResponseWriter, r *http.Request) {
		stateMu.Lock()
		defer stateMu.Unlock()

		var req StartRequest
		if r.Body != nil {
			json.NewDecoder(r.Body).Decode(&req)
		}

		if len(req.HeroIDs) > 0 {
			activeHeroIDs = req.HeroIDs
			state = createCustomState(activeHeroIDs)
		}

		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(buildResponse(state, []VisualStep{}))
	})

	http.HandleFunc("/api/battle", func(w http.ResponseWriter, r *http.Request) {
		stateMu.Lock()
		defer stateMu.Unlock()
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(buildResponse(state, []VisualStep{}))
	})

	http.HandleFunc("/api/reset", func(w http.ResponseWriter, r *http.Request) {
		stateMu.Lock()
		defer stateMu.Unlock()
		state = createCustomState(activeHeroIDs)
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(buildResponse(state, []VisualStep{}))
	})

	http.HandleFunc("/api/action", func(w http.ResponseWriter, r *http.Request) {
		stateMu.Lock()
		defer stateMu.Unlock()

		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json")

		var req ActionRequest
		if r.Body != nil {
			json.NewDecoder(r.Body).Decode(&req)
		}

		actor := state.GetNextActor()
		steps := []VisualStep{}

		if actor != nil {
			actor.TickStatusEffects()
			actor.TickCooldowns()

			if actor.Alive {
				skillIndex := req.SkillIndex
				var target *entity.Entity

				if actor.Team == entity.TeamMonsters {
					aiSkillIndex, aiTarget := getAIBestAction(actor, state.Entities)
					skillIndex = aiSkillIndex
					target = aiTarget
				}

				if skillIndex < 0 || skillIndex >= len(actor.Skills) {
					skillIndex = 0
				}
				skill := actor.Skills[skillIndex]

				if skill.CurrentCooldown > 0 {
					state.ActionLog = append(state.ActionLog, fmt.Sprintf("%s tried to use %s but it was on cooldown!", actor.Name, skill.Name))
					skillIndex = 0
					skill = actor.Skills[0]
				}

				var targets []*entity.Entity

				if actor.Team == entity.TeamMonsters {
					if target != nil {
						targets = append(targets, target)
					}
				} else {
					if skill.TargetType == "ALL_ENEMIES" {
						for _, e := range state.Entities {
							if e.Alive && e.Team != actor.Team && !e.HasStatus("BANISHED") {
								targets = append(targets, e)
							}
						}
					} else if skill.TargetType == "SELF" {
						targets = append(targets, actor)
					} else {
						if req.TargetID != "" {
							for _, e := range state.Entities {
								if e.ID == req.TargetID && e.Alive && e.Team != actor.Team && !e.HasStatus("BANISHED") {
									target = e
									break
								}
							}
						}
						if target == nil {
							for _, e := range state.Entities {
								if e.Alive && e.Team != actor.Team && !e.HasStatus("BANISHED") {
									target = e
									break
								}
							}
						}
						if target != nil {
							targets = append(targets, target)
						}
					}
				}

				if len(targets) > 0 {
					logMsg := fmt.Sprintf("%s uses %s!", actor.Name, skill.Name)
					state.ActionLog = append(state.ActionLog, logMsg)
					fmt.Println(logMsg)

					totalExtraDamageMultiplier := 0.0
					for _, effect := range skill.Effects {
						if effect.Type == "EXTRA_DAMAGE" {
							totalExtraDamageMultiplier += effect.Multiplier
						}
					}

					for _, target := range targets {
						for _, effect := range skill.Effects {
							if effect.Type == "BUFF" {
								target.StatusEffects = append(target.StatusEffects, entity.StatusEffect{Type: effect.StatusType, Value: effect.Value, Duration: effect.Duration, IsDebuff: false})
								state.ActionLog = append(state.ActionLog, fmt.Sprintf("  -> %s gains %s.", target.Name, effect.StatusType))
								steps = append(steps, VisualStep{Type: "BUFF", Target: target.ID, Text: effect.StatusType})
							}
							if effect.Type == "DEBUFF" {
								effRes := 100 + (target.Stats.EffectResistance - actor.Stats.Effectiveness)
								if effRes < 15 {
									effRes = 15
								}
								hitRoll := rand.Intn(100)
								if hitRoll >= effRes {
									if target.HasPassive("ANTIDOTE") {
										state.ActionLog = append(state.ActionLog, fmt.Sprintf("  -> %s's Antidote passive intercepts %s!", target.Name, effect.StatusType))
										steps = append(steps, VisualStep{Type: "INTERCEPT", Target: target.ID, Text: "Antidote!"})
									} else {
										target.StatusEffects = append(target.StatusEffects, entity.StatusEffect{Type: effect.StatusType, Value: effect.Value, Duration: effect.Duration, IsDebuff: true})
										state.ActionLog = append(state.ActionLog, fmt.Sprintf("  -> %s is afflicted with %s.", target.Name, effect.StatusType))
										steps = append(steps, VisualStep{Type: "DEBUFF", Target: target.ID, Text: effect.StatusType})
									}
								} else {
									state.ActionLog = append(state.ActionLog, fmt.Sprintf("  -> %s resisted %s.", target.Name, effect.StatusType))
									steps = append(steps, VisualStep{Type: "RESIST", Target: target.ID, Text: "Resisted"})
								}
							}
							if effect.Type == "DAMAGE" {
								baseDamage := float64(actor.Stats.Attack) * effect.Multiplier
								finalDamage := int(baseDamage) - target.GetEffectiveDefense()
								if finalDamage < 1 {
									finalDamage = 1
								}
								shieldAbsorbed, hpLost := target.TakeDamage(finalDamage)
								if shieldAbsorbed > 0 {
									state.ActionLog = append(state.ActionLog, fmt.Sprintf("  -> %s's shield absorbs %d damage.", target.Name, shieldAbsorbed))
									steps = append(steps, VisualStep{Type: "SHIELD_DAMAGE", Target: target.ID, Value: shieldAbsorbed})
								}
								if hpLost > 0 {
									state.ActionLog = append(state.ActionLog, fmt.Sprintf("  -> %s takes %d damage.", target.Name, hpLost))
									steps = append(steps, VisualStep{Type: "DAMAGE", Target: target.ID, Value: hpLost})
								}
							}
							if effect.Type == "EXECUTE" {
								if target.Alive && target.Stats.HP > 0 {
									threshold := (target.Stats.MaxHP * effect.Value) / 100
									if target.Stats.HP <= threshold {
										state.ActionLog = append(state.ActionLog, fmt.Sprintf("  -> %s EXECUTED %s!", actor.Name, target.Name))
										steps = append(steps, VisualStep{Type: "EXECUTE", Target: target.ID})
										target.Stats.HP = 0
										target.Alive = false
										target.IsUnrevivable = true
									}
								}
							}
						}

						if target.Stats.HP > 0 && target.Alive && target.HasPassive("COUNTER") {
							counterDmg := target.Stats.Attack / 2
							finalCounterDmg := counterDmg - actor.GetEffectiveDefense()
							if finalCounterDmg < 1 {
								finalCounterDmg = 1
							}
							actor.TakeDamage(finalCounterDmg)
							state.ActionLog = append(state.ActionLog, fmt.Sprintf("  -> %s counter-attacks %s!", target.Name, actor.Name))
							steps = append(steps, VisualStep{Type: "COUNTER", Target: actor.ID, Value: finalCounterDmg})
							if actor.Stats.HP <= 0 {
								actor.Stats.HP = 0
								actor.Alive = false
								state.ActionLog = append(state.ActionLog, fmt.Sprintf("  -> %s has died from the counter-attack!", actor.Name))
							}
						}

						hasDual := false
						for _, e := range skill.Effects {
							if e.Type == "DUAL_ATTACK" {
								hasDual = true
							}
						}
						if hasDual && target.Alive && target.Stats.HP > 0 {
							var allies []*entity.Entity
							for _, e := range state.Entities {
								if e.Alive && e.Team == actor.Team && e.ID != actor.ID {
									allies = append(allies, e)
								}
							}
							if len(allies) > 0 {
								ally := allies[rand.Intn(len(allies))]
								dualDmg := ally.Stats.Attack - target.GetEffectiveDefense()
								if dualDmg < 1 {
									dualDmg = 1
								}
								shieldAbsorbed, hpLost := target.TakeDamage(dualDmg)
								state.ActionLog = append(state.ActionLog, fmt.Sprintf("  -> %s joins for a DUAL ATTACK!", ally.Name))
								steps = append(steps, VisualStep{Type: "DUAL_ATTACK", Target: ally.ID, Text: "Dual Attack!"})
								if shieldAbsorbed > 0 {
									steps = append(steps, VisualStep{Type: "SHIELD_DAMAGE", Target: target.ID, Value: shieldAbsorbed})
								}
								if hpLost > 0 {
									state.ActionLog = append(state.ActionLog, fmt.Sprintf("  -> %s takes %d dual damage.", target.Name, hpLost))
									steps = append(steps, VisualStep{Type: "DAMAGE", Target: target.ID, Value: hpLost})
								}
							}
						}

						if target.Stats.HP <= 0 && target.Alive {
							target.Stats.HP = 0
							target.Alive = false
							state.ActionLog = append(state.ActionLog, fmt.Sprintf("  -> %s has died!", target.Name))
						}
					}

					if totalExtraDamageMultiplier > 0 {
						state.ActionLog = append(state.ActionLog, "  -> A massive EXPLOSION rocks the field!")
						for _, enemy := range state.Entities {
							if enemy.Alive && enemy.Team != actor.Team && !enemy.HasStatus("BANISHED") {
								extraDmg := int(float64(actor.Stats.Attack) * totalExtraDamageMultiplier)
								shieldAbsorbed, hpLost := enemy.TakeDamage(extraDmg)
								if shieldAbsorbed > 0 {
									steps = append(steps, VisualStep{Type: "SHIELD_DAMAGE", Target: enemy.ID, Value: shieldAbsorbed})
								}
								if hpLost > 0 {
									state.ActionLog = append(state.ActionLog, fmt.Sprintf("  -> %s takes %d extra damage!", enemy.Name, hpLost))
									steps = append(steps, VisualStep{Type: "EXTRA_DAMAGE", Target: enemy.ID, Value: hpLost})
								}
								if enemy.Stats.HP <= 0 && enemy.Alive {
									enemy.Stats.HP = 0
									enemy.Alive = false
									state.ActionLog = append(state.ActionLog, fmt.Sprintf("  -> %s has died from the explosion!", enemy.Name))
								}
							}
						}
					}
					if skill.Cooldown > 0 {
						actor.Skills[skillIndex].CurrentCooldown = skill.Cooldown
					}
				}
			}
		}
		state.ProcessEndOfTurn(actor.ID)
		json.NewEncoder(w).Encode(buildResponse(state, steps))
	})

	port := ":8080"
	fmt.Printf("Go Server running on http://localhost%s\n", port)
	log.Fatal(http.ListenAndServe(port, nil))
}
