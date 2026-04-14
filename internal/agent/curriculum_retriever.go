// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package agent

import (
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/p-n-ai/pai-bot/internal/curriculum"
	"github.com/p-n-ai/pai-bot/internal/progress"
	"github.com/p-n-ai/pai-bot/internal/retrieval"
)

const (
	bm25K1              = 1.2
	bm25B               = 0.75
	minTopicScore       = 0.75
	formMismatchPenalty = 0.1
)

var (
	formPattern    = regexp.MustCompile(`(?i)\b(?:tingkatan|form|f)[\s\-_]*([123])\b`)
	headingPattern = regexp.MustCompile(`^#{2,6}\s+`)
	fieldWeights   = map[string]float64{
		"title":      3.2,
		"aliases":    2.8,
		"subject":    1.6,
		"syllabus":   1.2,
		"objectives": 2.2,
		"heading":    1.8,
		"body":       1.0,
	}
	retrievalStopWords = map[string]struct{}{
		"and": {}, "the": {}, "for": {}, "with": {}, "that": {}, "this": {}, "what": {}, "how": {}, "from": {},
		"is": {}, "are": {}, "was": {}, "were": {}, "be": {}, "am": {}, "do": {}, "did": {}, "does": {},
		"you": {}, "your": {}, "me": {}, "my": {}, "we": {}, "our": {}, "they": {}, "their": {}, "a": {},
		"an": {}, "to": {}, "of": {}, "in": {}, "on": {}, "at": {}, "by": {}, "or": {}, "if": {}, "it": {},
	}
	followUpKeywords = map[string]struct{}{
		"why": {}, "how": {}, "move": {}, "next": {}, "then": {}, "stuck": {}, "confused": {}, "again": {},
		"step": {}, "semak": {}, "kenapa": {}, "macam": {}, "mana": {}, "lepas": {}, "selepas": {}, "faham": {},
		"tak": {}, "lagi": {},
	}
	genericFollowUpTokens = map[string]struct{}{
		"why": {}, "how": {}, "move": {}, "other": {}, "side": {}, "next": {}, "then": {}, "stuck": {},
		"confused": {}, "again": {}, "step": {}, "first": {}, "only": {}, "semak": {}, "kenapa": {},
		"macam": {}, "mana": {}, "lepas": {}, "selepas": {}, "faham": {}, "tak": {}, "lagi": {},
	}
	stemOverrides = map[string]string{
		"equation": "equation", "equations": "equation",
		"inequality": "inequality", "inequalities": "inequality",
		"variable": "variable", "variables": "variable",
		"expression": "expression", "expressions": "expression",
		"graphs": "graph", "lines": "line",
	}
)

type curriculumRetrieverConfig struct {
	service     *retrieval.Service
	store       ConversationStore
	tracker     progress.Tracker
	prereqGraph *curriculum.PrereqGraph
}

type curriculumRetriever struct {
	loader      *curriculum.Loader
	service     *retrieval.Service
	store       ConversationStore
	tracker     progress.Tracker
	prereqGraph *curriculum.PrereqGraph
	docs        []retrievalDoc
	docFreq     map[string]int
	avgFieldLen map[string]float64
	topics      map[string]curriculum.Topic
}

type retrievalDoc struct {
	ID         string
	TopicID    string
	SubjectID  string
	SyllabusID string
	Form       string
	Kind       string
	Fields     map[string]string
	Terms      map[string]map[string]int
	Lengths    map[string]int
}

type scoredDoc struct {
	Doc             retrievalDoc
	Score           float64
	MatchedTerms    int
	HighSignalTerms int
}

type scoredTopic struct {
	Topic      curriculum.Topic
	Score      float64
	Docs       []scoredDoc
	Confidence string
}

type retrievalResult struct {
	Topic      *curriculum.Topic
	Notes      string
	Score      float64
	Confidence string
}

