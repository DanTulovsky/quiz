package main

import (
	"fmt"
	"os"
	"github.com/neurosnap/sentences"
)

func main() {
	modelPath := "internal/resources/punkt/russian.json"
	data, err := os.ReadFile(modelPath)
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		os.Exit(1)
	}

	storage, err := sentences.LoadTraining(data)
	if err != nil {
		fmt.Printf("Error loading training: %v\n", err)
		os.Exit(1)
	}

	tokenizer := sentences.NewSentenceTokenizer(storage)
	text := `"Прежде чем мы поедем в Тверь, я должна найти документы из детского дома. Это важно. А.С. Пушкин был великим поэтом."`
	tokenized := tokenizer.Tokenize(text)

	fmt.Printf("Original text:\n%s\n\n", text)
	fmt.Printf("Sentences found: %d\n", len(tokenized))
	for i, sent := range tokenized {
		fmt.Printf("%d: %s\n", i+1, sent.Text)
	}
}
