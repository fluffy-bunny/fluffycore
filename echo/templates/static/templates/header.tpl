{{define "head_links"}}
  {{range $idx,$link := .headLinks}}
    <link href="{{ $link.HREF }}" rel="{{ $link.REL }}" />      
  {{end}}
{{end}}

{{define "header"}}
<head>
      <meta charset="utf-8" />
      <meta name="viewport" content="width=device-width, initial-scale=1, shrink-to-fit=no" />
      <meta name="description" content="" />
      <meta name="author" content="" />
      <title>{{ .title }}</title>
      <!-- Favicon-->
      <link rel="icon" type="image/x-icon" href="static/assets/favicon.ico" />
      <!-- Core theme CSS (includes Bootstrap)-->
      {{template "head_links" .}}

</head>
{{end}}