func newCurriculumRetriever(loader *curriculum.Loader, cfg curriculumRetrieverConfig) *curriculumRetriever {
	if loader == nil {
		return nil
	}
	r := &curriculumRetriever{
		loader:      loader,
		service:     cfg.service,
		store:       cfg.store,
		tracker:     cfg.tracker,
		prereqGraph: cfg.prereqGraph,
		docFreq:     make(map[string]int),
		avgFieldLen: make(map[string]float64),
		topics:      make(map[string]curriculum.Topic),
	}
	if r.service == nil {
		// Fallback mode: build a private in-memory index from curriculum files.
		// Shared-service mode skips this and delegates retrieval to internal/retrieval.
		r.buildIndex()
	}
	for _, topic := range loader.AllTopics() {
		r.topics[topic.ID] = topic
	}
	return r
}

func (r *curriculumRetriever) Resolve(query ContextQuery) retrievalResult {
	if r == nil {
		return retrievalResult{}
	}
	if r.service != nil {
		// Preferred path: use the shared retrieval service so agent retrieval and
		// API retrieval stay aligned.
		return r.resolveWithService(query)
	}
	if len(r.docs) == 0 {
		return retrievalResult{}
	}

	terms := tokenizeRetrievalText(query.Text)
	if result, ok := r.resolveActiveFollowUp(query, terms); ok {
		return result
	}
	if len(terms) == 0 {
		return retrievalResult{}
	}

	explicitForm := detectFormInText(query.Text)
	userForm := ""
	if explicitForm == "" && r.store != nil && query.UserID != "" {
		userForm, _ = r.store.GetUserForm(query.UserID)
	}
	targetForm := explicitForm
	if targetForm == "" {
		targetForm = userForm
	}

	progressBoosts := map[string]float64{}
	if r.tracker != nil && query.UserID != "" {
		if items, err := r.tracker.GetAllProgress(query.UserID); err == nil {
			for _, item := range items {
				if item.MasteryScore > 0 && item.MasteryScore < 0.6 {
					progressBoosts[item.TopicID] = (0.6 - item.MasteryScore) * 0.6
				}
			}
		}
	}

	neighborBoosts := map[string]float64{}
	if r.prereqGraph != nil && query.ConversationTopicID != "" {
		for _, topicID := range r.prereqGraph.RequiredPrereqs(query.ConversationTopicID) {
			neighborBoosts[topicID] = 0.25
		}
		for _, topicID := range r.prereqGraph.DependentsOf(query.ConversationTopicID) {
			if neighborBoosts[topicID] < 0.25 {
				neighborBoosts[topicID] = 0.25
			}
		}
	}

	candidates := r.scoreDocs(terms, targetForm, false, progressBoosts, neighborBoosts, query)
	if len(candidates) == 0 && targetForm != "" {
		candidates = r.scoreDocs(terms, targetForm, true, progressBoosts, neighborBoosts, query)
	}
	if len(candidates) == 0 {
		return retrievalResult{}
	}

	topics := r.rankTopics(candidates)
	if len(topics) == 0 || topics[0].Score < minTopicScore {
		return retrievalResult{}
	}
	if topics[0].Confidence == "assessment" && topics[0].Docs[0].HighSignalTerms == 0 && topics[0].Docs[0].MatchedTerms < 2 {
		return retrievalResult{}
	}
	if r.topicHasWeakSpecificCoverage(topics[0].Topic.ID, terms) {
		return retrievalResult{}
	}
	if len(topics) > 1 && targetForm == "" {
		margin := topics[0].Score - topics[1].Score
		if margin <= 0.35 {
			return retrievalResult{}
		}
	}
	if len(topics) > 1 && topics[0].Score < 2.2 {
		margin := topics[0].Score - topics[1].Score
		if margin <= 0.15*topics[0].Score {
			return retrievalResult{}
		}
	}

	best := topics[0]
	topic := best.Topic
	return retrievalResult{
		Topic:      &topic,
		Notes:      r.composeNotes(best),
		Score:      best.Score,
		Confidence: best.Confidence,
	}
}

