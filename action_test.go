package holochain

import (
	// "fmt"
	"fmt"
	. "github.com/holochain/holochain-proto/hash"
	b58 "github.com/jbenet/go-base58"
	peer "github.com/libp2p/go-libp2p-peer"
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

func TestValidateAction(t *testing.T) {
	d, _, h := PrepareTestChain("test")
	defer CleanupTestChain(h, d)
	var err error

	// these test the generic properties of ValidateAction using a commit action as an example
	Convey("it should fail if a validator doesn't exist for the entry type", t, func() {
		entry := &GobEntry{C: "foo"}
		a := NewCommitAction("bogusType", entry)
		_, err = h.ValidateAction(a, a.entryType, nil, []peer.ID{h.nodeID})
		So(err.Error(), ShouldEqual, "no definition for entry type: bogusType")
	})

	Convey("a valid entry returns the entry def", t, func() {
		entry := &GobEntry{C: "2"}
		a := NewCommitAction("evenNumbers", entry)
		var d *EntryDef
		d, err = h.ValidateAction(a, a.entryType, nil, []peer.ID{h.nodeID})
		So(err, ShouldBeNil)
		So(fmt.Sprintf("%v", d), ShouldEqual, "&{evenNumbers zygo public  <nil>}")
	})
	Convey("an invalid action returns the ValidationFailedErr", t, func() {
		entry := &GobEntry{C: "1"}
		a := NewCommitAction("evenNumbers", entry)
		_, err = h.ValidateAction(a, a.entryType, nil, []peer.ID{h.nodeID})
		So(IsValidationFailedErr(err), ShouldBeTrue)
	})

	// these test the sys type cases
	Convey("adding or changing dna should fail", t, func() {
		entry := &GobEntry{C: "fakeDNA"}
		a := NewCommitAction(DNAEntryType, entry)
		_, err = h.ValidateAction(a, a.entryType, nil, []peer.ID{h.nodeID})
		So(err, ShouldEqual, ErrNotValidForDNAType)
		ap := NewPutAction(DNAEntryType, entry, nil)
		_, err = h.ValidateAction(ap, ap.entryType, nil, []peer.ID{h.nodeID})
		So(err, ShouldEqual, ErrNotValidForDNAType)
		am := NewModAction(DNAEntryType, entry, HashFromPeerID(h.nodeID))
		_, err = h.ValidateAction(am, am.entryType, nil, []peer.ID{h.nodeID})
		So(err, ShouldEqual, ErrNotValidForDNAType)
	})

	Convey("modifying a headers entry should fail", t, func() {
		hd := h.Chain().Top()
		j, _ := hd.ToJSON()
		entryStr := fmt.Sprintf(`[{"Header":%s,"Role":"someRole","Source":"%s"}]`, j, h.nodeID.Pretty())
		am := NewModAction(HeadersEntryType, &GobEntry{C: entryStr}, HashFromPeerID(h.nodeID))
		_, err = h.ValidateAction(am, am.entryType, nil, []peer.ID{h.nodeID})
		So(err, ShouldEqual, ErrNotValidForHeadersType)
	})

	Convey("deleting should fail for all sys entry types except delete", t, func() {
		a := NewDelAction(DelEntry{})
		_, err = h.ValidateAction(a, DNAEntryType, nil, []peer.ID{h.nodeID})
		So(err, ShouldEqual, ErrEntryDefInvalid)

		_, err = h.ValidateAction(a, KeyEntryType, nil, []peer.ID{h.nodeID})
		So(err, ShouldEqual, ErrEntryDefInvalid)

		_, err = h.ValidateAction(a, AgentEntryType, nil, []peer.ID{h.nodeID})
		So(err, ShouldEqual, ErrEntryDefInvalid)

		_, err = h.ValidateAction(a, HeadersEntryType, nil, []peer.ID{h.nodeID})
		So(err, ShouldEqual, ErrEntryDefInvalid)
	})
}

func TestSysValidateEntry(t *testing.T) {
	d, _, h := PrepareTestChain("test")
	defer CleanupTestChain(h, d)

	Convey("key entry should be a public key", t, func() {
		e := &GobEntry{}
		err := sysValidateEntry(h, KeyEntryDef, e, nil)
		So(IsValidationFailedErr(err), ShouldBeTrue)
		e.C = []byte{1, 2, 3}
		err = sysValidateEntry(h, KeyEntryDef, e, nil)
		So(IsValidationFailedErr(err), ShouldBeTrue)

		e.C = "not b58 encoded public key!"
		err = sysValidateEntry(h, KeyEntryDef, e, nil)
		So(IsValidationFailedErr(err), ShouldBeTrue)

		pk, _ := h.agent.EncodePubKey()
		e.C = pk
		err = sysValidateEntry(h, KeyEntryDef, e, nil)
		So(err, ShouldBeNil)
	})

	Convey("an agent entry should have the correct structure as defined", t, func() {
		e := &GobEntry{}
		err := sysValidateEntry(h, AgentEntryDef, e, nil)
		So(IsValidationFailedErr(err), ShouldBeTrue)

		// bad agent entry (empty)
		e.C = ""
		err = sysValidateEntry(h, AgentEntryDef, e, nil)
		So(IsValidationFailedErr(err), ShouldBeTrue)

		ae, _ := h.agent.AgentEntry(nil)
		// bad public key
		ae.PublicKey = ""
		a, _ := ae.ToJSON()
		e.C = a
		err = sysValidateEntry(h, AgentEntryDef, e, nil)
		So(IsValidationFailedErr(err), ShouldBeTrue)

		ae, _ = h.agent.AgentEntry(nil)
		// bad public key
		ae.PublicKey = "not b58 encoded public key!"
		a, _ = ae.ToJSON()
		e.C = a
		err = sysValidateEntry(h, AgentEntryDef, e, nil)
		So(IsValidationFailedErr(err), ShouldBeTrue)

		ae, _ = h.agent.AgentEntry(nil)
		// bad revocation
		ae.Revocation = string([]byte{1, 2, 3})
		a, _ = ae.ToJSON()
		e.C = a
		err = sysValidateEntry(h, AgentEntryDef, e, nil)
		So(IsValidationFailedErr(err), ShouldBeTrue)

		ae, _ = h.agent.AgentEntry(nil)
		a, _ = ae.ToJSON()
		e.C = a
		err = sysValidateEntry(h, AgentEntryDef, e, nil)
		So(err, ShouldBeNil)
	})

	_, def, _ := h.GetEntryDef("rating")

	Convey("a nil entry is invalid", t, func() {
		err := sysValidateEntry(h, def, nil, nil)
		So(IsValidationFailedErr(err), ShouldBeTrue)
		So(err.Error(), ShouldEqual, "Validation Failed: nil entry invalid")
	})

	Convey("validate on a schema based entry should check entry against the schema", t, func() {
		profile := `{"firstName":"Eric"}` // missing required lastName
		_, def, _ := h.GetEntryDef("profile")

		err := sysValidateEntry(h, def, &GobEntry{C: profile}, nil)
		So(IsValidationFailedErr(err), ShouldBeTrue)
		So(err.Error(), ShouldEqual, "Validation Failed: validator profile failed: object property 'lastName' is required")
	})

	Convey("validate on a links entry should fail if not formatted correctly", t, func() {
		err := sysValidateEntry(h, def, &GobEntry{C: "badjson"}, nil)
		So(err, ShouldNotBeNil)
		So(err.Error(), ShouldEqual, "invalid links entry, invalid json: invalid character 'b' looking for beginning of value")

		err = sysValidateEntry(h, def, &GobEntry{C: `{}`}, nil)
		So(err, ShouldNotBeNil)
		So(err.Error(), ShouldEqual, "invalid links entry: you must specify at least one link")

		err = sysValidateEntry(h, def, &GobEntry{C: `{"Links":[{}]}`}, nil)
		So(err, ShouldNotBeNil)
		So(err.Error(), ShouldEqual, "invalid links entry: missing Base")

		err = sysValidateEntry(h, def, &GobEntry{C: `{"Links":[{"Base":"x","Link":"x","Tag":"sometag"}]}`}, nil)
		So(err, ShouldNotBeNil)
		So(err.Error(), ShouldEqual, "invalid links entry: Base multihash too short. must be > 3 bytes")
		err = sysValidateEntry(h, def, &GobEntry{C: `{"Links":[{"Base":"QmdRXz53TVT9qBYfbXctHyy2GpTNa6YrpAy6ZcDGG8Xhc5","Link":"x","Tag":"sometag"}]}`}, nil)
		So(err, ShouldNotBeNil)
		So(err.Error(), ShouldEqual, "invalid links entry: Link multihash too short. must be > 3 bytes")

		err = sysValidateEntry(h, def, &GobEntry{C: `{"Links":[{"Base":"QmdRXz53TVT9qBYfbXctHyy2GpTNa6YrpAy6ZcDGG8Xhc5","Link":"QmdRXz53TVT9qBYfbXctHyy2GpTNa6YrpAy6ZcDGG8Xhc5"}]}`}, nil)
		So(err, ShouldNotBeNil)
		So(err.Error(), ShouldEqual, "invalid links entry: missing Tag")
	})

	Convey("validate headers entry should fail if it doesn't match the headers entry schema", t, func() {
		err := sysValidateEntry(h, HeadersEntryDef, &GobEntry{C: ""}, nil)
		So(err, ShouldNotBeNil)
		So(err.Error(), ShouldEqual, "unexpected end of JSON input")

		err = sysValidateEntry(h, HeadersEntryDef, &GobEntry{C: `{"Fish":2}`}, nil)
		So(err, ShouldNotBeNil)
		So(err.Error(), ShouldEqual, "Validation Failed: validator %header failed: value must be a slice (was: map[string]interface {})")

	})

	Convey("validate headers entry should succeed on valid entry", t, func() {
		hd := h.Chain().Top()
		j, _ := hd.ToJSON()
		entryStr := fmt.Sprintf(`[{"Header":%s,"Role":"someRole","Source":"%s"}]`, j, h.nodeID.Pretty())
		err := sysValidateEntry(h, HeadersEntryDef, &GobEntry{C: entryStr}, nil)
		So(err, ShouldBeNil)
	})

	Convey("validate del entry should fail if it doesn't match the del entry schema", t, func() {
		err := sysValidateEntry(h, DelEntryDef, &GobEntry{C: ""}, nil)
		So(err, ShouldNotBeNil)
		So(err.Error(), ShouldEqual, "unexpected end of JSON input")

		err = sysValidateEntry(h, DelEntryDef, &GobEntry{C: `{"Fish":2}`}, nil)
		So(err, ShouldNotBeNil)
		So(err.Error(), ShouldEqual, "Validation Failed: validator %del failed: object property 'Hash' is required")

		err = sysValidateEntry(h, DelEntryDef, &GobEntry{C: `{"Hash": "not-a-hash"}`}, nil)
		So(err, ShouldNotBeNil)
		So(err.Error(), ShouldEqual, "Validation Failed: Error (input isn't valid multihash) when decoding Hash value 'not-a-hash'")

		err = sysValidateEntry(h, DelEntryDef, &GobEntry{C: `{"Hash": 1}`}, nil)
		So(err, ShouldNotBeNil)
		So(err.Error(), ShouldEqual, "Validation Failed: validator %del failed: object property 'Hash' validation failed: value is not a string (Kind: float64)")

	})

	Convey("validate del entry should succeed on valid entry", t, func() {
		err := sysValidateEntry(h, DelEntryDef, &GobEntry{C: `{"Hash": "QmUfY4WeqD3UUfczjdkoFQGEgCAVNf7rgFfjdeTbr7JF1C","Message": "obsolete"}`}, nil)
		So(err, ShouldBeNil)
	})

}

func TestSysValidateMod(t *testing.T) {
	d, _, h := PrepareTestChain("test")
	defer CleanupTestChain(h, d)

	hash := commit(h, "evenNumbers", "2")
	_, def, _ := h.GetEntryDef("evenNumbers")

	/* This is actually bogus because it assumes we have the entry type in our chain but
	           might be in a different chain.
		Convey("it should check that entry types match on mod", t, func() {
			a := NewModAction("oddNumbers", &GobEntry{}, hash)
			err := a.SysValidation(h, def, nil, []peer.ID{h.nodeID})
			So(err, ShouldEqual, ErrEntryTypeMismatch)
		})
	*/

	Convey("it should check that entry isn't linking ", t, func() {
		a := NewModAction("rating", &GobEntry{}, hash)
		_, ratingsDef, _ := h.GetEntryDef("rating")
		err := a.SysValidation(h, ratingsDef, nil, []peer.ID{h.nodeID})
		So(err, ShouldEqual, ErrModInvalidForLinks)
	})

	Convey("it should check that entry validates", t, func() {
		a := NewModAction("evenNumbers", nil, hash)
		err := a.SysValidation(h, def, nil, []peer.ID{h.nodeID})
		So(err, ShouldEqual, ErrNilEntryInvalid)
	})

	Convey("it should check that header isn't missing", t, func() {
		a := NewModAction("evenNumbers", &GobEntry{}, hash)
		err := a.SysValidation(h, def, nil, []peer.ID{h.nodeID})
		So(err, ShouldBeError)
		So(err, ShouldEqual, ErrModMissingHeader)
	})

	Convey("it should check that replaces is doesn't make a loop", t, func() {
		a := NewModAction("evenNumbers", &GobEntry{}, hash)
		a.header = &Header{EntryLink: hash}
		err := a.SysValidation(h, def, nil, []peer.ID{h.nodeID})
		So(err, ShouldBeError)
		So(err, ShouldEqual, ErrModReplacesHashNotDifferent)
	})

}

func TestSysValidateDel(t *testing.T) {
	d, _, h := PrepareTestChain("test")
	defer CleanupTestChain(h, d)

	hash := commit(h, "evenNumbers", "2")
	//	_, def, _ := h.GetEntryDef("evenNumbers")

	Convey("it should check that entry isn't linking ", t, func() {
		a := NewDelAction(DelEntry{Hash: hash})
		_, ratingsDef, _ := h.GetEntryDef("rating")
		err := a.SysValidation(h, ratingsDef, nil, []peer.ID{h.nodeID})
		So(err, ShouldBeError)
	})
}

func TestCheckArgCount(t *testing.T) {
	Convey("it should check for wrong number of args", t, func() {
		args := []Arg{{}}
		err := checkArgCount(args, 2)
		So(err, ShouldEqual, ErrWrongNargs)

		// test with args that are optional: two that are required and one not
		args = []Arg{{}, {}, {Optional: true}}
		err = checkArgCount(args, 1)
		So(err, ShouldEqual, ErrWrongNargs)

		err = checkArgCount(args, 2)
		So(err, ShouldBeNil)

		err = checkArgCount(args, 3)
		So(err, ShouldBeNil)

		err = checkArgCount(args, 4)
		So(err, ShouldEqual, ErrWrongNargs)
	})
}

func TestActionCommit(t *testing.T) {
	nodesCount := 3
	mt := setupMultiNodeTesting(nodesCount)
	defer mt.cleanupMultiNodeTesting()

	h := mt.nodes[0]

	profileHash := commit(h, "profile", `{"firstName":"Zippy","lastName":"Pinhead"}`)
	linksHash := commit(h, "rating", fmt.Sprintf(`{"Links":[{"Base":"%s","Link":"%s","Tag":"4stars"}]}`, h.nodeIDStr, profileHash.String()))
	ringConnect(t, mt.ctx, mt.nodes, nodesCount)

	Convey("when committing a link the linkEntry itself should be published to the DHT", t, func() {
		req := GetReq{H: linksHash, GetMask: GetMaskEntry}
		_, err := callGet(h, req, &GetOptions{GetMask: req.GetMask})
		So(err, ShouldBeNil)

		h2 := mt.nodes[2]
		_, err = callGet(h2, req, &GetOptions{GetMask: req.GetMask})
		So(err, ShouldBeNil)

	})
}

func TestActionDelete(t *testing.T) {
	nodesCount := 3
	mt := setupMultiNodeTesting(nodesCount)
	defer mt.cleanupMultiNodeTesting()

	h := mt.nodes[0]
	ringConnect(t, mt.ctx, mt.nodes, nodesCount)

	profileHash := commit(h, "profile", `{"firstName":"Zippy","lastName":"Pinhead"}`)
	entry := DelEntry{Hash: profileHash, Message: "expired"}
	a := &ActionDel{entry: entry}
	response, err := h.commitAndShare(a, NullHash())
	if err != nil {
		panic(err)
	}
	deleteHash := response.(Hash)

	Convey("when deleting a hash the del entry itself should be published to the DHT", t, func() {
		req := GetReq{H: deleteHash, GetMask: GetMaskEntry}
		_, err := callGet(h, req, &GetOptions{GetMask: req.GetMask})
		So(err, ShouldBeNil)

		h2 := mt.nodes[2]
		_, err = callGet(h2, req, &GetOptions{GetMask: req.GetMask})
		So(err, ShouldBeNil)
	})
}

func TestActionGet(t *testing.T) {
	nodesCount := 3
	mt := setupMultiNodeTesting(nodesCount)
	defer mt.cleanupMultiNodeTesting()

	h := mt.nodes[0]

	e := GobEntry{C: "3"}
	hash, _ := e.Sum(h.hashSpec)

	Convey("receive should return not found if it doesn't exist", t, func() {
		m := h.node.NewMessage(GET_REQUEST, GetReq{H: hash})
		_, err := ActionReceiver(h, m)
		So(err, ShouldEqual, ErrHashNotFound)

		options := GetOptions{}
		a := ActionGet{GetReq{H: hash}, &options}
		fn := &APIFnGet{action: a}
		response, err := fn.Call(h)
		So(err, ShouldEqual, ErrHashNotFound)
		So(fmt.Sprintf("%v", response), ShouldEqual, "<nil>")

	})

	commit(h, "oddNumbers", "3")
	m := h.node.NewMessage(GET_REQUEST, GetReq{H: hash})
	Convey("receive should return value if it exists", t, func() {
		r, err := ActionReceiver(h, m)
		So(err, ShouldBeNil)
		resp := r.(GetResp)
		So(resp.Entry.Content().(string), ShouldEqual, "3")
	})

	ringConnect(t, mt.ctx, mt.nodes, nodesCount)
	Convey("receive should return closer peers if it can", t, func() {
		h2 := mt.nodes[2]
		r, err := ActionReceiver(h2, m)
		So(err, ShouldBeNil)
		resp := r.(CloserPeersResp)
		So(len(resp.CloserPeers), ShouldEqual, 1)
		So(peer.ID(resp.CloserPeers[0].ID).Pretty(), ShouldEqual, "QmUfY4WeqD3UUfczjdkoFQGEgCAVNf7rgFfjdeTbr7JF1C")
	})

	Convey("get should return not found if hash doesn't exist and we are connected", t, func() {
		hash, err := NewHash("QmY8Mzg9F69e5P9AoQPYat655HEhc1TVGs11tmfNSzfrom")
		if err != nil {
			panic(err)
		}

		options := GetOptions{}
		a := ActionGet{GetReq{H: hash}, &options}
		fn := &APIFnGet{action: a}
		response, err := fn.Call(h)
		So(err, ShouldEqual, ErrHashNotFound)
		So(response, ShouldBeNil)

	})

}

func TestActionGetLocal(t *testing.T) {
	d, _, h := PrepareTestChain("test")
	defer CleanupTestChain(h, d)

	hash := commit(h, "secret", "31415")

	Convey("non local get should fail for private entries", t, func() {
		req := GetReq{H: hash, GetMask: GetMaskEntry}
		_, err := callGet(h, req, &GetOptions{GetMask: req.GetMask})
		So(err.Error(), ShouldEqual, "hash not found")
	})

	Convey("it should fail to get non-existent private local values", t, func() {
		badHash, _ := NewHash("QmY8Mzg9F69e5P9AoQPYat655HEhc1TVGs11tmfNSzkqh2")
		req := GetReq{H: badHash, GetMask: GetMaskEntry}
		_, err := callGet(h, req, &GetOptions{GetMask: req.GetMask, Local: true})
		So(err.Error(), ShouldEqual, "hash not found")
	})

	Convey("it should get private local values", t, func() {
		req := GetReq{H: hash, GetMask: GetMaskEntry}
		rsp, err := callGet(h, req, &GetOptions{GetMask: req.GetMask, Local: true})
		So(err, ShouldBeNil)
		getResp := rsp.(GetResp)
		So(getResp.Entry.Content().(string), ShouldEqual, "31415")
	})

	Convey("it should get local bundle values", t, func() {
		_, err := NewStartBundleAction(0, "myBundle").Call(h)
		So(err, ShouldBeNil)
		hash := commit(h, "oddNumbers", "3141")
		req := GetReq{H: hash, GetMask: GetMaskEntry}
		_, err = callGet(h, req, &GetOptions{GetMask: req.GetMask, Local: true})
		So(err, ShouldEqual, ErrHashNotFound)
		rsp, err := callGet(h, req, &GetOptions{GetMask: req.GetMask, Bundle: true})
		So(err, ShouldBeNil)
		getResp := rsp.(GetResp)
		So(getResp.Entry.Content().(string), ShouldEqual, "3141")
	})
}

func TestActionBundle(t *testing.T) {
	d, _, h := PrepareTestChain("test")
	defer CleanupTestChain(h, d)
	Convey("bundle action constructor should set timeout", t, func() {
		a := NewStartBundleAction(0, "myBundle")
		So(a.timeout, ShouldEqual, DefaultBundleTimeout)
		So(a.userParam, ShouldEqual, "myBundle")
		a = NewStartBundleAction(123, "myBundle")
		So(a.timeout, ShouldEqual, 123)
	})

	Convey("starting a bundle should set the bundle start point", t, func() {
		c := h.Chain()
		So(c.BundleStarted(), ShouldBeNil)
		a := NewStartBundleAction(100, "myBundle")
		_, err := a.Call(h)
		So(err, ShouldBeNil)
		So(c.BundleStarted().idx, ShouldEqual, c.Length()-1)
	})
	var hash Hash
	Convey("commit actions should commit to bundle after it's started", t, func() {
		So(h.chain.Length(), ShouldEqual, 2)
		So(h.chain.bundle.chain.Length(), ShouldEqual, 0)
		hash = commit(h, "oddNumbers", "99")

		So(h.chain.Length(), ShouldEqual, 2)
		So(h.chain.bundle.chain.Length(), ShouldEqual, 1)
	})
	Convey("but those commits should not show in the DHT", t, func() {
		_, _, _, _, err := h.dht.Get(hash, StatusDefault, GetMaskDefault)
		So(err, ShouldEqual, ErrHashNotFound)
	})

	Convey("closing a bundle should commit its entries to the chain", t, func() {
		So(h.chain.Length(), ShouldEqual, 2)
		a := &APIFnCloseBundle{commit: true}
		So(a.commit, ShouldEqual, true)
		_, err := a.Call(h)
		So(err, ShouldBeNil)
		So(h.chain.Length(), ShouldEqual, 3)
	})
	Convey("and those commits should now show in the DHT", t, func() {
		data, _, _, _, err := h.dht.Get(hash, StatusDefault, GetMaskDefault)
		So(err, ShouldBeNil)
		var e GobEntry
		err = e.Unmarshal(data)

		So(e.C, ShouldEqual, "99")
	})

	Convey("canceling a bundle should not commit entries to chain and should execute the bundleCanceled callback", t, func() {
		So(h.chain.Length(), ShouldEqual, 3)

		_, err := NewStartBundleAction(0, "debugit").Call(h)
		So(err, ShouldBeNil)
		commit(h, "oddNumbers", "7")

		a := &APIFnCloseBundle{commit: false}
		So(a.commit, ShouldEqual, false)
		ShouldLog(h.nucleus.alog, func() {
			_, err = a.Call(h)
			So(err, ShouldBeNil)
		}, `debug message during bundleCanceled with reason: userCancel`)
		So(h.chain.Length(), ShouldEqual, 3)
		So(h.chain.BundleStarted(), ShouldBeNil)
	})
	Convey("canceling a bundle should still commit entries if bundleCanceled returns BundleCancelResponseCommit", t, func() {
		So(h.chain.Length(), ShouldEqual, 3)

		_, err := NewStartBundleAction(0, "cancelit").Call(h)
		So(err, ShouldBeNil)
		commit(h, "oddNumbers", "7")
		a := &APIFnCloseBundle{commit: false}
		So(a.commit, ShouldEqual, false)
		ShouldLog(h.nucleus.alog, func() {
			_, err = a.Call(h)
			So(err, ShouldBeNil)
		}, `debug message during bundleCanceled: canceling cancel!`)
		So(h.chain.BundleStarted(), ShouldNotBeNil)
	})
}

func TestActionSigning(t *testing.T) {
	d, _, h := PrepareTestChain("test")
	defer CleanupTestChain(h, d)

	privKey := h.agent.PrivKey()
	sig, err := privKey.Sign([]byte("3"))
	if err != nil {
		panic(err)
	}

	var b58sig string
	Convey("sign action should return a b58 encoded signature", t, func() {
		fn := &APIFnSign{[]byte("3")}
		result, err := fn.Call(h)
		So(err, ShouldBeNil)
		b58sig = result.(string)

		So(b58sig, ShouldEqual, b58.Encode(sig))
	})
	var pubKey string
	pubKey, err = h.agent.EncodePubKey()
	if err != nil {
		panic(err)
	}

	Convey("verify signture action should test a signature", t, func() {
		fn := &APIFnVerifySignature{b58signature: b58sig, data: string([]byte("3")), b58pubKey: pubKey}
		result, err := fn.Call(h)
		So(err, ShouldBeNil)
		So(result.(bool), ShouldBeTrue)
		fn = &APIFnVerifySignature{b58signature: b58sig, data: string([]byte("34")), b58pubKey: pubKey}
		result, err = fn.Call(h)
		So(err, ShouldBeNil)
		So(result.(bool), ShouldBeFalse)
	})
}
