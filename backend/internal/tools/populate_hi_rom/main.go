// Command populate_hi_rom populates full tense blocks for romanized Hindi verb JSONs
// by using karna.json as a template and applying simple stem/irregular rules.
// Command populate_hi_rom generates or updates Hindi verb conjugation JSON files
// by applying irregular stem rules to a template verb and writing results for
// several target verbs. It is a developer utility, not part of runtime.
package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type Conjugation struct {
	Pronoun           string `json:"pronoun"`
	Form              string `json:"form"`
	ExampleSentence   string `json:"exampleSentence"`
	ExampleSentenceEn string `json:"exampleSentenceEn"`
}

type Tense struct {
	TenseID      string        `json:"tenseId"`
	TenseName    string        `json:"tenseName"`
	TenseNameEn  string        `json:"tenseNameEn"`
	Description  string        `json:"description"`
	Conjugations []Conjugation `json:"conjugations"`
}

type Verb struct {
	Language     string  `json:"language"`
	LanguageName string  `json:"languageName"`
	Infinitive   string  `json:"infinitive"`
	InfinitiveEn string  `json:"infinitiveEn"`
	Slug         string  `json:"slug"`
	Category     string  `json:"category"`
	Tenses       []Tense `json:"tenses"`
}

type irregular struct {
	habitualStem   string // for करता/करती → habitualStem + "ता/ती"
	continuousStem string // for कर रहा → continuousStem + " रहा"
	perfective     string // for किया → perfective
}

var irregulars = map[string]irregular{
	"karna":   {habitualStem: "कर", continuousStem: "कर", perfective: "किया"},
	"aana":    {habitualStem: "आ", continuousStem: "आ", perfective: "आया"},
	"jana":    {habitualStem: "जा", continuousStem: "जा", perfective: "गया"},
	"dena":    {habitualStem: "दे", continuousStem: "दे", perfective: "दिया"},
	"lena":    {habitualStem: "ले", continuousStem: "ले", perfective: "लिया"},
	"kahana":  {habitualStem: "कह", continuousStem: "कह", perfective: "कहा"},
	"dekhana": {habitualStem: "देख", continuousStem: "देख", perfective: "देखा"},
	"sochana": {habitualStem: "सोच", continuousStem: "सोच", perfective: "सोचा"},
	// होना handled specially for present simple; otherwise use stems
	"hona": {habitualStem: "हो", continuousStem: "हो", perfective: "हुआ"},
	// जानना
	"janana": {habitualStem: "जान", continuousStem: "जान", perfective: "जाना"},
}

// Pronoun order must match the template files: मैं, तुम, वह/यह, हम, आप, वे/ये
var pronounToIndex = map[string]int{
	"मैं":   0,
	"तुम":   1,
	"वह/यह": 2,
	"हम":    3,
	"आप":    4,
	"वे/ये": 5,
}

type examples struct {
	Hi [6]string
	En [6]string
}