func (r *curriculumRetriever) resolveWithService(query ContextQuery) retrievalResult {
	// Shared-service retrieval flow:
	//  1. try safe active-topic follow-up reuse for generic follow-ups
	//  2. derive form priors from text or user profile
	//  3. query the shared retrieval service
	//  4. add agent-specific boosts (active topic, weak mastery, prereq neighbors)
	//  5. reject low-confidence or ambiguous matches instead of poisoning the prompt
	terms := tokenizeRetrievalText(query.Text)
	if result, ok := r.resolveActiveFollowUpWithService(query, terms); ok {
		return result
	}
	if len(terms) == 0 {
		return retrievalResult{}
	}

	explicitForm := detectFormInText(query.Text)
	userForm := ""
	if explicitForm == "" && r.store != nil && query.UserID != "" {
		userForm, _ = r.store.GetUserForm(query.UserID)
	}
	targetForm := explicitForm
	if targetForm == "" {
		targetForm = userForm
	}

	progressBoosts := map[string]float64{}
	if r.tracker != nil && query.UserID != "" {
		if items, err := r.tracker.GetAllProgress(query.UserID); err == nil {
			for _, item := range items {
				if item.MasteryScore > 0 && item.MasteryScore < 0.6 {
					progressBoosts[item.TopicID] = (0.6 - item.MasteryScore) * 0.6
				}
			}
		}
	}

	neighborBoosts := map[string]float64{}
	if r.prereqGraph != nil && query.ConversationTopicID != "" {
		for _, topicID := range r.prereqGraph.RequiredPrereqs(query.ConversationTopicID) {
			neighborBoosts[topicID] = 0.25
		}
		for _, topicID := range r.prereqGraph.DependentsOf(query.ConversationTopicID) {
			if neighborBoosts[topicID] < 0.25 {
				neighborBoosts[topicID] = 0.25
			}
		}
	}

	results, _ := r.searchService(query.Text, targetForm, false)
	usedFallback := false
	if len(results) == 0 && targetForm != "" {
		results, _ = r.searchService(query.Text, targetForm, true)
		usedFallback = true
	}
	if len(results) == 0 {
		return retrievalResult{}
	}

	candidates := r.scoredDocsFromSearchResults(results, progressBoosts, neighborBoosts, query, targetForm, usedFallback)
	if len(candidates) == 0 {
		return retrievalResult{}
	}

	topics := r.rankTopics(candidates)
	if len(topics) == 0 || topics[0].Score < minTopicScore {
		return retrievalResult{}
	}
	if topics[0].Confidence == "assessment" && topics[0].Docs[0].HighSignalTerms == 0 && topics[0].Docs[0].MatchedTerms < 2 {
		return retrievalResult{}
	}
	if r.topicHasWeakSpecificCoverage(topics[0].Topic.ID, terms) {
		return retrievalResult{}
	}
	if len(topics) > 1 && targetForm == "" {
		margin := topics[0].Score - topics[1].Score
		if margin <= 0.35 {
			return retrievalResult{}
		}
	}
	if len(topics) > 1 && topics[0].Score < 2.2 {
		margin := topics[0].Score - topics[1].Score
		if margin <= 0.15*topics[0].Score {
			return retrievalResult{}
		}
	}

	best := topics[0]
	topic := best.Topic
	return retrievalResult{
		Topic:      &topic,
		Notes:      r.composeNotes(best),
		Score:      best.Score,
		Confidence: best.Confidence,
	}
}

func (r *curriculumRetriever) searchService(text, targetForm string, allowFormFallback bool) ([]retrieval.SearchHit, error) {
	request := retrieval.SearchRequest{
		Query:           text,
		IncludeInactive: false,
		Limit:           24,
	}
	if targetForm != "" && !allowFormFallback {
		request.Metadata = map[string]string{"form": targetForm}
	}
	return r.service.Search(request)
}

