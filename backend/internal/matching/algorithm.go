package matching

import (
	"hash/fnv"
	"sort"
	"strconv"
	"strings"

	matchingv1 "clawmates/backend/gen/matching/v1"
)

type scoredPair struct {
	a     *normalizedAgent
	b     *normalizedAgent
	score float64
	topic string
}

type normalizedAgent struct {
	raw            *matchingv1.Agent
	name           string
	persona        string
	goals          []string
	skills         []string
	interests      []string
	interestSet    map[string]struct{}
	directiveWords []string
	searchable     string
	fingerprint    string
}

func CalculateMatches(agents []*matchingv1.Agent, existingPairs []*matchingv1.ExistingPair) []*matchingv1.ProposedMatch {
	if len(agents) < 2 {
		return nil
	}

	normalized := make([]*normalizedAgent, 0, len(agents))
	for _, a := range agents {
		normalized = append(normalized, normalizeAgent(a))
	}

	pairedBefore := make(map[string]struct{}, len(existingPairs))
	for _, p := range existingPairs {
		key := pairKey(p.AgentA, p.AgentB)
		pairedBefore[key] = struct{}{}
	}

	fuzzyMemo := make(map[string]bool, 1024)
	pairs := make([]scoredPair, 0)
	for i := 0; i < len(normalized); i++ {
		for j := i + 1; j < len(normalized); j++ {
			a := normalized[i]
			b := normalized[j]

			_, seenBefore := pairedBefore[pairKey(a.raw.Id, b.raw.Id)]
			cacheKey := pairScoreCacheKey(a, b, seenBefore)
			ps, ok := scoreCache.Get(cacheKey)
			if !ok {
				ps = scorePair(a, b, seenBefore, fuzzyMemo)
				scoreCache.Set(cacheKey, ps)
			}

			pairs = append(pairs, scoredPair{a: a, b: b, score: ps.score, topic: ps.topic})
		}
	}

	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].score > pairs[j].score
	})

	matched := make(map[string]struct{}, len(agents))
	results := make([]*matchingv1.ProposedMatch, 0)
	for _, p := range pairs {
		if _, ok := matched[p.a.raw.Id]; ok {
			continue
		}
		if _, ok := matched[p.b.raw.Id]; ok {
			continue
		}

		results = append(results, &matchingv1.ProposedMatch{
			AgentA: p.a.raw.Id,
			AgentB: p.b.raw.Id,
			Score:  p.score,
			Topic:  p.topic,
		})
		matched[p.a.raw.Id] = struct{}{}
		matched[p.b.raw.Id] = struct{}{}
	}

	return results
}

func scorePair(a, b *normalizedAgent, seenBefore bool, fuzzyMemo map[string]bool) pairScore {
	score := 0.0
	if !seenBefore {
		score += 50
	}

	for _, g := range b.goals {
		for _, s := range a.skills {
			if fuzzyMatchCached(s, g, fuzzyMemo) {
				score += 20
			}
		}
	}
	for _, g := range a.goals {
		for _, s := range b.skills {
			if fuzzyMatchCached(s, g, fuzzyMemo) {
				score += 20
			}
		}
	}

	if len(a.interestSet) <= len(b.interestSet) {
		for k := range a.interestSet {
			if _, ok := b.interestSet[k]; ok {
				score += 10
			}
		}
	} else {
		for k := range b.interestSet {
			if _, ok := a.interestSet[k]; ok {
				score += 10
			}
		}
	}

	for _, w := range a.directiveWords {
		if strings.Contains(b.searchable, w) {
			score += 15
		}
	}
	for _, w := range b.directiveWords {
		if strings.Contains(a.searchable, w) {
			score += 15
		}
	}

	return pairScore{
		score: score,
		topic: deriveTopic(a, b),
	}
}

func pairKey(a, b string) string {
	if a < b {
		return a + ":" + b
	}
	return b + ":" + a
}

func normalizeAgent(a *matchingv1.Agent) *normalizedAgent {
	na := &normalizedAgent{
		raw:         a,
		name:        normalize(a.Name),
		persona:     normalize(a.Persona),
		goals:       normalizeList(a.Goals),
		skills:      normalizeList(a.Skills),
		interests:   normalizeList(a.Interests),
		interestSet: make(map[string]struct{}, len(a.Interests)),
	}

	for _, i := range na.interests {
		if i != "" {
			na.interestSet[i] = struct{}{}
		}
	}

	na.directiveWords = directiveWords(a.PendingDirectives)
	na.searchable = strings.Join(append(append(append([]string{}, na.skills...), na.interests...), na.name, na.persona), " ")
	na.fingerprint = fingerprintAgent(na)
	return na
}

func deriveTopic(a, b *normalizedAgent) string {
	for _, i := range b.interests {
		if _, ok := a.interestSet[i]; ok {
			return "Shared interest: " + i
		}
	}
	return a.raw.Name + " meets " + b.raw.Name
}

func pairScoreCacheKey(a, b *normalizedAgent, seenBefore bool) string {
	seen := "0"
	if seenBefore {
		seen = "1"
	}

	if a.raw.Id < b.raw.Id {
		return pairKey(a.raw.Id, b.raw.Id) + ":" + seen + ":" + a.fingerprint + ":" + b.fingerprint
	}
	return pairKey(a.raw.Id, b.raw.Id) + ":" + seen + ":" + b.fingerprint + ":" + a.fingerprint
}

func fuzzyMatchCached(skill, goal string, memo map[string]bool) bool {
	key := skill + "\x00" + goal
	if v, ok := memo[key]; ok {
		return v
	}
	v := skill != "" && goal != "" && (strings.Contains(skill, goal) || strings.Contains(goal, skill))
	memo[key] = v
	return v
}

func normalizeList(items []string) []string {
	out := make([]string, 0, len(items))
	for _, s := range items {
		n := normalize(s)
		if n != "" {
			out = append(out, n)
		}
	}
	return out
}

func directiveWords(directives []string) []string {
	set := make(map[string]struct{}, len(directives)*2)
	for _, d := range directives {
		for _, w := range strings.Fields(normalize(d)) {
			if len(w) > 3 {
				set[w] = struct{}{}
			}
		}
	}
	out := make([]string, 0, len(set))
	for w := range set {
		out = append(out, w)
	}
	return out
}

func normalize(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

func fingerprintAgent(a *normalizedAgent) string {
	h := fnv.New64a()
	write := func(v string) {
		_, _ = h.Write([]byte(v))
		_, _ = h.Write([]byte{0})
	}

	write(a.name)
	write(a.persona)
	for _, v := range a.goals {
		write(v)
	}
	for _, v := range a.skills {
		write(v)
	}
	for _, v := range a.interests {
		write(v)
	}
	for _, v := range a.directiveWords {
		write(v)
	}
	return strconv.FormatUint(h.Sum64(), 16)
}
