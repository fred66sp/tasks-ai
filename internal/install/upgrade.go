package install

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"tasks-ai/internal/core"
	"tasks-ai/internal/settings"
)

// LegacyVersion is the version of an installation made before the product
// recorded one in its folder.
const LegacyVersion = "0.1.0"

// legacyExtensionFiles are the files v0.1.0 distributed inside each extension
// folder, with the SHA-256 of each file as the release shipped it. The folder
// is retired file by file only when both name and hash match: an original name
// with a different content belongs to the project and stays.
var legacyExtensionFiles = map[string]map[string]string{
	"opencode": {
		"index.test.ts":     "04344941e63b695ce45ede652cb55d8b03766bbcf27abf67e8cde048606ea6c6",
		"index.ts":          "e775ea5459bf88be472e6c3af8ea82799ee897b88fafb544f0c6ea48cd7a82f0",
		"package-lock.json": "0b59d9492c29f936198179b30c9f04d8ae8b9b6e52e3a7b4869084433ed529cc",
		"package.json":      "d1550b6863970765080b5aed40cb53c45774bb08b78e11a5f2eee01999c5ab0f",
		"tasks.ts":          "9e5b14d370b0bd8eac9cfa0513f0ae864271b542537c42f056e57098c737febc",
		"tsconfig.json":     "d06b3e0a809c81cbdadcb4f08ef13e5ce81af8f05c455554679a20a50a2d5293",
		filepath.Join("types", "opencode-plugin.d.ts"): "555d38c5e01213945cb8cc0324abb42dc1e1b3d5dbe162a26dbfa54a16573802",
	},
	"pi": {
		"binary-name.test.ts": "c8bd5ae88c435f4474466978a0306ca2353a4c1980444d037f3232e8aed4a02fe",
		"board.test.ts":       "298626de1648c4bf61e38680b3951183ba8f8c66a3b2d38d9d6604710b2d5714",
		"board.ts":            "424ebd30c604d93d92af13d0af84e8b0aaaa84ada941da8c5a236d6d99b39530",
		"cli.test.ts":         "c83178a7f45e5e01818cfdc6c981824bf2577a3a9be7f89bc6009d129d0f5f66",
		"cli.ts":              "f93593e9152e7d2a1f5ead6d2181017c701499b0e48a65de36d7fea7e236a835",
		"index.test.ts":       "847c8ae18785f4df3ed2f4f5f216a4f42cc86232337b07de77353a5d6089cfb5",
		"index.ts":            "ca804de3827ad8a70d4851e80c1cca3267020e02be90361fa400233c903e036e",
		"package-lock.json":   "377b53302d41d4549387fe0a89a9f50cf2f580424b4c781e691398e93b4c08cd",
		"package.json":        "f700aeaa9b63cff2fb084202876857bbdf03d3a0cdfe6bc4c6306137fe7fe6fd",
		"test-binary.ts":      "8ce39f07155510f34a617ff7c016acdb046735b29e13327d1c58d7589f6506a1",
		"tsconfig.json":       "d06b3e0a809c81cbdadcb4f08ef13e5ce81af8f05c455554679a20a50a2d5293",
	},
}

// ErrUpgradeRefused marks an upgrade the product refuses before writing
// anything: there is no installation, its version cannot be read, or this
// binary is older than the installed one.
var ErrUpgradeRefused = errors.New("upgrade refused")

// The outcomes of refreshing the instructions.
const (
	instructionsUpdated = "updated"
	instructionsCurrent = "current"
	instructionsPending = "pending"
)

// UpgradeReport is what an upgrade did and what it left pending. A pending list
// that is not empty means the installation is not complete and its version was
// not recorded, so the command answers with a failure code.
type UpgradeReport struct {
	Dir          string   `json:"dir"`
	From         string   `json:"from"`
	To           string   `json:"to"`
	Version      string   `json:"version"`
	Instructions string   `json:"instructions"`
	Contract     string   `json:"contract"`
	Steps        []string `json:"steps"`
	Pending      []string `json:"pending,omitempty"`
}

