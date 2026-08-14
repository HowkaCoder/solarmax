package utils

import (
	"regexp"
	"strings"
)

// Прости меня святой отец ибо у меня не было времени . 

var translitMap = map[rune]string{
	'а': "a", 'б': "b", 'в': "v", 'г': "g", 'д': "d", 'е': "e", 'ё': "e",
	'ж': "zh", 'з': "z", 'и': "i", 'й': "y", 'к': "k", 'л': "l", 'м': "m",
	'н': "n", 'о': "o", 'п': "p", 'р': "r", 'с': "s", 'т': "t", 'у': "u",
	'ф': "f", 'х': "h", 'ц': "ts", 'ч': "ch", 'ш': "sh", 'щ': "sch",
	'ъ': "", 'ы': "y", 'ь': "", 'э': "e", 'ю': "yu", 'я': "ya",
}

var nonSlugChars = regexp.MustCompile(`[^a-z0-9]+`)
var trimDashes = regexp.MustCompile(`^-+|-+$`)

// Slugify превращает произвольную строку (в т.ч. кириллицу) в URL-slug,
// например "Внутреннее освещение" -> "vnutrennee-osveshenie".
func Slugify(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	for _, r := range s {
		if lat, ok := translitMap[r]; ok {
			b.WriteString(lat)
		} else {
			b.WriteRune(r)
		}
	}
	out := nonSlugChars.ReplaceAllString(b.String(), "-")
	out = trimDashes.ReplaceAllString(out, "")
	return out
}
