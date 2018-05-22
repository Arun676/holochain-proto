package holochain

import (
	"bytes"
	"encoding/json"
	"fmt"
	. "github.com/holochain/holochain-proto/hash"
	. "github.com/smartystreets/goconvey/convey"

	"path/filepath"
	"testing"
)

func TestGob(t *testing.T) {
	g := GobEntry{C: mkTestHeader("evenNumbers")}
	v, err := g.Marshal()
	Convey("it should encode", t, func() {
		So(err, ShouldBeNil)
	})
	var g2 GobEntry
	err = g2.Unmarshal(v)
	Convey("it should decode", t, func() {
		sg1 := fmt.Sprintf("%v", g)
		sg2 := fmt.Sprintf("%v", g)
		So(err, ShouldBeNil)
		So(sg1, ShouldEqual, sg2)
	})
}

func TestJSONEntry(t *testing.T) {
	/* Not yet implemented or used
	g := JSONEntry{C:Config{Port:8888}}
	v,err := g.Marshal()
	ExpectNoErr(t,err)
	var g2 JSONEntry
	err = g2.Unmarshal(v)
	ExpectNoErr(t,err)
	if g2!=g {t.Error("expected JSON match! "+fmt.Sprintf("%v",g)+" "+fmt.Sprintf("%v",g2))}
	*/
}

func testValidateJSON(ed EntryDef, err error) {
	So(err, ShouldBeNil)
	So(ed.validator, ShouldNotBeNil)
	profile := `{"firstName":"Eric","lastName":"H-B"}`

	var input interface{}
	if err = json.Unmarshal([]byte(profile), &input); err != nil {
		panic(err)
	}
	err = ed.validator.Validate(input)
	So(err, ShouldBeNil)
	profile = `{"firstName":"Eric"}`
	if err = json.Unmarshal([]byte(profile), &input); err != nil {
		panic(err)
	}

	err = ed.validator.Validate(input)
	So(err, ShouldNotBeNil)
	So(err.Error(), ShouldEqual, "validator schema_profile.json failed: object property 'lastName' is required")
}

func TestJSONSchemaValidator(t *testing.T) {
	d, _ := setupTestService()
	defer CleanupTestDir(d)

	schema := `{
	"title": "Profile Schema",
	"type": "object",
	"properties": {
		"firstName": {
			"type": "string"
		},
		"lastName": {
			"type": "string"
		},
		"age": {
			"description": "Age in years",
			"type": "integer",
			"minimum": 0
		}
	},
	"required": ["firstName", "lastName"]
}`
	edf := EntryDefFile{SchemaFile: "schema_profile.json"}

	if err := WriteFile([]byte(schema), d, edf.SchemaFile); err != nil {
		panic(err)
	}

	ed := EntryDef{Name: "schema_profile.json"}

	Convey("it should validate JSON entries from schema file", t, func() {
		err := ed.BuildJSONSchemaValidator(filepath.Join(d, ed.Name))
		testValidateJSON(ed, err)
	})

	Convey("it should validate JSON entries from string", t, func() {
		err := ed.BuildJSONSchemaValidatorFromString(schema)
		testValidateJSON(ed, err)
	})
}

func TestMarshalEntry(t *testing.T) {

	e := GobEntry{C: "some  data"}

	Convey("it should round-trip", t, func() {
		var b bytes.Buffer
		err := MarshalEntry(&b, &e)
		So(err, ShouldBeNil)
		var ne Entry
		ne, err = UnmarshalEntry(&b)
		So(err, ShouldBeNil)
		So(fmt.Sprintf("%v", ne), ShouldEqual, fmt.Sprintf("%v", &e))
	})
}

func TestAgentEntryToJSON(t *testing.T) {
	a := AgentIdentity("zippy@someemail.com")
	a1, _ := NewAgent(LibP2P, a, MakeTestSeed(""))
	pk, _ := a1.EncodePubKey()
	ae := AgentEntry{
		Identity:  a,
		PublicKey: pk,
	}

	var j string
	var err error
	Convey("it should convert to JSON", t, func() {
		j, err = ae.ToJSON()
		So(err, ShouldBeNil)
		So(j, ShouldEqual, `{"Identity":"zippy@someemail.com","Revocation":"","PublicKey":"4XTTM4Lf8pAWo6dfra223t4ZK7gjAjFA49VdwrC1wVHQqb8nH"}`)
	})

	Convey("it should convert from JSON", t, func() {
		ae2, err := AgentEntryFromJSON(j)
		So(err, ShouldBeNil)
		So(ae2, ShouldResemble, ae)
	})

}

func TestDelEntryToJSON(t *testing.T) {
	hashStr := "QmVGtdTZdTFaLsaj2RwdVG8jcjNNcp1DE914DKZ2kHmXHx"
	hash, _ := NewHash(hashStr)
	e1 := DelEntry{
		Hash:    hash,
		Message: "my message",
	}

	var j string
	var err error
	Convey("it should convert to JSON", t, func() {
		j, err = e1.ToJSON()
		So(err, ShouldBeNil)
		So(j, ShouldEqual, fmt.Sprintf(`{"Hash":"%s","Message":"my message"}`, hashStr))
	})

	Convey("it should convert from JSON", t, func() {
		e2, err := DelEntryFromJSON(j)
		So(err, ShouldBeNil)
		So(e2.Message, ShouldEqual, e1.Message)
		So(e2.Hash.String(), ShouldEqual, e1.Hash.String())
	})

}
