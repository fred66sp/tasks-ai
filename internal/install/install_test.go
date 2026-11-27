package install_test

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"tasks-ai/internal/core"
	"tasks-ai/internal/install"
	"tasks-ai/internal/settings"
	"tasks-ai/internal/store"
	"testing"
)

func TestRoot_staysInTheRepositoryTheFolderBelongsTo(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	mkdir(t, filepath.Join(dir, ".git"))

	deep := filepath.Join(dir, "uno", "dos")
	mkdir(t, deep)

	if got := install.Root(deep); got != dir {
		t.Errorf("Root(%q) = %q, se esperaba %q", deep, got, dir)
	}
}

func TestRoot_findsARepositoryWhoseGitEntryIsAFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".git"), []byte("gitdir: outside"), 0o600); err != nil {
		t.Fatalf("preparando el .git como fichero: %v", err)
	}

	deep := filepath.Join(dir, "uno", "dos")
	mkdir(t, deep)

	if got := install.Root(deep); got != dir {
		t.Errorf("Root(%q) = %q, se esperaba %q", deep, got, dir)
	}
}

func TestRoot_fallsBackToTheFolderItWasGiven(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	deep := filepath.Join(dir, "uno")
	mkdir(t, deep)

	if got := install.Root(deep); got != deep {
		t.Errorf("Root(%q) = %q, se esperaba la carpeta dada", deep, got)
	}
}

func TestProject_findsTheInstalledFolderAbove(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	mkdir(t, filepath.Join(dir, install.DirName))

	deep := filepath.Join(dir, "uno", "dos")
	mkdir(t, deep)

	got, err := install.Project(deep)
	if err != nil {
		t.Fatalf("Project devolvió %v", err)
	}

	if got != dir {
		t.Errorf("Project(%q) = %q, se esperaba %q", deep, got, dir)
	}
}

func TestProject_saysWhenTheProductIsNotInstalled(t *testing.T) {
	t.Parallel()

	if _, err := install.Project(t.TempDir()); err == nil {
		t.Error("Project aceptó una carpeta sin el producto instalado")
	}
}

func TestRun_leavesItsFolderWithTheBaseTheSettingsAndTheInstructions(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	result, err := install.Run(t.Context(), dir, "")
	if err != nil {
		t.Fatalf("Run devolvió %v", err)
	}

	product := filepath.Join(dir, install.DirName)
	if result.Dir != product {
		t.Errorf("la instalación quedó en %q, se esperaba %q", result.Dir, product)
	}

	if result.Prefix != settings.DefaultPrefix {
		t.Errorf("la instalación devolvió %+v, se esperaba el prefijo por omisión", result)
	}

	for _, fragment := range []string{"AGENTS.md", "CONTRATO_ETIQUETAS.md"} {
		if !strings.Contains(result.Pointer, fragment) {
			t.Errorf("el texto de fuera no menciona %q: %q", fragment, result.Pointer)
		}
	}

	for _, name := range []string{"tasks.db", "settings.json", "AGENTS.md", install.VersionName} {
		if _, err := os.Stat(filepath.Join(product, name)); err != nil {
			t.Errorf("la instalación no dejó %s: %v", name, err)
		}
	}

	if _, err := os.Stat(result.Instructions); err != nil {
		t.Errorf("las instrucciones que anuncia la instalación no están: %v", err)
	}
}

// TestInstructionsInTheRepositoryAreTheShippedOnes guarda la deriva de las
// instrucciones instaladas de este propio repositorio respecto de las que
// viajan en el binario: el proyecto anfitrión se actualiza con el mismo
// upgrade que entrega y sus instrucciones no se personalizan a mano.
func TestRun_leavesTheBinaryTheVersionAndNoExtensions(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	result, err := install.Run(t.Context(), dir, "")
	if err != nil {
		t.Fatalf("Run devolvió %v", err)
	}

	product := filepath.Join(dir, install.DirName)

	bin := filepath.Join(product, "bin", "tasks-ai")

	info, err := os.Stat(bin)
	if err != nil {
		t.Fatalf("la instalación no dejó el binario: %v", err)
	}

	if info.Mode()&0o111 == 0 {
		t.Errorf("el binario instalado no es ejecutable: %v", info.Mode())
	}

	if result.Binary != bin {
		t.Errorf("la instalación devolvió el binario %q, se esperaba %q", result.Binary, bin)
	}

	version, err := os.ReadFile(filepath.Join(product, install.VersionName)) //nolint:gosec // la ruta la prepara la prueba
	if err != nil {
		t.Fatalf("la instalación no dejó la versión: %v", err)
	}

	if string(version) != core.Version+"\n" {
		t.Errorf("la instalación dejó la versión %q, se esperaba %q", version, core.Version)
	}

	if result.Version != core.Version {
		t.Errorf("la instalación devolvió la versión %q, se esperaba %q", result.Version, core.Version)
	}

	if _, err := os.Stat(filepath.Join(product, "extensions")); err == nil {
		t.Error("la instalación dejó la carpeta de extensiones retirada")
	}
}

