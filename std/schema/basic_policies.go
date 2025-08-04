package schema

import (
	"bytes"
	"fmt"
	"sync"
	"time"

	enc "github.com/named-data/ndnd/std/encoding"
	basic_engine "github.com/named-data/ndnd/std/engine/basic"
	"github.com/named-data/ndnd/std/ndn"
	"github.com/named-data/ndnd/std/security/signer"
	"github.com/named-data/ndnd/std/utils"
)

type RegisterPolicy struct {
	RegisterIf bool
	// It is map[string]any in json
	// but the any can be a string
	Patterns enc.Matching
}

// Returns the policy instance itself as a Policy interface, enabling RegisterPolicy to satisfy the Policy interface requirement.
func (p *RegisterPolicy) PolicyTrait() Policy {
	return p
}

// Registers a route for the node's name prefix when the policy is attached, ensuring the prefix is initialized and registered with the network.
func (p *RegisterPolicy) onAttach(event *Event) any {
	node := event.TargetNode
	mNode := node.Apply(p.Patterns)
	if mNode == nil {
		panic("cannot initialize the name prefix to register")
	}
	err := node.Engine().RegisterRoute(mNode.Name)
	if err != nil {
		panic(fmt.Errorf("prefix registration failed: %+v", err))
	}
	return nil
}

// Applies the registration policy by attaching the `onAttach` callback to the node's `PropOnAttach` event if `RegisterIf` is true.
func (p *RegisterPolicy) Apply(node *Node) {
	if p.RegisterIf {
		var callback Callback = p.onAttach
		node.AddEventListener(PropOnAttach, &callback)
	}
}

// Constructs a RegisterPolicy with the RegisterIf flag set to true, enabling default registration behavior.
func NewRegisterPolicy() Policy {
	return &RegisterPolicy{
		RegisterIf: true,
	}
}

type Sha256SignerPolicy struct{}

// Returns the Policy interface that this Sha256SignerPolicy implements, enabling type-safe access to base policy functionality.
func (p *Sha256SignerPolicy) PolicyTrait() Policy {
	return p
}

// Constructs a SHA-256 signer policy that enforces the use of SHA-256 for signing data packets.
func NewSha256SignerPolicy() Policy {
	return &Sha256SignerPolicy{}
}

// Returns a SHA-256 signer used to sign Data packets with the SHA-256 algorithm.
func (p *Sha256SignerPolicy) onGetDataSigner(*Event) any {
	return signer.NewSha256Signer()
}

// Validates a Data packet's SHA-256 signature by comparing the provided signature value with a computed signature using the SHA-256 algorithm, returning validation pass/fail/silence based on the result.
func (p *Sha256SignerPolicy) onValidateData(event *Event) any {
	sigCovered := event.SigCovered
	signature := event.Signature
	if sigCovered == nil || signature == nil || signature.SigType() != ndn.SignatureDigestSha256 {
		return VrSilence
	}
	val, _ := signer.NewSha256Signer().Sign(sigCovered)
	if bytes.Equal(signature.SigValue(), val) {
		return VrPass
	} else {
		return VrFail
	}
}

// Applies a SHA-256-based signing and validation policy to a node by registering event handlers for data signer retrieval and validation, ensuring the node supports data validation or triggering a panic otherwise.
func (p *Sha256SignerPolicy) Apply(node *Node) {
	// IdPtr must be used
	evt := node.GetEvent(PropOnGetDataSigner)
	if evt != nil {
		evt.Add(utils.IdPtr(p.onGetDataSigner))
	}
	// PropOnValidateData must exist. Otherwise it is at an invalid path.
	evt = node.GetEvent(PropOnValidateData)
	if evt != nil {
		evt.Add(utils.IdPtr(p.onValidateData))
	} else {
		panic("attaching Sha256SignerPolicy to a node that does not need to validate Data. What is the use?")
	}
}

type CacheEntry struct {
	RawData  enc.Wire
	Validity time.Time
}

// MemStoragePolicy is a policy that stored data in a memory storage.
// It will iteratively applies to all children in a subtree.
type MemStoragePolicy struct {
	timer ndn.Timer
	lock  sync.RWMutex
	// TODO: A better implementation would be MemStoragePolicy refers to an external storage
	// but not implement one itself.
	tree *basic_engine.NameTrie[CacheEntry]
}

// Returns the receiver as a Policy interface, satisfying the Policy interface's requirement for a method that returns itself.
func (p *MemStoragePolicy) PolicyTrait() Policy {
	return p
}

