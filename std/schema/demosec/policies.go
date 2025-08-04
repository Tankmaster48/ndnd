package demosec

import (
	"fmt"
	"sync"

	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/log"
	"github.com/named-data/ndnd/std/ndn"
	"github.com/named-data/ndnd/std/schema"
	"github.com/named-data/ndnd/std/security/signer"
	"github.com/named-data/ndnd/std/utils"
)

// KeyStoragePolicy is a policy that stored HMAC keys in a memory storage.
type KeyStoragePolicy struct {
	lock     sync.RWMutex
	KeyStore *DemoHmacKeyStore
}

// Implements the Stringer interface to return the string "KeyStoragePolicy" for instances of KeyStoragePolicy.
func (p *KeyStoragePolicy) String() string {
	return "KeyStoragePolicy"
}

// Returns the KeyStoragePolicy instance as a schema.Policy interface, enabling its use in contexts requiring a generic policy trait.
func (p *KeyStoragePolicy) PolicyTrait() schema.Policy {
	return p
}

// Handles a search event by retrieving the certificate data for a key if the target name is exact (non-prefix), returning it wrapped in a Wire structure or nil if the key isn't found or the query is invalid.
func (p *KeyStoragePolicy) onSearch(event *schema.Event) any {
	p.lock.RLock()
	defer p.lock.RUnlock()

	// event.IntConfig is always valid for onSearch, no matter if there is an Interest.
	if event.IntConfig.CanBePrefix {
		log.Error(p, "the Demo HMAC key storage does not support CanBePrefix Interest to fetch certificates.")
		return nil
	}
	key := p.KeyStore.GetKey(event.Target.Name)
	if key == nil {
		return nil
	}
	return enc.Wire{key.CertData}
}

// Saves the key associated with the event's target name, along with the event's content and raw packet data, into the key store in a thread-safe manner.
func (p *KeyStoragePolicy) onSave(event *schema.Event) any {
	p.lock.Lock()
	defer p.lock.Unlock()

	// NOTE: here we consider keys are fresh forever for simplicity
	p.KeyStore.SaveKey(event.Target.Name, event.Content.Join(), event.RawPacket.Join())
	return nil
}

// Validates that the KeyStore is set to a DemoHmacKeyStore instance when the policy is attached, panicking if it is not configured.
func (p *KeyStoragePolicy) onAttach(event *schema.Event) any {
	if p.KeyStore == nil {
		panic("you must set KeyStore property to be a DemoHmacKeyStore instance in Go.")
	}
	return nil
}

// Applies a key storage policy by registering event handlers for onAttach, onSearch, and onSave operations on the given node and recursively on all its children.
func (p *KeyStoragePolicy) Apply(node *schema.Node) {
	// TODO: onAttach does not need to be called on every child...
	// But I don't have enough time to fix this
	if event := node.GetEvent(schema.PropOnAttach); event != nil {
		event.Add(utils.IdPtr(p.onAttach))
	}
	if event := node.GetEvent(schema.PropOnSearchStorage); event != nil {
		event.Add(utils.IdPtr(p.onSearch))
	}
	if event := node.GetEvent(schema.PropOnSaveStorage); event != nil {
		event.Add(utils.IdPtr(p.onSave))
	}
	chd := node.Children()
	for _, c := range chd {
		p.Apply(c)
	}
}

// Constructs a KeyStoragePolicy instance that implements the schema.Policy interface for key storage management.
func NewKeyStoragePolicy() schema.Policy {
	return &KeyStoragePolicy{}
}

// SignedByPolicy is a demo policy that specifies the trust schema.
type SignedByPolicy struct {
	Mapping     map[string]any
	KeyStore    *DemoHmacKeyStore
	KeyNodePath string

	keyNode *schema.Node
}

// Returns the string representation "SignedByPolicy" for logging or debugging purposes.
func (p *SignedByPolicy) String() string {
	return "SignedByPolicy"
}

// Implements the Policy trait by returning the receiver as a schema.Policy interface.
func (p *SignedByPolicy) PolicyTrait() schema.Policy {
	return p
}

// ConvertName converts a Data name to the name of the key to sign it.
// In real-world scenario, there should be two functions:
// - one suggests the key for the data produced by the current node
// - one checks if the signing key for a fetched data is correct
// In this simple demo I merge them into one for simplicity
func (p *SignedByPolicy) ConvertName(mNode *schema.MatchedNode) *schema.MatchedNode {
	newMatching := make(enc.Matching, len(mNode.Matching))
	for k, v := range mNode.Matching {
		if newV, ok := p.Mapping[k]; ok {
			// Be careful of crash
			newMatching[k] = []byte(newV.(string))
		} else {
			newMatching[k] = v
		}
	}
	return p.keyNode.Apply(newMatching)
}

