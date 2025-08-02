package scenario

// BuiltIn returns predefined story arcs with mission descriptions.
func BuiltIn() map[string]Scenario {
	return map[string]Scenario{
		"escort": {
			Name:        "Escort",
			Description: "Protect a vulnerable convoy as it travels through hostile territory to the forward operating base.",
			Phases: []Phase{
				{
					Name:        "setup",
					Description: "Convoy forms up and prepares to depart.",
					Triggers:    []Trigger{{Event: "time_elapsed", Value: 30, Next: "escalation"}},
				},
				{
					Name:            "escalation",
					Description:     "Light enemy forces probe the escort.",
					EnemyObjectives: []EnemyObjective{{ID: "bandits", Action: "harass", Target: "convoy"}},
					Triggers:        []Trigger{{Event: "enemy_destroyed", Value: 3, Next: "climax"}},
				},
				{
					Name:            "climax",
					Description:     "A coordinated ambush hits the convoy near a choke point.",
					EnemyObjectives: []EnemyObjective{{ID: "ambush", Action: "attack", Target: "convoy"}},
					Triggers:        []Trigger{{Event: "enemy_destroyed", Value: 5, Next: "resolution"}},
				},
				{
					Name:        "resolution",
					Description: "Remaining threats fall back as the convoy reaches safety.",
				},
			},
		},
		"search-and-rescue": {
			Name:        "Search and Rescue",
			Description: "Locate and recover a downed pilot before hostile forces arrive.",
			Phases: []Phase{
				{
					Name:        "setup",
					Description: "Rescue team briefs and launches into the search area.",
					Triggers:    []Trigger{{Event: "time_elapsed", Value: 20, Next: "escalation"}},
				},
				{
					Name:        "escalation",
					Description: "Search intensifies as signs of the pilot are found.",
					Triggers:    []Trigger{{Event: "survivor_found", Value: 1, Next: "climax"}},
				},
				{
					Name:            "climax",
					Description:     "Enemy patrols close in while the extraction is attempted.",
					EnemyObjectives: []EnemyObjective{{ID: "enemy-patrol", Action: "attack", Target: "rescue-team"}},
					Triggers:        []Trigger{{Event: "survivor_extracted", Value: 1, Next: "resolution"}},
				},
				{
					Name:        "resolution",
					Description: "Team departs the area with the survivor; remaining enemies retreat.",
				},
			},
		},
		"defensive-stand": {
			Name:        "Defensive Stand",
			Description: "Hold a critical relay station against waves of hostile drones.",
			Phases: []Phase{
				{
					Name:        "setup",
					Description: "Defenders fortify the station and establish fields of fire.",
					Triggers:    []Trigger{{Event: "time_elapsed", Value: 30, Next: "escalation"}},
				},
				{
					Name:            "escalation",
					Description:     "The first wave tests the defenses.",
					EnemyObjectives: []EnemyObjective{{ID: "wave1", Action: "attack", Target: "station"}},
					Triggers:        []Trigger{{Event: "enemy_destroyed", Value: 5, Next: "climax"}},
				},
				{
					Name:            "climax",
					Description:     "A massive assault threatens to overwhelm the defenders.",
					EnemyObjectives: []EnemyObjective{{ID: "wave2", Action: "attack", Target: "station"}},
					Triggers:        []Trigger{{Event: "time_elapsed", Value: 120, Next: "resolution"}},
				},
				{
					Name:        "resolution",
					Description: "Enemy forces withdraw and the station remains secure.",
				},
			},
		},
	}
}
