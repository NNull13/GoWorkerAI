package tools

import (
	"errors"
	"fmt"
	"log"
	"strings"
)

var ErrorRejected = errors.New("REJECTED")

type ReviewerAction struct {
	Answer string `json:"answer"`
	Reason string `json:"reason"`
}

func executeReviewerAction(action ToolTask) (string, error) {
	h, ok := reviewerDispatch[action.Key]
	if !ok {
		log.Printf("❌ Unknown tool key: %s\n", action.Key)
		return "", fmt.Errorf("unknown tool key: %s", action.Key)
	}
	return h(action.Parameters)
}

var reviewerDispatch = map[string]func(any) (string, error){
	true_or_false: func(p any) (string, error) {
		return withParsed[ReviewerAction](p, true_or_false, func(a ReviewerAction) (string, error) {
			return YesOrNo(a)
		})
	},
}

func YesOrNo(p ReviewerAction) (string, error) {
	decision := strings.ToLower(p.Answer)
	switch decision {
	case "true":
		return "✅ " + p.Reason, nil
	case "false":
		return "❌ " + p.Reason, ErrorRejected
	default:
		return "", fmt.Errorf("unknown decision: %s", decision)
	}
}
