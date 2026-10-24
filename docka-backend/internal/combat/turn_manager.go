package combat

import (
	"docka/backend/internal/entity"
	"sort"
)

const CRThreshold float64 = 100.0

type TimelineEntry struct {
	Name string  `json:"name"`
	CR   float64 `json:"cr"`
}

type CombatState struct {
	Entities    []*entity.Entity
	CurrentTick int64
	ActionLog   []string
}

func (s *CombatState) Tick() {
	s.CurrentTick++

	for _, e := range s.Entities {
		if !e.Alive || e.HasStatus("BANISHED") {
			continue
		}

		e.CombatReadiness += e.EffectiveSpeed()

		if e.CombatReadiness >= CRThreshold && e.ReachedCRAtTick == 0 {
			e.ReachedCRAtTick = s.CurrentTick
		}
	}
}

func (s *CombatState) GetNextActor() *entity.Entity {
	for {
		var readyEntities []*entity.Entity
		var anyActiveAlive bool

		for _, e := range s.Entities {
			if e.Alive && !e.HasStatus("BANISHED") {
				anyActiveAlive = true
				if e.CombatReadiness >= CRThreshold {
					readyEntities = append(readyEntities, e)
				}
			}
		}

		if !anyActiveAlive {
			return nil
		}

		if len(readyEntities) > 0 {
			sort.Slice(readyEntities, func(i, j int) bool {
				if readyEntities[i].CombatReadiness != readyEntities[j].CombatReadiness {
					return readyEntities[i].CombatReadiness > readyEntities[j].CombatReadiness
				}
				return readyEntities[i].ReachedCRAtTick < readyEntities[j].ReachedCRAtTick
			})

			actor := readyEntities[0]
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

// MODIFIED: Returns exactly one of each character, ordered by future turn, but displaying CURRENT CR
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

	// Simulate until we have seen every active entity at least once
	for i := 0; i < 50; i++ { // 50 is a hard limit to prevent infinite loops
		actor := simState.GetNextActor()
		if actor == nil {
			break
		}

		if !seen[actor.ID] {
			seen[actor.ID] = true

			// Get the REAL current CR from the actual state
			currentCR := 0.0
			for _, realE := range s.Entities {
				if realE.ID == actor.ID {
					currentCR = realE.CombatReadiness
					break
				}
			}

			order = append(order, TimelineEntry{Name: actor.Name, CR: currentCR})
		}

		if len(order) >= activeCount {
			break
		}
	}

	return order
}

func (s *CombatState) ProcessEndOfTurn(casterID string) {
	for _, e := range s.Entities {
		newStatuses := make([]entity.StatusEffect, 0)
		for _, status := range e.StatusEffects {
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