// Upgrade brings the installation of the project dir up to this binary, without
// downloading anything: it copies the running binary, refreshes the label
// contract, replaces the instructions of a project that still carries the v0.1.0
// template, removes the host references and the extension folders of the retired
// integrations and records the version last. Every step compares before writing,
// so an interrupted upgrade finishes on the next run and a completed one is left
// alone.
func Upgrade(dir string) (UpgradeReport, error) {
	product, installed := installedAt(dir)
	if !installed {
		return UpgradeReport{}, fmt.Errorf(
			"%w: there is no project with the product installed in %s or in the folders that contain it",
			ErrUpgradeRefused, dir,
		)
	}

	report := UpgradeReport{Dir: product, To: core.Version}

	from, err := readVersion(product)
	if err != nil {
		return UpgradeReport{}, err
	}

	report.From = from

	if compareVersions(core.Version, from) < 0 {
		return UpgradeReport{}, fmt.Errorf(
			"%w: the installation in %s is version %s and this binary is %s: a downgrade is not allowed",
			ErrUpgradeRefused, product, from, core.Version,
		)
	}

	cfg, err := settings.Load(product)
	if err != nil {
		return UpgradeReport{}, err
	}

	refreshed, err := refreshBinary(product)
	if err != nil {
		return report, err
	}

	if refreshed {
		report.Steps = append(report.Steps, "binary refreshed")
	} else {
		report.Steps = append(report.Steps, "binary current")
	}

	contract, err := refreshContract(product)
	if err != nil {
		return report, err
	}

	report.Contract = contract
	report.Steps = append(report.Steps, "contract "+contract)

	instructions, err := refreshInstructions(product, cfg.Prefix)
	if err != nil {
		return report, err
	}

	report.Instructions = instructions
	report.Steps = append(report.Steps, "instructions "+instructions)

	steps, pending, err := upgradeHosts(filepath.Dir(product), product)
	if err != nil {
		return report, err
	}

	report.Steps = append(report.Steps, steps...)
	report.Pending = append(report.Pending, pending...)

	// La versión es el acreditor de la actualización completa: solo se escribe
	// cuando no queda nada pendiente.
	if len(report.Pending) == 0 {
		if err := writeVersion(product, core.Version); err != nil {
			return report, err
		}

		report.Version = core.Version
		report.Steps = append(report.Steps, "version recorded")
	}

	return report, nil
}

// upgradeHosts removes the host references to the retired extensions and the
// extension folders that no exact reference keeps alive, returning what it did
// and what it left pending.
func upgradeHosts(root, product string) (steps, pending []string, err error) {
	refs := []hostRef{
		{path: filepath.Join(root, "opencode.jsonc"), key: "plugins", element: "./" + DirName + "/extensions/opencode", extension: "opencode"},
		{path: filepath.Join(root, ".pi", "settings.json"), key: "packages", element: "../" + DirName + "/extensions/pi", extension: "pi"},
	}

	for _, ref := range refs {
		outcome, err := removeHostRef(ref.path, ref.key, ref.element)
		if err != nil {
			return steps, pending, err
		}

		switch outcome {
		case refRemoved:
			steps = append(steps, ref.path+": reference removed")
		case refBlocked:
			pending = append(pending, fmt.Sprintf(
				"%s: the %q array has a comment inside; remove %q by hand: the extension folder is kept until the reference is removed",
				ref.path, ref.key, ref.element,
			))

			continue
		case refMissing, refAbsent:
			steps = append(steps, ref.path+": no reference")
		}

		alive, err := hasHostRef(ref.path, ref.key, ref.element)
		if err != nil {
			return steps, pending, err
		}

		if alive {
			pending = append(pending, fmt.Sprintf(
				"%s still points at %s: the extension folder is kept until the reference is removed",
				ref.path, ref.element,
			))

			continue
		}

		step, err := retireExtension(product, ref.extension)
		if err != nil {
			return steps, pending, err
		}

		steps = append(steps, step)
	}

	// El padre de las extensiones retiradas queda vacío: si no cupo nada más
	// dentro, se retira también; un resto del proyecto lo deja intacto.
	if err := os.Remove(filepath.Join(product, "extensions")); err != nil && !os.IsNotExist(err) {
		steps = append(steps, "the extensions folder keeps something: it is left alone")

		return steps, pending, nil
	}

	return steps, pending, nil
}