// TestRun_refusesToWriteThroughASymbolicLink comprueba que un repositorio no
// puede llevar el contenido del producto a un fichero de la persona. El árbol
// que se clona trae el enlace; la instalación escribe con los permisos de quien
// la ejecuta.
func TestRun_refusesToWriteThroughASymbolicLink(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	kept := "no se toca"

	target := filepath.Join(t.TempDir(), "un-fichero-de-la-persona")
	if err := os.WriteFile(target, []byte(kept), 0o600); err != nil {
		t.Fatalf("preparando el fichero de fuera: %v", err)
	}

	productBinary := filepath.Join(dir, install.DirName, "bin")
	mkdir(t, productBinary)

	if err := os.Symlink(target, filepath.Join(productBinary, "tasks-ai")); err != nil {
		t.Fatalf("preparando el enlace simbólico: %v", err)
	}

	if _, err := install.Run(t.Context(), dir, ""); err == nil {
		t.Fatal("la instalación escribió a través de un enlace simbólico")
	}

	raw, err := os.ReadFile(target) //nolint:gosec // el destino lo prepara la prueba en su propio directorio
	if err != nil {
		t.Fatalf("leyendo el fichero de fuera: %v", err)
	}

	if string(raw) != kept {
		t.Fatalf("el fichero de fuera del proyecto cambió: %q", raw)
	}
}

// TestRun_refusesToWriteThroughALinkedFolder comprueba lo mismo con la carpeta
// entera: un enlace en lugar de bin lleva el binario del producto a la carpeta a
// la que apunta.
func TestRun_refusesToWriteThroughALinkedFolder(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	outside := t.TempDir()

	mkdir(t, filepath.Join(dir, install.DirName))

	if err := os.Symlink(outside, filepath.Join(dir, install.DirName, "bin")); err != nil {
		t.Fatalf("preparando el enlace simbólico: %v", err)
	}

	if _, err := install.Run(t.Context(), dir, ""); err == nil {
		t.Fatal("la instalación escribió en una carpeta enlazada")
	}

	written, err := os.ReadDir(outside)
	if err != nil {
		t.Fatalf("leyendo la carpeta de fuera: %v", err)
	}

	if len(written) != 0 {
		t.Fatalf("la instalación dejó %d entradas en la carpeta enlazada", len(written))
	}
}

func TestRun_completesAnInstallationFromAnOlderVersion(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	if _, err := install.Run(t.Context(), dir, ""); err != nil {
		t.Fatalf("la primera instalación devolvió %v", err)
	}

	product := filepath.Join(dir, install.DirName)
	for _, name := range []string{"bin", install.VersionName} {
		if err := os.RemoveAll(filepath.Join(product, name)); err != nil {
			t.Fatalf("retirando %s: %v", name, err)
		}
	}

	result, err := install.Run(t.Context(), dir, "")
	if err != nil {
		t.Fatalf("completar una instalación anterior devolvió %v", err)
	}

	for _, path := range []string{result.Binary, filepath.Join(product, install.VersionName)} {
		if _, err := os.Stat(path); err != nil {
			t.Errorf("completar la instalación no restituyó %s: %v", path, err)
		}
	}
}

func TestRun_refusesToInstallOverAnInstallation(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	if _, err := install.Run(t.Context(), dir, ""); err != nil {
		t.Fatalf("la primera instalación devolvió %v", err)
	}

	if _, err := install.Run(t.Context(), dir, ""); err == nil {
		t.Error("Run volvió a instalar sobre un proyecto que ya lo tenía")
	}
}