// Retrieves the most appropriate Data packet from in-memory storage by exact name match, returning fresh data if available and satisfying validity constraints.
func (p *MemStoragePolicy) Get(name enc.Name, canBePrefix bool, mustBeFresh bool) enc.Wire {
	p.lock.RLock()
	defer p.lock.RUnlock()

	node := p.tree.ExactMatch(name)
	now := time.Time{}
	if p.timer != nil {
		now = p.timer.Now()
	}
	if node == nil {
		return nil
	}
	freshTest := func(entry CacheEntry) bool {
		return len(entry.RawData) > 0 && (!mustBeFresh || entry.Validity.After(now))
	}
	if freshTest(node.Value()) {
		return node.Value().RawData
	}
	dataNode := node.FirstNodeIf(freshTest)
	if dataNode != nil {
		return dataNode.Value().RawData
	} else {
		return nil
	}
}

// Stores the provided data in the in-memory cache with the specified name and validity period for data retrieval.
func (p *MemStoragePolicy) Put(name enc.Name, rawData enc.Wire, validity time.Time) {
	p.lock.Lock()
	defer p.lock.Unlock()

	node := p.tree.MatchAlways(name)
	node.SetValue(CacheEntry{
		RawData:  rawData,
		Validity: validity,
	})
}

// Initializes the MemStoragePolicy's timer by obtaining it from the node's engine when the policy is attached to a node.
func (p *MemStoragePolicy) onAttach(event *Event) any {
	p.timer = event.TargetNode.Engine().Timer()
	return nil
}

// Handles a search event by retrieving data from memory storage using the target name and search configuration parameters (allowing prefix matching and requiring freshness as specified).
func (p *MemStoragePolicy) onSearch(event *Event) any {
	// event.IntConfig is always valid for onSearch, no matter if there is an Interest.
	return p.Get(event.Target.Name, event.IntConfig.CanBePrefix, event.IntConfig.MustBeFresh)
}

// Saves the event's raw packet into memory storage with a validity period calculated by adding the event's duration to the current time, ensuring automatic expiration after the specified period.
func (p *MemStoragePolicy) onSave(event *Event) any {
	validity := p.timer.Now().Add(*event.ValidDuration)
	p.Put(event.Target.Name, event.RawPacket, validity)
	return nil
}

// Applies the memory storage policy to a node and its children by registering event handlers for attachment, storage searches, and saves, recursively propagating the policy through the node hierarchy.
func (p *MemStoragePolicy) Apply(node *Node) {
	// TODO: onAttach does not need to be called on every child...
	// But I don't have enough time to fix this
	if event := node.GetEvent(PropOnAttach); event != nil {
		event.Add(utils.IdPtr(p.onAttach))
	}
	if event := node.GetEvent(PropOnSearchStorage); event != nil {
		event.Add(utils.IdPtr(p.onSearch))
	}
	if event := node.GetEvent(PropOnSaveStorage); event != nil {
		event.Add(utils.IdPtr(p.onSave))
	}
	chd := node.Children()
	for _, c := range chd {
		p.Apply(c)
	}
}

// Constructs a new in-memory storage policy using a name trie to manage cached entries.
func NewMemStoragePolicy() Policy {
	return &MemStoragePolicy{
		tree: basic_engine.NewNameTrie[CacheEntry](),
	}
}

type FixedHmacSignerPolicy struct {
	Key         string
	KeyName     enc.Name
	SignForCert bool
	ExpireTime  time.Duration
}

// Returns the FixedHmacSignerPolicy instance as a Policy interface, enabling type assertion for interface implementation.
func (p *FixedHmacSignerPolicy) PolicyTrait() Policy {
	return p
}

// Constructs a fixed HMAC signer policy with certificate signing disabled.
func NewFixedHmacSignerPolicy() Policy {
	return &FixedHmacSignerPolicy{
		SignForCert: false,
	}
}

// Returns an HMAC signer initialized with the fixed key stored in the policy, used to sign Data packets upon request.
func (p *FixedHmacSignerPolicy) onGetDataSigner(*Event) any {
	return signer.NewHmacSigner([]byte(p.Key))
}

// Validates an HMAC-SHA256 signature on a Data packet using a fixed pre-shared key, returning validation pass/fail results or silence for incompatible signature types.
func (p *FixedHmacSignerPolicy) onValidateData(event *Event) any {
	sigCovered := event.SigCovered
	signature := event.Signature
	if sigCovered == nil || signature == nil || signature.SigType() != ndn.SignatureHmacWithSha256 {
		return VrSilence
	}
	if signer.ValidateHmac(sigCovered, signature, []byte(p.Key)) {
		return VrPass
	} else {
		return VrFail
	}
}

// Applies an HMAC-based signing and validation policy to a node, enforcing the use of a fixed key for data signing and ensuring data validation is enabled.
func (p *FixedHmacSignerPolicy) Apply(node *Node) {
	// key must present
	if len(p.Key) == 0 {
		panic("FixedHmacSignerPolicy requires key to present before apply.")
	}
	// IdPtr must be used
	evt := node.GetEvent(PropOnGetDataSigner)
	if evt != nil {
		evt.Add(utils.IdPtr(p.onGetDataSigner))
	}
	// PropOnValidateData must exist. Otherwise it is at an invalid path.
	evt = node.GetEvent(PropOnValidateData)
	if evt != nil {
		evt.Add(utils.IdPtr(p.onValidateData))
	} else {
		panic("applying FixedHmacSignerPolicy to a node that does not need to validate Data. What is the use?")
	}
}

