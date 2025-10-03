package main

import (
	"encoding/json"
	"fmt"

	"quizapp/internal/models"
)

func main() {
	// Test case 1: null section_length_override (should work)
	req1 := models.CreateStoryRequest{
		Title:                 "Test Story",
		Subject:               nil,
		SectionLengthOverride: nil,
	}

	jsonData1, _ := json.Marshal(req1)
	fmt.Printf("Request 1 (null section_length_override): %s\n", string(jsonData1))

	// Simulate what happens when frontend sends null
	jsonStr1 := `{"title":"Test Story","subject":null,"section_length_override":null}`
	var req2 models.CreateStoryRequest
	json.Unmarshal([]byte(jsonStr1), &req2)

	// Check if the pointer is nil or points to empty string
	if req2.SectionLengthOverride == nil {
		fmt.Printf("Request 2 (unmarshaled from null): section_length_override = nil\n")
	} else {
		fmt.Printf("Request 2 (unmarshaled from null): section_length_override = %v, *value = %q\n", req2.SectionLengthOverride, *req2.SectionLengthOverride)
	}

	// Test validation
	err := req2.Validate()
	fmt.Printf("Validation error for request 2: %v\n", err)
}