// Curated per-verb, per-tense examples (Hindi and English), aligned by pronoun index.
var exampleDB = map[string]map[string]examples{
	"aana": {
		"present_simple":             {Hi: [6]string{"मैं घर आता/आती हूँ।", "तुम कब आते हो?", "वह जल्दी आता/आती है।", "हम साथ आते हैं।", "आप भी आते हैं।", "वे यहाँ आते हैं।"}, En: [6]string{"I come home.", "When do you come?", "He/She comes early.", "We come together.", "You also come.", "They come here."}},
		"present_continuous":         {Hi: [6]string{"मैं आ रहा/रही हूँ।", "तुम आ रहे हो।", "वह/यह आ रहा/रही है।", "हम आ रहे हैं।", "आप आ रहे हैं।", "वे/ये आ रहे हैं।"}, En: [6]string{"I am coming.", "You are coming.", "He/She is coming.", "We are coming.", "You are coming.", "They are coming."}},
		"present_perfect":            {Hi: [6]string{"मैं आ चुका/चुकी हूँ।", "तुम आ चुके हो।", "वह/यह आ चुका/चुकी है।", "हम आ चुके हैं।", "आप आ चुके हैं।", "वे/ये आ चुके हैं।"}, En: [6]string{"I have come.", "You have come.", "He/She has come.", "We have come.", "You have come.", "They have come."}},
		"present_perfect_continuous": {Hi: [6]string{"मैं आता/आती आ रहा/रही हूँ।", "तुम आते आ रहे हो।", "वह/यह आता/आती आ रहा/रही है।", "हम आते आ रहे हैं।", "आप आते आ रहे हैं।", "वे/ये आते आ रहे हैं।"}, En: [6]string{"I have been coming.", "You have been coming.", "He/She has been coming.", "We have been coming.", "You have been coming.", "They have been coming."}},
		"past_simple":                {Hi: [6]string{"मैं आया।", "तुम आए।", "वह/यह आया/आई।", "हम आए।", "आप आए।", "वे/ये आए।"}, En: [6]string{"I came.", "You came.", "He/She came.", "We came.", "You came.", "They came."}},
		"past_continuous":            {Hi: [6]string{"मैं आ रहा/रही था/थी।", "तुम आ रहे थे।", "वह/यह आ रहा/रही था/थी।", "हम आ रहे थे।", "आप आ रहे थे।", "वे/ये आ रहे थे।"}, En: [6]string{"I was coming.", "You were coming.", "He/She was coming.", "We were coming.", "You were coming.", "They were coming."}},
		"past_perfect":               {Hi: [6]string{"मैं आ चुका/चुकी था/थी।", "तुम आ चुके थे।", "वह/यह आ चुका/चुकी था/थी।", "हम आ चुके थे।", "आप आ चुके थे।", "वे/ये आ चुके थे।"}, En: [6]string{"I had come.", "You had come.", "He/She had come.", "We had come.", "You had come.", "They had come."}},
		"past_perfect_continuous":    {Hi: [6]string{"मैं आता/आती आ रहा/रही था/थी।", "तुम आते आ रहे थे।", "वह/यह आता/आती आ रहा/रही था/थी।", "हम आते आ रहे थे।", "आप आते आ रहे थे।", "वे/ये आते आ रहे थे।"}, En: [6]string{"I had been coming.", "You had been coming.", "He/She had been coming.", "We had been coming.", "You had been coming.", "They had been coming."}},
		"future_simple":              {Hi: [6]string{"मैं आऊँगा/आऊँगी।", "तुम आओगे।", "वह/यह आएगा/आएगी।", "हम आएंगे।", "आप आएंगे।", "वे/ये आएंगे।"}, En: [6]string{"I will come.", "You will come.", "He/She will come.", "We will come.", "You will come.", "They will come."}},
		"future_continuous":          {Hi: [6]string{"मैं आ रहा/रही होऊँगा/हूँगी।", "तुम आ रहे होगे।", "वह/यह आ रहा/रही होगा/होगी।", "हम आ रहे होंगे।", "आप आ रहे होंगे।", "वे/ये आ रहे होंगे।"}, En: [6]string{"I will be coming.", "You will be coming.", "He/She will be coming.", "We will be coming.", "You will be coming.", "They will be coming."}},
		"future_perfect":             {Hi: [6]string{"मैं आ चुका/चुकी होऊँगा/हूँगी।", "तुम आ चुके होगे।", "वह/यह आ चुका/चुकी होगा/होगी।", "हम आ चुके होंगे।", "आप आ चुके होंगे।", "वे/ये आ चुके होंगे।"}, En: [6]string{"I will have come.", "You will have come.", "He/She will have come.", "We will have come.", "You will have come.", "They will have come."}},
		"future_perfect_continuous":  {Hi: [6]string{"मैं आता/आती आ रहा/रही होऊँगा/हूँगी।", "तुम आते आ रहे होगे।", "वह/यह आता/आती आ रहा/रही होगा/होगी।", "हम आते आ रहे होंगे।", "आप आते आ रहे होंगे।", "वे/ये आते आ रहे होंगे।"}, En: [6]string{"I will have been coming.", "You will have been coming.", "He/She will have been coming.", "We will have been coming.", "You will have been coming.", "They will have been coming."}},
	},
	"jana": {
		"present_simple":             {Hi: [6]string{"मैं स्कूल जाता/जाती हूँ।", "तुम कहाँ जाते हो?", "वह बाज़ार जाता/जाती है।", "हम एक साथ जाते हैं।", "आप रोज़ जाते हैं।", "वे मन्दिर जाते हैं।"}, En: [6]string{"I go to school.", "Where do you go?", "He/She goes to the market.", "We go together.", "You go daily.", "They go to the temple."}},
		"present_continuous":         {Hi: [6]string{"मैं जा रहा/रही हूँ।", "तुम जा रहे हो।", "वह/यह जा रहा/रही है।", "हम जा रहे हैं।", "आप जा रहे हैं।", "वे/ये जा रहे हैं।"}, En: [6]string{"I am going.", "You are going.", "He/She is going.", "We are going.", "You are going.", "They are going."}},
		"present_perfect":            {Hi: [6]string{"मैं जा चुका/चुकी हूँ।", "तुम जा चुके हो।", "वह/यह जा चुका/चुकी है।", "हम जा चुके हैं।", "आप जा चुके हैं।", "वे/ये जा चुके हैं।"}, En: [6]string{"I have gone.", "You have gone.", "He/She has gone.", "We have gone.", "You have gone.", "They have gone."}},
		"present_perfect_continuous": {Hi: [6]string{"मैं जाता/जाती आ रहा/रही हूँ।", "तुम जाते आ रहे हो।", "वह/यह जाता/जाती आ रहा/रही है।", "हम जाते आ रहे हैं।", "आप जाते आ रहे हैं।", "वे/ये जाते आ रहे हैं।"}, En: [6]string{"I have been going.", "You have been going.", "He/She has been going.", "We have been going.", "You have been going.", "They have been going."}},
		"past_simple":                {Hi: [6]string{"मैं गया/गई।", "तुम गए।", "वह/यह गया/गई।", "हम गए।", "आप गए।", "वे/ये गए।"}, En: [6]string{"I went.", "You went.", "He/She went.", "We went.", "You went.", "They went."}},
		"past_continuous":            {Hi: [6]string{"मैं जा रहा/रही था/थी।", "तुम जा रहे थे।", "वह/यह जा रहा/रही था/थी।", "हम जा रहे थे।", "आप जा रहे थे।", "वे/ये जा रहे थे।"}, En: [6]string{"I was going.", "You were going.", "He/She was going.", "We were going.", "You were going.", "They were going."}},
		"past_perfect":               {Hi: [6]string{"मैं जा चुका/चुकी था/थी।", "तुम जा चुके थे।", "वह/यह जा चुका/चुकी था/थी।", "हम जा चुके थे।", "आप जा चुके थे।", "वे/ये जा चुके थे।"}, En: [6]string{"I had gone.", "You had gone.", "He/She had gone.", "We had gone.", "You had gone.", "They had gone."}},
		"past_perfect_continuous":    {Hi: [6]string{"मैं जाता/जाती आ रहा/रही था/थी।", "तुम जाते आ रहे थे।", "वह/यह जाता/जाती आ रहा/रही था/थी।", "हम जाते आ रहे थे।", "आप जाते आ रहे थे।", "वे/ये जाते आ रहे थे।"}, En: [6]string{"I had been going.", "You had been going.", "He/She had been going.", "We had been going.", "You had been going.", "They had been going."}},
		"future_simple":              {Hi: [6]string{"मैं जाऊँगा/जाऊँगी।", "तुम जाओगे।", "वह/यह जाएगा/जाएगी।", "हम जाएंगे।", "आप जाएंगे।", "वे/ये जाएंगे।"}, En: [6]string{"I will go.", "You will go.", "He/She will go.", "We will go.", "You will go.", "They will go."}},
		"future_continuous":          {Hi: [6]string{"मैं जा रहा/रही होऊँगा/हूँगी।", "तुम जा रहे होगे।", "वह/यह जा रहा/रही होगा/होगी।", "हम जा रहे होंगे।", "आप जा रहे होंगे।", "वे/ये जा रहे होंगे।"}, En: [6]string{"I will be going.", "You will be going.", "He/She will be going.", "We will be going.", "You will be going.", "They will be going."}},
		"future_perfect":             {Hi: [6]string{"मैं जा चुका/चुकी होऊँगा/हूँगी।", "तुम जा चुके होगे।", "वह/यह जा चुका/चुकी होगा/होगी।", "हम जा चुके होंगे।", "आप जा चुके होंगे।", "वे/ये जा चुके होंगे।"}, En: [6]string{"I will have gone.", "You will have gone.", "He/She will have gone.", "We will have gone.", "You will have gone.", "They will have gone."}},
		"future_perfect_continuous":  {Hi: [6]string{"मैं जाता/जाती आ रहा/रही होऊँगा/हूँगी।", "तुम जाते आ रहे होगे।", "वह/यह जाता/जाती आ रहा/रही होगा/होगी।", "हम जाते आ रहे होंगे।", "आप जाते आ रहे होंगे।", "वे/ये जाते आ रहे होंगे।"}, En: [6]string{"I will have been going.", "You will have been going.", "He/She will have been going.", "We will have been going.", "You will have been going.", "They will have been going."}},
	},
	"dena":    exampleSetForTransitive("दे", "देता/देती", "दिया", "दे रहा/रही", "दे चुके", "to give", "a book", "किताब"),
	"lena":    exampleSetForTransitive("ले", "लेता/लेती", "लिया", "ले रहा/रही", "ले चुके", "to take", "a book", "किताब"),
	"dekhana": exampleSetForTransitive("देख", "देखता/देखती", "देखा", "देख रहा/रही", "देख चुके", "to watch", "a movie", "फ़िल्म"),
	"sochana": exampleSetForIntransitive("सोच", "सोचता/सोचती", "सोचा", "सोच रहा/रही", "सोच चुके", "to think"),
	"kahana":  exampleSetForIntransitive("कह", "कहता/कहती", "कहा", "कह रहा/रही", "कह चुके", "to say"),
	"janana":  exampleSetForIntransitive("जान", "जानता/जानती", "जाना", "जान रहा/रही", "जान चुके", "to know"),
}