// retireExtension retira la extensión de una instalación heredada: solo los
// ficheros con el nombre y el contenido que la v0.1.0 los entregó —un nombre
// con otro contenido es un fichero del proyecto y se conserva, contado—, y el
// directorio cuando queda vacío. Lo demás es del proyecto y permanece: su
// presencia se informa y no bloquea la actualización.
func retireExtension(product, name string) (string, error) {
	extension := filepath.Join(product, "extensions", name)

	if !exists(extension) {
		return "extension " + name + " already gone", nil
	}

	removed := 0

	kept := 0

	for file, hash := range legacyExtensionFiles[name] {
		path := filepath.Join(extension, file)

		data, err := os.ReadFile(path) //nolint:gosec // la ruta la compone la instalación, no una entrada externa
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}

			return "", fmt.Errorf("reading %s: %w", path, err)
		}

		if sha256.Sum256(data) != legacySum(hash) {
			kept++

			continue
		}

		if err := os.Remove(path); err != nil {
			return "", fmt.Errorf("removing %s: %w", path, err)
		}

		removed++

		// El fichero puede ser el último de una subcarpeta propia: si quedó
		// vacía, se retira con él.
		if folder := filepath.Dir(file); folder != "." {
			_ = os.Remove(filepath.Join(extension, folder))
		}
	}

	// El directorio se retira solo si quedó vacío: cualquier resto propio lo
	// conserva; se separa del producto pero la actualización sigue completa.
	if empty := os.Remove(extension) == nil; !empty {
		return fmt.Sprintf("extension %s removed %d files and keeps %d by the project: they are left alone", name, removed, kept), nil
	}

	if kept > 0 {
		return fmt.Sprintf("extension %s removed (%d files) and kept %d edited by the project", name, removed, kept), nil
	}

	return fmt.Sprintf("extension %s removed (%d files)", name, removed), nil
}

// legacyHash decodes the hash recorded with each distributed file.
func legacySum(release string) [sha256.Size]byte {
	var sum [sha256.Size]byte

	if _, err := hex.Decode(sum[:], []byte(release)); err != nil {
		return [sha256.Size]byte{}
	}

	return sum
}

// installedAt finds the product folder above dir, with the structure a usable
// installation has.
func installedAt(dir string) (string, bool) {
	root, found := walkUp(dir, func(current string) bool {
		return validInstallation(filepath.Join(current, DirName))
	})
	if !found {
		return "", false
	}

	return filepath.Join(root, DirName), true
}

// validInstallation says whether a product folder has the files a usable
// installation needs.
func validInstallation(product string) bool {
	if !exists(filepath.Join(product, settings.FileName)) || !exists(filepath.Join(product, "bin", binName)) {
		return false
	}

	cfg, err := settings.Load(product)
	if err != nil {
		return false
	}

	return exists(settings.DatabasePath(product, cfg))
}

// readVersion reads the version the installation recorded. An installation
// without the file is a legacy one, made before the product recorded versions.
func readVersion(product string) (string, error) {
	path := filepath.Join(product, VersionName)

	raw, err := os.ReadFile(path) //nolint:gosec // la ruta la compone la instalación, no una entrada externa
	if errors.Is(err, fs.ErrNotExist) {
		return LegacyVersion, nil
	}

	if err != nil {
		return "", fmt.Errorf("reading %s: %w", path, err)
	}

	version := strings.TrimSpace(string(raw))
	if version == "" || strings.ContainsAny(version, "\r\n") {
		return "", fmt.Errorf("%w: %s must hold one line with the version", ErrUpgradeRefused, path)
	}

	if _, err := parseVersion(version); err != nil {
		return "", fmt.Errorf("%w: %s: %w", ErrUpgradeRefused, path, err)
	}

	return version, nil
}