func (r *curriculumRetriever) resolveActiveFollowUpWithService(query ContextQuery, terms []string) (retrievalResult, bool) {
	if query.ConversationTopicID == "" || !looksLikeFollowUp(query.Text) || !isGenericFollowUpQuery(terms) {
		return retrievalResult{}, false
	}

	topic, ok := r.topics[query.ConversationTopicID]
	if !ok {
		return retrievalResult{}, false
	}

	results, err := r.service.Search(retrieval.SearchRequest{
		Query:           query.Text,
		Metadata:        map[string]string{"topic_id": query.ConversationTopicID},
		IncludeInactive: false,
		Limit:           6,
	})
	if err != nil || len(results) == 0 {
		return retrievalResult{}, false
	}

	docs := r.scoredDocsFromSearchResults(results, nil, nil, query, "", false)
	if len(docs) == 0 {
		return retrievalResult{}, false
	}
	score := docs[0].Score + 1.5
	return retrievalResult{
		Topic:      &topic,
		Notes:      r.composeNotes(scoredTopic{Topic: topic, Score: score, Docs: docs, Confidence: "follow_up"}),
		Score:      score,
		Confidence: "follow_up",
	}, true
}

func (r *curriculumRetriever) scoredDocsFromSearchResults(results []retrieval.SearchHit, progressBoosts, neighborBoosts map[string]float64, query ContextQuery, targetForm string, usedFallback bool) []scoredDoc {
	out := make([]scoredDoc, 0, len(results))
	followUp := looksLikeFollowUp(query.Text)
	for _, result := range results {
		doc := docFromDocument(result.Document)
		score := result.Score
		if usedFallback && targetForm != "" && doc.Form != "" && doc.Form != targetForm {
			score -= formMismatchPenalty
		}
		if query.ConversationTopicID != "" && doc.TopicID == query.ConversationTopicID && (score > 0 || followUp) {
			score += 0.9
		}
		if boost := progressBoosts[doc.TopicID]; boost > 0 {
			score += boost
		}
		if boost := neighborBoosts[doc.TopicID]; boost > 0 {
			score += boost
		}
		if score <= 0 {
			continue
		}
		out = append(out, scoredDoc{
			Doc:             doc,
			Score:           score,
			MatchedTerms:    result.MatchedTerms,
			HighSignalTerms: result.HighSignalTerms,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Score == out[j].Score {
			return out[i].Doc.ID < out[j].Doc.ID
		}
		return out[i].Score > out[j].Score
	})
	return out
}

func docFromDocument(document retrieval.Document) retrievalDoc {
	kind := strings.TrimSpace(document.Metadata["kind"])
	if kind == "" {
		kind = document.Kind
	}
	return retrievalDoc{
		ID:         document.ID,
		TopicID:    strings.TrimSpace(document.Metadata["topic_id"]),
		SubjectID:  strings.TrimSpace(document.Metadata["subject_id"]),
		SyllabusID: strings.TrimSpace(document.Metadata["syllabus_id"]),
		Form:       strings.TrimSpace(document.Metadata["form"]),
		Kind:       kind,
		Fields: map[string]string{
			"heading": document.Title,
			"aliases": strings.Join(document.Tags, " "),
			"body":    document.Body,
		},
	}
}

func (r *curriculumRetriever) buildIndex() {
	fieldTotals := make(map[string]int)
	for _, topic := range r.loader.AllTopics() {
		r.topics[topic.ID] = topic
		subject, _ := r.loader.GetSubject(topic.SubjectID)
		syllabus, _ := r.loader.GetSyllabus(topic.SyllabusID)
		form := inferTopicForm(topic, subject)

		r.addDoc(newRetrievalDoc("topic:"+topic.ID, topic.ID, topic.SubjectID, topic.SyllabusID, form, "topic", map[string]string{
			"title":      topic.Name,
			"aliases":    strings.Join(topicAliases(topic), " "),
			"subject":    joinNonEmpty(subject.Name, subject.NameEN, subject.GradeID),
			"syllabus":   syllabus.Name,
			"objectives": strings.Join(topicObjectives(topic), " "),
			"body":       joinNonEmpty(topic.OfficialRef, topic.Difficulty, topic.Tier, topic.Provenance),
		}), fieldTotals)

		if notes, ok := r.loader.GetTeachingNotes(topic.ID); ok && strings.TrimSpace(notes) != "" {
			for i, section := range splitTeachingNoteSections(notes) {
				r.addDoc(newRetrievalDoc("note:"+topic.ID+":"+strconv.Itoa(i), topic.ID, topic.SubjectID, topic.SyllabusID, form, "teaching_note", map[string]string{
					"title":      topic.Name,
					"aliases":    strings.Join(topicAliases(topic), " "),
					"subject":    joinNonEmpty(subject.Name, subject.NameEN, subject.GradeID),
					"syllabus":   syllabus.Name,
					"heading":    section.Title,
					"objectives": strings.Join(topicObjectives(topic), " "),
					"body":       section.Body,
				}), fieldTotals)
			}
		}

		if assessment, ok := r.loader.GetAssessment(topic.ID); ok {
			for i, question := range assessment.Questions {
				r.addDoc(newRetrievalDoc("assessment:"+topic.ID+":"+strconv.Itoa(i), topic.ID, topic.SubjectID, topic.SyllabusID, form, "assessment", map[string]string{
					"title":      topic.Name,
					"aliases":    strings.Join(topicAliases(topic), " "),
					"subject":    joinNonEmpty(subject.Name, subject.NameEN, subject.GradeID),
					"syllabus":   syllabus.Name,
					"heading":    question.Text,
					"objectives": strings.Join(topicObjectives(topic), " "),
					"body":       joinNonEmpty(question.Answer.Working, joinHints(question.Hints), joinDistractors(question.Distractors)),
				}), fieldTotals)
			}
		}
	}

	if len(r.docs) == 0 {
		return
	}
	for field, total := range fieldTotals {
		r.avgFieldLen[field] = float64(total) / float64(len(r.docs))
	}
}

func (r *curriculumRetriever) addDoc(doc retrievalDoc, fieldTotals map[string]int) {
	if len(doc.Terms) == 0 {
		return
	}
	seen := map[string]struct{}{}
	for field, tf := range doc.Terms {
		fieldTotals[field] += doc.Lengths[field]
		for term := range tf {
			seen[term] = struct{}{}
		}
	}
	for term := range seen {
		r.docFreq[term]++
	}
	r.docs = append(r.docs, doc)
}

func (r *curriculumRetriever) scoreDocs(terms []string, targetForm string, allowFormFallback bool, progressBoosts, neighborBoosts map[string]float64, query ContextQuery) []scoredDoc {
	followUp := looksLikeFollowUp(query.Text)
	var out []scoredDoc
	for _, doc := range r.docs {
		if targetForm != "" && doc.Form != "" && doc.Form != targetForm && !allowFormFallback {
			continue
		}

		score, matchedTerms, highSignalTerms := r.bm25Score(doc, terms)
		if score == 0 && (!followUp || query.ConversationTopicID == "" || doc.TopicID != query.ConversationTopicID) {
			continue
		}
		if allowFormFallback && targetForm != "" && doc.Form != "" && doc.Form != targetForm {
			score -= formMismatchPenalty
		}
		if query.ConversationTopicID != "" && doc.TopicID == query.ConversationTopicID && (score > 0 || followUp) {
			score += 0.9
		}
		if boost := progressBoosts[doc.TopicID]; boost > 0 {
			score += boost
		}
		if boost := neighborBoosts[doc.TopicID]; boost > 0 {
			score += boost
		}
		if score > 0 {
			out = append(out, scoredDoc{
				Doc:             doc,
				Score:           score,
				MatchedTerms:    matchedTerms,
				HighSignalTerms: highSignalTerms,
			})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Score == out[j].Score {
			return out[i].Doc.ID < out[j].Doc.ID
		}
		return out[i].Score > out[j].Score
	})
	if len(out) > 24 {
		out = out[:24]
	}
	return out
}

func (r *curriculumRetriever) resolveActiveFollowUp(query ContextQuery, terms []string) (retrievalResult, bool) {
	if query.ConversationTopicID == "" || !looksLikeFollowUp(query.Text) || !isGenericFollowUpQuery(terms) {
		return retrievalResult{}, false
	}

	topic, ok := r.topics[query.ConversationTopicID]
	if !ok {
		return retrievalResult{}, false
	}

	docs := r.scoredDocsForTopic(query.ConversationTopicID, terms)
	if len(docs) == 0 {
		return retrievalResult{}, false
	}

	score := docs[0].Score + 1.5
	return retrievalResult{
		Topic:      &topic,
		Notes:      r.composeNotes(scoredTopic{Topic: topic, Score: score, Docs: docs, Confidence: "follow_up"}),
		Score:      score,
		Confidence: "follow_up",
	}, true
}

func isGenericFollowUpQuery(tokens []string) bool {
	if len(tokens) == 0 {
		return false
	}
	for _, token := range tokens {
		if _, ok := genericFollowUpTokens[token]; !ok {
			return false
		}
	}
	return true
}

func (r *curriculumRetriever) topicHasWeakSpecificCoverage(topicID string, terms []string) bool {
	vocab := make(map[string]struct{})
	matchedContentTerms := 0
	hasUnmatchedSpecific := false

	if r.service != nil {
		documents, _ := r.service.ListDocument(retrieval.ListDocumentsRequest{
			CollectionIDs:   []string{"curriculum:" + r.topics[topicID].SubjectID},
			Metadata:        map[string]string{"topic_id": topicID},
			IncludeInactive: true,
		})
		for _, document := range documents {
			for _, term := range tokenizeRetrievalText(document.Title + " " + flattenMetadataMap(document.Metadata) + " " + strings.Join(document.Tags, " ") + " " + document.Body) {
				vocab[term] = struct{}{}
			}
		}
	} else {
		for _, doc := range r.docs {
			if doc.TopicID != topicID {
				continue
			}
			for _, tf := range doc.Terms {
				for term := range tf {
					vocab[term] = struct{}{}
				}
			}
		}
	}

	for _, term := range terms {
		if _, ok := vocab[term]; ok {
			if _, generic := genericFollowUpTokens[term]; !generic {
				matchedContentTerms++
			}
			continue
		}
		if _, ok := genericFollowUpTokens[term]; ok {
			continue
		}
		hasUnmatchedSpecific = true
	}

	return hasUnmatchedSpecific && matchedContentTerms <= 1
}

func (r *curriculumRetriever) rankTopics(docs []scoredDoc) []scoredTopic {
	grouped := map[string][]scoredDoc{}
	for _, doc := range docs {
		grouped[doc.Doc.TopicID] = append(grouped[doc.Doc.TopicID], doc)
	}

	out := make([]scoredTopic, 0, len(grouped))
	for topicID, topicDocs := range grouped {
		sort.Slice(topicDocs, func(i, j int) bool {
			return topicDocs[i].Score > topicDocs[j].Score
		})
		score := topicDocs[0].Score
		if len(topicDocs) > 1 {
			score += topicDocs[1].Score * 0.35
		}
		if len(topicDocs) > 2 {
			score += topicDocs[2].Score * 0.15
		}
		out = append(out, scoredTopic{
			Topic:      r.topics[topicID],
			Score:      score,
			Docs:       topicDocs,
			Confidence: topicDocs[0].Doc.Kind,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Score == out[j].Score {
			return out[i].Topic.ID < out[j].Topic.ID
		}
		return out[i].Score > out[j].Score
	})
	return out
}

func (r *curriculumRetriever) scoredDocsForTopic(topicID string, terms []string) []scoredDoc {
	if r.service != nil {
		results, err := r.service.Search(retrieval.SearchRequest{
			Query:           strings.Join(terms, " "),
			Metadata:        map[string]string{"topic_id": topicID},
			IncludeInactive: false,
			Limit:           6,
		})
		if err != nil {
			return nil
		}
		return r.scoredDocsFromSearchResults(results, nil, nil, ContextQuery{ConversationTopicID: topicID, Text: strings.Join(terms, " ")}, "", false)
	}

	out := make([]scoredDoc, 0, 4)
	for _, doc := range r.docs {
		if doc.TopicID != topicID {
			continue
		}

		score, matchedTerms, highSignalTerms := r.bm25Score(doc, terms)
		switch doc.Kind {
		case "teaching_note":
			score += 1.2
		case "topic":
			score += 0.8
		default:
			score += 0.4
		}

		out = append(out, scoredDoc{
			Doc:             doc,
			Score:           score,
			MatchedTerms:    matchedTerms,
			HighSignalTerms: highSignalTerms,
		})
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].Score == out[j].Score {
			return out[i].Doc.ID < out[j].Doc.ID
		}
		return out[i].Score > out[j].Score
	})
	return out
}

