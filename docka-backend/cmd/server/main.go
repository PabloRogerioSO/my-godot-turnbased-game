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
	activeHeroIDs = []string{"h1", "h2", "h3"}
)

type Reaction struct {
	Attacker *entity.Entity
	Target   *entity.Entity
	Type     string
}

func makeClericSkills() []entity.Skill {
	return []entity.Skill{
		{
			ID: "s1", Name: "Basic Attack", TargetType: "SINGLE_ENEMY", Cooldown: 0, CurrentCooldown: 0,
			Effects: []entity.SkillEffect{{Type: "DAMAGE", Multiplier: 1.0}},
		},
		{
			ID: "s2", Name: "Resurrection", TargetType: "SINGLE_ALLY", Cooldown: 3, CurrentCooldown: 0,
			Effects: []entity.SkillEffect{
				{Type: "REVIVE", Value: 30, EffectTargetType: "SINGLE_ALLY"}, // Revives with 30% HP
			},
		},
		{
			ID: "s3", Name: "Debug Stack", TargetType: "SELF", Cooldown: 0, CurrentCooldown: 0,
			Effects: []entity.SkillEffect{
				{Type: "BUFF", StatusType: "IMMORTALITY", Duration: 2, EffectTargetType: "SELF"}, // Test Immortality
				{Type: "SHIELD", Value: 50, Duration: 10, EffectTargetType: "SELF"},
			},
		},
	}
}

func makePlayerSkills() []entity.Skill {
	return []entity.Skill{
		{
			ID: "s1", Name: "Basic Attack", TargetType: "SINGLE_ENEMY", Cooldown: 0, CurrentCooldown: 0,
			Effects: []entity.SkillEffect{{Type: "DAMAGE", Multiplier: 1.0}},
		},
		{
			ID: "s2", Name: "Tactical Strike", TargetType: "SINGLE_ENEMY", Cooldown: 2, CurrentCooldown: 0,
			Effects: []entity.SkillEffect{
				{Type: "DAMAGE", Multiplier: 1.5},
				{Type: "EXTRA_DAMAGE", Multiplier: 0.4, EffectTargetType: "ALL_ENEMIES"},
				{Type: "DEBUFF", StatusType: "DEF_DOWN", Value: 30, Duration: 2, IsDebuff: true, EffectTargetType: "ALL_ENEMIES"},
			},
		},
		{
			ID: "s3", Name: "Debug Stack", TargetType: "SELF", Cooldown: 0, CurrentCooldown: 0,
			Effects: []entity.SkillEffect{
				{Type: "BUFF", StatusType: "HASTE", Duration: 10, EffectTargetType: "SELF"},
				{Type: "BUFF", StatusType: "MORALE", Duration: 10, EffectTargetType: "SELF"},
			},
		},
	}
}

func makeMonsterSkills() []entity.Skill {
	return []entity.Skill{
		{
			ID: "s1", Name: "Basic Attack", TargetType: "SINGLE_ENEMY", Cooldown: 0, CurrentCooldown: 0,
			Effects: []entity.SkillEffect{{Type: "DAMAGE", Multiplier: 1.0}},
		},
		{
			ID: "s2", Name: "Venom Strike", TargetType: "SINGLE_ENEMY", Cooldown: 2, CurrentCooldown: 0,
			Effects: []entity.SkillEffect{
				{Type: "DAMAGE", Multiplier: 0.8},
				{Type: "DEBUFF", StatusType: "POISON", Value: 15, Duration: 3, IsDebuff: true},
			},
		},
		{
			ID: "s3", Name: "Extinct Execute", TargetType: "SINGLE_ENEMY", Cooldown: 3, CurrentCooldown: 0,
			Effects: []entity.SkillEffect{
				{Type: "DAMAGE", Multiplier: 0.5},
				{Type: "EXECUTE", Value: 50}, // Kills if below 50%, sets Unrevivable
			},
		},
	}
}

