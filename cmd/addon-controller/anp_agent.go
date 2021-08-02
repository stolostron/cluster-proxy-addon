package main

import (
	"context"
	"fmt"
	"github.com/openshift/library-go/pkg/controller/factory"
	"github.com/openshift/library-go/pkg/operator/events"
)

// agent are reponse to get ca.crt to access
type anpAgentController struct {
	clusterName string
	recorder    events.Recorder
}

func NewANPAgentController(
	recorder events.Recorder,
) factory.Controller {
	c := &anpAgentController{}
	return factory.New().WithSync(c.sync).ToController(fmt.Sprintf("anp-agent-controller"), recorder)
}

func (a *anpAgentController) sync(ctx context.Context, syncCtx factory.SyncContext) error {
	return nil
}
