package output

import (
	"html/template"
	"io"
	"strings"

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
		"sevClass": func(sev string) string {
			s := strings.ToLower(sev)
			switch s {
			case "high", "critique", "critical":
				return "sev-high"
			case "medium", "moyen", "med":
				return "sev-med"
			case "low", "faible":
				return "sev-low"
			}
			return "sev-other"
		},
		"groupBySeverity": func(findings []core.Finding) []findingGroup {
			high, med, low, other := []core.Finding{}, []core.Finding{}, []core.Finding{}, []core.Finding{}
			for _, f := range findings {
				switch strings.ToLower(string(f.Severity)) {
				case "high", "critique", "critical":
					high = append(high, f)
				case "medium", "moyen", "med":
					med = append(med, f)
				case "low", "faible":
					low = append(low, f)
				default:
					other = append(other, f)
				}
			}
			groups := []findingGroup{
				{Label: "Critique", Class: "sev-high", Items: high},
				{Label: "Moyen", Class: "sev-med", Items: med},
				{Label: "Faible", Class: "sev-low", Items: low},
			}
			if len(other) > 0 {
				groups = append(groups, findingGroup{Label: "Autre", Class: "sev-other", Items: other})
			}
			return groups
		},
		"detectLanguage": func(path string) string {
			lp := strings.ToLower(path)
			switch {
			case strings.HasSuffix(lp, ".go"):
				return "go"
			case strings.HasSuffix(lp, ".py"):
				return "python"
			case strings.HasSuffix(lp, ".js"), strings.HasSuffix(lp, ".jsx"):
				return "javascript"
			case strings.HasSuffix(lp, ".ts"), strings.HasSuffix(lp, ".tsx"):
				return "typescript"
			}
			return "code"
		},
		"fileName": func(path string) string {
			p := strings.ReplaceAll(path, "\\", "/")
			if i := strings.LastIndex(p, "/"); i >= 0 {
				return p[i+1:]
			}
			return p
		},
	}).Parse(htmlSrc))
}

type findingGroup struct {
	Label string
	Class string
	Items []core.Finding
}

// PrintHTML writes a standalone HTML report to w.
func PrintHTML(w io.Writer, r core.Report) error {
	return htmlTmpl.Execute(w, r)
}

