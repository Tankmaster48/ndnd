/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package mgmt

import (
	"github.com/named-data/ndnd/fw/core"
	"github.com/named-data/ndnd/fw/defn"
	"github.com/named-data/ndnd/fw/fw"
	"github.com/named-data/ndnd/fw/table"
	enc "github.com/named-data/ndnd/std/encoding"
	mgmt "github.com/named-data/ndnd/std/ndn/mgmt_2022"
)

// StrategyChoiceModule is the module that handles Strategy Choice Management.
type StrategyChoiceModule struct {
	manager *Thread
}

// Returns the string "mgmt-strategy" as the identifier for the management strategy module.
func (s *StrategyChoiceModule) String() string {
	return "mgmt-strategy"
}

// Registers the provided Thread instance as the manager for this StrategyChoiceModule, establishing a reference for future interactions.
func (s *StrategyChoiceModule) registerManager(manager *Thread) {
	s.manager = manager
}

// Returns the Thread manager associated with this StrategyChoiceModule instance.
func (s *StrategyChoiceModule) getManager() *Thread {
	return s.manager
}

// Handles incoming strategy management Interests by validating they originate from the local host, then dispatching "set", "unset", or "list" commands to configure or query strategy choices, or returning an error for unknown commands.
func (s *StrategyChoiceModule) handleIncomingInterest(interest *Interest) {
	// Only allow from /localhost
	if !LOCAL_PREFIX.IsPrefix(interest.Name()) {
		core.Log.Warn(s, "Received strategy management Interest from non-local source - DROP")
		return
	}

	// Dispatch by verb
	verb := interest.Name()[len(LOCAL_PREFIX)+1].String()
	switch verb {
	case "set":
		s.set(interest)
	case "unset":
		s.unset(interest)
	case "list":
		s.list(interest)
	default:
		s.manager.sendCtrlResp(interest, 501, "Unknown verb", nil)
		return
	}
}

// Sets the forwarding strategy for a given name in the FIB based on control parameters in an Interest, validating strategy name, version, and availability before applying the configuration.
func (s *StrategyChoiceModule) set(interest *Interest) {
	if len(interest.Name()) < len(LOCAL_PREFIX)+3 {
		s.manager.sendCtrlResp(interest, 400, "ControlParameters is incorrect", nil)
		return
	}

	params := decodeControlParameters(s, interest)
	if params == nil {
		s.manager.sendCtrlResp(interest, 400, "ControlParameters is incorrect", nil)
		return
	}

	if params.Name == nil {
		s.manager.sendCtrlResp(interest, 400, "ControlParameters is incorrect (missing Name)", nil)
		return
	}

	if params.Strategy == nil {
		s.manager.sendCtrlResp(interest, 400, "ControlParameters is incorrect (missing Strategy)", nil)
		return
	}

	if !defn.STRATEGY_PREFIX.IsPrefix(params.Strategy.Name) {
		core.Log.Warn(s, "Invalid strategy", "strategy", params.Strategy.Name)
		s.manager.sendCtrlResp(interest, 404, "Invalid strategy", nil)
		return
	}

	strategyName := params.Strategy.Name[len(defn.STRATEGY_PREFIX)].String()
	availableVersions, ok := fw.StrategyVersions[strategyName]
	if !ok {
		core.Log.Warn(s, "Unknown strategy", "strategy", params.Strategy.Name)
		s.manager.sendCtrlResp(interest, 404, "Unknown strategy", nil)
		return
	}

	// Add/verify version information for strategy
	strategyVersion := availableVersions[0]
	for _, version := range availableVersions {
		if version > strategyVersion {
			strategyVersion = version
		}
	}
	if len(params.Strategy.Name) > len(defn.STRATEGY_PREFIX)+1 && !params.Strategy.Name[len(defn.STRATEGY_PREFIX)+1].IsVersion() {
		core.Log.Warn(s, "Unknown strategy version", "strategy", params.Strategy.Name, "version", params.Strategy.Name[len(defn.STRATEGY_PREFIX)+1])
		s.manager.sendCtrlResp(interest, 404, "Unknown strategy version", nil)
		return
	} else if len(params.Strategy.Name) > len(defn.STRATEGY_PREFIX)+1 {
		strategyVersionBytes := params.Strategy.Name[len(defn.STRATEGY_PREFIX)+1].Val
		strategyVersion, _, err := enc.ParseNat(strategyVersionBytes)
		if err != nil {
			core.Log.Warn(s, "Invalid strategy version", "strategy", params.Strategy.Name, "version", params.Strategy.Name[len(defn.STRATEGY_PREFIX)+1])
			s.manager.sendCtrlResp(interest, 404, "Invalid strategy version", nil)
			return
		}
		foundMatchingVersion := false
		for _, version := range availableVersions {
			if version == uint64(strategyVersion) {
				foundMatchingVersion = true
			}
		}
		if !foundMatchingVersion {
			core.Log.Warn(s, "Unknown strategy version", "strategy", params.Strategy.Name, "version", strategyVersion)
			s.manager.sendCtrlResp(interest, 404, "Unknown strategy version", nil)
			return
		}
	} else {
		// Add missing version information to strategy name
		params.Strategy.Name = params.Strategy.Name.
			Append(enc.NewVersionComponent(strategyVersion))
	}
	table.FibStrategyTable.SetStrategyEnc(params.Name, params.Strategy.Name)

	s.manager.sendCtrlResp(interest, 200, "OK", &mgmt.ControlArgs{
		Name:     params.Name,
		Strategy: params.Strategy,
	})

	core.Log.Info(s, "Set strategy", "name", params.Name, "strategy", params.Strategy.Name)
}