func exampleSetForTransitive(stemDeva, habitualMascFem, perfective, cont, perfectAux, enBase, enObject, hiObject string) map[string]examples {
	// Build simple object-using sentences across tenses
	ps := examples{Hi: [6]string{
		"मैं " + hiObject + " " + habitualMascFem + " हूँ।",
		"तुम " + hiObject + " " + pluralize(habitualMascFem) + " हो।",
		"वह/यह " + hiObject + " " + habitualMascFem + " है।",
		"हम " + hiObject + " " + pluralize(habitualMascFem) + " हैं।",
		"आप " + hiObject + " " + pluralize(habitualMascFem) + " हैं।",
		"वे/ये " + hiObject + " " + pluralize(habitualMascFem) + " हैं।",
	}, En: [6]string{
		"I " + strings.TrimPrefix(enBase, "to ") + " " + enObject + ".",
		"You " + strings.TrimPrefix(enBase, "to ") + " " + enObject + ".",
		"He/She " + strings.TrimPrefix(enBase, "to ") + " " + enObject + ".",
		"We " + strings.TrimPrefix(enBase, "to ") + " " + enObject + ".",
		"You " + strings.TrimPrefix(enBase, "to ") + " " + enObject + ".",
		"They " + strings.TrimPrefix(enBase, "to ") + " " + enObject + ".",
	}}
	pc := examples{Hi: [6]string{
		"मैं " + hiObject + " " + cont + " हूँ।",
		"तुम " + hiObject + " " + cont + " हो।",
		"वह/यह " + hiObject + " " + cont + " है।",
		"हम " + hiObject + " " + cont + " हैं।",
		"आप " + hiObject + " " + cont + " हैं।",
		"वे/ये " + hiObject + " " + cont + " हैं।",
	}, En: [6]string{"I am " + strings.TrimSuffix(strings.TrimPrefix(enBase, "to "), "e") + "ing " + enObject + ".", "You are " + strings.TrimSuffix(strings.TrimPrefix(enBase, "to "), "e") + "ing " + enObject + ".", "He/She is " + strings.TrimSuffix(strings.TrimPrefix(enBase, "to "), "e") + "ing " + enObject + ".", "We are " + strings.TrimSuffix(strings.TrimPrefix(enBase, "to "), "e") + "ing " + enObject + ".", "You are " + strings.TrimSuffix(strings.TrimPrefix(enBase, "to "), "e") + "ing " + enObject + ".", "They are " + strings.TrimSuffix(strings.TrimPrefix(enBase, "to "), "e") + "ing " + enObject + "."}}
	pp := examples{Hi: [6]string{"मैं " + hiObject + " दे चुका/चुकी हूँ।", "तुम " + hiObject + " दे चुके हो।", "वह/यह " + hiObject + " दे चुका/चुकी है।", "हम " + hiObject + " दे चुके हैं।", "आप " + hiObject + " दे चुके हैं।", "वे/ये " + hiObject + " दे चुके हैं।"}, En: [6]string{"I have " + pastParticiple(enBase) + " " + enObject + ".", "You have " + pastParticiple(enBase) + " " + enObject + ".", "He/She has " + pastParticiple(enBase) + " " + enObject + ".", "We have " + pastParticiple(enBase) + " " + enObject + ".", "You have " + pastParticiple(enBase) + " " + enObject + ".", "They have " + pastParticiple(enBase) + " " + enObject + "."}}
	// For transitive sets, replicate similar patterns across remaining tenses briefly
	return map[string]examples{
		"present_simple":             ps,
		"present_continuous":         pc,
		"present_perfect":            pp,
		"present_perfect_continuous": pc,
		"past_simple":                {Hi: [6]string{"मैंने " + hiObject + " " + perfective + "।", "तुमने " + hiObject + " " + perfective + "।", "उसने " + hiObject + " " + perfective + "।", "हमने " + hiObject + " " + perfective + "।", "आपने " + hiObject + " " + perfective + "।", "उन्होंने " + hiObject + " " + perfective + "।"}, En: [6]string{"I " + pastOf(enBase) + " " + enObject + ".", "You " + pastOf(enBase) + " " + enObject + ".", "He/She " + pastOf(enBase) + " " + enObject + ".", "We " + pastOf(enBase) + " " + enObject + ".", "You " + pastOf(enBase) + " " + enObject + ".", "They " + pastOf(enBase) + " " + enObject + "."}},
		"past_continuous":            pc,
		"past_perfect":               {Hi: [6]string{"मैं " + hiObject + " " + perfectAux + " था/थी।", "तुम " + hiObject + " " + perfectAux + " थे।", "वह/यह " + hiObject + " " + perfectAux + " था/थी।", "हम " + hiObject + " " + perfectAux + " थे।", "आप " + hiObject + " " + perfectAux + " थे।", "वे/ये " + hiObject + " " + perfectAux + " थे।"}, En: [6]string{"I had " + pastParticiple(enBase) + " " + enObject + ".", "You had " + pastParticiple(enBase) + " " + enObject + ".", "He/She had " + pastParticiple(enBase) + " " + enObject + ".", "We had " + pastParticiple(enBase) + " " + enObject + ".", "You had " + pastParticiple(enBase) + " " + enObject + ".", "They had " + pastParticiple(enBase) + " " + enObject + "."}},
		"past_perfect_continuous":    pc,
		"future_simple":              {Hi: [6]string{"मैं " + stemDeva + "ऊँगा/ऊँगी।", "तुम " + stemDeva + "ओगे।", "वह/यह " + stemDeva + "एगा/एगी।", "हम " + stemDeva + "एंगे।", "आप " + stemDeva + "एंगे।", "वे/ये " + stemDeva + "एंगे।"}, En: [6]string{"I will " + strings.TrimPrefix(enBase, "to ") + " " + enObject + ".", "You will " + strings.TrimPrefix(enBase, "to ") + " " + enObject + ".", "He/She will " + strings.TrimPrefix(enBase, "to ") + " " + enObject + ".", "We will " + strings.TrimPrefix(enBase, "to ") + " " + enObject + ".", "You will " + strings.TrimPrefix(enBase, "to ") + " " + enObject + ".", "They will " + strings.TrimPrefix(enBase, "to ") + " " + enObject + "."}},
		"future_continuous":          pc,
		"future_perfect":             {Hi: [6]string{"मैं " + hiObject + " " + perfectAux + " होऊँगा/हूँगी।", "तुम " + hiObject + " " + perfectAux + " होगे।", "वह/यह " + hiObject + " " + perfectAux + " होगा/होगी।", "हम " + hiObject + " " + perfectAux + " होंगे।", "आप " + hiObject + " " + perfectAux + " होंगे।", "वे/ये " + hiObject + " " + perfectAux + " होंगे।"}, En: [6]string{"I will have " + pastParticiple(enBase) + " " + enObject + ".", "You will have " + pastParticiple(enBase) + " " + enObject + ".", "He/She will have " + pastParticiple(enBase) + " " + enObject + ".", "We will have " + pastParticiple(enBase) + " " + enObject + ".", "You will have " + pastParticiple(enBase) + " " + enObject + ".", "They will have " + pastParticiple(enBase) + " " + enObject + "."}},
		"future_perfect_continuous":  pc,
	}
}

