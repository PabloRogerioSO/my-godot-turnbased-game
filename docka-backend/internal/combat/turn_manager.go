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
		// Skip dead and banished entities. They don't gain CR.
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
			// Banished entities are skipped
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

			// CR Overflow Logic: Subtract 100 when taking a turn
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

// SimulateTimeline returns the predicted order of the next N turns
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

	for i := 0; i < 50; i++ {
		actor := simState.GetNextActor()
		if actor == nil {
			break
		}

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
