package core

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
)

// The fields whose text the check reads, named the way the product names them in
// the message of a rejection.
const (
	fieldTitle       = "title"
	fieldDescription = "description"
	fieldNotes       = "notes"
	fieldComment     = "comment"

	// These name the rest of the entry fields the rules speak of. The message
	// of a rule keeps the label it reads with; the field name travels with the
	// error so a surface that can repair the entry cites the field, not the
	// text.
	fieldPriority = "priority"
	fieldReason   = "reason"
	fieldStatus   = "status"
	fieldTags     = "tags"
	fieldParent   = "parent"
	fieldBlocker  = "blocker"
	fieldID       = "id"
	fieldBody     = "body"
	fieldRef      = "ref"
)

// Las listas cerradas del contrato, leídas sobre el texto tal como se escribe y
// sin distinguir mayúsculas. La del hash de commit va partida, porque el
// contrato la escribe con una aserción que el motor de expresiones de Go no
// tiene —no mira hacia delante—: aquí se busca la tirada hexadecimal y la letra
// se comprueba después.
var (
	hexRunPattern   = regexp.MustCompile(`(?i)\b[0-9a-f]{7,40}\b`)
	branchPattern   = regexp.MustCompile(`(?i)\b(?:feat|feature|fix|hotfix|release|refactor|chore|docs)/[a-z0-9][a-z0-9._-]*`)
	remotePattern   = regexp.MustCompile(`(?i)\borigin/[a-z0-9][a-z0-9._-]*`)
	statePattern    = regexp.MustCompile(`(?i)(?:^|[.:;!?\n]\s*)((?:estado|hecho|hecha|terminad[oa]|cerrad[oa]|abiert[oa]|reabiert[oa]|en curso|pendiente|bloquead[oa]|a la espera|siguiente paso|próximo paso|status|done|pending|in progress|next step)\s*:)`)
	deliveryPattern = regexp.MustCompile(`(?i)\b(?:sin commitear|sin subir|sin fusionar|pendientes? de commit|pendientes? de push)\b`)

	// Una referencia de Git con nombre propio y un verbo de Git en la misma
	// cláusula: «mergeado a main» afirma el estado de Git sin escribir una rama.
	// Las terminaciones de «subir» e «integrar» van escritas una a una: abrir la
	// raíz casaría «subasta» e «integridad», que son prosa legítima.
	gitRefPattern  = regexp.MustCompile(`(?i)\b(?:main|master|HEAD)\b`)
	gitVerbPattern = regexp.MustCompile(`(?i)\b(?:merge[a-z]*|push[a-z]*|commit[a-z]*|checkout|rebas[a-z]*|fusion[a-z]*` +
		`|integr(?:ar|o|as|a|amos|áis|an|é|aste|ó|asteis|aron|e|es|emos|éis|en|ara|aras|áramos|arais|aran` +
		`|ase|ases|ásemos|aseis|asen|are|ares|áremos|areis|aren|ad|á|ando|ado|ada|ados|adas|ación)` +
		`|sub(?:ir|o|es|e|imos|ís|en|ía|ías|íamos|íais|ían|í|iste|ió|isteis|ieron|a|as|amos|áis|an` +
		`|iera|ieras|iéramos|ierais|ieran|iese|ieses|iésemos|ieseis|iesen|iere|ieres|iéremos|iereis|ieren` +
		`|id|iendo|ido|ida|idos|idas))(?:$|[^0-9A-Za-z])`)

	// Un sustantivo de Git con su identificador designado a continuación: «la
	// rama main», «el commit de origin/x». Sin identificador no casa.
	nounReferencePattern = regexp.MustCompile(`(?i)\b(?:commit|commits|merge|push|checkout|hash|rama|branch)\b[\s:]*(?:(?:de|del|en|con|a|al|la|el|los|las|the)\s+){0,3}(?:main|master|HEAD|origin/[a-z0-9][a-z0-9._-]*|(?:feat|feature|fix|hotfix|release|refactor|chore|docs)/[a-z0-9][a-z0-9._-]*)\b`)

	// La cláusula termina en punto, dos puntos, punto y coma, cierre o salto de
	// línea; la coma no la corta.
	clauseBreakPattern = regexp.MustCompile(`[.:;!?\n]`)
)

// textRule is one of the lists of the contract: what finds the fragment that
// writes the state of Git into a text, and the words the rejection adds after
// naming the field and the fragment. Se leen en el orden del contrato, que es el
// orden en que el rechazo nombra los fragmentos.
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

// Los verbos del mensaje y el remedio que comparten las listas de Git.
const (
	nameVerb    = "name"
	declareVerb = "declare"
	noGitRemedy = "the board does not store Git"
)

