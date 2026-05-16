package ignore

import "strings"

var defaultIgnoredDirectories = map[string]struct{}{
	".git":             {},
	".svn":             {},
	".hg":              {},
	".bzr":             {},
	".idea":            {},
	".vscode":          {},
	".vs":              {},
	".eclipse":         {},
	".settings":        {},
	"node_modules":     {},
	"bower_components": {},
	"jspm_packages":    {},
	"vendor":           {},
	"venv":             {},
	".venv":            {},
	"env":              {},
	".env":             {},
	"__pycache__":      {},
	".pytest_cache":    {},
	".mypy_cache":      {},
	"site-packages":    {},
	".tox":             {},
	"eggs":             {},
	".eggs":            {},
	"lib64":            {},
	"parts":            {},
	"sdist":            {},
	"wheels":           {},
	"dist":             {},
	"build":            {},
	"out":              {},
	"output":           {},
	"bin":              {},
	"obj":              {},
	"target":           {},
	".next":            {},
	".nuxt":            {},
	".output":          {},
	".vercel":          {},
	".netlify":         {},
	".serverless":      {},
	"_build":           {},
	"public/build":     {},
	".parcel-cache":    {},
	".turbo":           {},
	".svelte-kit":      {},
	"coverage":         {},
	".nyc_output":      {},
	"htmlcov":          {},
	".coverage":        {},
	"__tests__":        {},
	"__mocks__":        {},
	".jest":            {},
	"logs":             {},
	"log":              {},
	"tmp":              {},
	"temp":             {},
	"cache":            {},
	".cache":           {},
	".tmp":             {},
	".temp":            {},
	".generated":       {},
	"generated":        {},
	"auto-generated":   {},
	".terraform":       {},
	".husky":           {},
	".github":          {},
	".circleci":        {},
	".gitlab":          {},
	"fixtures":         {},
	"snapshots":        {},
	"__snapshots__":    {},
}

var ignoredExtensions = map[string]struct{}{
	".png":       {},
	".jpg":       {},
	".jpeg":      {},
	".gif":       {},
	".svg":       {},
	".ico":       {},
	".webp":      {},
	".bmp":       {},
	".tiff":      {},
	".tif":       {},
	".psd":       {},
	".ai":        {},
	".sketch":    {},
	".fig":       {},
	".xd":        {},
	".zip":       {},
	".tar":       {},
	".gz":        {},
	".rar":       {},
	".7z":        {},
	".bz2":       {},
	".xz":        {},
	".tgz":       {},
	".exe":       {},
	".dll":       {},
	".so":        {},
	".dylib":     {},
	".a":         {},
	".lib":       {},
	".o":         {},
	".obj":       {},
	".class":     {},
	".jar":       {},
	".war":       {},
	".ear":       {},
	".pyc":       {},
	".pyo":       {},
	".pyd":       {},
	".beam":      {},
	".wasm":      {},
	".node":      {},
	".ppt":       {},
	".pptx":      {},
	".odp":       {},
	".mp4":       {},
	".mp3":       {},
	".wav":       {},
	".mov":       {},
	".avi":       {},
	".mkv":       {},
	".flv":       {},
	".wmv":       {},
	".ogg":       {},
	".webm":      {},
	".flac":      {},
	".aac":       {},
	".m4a":       {},
	".woff":      {},
	".woff2":     {},
	".ttf":       {},
	".eot":       {},
	".otf":       {},
	".db":        {},
	".sqlite":    {},
	".sqlite3":   {},
	".mdb":       {},
	".accdb":     {},
	".min.js":    {},
	".min.css":   {},
	".bundle.js": {},
	".chunk.js":  {},
	".map":       {},
	".lock":      {},
	".pem":       {},
	".key":       {},
	".crt":       {},
	".cer":       {},
	".p12":       {},
	".pfx":       {},
	".parquet":   {},
	".avro":      {},
	".feather":   {},
	".npy":       {},
	".npz":       {},
	".pkl":       {},
	".pickle":    {},
	".h5":        {},
	".hdf5":      {},
	".bin":       {},
	".dat":       {},
	".data":      {},
	".raw":       {},
	".iso":       {},
	".img":       {},
	".dmg":       {},
}

var ignoredFiles = map[string]struct{}{
	"package-lock.json":  {},
	"yarn.lock":          {},
	"pnpm-lock.yaml":     {},
	"composer.lock":      {},
	"Gemfile.lock":       {},
	"poetry.lock":        {},
	"Cargo.lock":         {},
	"go.sum":             {},
	".gitignore":         {},
	".gitattributes":     {},
	".npmrc":             {},
	".yarnrc":            {},
	".editorconfig":      {},
	".prettierrc":        {},
	".prettierignore":    {},
	".eslintignore":      {},
	".dockerignore":      {},
	"Thumbs.db":          {},
	".DS_Store":          {},
	"LICENSE":            {},
	"LICENSE.md":         {},
	"LICENSE.txt":        {},
	"CHANGELOG.md":       {},
	"CHANGELOG":          {},
	"CONTRIBUTING.md":    {},
	"CODE_OF_CONDUCT.md": {},
	"SECURITY.md":        {},
	".env":               {},
	".env.local":         {},
	".env.development":   {},
	".env.production":    {},
	".env.test":          {},
	".env.example":       {},
}

func IsHardcodedIgnoredDirectory(name string) bool {
	_, ok := defaultIgnoredDirectories[name]
	return ok
}

func ShouldIgnorePath(filePath string) bool {
	normalized := NormalizePath(filePath)
	parts := strings.Split(normalized, "/")
	fileName := parts[len(parts)-1]
	fileNameLower := strings.ToLower(fileName)
	if strings.HasPrefix(fileName, ".") {
		return true
	}

	for index, part := range parts {
		if _, ok := defaultIgnoredDirectories[part]; ok {
			return true
		}
		if index < len(parts)-1 && part != "" && strings.HasPrefix(part, ".") {
			return true
		}
	}

	if _, ok := ignoredFiles[fileName]; ok {
		return true
	}
	if _, ok := ignoredFiles[fileNameLower]; ok {
		return true
	}

	if ignoredExtension(fileNameLower) {
		return true
	}

	return strings.Contains(fileNameLower, ".bundle.") ||
		strings.Contains(fileNameLower, ".chunk.") ||
		strings.Contains(fileNameLower, ".generated.") ||
		strings.HasSuffix(fileNameLower, ".d.ts")
}

func ignoredExtension(fileNameLower string) bool {
	lastDot := strings.LastIndex(fileNameLower, ".")
	if lastDot < 0 {
		return false
	}
	if _, ok := ignoredExtensions[fileNameLower[lastDot:]]; ok {
		return true
	}
	secondLastDot := strings.LastIndex(fileNameLower[:lastDot], ".")
	if secondLastDot < 0 {
		return false
	}
	_, ok := ignoredExtensions[fileNameLower[secondLastDot:]]
	return ok
}
