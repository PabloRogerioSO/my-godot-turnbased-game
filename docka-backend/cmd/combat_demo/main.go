package main

import (
	"docka/backend/internal/combat"
	"docka/backend/internal/entity"
	"fmt"
)

func main() {
	state := &combat.CombatState{}

	// Heroes
	state.Entities = append(state.Entities, &entity.Entity{
		ID: "h1", Name: "Knight", Team: entity.TeamPlayers, Alive: true,
		Stats: entity.Stats{MaxHP: 250, HP: 250, BaseSpeed: 95.0},
	})
	state.Entities = append(state.Entities, &entity.Entity{
		ID: "h2", Name: "Archer", Team: entity.TeamPlayers, Alive: true,
		Stats: entity.Stats{MaxHP: 100, HP: 100, BaseSpeed: 120.0},
	})
	state.Entities = append(state.Entities, &entity.Entity{
		ID: "h3", Name: "Mage", Team: entity.TeamPlayers, Alive: true,
		Stats: entity.Stats{MaxHP: 100, HP: 100, BaseSpeed: 105.0},
	})

	// Monsters
	state.Entities = append(state.Entities, &entity.Entity{
		ID: "m1", Name: "Goblin", Team: entity.TeamMonsters, Alive: true,
		Stats: entity.Stats{MaxHP: 80, HP: 80, BaseSpeed: 130.0},
	})
	state.Entities = append(state.Entities, &entity.Entity{
		ID: "m2", Name: "Slime", Team: entity.TeamMonsters, Alive: true,
		Stats: entity.Stats{MaxHP: 100, HP: 100, BaseSpeed: 90.0},
	})
	state.Entities = append(state.Entities, &entity.Entity{
		ID: "m3", Name: "Wolf", Team: entity.TeamMonsters, Alive: true,
		Stats: entity.Stats{MaxHP: 100, HP: 100, BaseSpeed: 115.0},
	})

	fmt.Println("--- COMBAT READINESS SIMULATION ---")
	for i := 1; i <= 8; i++ {
		actor := state.GetNextActor()
		fmt.Printf("Turn %d: [%s] takes action! (Speed: %.1f | Tick: %d)\n",
			i, actor.Name, actor.EffectiveSpeed(), state.CurrentTick)
	}
}
