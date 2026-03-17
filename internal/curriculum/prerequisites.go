package curriculum

const UnlockMasteryThreshold = 0.8

// PrereqGraph is a reverse dependency graph for curriculum topics.
// It maps each topic to the topics that depend on it (require it as a prerequisite).
type PrereqGraph struct {
	// dependents maps topicID → list of topic IDs that require it.
	dependents map[string][]string
	// prereqs maps topicID → list of required prerequisite topic IDs.
	prereqs map[string][]string
	// topics maps topicID → Topic for quick lookup.
	topics map[string]Topic
}

// NewPrereqGraph builds a prerequisite graph from a list of topics.
func NewPrereqGraph(topics []Topic) *PrereqGraph {
	g := &PrereqGraph{
		dependents: make(map[string][]string),
		prereqs:    make(map[string][]string),
		topics:     make(map[string]Topic),
	}
	for _, t := range topics {
		g.topics[t.ID] = t
		g.prereqs[t.ID] = t.Prerequisites.Required
		for _, req := range t.Prerequisites.Required {
			g.dependents[req] = append(g.dependents[req], t.ID)
		}
	}
	return g
}

// DependentsOf returns all topic IDs that require the given topic as a prerequisite.
func (g *PrereqGraph) DependentsOf(topicID string) []string {
	return g.dependents[topicID]
}

// RequiredPrereqs returns the required prerequisites for a topic.
func (g *PrereqGraph) RequiredPrereqs(topicID string) []string {
	return g.prereqs[topicID]
}

// UnlockableTopics returns topics that become newly unlockable after mastering
// the given topic. A topic is unlockable when:
// 1. It has required prerequisites (topics with no prereqs are always available)
// 2. ALL of its required prerequisites have mastery ≥ UnlockMasteryThreshold
// 3. The topic itself is NOT already mastered (no re-notification)
func (g *PrereqGraph) UnlockableTopics(masteredTopicID string, scores map[string]float64) []Topic {
	deps := g.dependents[masteredTopicID]
	if len(deps) == 0 {
		return nil
	}

	var unlocked []Topic
	for _, depID := range deps {
		// Skip if no prereqs (always available, no unlock needed).
		prereqs := g.prereqs[depID]
		if len(prereqs) == 0 {
			continue
		}

		// Skip if already mastered (don't re-notify).
		if scores[depID] >= UnlockMasteryThreshold {
			continue
		}

		// Check if ALL required prereqs are now mastered.
		allMet := true
		for _, req := range prereqs {
			if scores[req] < UnlockMasteryThreshold {
				allMet = false
				break
			}
		}
		if allMet {
			if t, ok := g.topics[depID]; ok {
				unlocked = append(unlocked, t)
			}
		}
	}
	return unlocked
}