// parseVersion splits a version made of numbers separated by dots.
func parseVersion(version string) ([]int, error) {
	parts := strings.Split(version, ".")
	numbers := make([]int, len(parts))

	for i, part := range parts {
		number, err := strconv.Atoi(part)
		if err != nil {
			return nil, fmt.Errorf("%q is not a version made of numbers separated by dots", version)
		}

		numbers[i] = number
	}

	return numbers, nil
}

// compareVersions orders two versions by their numbers, from left to right.
func compareVersions(left, right string) int {
	a, leftErr := parseVersion(left)
	b, rightErr := parseVersion(right)

	if leftErr != nil || rightErr != nil {
		return strings.Compare(left, right)
	}

	for i := range max(len(a), len(b)) {
		var x, y int

		if i < len(a) {
			x = a[i]
		}

		if i < len(b) {
			y = b[i]
		}

		if x != y {
			if x < y {
				return -1
			}

			return 1
		}
	}

	return 0
}

// refreshBinary copies the running binary over the installed one, unless they
// are the same file, and replaces it atomically.
func refreshBinary(product string) (bool, error) {
	self, err := os.Executable()
	if err != nil {
		return false, fmt.Errorf("finding the running binary: %w", err)
	}

	target := filepath.Join(product, "bin", binName)
	if sameFile(self, target) {
		return false, nil
	}

	if err := noLinks(product, target); err != nil {
		return false, err
	}

	temporary := target + ".tmp"

	if err := copyFile(self, temporary, 0o755); err != nil {
		_ = os.Remove(temporary)

		return false, fmt.Errorf("copying the binary to %s: %w", temporary, err)
	}

	if err := os.Rename(temporary, target); err != nil {
		_ = os.Remove(temporary)

		return false, fmt.Errorf("replacing %s: %w", target, err)
	}

	return true, nil
}

// sameFile says whether two paths are the same file on disk.
func sameFile(first, second string) bool {
	left, err := os.Stat(first)
	if err != nil {
		return false
	}

	right, err := os.Stat(second)
	if err != nil {
		return false
	}

	return os.SameFile(left, right)
}

// refreshInstructions replaces the instructions when the project still carries
// the v0.1.0 template, writes them when they are missing, and leaves a custom
// text alone, reporting it as pending for a person to review.
func refreshInstructions(product, prefix string) (string, error) {
	path := filepath.Join(product, instructionsName)

	current, err := os.ReadFile(path) //nolint:gosec // la ruta la compone la instalación, no una entrada externa
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return "", fmt.Errorf("reading %s: %w", path, err)
	}

	next := instructionsFor(prefix)

	if err == nil {
		if bytes.Equal(current, next) {
			return instructionsCurrent, nil
		}

		legacy := bytes.ReplaceAll([]byte(legacyAgentTemplate), []byte(settings.DefaultPrefix+"-"), []byte(prefix+"-"))
		if !bytes.Equal(current, legacy) {
			return instructionsPending, nil
		}
	}

	if err := noLinks(product, path); err != nil {
		return "", err
	}

	if err := writeFileAtomico(path, next, 0o644); err != nil {
		return "", fmt.Errorf("writing %s: %w", path, err)
	}

	return instructionsUpdated, nil
}

// refreshContract replaces the contract of the labels when what the project
// carries is the distribution text, writes it when it is missing, and leaves a
// custom edit alone, reporting it for a person to review. The template has not
// changed since v0.1.0, so equal to it means unmodified in both cases: no
// legacy copy is needed.
func refreshContract(product string) (string, error) {
	path := filepath.Join(product, contractName)

	current, err := os.ReadFile(path) //nolint:gosec // la ruta la compone la instalación, no una entrada externa
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return "", fmt.Errorf("reading %s: %w", path, err)
	}

	if err == nil {
		if bytes.Equal(current, contract) {
			return instructionsCurrent, nil
		}

		return instructionsPending, nil
	}

	if err := noLinks(product, path); err != nil {
		return "", err
	}

	if err := writeFileAtomico(path, contract, 0o644); err != nil {
		return "", fmt.Errorf("writing %s: %w", path, err)
	}

	return instructionsUpdated, nil
}