type FixedHmacIntSignerPolicy struct {
	Key    string
	signer ndn.Signer
}

// Returns the Policy implementation of the FixedHmacIntSignerPolicy, allowing the object to be accessed as a Policy trait.
func (p *FixedHmacIntSignerPolicy) PolicyTrait() Policy {
	return p
}

// Constructs a signing policy that uses a fixed HMAC internal key for signing operations.
func NewFixedHmacIntSignerPolicy() Policy {
	return &FixedHmacIntSignerPolicy{}
}

// Returns the HMAC signer associated with this FixedHmacIntSignerPolicy when the "GetIntSigner" event is triggered.
func (p *FixedHmacIntSignerPolicy) onGetIntSigner(*Event) any {
	return p.signer
}

// Validates an HMAC-SHA256 signature using a fixed pre-shared key, returning validation pass/fail results or silence if the signature type is invalid or missing.
func (p *FixedHmacIntSignerPolicy) onValidateInt(event *Event) any {
	sigCovered := event.SigCovered
	signature := event.Signature
	if sigCovered == nil || signature == nil || signature.SigType() != ndn.SignatureHmacWithSha256 {
		return VrSilence
	}
	if signer.ValidateHmac(sigCovered, signature, []byte(p.Key)) {
		return VrPass
	} else {
		return VrFail
	}
}

// Initializes the HMAC signer using the stored key when the policy is attached to an event.
func (p *FixedHmacIntSignerPolicy) onAttach(event *Event) any {
	p.signer = signer.NewHmacSigner([]byte(p.Key))
	return nil
}

// Applies a fixed HMAC-based signing policy to a node, ensuring it uses a provided key for signing Interests and validating incoming Interests by attaching event handlers for signing, validation, and node attachment.
func (p *FixedHmacIntSignerPolicy) Apply(node *Node) {
	// key must present
	if len(p.Key) == 0 {
		panic("FixedHmacSignerPolicy requires key to present before apply.")
	}
	// IdPtr must be used
	evt := node.GetEvent(PropOnGetIntSigner)
	if evt != nil {
		evt.Add(utils.IdPtr(p.onGetIntSigner))
	}
	// PropOnValidateInt must exist. Otherwise it is at an invalid path.
	evt = node.GetEvent(PropOnValidateInt)
	if evt != nil {
		evt.Add(utils.IdPtr(p.onValidateInt))
	} else {
		panic("applying FixedHmacSignerPolicy to a node that does not need to validate Interest. What is the use?")
	}

	node.AddEventListener(PropOnAttach, utils.IdPtr(p.onAttach))
}

// Registers predefined policy implementations (e.g., registration rules, signing methods, storage policies) with their respective configuration properties and constructor functions for dynamic policy creation in the NDN system.
func initPolicies() {
	registerPolicyDesc := &PolicyImplDesc{
		ClassName: "RegisterPolicy",
		Properties: map[PropKey]PropertyDesc{
			"RegisterIf": DefaultPropertyDesc("RegisterIf"),
			"Patterns":   MatchingPropertyDesc("Patterns"),
		},
		Create: NewRegisterPolicy,
	}
	sha256SignerPolicyDesc := &PolicyImplDesc{
		ClassName: "Sha256Signer",
		Create:    NewSha256SignerPolicy,
	}
	RegisterPolicyImpl(registerPolicyDesc)
	RegisterPolicyImpl(sha256SignerPolicyDesc)
	memoryStoragePolicyDesc := &PolicyImplDesc{
		ClassName: "MemStorage",
		Create:    NewMemStoragePolicy,
	}
	RegisterPolicyImpl(memoryStoragePolicyDesc)

	fixedHmacSignerPolicyDesc := &PolicyImplDesc{
		ClassName: "FixedHmacSigner",
		Create:    NewFixedHmacSignerPolicy,
		Properties: map[PropKey]PropertyDesc{
			"KeyValue":    DefaultPropertyDesc("Key"),
			"KeyName":     NamePropertyDesc("KeyName"),
			"SignForCert": DefaultPropertyDesc("SignForCert"),
			"ExpireTime":  TimePropertyDesc("ExpireTime"),
		},
	}
	RegisterPolicyImpl(fixedHmacSignerPolicyDesc)

	fixedHmacIntSignerPolicyDesc := &PolicyImplDesc{
		ClassName: "FixedHmacIntSigner",
		Create:    NewFixedHmacIntSignerPolicy,
		Properties: map[PropKey]PropertyDesc{
			"KeyValue": DefaultPropertyDesc("Key"),
		},
	}
	RegisterPolicyImpl(fixedHmacIntSignerPolicyDesc)
}
