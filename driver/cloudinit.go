package driver

import (
	"bytes"
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"

	"github.com/jumppad-labs/nomad-driver-libvirt/driver/iso9660util"
)

//go:embed templates
var templateFS embed.FS

const (
	templateFSRoot = "templates"

	cloudInitISOName = "cloud-init.iso"
)

func (d *LibVirtDriverPlugin) generateISO9660(allocDir string) error {
	layout, err := d.executeTemplate()
	if err != nil {
		return err
	}
	return iso9660util.Write(filepath.Join(allocDir, cloudInitISOName), "cidata", layout)
}

func (d *LibVirtDriverPlugin) executeTemplate() ([]iso9660util.Entry, error) {

	fsys, err := fs.Sub(templateFS, templateFSRoot)
	if err != nil {
		return nil, err
	}

	var layout []iso9660util.Entry
	walkFn := func(path string, dir fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		if dir.IsDir() {
			return nil
		}
		if !dir.Type().IsRegular() {
			return fmt.Errorf("got non-regular file %q", path)
		}

		templateB, err := fs.ReadFile(fsys, path)
		if err != nil {
			return err
		}

		d.logger.Error("file path", "fsys", fsys, "path", path, "bytes", string(templateB))

		layout = append(layout, iso9660util.Entry{
			Path:   path,
			Reader: bytes.NewReader(templateB),
		})
		return nil
	}

	if err := fs.WalkDir(fsys, ".", walkFn); err != nil {
		return nil, err
	}

	return layout, nil
}
