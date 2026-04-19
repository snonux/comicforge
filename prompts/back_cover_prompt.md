{{if .Language}}MANDATORY LANGUAGE RULE: This is a {{.Language}} comic book. All visible text (blurb box, labels, banners) MUST be in {{.Script}} script. English text anywhere on the back cover is STRICTLY FORBIDDEN.

{{end}}Art style: {{.Style}}.{{.RenderingRequirement}}
TRADITIONAL COMIC BOOK BACK COVER — {{if .RenderingRequirement}}single full-bleed image (photoreal — like a physical back-cover photo shoot){{else}}single full-bleed illustration{{end}}, landscape 16:9 format.
NO panel grid. NO speech bubbles.
{{if .Bible}}CHARACTER & SETTING REFERENCE (back cover — follow exactly, do NOT change clothing):
{{.Bible}}

{{end}}Layout rules (must follow exactly):
  • MAIN ART: a calm, warm, resolved scene filling the upper 60% of the cover — EXACTLY the named characters from the story (as described in the reference above) in a peaceful or triumphant ending moment, with the full story setting behind them. The scene must look PHOTOGRAPHED (live-action), not drawn. Do NOT invent new characters or use generic stand-ins.
  • BLURB BOX: {{.BlurbBox}}
  • BOTTOM STRIP: barcode box bottom-left (black-and-white barcode graphic), series title '{{.SeriesTitle}}' bottom-right — classic comic book back-cover production design.
IMPORTANT: only the characters named in the reference may appear on this back cover. Same age, same face, same clothing, same animals as in the interior pages. LANGUAGE REMINDER: all text in {{.Script}} script — see rule at top. Story ending hint:

{{.StoryText}}{{if .RenderingRequirement}}{{.RenderingRequirementEnd}}{{end}}
