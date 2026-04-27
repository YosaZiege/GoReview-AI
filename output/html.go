package output

import (
	"html/template"
	"io"

	"github.com/yourorg/gorview/core"
)

var htmlTmpl *template.Template

func init() {
	htmlTmpl = template.Must(template.New("report").Funcs(template.FuncMap{
		"scoreClass": func(score int) string {
			switch {
			case score >= 80:
				return "good"
			case score >= 60:
				return "warn"
			default:
				return "bad"
			}
		},
	}).Parse(htmlSrc))
}

// PrintHTML writes an HTML report to w.
func PrintHTML(w io.Writer, r core.Report) error {
	return htmlTmpl.Execute(w, r)
}

const htmlSrc = `<!DOCTYPE html>
<html lang="fr">
<head>
  <meta charset="UTF-8">
  <title>GoReview AI — Rapport</title>
  <style>
    body { font-family: system-ui, sans-serif; max-width: 960px; margin: 2rem auto; padding: 0 1rem; color: #1f2937; }
    h1 { color: #1a56db; margin-bottom: 0.25rem; }
    .meta { color: #6b7280; margin-bottom: 1rem; }
    .score { font-size: 1.75rem; font-weight: 700; margin: 1rem 0; }
    .score.good { color: #16a34a; }
    .score.warn { color: #d97706; }
    .score.bad  { color: #dc2626; }
    table { width: 100%; border-collapse: collapse; margin-top: 1rem; font-size: 0.9rem; }
    th { background: #1a56db; color: #fff; padding: 0.6rem 1rem; text-align: left; }
    td { border: 1px solid #e5e7eb; padding: 0.5rem 1rem; vertical-align: top; }
    tr:nth-child(even) td { background: #f9fafb; }
    .CRITIQUE { color: #dc2626; font-weight: 700; }
    .MOYEN    { color: #d97706; font-weight: 700; }
    .FAIBLE   { color: #2563eb; }
    .explanation { font-style: italic; color: #4b5563; font-size: 0.85rem; }
  </style>
</head>
<body>
<h1>GoReview AI</h1>
<p class="meta">Répertoire : <code>{{.Dir}}</code></p>
<div class="score {{scoreClass .Score}}">Score de maintenabilité : {{.Score}} / 100</div>

<table>
  <tr>
    <th>Sévérité</th>
    <th>Problème</th>
    <th>Localisation</th>
    <th>Composant</th>
    <th>Patron suggéré</th>
    <th>Effort</th>
  </tr>
  {{range .Findings}}
  <tr>
    <td class="{{.Severity}}">{{.Severity}}</td>
    <td>{{.SmellType}}</td>
    <td><code>{{.File}}:{{.Line}}</code></td>
    <td>{{.Component}}</td>
    <td>{{.Pattern}}</td>
    <td>{{.Effort}}</td>
  </tr>
  {{if .Explanation}}
  <tr>
    <td colspan="6" class="explanation">{{.Explanation}}</td>
  </tr>
  {{end}}
  {{end}}
</table>
</body>
</html>
`