func flattenMetadataMap(metadata map[string]string) string {
	if len(metadata) == 0 {
		return ""
	}
	parts := make([]string, 0, len(metadata)*2)
	for key, value := range metadata {
		parts = append(parts, key, value)
	}
	return strings.Join(parts, " ")
}

func (r *curriculumRetriever) composeNotes(topic scoredTopic) string {
	var parts []string
	used := map[string]struct{}{}
	size := 0
	appendDoc := func(doc retrievalDoc) {
		if _, ok := used[doc.ID]; ok {
			return
		}
		text := strings.TrimSpace(joinNonEmpty(doc.Fields["heading"], doc.Fields["objectives"], doc.Fields["body"]))
		if text == "" {
			return
		}
		used[doc.ID] = struct{}{}
		parts = append(parts, text)
		size += len(text)
	}

	for _, doc := range topic.Docs {
		if doc.Doc.Kind == "teaching_note" {
			appendDoc(doc.Doc)
		}
		if size >= 1800 || len(parts) >= 3 {
			break
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return truncateForPrompt(strings.Join(parts, "\n\n"), 1800)
}

func (r *curriculumRetriever) bm25Score(doc retrievalDoc, terms []string) (float64, int, int) {
	score := 0.0
	matchedTerms := 0
	highSignalTerms := 0
	for _, term := range terms {
		df := r.docFreq[term]
		if df == 0 {
			continue
		}
		idf := math.Log(1 + (float64(len(r.docs)-df)+0.5)/(float64(df)+0.5))
		termMatched := false
		termHighSignal := false
		for field, weight := range fieldWeights {
			tf := doc.Terms[field][term]
			if tf == 0 {
				continue
			}
			termMatched = true
			if field == "title" || field == "aliases" || field == "objectives" || field == "heading" {
				termHighSignal = true
			}
			fieldLen := float64(doc.Lengths[field])
			avgLen := r.avgFieldLen[field]
			if avgLen <= 0 {
				avgLen = 1
			}
			norm := float64(tf) + bm25K1*(1-bm25B+bm25B*(fieldLen/avgLen))
			score += weight * idf * ((float64(tf) * (bm25K1 + 1)) / norm)
		}
		if termMatched {
			matchedTerms++
		}
		if termHighSignal {
			highSignalTerms++
		}
	}
	return score, matchedTerms, highSignalTerms
}

func newRetrievalDoc(id, topicID, subjectID, syllabusID, form, kind string, fields map[string]string) retrievalDoc {
	doc := retrievalDoc{
		ID:         id,
		TopicID:    topicID,
		SubjectID:  subjectID,
		SyllabusID: syllabusID,
		Form:       form,
		Kind:       kind,
		Fields:     fields,
		Terms:      make(map[string]map[string]int),
		Lengths:    make(map[string]int),
	}
	for field, text := range fields {
		tokens := tokenizeRetrievalText(text)
		if len(tokens) == 0 {
			continue
		}
		tf := make(map[string]int, len(tokens))
		for _, token := range tokens {
			tf[token]++
		}
		doc.Terms[field] = tf
		doc.Lengths[field] = len(tokens)
	}
	return doc
}

func tokenizeRetrievalText(text string) []string {
	raw := tokenize(text)
	if len(raw) == 0 {
		return nil
	}
	tokens := make([]string, 0, len(raw))
	for _, token := range raw {
		if len(token) < 2 {
			continue
		}
		token = normalizeRetrievalToken(token)
		if token == "" {
			continue
		}
		if _, ok := retrievalStopWords[token]; ok {
			continue
		}
		tokens = append(tokens, token)
	}
	return tokens
}

func normalizeRetrievalToken(token string) string {
	if stem, ok := stemOverrides[token]; ok {
		return stem
	}
	switch {
	case strings.HasSuffix(token, "ies") && len(token) > 4:
		return token[:len(token)-3] + "y"
	case strings.HasSuffix(token, "es") && len(token) > 4:
		return token[:len(token)-2]
	case strings.HasSuffix(token, "s") && len(token) > 4:
		return token[:len(token)-1]
	default:
		return token
	}
}

func topicAliases(topic curriculum.Topic) []string {
	return []string{
		topic.Name,
		strings.ReplaceAll(topic.Name, "(", " "),
		strings.ReplaceAll(topic.Name, ")", " "),
	}
}

func topicObjectives(topic curriculum.Topic) []string {
	objectives := make([]string, 0, len(topic.LearningObjectives))
	for _, objective := range topic.LearningObjectives {
		objectives = append(objectives, objective.Text)
	}
	return objectives
}

func inferTopicForm(topic curriculum.Topic, subject curriculum.Subject) string {
	if form := detectFormInText(subject.GradeID); form != "" {
		return form
	}
	if form := detectFormInText(subject.Name); form != "" {
		return form
	}
	if form := detectFormInText(topic.SubjectID); form != "" {
		return form
	}
	return detectFormInText(topic.SyllabusID)
}

func detectFormInText(text string) string {
	match := formPattern.FindStringSubmatch(strings.ToLower(strings.TrimSpace(text)))
	if len(match) == 2 {
		return match[1]
	}
	return ""
}

func looksLikeFollowUp(text string) bool {
	tokens := tokenizeRetrievalText(text)
	if len(tokens) == 0 || len(tokens) > 10 {
		return false
	}
	for _, token := range tokens {
		if _, ok := followUpKeywords[token]; ok {
			return true
		}
	}
	return false
}

type noteSection struct {
	Title string
	Body  string
}

func splitTeachingNoteSections(markdown string) []noteSection {
	lines := strings.Split(markdown, "\n")
	var out []noteSection
	current := noteSection{Title: "Overview"}
	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		if headingPattern.MatchString(line) {
			if strings.TrimSpace(current.Body) != "" {
				out = append(out, current)
			}
			current = noteSection{Title: headingPattern.ReplaceAllString(line, "")}
			continue
		}
		if current.Body != "" {
			current.Body += "\n"
		}
		current.Body += line
	}
	if strings.TrimSpace(current.Body) != "" {
		out = append(out, current)
	}
	if len(out) == 0 && strings.TrimSpace(markdown) != "" {
		return []noteSection{{Title: "Overview", Body: markdown}}
	}
	return out
}

func joinHints(hints []curriculum.AssessmentHint) string {
	parts := make([]string, 0, len(hints))
	for _, hint := range hints {
		parts = append(parts, hint.Text)
	}
	return strings.Join(parts, " ")
}

func joinDistractors(distractors []curriculum.AssessmentDistractor) string {
	parts := make([]string, 0, len(distractors))
	for _, distractor := range distractors {
		parts = append(parts, distractor.Value, distractor.Feedback)
	}
	return strings.Join(parts, " ")
}

func joinNonEmpty(parts ...string) string {
	filtered := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			filtered = append(filtered, part)
		}
	}
	return strings.Join(filtered, " ")
}