func exampleSetForIntransitive(stemDeva, habitualMascFem, perfective, cont, perfectAux, enBase string) map[string]examples {
	base := strings.TrimPrefix(enBase, "to ")
	ps := examples{Hi: [6]string{"मैं " + habitualMascFem + " हूँ।", "तुम " + pluralize(habitualMascFem) + " हो।", "वह/यह " + habitualMascFem + " है।", "हम " + pluralize(habitualMascFem) + " हैं।", "आप " + pluralize(habitualMascFem) + " हैं।", "वे/ये " + pluralize(habitualMascFem) + " हैं।"}, En: [6]string{"I " + base + ".", "You " + base + ".", "He/She " + base + "s.", "We " + base + ".", "You " + base + ".", "They " + base + "."}}
	pc := examples{Hi: [6]string{"मैं " + cont + " हूँ।", "तुम " + cont + " हो।", "वह/यह " + cont + " है।", "हम " + cont + " हैं।", "आप " + cont + " हैं।", "वे/ये " + cont + " हैं।"}, En: [6]string{"I am " + base + "ing.", "You are " + base + "ing.", "He/She is " + base + "ing.", "We are " + base + "ing.", "You are " + base + "ing.", "They are " + base + "ing."}}
	pp := examples{Hi: [6]string{"मैं " + perfectAux + " हूँ।", "तुम " + perfectAux + " हो।", "वह/यह " + perfectAux + " है।", "हम " + perfectAux + " हैं।", "आप " + perfectAux + " हैं।", "वे/ये " + perfectAux + " हैं।"}, En: [6]string{"I have " + pastParticiple(enBase) + ".", "You have " + pastParticiple(enBase) + ".", "He/She has " + pastParticiple(enBase) + ".", "We have " + pastParticiple(enBase) + ".", "You have " + pastParticiple(enBase) + ".", "They have " + pastParticiple(enBase) + "."}}
	return map[string]examples{
		"present_simple":             ps,
		"present_continuous":         pc,
		"present_perfect":            pp,
		"present_perfect_continuous": pc,
		"past_simple":                {Hi: [6]string{"मैंने " + perfective + "।", "तुमने " + perfective + "।", "उसने " + perfective + "।", "हमने " + perfective + "।", "आपने " + perfective + "।", "उन्होंने " + perfective + "।"}, En: [6]string{"I " + pastOf(enBase) + ".", "You " + pastOf(enBase) + ".", "He/She " + pastOf(enBase) + ".", "We " + pastOf(enBase) + ".", "You " + pastOf(enBase) + ".", "They " + pastOf(enBase) + "."}},
		"past_continuous":            pc,
		"past_perfect":               {Hi: [6]string{"मैं " + perfectAux + " था/थी।", "तुम " + perfectAux + " थे।", "वह/यह " + perfectAux + " था/थी।", "हम " + perfectAux + " थे।", "आप " + perfectAux + " थे।", "वे/ये " + perfectAux + " थे।"}, En: [6]string{"I had " + pastParticiple(enBase) + ".", "You had " + pastParticiple(enBase) + ".", "He/She had " + pastParticiple(enBase) + ".", "We had " + pastParticiple(enBase) + ".", "You had " + pastParticiple(enBase) + ".", "They had " + pastParticiple(enBase) + "."}},
		"past_perfect_continuous":    pc,
		"future_simple":              {Hi: [6]string{"मैं " + stemDeva + "ऊँगा/ऊँगी।", "तुम " + stemDeva + "ओगे।", "वह/यह " + stemDeva + "एगा/एगी।", "हम " + stemDeva + "एंगे।", "आप " + stemDeva + "एंगे।", "वे/ये " + stemDeva + "एंगे।"}, En: [6]string{"I will " + base + ".", "You will " + base + ".", "He/She will " + base + ".", "We will " + base + ".", "You will " + base + ".", "They will " + base + "."}},
		"future_continuous":          pc,
		"future_perfect":             {Hi: [6]string{"मैं " + perfectAux + " होऊँगा/हूँगी।", "तुम " + perfectAux + " होगे।", "वह/यह " + perfectAux + " होगा/होगी।", "हम " + perfectAux + " होंगे।", "आप " + perfectAux + " होंगे।", "वे/ये " + perfectAux + " होंगे।"}, En: [6]string{"I will have " + pastParticiple(enBase) + ".", "You will have " + pastParticiple(enBase) + ".", "He/She will have " + pastParticiple(enBase) + ".", "We will have " + pastParticiple(enBase) + ".", "You will have " + pastParticiple(enBase) + ".", "They will have " + pastParticiple(enBase) + "."}},
		"future_perfect_continuous":  pc,
	}
}

