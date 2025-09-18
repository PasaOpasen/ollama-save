package util

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

func expand_user(path string) (string, error) {
    if len(path) == 0 || path[0] != '~' {
        return path, nil
    }

    usr, err := user.Current()
    if err != nil {
        return "", err
    }
    return filepath.Join(usr.HomeDir, path[1:]), nil
}


// ManifestLayer represents a layer in the Ollama manifest
type ManifestLayer struct {
	Digest string `json:"digest"`
}

// ManifestConfig represents the config section in the Ollama manifest
type ManifestConfig struct {
	Digest string `json:"digest"`
}

// Manifest represents the Ollama manifest file structure
type Manifest struct {
	Layers []ManifestLayer `json:"layers"`
	Config ManifestConfig  `json:"config"`
}

// ExportModels exports the specified models from the Ollama models directory to a tar.gz archive
func ExportModels(ollamaDir string, models []string, outputPath string) error {

	ollamaDir, err := expand_user(ollamaDir)
	if err != nil {
		return fmt.Errorf("failed to expand user: %v", err)
	}

	// Create the output file
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %v", err)
	}
	defer outputFile.Close()

	// Create gzip writer
	gzipWriter := gzip.NewWriter(outputFile)
	defer gzipWriter.Close()

	// Create tar writer
	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	// Process each model
	for _, modelSpec := range models {
		// Split model name and tag
		parts := strings.SplitN(modelSpec, ":", 2)
		model := parts[0]
		tag := "latest"
		if len(parts) > 1 {
			tag = parts[1]
		}

		// Get the specific manifest file
		manifestPath := filepath.Join(ollamaDir, getManifestPath(model, tag))
		if _, err := os.Stat(manifestPath); err != nil {
			return fmt.Errorf("manifest not found for model %s:%s: %v", model, tag, err)
		}

		// Read and parse manifest
		manifest, err := readManifest(manifestPath)
		if err != nil {
			return fmt.Errorf("failed to read manifest %s: %v", manifestPath, err)
		}

		// Add manifest to archive with full path structure
		manifestArchivePath := getManifestPath(model, tag)
		if err := addFileToTar(tarWriter, manifestPath, manifestArchivePath); err != nil {
			return fmt.Errorf("failed to add manifest to archive: %v", err)
		}

		// Add config blob
		configBlobPath := getBlobPath(ollamaDir, manifest.Config.Digest)
		if err := addFileToTar(tarWriter, configBlobPath, filepath.Join("blobs", filepath.Base(configBlobPath))); err != nil {
			return fmt.Errorf("failed to add config blob to archive: %v", err)
		}

		// Add layer blobs
		for _, layer := range manifest.Layers {
			blobPath := getBlobPath(ollamaDir, layer.Digest)
			if err := addFileToTar(tarWriter, blobPath, filepath.Join("blobs", filepath.Base(blobPath))); err != nil {
				return fmt.Errorf("failed to add layer blob to archive: %v", err)
			}
		}
	}

	return nil
}

// ImportModels imports models from a tar.gz archive into the Ollama models directory
func ImportModels(ollamaDir string, archivePath string) error {

	ollamaDir, err := expand_user(ollamaDir)
	if err != nil {
		return fmt.Errorf("failed to expand user: %v", err)
	}

	// Open the archive file
	archiveFile, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("failed to open archive: %v", err)
	}
	defer archiveFile.Close()

	// Create gzip reader
	gzipReader, err := gzip.NewReader(archiveFile)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %v", err)
	}
	defer gzipReader.Close()

	// Create tar reader
	tarReader := tar.NewReader(gzipReader)
	var processed = 0

	// Process each file in the archive
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar header: %v", err)
		}

		// Determine target path, preserving the full path structure
		targetPath := filepath.Join(ollamaDir, header.Name)

		// Validate the path is within the expected structure
		if strings.HasPrefix(header.Name, "manifests/") {
			if !strings.HasPrefix(header.Name, getManifestBasePath()) {
				return fmt.Errorf("invalid manifest path structure: %s", header.Name)
			}
		}

		// Create parent directories if they don't exist
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return fmt.Errorf("failed to create directories: %v", err)
		}

		// Create the file
		targetFile, err := os.Create(targetPath)
		if err != nil {
			return fmt.Errorf("failed to create file %s: %v", targetPath, err)
		}

		// Copy contents
		if _, err := io.Copy(targetFile, tarReader); err != nil {
			targetFile.Close()
			return fmt.Errorf("failed to write file contents: %v", err)
		}
		targetFile.Close()

		processed += 1
	}

	if processed == 0 {
		return fmt.Errorf("no models in archive")
	}

	return nil
}

// readManifest reads and parses an Ollama manifest file
func readManifest(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}

	return &manifest, nil
}

// getBlobPath returns the full path to a blob given its digest
func getBlobPath(ollamaDir string, digest string) string {
	// Extract the hash part from "sha256:hash"
	hash := strings.Split(digest, ":")[1]
	return filepath.Join(ollamaDir, "blobs", fmt.Sprintf("sha256-%s", hash))
}

// getManifestBasePath returns the base path for manifest files
func getManifestBasePath() string {
	return filepath.Join("manifests", "registry.ollama.ai", "library")
}

// getManifestPath returns the full path for a model's manifest file
func getManifestPath(model string, tag string) string {
	return filepath.Join(getManifestBasePath(), model, tag)
}

// addFileToTar adds a file to a tar archive
func addFileToTar(tw *tar.Writer, sourcePath string, targetPath string) error {
	file, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return err
	}

	header := &tar.Header{
		Name:    targetPath,
		Size:    stat.Size(),
		Mode:    int64(stat.Mode()),
		ModTime: stat.ModTime(),
	}

	if err := tw.WriteHeader(header); err != nil {
		return err
	}

	if _, err := io.Copy(tw, file); err != nil {
		return err
	}

	return nil
}
