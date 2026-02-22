package routes

import "github.com/fr0stylo/ddash/views/components"

func nextServiceStatus(status components.Status) components.Status {
	switch status {
	case components.StatusSynced:
		return components.StatusOutOfSync
	case components.StatusOutOfSync:
		return components.StatusProgressing
	case components.StatusProgressing:
		return components.StatusSynced
	case components.StatusWarning, components.StatusUnknown, components.StatusAll:
		return components.StatusSynced
	default:
		return components.StatusSynced
	}
}
