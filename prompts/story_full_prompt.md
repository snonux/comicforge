Write a story of about 250 words in {{.LanguageName}} that naturally uses all of the following words.
The story must be {{.Genre}}; do not write a generic fairy tale.
Use {{.ScriptName}} script only and do not mix in another language or script.
Use every word naturally, but do not repeat the word list, bullet points, labels, or prompt instructions in the story.

Words to include:
{{.Words}}

After the story text, write exactly this separator line by itself:
{{.StoryBibleSeparator}}

Then write a CHARACTER CONSISTENCY GUIDE in {{.LanguageName}} using {{.ScriptName}} script for an illustrator.
IMPORTANT: all human characters must be adults (18+). Do not describe anyone as a child, teenager, or minor.
For every named HUMAN character: name, approximate adult age, hair (colour + style), eye colour, skin tone, build, and EXACT clothing (garment, colour, pattern, fit). Apparent age and clothing must stay the same for all appearances.
For every named ANIMAL character: name, species, exact breed, fur/feather/scale colour and pattern, eye colour, size, distinctive markings, and typical body posture. The animal must look IDENTICAL everywhere.
Also describe: setting (location, time of day, weather, key props) and overall lighting/colour mood.
Maximum 280 words for the guide. No headers, just dense descriptive prose.

After the guide, write exactly this separator line by itself:
{{.StoryTitleSeparator}}

Then write a short comic title in {{.LanguageName}} using {{.ScriptName}} script: 2-4 words that capture the story's theme and characters. Output only the title: no quotes, punctuation, or explanation.

After the title, write exactly this separator line by itself:
{{.StoryPanelSeparator}}

Write exactly {{.TotalPanels}} visual panel descriptions for the comic illustrator, one per line, in strict chronological story order.
Each line must use exactly this format: P{page}-{panel}: {description}
where page is 1 through {{.StoryPages}}, and panel is one of {{.PanelLabelsText}}.
Each description must be 1-2 sentences and include who is in the panel, what they do, where they are, their expression or body language, and, for most panels, exact dialogue or thought text in {{.LanguageName}} inside quotation marks.
At least {{.RequiredDialoguePanels}} of the {{.PanelsPerPage}} panels on every page MUST include dialogue or thought text in {{.LanguageName}} for a speech or thought bubble. Dialogue must come from the story and use {{.ScriptName}} script.
The {{.TotalPanels}} panels must retell the story from beginning to end; each page covers one stage and each panel moves the action forward.
Do not repeat the same scene or camera angle in consecutive panels.