func pluralize(habitualMascFem string) string {
	// Replace ta/ti with te for plural where applicable
	return strings.ReplaceAll(strings.ReplaceAll(habitualMascFem, "ता/ती", "ते"), "ता/ती", "ते")
}

func pastOf(enBase string) string {
	b := strings.TrimPrefix(enBase, "to ")
	switch b {
	case "give":
		return "gave"
	case "take":
		return "took"
	case "watch", "see":
		return b + "ed"
	case "think":
		return "thought"
	case "say":
		return "said"
	case "know":
		return "knew"
	default:
		return b + "ed"
	}
}

func pastParticiple(enBase string) string {
	b := strings.TrimPrefix(enBase, "to ")
	switch b {
	case "give":
		return "given"
	case "take":
		return "taken"
	case "watch":
		return "watched"
	case "see":
		return "seen"
	case "think":
		return "thought"
	case "say":
		return "said"
	case "know":
		return "known"
	default:
		return b + "ed"
	}
}

func main() {
	baseDir := "/workspaces/quiz/backend/internal/handlers/data/verb-conjugations/hi"
	templatePath := filepath.Join(baseDir, "karna.json")

	templateVerb, err := readVerb(templatePath)
	if err != nil {
		panic(err)
	}

	targets := []string{"aana.json", "dena.json", "dekhana.json", "hona.json", "jana.json", "janana.json", "kahana.json", "lena.json", "sochana.json"}

	for _, fname := range targets {
		path := filepath.Join(baseDir, fname)
		verb, err := readVerb(path)
		if err != nil {
			panic(err)
		}

		slug := strings.TrimSuffix(fname, ".json")
		// Build full tenses from template
		fullTenses := make([]Tense, 0, len(templateVerb.Tenses))
		for _, t := range templateVerb.Tenses {
			// Special handling: preserve present_simple for hona as-is if it already exists (more accurate)
			if slug == "hona" && t.TenseID == "present_simple" {
				existing := findTense(verb.Tenses, t.TenseID)
				if existing != nil {
					fullTenses = append(fullTenses, *existing)
					continue
				}
			}

			gen := generateTenseForVerb(t, slug, verb)
			fullTenses = append(fullTenses, gen)
		}

		verb.Tenses = fullTenses
		if err := writeVerb(path, verb); err != nil {
			panic(err)
		}
		fmt.Printf("Updated %s\n", fname)
	}
}

