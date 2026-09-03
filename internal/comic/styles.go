package comic

// stylePrompt is a style line with its English prompt text and Cyrillic localization.
// Localization is attached here so edits cannot silently desync from localizedStylePrompt.
type stylePrompt struct {
	English   string
	Localized string
}

// defaultStyles holds fallback prompt pools keyed by style mode.
var defaultStyles = map[string][]stylePrompt{
	styleModeRealistic: {
		{
			English:   "ultra-realistic DSLR photography, cinematic 35mm lens, natural lighting, hyper-detailed textures",
			Localized: "ултрареалистична фотография с кинематографичен 35 мм обектив, естествено осветление и силно детайлни текстури",
		},
		{
			English:   "cinematic still photography, golden-hour lighting, shallow depth of field, photojournalism quality",
			Localized: "кинематографична фотография с светлина на златен час, плитка дълбочина на полето и качеството на фоторепортаж",
		},
		{
			English:   "hyper-realistic photography, studio-quality lighting, sharp focus, true-to-life colors and textures",
			Localized: "хиперреалистична фотография със студийно осветление, остър фокус и естествени цветове и текстури",
		},
	},
	styleModeComic: {
		{
			English:   "classic American comic book with bold ink outlines, halftone dots, and primary colors",
			Localized: "класически американски комикс с дебели мастилени контури, халф-тонови точки и основни цветове",
		},
		{
			English:   "retro 1960s pop art in the style of Roy Lichtenstein with thick outlines and Ben-Day dots",
			Localized: "ретро поп арт от 60-те години с дебели контури и точки в стил Бен-Дей",
		},
		{
			English:   "European bande dessinée with detailed backgrounds, clear lines, and rich flat colors",
			Localized: "европейски банд десине с детайлни фонове, ясни линии и наситени плоски цветове",
		},
		{
			English:   "noir black-and-white graphic novel with heavy shadows and high contrast",
			Localized: "ноар черно-бял графичен роман с тежки сенки и силен контраст",
		},
		{
			English:   "children's picture book with bright, friendly illustrations and thick outlines",
			Localized: "детска книжка с ярки, приветливи илюстрации и дебели контури",
		},
		{
			English:   "painterly oil-on-canvas comic with loose brushwork and vivid impressionist colors",
			Localized: "живописен комикс с маслени щрихи и ярки импресионистични цветове",
		},
	},
	styleModeCartoon: {
		{
			English:   "extremely cartoonish 1930s rubber-hose funny-animal comic style, original mouse-and-duck-era mascot energy, pie-cut eyes, button noses or beaks, white gloves, oversized shoes, noodle arms, squash-and-stretch bodies, round heads, huge expressions, flat candy colors, simple gag staging, absolutely no realism",
			Localized: "крайно карикатурен комикс стил от 30-те с гумени маркучи, енергия на оригинални маскоти от ерата на мишки и патици, очи като пайове, копчени носове или човки, бели ръкавици, огромни обувки, ръце като юфка, тела със смачкване и разтягане, кръгли глави, огромни изражения, плоски бонбонени цветове, проста постановка на гагове, абсолютно без реализъм",
		},
		{
			English:   "vintage theatrical funny-animal cartoon comic style, redesign people as original anthropomorphic mascot characters with beaks or round snouts, pie-cut eyes, white gloves, oversized shoes, elastic limbs, cheerful slapstick poses, bold clean outlines, bright flat backgrounds, no realistic anatomy",
			Localized: "винтажен театрален комикс стил с забавни животни, хора преработени като оригинални антропоморфни маскоти с човки или кръгли муцуни, очи като пайове, бели ръкавици, огромни обувки, еластични крайници, весели слепстик пози, смели чисти контури, ярки плоски фонове, без реалистична анатомия",
		},
		{
			English:   "old black-and-white short cartoon style translated into comic panels, original rubber-hose mascot characters, pie eyes, rubber limbs, bouncing motion arcs, simple rounded shapes, comic mischief, exaggerated face acting, hand-inked animation cels",
			Localized: "стар черно-бял кратък анимационен стил преведен в комикс панели, оригинални маскоти с гумени маркучи, очи като пайове, гумени крайници, подскачащи дъги на движение, прости заоблени форми, комикс пакости, преувеличена мимика, ръчно мастилени анимационни целулоиди",
		},
	},
	styleModeAction90s: {
		{
			English:   "1990s superhero comic splash-page style like a bold caped hero punching a masked villain in a city, huge muscles, spandex costumes, flowing cape, explosive impact burst, speed lines, smoke, rubble, halftone print texture, heavy black inks, saturated primary colors",
			Localized: "супергеройски комикс стил от 90-те като смел герой с пелерина удря маскиран злодей в град, огромни мускули, спандекс костюми, развяваща се пелерина, експлозивен удар, скоростни линии, дим, развалини, халф-тонова печатна текстура, тежки черни мастила, наситени основни цветове",
		},
		{
			English:   "classic 90s action superhero cover style, oversized yellow masthead energy, caped spandex hero, armored villain, clenched fists, dramatic foreshortening, city skyline, explosions, speech balloons, thick ink outlines, Ben-Day dots",
			Localized: "класическа екшън супергеройска корица от 90-те, енергия на огромно жълто заглавие, герой в спандекс с пелерина, брониран злодей, стиснати юмруци, драматично скъсяване, градски силует, експлозии, балончета с реплики, дебели мастилени контури, точки Бен-Дей",
		},
		{
			English:   "extreme 1990s superhero action comic style, heroic anatomy, masked villain, motion streaks, lightning impact shapes, cross-hatching, smoky city destruction, glossy saturated colors, bold comic-book lettering spaces",
			Localized: "екстремен супергеройски екшън комикс стил от 90-те, героична анатомия, маскиран злодей, ивици на движение, светкавични ударни форми, кръстосано щриховане, димно градско разрушение, лъскави наситени цветове, смели пространства за комикс надписи",
		},
	},
	styleModeManga: {
		{
			English:   "true monochrome Japanese shonen manga page, black ink only with grey screentone, no western color rendering, large expressive anime eyes, sharp spiky hair silhouettes, sweat drops, impact bursts, speed-line backgrounds, exaggerated emotion close-ups, dynamic diagonal manga panel language",
			Localized: "истинска монохромна японска шонен манга страница, само черно мастило със сив скрийнтон, без западно цветно рендиране, големи изразителни аниме очи, остри настръхнали силуети на коса, капки пот, ударни изблици, фонове със скоростни линии, преувеличени емоционални близки планове, динамичен диагонален език на манга панели",
		},
		{
			English:   "authentic serialized manga action art, crisp black linework, dense screentone gradients, white paper highlights, hand-drawn Japanese comic energy, chibi reaction inset where appropriate, dramatic sound-effect shapes, cinematic close-ups, no painterly western comic coloring",
			Localized: "автентично серийно манга екшън изкуство, ясни черни линии, плътни скрийнтон градиенти, бели хартиени светлини, ръчно рисувана японска комикс енергия, чиби реакционна вставка където е подходящо, драматични форми на звукови ефекти, кинематографични близки планове, без живописно западно комикс оцветяване",
		},
		{
			English:   "modern black-and-white manga fantasy style, anime character proportions, big eyes, angular hair, thin-to-thick ink strokes, patterned screentones, motion streaks, expressive face acting, explosive action frames, minimal color accents only if unavoidable",
			Localized: "модерен черно-бял манга фентъзи стил, аниме пропорции на герои, големи очи, ъгловата коса, тънки до дебели мастилени щрихи, шарени скрийнтони, ивици на движение, изразителна мимика, експлозивни екшън кадри, минимални цветни акценти само ако са неизбежни",
		},
	},
	styleModeHorror: {
		{
			English:   "vintage horror comic style, eerie shadows, gothic atmosphere, unsettling monsters, dramatic candlelit contrast, suspenseful panel staging",
			Localized: "винтажен хорър комикс стил, зловещи сенки, готическа атмосфера, обезпокоителни чудовища, драматичен контраст от свещи, напрегната постановка на панели",
		},
		{
			English:   "creepy supernatural horror comic style, fog, haunted architecture, anxious expressions, heavy inks, sickly green and crimson accents",
			Localized: "страшен свръхестествен хорър комикс стил, мъгла, обитавана архитектура, тревожни изражения, тежки мастила, болезнено зелени и пурпурни акценти",
		},
		{
			English:   "old anthology horror comic style, graveyard mist, warped shadows, tense faces, scratchy ink texture, ominous colored lighting",
			Localized: "стар антологичен хорър комикс стил, гробищна мъгла, изкривени сенки, напрегнати лица, надраскана мастилена текстура, зловещо цветно осветление",
		},
	},
	styleModeWatercolor: {
		{
			English:   "pure transparent watercolor storybook comic style, visible cold-press paper grain, watery pigment blooms, uneven wash edges, granulation, soft bleeding colors, pale layered washes, loose pencil underdrawing, no digital airbrush, no oil-paint texture, no glossy comic coloring",
			Localized: "чист прозрачен акварелен приказен комикс стил, видимо зърно на студено пресована хартия, воднисти пигментни цветове, неравни краища на мивки, гранулация, меки разливащи се цветове, бледи наслоени мивки, свободна моливна подрисунка, без цифров аерограф, без маслена текстура, без лъскаво комикс оцветяване",
		},
		{
			English:   "hand-painted watercolor children's adventure style, translucent pastel washes, wet-on-wet skies, dry-brush texture, white paper highlights, delicate ink-and-pencil outlines, puddled pigment edges, airy negative space, gentle low-contrast rendering",
			Localized: "ръчно рисуван акварелен детски приключенски стил, полупрозрачни пастелни мивки, мокро върху мокро небе, текстура на суха четка, бели хартиени светлини, деликатни мастилено-моливни контури, локвички от пигмент по краищата, ефирно негативно пространство, нежно нискоконтрастно рендиране",
		},
		{
			English:   "storybook watercolor fantasy comic style, washed paper texture, transparent layered color, soft backruns and blossoms, feathered brush edges, muted handmade palette, tender expressions, dreamy illustrated settings, visibly painted on paper",
			Localized: "приказен акварелен фентъзи комикс стил, измита хартиена текстура, прозрачен наслоен цвят, меки обратни тичания и цъфтежи, перести ръбове на четката, приглушена ръчна палитра, нежни изражения, мечтателни илюстрирани декори, видимо рисувано върху хартия",
		},
	},
	styleModePulp: {
		{
			English:   "retro pulp adventure cover style, painted poster energy, dramatic rim light, saturated inks, worn paper grain, bold composition, vintage print imperfections",
			Localized: "ретро пулп приключенски коричен стил, енергия на рисуван плакат, драматична контурна светлина, наситени мастила, износено хартиено зърно, смела композиция, винтажни печатни несъвършенства",
		},
		{
			English:   "mid-century pulp sci-fi comic style, chromatic airbrushed highlights, atomic-age props, bold diagonals, distressed halftone texture, high adventure mood",
			Localized: "пулп научнофантастичен комикс стил от средата на века, цветни аерографски светлини, реквизит от атомната ера, смели диагонали, износена халф-тонова текстура, настроение на голямо приключение",
		},
	},
	styleModeMecha: {
		{
			English:   "high-detail mecha comic style, hard-surface armor plates, mechanical joints, cockpit glass reflections, technical line art, industrial lighting",
			Localized: "високодетайлен меха комикс стил, твърди броневи плочи, механични стави, отражения на кабинково стъкло, технически линеен рисунък, индустриално осветление",
		},
		{
			English:   "anime mecha action page style, towering robots, dynamic perspective, panel-like motion energy, sparks and smoke, crisp cel-shaded rendering",
			Localized: "аниме меха екшън страничен стил, високи роботи, динамична перспектива, панелна енергия на движение, искри и дим, ясно цел-сенчесто рендиране",
		},
	},
	styleModePixelArt: {
		{
			English:   "16-bit pixel-art comic scene, limited palette, clean pixel clusters, deliberate dithering, retro game-era perspective, sharp pixel edges",
			Localized: "16-битова пиксел-арт комикс сцена, ограничена палитра, чисти пикселни клъстери, нарочен дидеринг, ретро игрова перспектива, остри пикселни ръбове",
		},
		{
			English:   "retro pixel-art storyboard style, chunky sprites, high-contrast palette ramps, tiled-environment texture, no anti-aliasing, classic console look",
			Localized: "ретро пиксел-арт сториборд стил, едри спрайтове, висококонтрастни палитрени рампи, текстура на плочков декор, без заглаждане, класически конзолен вид",
		},
	},
	styleModeInkWash: {
		{
			English:   "ink-wash brush painting style, sumi-e inspired strokes, expressive black ink gradients, controlled bleed on paper, minimal but dramatic composition",
			Localized: "мастилено-мивков стил с четка, суми-е вдъхновени щрихи, изразителни черни мастилени градиенти, контролирано разливане върху хартия, минимална но драматична композиция",
		},
		{
			English:   "monochrome ink-and-wash comic style, calligraphic brush lines, soft gray washes, negative-space storytelling, textured rice-paper feel",
			Localized: "монохромен мастилено-мивков комикс стил, калиграфски линии с четка, меки сиви мивки, разказ чрез негативно пространство, усещане за текстурирана оризова хартия",
		},
	},
	styleModeClay: {
		{
			English:   "stop-motion clay diorama style, hand-sculpted clay characters, thumbprint texture, miniature set lighting, tactile handcrafted look",
			Localized: "стоп-моушън глинен диорамен стил, ръчно скулптурирани глинени герои, текстура от отпечатъци на палец, осветение на миниатюрен декор, тактилен ръчно изработен вид",
		},
		{
			English:   "plasticine comic tableau style, sculpted figures, soft studio shadows, practical miniature props, playful clay surface detail",
			Localized: "пластилинов комикс табло стил, скулптурирани фигури, меки студийни сенки, практични миниатюрни реквизити, игриви детайли на глинената повърхност",
		},
	},
}

