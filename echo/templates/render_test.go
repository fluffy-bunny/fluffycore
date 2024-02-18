package templates

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/rs/xid"
	"github.com/stretchr/testify/require"
)

const (
	customHeaderStuff = `
<link href="/static/css/styles.css" rel="stylesheet" />
<link href="/static/css/json-viewer.css" rel="stylesheet" />
`
)

type HeadLink struct {
	HREF string
	REL  string
}

var headLinks = []HeadLink{
	{HREF: "/static/css/styles.css", REL: "stylesheet"},
	{HREF: "/static/css/json-viewer.css", REL: "stylesheet"},
}

func TestPlainTextTemplate(t *testing.T) {
	htmlTemplate, err := FindAndParseTemplates("./static/templates", nil)
	require.NoError(t, err)
	require.NotNil(t, htmlTemplate)

	// create a stream writer to a buffer
	streamWriter := new(bytes.Buffer)

	rand := xid.New().String()
	data := map[string]interface{}{
		"user": rand,
	}
	err = htmlTemplate.ExecuteTemplate(streamWriter, "emails/test/txt", data)
	require.NoError(t, err)
	// write out the buffer
	bb := streamWriter.Bytes()
	require.NotNil(t, bb)
	bbS := string(bb)
	fmt.Println(string(bb))

	require.Contains(t, bbS, fmt.Sprintf("Hello %s", rand))

}

func TestHtmlTemplate(t *testing.T) {
	htmlTemplate, err := FindAndParseTemplates("./static/templates", nil)
	require.NoError(t, err)
	require.NotNil(t, htmlTemplate)

	// create a stream writer to a buffer
	streamWriter := new(bytes.Buffer)

	rand := xid.New().String()
	data := map[string]interface{}{
		"title":             rand,
		"baseBodyClass":     rand,
		"customHeaderStuff": customHeaderStuff,
		"headLinks":         headLinks,
	}
	//	bbD, err := json.Marshal(data)
	//	require.NoError(t, err)
	//	err = json.Unmarshal(bbD, &data)
	//	require.NoError(t, err)

	err = htmlTemplate.ExecuteTemplate(streamWriter, "emails/test/index", data)
	require.NoError(t, err)

	// write out the buffer
	bb := streamWriter.Bytes()
	require.NotNil(t, bb)
	bbS := string(bb)
	fmt.Println(string(bb))

	require.Contains(t, bbS, fmt.Sprintf("<title>%s</title>", rand))
	require.Contains(t, bbS, fmt.Sprintf("<body class=\"%s\">", rand))
}
