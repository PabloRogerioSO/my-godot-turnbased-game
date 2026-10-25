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

func GetFactionMultiplier(attackerFaction, defenderFaction string) float64 {
	if attackerFaction == "LOJINHA" {
		if defenderFaction == "CHAOS" || defenderFaction == "PHYSICAL" {
			return 1.5
		}
		return 1.0
	}
	if attackerFaction == "CHAOS" && defenderFaction == "LOJINHA" {
		return 0.75
	}

	if (attackerFaction == "META" && defenderFaction == "CHAOS") ||
		(attackerFaction == "CHAOS" && defenderFaction == "STRUCTURE") ||
		(attackerFaction == "STRUCTURE" && defenderFaction == "TOXIC") ||
		(attackerFaction == "TOXIC" && defenderFaction == "PHYSICAL") ||
		(attackerFaction == "PHYSICAL" && defenderFaction == "META") {
		return 1.5
	}
	if (attackerFaction == "CHAOS" && defenderFaction == "META") ||
		(attackerFaction == "STRUCTURE" && defenderFaction == "CHAOS") ||
		(attackerFaction == "TOXIC" && defenderFaction == "STRUCTURE") ||
		(attackerFaction == "PHYSICAL" && defenderFaction == "TOXIC") ||
		(attackerFaction == "META" && defenderFaction == "PHYSICAL") {
		return 0.75
	}
	return 1.0
}

func CheckHit(actor, target *entity.Entity) (bool, bool) {
	baseHit := actor.GetHitChance()
	if actor.HasStatus("BLIND") {
		baseHit -= 50.0
	}
	isDisadvantaged := GetFactionMultiplier(actor.Faction, target.Faction) == 0.75
	if isDisadvantaged {
		baseHit -= 50.0
	}

	evasion := target.Stats.Evasion
	if evasion > 90.0 {
		evasion = 90.0
	}

	finalChance := baseHit - evasion
	if finalChance < 0 {
		finalChance = 0
	}

	roll := rand.Float64() * 100.0
	return roll <= finalChance, isDisadvantaged
}

func CheckEffect(actor, target *entity.Entity, effect *entity.SkillEffect) bool {
	skillChance := effect.EffectChance
	if skillChance == 0 {
		skillChance = 100.0
	}
	if rand.Float64()*100.0 > skillChance {
		return false
	}

	res := 100.0 + float64(target.Stats.EffectResistance-actor.Stats.Effectiveness)
	if res < 15.0 {
		res = 15.0
	}

	roll := rand.Float64() * 100.0
	return roll >= res
}

func makeSofaSkills() []entity.Skill {
	return []entity.Skill{
		{ID: "s1", Name: "Keyboard Smash", TargetType: "SINGLE_ENEMY", Cooldown: 0, Effects: []entity.SkillEffect{{Type: "DAMAGE", Multiplier: 1.0}}},
		{ID: "s2", Name: "Patch Notes", TargetType: "ALL_ENEMIES", Cooldown: 2, Effects: []entity.SkillEffect{{Type: "DAMAGE", Multiplier: 0.8}, {Type: "DEBUFF", StatusType: "BLIND", Duration: 2, IsDebuff: true, EffectChance: 80}}},
		{ID: "s3", Name: "Meta Call", TargetType: "SINGLE_ALLY", Cooldown: 3, Effects: []entity.SkillEffect{{Type: "HEAL", Value: 50}, {Type: "BUFF", StatusType: "HASTE", Value: 30, Duration: 2}}},
	}
}

func makeAssemelhadoSkills() []entity.Skill {
	return []entity.Skill{
		{ID: "s1", Name: "Glitch Out", TargetType: "SINGLE_ENEMY", Cooldown: 0, Effects: []entity.SkillEffect{{Type: "DAMAGE", Multiplier: 1.0}}},
		{ID: "s2", Name: "Frenzy", TargetType: "SINGLE_ENEMY", Cooldown: 2, Effects: []entity.SkillEffect{{Type: "DAMAGE", Multiplier: 1.5}}},
		{ID: "s3", Name: "Mental Break", TargetType: "SINGLE_ENEMY", Cooldown: 3, Effects: []entity.SkillEffect{{Type: "DAMAGE", Multiplier: 0.5}, {Type: "EXECUTE", Value: 30}}},
	}
}