// cyrillicStyleByEnglish maps every built-in English style line to its Localized form.
var cyrillicStyleByEnglish = buildCyrillicStyleByEnglish()

// DefaultStyleLines returns English-only style pools for configuration defaults.
func DefaultStyleLines() map[string][]string {
	out := make(map[string][]string, len(defaultStyles))
	for mode, prompts := range defaultStyles {
		lines := make([]string, len(prompts))
		for i, p := range prompts {
			lines[i] = p.English
		}
		out[mode] = lines
	}
	return out
}

func buildCyrillicStyleByEnglish() map[string]string {
	out := make(map[string]string)
	for mode, prompts := range defaultStyles {
		for _, p := range prompts {
			if p.English == "" {
				panic("comic: empty English style in mode " + mode)
			}
			if p.Localized == "" {
				panic("comic: missing Cyrillic localization for style " + p.English)
			}
			if containsLatinLetters(p.Localized) {
				panic("comic: Cyrillic localization contains Latin letters for style " + p.English)
			}
			if _, dup := out[p.English]; dup {
				panic("comic: duplicate English style " + p.English)
			}
			out[p.English] = p.Localized
		}
	}
	return out
}

func englishStyleLines(prompts []stylePrompt) []string {
	if len(prompts) == 0 {
		return nil
	}
	lines := make([]string, len(prompts))
	for i, p := range prompts {
		lines[i] = p.English
	}
	return lines
}