func TestRun_completesAnInstallationThatLostItsInstructions(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	if _, err := install.Run(t.Context(), dir, ""); err != nil {
		t.Fatalf("la primera instalación devolvió %v", err)
	}

	instructions := filepath.Join(dir, install.DirName, "AGENTS.md")
	if err := os.Remove(instructions); err != nil {
		t.Fatalf("quitando las instrucciones: %v", err)
	}

	if _, err := install.Run(t.Context(), dir, ""); err != nil {
		t.Fatalf("volver a instalar sobre una instalación incompleta devolvió %v", err)
	}

	if _, err := os.Stat(instructions); err != nil {
		t.Errorf("volver a instalar no devolvió las instrucciones: %v", err)
	}
}

func TestRun_completesAnInstallationOverABaseComposedWithAnotherPrefix(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	product := filepath.Join(dir, install.DirName)

	if _, err := install.Run(t.Context(), dir, "px"); err != nil {
		t.Fatalf("la primera instalación devolvió %v", err)
	}

	// La instalación queda incompleta y la base se queda con el prefijo con el
	// que se compuso.
	for _, name := range []string{settings.FileName, "AGENTS.md"} {
		if err := os.Remove(filepath.Join(product, name)); err != nil {
			t.Fatalf("quitando %s: %v", name, err)
		}
	}

	result, err := install.Run(t.Context(), dir, "")
	if err != nil {
		t.Fatalf("completar la instalación devolvió %v", err)
	}

	if result.Prefix != settings.DefaultPrefix {
		t.Errorf("la instalación quedó con el prefijo %q, se esperaba el de los ajustes", result.Prefix)
	}

	cfg, err := settings.Load(product)
	if err != nil {
		t.Fatalf("Load devolvió %v", err)
	}

	base, err := store.Open(t.Context(), settings.DatabasePath(product, cfg))
	if err != nil {
		t.Fatalf("abriendo la base instalada: %v", err)
	}

	t.Cleanup(func() { _ = base.Close() })

	recorded, err := base.RecordedPrefix(t.Context())
	if err != nil {
		t.Fatalf("RecordedPrefix devolvió %v", err)
	}

	if recorded != "px" {
		t.Errorf("la instalación reescribió el prefijo de la base, quedó en %q y debía seguir en px", recorded)
	}

	// El desajuste sigue siendo real después de completar: eso es lo que init
	// tiene que avisar por la salida de errores.
	note, err := core.New(base, cfg.Prefix).CheckPrefix(t.Context())
	if err != nil {
		t.Fatalf("CheckPrefix devolvió %v", err)
	}

	if note == "" {
		t.Error("completar la instalación se cargó el desajuste de prefijo que había que avisar")
	}
}

func TestRun_keepsTheSettingsThatAreAlreadyThere(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	product := filepath.Join(dir, install.DirName)
	mkdir(t, product)

	cfg := settings.Default()
	cfg.Prefix = "px"

	if err := settings.Save(product, cfg); err != nil {
		t.Fatalf("preparando los ajustes del proyecto: %v", err)
	}

	result, err := install.Run(t.Context(), dir, "")
	if err != nil {
		t.Fatalf("Run devolvió %v", err)
	}

	if result.Prefix != "px" {
		t.Errorf("la instalación se quedó con el prefijo %q, se esperaba el que el proyecto había puesto", result.Prefix)
	}

	read, err := settings.Load(product)
	if err != nil {
		t.Fatalf("Load devolvió %v", err)
	}

	if read != cfg {
		t.Errorf("la instalación reescribió los ajustes: %+v frente a %+v", read, cfg)
	}

	base, err := store.Open(t.Context(), settings.DatabasePath(product, read))
	if err != nil {
		t.Fatalf("abriendo la base instalada: %v", err)
	}

	t.Cleanup(func() { _ = base.Close() })

	recorded, err := base.RecordedPrefix(t.Context())
	if err != nil {
		t.Fatalf("RecordedPrefix devolvió %v", err)
	}

	if recorded != "px" {
		t.Errorf("la base se anotó con el prefijo %q, se esperaba el de los ajustes", recorded)
	}
}

