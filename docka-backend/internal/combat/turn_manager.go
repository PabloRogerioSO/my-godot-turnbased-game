package combat

import (
	"docka/backend/internal/entity"
	"sort"
)

// CRThreshold is the value an entity's Combat Readiness must reach to take a turn.
const CRThreshold float64 = 100.0

// TimelineEntry represents a single unit's predicted turn in the UI timeline.
type TimelineEntry struct {
	Name string  `json:"name"`
	CR   float64 `json:"cr"`
}

// CombatState holds the entire state of a single battle.
type CombatState struct {
	Entities    []*entity.Entity
	CurrentTick int64
	ActionLog   []string
}

// Tick advances the simulation time by one step, increasing CR based on speed.
func (s *CombatState) Tick() {
	s.CurrentTick++

	for _, e := range s.Entities {
		// Skip dead entities and Banished entities (they don't generate CR while off-field).
		if !e.Alive || e.HasStatus("BANISHED") {
			continue
		}

		e.CombatReadiness += e.EffectiveSpeed()

		// Record the exact tick they reached the threshold for FIFO tie-breaking.
		if e.CombatReadiness >= CRThreshold && e.ReachedCRAtTick == 0 {
			e.ReachedCRAtTick = s.CurrentTick
		}
	}
}

// GetNextActor runs the tick loop until at least one entity is ready, then returns the one who acts.
func (s *CombatState) GetNextActor() *entity.Entity {
	for {
		var readyEntities []*entity.Entity
		var anyActiveAlive bool

		for _, e := range s.Entities {
			// Banished entities are skipped and cannot act.
			if e.Alive && !e.HasStatus("BANISHED") {
				anyActiveAlive = true
				if e.CombatReadiness >= CRThreshold {
					readyEntities = append(readyEntities, e)
				}
			}
		}

		// If everyone is dead or banished, the battle is over.
		if !anyActiveAlive {
			return nil
		}

		if len(readyEntities) > 0 {
			// Sort by Highest CR. If tie, the one who reached it first (lower tick) goes first (FIFO).
			sort.Slice(readyEntities, func(i, j int) bool {
				if readyEntities[i].CombatReadiness != readyEntities[j].CombatReadiness {
					return readyEntities[i].CombatReadiness > readyEntities[j].CombatReadiness
				}
				return readyEntities[i].ReachedCRAtTick < readyEntities[j].ReachedCRAtTick
			})

			actor := readyEntities[0]

			// CR Overflow: Subtract 100 when taking a turn.
			// If they had 150%, they act, and keep 50% for their next turn.
			actor.CombatReadiness -= CRThreshold
			if actor.CombatReadiness < 0 {
				actor.CombatReadiness = 0
			}
			actor.ReachedCRAtTick = 0

			return actor
		}

		s.Tick()
	}
}

// PeekNextActor checks who will act next WITHOUT modifying the actual game state.
func (s *CombatState) PeekNextActor() *entity.Entity {
	simEntities := make([]*entity.Entity, len(s.Entities))
	for i, e := range s.Entities {
		copyE := *e
		simEntities[i] = &copyE
	}

	simState := &CombatState{
		Entities:    simEntities,
		CurrentTick: s.CurrentTick,
	}

	return simState.GetNextActor()
}

// SimulateTimeline predicts the upcoming turn order for the UI.
// It shows the predicted CR (e.g., 130%) instead of their current CR, so the UI
// numbers descend naturally from top to bottom (130% -> 120% -> 100%).
func (s *CombatState) SimulateTimeline() []TimelineEntry {
	simEntities := make([]*entity.Entity, len(s.Entities))
	for i, e := range s.Entities {
		copyE := *e
		simEntities[i] = &copyE
	}

	simState := &CombatState{
		Entities:    simEntities,
		CurrentTick: s.CurrentTick,
	}

	var order []TimelineEntry
	seen := make(map[string]bool)

	activeCount := 0
	for _, e := range s.Entities {
		if e.Alive && !e.HasStatus("BANISHED") {
			activeCount++
		}
	}

	// Simulate up to 50 turns ahead to fill the timeline
	for i := 0; i < 50; i++ {
		actor := simState.GetNextActor()
		if actor == nil {
			break
		}

		// Only show each entity once on the timeline
		if !seen[actor.ID] {
			seen[actor.ID] = true
			// The GetNextActor function subtracts 100. We add 100 back to show the "peak" CR they reached.
			predictedCR := actor.CombatReadiness + 100.0
			order = append(order, TimelineEntry{Name: actor.Name, CR: predictedCR})
		}

		if len(order) >= activeCount {
			break
		}
	}

	return order
}

// ProcessEndOfTurn handles durations and expirations that happen at the end of a full turn.
func (s *CombatState) ProcessEndOfTurn(casterID string) {
	for _, e := range s.Entities {
		newStatuses := make([]entity.StatusEffect, 0)
		for _, status := range e.StatusEffects {

			// BANISHMENT LOGIC:
			// The timer only ticks down when the CASTER takes a turn.
			// If the caster dies, the banishment is lifted instantly.
			if status.Type == "BANISHED" {
				casterAlive := false
				for _, c := range s.Entities {
					if c.ID == status.SourceID && c.Alive {
						casterAlive = true
						break
					}
				}

				if !casterAlive {
					s.ActionLog = append(s.ActionLog, e.Name+" has returned to the field (caster died)!")
					continue
				}

				if status.SourceID == casterID {
					status.Duration--
					if status.Duration <= 0 {
						s.ActionLog = append(s.ActionLog, e.Name+" has returned to the field (banish expired)!")
						continue
					}
				}
			}
			newStatuses = append(newStatuses, status)
		}
		e.StatusEffects = newStatuses
	}
}
