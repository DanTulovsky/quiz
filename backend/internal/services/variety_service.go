package services

import (
	"context"
	"math/rand"

	"go.opentelemetry.io/otel/attribute"

	"quizapp/internal/config"
	"quizapp/internal/observability"
)

// VarietyService handles the selection of variety elements for question generation
type VarietyService struct {
	cfg    *config.Config
	logger *observability.Logger
}

// VarietyElements holds the randomly selected variety elements for a question generation request
type VarietyElements struct {
	TopicCategory      string
	GrammarFocus       string
	VocabularyDomain   string
	Scenario           string
	StyleModifier      string
	DifficultyModifier string
	TimeContext        string
}

// NewVarietyServiceWithLogger creates a new VarietyService with logger
func NewVarietyServiceWithLogger(cfg *config.Config, logger *observability.Logger) *VarietyService {
	return &VarietyService{
		cfg:    cfg,
		logger: logger,
	}
}

// SelectVarietyElements randomly selects variety elements for question generation
// If highPriorityTopics or userWeakAreas are provided, bias topic selection toward those topics first, then gapAnalysis.
func (vs *VarietyService) SelectVarietyElements(ctx context.Context, level string, highPriorityTopics, userWeakAreas []string, gapAnalysis map[string]int) *VarietyElements {
	_, span := observability.TraceVarietyFunction(ctx, "select_variety_elements",
		attribute.String("variety.level", level),
		attribute.Int("variety.high_priority_topics_count", len(highPriorityTopics)),
		attribute.Int("variety.user_weak_areas_count", len(userWeakAreas)),
		attribute.Int("variety.gap_analysis_count", len(gapAnalysis)),
	)
	defer span.End()

	// Get variety configuration from config
	if vs.cfg.Variety != nil {
		variety := vs.cfg.Variety
		elements := &VarietyElements{}

		// Helper function to get weighted selection from gap analysis
		getWeightedSelection := func(gapType string, availableOptions []string) string {
			if len(gapAnalysis) == 0 || len(availableOptions) == 0 {
				return ""
			}

			var weightedOptions []string
			for _, option := range availableOptions {
				gapKey := gapType + "_" + option
				if count, ok := gapAnalysis[gapKey]; ok && count > 0 {
					// Intensify weighting by squaring the severity to reduce randomness sensitivity
					weight := count * count
					for range weight {
						weightedOptions = append(weightedOptions, option)
					}
				}
			}

			if len(weightedOptions) > 0 {
				return weightedOptions[rand.Intn(len(weightedOptions))]
			}
			return ""
		}

		// Define all possible variety elements with their selection functions
		type varietySelector struct {
			name     string
			selector func() string
		}

		var selectors []varietySelector

		// Topic category selector (biased by userWeakAreas, highPriorityTopics, then gapAnalysis if provided)
		if len(variety.TopicCategories) > 0 {
			selectors = append(selectors, varietySelector{
				name: "topic_category",
				selector: func() string {
					// 1. UserWeakAreas
					if len(userWeakAreas) > 0 {
						var matching []string
						for _, topic := range variety.TopicCategories {
							for _, weak := range userWeakAreas {
								if topic == weak {
									matching = append(matching, topic)
								}
							}
						}
						if len(matching) > 0 {
							elements.TopicCategory = matching[rand.Intn(len(matching))]
							return elements.TopicCategory
						}
					}
					// 2. HighPriorityTopics
					if len(highPriorityTopics) > 0 {
						var matching []string
						for _, topic := range variety.TopicCategories {
							for _, high := range highPriorityTopics {
								if topic == high {
									matching = append(matching, topic)
								}
							}
						}
						if len(matching) > 0 {
							elements.TopicCategory = matching[rand.Intn(len(matching))]
							return elements.TopicCategory
						}
					}
					// 3. GapAnalysis for topics
					if selected := getWeightedSelection("topic_category", variety.TopicCategories); selected != "" {
						elements.TopicCategory = selected
						return elements.TopicCategory
					}
					// Fallback to random
					elements.TopicCategory = variety.TopicCategories[rand.Intn(len(variety.TopicCategories))]
					return elements.TopicCategory
				},
			})
		}

		// Grammar focus selector (now with gap analysis support)
		if grammarByLevel, exists := variety.GrammarFocusByLevel[level]; exists && len(grammarByLevel) > 0 {
			selectors = append(selectors, varietySelector{
				name: "grammar_focus",
				selector: func() string {
					// Check for grammar gaps first
					if selected := getWeightedSelection("grammar_focus", grammarByLevel); selected != "" {
						elements.GrammarFocus = selected
						return elements.GrammarFocus
					}
					// Fallback to random
					elements.GrammarFocus = grammarByLevel[rand.Intn(len(grammarByLevel))]
					return elements.GrammarFocus
				},
			})
		} else if len(variety.GrammarFocus) > 0 {
			selectors = append(selectors, varietySelector{
				name: "grammar_focus",
				selector: func() string {
					// Check for grammar gaps first
					if selected := getWeightedSelection("grammar_focus", variety.GrammarFocus); selected != "" {
						elements.GrammarFocus = selected
						return elements.GrammarFocus
					}
					// Fallback to random
					elements.GrammarFocus = variety.GrammarFocus[rand.Intn(len(variety.GrammarFocus))]
					return elements.GrammarFocus
				},
			})
		}

		// Vocabulary domain selector (now with gap analysis support)
		if len(variety.VocabularyDomains) > 0 {
			selectors = append(selectors, varietySelector{
				name: "vocabulary_domain",
				selector: func() string {
					// Check for vocabulary gaps first
					if selected := getWeightedSelection("vocabulary_domain", variety.VocabularyDomains); selected != "" {
						elements.VocabularyDomain = selected
						return elements.VocabularyDomain
					}
					// Fallback to random
					elements.VocabularyDomain = variety.VocabularyDomains[rand.Intn(len(variety.VocabularyDomains))]
					return elements.VocabularyDomain
				},
			})
		}

		// Scenario selector (now with gap analysis support)
		if len(variety.Scenarios) > 0 {
			selectors = append(selectors, varietySelector{
				name: "scenario",
				selector: func() string {
					// Check for scenario gaps first
					if selected := getWeightedSelection("scenario", variety.Scenarios); selected != "" {
						elements.Scenario = selected
						return elements.Scenario
					}
					// Fallback to random
					elements.Scenario = variety.Scenarios[rand.Intn(len(variety.Scenarios))]
					return elements.Scenario
				},
			})
		}

		// Style modifier selector
		if len(variety.StyleModifiers) > 0 {
			selectors = append(selectors, varietySelector{
				name: "style_modifier",
				selector: func() string {
					elements.StyleModifier = variety.StyleModifiers[rand.Intn(len(variety.StyleModifiers))]
					return elements.StyleModifier
				},
			})
		}

		// Difficulty modifier selector
		if len(variety.DifficultyModifiers) > 0 {
			selectors = append(selectors, varietySelector{
				name: "difficulty_modifier",
				selector: func() string {
					elements.DifficultyModifier = variety.DifficultyModifiers[rand.Intn(len(variety.DifficultyModifiers))]
					return elements.DifficultyModifier
				},
			})
		}

		// Time context selector
		if len(variety.TimeContexts) > 0 {
			selectors = append(selectors, varietySelector{
				name: "time_context",
				selector: func() string {
					elements.TimeContext = variety.TimeContexts[rand.Intn(len(variety.TimeContexts))]
					return elements.TimeContext
				},
			})
		}

		// Randomly select 2-3 variety elements (instead of all 7)
		numToSelect := 2
		if len(selectors) > 2 {
			// 70% chance of 2 elements, 30% chance of 3 elements
			if rand.Float64() < 0.3 {
				numToSelect = 3
			}
		}

		// Shuffle and select the first numToSelect elements
		rand.Shuffle(len(selectors), func(i, j int) {
			selectors[i], selectors[j] = selectors[j], selectors[i]
		})

		// Apply the selected variety elements
		for i := 0; i < numToSelect && i < len(selectors); i++ {
			selected := selectors[i].selector()
			span.SetAttributes(attribute.String("variety."+selectors[i].name, selected))
		}

		span.SetAttributes(
			attribute.String("variety.topic_category", elements.TopicCategory),
			attribute.String("variety.grammar_focus", elements.GrammarFocus),
			attribute.String("variety.vocabulary_domain", elements.VocabularyDomain),
			attribute.String("variety.scenario", elements.Scenario),
			attribute.String("variety.style_modifier", elements.StyleModifier),
			attribute.String("variety.difficulty_modifier", elements.DifficultyModifier),
			attribute.String("variety.time_context", elements.TimeContext),
			attribute.Int("variety.elements_selected", numToSelect),
		)

		span.SetAttributes(attribute.String("variety.result", "success"))
		return elements
	}

	span.SetAttributes(attribute.String("variety.result", "no_config"))
	return &VarietyElements{} // Return empty if no variety config
}

// SelectMultipleVarietyElements selects multiple sets of variety elements for batch generation
func (vs *VarietyService) SelectMultipleVarietyElements(ctx context.Context, level string, count int) []*VarietyElements {
	ctx, span := observability.TraceVarietyFunction(ctx, "select_multiple_variety_elements",
		attribute.String("variety.level", level),
		attribute.Int("variety.count", count),
	)
	defer span.End()

	elements := make([]*VarietyElements, count)
	for i := 0; i < count; i++ {
		elements[i] = vs.SelectVarietyElements(ctx, level, nil, nil, nil)
	}

	span.SetAttributes(attribute.String("variety.result", "success"), attribute.Int("variety.elements_count", len(elements)))
	return elements
}