func makeBolsistaSkills() []entity.Skill {
	return []entity.Skill{
		{ID: "s1", Name: "Heavy Backpack", TargetType: "SINGLE_ENEMY", Cooldown: 0, Effects: []entity.SkillEffect{{Type: "DAMAGE", Multiplier: 1.0}}},
		{ID: "s2", Name: "Grind Mode", TargetType: "SELF", Cooldown: 3, Effects: []entity.SkillEffect{{Type: "BUFF", StatusType: "DEF_UP", Value: 50, Duration: 2}}},
		{ID: "s3", Name: "Study Group", TargetType: "ALL_ALLIES", Cooldown: 3, Effects: []entity.SkillEffect{{Type: "HEAL", Value: 30}}},
	}
}

func makeSinucaSkills() []entity.Skill {
	return []entity.Skill{
		{ID: "s1", Name: "Cue Strike", TargetType: "SINGLE_ENEMY", Cooldown: 0, Effects: []entity.SkillEffect{{Type: "DAMAGE", Multiplier: 1.0}}},
		{ID: "s2", Name: "Trick Shot", TargetType: "SINGLE_ENEMY", Cooldown: 2, Effects: []entity.SkillEffect{{Type: "DAMAGE", Multiplier: 1.8}}},
		{ID: "s3", Name: "Break Shot", TargetType: "ALL_ENEMIES", Cooldown: 3, Effects: []entity.SkillEffect{{Type: "DAMAGE", Multiplier: 1.2}}},
	}
}

func makeLojinhaSkills() []entity.Skill {
	return []entity.Skill{
		{ID: "s1", Name: "Ruler's Decree", TargetType: "SINGLE_ENEMY", Cooldown: 0, Effects: []entity.SkillEffect{{Type: "DAMAGE", Multiplier: 1.0}}},
		{ID: "s2", Name: "Call Security", TargetType: "SINGLE_ENEMY", Cooldown: 2, Effects: []entity.SkillEffect{{Type: "DAMAGE", Multiplier: 1.2}, {Type: "DEBUFF", StatusType: "STUN", Duration: 1, IsDebuff: true, EffectChance: 60}}},
		{ID: "s3", Name: "Price Gouge", TargetType: "SINGLE_ENEMY", Cooldown: 3, Effects: []entity.SkillEffect{{Type: "DAMAGE", Multiplier: 2.0}}},
	}
}

func makeMonsterSkills() []entity.Skill {
	return []entity.Skill{
		{ID: "s1", Name: "Basic Attack", TargetType: "SINGLE_ENEMY", Cooldown: 0, Effects: []entity.SkillEffect{{Type: "DAMAGE", Multiplier: 1.0}}},
		{ID: "s2", Name: "Venom Strike", TargetType: "SINGLE_ENEMY", Cooldown: 2, Effects: []entity.SkillEffect{{Type: "DAMAGE", Multiplier: 0.8}, {Type: "DEBUFF", StatusType: "POISON", Value: 15, Duration: 3, IsDebuff: true, EffectChance: 75}}},
		{ID: "s3", Name: "Execute Strike", TargetType: "SINGLE_ENEMY", Cooldown: 3, Effects: []entity.SkillEffect{{Type: "DAMAGE", Multiplier: 0.5}, {Type: "EXECUTE", Value: 30}}},
	}
}

