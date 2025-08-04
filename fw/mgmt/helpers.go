/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package mgmt

import (
	"github.com/named-data/ndnd/fw/core"
	enc "github.com/named-data/ndnd/std/encoding"
	mgmt "github.com/named-data/ndnd/std/ndn/mgmt_2022"
)

// Decodes control parameters from a specific component of the given Interest's name (after the LOCAL_PREFIX) into a ControlArgs object, returning nil if parsing fails.
func decodeControlParameters(m Module, interest *Interest) *mgmt.ControlArgs {
	paramVal := interest.Name()[len(LOCAL_PREFIX)+2].Val
	params, err := mgmt.ParseControlParameters(enc.NewBufferView(paramVal), true)
	if err != nil {
		core.Log.Warn(m, "Could not decode ControlParameters", "name", interest.Name(), "err", err)
		return nil
	}
	return params.Val
}