// textRules are the lists `R1` to `R7` of the contract, en su orden.
var textRules = []textRule{
	{
		find:   commitHash,
		verb:   nameVerb,
		what:   "a commit",
		remedy: "a commit reaches its task through the commit message, with Refs: tk-N",
	},
	{
		find:   branchName,
		verb:   nameVerb,
		what:   "a branch",
		remedy: noGitRemedy,
	},
	{
		find:   remotePattern.FindString,
		verb:   nameVerb,
		what:   "a remote reference",
		remedy: noGitRemedy,
	},
	{
		find:   stateMarker,
		verb:   declareVerb,
		what:   "the state of the task",
		remedy: "state what was done, found or decided instead",
	},
	{
		find:   deliveryPattern.FindString,
		verb:   declareVerb,
		what:   "the state of the delivery",
		remedy: noGitRemedy,
	},
	{
		find:   gitReferenceWithVerb,
		verb:   declareVerb,
		what:   "the state of Git",
		remedy: noGitRemedy,
	},
	{
		find:   nounReference,
		verb:   nameVerb,
		what:   "a Git reference",
		remedy: noGitRemedy,
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
// nada. El mensaje nombra todos los fragmentos que casaron y añade una sola vez
// el remedio de cada lista.
func checkText(field, text string) error {
	var (
		fragments []string
		remedies  []string
	)

	for _, rule := range textRules {
		fragment := rule.find(text)
		if fragment == "" {
			continue
		}

		fragments = append(fragments, fmt.Sprintf("the %s %s %s: %q", field, rule.agrees(field), rule.what, fragment))

		if !slices.Contains(remedies, rule.remedy) {
			remedies = append(remedies, rule.remedy)
		}
	}

	if len(fragments) == 0 {
		return nil
	}

	return invalidField(field, "%s; %s", strings.Join(fragments, "; "), strings.Join(remedies, "; "))
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

// branchName finds the name of a branch. La forma `docs/` se limita al slug
// suelto —sin punto y sin continuar como ruta—, porque en este repositorio
// `docs/<algo>` es también la ruta de un documento y la lista no debe rechazar
// una ruta: «docs/unify-design-system» es una rama; «docs/contracts.md» y
// «docs/decisions/», documentos.
func branchName(text string) string {
	for _, found := range branchPattern.FindAllStringIndex(text, -1) {
		name := text[found[0]:found[1]]

		if !strings.HasPrefix(strings.ToLower(name), "docs/") {
			return name
		}

		name = strings.TrimRight(name, ".")

		if strings.Contains(name, ".") || (found[1] < len(text) && text[found[1]] == '/') {
			continue
		}

		return name
	}

	return ""
}

// gitReferenceWithVerb finds a named Git reference —main, master o HEAD— con un
// verbo de Git en la misma cláusula: «mergeado a main» afirma el estado de Git,
// igual que nombrar una rama. La referencia sola, sin verbo, no casa. Una
// referencia pegada a una extensión de fichero es el nombre de un fichero —en
// este repositorio, la prosa habla de `main.go`—, así que se descarta: la
// expresión no mira hacia delante y el descarte se hace aquí. El patrón no
// distingue mayúsculas, y el descarte tampoco: `head.h` es un fichero igual que
// `HEAD.h`.
func gitReferenceWithVerb(text string) string {
	for _, ref := range gitRefPattern.FindAllStringIndex(text, -1) {
		if isFileName(text[ref[1]:]) {
			continue
		}

		start := clauseStart(text[:ref[0]])
		clause := text[start:clauseEnd(text, ref[1])]

		verb := gitVerbPattern.FindStringIndex(clause)
		if verb == nil {
			continue
		}

		from, to := ref[0]-start, ref[1]-start
		if verb[0] < from {
			from = verb[0]
		}

		if verb[1] > to {
			to = verb[1]
		}

		return strings.TrimSpace(clause[from:to])
	}

	return ""
}

// nounReference finds a Git noun with its designated identifier. La referencia
// propia que el sustantivo designa queda invalidada cuando va pegada a una
// extensión de fichero —`el merge de main.go` nombra un fichero—, con la misma
// guarda y el mismo motivo que en `gitReferenceWithVerb`.
func nounReference(text string) string {
	for _, found := range nounReferencePattern.FindAllStringIndex(text, -1) {
		fragment := text[found[0]:found[1]]

		if isNamedRef(fragment) && isFileName(text[found[1]:]) {
			continue
		}

		return fragment
	}

	return ""
}

// isNamedRef reports whether the fragment ends in a reference with a proper
// name: así la guarda de ficheros se aplica a `main`, `master` y `HEAD`, y no a
// las formas de rama y remoto, cuyo slug ya absorbe los puntos.
func isNamedRef(fragment string) bool {
	for _, name := range []string{"main", "master", "head"} {
		if len(fragment) >= len(name) && strings.EqualFold(fragment[len(fragment)-len(name):], name) {
			return true
		}
	}

	return false
}

// isFileName reports whether what follows the reference is a file extension: un
// punto y una letra o un dígito. Un punto seguido de espacio o del final del
// texto cierra la frase y la referencia sigue siendo una referencia.
func isFileName(rest string) bool {
	if len(rest) < 2 || rest[0] != '.' {
		return false
	}

	next := rest[1]

	return next >= '0' && next <= '9' || next >= 'a' && next <= 'z' || next >= 'A' && next <= 'Z'
}

// clauseStart returns where the clause that ends in the text starts: el último
// corte más uno.
func clauseStart(text string) int {
	breaks := clauseBreakPattern.FindAllStringIndex(text, -1)
	if len(breaks) == 0 {
		return 0
	}

	return breaks[len(breaks)-1][0] + 1
}

// clauseEnd returns where the clause that starts before the position ends: el
// siguiente corte, o el final del texto.
func clauseEnd(text string, from int) int {
	if next := clauseBreakPattern.FindStringIndex(text[from:]); next != nil {
		return from + next[0]
	}

	return len(text)
}