// Handles an Interest request to remove a strategy configuration for a specified name by validating the input parameters, decoding the control parameters, and unsetting the corresponding strategy entry in the FIB strategy table.
func (s *StrategyChoiceModule) unset(interest *Interest) {
	if len(interest.Name()) < len(LOCAL_PREFIX)+3 {
		s.manager.sendCtrlResp(interest, 400, "ControlParameters is incorrect", nil)
		return
	}

	params := decodeControlParameters(s, interest)
	if params == nil {
		s.manager.sendCtrlResp(interest, 400, "ControlParameters is incorrect", nil)
		return
	}

	if params.Name == nil {
		s.manager.sendCtrlResp(interest, 400, "ControlParameters is incorrect (missing Name)", nil)
		return
	}

	if len(params.Name) == 0 {
		s.manager.sendCtrlResp(interest, 400, "ControlParameters is incorrect (empty Name)", nil)
		return
	}

	table.FibStrategyTable.UnSetStrategyEnc(params.Name)
	core.Log.Info(s, "Unset Strategy", "name", params.Name)

	s.manager.sendCtrlResp(interest, 200, "OK", &mgmt.ControlArgs{Name: params.Name})
}

// Constructs and sends a response Data packet listing all configured forwarding strategies by encoding them into a StrategyChoiceMsg, triggered by an Interest with a name matching the strategy-choice list endpoint.
func (s *StrategyChoiceModule) list(interest *Interest) {
	if len(interest.Name()) > len(LOCAL_PREFIX)+2 {
		// Ignore because contains version and/or segment components
		return
	}

	// Generate new dataset
	// TODO: For thread safety, we should lock the Strategy table from writes until we are done
	entries := table.FibStrategyTable.GetAllForwardingStrategies()
	choices := []*mgmt.StrategyChoice{}
	for _, fsEntry := range entries {
		choices = append(choices, &mgmt.StrategyChoice{
			Name:     fsEntry.Name(),
			Strategy: &mgmt.Strategy{Name: fsEntry.GetStrategy()},
		})
	}
	dataset := &mgmt.StrategyChoiceMsg{StrategyChoices: choices}

	name := LOCAL_PREFIX.Append(
		enc.NewGenericComponent("strategy-choice"),
		enc.NewGenericComponent("list"),
	)
	s.manager.sendStatusDataset(interest, name, dataset.Encode())
}