func readVerb(path string) (Verb, error) {
	var v Verb
	b, err := os.ReadFile(path)
	if err != nil {
		return v, err
	}
	if err := json.Unmarshal(b, &v); err != nil {
		return v, err
	}
	return v, nil
}

func writeVerb(path string, v Verb) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	// ensure trailing newline like existing files
	if len(b) == 0 || b[len(b)-1] != '\n' {
		b = append(b, '\n')
	}
	return os.WriteFile(path, b, fs.FileMode(0o644))
}

func findTense(tenses []Tense, id string) *Tense {
	for i := range tenses {
		if tenses[i].TenseID == id {
			return &tenses[i]
		}
	}
	return nil
}

func findConjByPronoun(conjs []Conjugation, pronoun string) *Conjugation {
	for i := range conjs {
		if conjs[i].Pronoun == pronoun {
			return &conjs[i]
		}
	}
	return nil
}

var (
	reKarToken         = regexp.MustCompile("कर")
	reKiya             = regexp.MustCompile("किया")
	reKarRahaMasculine = regexp.MustCompile("कर रहा")
	reKarRahiFeminine  = regexp.MustCompile("कर रही")
	reKarRahePlural    = regexp.MustCompile("कर रहे")
)

func generateTenseForVerb(t Tense, slug string, verb Verb) Tense {
	irr, ok := irregulars[slug]
	if !ok {
		// derive a simple stem from slug (romanized) → map to Devanagari naive fallback
		// If unknown, default to "कर" behavior so structure at least exists
		irr = irregular{habitualStem: "कर", continuousStem: "कर", perfective: "किया"}
	}

	// Clone tense
	out := Tense{
		TenseID:     t.TenseID,
		TenseName:   t.TenseName,
		TenseNameEn: t.TenseNameEn,
		Description: t.Description,
	}

	out.Conjugations = make([]Conjugation, len(t.Conjugations))
	for i, c := range t.Conjugations {
		newC := Conjugation{Pronoun: c.Pronoun}

		// Start with template form
		form := c.Form

		switch t.TenseID {
		case "past_simple":
			form = reKiya.ReplaceAllString(form, irr.perfective)
		case "present_simple", "present_perfect", "past_perfect", "future_simple", "future_perfect":
			form = reKarToken.ReplaceAllString(form, irr.habitualStem)
		default:
			// continuous families
			form = replaceContinuousAll(form, irr.continuousStem)
		}

		// Correct future simple spellings for long 'aa' stems like आ/जा
		if t.TenseID == "future_simple" {
			form = fixFutureSimpleForAA(irr.habitualStem, form)
		}

		// Use curated examples if available
		if vmap, ok := exampleDB[slug]; ok {
			if exs, ok2 := vmap[t.TenseID]; ok2 {
				if idx, ok3 := pronounToIndex[c.Pronoun]; ok3 {
					newC.Form = form
					newC.ExampleSentence = exs.Hi[idx]
					newC.ExampleSentenceEn = exs.En[idx]
					out.Conjugations[i] = newC
					continue
				}
			}
		}

		// Otherwise, keep present_simple original examples if present
		if t.TenseID == "present_simple" {
			if existing := findTense(verb.Tenses, t.TenseID); existing != nil {
				if ex := findConjByPronoun(existing.Conjugations, c.Pronoun); ex != nil {
					newC.Form = form
					newC.ExampleSentence = ex.ExampleSentence
					newC.ExampleSentenceEn = ex.ExampleSentenceEn
					out.Conjugations[i] = newC
					continue
				}
			}
		}

		// Last resort fallback
		newC.Form = form
		newC.ExampleSentence = strings.TrimSpace(c.Pronoun + " " + form + "।")
		newC.ExampleSentenceEn = ""
		out.Conjugations[i] = newC
	}
	return out
}

func replaceContinuousAll(s, stem string) string {
	s = reKarRahaMasculine.ReplaceAllString(s, stem+" रहा")
	s = reKarRahiFeminine.ReplaceAllString(s, stem+" रही")
	s = reKarRahePlural.ReplaceAllString(s, stem+" रहे")
	// fallback generic कर → stem
	s = reKarToken.ReplaceAllString(s, stem)
	return s
}

func fixFutureSimpleForAA(stem, s string) string {
	if stem != "आ" && stem != "जा" { // focus on the common long-aa stems
		return s
	}
	// 1st person
	s = strings.ReplaceAll(s, stem+"ूँगा", stem+"ऊँगा")
	s = strings.ReplaceAll(s, stem+"ूँगी", stem+"ऊँगी")
	// 2nd person
	s = strings.ReplaceAll(s, stem+"ोगे", stem+"ओगे")
	s = strings.ReplaceAll(s, stem+"ोगी", stem+"ओगी")
	// 3rd/plural
	s = strings.ReplaceAll(s, stem+"ेगा", stem+"एगा")
	s = strings.ReplaceAll(s, stem+"ेगी", stem+"एगी")
	s = strings.ReplaceAll(s, stem+"ेंगे", stem+"एंगे")
	return s
}
