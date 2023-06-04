package utils

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/fluffy-bunny/fluffycore/utils"
	"github.com/stretchr/testify/require"
)

func TestRedact(t *testing.T) {
	type Sensitive struct {
		Name     string `json:"name"`
		Password string `json:"password" redact:"true"`
	}
	obj := &Sensitive{
		Name:     "John",
		Password: "secret",
	}
	fmt.Println(utils.PrettyJSON(obj))
	jsonV, _ := json.Marshal(obj)
	fmt.Println(string(jsonV))

	dst := &Sensitive{}
	PrettyPrintRedacted(obj, dst)
	require.NotEqual(t, obj.Password, dst.Password)
}

func TestCloneAndRedact(t *testing.T) {
	type Sensitive struct {
		Name     string `json:"name"`
		Password string `json:"password" redact:"true"`
	}
	obj := &Sensitive{
		Name:     "John",
		Password: "secret",
	}
	fmt.Println(utils.PrettyJSON(obj))
	jsonV, _ := json.Marshal(obj)
	fmt.Println(string(jsonV))

	dst, err := CloneAndRedact(obj)
	require.NoError(t, err)
	sDst := dst.(*Sensitive)
	require.NotEqual(t, obj.Password, sDst.Password)

	jsonV, _ = json.Marshal(dst)
	fmt.Println(string(jsonV))

}
