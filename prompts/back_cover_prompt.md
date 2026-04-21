This is a {{.LanguageName}} comic. All visible back-cover text, banners, boxes, and labels must use {{.ScriptName}} script only.

Style: {{.Style}}.{{.RenderingRequirement}}
Full back cover {{if .RenderingRequirement}}photographic scene{{else}}illustration{{end}} in landscape 16:9 format.
No panel grid. No speech bubbles.
{{if .Bible}}CHARACTER AND SETTING REFERENCE:
{{.Bible}}

{{end}}Composition rules:
  • MAIN ART: a calm, warm, resolved image in the upper 60% of the cover — EXACTLY the characters described in the reference in a peaceful or triumphant final moment, with the full setting behind them.
  • TEXT BOX: {{.BlurbBox}}
  • BOTTOM STRIP: barcode box at bottom-left, series title "{{.SeriesTitle}}" at bottom-right — classic comic back-cover layout.
IMPORTANT: only characters named in the reference may appear. Same age, face, clothing, and animal companions as in the interior pages. All visible text must use {{.LanguageName}} and {{.ScriptName}} script only. Story ending hint:

{{.StoryText}}{{if .RenderingRequirement}}{{.RenderingRequirementEnd}}{{end}}
