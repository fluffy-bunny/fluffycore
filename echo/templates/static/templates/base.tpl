{{define "base_begin"}}
{{template "html_begin" .}}
{{template "header" .}}
<body class="{{ .baseBodyClass }}">
{{end}}
 
{{define "base_end"}}
</body>
{{template "footer" .}}
{{template "html_end" .}}
{{end}}