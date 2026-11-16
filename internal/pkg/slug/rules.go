package goslug

// SymbolReplacements maps common symbols into readable English words.
var SymbolReplacements = map[rune]string{
	'&': "and",
	'@': "at",
	'₹': "inr",
	'$': "usd",
	'€': "eur",
	'£': "gbp",
	'%': "percent",
	'+': "plus",
	'#': "hash",
	'=': "equals",
}

// DiacriticReplacements maps European accented Latin characters to basic ASCII.
var DiacriticReplacements = map[rune]rune{
	'À': 'A', 'Á': 'A', 'Â': 'A', 'Ã': 'A', 'Ä': 'A', 'Å': 'A', 'à': 'a', 'á': 'a', 'â': 'a', 'ã': 'a', 'ä': 'a', 'å': 'a',
	'È': 'E', 'É': 'E', 'Ê': 'E', 'Ë': 'E', 'è': 'e', 'é': 'e', 'ê': 'e', 'ë': 'e',
	'Ì': 'I', 'Í': 'I', 'Î': 'I', 'Ï': 'I', 'ì': 'i', 'í': 'i', 'î': 'i', 'ï': 'i',
	'Ò': 'O', 'Ó': 'O', 'Ô': 'O', 'Õ': 'O', 'Ö': 'O', 'Ø': 'O', 'ò': 'o', 'ó': 'o', 'ô': 'o', 'õ': 'o', 'ö': 'o', 'ø': 'o',
	'Ù': 'U', 'Ú': 'U', 'Û': 'U', 'Ü': 'U', 'ù': 'u', 'ú': 'u', 'û': 'u', 'ü': 'u',
	'Ý': 'Y', 'ý': 'y', 'ÿ': 'y',
	'Ç': 'C', 'ç': 'c',
	'Ñ': 'N', 'ñ': 'n',
	'ß': 's',
}

// DevanagariReplacements maps Hindi/Devanagari characters into Romanized transliterations.
var DevanagariReplacements = map[rune]string{
	// Vowels
	'अ': "a", 'आ': "aa", 'इ': "i", 'ई': "ee", 'उ': "u", 'ऊ': "oo", 'ऋ': "ri",
	'ए': "e", 'ऐ': "ai", 'ओ': "o", 'औ': "au",
	// Matras (vowel signs) & Marks
	'ा': "a", 'ि': "i", 'ी': "i", 'ु': "u", 'ू': "u", 'ृ': "ri",
	'े': "e", 'ै': "ai", 'ो': "o", 'ौ': "au", 'ं': "n", 'ः': "h", 'ँ': "n", '्': "", '़': "",
	'ॐ': "om",
	// Consonants
	'क': "k", 'ख': "kh", 'ग': "g", 'घ': "gh", 'ङ': "ng",
	'च': "ch", 'छ': "chh", 'ज': "j", 'झ': "jh", 'ञ': "ny",
	'ट': "t", 'ठ': "th", 'ड': "d", 'ढ': "dh", 'ण': "n",
	'त': "t", 'थ': "th", 'द': "d", 'ध': "dh", 'न': "n",
	'प': "p", 'फ': "ph", 'ब': "b", 'भ': "bh", 'म': "m",
	'य': "y", 'र': "r", 'ल': "l", 'व': "v", 'श': "sh", 'ष': "sh", 'स': "s", 'ह': "h",
}
