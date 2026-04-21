This comic page is in {{.LanguageName}}. All speech, thoughts, captions, and labels inside panels must use {{.ScriptName}} script only.

UNIQUE STORY PAGE {{.PageNum}} OF {{.TotalPages}} — render this page as a NEW story beat, not a repeat of any previous page.
The exact page-specific panel plan is the highest priority instruction:
{{.PanelLayout}}

CONTINUITY RULE:
  • This page must continue the story with different actions, settings, poses, camera angles, and foreground objects from all earlier pages.
  • If reference images are provided, use them only for character identity and costume consistency. Do not copy their composition, location, pose, lighting, or background.
  • Every panel on this page must clearly show a different moment in the sequence.

{{.Words}}
Style: {{.Style}}.{{.RenderingRequirement}}
Interior story page only. Do NOT render a cover masthead, title banner, subtitle, price box, issue number, date, page header, page footer, logo bug, or publisher badge.
{{if .Bible}}CHARACTER AND SETTING REFERENCE:
{{.Bible}}

{{end}}Each panel is separated by a thin black gutter line. All {{.PanelsPerPage}} panels must be clearly distinct scenes, not one continuous image. Fill the full image area with the {{.PanelsPerPage}} panels.
MANDATORY SPEECH BUBBLES — this is a comic; characters must speak:
  • At least {{.RequiredDialoguePanels}} of the {{.PanelsPerPage}} panels on this page must contain a speech bubble or thought bubble.
  • If the panel description includes quoted dialogue, render it exactly inside the bubble.
  • Speech bubbles have a white background, black outline, and a tail pointing to the speaker.
  • Thought bubbles use a cloud shape with small circles leading to the thinker.
  • All bubble text must use {{.LanguageName}} and {{.ScriptName}} script only.
VARIETY — every panel must differ from the others in at least 3 of these dimensions: camera angle, character pose/action, location/background detail, lighting/time of day, and foreground objects. Do not repeat the same composition.
{{.RenderingRequirement}}STRICT CONSISTENCY RULES — for every panel:
  • Human characters: identical face, apparent age, hair, and clothing compared to the reference.
  • Animals: identical species/breed, fur pattern, markings, and eye colour.
  • Clothing changes only if this page explicitly describes a change.
  • All visible text must use {{.LanguageName}} and {{.ScriptName}} script only.
{{if .RenderingRequirement}}{{.RenderingRequirementEnd}}{{end}}