func getHeroByID(id string) *entity.Entity {
	switch id {
	case "h1":
		return &entity.Entity{ID: "h1", Name: "Knight", Team: entity.TeamPlayers, Alive: true,
			Stats:  entity.Stats{MaxHP: 250, HP: 250, Attack: 40, Defense: 20, BaseSpeed: 95.0, Effectiveness: 30, EffectResistance: 20},
			Skills: makePlayerSkills(), Passives: []entity.Passive{{Type: "COUNTER", Chance: 50.0}}, StatusEffects: make([]entity.StatusEffect, 0), SkillCounters: make(map[string]int)}
	case "h2":
		return &entity.Entity{ID: "h2", Name: "Archer", Team: entity.TeamPlayers, Alive: true,
			Stats:  entity.Stats{MaxHP: 100, HP: 100, Attack: 55, Defense: 10, BaseSpeed: 120.0, Effectiveness: 50, EffectResistance: 15},
			Skills: makePlayerSkills(), Passives: []entity.Passive{{Type: "ALLY_COUNTER", Chance: 30.0}, {Type: "ALLY_EXTRA_DAMAGE", Value: 0.3}}, StatusEffects: make([]entity.StatusEffect, 0), SkillCounters: make(map[string]int)}
	case "h3":
		return &entity.Entity{ID: "h3", Name: "Mage", Team: entity.TeamPlayers, Alive: true,
			Stats:  entity.Stats{MaxHP: 100, HP: 100, Attack: 60, Defense: 8, BaseSpeed: 105.0, Effectiveness: 60, EffectResistance: 10},
			Skills: makePlayerSkills(), Passives: make([]entity.Passive, 0), StatusEffects: make([]entity.StatusEffect, 0), SkillCounters: make(map[string]int)}
	case "h4":
		return &entity.Entity{ID: "h4", Name: "Cleric", Team: entity.TeamPlayers, Alive: true,
			Stats:  entity.Stats{MaxHP: 150, HP: 150, Attack: 30, Defense: 15, BaseSpeed: 100.0, Effectiveness: 40, EffectResistance: 30},
			Skills: makeClericSkills(), Passives: make([]entity.Passive, 0), StatusEffects: make([]entity.StatusEffect, 0), SkillCounters: make(map[string]int)}
	case "h7":
		return &entity.Entity{ID: "h7", Name: "Anchor Knight", Team: entity.TeamPlayers, Alive: true,
			Stats:  entity.Stats{MaxHP: 400, HP: 400, Attack: 30, Defense: 30, BaseSpeed: 90.0, Effectiveness: 20, EffectResistance: 30},
			Skills: makePlayerSkills(), Passives: make([]entity.Passive, 0), StatusEffects: make([]entity.StatusEffect, 0), SkillCounters: make(map[string]int)}
	}
	return nil
}