const htmlSrc = `<!DOCTYPE html>
<html lang="fr">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>GoReview AI — Rapport</title>
  <style>
    :root {
      --bg: #0f172a;
      --panel: #1e293b;
      --panel-2: #273449;
      --code-bg: #0b1220;
      --code-header: #1a2436;
      --text: #e2e8f0;
      --muted: #94a3b8;
      --border: #334155;
      --accent: #60a5fa;
      --good: #22c55e;
      --warn: #f59e0b;
      --bad: #ef4444;
      --high: #ef4444;
      --med:  #f59e0b;
      --low:  #3b82f6;
    }
    * { box-sizing: border-box; }
    body {
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", system-ui, sans-serif;
      background: var(--bg);
      color: var(--text);
      margin: 0;
      line-height: 1.5;
    }
    code, pre {
      font-family: "JetBrains Mono", "Fira Code", "SF Mono", Consolas, Menlo, monospace;
    }
    .container { max-width: 1100px; margin: 0 auto; padding: 2rem 1.5rem 4rem; }

    header {
      display: flex;
      justify-content: space-between;
      align-items: flex-end;
      gap: 2rem;
      flex-wrap: wrap;
      padding-bottom: 1.5rem;
      border-bottom: 1px solid var(--border);
      margin-bottom: 2rem;
    }
    h1 { font-size: 1.75rem; margin: 0 0 0.25rem; letter-spacing: -0.02em; }
    h1 .accent { color: var(--accent); }
    .meta { color: var(--muted); font-size: 0.9rem; }
    .meta code {
      background: var(--panel);
      padding: 0.15rem 0.45rem;
      border-radius: 4px;
      font-size: 0.85rem;
    }

    .score-card {
      background: var(--panel);
      border: 1px solid var(--border);
      border-radius: 12px;
      padding: 1.25rem 1.75rem;
      min-width: 220px;
      text-align: right;
    }
    .score-label {
      font-size: 0.75rem;
      text-transform: uppercase;
      letter-spacing: 0.1em;
      color: var(--muted);
    }
    .score-value {
      font-size: 2.75rem;
      font-weight: 700;
      line-height: 1;
      margin-top: 0.25rem;
      font-variant-numeric: tabular-nums;
    }
    .score-value.good { color: var(--good); }
    .score-value.warn { color: var(--warn); }
    .score-value.bad  { color: var(--bad); }
    .score-value .denom { font-size: 1rem; color: var(--muted); font-weight: 500; }

    .summary {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(140px, 1fr));
      gap: 0.75rem;
      margin-bottom: 2rem;
    }
    .pill {
      background: var(--panel);
      border: 1px solid var(--border);
      border-radius: 10px;
      padding: 0.75rem 1rem;
      display: flex;
      align-items: center;
      gap: 0.75rem;
    }
    .pill .dot { width: 10px; height: 10px; border-radius: 50%; flex-shrink: 0; }
    .pill.sev-high .dot { background: var(--high); }
    .pill.sev-med  .dot { background: var(--med); }
    .pill.sev-low  .dot { background: var(--low); }
    .pill.sev-other .dot { background: var(--muted); }
    .pill .count { font-size: 1.25rem; font-weight: 600; }
    .pill .label { color: var(--muted); font-size: 0.85rem; }

    .group { margin-top: 2rem; }
    .group-header {
      display: flex;
      align-items: baseline;
      gap: 0.75rem;
      margin-bottom: 1rem;
    }
    .group-header h2 { font-size: 1.1rem; margin: 0; letter-spacing: -0.01em; }
    .group-header .group-count { color: var(--muted); font-size: 0.9rem; }
    .badge {
      display: inline-block;
      padding: 0.15rem 0.55rem;
      border-radius: 999px;
      font-size: 0.7rem;
      font-weight: 600;
      letter-spacing: 0.05em;
      text-transform: uppercase;
    }
    .badge.sev-high { background: rgba(239, 68, 68, 0.15); color: var(--high); }
    .badge.sev-med  { background: rgba(245, 158, 11, 0.15); color: var(--med); }
    .badge.sev-low  { background: rgba(59, 130, 246, 0.15); color: var(--low); }
    .badge.sev-other{ background: rgba(148, 163, 184, 0.15); color: var(--muted); }

    details.finding {
      background: var(--panel);
      border: 1px solid var(--border);
      border-left-width: 3px;
      border-radius: 8px;
      margin-bottom: 0.6rem;
      overflow: hidden;
    }
    details.finding.sev-high { border-left-color: var(--high); }
    details.finding.sev-med  { border-left-color: var(--med); }
    details.finding.sev-low  { border-left-color: var(--low); }
    details.finding.sev-other{ border-left-color: var(--muted); }

    details.finding summary {
      cursor: pointer;
      padding: 0.85rem 1rem;
      display: grid;
      grid-template-columns: auto 1fr auto;
      align-items: center;
      gap: 0.85rem;
      list-style: none;
    }
    details.finding summary::-webkit-details-marker { display: none; }
    details.finding summary::before {
      content: "▸";
      color: var(--muted);
      font-size: 0.8rem;
      transition: transform 0.15s;
      display: inline-block;
    }
    details.finding[open] summary::before { transform: rotate(90deg); }

    .finding-title { font-weight: 600; font-size: 0.95rem; }
    .finding-meta { color: var(--muted); font-size: 0.82rem; margin-top: 0.15rem; }
    .finding-meta code {
      background: var(--panel-2);
      padding: 0.1rem 0.35rem;
      border-radius: 3px;
      font-size: 0.78rem;
    }
    .finding-side { text-align: right; font-size: 0.8rem; color: var(--muted); }
    .finding-side .pattern { color: var(--text); font-weight: 500; }

    .finding-body {
      padding: 0 1rem 1rem;
      border-top: 1px solid var(--border);
      padding-top: 1rem;
    }
    .finding-body h3 {
      font-size: 0.75rem;
      text-transform: uppercase;
      letter-spacing: 0.08em;
      color: var(--muted);
      margin: 1.25rem 0 0.5rem;
    }
    .finding-body h3:first-child { margin-top: 0; }
    .finding-body p { margin: 0.5rem 0; }
    .metrics { display: flex; flex-wrap: wrap; gap: 0.5rem; }
    .metric {
      background: var(--panel-2);
      padding: 0.25rem 0.6rem;
      border-radius: 4px;
      font-size: 0.8rem;
      color: var(--muted);
    }
    .metric strong { color: var(--text); }

    /* === macOS-style code panel === */
    .code-panel {
      background: var(--code-bg);
      border: 1px solid var(--border);
      border-radius: 10px;
      overflow: hidden;
      box-shadow: 0 4px 12px rgba(0, 0, 0, 0.25);
      display: flex;
      flex-direction: column;
      min-width: 0;
    }
    .code-header {
      background: var(--code-header);
      padding: 0.5rem 0.85rem;
      display: flex;
      align-items: center;
      gap: 0.65rem;
      border-bottom: 1px solid var(--border);
    }
    .traffic-lights { display: flex; gap: 0.4rem; flex-shrink: 0; }
    .traffic-lights span {
      width: 12px;
      height: 12px;
      border-radius: 50%;
      display: block;
    }
    .traffic-lights .tl-close { background: #ff5f57; }
    .traffic-lights .tl-min   { background: #febc2e; }
    .traffic-lights .tl-max   { background: #28c840; }
    .code-title {
      flex: 1;
      text-align: center;
      font-size: 0.78rem;
      color: var(--muted);
      font-family: "JetBrains Mono", Consolas, monospace;
      letter-spacing: 0.02em;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
      padding: 0 0.5rem;
    }
    .code-tag {
      font-size: 0.65rem;
      background: var(--panel-2);
      color: var(--muted);
      padding: 0.15rem 0.5rem;
      border-radius: 999px;
      text-transform: uppercase;
      letter-spacing: 0.08em;
      flex-shrink: 0;
    }
    .copy-btn {
      background: transparent;
      border: 1px solid var(--border);
      color: var(--muted);
      font-size: 0.7rem;
      padding: 0.2rem 0.55rem;
      border-radius: 4px;
      cursor: pointer;
      font-family: inherit;
      flex-shrink: 0;
      transition: all 0.15s;
    }
    .copy-btn:hover {
      color: var(--text);
      border-color: var(--muted);
      background: var(--panel-2);
    }
    .copy-btn.copied { color: var(--good); border-color: var(--good); }

    .code-panel pre {
      background: transparent;
      border: none;
      border-radius: 0;
      margin: 0;
      padding: 0.9rem 1rem;
      overflow-x: auto;
      font-size: 0.82rem;
      line-height: 1.55;
      color: var(--text);
      flex: 1;
    }

    .code-grid {
      display: grid;
      grid-template-columns: 1fr;
      gap: 1rem;
    }
    @media (min-width: 900px) {
      .code-grid.has-both { grid-template-columns: 1fr 1fr; }
    }

    .empty {
      text-align: center;
      padding: 3rem 1rem;
      color: var(--muted);
      background: var(--panel);
      border: 1px dashed var(--border);
      border-radius: 12px;
    }

    footer {
      margin-top: 3rem;
      padding-top: 1.5rem;
      border-top: 1px solid var(--border);
      color: var(--muted);
      font-size: 0.8rem;
      text-align: center;
    }
  </style>
</head>
<body>
<div class="container">

  <header>
    <div>
      <h1><span class="accent">GoReview</span> AI</h1>
      <div class="meta">Répertoire analysé : <code>{{.Dir}}</code></div>
    </div>
    <div class="score-card">
      <div class="score-label">Score de maintenabilité</div>
      <div class="score-value {{scoreClass .Score}}">{{.Score}}<span class="denom"> / 100</span></div>
    </div>
  </header>

  {{$groups := groupBySeverity .Findings}}
  <div class="summary">
    {{range $groups}}
    <div class="pill {{.Class}}">
      <span class="dot"></span>
      <div>
        <div class="count">{{len .Items}}</div>
        <div class="label">{{.Label}}</div>
      </div>
    </div>
    {{end}}
  </div>

  {{if not .Findings}}
    <div class="empty">Aucun problème détecté. 🎉</div>
  {{end}}

  {{range $groups}}
    {{if .Items}}
    <section class="group">
      <div class="group-header">
        <span class="badge {{.Class}}">{{.Label}}</span>
        <h2>{{.Label}}</h2>
        <span class="group-count">· {{len .Items}} problème{{if gt (len .Items) 1}}s{{end}}</span>
      </div>

      {{range .Items}}
      {{$lang := detectLanguage .File}}
      {{$fname := fileName .File}}
      <details class="finding {{sevClass (printf "%s" .Severity)}}">
        <summary>
          <span></span>
          <div>
            <div class="finding-title">{{.SmellType}} — {{.Component}}</div>
            <div class="finding-meta"><code>{{.File}}:{{.Line}}</code></div>
          </div>
          <div class="finding-side">
            <div class="pattern">{{.Pattern}}</div>
            <div>Effort : {{.Effort}}</div>
          </div>
        </summary>
        <div class="finding-body">
          {{if .Metrics}}
            <h3>Métriques</h3>
            <div class="metrics">
              {{range $k, $v := .Metrics}}
                <span class="metric"><strong>{{$k}}</strong> = {{$v}}</span>
              {{end}}
            </div>
          {{end}}

          {{if .Explanation}}
            <h3>Explication</h3>
            <p>{{.Explanation}}</p>
          {{end}}

          {{if or .RefactorBefore .RefactorAfter}}
            <h3>Refactoring suggéré</h3>
            <div class="code-grid {{if and .RefactorBefore .RefactorAfter}}has-both{{end}}">
              {{if .RefactorBefore}}
              <div class="code-panel">
                <div class="code-header">
                  <div class="traffic-lights">
                    <span class="tl-close"></span>
                    <span class="tl-min"></span>
                    <span class="tl-max"></span>
                  </div>
                  <div class="code-title">{{$fname}} — Avant</div>
                  <span class="code-tag">{{$lang}}</span>
                  <button class="copy-btn" type="button" data-copy>Copier</button>
                </div>
                <pre><code>{{.RefactorBefore}}</code></pre>
              </div>
              {{end}}
              {{if .RefactorAfter}}
              <div class="code-panel">
                <div class="code-header">
                  <div class="traffic-lights">
                    <span class="tl-close"></span>
                    <span class="tl-min"></span>
                    <span class="tl-max"></span>
                  </div>
                  <div class="code-title">{{$fname}} — Après</div>
                  <span class="code-tag">{{$lang}}</span>
                  <button class="copy-btn" type="button" data-copy>Copier</button>
                </div>
                <pre><code>{{.RefactorAfter}}</code></pre>
              </div>
              {{end}}
            </div>
          {{end}}
        </div>
      </details>
      {{end}}
    </section>
    {{end}}
  {{end}}

  <footer>
    Généré par GoReview AI · {{len .Findings}} problème{{if gt (len .Findings) 1}}s{{end}} au total
  </footer>

</div>

<script>
  document.querySelectorAll('button[data-copy]').forEach(function (btn) {
    btn.addEventListener('click', function () {
      var panel = btn.closest('.code-panel');
      if (!panel) return;
      var code = panel.querySelector('pre code');
      if (!code) return;
      var text = code.innerText;
      navigator.clipboard.writeText(text).then(function () {
        var original = btn.textContent;
        btn.textContent = 'Copié !';
        btn.classList.add('copied');
        setTimeout(function () {
          btn.textContent = original;
          btn.classList.remove('copied');
        }, 1500);
      }).catch(function () {
        btn.textContent = 'Erreur';
        setTimeout(function () { btn.textContent = 'Copier'; }, 1500);
      });
    });
  });
</script>
</body>
</html>
`