package main

import (
	"context"
	"fmt"
	"time"

	"github.com/Websoft9/waterflow/pkg/dsl/node"
)

type GreeterNode struct{}

func (n *GreeterNode) Name() string {
	return "custom/greeter"
}

func (n *GreeterNode) Version() string {
	return "v1"
}

func (n *GreeterNode) Params() map[string]node.ParamSpec {
	return map[string]node.ParamSpec{
		"name": {
			Type:        "string",
			Required:    true,
			Description: "Person's name to greet (supports Unicode)",
		},
		"language": {
			Type:        "string",
			Required:    false,
			Description: "Greeting language",
			Enum:        []interface{}{"en", "zh", "es", "fr"},
			Default:     "en",
		},
		"time_of_day": {
			Type:        "string",
			Required:    false,
			Description: "Time period (morning, afternoon, evening)",
			Enum:        []interface{}{"morning", "afternoon", "evening", "auto"},
			Default:     "auto",
		},
	}
}

func (n *GreeterNode) Execute(ctx context.Context, inputs map[string]interface{}) (*node.NodeResult, error) {
	startTime := time.Now()

	if err := node.ValidateInputs(inputs, n.Params()); err != nil {
		return nil, err
	}

	name := inputs["name"].(string)
	language := "en"
	if lang, ok := inputs["language"].(string); ok {
		language = lang
	}

	timeOfDay := "auto"
	if tod, ok := inputs["time_of_day"].(string); ok {
		timeOfDay = tod
	}

	if timeOfDay == "auto" {
		hour := time.Now().Hour()
		switch {
		case hour < 12:
			timeOfDay = "morning"
		case hour < 18:
			timeOfDay = "afternoon"
		default:
			timeOfDay = "evening"
		}
	}

	greeting := generateGreeting(name, language, timeOfDay)

	result := node.NewNodeResult()
	result.SetOutput("greeting", greeting)
	result.SetOutput("language", language)
	result.SetOutput("time_of_day", timeOfDay)
	result.AddLog(fmt.Sprintf("Generated %s greeting for %s", language, name))
	result.Duration = time.Since(startTime)

	return result, nil
}

func (n *GreeterNode) Metadata() node.NodeMetadata {
	return node.NodeMetadata{
		Description: "Generates multi-language greetings based on time of day",
		Category:    "custom",
		InputSchema: n.Params(),
		OutputSchema: map[string]interface{}{
			"greeting":    "string - The generated greeting message",
			"language":    "string - The language used",
			"time_of_day": "string - The detected or specified time period",
		},
	}
}

func Register() node.Node {
	return &GreeterNode{}
}

func generateGreeting(name, language, timeOfDay string) string {
	greetings := map[string]map[string]string{
		"en": {
			"morning":   "Good morning, %s!",
			"afternoon": "Good afternoon, %s!",
			"evening":   "Good evening, %s!",
		},
		"zh": {
			"morning":   "早上好,%s!",
			"afternoon": "下午好,%s!",
			"evening":   "晚上好,%s!",
		},
		"es": {
			"morning":   "Buenos días, %s!",
			"afternoon": "Buenas tardes, %s!",
			"evening":   "Buenas noches, %s!",
		},
		"fr": {
			"morning":   "Bonjour, %s!",
			"afternoon": "Bon après-midi, %s!",
			"evening":   "Bonsoir, %s!",
		},
	}

	template := greetings[language][timeOfDay]
	return fmt.Sprintf(template, name)
}