func createCustomState(heroIDs []string) *combat.CombatState {
	newState := &combat.CombatState{
		ActionLog: make([]string, 0),
	}

	for _, id := range heroIDs {
		hero := getHeroByID(id)
		if hero != nil {
			newState.Entities = append(newState.Entities, hero)
		}
	}

	newState.Entities = append(newState.Entities, &entity.Entity{
		ID: "m1", Name: "Goblin", Team: entity.TeamMonsters, Alive: true,
		Stats:  entity.Stats{MaxHP: 80, HP: 80, Attack: 30, Defense: 5, BaseSpeed: 130.0, Effectiveness: 100, EffectResistance: 30},
		Skills: makeMonsterSkills(), Passives: make([]entity.Passive, 0), StatusEffects: make([]entity.StatusEffect, 0), SkillCounters: make(map[string]int),
	})
	newState.Entities = append(newState.Entities, &entity.Entity{
		ID: "m2", Name: "Slime", Team: entity.TeamMonsters, Alive: true,
		Stats:  entity.Stats{MaxHP: 100, HP: 100, Attack: 20, Defense: 15, BaseSpeed: 90.0, Effectiveness: 100, EffectResistance: 50},
		Skills: makeMonsterSkills(), Passives: make([]entity.Passive, 0), StatusEffects: make([]entity.StatusEffect, 0), SkillCounters: make(map[string]int),
	})
	newState.Entities = append(newState.Entities, &entity.Entity{
		ID: "m3", Name: "Wolf", Team: entity.TeamMonsters, Alive: true,
		Stats:  entity.Stats{MaxHP: 100, HP: 100, Attack: 45, Defense: 12, BaseSpeed: 115.0, Effectiveness: 100, EffectResistance: 20},
		Skills: makeMonsterSkills(), Passives: []entity.Passive{{Type: "COUNTER", Chance: 100.0}}, StatusEffects: make([]entity.StatusEffect, 0), SkillCounters: make(map[string]int),
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

	for i, skill := range actor.Skills {
		if skill.CurrentCooldown > 0 || skill.TargetType != "SINGLE_ENEMY" {
			continue
		}

		damageMultiplier := 0.0
		for _, effect := range skill.Effects {
			if effect.Type == "DAMAGE" {
				damageMultiplier = effect.Multiplier
			}
		}
		if damageMultiplier > maxDamageMultiplier {
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

func getEffectTargets(actor *entity.Entity, skillTargetType string, effect *entity.SkillEffect, entities []*entity.Entity, requestedTargetID string) []*entity.Entity {
	targetType := skillTargetType
	if effect.EffectTargetType != "" {
		targetType = effect.EffectTargetType
	}

	var targets []*entity.Entity
	switch targetType {
	case "ALL_ENEMIES":
		for _, e := range entities {
			if e.Alive && e.Team != actor.Team && !e.HasStatus("BANISHED") {
				targets = append(targets, e)
			}
		}
	case "ALL_ALLIES":
		for _, e := range entities {
			if e.Alive && e.Team == actor.Team && !e.HasStatus("BANISHED") {
				targets = append(targets, e)
			}
		}
	case "SELF":
		targets = append(targets, actor)
	case "SINGLE_ALLY":
		// For revives, we must allow targeting dead allies
		if requestedTargetID != "" {
			for _, e := range entities {
				if e.ID == requestedTargetID && e.Team == actor.Team {
					return []*entity.Entity{e}
				}
			}
		}
	case "SINGLE_ENEMY":
		if requestedTargetID != "" {
			for _, e := range entities {
				if e.ID == requestedTargetID && e.Alive && e.Team != actor.Team && !e.HasStatus("BANISHED") {
					return []*entity.Entity{e}
				}
			}
		}
		for _, e := range entities {
			if e.Alive && e.Team != actor.Team && !e.HasStatus("BANISHED") {
				return []*entity.Entity{e}
			}
		}
	}
	return targets
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
					skillIndex = 0
					skill = actor.Skills[0]
				}

				var requestedID string
				if target != nil {
					requestedID = target.ID
				} else {
					requestedID = req.TargetID
				}

				primaryTargets := getEffectTargets(actor, skill.TargetType, &entity.SkillEffect{}, state.Entities, requestedID)

				if len(primaryTargets) > 0 {
					logMsg := fmt.Sprintf("%s uses %s!", actor.Name, skill.Name)
					state.ActionLog = append(state.ActionLog, logMsg)
					fmt.Println(logMsg)

					currentStacks := actor.SkillCounters[skill.ID]

					reactionQueue := []Reaction{}
					extraDamagePool := make(map[string]int)

					for _, effect := range skill.Effects {
						effectTargets := getEffectTargets(actor, skill.TargetType, &effect, state.Entities, requestedID)

						for _, targetEntity := range effectTargets {
							// Skip if target died from a previous effect (unless it's a REVIVE)
							if !targetEntity.Alive && effect.Type != "REVIVE" {
								continue
							}

							damageDealtThisAction := 0

							if effect.Type == "EXTRA_DAMAGE" {
								extraDamagePool[targetEntity.ID] += int(float64(actor.Stats.Attack) * effect.Multiplier)
								continue
							}

							if effect.Type == "REVIVE" {
								if !targetEntity.Alive && !targetEntity.IsUnrevivable {
									targetEntity.Alive = true
									targetEntity.Stats.HP = int(float64(targetEntity.Stats.MaxHP) * (float64(effect.Value) / 100.0))
									state.ActionLog = append(state.ActionLog, fmt.Sprintf("  -> %s has been REVIVED with %d HP!", targetEntity.Name, targetEntity.Stats.HP))
									steps = append(steps, VisualStep{Type: "REVIVE", Target: targetEntity.ID, Value: targetEntity.Stats.HP})
								} else if targetEntity.IsUnrevivable {
									state.ActionLog = append(state.ActionLog, fmt.Sprintf("  -> %s cannot be revived (Extinct)!", targetEntity.Name))
								}
								continue
							}

							if effect.Type == "BUFF" {
								targetEntity.StatusEffects = append(targetEntity.StatusEffects, entity.StatusEffect{Type: effect.StatusType, Value: effect.Value, Duration: effect.Duration, IsDebuff: false})
								state.ActionLog = append(state.ActionLog, fmt.Sprintf("  -> %s gains %s.", targetEntity.Name, effect.StatusType))
								steps = append(steps, VisualStep{Type: "BUFF", Target: targetEntity.ID, Text: effect.StatusType})
							}
							if effect.Type == "SHIELD" {
								targetEntity.StatusEffects = append(targetEntity.StatusEffects, entity.StatusEffect{Type: "SHIELD", Value: effect.Value, Duration: effect.Duration, IsDebuff: false})
								state.ActionLog = append(state.ActionLog, fmt.Sprintf("  -> %s gains a shield.", targetEntity.Name))
								steps = append(steps, VisualStep{Type: "SHIELD_ADD", Target: targetEntity.ID, Value: effect.Value})
							}
							if effect.Type == "HEAL" {
								actualHeal := targetEntity.Heal(effect.Value)
								if actualHeal > 0 {
									state.ActionLog = append(state.ActionLog, fmt.Sprintf("  -> %s heals for %d HP.", targetEntity.Name, actualHeal))
									steps = append(steps, VisualStep{Type: "HEAL", Target: targetEntity.ID, Value: actualHeal})
								}
							}
							if effect.Type == "DEBUFF" {
								effRes := 100 + (targetEntity.Stats.EffectResistance - actor.Stats.Effectiveness)
								if effRes < 15 {
									effRes = 15
								}
								hitRoll := rand.Intn(100)
								if hitRoll >= effRes {
									targetEntity.StatusEffects = append(targetEntity.StatusEffects, entity.StatusEffect{Type: effect.StatusType, Value: effect.Value, Duration: effect.Duration, IsDebuff: true})
									state.ActionLog = append(state.ActionLog, fmt.Sprintf("  -> %s is afflicted with %s.", targetEntity.Name, effect.StatusType))
									steps = append(steps, VisualStep{Type: "DEBUFF", Target: targetEntity.ID, Text: effect.StatusType})
								} else {
									state.ActionLog = append(state.ActionLog, fmt.Sprintf("  -> %s resisted %s.", targetEntity.Name, effect.StatusType))
									steps = append(steps, VisualStep{Type: "RESIST", Target: targetEntity.ID, Text: "Resisted"})
								}
							}
							if effect.Type == "DAMAGE" {
								baseDamage := (float64(actor.Stats.Attack) * effect.Multiplier) + (float64(actor.Stats.MaxHP) * effect.HPMultiplier)

								if currentStacks > 0 && skill.StackMultiplier > 0 {
									baseDamage = baseDamage * (1.0 + (float64(currentStacks) * skill.StackMultiplier))
								}

								defense := float64(targetEntity.GetEffectiveDefense())
								if effect.Condition == "TARGET_HAS_SHIELD" && targetEntity.HasStatus("SHIELD") {
									defense = defense * (1.0 - (float64(effect.DefPenetration) / 100.0))
								}

								finalDamage := int(baseDamage) - int(defense)
								if finalDamage < 1 {
									finalDamage = 1
								}

								shieldAbsorbed, hpLost := targetEntity.TakeDamage(finalDamage)
								damageDealtThisAction += hpLost

								if shieldAbsorbed > 0 {
									state.ActionLog = append(state.ActionLog, fmt.Sprintf("  -> %s's shield absorbs %d damage.", targetEntity.Name, shieldAbsorbed))
									steps = append(steps, VisualStep{Type: "SHIELD_DAMAGE", Target: targetEntity.ID, Value: shieldAbsorbed})
								}
								if hpLost > 0 {
									state.ActionLog = append(state.ActionLog, fmt.Sprintf("  -> %s takes %d damage.", targetEntity.Name, hpLost))
									steps = append(steps, VisualStep{Type: "DAMAGE", Target: targetEntity.ID, Value: hpLost})
								}

								// Check for Death (Immortality already handled inside TakeDamage)
								if targetEntity.Stats.HP <= 0 && targetEntity.Alive {
									targetEntity.Stats.HP = 0
									targetEntity.Alive = false
									state.ActionLog = append(state.ActionLog, fmt.Sprintf("  -> %s has died!", targetEntity.Name))
								}

								// Only trigger counters if the target is still alive
								if targetEntity.Alive {
									if p := targetEntity.GetPassive("COUNTER"); p != nil {
										if rand.Float64()*100 <= p.Chance {
											reactionQueue = append(reactionQueue, Reaction{Attacker: targetEntity, Target: actor, Type: "COUNTER"})
										}
									}

									for _, ally := range state.Entities {
										if ally.Alive && ally.Team == targetEntity.Team && ally.ID != targetEntity.ID {
											if p := ally.GetPassive("ALLY_COUNTER"); p != nil {
												if rand.Float64()*100 <= p.Chance {
													reactionQueue = append(reactionQueue, Reaction{Attacker: ally, Target: actor, Type: "ALLY_COUNTER"})
												}
											}
										}
									}
								}
							}
							if effect.Type == "LIFESTEAL" {
								if damageDealtThisAction > 0 {
									healAmount := int(float64(damageDealtThisAction) * (float64(effect.Value) / 100.0))
									actualHeal := actor.Heal(healAmount)
									if actualHeal > 0 {
										state.ActionLog = append(state.ActionLog, fmt.Sprintf("  -> %s drains %d HP.", actor.Name, actualHeal))
										steps = append(steps, VisualStep{Type: "HEAL", Target: actor.ID, Value: actualHeal})
									}
								}
							}
							if effect.Type == "EXECUTE" {
								if targetEntity.Alive && targetEntity.Stats.HP > 0 {
									threshold := (targetEntity.Stats.MaxHP * effect.Value) / 100
									if targetEntity.Stats.HP <= threshold {
										state.ActionLog = append(state.ActionLog, fmt.Sprintf("  -> %s EXECUTED %s!", actor.Name, targetEntity.Name))
										steps = append(steps, VisualStep{Type: "EXECUTE", Target: targetEntity.ID})
										targetEntity.Stats.HP = 0
										targetEntity.Alive = false
										targetEntity.IsUnrevivable = true // Prevents resurrection
										state.ActionLog = append(state.ActionLog, fmt.Sprintf("  -> %s is now Extinct and cannot be revived.", targetEntity.Name))
									}
								}
							}
						}

						if effect.Type == "DAMAGE" {
							for _, ally := range state.Entities {
								if ally.Alive && ally.Team == actor.Team && ally.ID != actor.ID {
									if p := ally.GetPassive("ALLY_EXTRA_DAMAGE"); p != nil {
										for _, enemy := range state.Entities {
											if enemy.Alive && enemy.Team != actor.Team {
												extraDamagePool[enemy.ID] += int(float64(ally.Stats.Attack) * p.Value)
											}
										}
									}
								}
							}
						}
					}

					if skill.MaxStacks > 0 && currentStacks < skill.MaxStacks {
						actor.SkillCounters[skill.ID]++
						state.ActionLog = append(state.ActionLog, fmt.Sprintf("  -> %s's Anchor Strike is now at %d stacks!", actor.Name, actor.SkillCounters[skill.ID]))
					}

					for len(reactionQueue) > 0 {
						react := reactionQueue[0]
						reactionQueue = reactionQueue[1:]

						if react.Attacker.Alive && react.Target.Alive {
							if react.Type == "COUNTER" {
								state.ActionLog = append(state.ActionLog, fmt.Sprintf("  -> %s counter-attacks %s!", react.Attacker.Name, react.Target.Name))
							} else if react.Type == "ALLY_COUNTER" {
								state.ActionLog = append(state.ActionLog, fmt.Sprintf("  -> %s counter-attacks to protect an ally!", react.Attacker.Name))
							}

							counterDmg := react.Attacker.Stats.Attack / 2
							finalCounterDmg := counterDmg - react.Target.GetEffectiveDefense()
							if finalCounterDmg < 1 {
								finalCounterDmg = 1
							}

							shieldAbsorbed, hpLost := react.Target.TakeDamage(finalCounterDmg)

							if shieldAbsorbed > 0 {
								steps = append(steps, VisualStep{Type: "SHIELD_DAMAGE", Target: react.Target.ID, Value: shieldAbsorbed})
							}
							if hpLost > 0 {
								state.ActionLog = append(state.ActionLog, fmt.Sprintf("     -> %s takes %d damage.", react.Target.Name, hpLost))
								steps = append(steps, VisualStep{Type: "COUNTER", Target: react.Target.ID, Value: hpLost})
							}

							if react.Target.Stats.HP <= 0 {
								react.Target.Stats.HP = 0
								react.Target.Alive = false
								state.ActionLog = append(state.ActionLog, fmt.Sprintf("     -> %s has died from the counter-attack!", react.Target.Name))
							}
						}
					}

					if len(extraDamagePool) > 0 {
						state.ActionLog = append(state.ActionLog, "  -> A massive EXPLOSION rocks the field!")
						for targetID, totalDmg := range extraDamagePool {
							for _, e := range state.Entities {
								if e.ID == targetID && e.Alive {
									shieldAbsorbed, hpLost := e.TakeDamage(totalDmg)
									if shieldAbsorbed > 0 {
										steps = append(steps, VisualStep{Type: "SHIELD_DAMAGE", Target: e.ID, Value: shieldAbsorbed})
									}
									if hpLost > 0 {
										state.ActionLog = append(state.ActionLog, fmt.Sprintf("  -> %s takes %d extra damage!", e.Name, hpLost))
										steps = append(steps, VisualStep{Type: "EXTRA_DAMAGE", Target: e.ID, Value: hpLost})
									}
									if e.Stats.HP <= 0 {
										e.Stats.HP = 0
										e.Alive = false
										state.ActionLog = append(state.ActionLog, fmt.Sprintf("  -> %s has died from the explosion!", e.Name))
									}
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
