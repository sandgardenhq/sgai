package main

import (
	"log"

	"github.com/sandgardenhq/sgai/pkg/state"
)

func saveState(coord *state.Coordinator, s state.Workflow) {
	if errUpdate := coord.UpdateState(func(wf *state.Workflow) {
		*wf = s
	}); errUpdate != nil {
		log.Fatalln("failed to save state:", errUpdate)
	}
}

func countPendingTodos(wfState state.Workflow, agent string) int {
	if agent == "coordinator" {
		return 0
	}
	count := 0
	for _, todo := range wfState.Todos {
		if todo.Status != "completed" && todo.Status != "cancelled" {
			count++
		}
	}
	return count
}
