package file

import (
	"encoding/json"
	"os"
	"path"
)

func StoreJSON(name string, tmpExt string, flag int, perm os.FileMode, data any) error {
	fdir, _ := path.Split(name)
	if _, err := os.Stat(fdir); err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		err = os.MkdirAll(fdir, 0755)
		if err != nil {
			return err
		}
	}
	err := storeJSON(name, tmpExt, flag, perm, data)
	if err != nil {
		os.Remove(name + tmpExt)
		return err
	}
	return os.Rename(name+tmpExt, name)
}

func storeJSON(name string, tmpExt string, flag int, perm os.FileMode, data any) error {
	f, err := os.OpenFile(name+tmpExt, flag, perm)
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewEncoder(f).Encode(data)
}

// Store writes data to name by filling name+tmpExt first and renaming it into
// place, so a concurrent reader sees either the previous contents or the new
// ones, never a half-written file.
func Store(name string, tmpExt string, perm os.FileMode, data []byte) error {
	fdir, _ := path.Split(name)
	if fdir != "" {
		if err := os.MkdirAll(fdir, 0755); err != nil {
			return err
		}
	}
	if err := os.WriteFile(name+tmpExt, data, perm); err != nil {
		os.Remove(name + tmpExt)
		return err
	}
	return os.Rename(name+tmpExt, name)
}