func TestRun_installsWithThePrefixTheProjectAsks(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	result, err := install.Run(t.Context(), dir, "tai")
	if err != nil {
		t.Fatalf("Run devolvió %v", err)
	}

	if result.Prefix != "tai" {
		t.Errorf("la instalación devolvió el prefijo %q, se esperaba tai", result.Prefix)
	}

	product := filepath.Join(dir, install.DirName)

	cfg, err := settings.Load(product)
	if err != nil {
		t.Fatalf("Load devolvió %v", err)
	}

	if cfg.Prefix != "tai" {
		t.Errorf("los ajustes guardan el prefijo %q, se esperaba tai", cfg.Prefix)
	}

	written, err := os.ReadFile(result.Instructions)
	if err != nil {
		t.Fatalf("leyendo las instrucciones: %v", err)
	}

	if !strings.Contains(string(written), "tai-1") || strings.Contains(string(written), "tk-") {
		t.Errorf("las instrucciones no componen sus ejemplos con el prefijo del proyecto")
	}

	base, err := store.Open(t.Context(), settings.DatabasePath(product, cfg))
	if err != nil {
		t.Fatalf("abriendo la base instalada: %v", err)
	}

	t.Cleanup(func() { _ = base.Close() })

	recorded, err := base.RecordedPrefix(t.Context())
	if err != nil {
		t.Fatalf("RecordedPrefix devolvió %v", err)
	}

	if recorded != "tai" {
		t.Errorf("la base se anotó con el prefijo %q, se esperaba tai", recorded)
	}

	note, err := core.New(base, cfg.Prefix).CheckPrefix(t.Context())
	if err != nil {
		t.Fatalf("CheckPrefix devolvió %v", err)
	}

	if note != "" {
		t.Errorf("la instalación dejó un aviso para los comandos siguientes: %q", note)
	}
}

func TestRun_refusesAPrefixThatTheProjectSettingsDoNotHave(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	product := filepath.Join(dir, install.DirName)
	mkdir(t, product)

	cfg := settings.Default()
	cfg.Prefix = "px"

	if err := settings.Save(product, cfg); err != nil {
		t.Fatalf("preparando los ajustes del proyecto: %v", err)
	}

	if _, err := install.Run(t.Context(), dir, "tai"); err == nil {
		t.Error("Run aceptó un prefijo distinto del que el proyecto ya tenía en sus ajustes")
	}

	result, err := install.Run(t.Context(), dir, "px")
	if err != nil {
		t.Fatalf("Run rechazó el prefijo que el proyecto ya tenía: %v", err)
	}

	if result.Prefix != "px" {
		t.Errorf("la instalación devolvió el prefijo %q, se esperaba px", result.Prefix)
	}
}

func mkdir(t *testing.T, path string) {
	t.Helper()

	if err := os.MkdirAll(path, 0o755); err != nil { //nolint:gosec // la carpeta de prueba usa los permisos del producto real
		t.Fatalf("creando %s: %v", path, err)
	}
}

func TestRun_leavesTheCatalogueAndTheTagsContract(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	result, err := install.Run(t.Context(), dir, "")
	if err != nil {
		t.Fatalf("Run devolvió %v", err)
	}

	if _, err := os.Stat(result.Contract); err != nil {
		t.Errorf("el contrato de etiquetas que anuncia la instalación no está: %v", err)
	}

	names := make([]string, 0)

	for _, tag := range tagsOf(t, result.DB) {
		names = append(names, tag.Name)

		if tag.Description == nil || strings.TrimSpace(*tag.Description) == "" {
			t.Errorf("la etiqueta %q nace sin descripción", tag.Name)
		}
	}

	slices.Sort(names)

	want := []string{"ODD", "chore", "docs", "epic", "feat", "fix", "improvement", "refactor", "style", "task", "test"}
	if !slices.Equal(names, want) {
		t.Errorf("la instalación dejó %v, se esperaba %v", names, want)
	}
}

func TestRun_keepsTheCatalogueAnInstallationAlreadyHas(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	first, err := install.Run(t.Context(), dir, "")
	if err != nil {
		t.Fatalf("Run devolvió %v", err)
	}

	// La persona cambia una etiqueta y el contrato desaparece: completar la
	// instalación no puede pisar ni repetir lo que ya está.
	base, err := store.Open(t.Context(), first.DB)
	if err != nil {
		t.Fatalf("abriendo la base: %v", err)
	}

	mine := "la que yo quiero"

	if _, err := base.UpdateTag(t.Context(), "feat", func(tag store.Tag) (store.Tag, error) {
		tag.Color = "red"
		tag.Description = &mine

		return tag, nil
	}); err != nil {
		t.Fatalf("cambiando la etiqueta: %v", err)
	}

	_ = base.Close()

	if err := os.Remove(first.Contract); err != nil {
		t.Fatalf("retirando el contrato: %v", err)
	}

	if _, err := install.Run(t.Context(), dir, ""); err != nil {
		t.Fatalf("completar la instalación devolvió %v", err)
	}

	catalogue := tagsOf(t, first.DB)
	if len(catalogue) != 11 {
		t.Errorf("el catálogo quedó con %d etiquetas, se esperaban 11", len(catalogue))
	}

	for _, tag := range catalogue {
		if tag.Name != "feat" {
			continue
		}

		if tag.Color != "red" || tag.Description == nil || *tag.Description != mine {
			t.Errorf("la etiqueta feat quedó en %q y %v, se esperaba respetar el cambio de la persona", tag.Color, tag.Description)
		}
	}
}

