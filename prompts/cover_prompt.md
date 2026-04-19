{{if .Language}}MANDATORY LANGUAGE RULE: This is a {{.Language}} comic book. All text on the cover (cover lines, banners, labels) MUST be in {{.Script}} script. The masthead title must also be rendered in a striking comic-book font.

{{end}}Art style: {{.Style}}.{{.RenderingRequirement}}
TRADITIONAL COMIC BOOK FRONT COVER — {{if .RenderingRequirement}}single full-bleed image (photoreal — like a physical comic book cover photo shoot){{else}}single full-bleed illustration{{end}}, landscape 16:9 format.
NO panel grid. NO speech bubbles.
{{if .Bible}}CHARACTER & SETTING REFERENCE (cover — follow exactly, do NOT change clothing):
{{.Bible}}

{{end}}MANDATORY MASTHEAD — the most important visual element on this cover:
  • Invent a DRAMATIC, STORY-SPECIFIC comic book title that fits the characters and theme of the story teaser below (e.g. for a space story: 'ГАЛАКТИЧЕСКИ ГЕРОИ', for a mystery: 'ТАЙНАТА НА ГОРАТА'). The title must be in HUGE, dominant lettering across the very top of the cover — bold comic-book masthead font, thick outlines, bright contrasting colours (yellow, red, or white on dark), taking up the top 20% of the image. This title MUST be legible and unmissable.
  • Directly below the main title, add a smaller subtitle banner: '{{.Subtitle}}' in a contrasting accent colour.
  • Add a bold comic-book LOGO BUG (small circular or star-shaped badge) in the top-left corner — e.g. a planet, rocket, magnifying glass, sword — matching the story theme. The logo should feel like a real publisher imprint.
Remaining layout rules:
  • MAIN ART: below the masthead, a dramatic {{if .RenderingRequirement}}photographed cinematic scene{{else}}illustration{{end}} of EXACTLY the named characters from the story (as described in the reference above) — same faces, same ages, same clothing, same animals. Do NOT invent new characters or use generic stand-ins.
  • COVER LINES: 2–3 short {{.Language}} teaser phrases in bold display type (e.g. 'A GREAT ADVENTURE!' or 'THE MYSTERY UNFOLDS!')
  • BOTTOM STRIP: price box bottom-left, issue number bottom-right — classic Silver-Age / Bronze-Age comic production design.
IMPORTANT: only the characters named in the reference may appear on this cover. Same age, same face, same clothing as in the interior pages. LANGUAGE REMINDER: all cover text in {{.Script}} script — see rule at top. Story teaser:

{{.StoryText}}{{if .RenderingRequirement}}{{.RenderingRequirementEnd}}{{end}}
