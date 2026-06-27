// orbit-pack builds extension.orbit dev packages from dist/<pluginId>/.
package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/andybalholm/brotli"
)

const (
	wasmEntry     = "main.wasm.br"
	manifestName  = "manifest.json"
	readmeName    = "README.md"
	orbitFilename   = "extension.orbit"
	defaultWasmName = "plugin.wasm"
)

func main() {
	distDir := flag.String("dist", "", "plugin dist directory (e.g. dist/juejin)")
	srcDir := flag.String("src", "", "plugin source directory (sync manifest.json before pack)")
	flag.Parse()
	if *distDir == "" {
		fmt.Fprintln(os.Stderr, "usage: orbit-pack -dist dist/<pluginId> [-src plugins/<category>/<id>]")
		os.Exit(1)
	}

	if *srcDir != "" {
		if err := syncDistFromSrc(*srcDir, *distDir); err != nil {
			fmt.Fprintf(os.Stderr, "sync from src: %v\n", err)
			os.Exit(1)
		}
	}

	wasmPath := filepath.Join(*distDir, defaultWasmName)
	manifestPath := filepath.Join(*distDir, manifestName)
	readmePath := filepath.Join(*distDir, readmeName)
	if _, err := os.Stat(wasmPath); err != nil {
		fmt.Fprintf(os.Stderr, "missing %s (run make build or make package first)\n", wasmPath)
		os.Exit(1)
	}
	if _, err := os.Stat(manifestPath); err != nil {
		fmt.Fprintf(os.Stderr, "missing %s\n", manifestPath)
		os.Exit(1)
	}

	wasmRaw, err := os.ReadFile(wasmPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read wasm: %v\n", err)
		os.Exit(1)
	}

	manifestRaw, err := os.ReadFile(manifestPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read manifest: %v\n", err)
		os.Exit(1)
	}

	// README.md is optional
	var readmeRaw []byte
	if _, err := os.Stat(readmePath); err == nil {
		readmeRaw, err = os.ReadFile(readmePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "read readme: %v\n", err)
			os.Exit(1)
		}
	}

	orbitManifest, err := patchManifestEntry(manifestRaw)
	if err != nil {
		fmt.Fprintf(os.Stderr, "patch manifest: %v\n", err)
		os.Exit(1)
	}

	wasmBR, err := brotliCompress(wasmRaw)
	if err != nil {
		fmt.Fprintf(os.Stderr, "brotli: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(filepath.Join(*distDir, wasmEntry), wasmBR, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "write %s: %v\n", wasmEntry, err)
		os.Exit(1)
	}

	orbitPath := filepath.Join(*distDir, orbitFilename)
	if err := writeOrbitZip(orbitPath, orbitManifest, wasmBR, readmeRaw); err != nil {
		fmt.Fprintf(os.Stderr, "zip: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("packed %s (%d bytes wasm -> %d bytes br)\n", orbitPath, len(wasmRaw), len(wasmBR))
}

func syncDistFromSrc(src, dist string) error {
	if err := os.MkdirAll(dist, 0o755); err != nil {
		return err
	}
	for _, name := range []string{manifestName, readmeName} {
		srcPath := filepath.Join(src, name)
		info, err := os.Stat(srcPath)
		if err != nil {
			if os.IsNotExist(err) {
				if name == manifestName {
					return fmt.Errorf("missing %s", srcPath)
				}
				continue
			}
			return err
		}
		if info.IsDir() {
			return fmt.Errorf("%s is a directory", srcPath)
		}
		data, err := os.ReadFile(srcPath)
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(dist, name), data, 0o644); err != nil {
			return err
		}
	}
	assetsSrc := filepath.Join(src, "assets")
	if info, err := os.Stat(assetsSrc); err == nil && info.IsDir() {
		assetsDist := filepath.Join(dist, "assets")
		if err := os.RemoveAll(assetsDist); err != nil {
			return err
		}
		if err := copyDir(assetsSrc, assetsDist); err != nil {
			return err
		}
	}
	return nil
}

func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
}

func patchManifestEntry(raw []byte) ([]byte, error) {
	var doc map[string]interface{}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	cfg, _ := doc["config"].(map[string]interface{})
	if cfg == nil {
		cfg = map[string]interface{}{}
		doc["config"] = cfg
	}
	wasmCfg, _ := cfg["wasm"].(map[string]interface{})
	if wasmCfg == nil {
		wasmCfg = map[string]interface{}{}
		cfg["wasm"] = wasmCfg
	}
	wasmCfg["entry"] = wasmEntry
	return json.MarshalIndent(doc, "", "  ")
}

func brotliCompress(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	w := brotli.NewWriterLevel(&buf, brotli.BestCompression)
	if _, err := w.Write(data); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func writeOrbitZip(path string, manifest, wasmBR, readme []byte) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	if err := zipEntry(zw, manifestName, manifest); err != nil {
		return err
	}
	if err := zipEntry(zw, wasmEntry, wasmBR); err != nil {
		return err
	}
	if len(readme) > 0 {
		if err := zipEntry(zw, readmeName, readme); err != nil {
			return err
		}
	}
	return zw.Close()
}

func zipEntry(zw *zip.Writer, name string, data []byte) error {
	w, err := zw.Create(name)
	if err != nil {
		return err
	}
	_, err = io.Copy(w, bytes.NewReader(data))
	return err
}
