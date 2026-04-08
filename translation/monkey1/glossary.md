# Monkey Island 1 Swedish Translation Glossary

## Translation Philosophy

**General rule**: Translate descriptive names. Keep only non-descriptive proper names in English.

**Exceptions**: Names that appear in hardcoded graphics stay in English in the graphics, but translate in dialogue/text:
- LUCASFILM GAMES - keep as-is everywhere
- Mêlée Island - keep as-is everywhere
- SCUMM BAR - keep as-is everywhere
- STAN'S PREVIOUSLY OWNED VESSELS - graphic stays English, but dialogue uses **"Stans Begagnade Fartyg"**

## Character Names - Keep in English

- **Guybrush Threepwood** - protagonist
- **Elaine Marley** - Governor
- **LeChuck** - ghost pirate captain
- **Herman Toothrot** - castaway
- **Carla** - the Sword Master
- **Fester Shinetop** - sheriff
- **Stan** - used ship salesman
- **Captain Smirk** - sword fighting trainer
- **Otis** - prisoner
- **Meathook** - tough pirate with tattoo (descriptive nickname, but established as his name)
- **Mancomb Seepgood** - pirate in bar

### Secondary Characters
- **Fettucini Brothers** → **Fettucini Bröderna** - circus performers (Alfredo and Bill)

### Place Names (from graphics - keep English)
- **Mêlée Island** (with circumflex on first e) - appears in graphics
- **Monkey Island** - game title, keep as is
- **Scumm Bar** - appears in graphics (meta-joke about SCUMM engine)

### Business/Location Names - Translate Descriptions
- **Stan's Previously Owned Vessels** → **Stans Begagnade Fartyg** 
- **Captain Smirk's Big Body Pirate Gym** → **Kapten Smirks Stora Piratgym**

### Ship Names - Translate (Descriptive)
- **Sea Monkey** → **Havsapan** or **Sjöapan**
- **Ghost Ship** → **Spökskepp**

### Island Geography - Translate (Descriptive)
- **Lookout Point** → **Utkiksposten**
- **Governor's Mansion** → **Guvernörens herrgård** or **Guvernörens villa**

### Brand References

**Real game references (keep English):**
- **Monkey Island®** - registered trademark (preserve \015 code)
- **LOOM®** - reference to another LucasArts game

**Fictional in-game brands (translate the joke):**
- **Davey Jones® Lockers** → **Davy Jones® Skåp** (pun on "Davy Jones' Locker" = bottom of the sea. In Swedish: "Davy Jones' skåp")
- **GRIPMASTER®** → **GREPPMASTERN®** (fictional brand for handles/grip strengtheners)
- **BREATHMASTER®** → **ANDEMASTERN®** (fictional brand for breath fresheners, "för piraten som bryr sig om första intrycket")

## Common Recurring Terms

### Pirate/Nautical Terms - TRANSLATE
- pirate → pirat
- ship → skepp
- crew → besättning
- captain → kapten
- sail → segla
- treasure → skatt
- sword → svärd
- island → ö
- sea → hav/havet
- dock → kaj/hamn
- cabin → hytt

### Pirate Ranks/Types - TRANSLATE
- buccaneer → sjörövare
- swashbuckler → äventyrare
- scallywag → skurk
- scurvy dog → skabbiga hund

### Recurring Character Descriptions - TRANSLATE
- **Important-looking pirates** → viktiga pirater / imponerande pirater (pirate leaders in Scumm Bar)
- **Men of Low Moral Fiber** → Män av Tvivelaktig Moral / Män med Låg Moral (street corner pirates - this is a humorous English idiom, translate the intent)

### Game-specific Terms - TRANSLATE

- trial/test → prov/prövning
- quest → uppdrag
- riddle → gåta
- clue → ledtråd
- "pieces of eight" → åttaöring/åttaöringar (pirate currency)
- "Jolly Roger" → Jolly Roger (historical pirate flag name, can keep or translate as "piratflaggan")

### Object Names - TRANSLATE

- **Idol of Many Hands** → **Idolen med Många Händer**
- **rubber chicken with a pulley in the middle** → **gummikyckling med en trissa i mitten**
- T-shirt → T-shirt (modern item, international word)

**Translate normally:**
- sword → svärd
- map → karta
- key → nyckel
- compass → kompass
- shovel → spade
- rope → rep
- bottle → flaska
- root → rot (voodoo root)
- grog → grogg (Swedish has this word!)

### Food/Drink
- grog → grogg (keep similar)
- root beer → rotöl
- stew → gryta
- meat → kött
- fish → fisk

### Voodoo/Magic Terms
- voodoo → voodoo (international term)
- spell → besvärjelse/trollformel
- magic → magi
- ghost → spöke/ande
- curse → förbannelse

## Special Cases

### "SCUMM" References
The word "SCUMM" appears in:
- "Scumm Bar" - keep as is (it's the game engine name)
- Mentions of "scumm" as ingredient in grog - keep for consistency

### Humor Preservation
Certain names are jokes that work in English:
- **Men of Low Moral Fiber** - translate concept, not literal words → "Pirater med Låg Moral"
- **Fettucini Brothers** - Italian names, keep
- **Previously Owned Vessels** - euphemism for "used ships", translate the euphemism

### Control Codes (NEVER TRANSLATE)
- `\255\003` - pause/line break
- `\255\006\NNN\000` - variable substitution  
- `\015` - registered trademark ®
- `\250` - non-breaking space

### Dialog Attribution Codes (DO NOT TRANSLATE)
- `(D8)` - Guybrush dialog
- `(14)` - NPC dialog
- `(94)` - Various
- `(FA)` - Player choice
- `(13)` - Name display
- Other hex codes

## Consistency Notes

### Titles and Honorifics
- Governor → Guvernör
- Captain → Kapten
- Master (as in Sword Master) → Mästare
- Sheriff → Sheriff (keep English, or use "Länsman")

### Time References
- hours → timmar
- days → dagar  
- minutes → minuter
- "Meanwhile..." → "Under tiden..."

### Measurements
Keep pirate-era measurements:
- pieces of eight (currency)
- feet (measurement) → fot

## Style Guidelines

1. **Pirate Speech**: Swedish doesn't have exact equivalents for pirate dialect, but use slightly archaic/formal Swedish for pirate leaders, more casual for regular pirates

2. **Guybrush's Voice**: Modern, slightly sarcastic, often self-deprecating

3. **Formality Levels**:
   - Governor Marley: Formal/educated Swedish
   - Pirates: Casual, some slang
   - Herman Toothrot: Rambling, slightly crazy
   - Stan: Fast-talking salesman

4. **Humor**: Prioritize funny over literal. If a joke doesn't work, find Swedish equivalent.

## Swedish Character Encoding
The TRANSLATE_TABLE reference file shows these mappings:
- å, ä, ö (Swedish characters are supported)
- Use proper Swedish spelling throughout

## Notes on Object Name Consistency
When an object appears as both OBNA (object name) and in dialog:
- **fabulous idol** / **Idol of Many Hands** - same object, maintain consistency
- **rubber chicken with a pulley in the middle** - iconic phrase, translate but keep recognizable
- Ship names in object vs. dialog must match