func getHeroByID(id string) *entity.Entity {
	switch id {
	case "h1":
		return &entity.Entity{ID: "h1", Name: "Sofa Cigs", Team: entity.TeamPlayers, Faction: "META", Subclass: "CIGS", Alive: true,
			Stats:         entity.Stats{MaxHP: 250, HP: 250, Attack: 40, Defense: 20, BaseSpeed: 95.0, Effectiveness: 30, EffectResistance: 20, HitChance: 100, Evasion: 0},
			Skills:        makeSofaSkills(),
			StatusEffects: make([]entity.StatusEffect, 0),
			SkillCounters: make(map[string]int)}
	case "h2":
		return &entity.Entity{ID: "h2", Name: "Assemelhado Verde", Team: entity.TeamPlayers, Faction: "CHAOS", Subclass: "VERDE", Alive: true,
			Stats:         entity.Stats{MaxHP: 100, HP: 100, Attack: 55, Defense: 10, BaseSpeed: 120.0, Effectiveness: 50, EffectResistance: 15, HitChance: 100, Evasion: 30},
			Skills:        makeAssemelhadoSkills(),
			StatusEffects: make([]entity.StatusEffect, 0),
			SkillCounters: make(map[string]int)}
	case "h3":
		return &entity.Entity{ID: "h3", Name: "Bolsista Ceva", Team: entity.TeamPlayers, Faction: "STRUCTURE", Subclass: "CEVA", Alive: true,
			Stats:         entity.Stats{MaxHP: 150, HP: 150, Attack: 30, Defense: 15, BaseSpeed: 100.0, Effectiveness: 40, EffectResistance: 30, HitChance: 100, Evasion: 0},
			Skills:        makeBolsistaSkills(),
			StatusEffects: make([]entity.StatusEffect, 0),
			SkillCounters: make(map[string]int)}
	case "h4":
		return &entity.Entity{ID: "h4", Name: "Sinuca Truco", Team: entity.TeamPlayers, Faction: "PHYSICAL", Subclass: "TRUCO", Alive: true,
			Stats:         entity.Stats{MaxHP: 300, HP: 300, Attack: 35, Defense: 25, BaseSpeed: 85.0, Effectiveness: 20, EffectResistance: 40, HitChance: 100, Evasion: 0},
			Skills:        makeSinucaSkills(),
			StatusEffects: make([]entity.StatusEffect, 0),
			SkillCounters: make(map[string]int)}
	case "h5":
		return &entity.Entity{ID: "h5", Name: "Lojinha Cafeina", Team: entity.TeamPlayers, Faction: "LOJINHA", Subclass: "CAFEINA", Alive: true,
			Stats:         entity.Stats{MaxHP: 200, HP: 200, Attack: 45, Defense: 20, BaseSpeed: 110.0, Effectiveness: 30, EffectResistance: 40, HitChance: 100, Evasion: 10},
			Skills:        makeLojinhaSkills(),
			StatusEffects: make([]entity.StatusEffect, 0),
			SkillCounters: make(map[string]int)}
	}
	return nil
}

