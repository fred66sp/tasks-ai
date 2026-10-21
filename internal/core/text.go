package core

import (
	"regexp"
	"strings"
)

// The fields whose text the check reads, named the way the product names them in
// the message of a rejection.
const (
	fieldTitle       = "title"
	fieldDescription = "description"
	fieldNotes       = "notes"
	fieldComment     = "comment"
)

// Las listas cerradas del contrato, leídas sobre el texto tal como se escribe y
// sin distinguir mayúsculas. La del hash de commit va partida, porque el
// contrato la escribe con una aserción que el motor de expresiones de Go no
// tiene —no mira hacia delante—: aquí se busca la tirada hexadecimal y la letra
// se comprueba después.
var (
	hexRunPattern   = regexp.MustCompile(`(?i)\b[0-9a-f]{7,40}\b`)
	branchPattern   = regexp.MustCompile(`(?i)\b(?:feat|feature|fix|hotfix|release|refactor|chore)/[a-z0-9][a-z0-9._-]*`)
	remotePattern   = regexp.MustCompile(`(?i)\borigin/[a-z0-9][a-z0-9._-]*`)
	statePattern    = regexp.MustCompile(`(?i)(?:^|[.:;!?\n]\s*)((?:estado|hecho|hecha|terminad[oa]|cerrad[oa]|abiert[oa]|reabiert[oa]|en curso|pendiente|bloquead[oa]|a la espera|siguiente paso|próximo paso|status|done|pending|in progress|next step)\s*:)`)
	deliveryPattern = regexp.MustCompile(`(?i)\b(?:sin commitear|sin subir|sin fusionar|pendiente de commit|pendiente de push)\b`)
)

// textRule is one of the lists of the contract: what finds the fragment that
// writes the state of Git into a text, and the words the rejection adds after
// naming the field and the fragment. Se leen en el orden del contrato, así que
// un texto que cae en más de una lista se nombra por la primera.
type textRule struct {
	// find devuelve el fragmento del texto que cae en la lista, vacío cuando no
	// cae en ella.
	find func(text string) string
	// verb is the verb of the message in the form the plural subject takes
	// —«the notes name»—: el sujeto en singular le añade la ese.
	verb string
	// what says what the fragment is, and remedy what to write instead.
	what   string
	remedy string
}

// textRules are the lists `R1` to `R5` of the contract, en su orden.
var textRules = []textRule{
	{
		find:   commitHash,
		verb:   "name",
		what:   "a commit",
		remedy: "a commit reaches its task through the commit message, with Refs: tk-N",
	},
	{
		find:   branchPattern.FindString,
		verb:   "name",
		what:   "a branch",
		remedy: "the board does not store Git",
	},
	{
		find:   remotePattern.FindString,
		verb:   "name",
		what:   "a remote reference",
		remedy: "the board does not store Git",
	},
	{
		find:   stateMarker,
		verb:   "declare",
		what:   "the state of the task",
		remedy: "state what was done, found or decided instead",
	},
	{
		find:   deliveryPattern.FindString,
		verb:   "declare",
		what:   "the state of the delivery",
		remedy: "the board does not store Git",
	},
}

// stateMarker finds a declaration of the state of the task: el marcador con sus
// dos puntos, sin la puntuación que lo precede. La lista solo lo reconoce al
// principio del texto o detrás de un signo de puntuación, porque las mismas
// palabras describen lo que el producto hace.
func stateMarker(text string) string {
	found := statePattern.FindStringSubmatch(text)
	if len(found) < 2 {
		return ""
	}

	return strings.TrimSpace(found[1])
}

// checkText refuses a text that writes the state of Git or the state of the task
// into the field named, venga de la superficie que venga: el título, la
// descripción, las notas y el cuerpo de un comentario. Rechazarlo es no escribir
// nada.
func checkText(field, text string) error {
	for _, rule := range textRules {
		fragment := rule.find(text)
		if fragment == "" {
			continue
		}

		return invalid("the %s %s %s: %q; %s", field, rule.agrees(field), rule.what, fragment, rule.remedy)
	}

	return nil
}

// agrees returns the verb with the subject the field makes: el producto nombra
// las notas en plural y el resto en singular.
func (r textRule) agrees(field string) string {
	if field == fieldNotes {
		return r.verb
	}

	return r.verb + "s"
}

// commitHash finds a commit hash: seven to forty hexadecimal characters with at
// least one letter.
func commitHash(text string) string {
	for _, fragment := range hexRunPattern.FindAllString(text, -1) {
		if strings.ContainsAny(fragment, "abcdefABCDEF") {
			return fragment
		}
	}

	return ""
}
