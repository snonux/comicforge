{{if .Language}}MANDATORY LANGUAGE RULE: This is a {{.Language}} comic book. Every word of text inside speech bubbles, thought bubbles, caption boxes, and panel labels MUST be written in {{.Script}} script (for example: "Hello! What are you doing? Hurry up!"). English text anywhere in the panels is STRICTLY FORBIDDEN — use ONLY {{.Language}}.

{{end}}{{.Words}}
Art style: {{.Style}}.{{.RenderingRequirement}}
Comic book story page {{.PageNum}} of {{.TotalPages}}.
{{if .Bible}}CHARACTER & SETTING REFERENCE (story page {{.PageNum}} of {{.TotalPages}} — follow exactly, do NOT change clothing):
{{.Bible}}

{{end}}{{.PanelLayout}}Each panel is separated by a thin black gutter line. All 4 panels must be clearly distinct scenes — NOT one continuous image. The full image area must be covered by the 4 panels with no empty space.
MANDATORY SPEECH BUBBLES — this is a comic book; characters MUST speak:
  • At least 3 of the 4 panels MUST contain a speech bubble or thought bubble.
  • If the panel description includes quoted dialogue, render it EXACTLY inside a speech bubble.
  • Speech bubbles have a white background, black outline, and a tail pointing to the speaker.
  • Thought bubbles use a cloud shape with small circles leading to the thinker.
  • ALL bubble text is in {{.Script}} script — never Roman letters.
VARIETY MANDATE — every panel MUST differ from the others in at least 3 of these dimensions: camera angle (close-up, medium shot, wide shot, bird's-eye, low angle), character pose or action, location or background detail, lighting or time-of-day, and foreground objects. Repeating the same angle or composition across panels is FORBIDDEN.
{{.RenderingRequirement}}STRICT CONSISTENCY RULES — apply to every single panel:
  • Human characters: identical face, AGE APPEARANCE, hair colour/style, and clothing to the reference — a child must never look older or younger as defined.
  • Animal characters: identical breed, fur colour/pattern, markings, and eye colour — NEVER substitute a different animal or a generic version of the species.
  • Clothing changes only if this page's description explicitly describes a change.
  • LANGUAGE: all speech, thought, and caption text — {{.Script}} ONLY.
{{if .RenderingRequirement}}{{.RenderingRequirementEnd}}{{end}}
