package policy

import (
	"context"
	"fmt"
	"github.com/open-policy-agent/opa/rego"
	log "github.com/sirupsen/logrus"
	"net/http"
)

type ImageRequest struct {
	Path           string
	Method         string
	Header         http.Header
	NumberOfImages int
	SizeOfImages   int64
}

type ImageRequestsEngine interface {
	IsAllowed(ctx context.Context, request ImageRequest) (bool, error)
}

func NewImageRequestsEngine(regoPath string) ImageRequestsEngine {
	r := rego.New(
		rego.Query("data.load.image_requests"),
		rego.Load([]string{regoPath}, nil),
	)
	return &imageRequestsEngine{
		r,
	}
}

type imageRequestsEngine struct {
	*rego.Rego
}

func (e imageRequestsEngine) IsAllowed(ctx context.Context, request ImageRequest) (bool, error) {
	query, err := e.PrepareForEval(ctx)
	if err != nil {
		return false, err
	}
	results, err := query.Eval(ctx, rego.EvalInput(request))
	if err != nil {
		log.Errorf("error evaluating policy: %s", err)
		return false, err
	}
	if len(results) > 1 {
		log.Errorf("more than 1 results")
		return false, fmt.Errorf("error evaluating policy: %s", err)
	}
	valuesI := results[0].Expressions[0].Value
	values := valuesI.(map[string]interface{})
	allow := values["allow"].(bool)
	if !allow {
		message := values["message"].([]interface{})[0].(string)
		log.Infof("Decision: %t, message: %s", allow, message)
	}

	return allow, nil
}
