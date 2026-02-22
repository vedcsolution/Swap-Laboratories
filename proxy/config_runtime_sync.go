package proxy

import (
	"reflect"
	"slices"

	"github.com/mostlygeek/llama-swap/proxy/config"
)

// applyConfigAndSyncProcessGroups atomically swaps in a new config and keeps
// runtime process groups in sync so newly managed models are immediately
// loadable/unloadable from the UI.
func (pm *ProxyManager) applyConfigAndSyncProcessGroups(newConfig config.Config) {
	pm.Lock()
	oldConfig := pm.config
	oldGroups := pm.processGroups

	nextGroups := make(map[string]*ProcessGroup, len(newConfig.Groups))
	groupsToShutdown := make([]*ProcessGroup, 0)

	for groupID := range newConfig.Groups {
		if oldGroup, ok := oldGroups[groupID]; ok && runtimeGroupCompatible(oldConfig, newConfig, groupID) {
			oldGroup.Lock()
			oldGroup.config = newConfig
			oldGroup.swap = newConfig.Groups[groupID].Swap
			oldGroup.exclusive = newConfig.Groups[groupID].Exclusive
			oldGroup.persistent = newConfig.Groups[groupID].Persistent
			oldGroup.Unlock()
			nextGroups[groupID] = oldGroup
			continue
		}

		if oldGroup, ok := oldGroups[groupID]; ok {
			groupsToShutdown = append(groupsToShutdown, oldGroup)
		}
		nextGroups[groupID] = NewProcessGroup(groupID, newConfig, pm.proxyLogger, pm.upstreamLogger)
	}

	for groupID, oldGroup := range oldGroups {
		if _, ok := nextGroups[groupID]; !ok {
			groupsToShutdown = append(groupsToShutdown, oldGroup)
		}
	}

	pm.config = newConfig
	pm.processGroups = nextGroups
	pm.Unlock()

	for _, group := range groupsToShutdown {
		group.Shutdown()
	}
}

func runtimeGroupCompatible(oldConfig, newConfig config.Config, groupID string) bool {
	oldGroup, ok := oldConfig.Groups[groupID]
	if !ok {
		return false
	}
	newGroup, ok := newConfig.Groups[groupID]
	if !ok {
		return false
	}

	if oldGroup.Swap != newGroup.Swap ||
		oldGroup.Exclusive != newGroup.Exclusive ||
		oldGroup.Persistent != newGroup.Persistent {
		return false
	}

	if !slices.Equal(oldGroup.Members, newGroup.Members) {
		return false
	}

	for _, member := range newGroup.Members {
		oldModelCfg, oldResolvedName, okOld := oldConfig.FindConfig(member)
		newModelCfg, newResolvedName, okNew := newConfig.FindConfig(member)
		if okOld != okNew || oldResolvedName != newResolvedName {
			return false
		}
		if !reflect.DeepEqual(oldModelCfg, newModelCfg) {
			return false
		}
	}

	return true
}