// Returns an HMAC signer for the Data packet using a key derived from the event's target name, or nil if key construction or retrieval fails.
func (p *SignedByPolicy) onGetDataSigner(event *schema.Event) any {
	keyMNode := p.ConvertName(event.Target)
	if keyMNode == nil {
		log.Error(p, "Cannot construct the key name to sign this data. Leave unsigned.")
		return nil
	}
	key := p.KeyStore.GetKey(keyMNode.Name)
	if key == nil {
		log.Error(p, "The key to sign this data is missing. Leave unsigned.")
		return nil
	}
	return signer.NewHmacSigner(key.KeyBits)
}

// Validates a Data packet's HMAC-SHA256 signature by fetching the corresponding signing key and verifying the signature's authenticity.
func (p *SignedByPolicy) onValidateData(event *schema.Event) any {
	sigCovered := event.SigCovered
	signature := event.Signature
	if sigCovered == nil || signature == nil || signature.SigType() != ndn.SignatureHmacWithSha256 {
		return schema.VrSilence
	}
	keyMNode := p.ConvertName(event.Target)
	//TODO: Compute the deadline
	result := <-keyMNode.Call("NeedChan").(chan schema.NeedResult)
	if result.Status != ndn.InterestResultData {
		log.Warn(p, "Unable to fetch the key that signed this data.")
		return schema.VrFail
	}
	if signer.ValidateHmac(sigCovered, signature, result.Content.Join()) {
		return schema.VrPass
	} else {
		log.Warn(p, "Failed to verify the signature.")
		return schema.VrFail
	}
}

// Initializes the SignedByPolicy by validating the KeyStore and ensuring the KeyNodePath points to a valid node in the event's data structure, panicking if either check fails.
func (p *SignedByPolicy) onAttach(event *schema.Event) any {
	if p.KeyStore == nil {
		panic("you must set KeyStore property to be a DemoHmacKeyStore instance in Go.")
	}

	pathPat, err := enc.NamePatternFromStr(p.KeyNodePath)
	if err != nil {
		panic(fmt.Errorf("KeyNodePath is invalid: %+v", p.KeyNodePath))
	}
	p.keyNode = event.TargetNode.RootNode().At(pathPat)
	if p.keyNode == nil {
		panic(fmt.Errorf("specified KeyNodePath does not correspond to a valid node: %+v", p.KeyNodePath))
	}

	return nil
}

// Applies a policy to a node by registering event handlers for attachment, data signer retrieval, and data validation, ensuring the node enforces signed data validation.
func (p *SignedByPolicy) Apply(node *schema.Node) {
	if event := node.GetEvent(schema.PropOnAttach); event != nil {
		event.Add(utils.IdPtr(p.onAttach))
	}
	evt := node.GetEvent(schema.PropOnGetDataSigner)
	if evt != nil {
		evt.Add(utils.IdPtr(p.onGetDataSigner))
	}
	// PropOnValidateData must exist. Otherwise it is at an invalid path.
	evt = node.GetEvent(schema.PropOnValidateData)
	if evt != nil {
		evt.Add(utils.IdPtr(p.onValidateData))
	} else {
		panic("attaching SignedByPolicy to a node that does not need to validate Data. What is the use?")
	}
}

// Constructs a signing policy that requires data to be signed by a specific identity or key.
func NewSignedByPolicy() schema.Policy {
	return &SignedByPolicy{}
}

// Registers the KeyStoragePolicy and SignedBy policy implementations with the schema package, defining their creation functions and associated properties like KeyStore, Mapping, and KeyNodePath.
func init() {
	keyStoragePolicyDesc := &schema.PolicyImplDesc{
		ClassName: "KeyStoragePolicy",
		Create:    NewKeyStoragePolicy,
		Properties: map[schema.PropKey]schema.PropertyDesc{
			"KeyStore": schema.DefaultPropertyDesc("KeyStore"),
		},
	}
	schema.RegisterPolicyImpl(keyStoragePolicyDesc)

	signedByPolicyDesc := &schema.PolicyImplDesc{
		ClassName: "SignedBy",
		Create:    NewSignedByPolicy,
		Properties: map[schema.PropKey]schema.PropertyDesc{
			"Mapping":     schema.DefaultPropertyDesc("Mapping"),
			"KeyStore":    schema.DefaultPropertyDesc("KeyStore"),
			"KeyNodePath": schema.DefaultPropertyDesc("KeyNodePath"),
		},
	}
	schema.RegisterPolicyImpl(signedByPolicyDesc)
}