func TestRun_writesTheContractTheProductShips(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	result, err := install.Run(t.Context(), dir, "")
	if err != nil {
		t.Fatalf("Run devolvió %v", err)
	}

	written, err := os.ReadFile(result.Contract)
	if err != nil {
		t.Fatalf("leyendo el contrato instalado: %v", err)
	}

	shipped, err := os.ReadFile("contract_tags.md")
	if err != nil {
		t.Fatalf("leyendo el contrato que viaja en el binario: %v", err)
	}

	if !bytes.Equal(written, shipped) {
		t.Errorf("el contrato instalado no es el que viaja en el binario")
	}

	repository, err := os.ReadFile(filepath.Join("..", "..", ".tasks-ai", "CONTRATO_ETIQUETAS.md"))
	if err != nil {
		t.Fatalf("leyendo el contrato del repositorio: %v", err)
	}

	if !bytes.Equal(shipped, repository) {
		t.Errorf("el contrato del repositorio no es el que viaja en el binario")
	}
}

// tagsOf abre la base de una instalación y devuelve su catálogo.
func tagsOf(t *testing.T, dbPath string) []store.TagUse {
	t.Helper()

	base, err := store.Open(t.Context(), dbPath)
	if err != nil {
		t.Fatalf("abriendo la base %s: %v", dbPath, err)
	}

	t.Cleanup(func() { _ = base.Close() })

	catalogue, err := base.Tags(t.Context())
	if err != nil {
		t.Fatalf("leyendo el catálogo: %v", err)
	}

	return catalogue
}

// TestInstall_suggestsThePersonFromGitOnce deja escrito que la instalación
// propone la persona una sola vez, desde Git, y que no la inventa si Git no
// devuelve nombre.
func TestInstall_suggestsThePersonFromGitOnce(t *testing.T) {
	tests := []struct {
		name   string
		script string
		want   string
	}{
		{name: "con nombre", script: "#!/bin/sh\nprintf 'Alfredo González P.'\n", want: "Alfredo González P."},
		{name: "sin nombre", script: "#!/bin/sh\nexit 1\n", want: ""},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			stub := t.TempDir()

			script := filepath.Join(stub, "git")
			if err := os.WriteFile(script, []byte(test.script), 0o700); err != nil { //nolint:gosec // la prueba prepara su propio ejecutable
				t.Fatalf("preparando el git de la prueba: %v", err)
			}

			t.Setenv("PATH", stub)

			project := t.TempDir()
			mkdir(t, filepath.Join(project, ".git"))

			if _, err := install.Run(t.Context(), project, ""); err != nil {
				t.Fatalf("instalando el producto: %v", err)
			}

			cfg, err := settings.Load(filepath.Join(project, install.DirName))
			if err != nil {
				t.Fatalf("leyendo los ajustes: %v", err)
			}

			if cfg.Person != test.want {
				t.Errorf("la persona quedó %q, se esperaba %q", cfg.Person, test.want)
			}
		})
	}
}

// TestInstructionsInTheRepositoryAreTheShippedOnes guarda la deriva de las
// instrucciones instaladas de este propio repositorio respecto de las que
// viajan en el binario: el proyecto anfitrión se actualiza con el mismo
// upgrade que entrega y sus instrucciones no se personalizan a mano.
func TestInstructionsInTheRepositoryAreTheShippedOnes(t *testing.T) {
	t.Parallel()

	shipped, err := os.ReadFile("agent.md")
	if err != nil {
		t.Fatalf("leyendo las instrucciones que viajan en el binario: %v", err)
	}

	repository, err := os.ReadFile(filepath.Join("..", "..", ".tasks-ai", "AGENTS.md"))
	if err != nil {
		t.Fatalf("leyendo las instrucciones del repositorio: %v", err)
	}

	if !bytes.Equal(shipped, repository) {
		t.Errorf("las instrucciones del repositorio no son las que viajan en el binario")
	}
}