func createCustomState(heroIDs []string) *combat.CombatState {
	newState := &combat.CombatState{ActionLog: make([]string, 0)}
	for _, id := range heroIDs {
		if hero := getHeroByID(id); hero != nil {
			newState.Entities = append(newState.Entities, hero)
		}
	}
	newState.Entities = append(newState.Entities, &entity.Entity{
		ID: "m1", Name: "Goblin", Team: entity.TeamMonsters, Faction: "PHYSICAL", Alive: true,
		Stats:         entity.Stats{MaxHP: 80, HP: 80, Attack: 30, Defense: 5, BaseSpeed: 130.0, Effectiveness: 100, EffectResistance: 30, HitChance: 100, Evasion: 10},
		Skills:        makeMonsterSkills(),
		StatusEffects: make([]entity.StatusEffect, 0),
		SkillCounters: make(map[string]int),
	})
	newState.Entities = append(newState.Entities, &entity.Entity{
		ID: "m2", Name: "Slime", Team: entity.TeamMonsters, Faction: "GHOST", Alive: true,
		Stats:         entity.Stats{MaxHP: 100, HP: 100, Attack: 20, Defense: 15, BaseSpeed: 90.0, Effectiveness: 100, EffectResistance: 50, HitChance: 100, Evasion: 50},
		Skills:        makeMonsterSkills(),
		StatusEffects: make([]entity.StatusEffect, 0),
		SkillCounters: make(map[string]int),
	})
	newState.Entities = append(newState.Entities, &entity.Entity{
		ID: "m3", Name: "Wolf", Team: entity.TeamMonsters, Faction: "CHAOS", Alive: true,
		Stats:         entity.Stats{MaxHP: 100, HP: 100, Attack: 45, Defense: 12, BaseSpeed: 115.0, Effectiveness: 100, EffectResistance: 20, HitChance: 100, Evasion: 0},
		Skills:        makeMonsterSkills(),
		Passives:      []entity.Passive{{Type: "COUNTER", Chance: 100.0}},
		StatusEffects: make([]entity.StatusEffect, 0),
		SkillCounters: make(map[string]int),
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
		CurrentTick: state.CurrentTick,
		Entities:    state.Entities,
		Timeline:    state.SimulateTimeline(),
		Steps:       steps,
		ActionLog:   state.ActionLog,
		IsGameOver:  false,
	}
	if nextActor != nil {
		resp.ActiveActorID = nextActor.ID
		resp.ActiveTeam = int(nextActor.Team)
	} else {
		resp.IsGameOver = true
	}

	playersAlive, monstersAlive := 0, 0
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
			// --- SUBCLASS TRIGGERS (Start of Turn) ---
			if actor.Subclass == "CAFEINA" {
				actor.CombatReadiness += 15.0
				state.ActionLog = append(state.ActionLog, fmt.Sprintf("  -> [Cafeina] %s is jittery and gains +15%% CR!", actor.Name))
			}
			if actor.Subclass == "CEVA" && actor.Stats.HP > (actor.Stats.MaxHP*70/100) {
				actor.StatusEffects = append(actor.StatusEffects, entity.StatusEffect{Type: "JOY", Value: 20, Duration: 1})
				state.ActionLog = append(state.ActionLog, fmt.Sprintf("  -> [Ceva] %s feels happy! (+20%% ATK)", actor.Name))
			}

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

					currentStacks := actor.SkillCounters[skill.ID]
					reactionQueue := []Reaction{}
					extraDamagePool := make(map[string]int)

					for _, effect := range skill.Effects {
						effectTargets := getEffectTargets(actor, skill.TargetType, &effect, state.Entities, requestedID)

						for _, targetEntity := range effectTargets {
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
								didHit, _ := CheckHit(actor, targetEntity)
								if didHit {
									if CheckEffect(actor, targetEntity, &effect) {
										targetEntity.StatusEffects = append(targetEntity.StatusEffects, entity.StatusEffect{Type: effect.StatusType, Value: effect.Value, Duration: effect.Duration, IsDebuff: true})
										state.ActionLog = append(state.ActionLog, fmt.Sprintf("  -> %s is afflicted with %s.", targetEntity.Name, effect.StatusType))
										steps = append(steps, VisualStep{Type: "DEBUFF", Target: targetEntity.ID, Text: effect.StatusType})
									} else {
										state.ActionLog = append(state.ActionLog, fmt.Sprintf("  -> %s resisted %s.", targetEntity.Name, effect.StatusType))
										steps = append(steps, VisualStep{Type: "RESIST", Target: targetEntity.ID, Text: "Resisted"})
									}
								}
							}
							if effect.Type == "DAMAGE" {
								didHit, isDisadvantaged := CheckHit(actor, targetEntity)

								baseDamage := (float64(actor.Stats.Attack) * effect.Multiplier) + (float64(actor.Stats.MaxHP) * effect.HPMultiplier)
								if currentStacks > 0 && skill.StackMultiplier > 0 {
									baseDamage *= (1.0 + (float64(currentStacks) * skill.StackMultiplier))
								}

								for _, s := range actor.StatusEffects {
									if s.Type == "JOY" {
										baseDamage *= (1.0 + float64(s.Value)/100.0)
									}
								}

								factionMult := GetFactionMultiplier(actor.Faction, targetEntity.Faction)
								baseDamage *= factionMult

								// E7 MISS LOGIC: Misses still deal damage, but reduce it by 25% (0.75 multiplier)
								if !didHit {
									state.ActionLog = append(state.ActionLog, fmt.Sprintf("  -> %s's attack MISSED %s! (Damage reduced by 25%%)", actor.Name, targetEntity.Name))
									steps = append(steps, VisualStep{Type: "MISS", Target: targetEntity.ID})
									baseDamage *= 0.75
								} else if factionMult > 1.0 {
									state.ActionLog = append(state.ActionLog, fmt.Sprintf("  -> Faction Advantage! (%s beats %s)", actor.Faction, targetEntity.Faction))
								} else if isDisadvantaged {
									state.ActionLog = append(state.ActionLog, fmt.Sprintf("  -> Faction Disadvantage! (%s loses to %s)", actor.Faction, targetEntity.Faction))
								}

								defense := float64(targetEntity.GetEffectiveDefense())
								if effect.Condition == "TARGET_HAS_SHIELD" && targetEntity.HasStatus("SHIELD") {
									defense *= (1.0 - (float64(effect.DefPenetration) / 100.0))
								}

								finalDamage := int(baseDamage) - int(defense)
								if finalDamage < 1 {
									finalDamage = 1
								}

								shieldAbsorbed, hpLost := targetEntity.TakeDamage(finalDamage)
								damageDealtThisAction += hpLost

								if shieldAbsorbed > 0 {
									steps = append(steps, VisualStep{Type: "SHIELD_DAMAGE", Target: targetEntity.ID, Value: shieldAbsorbed})
								}
								if hpLost > 0 {
									state.ActionLog = append(state.ActionLog, fmt.Sprintf("  -> %s takes %d damage.", targetEntity.Name, hpLost))
									steps = append(steps, VisualStep{Type: "DAMAGE", Target: targetEntity.ID, Value: hpLost})
								}

								// If the attack MISSED, do not trigger Subclass on-hit effects (like Smoke)
								if didHit && skill.ID == "s1" && hpLost > 0 {
									if actor.Subclass == "CIGS" && rand.Float64()*100 <= 40.0 {
										targetEntity.StatusEffects = append(targetEntity.StatusEffects, entity.StatusEffect{Type: "BLIND", Value: 50, Duration: 2, IsDebuff: true})
										state.ActionLog = append(state.ActionLog, fmt.Sprintf("  -> [Cigs] %s's smoke blinds %s!", actor.Name, targetEntity.Name))
										steps = append(steps, VisualStep{Type: "DEBUFF", Target: targetEntity.ID, Text: "BLIND"})
									}
									if actor.Subclass == "VERDE" && rand.Float64()*100 <= 50.0 {
										targetEntity.StatusEffects = append(targetEntity.StatusEffects, entity.StatusEffect{Type: "SLOW", Value: 30, Duration: 2, IsDebuff: true})
										state.ActionLog = append(state.ActionLog, fmt.Sprintf("  -> [Verde] %s slows %s with a lazy haze!", actor.Name, targetEntity.Name))
										steps = append(steps, VisualStep{Type: "DEBUFF", Target: targetEntity.ID, Text: "SLOW"})
									}
									if actor.Subclass == "TRUCO" && rand.Float64()*100 <= 30.0 {
										state.ActionLog = append(state.ActionLog, fmt.Sprintf("  -> [Truco] %s calls a bluff and gains CR!", actor.Name))
										actor.CombatReadiness += 20.0
									}
								}

								if targetEntity.Stats.HP <= 0 && targetEntity.Alive {
									targetEntity.Stats.HP = 0
									targetEntity.Alive = false
									state.ActionLog = append(state.ActionLog, fmt.Sprintf("  -> %s has died!", targetEntity.Name))
								}

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
										targetEntity.IsUnrevivable = true
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
