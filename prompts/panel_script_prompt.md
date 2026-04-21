Write exactly {{.TotalPanels}} visual panel descriptions for the comic illustrator, one per line, in strict chronological story order.
Each line must use exactly this format: P{page}-{panel}: {description}
where page is 1 through {{.StoryPages}}, and panel is one of {{.PanelLabelsText}}.
Each description must be 1-2 sentences and include who is in the panel, what they do, where they are, their expression or body language, and, for most panels, exact dialogue or thought text in {{.LanguageName}} inside quotation marks.
At least {{.RequiredDialoguePanels}} of the {{.PanelsPerPage}} panels on every page MUST include dialogue or thought text in {{.LanguageName}} for a speech or thought bubble. Dialogue must come directly from the story and use {{.ScriptName}} script.
The {{.TotalPanels}} panels must retell the story from beginning to end; each page covers one stage and each panel moves the action forward.
Do not repeat the same scene or camera angle in consecutive panels.
