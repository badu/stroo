package stroo

import (
	"errors"
	"fmt"
	"golang.org/x/tools/go/packages"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

type TemporaryProject struct {
	Config       *packages.Config
	Modules      []TemporaryPackage
	TempDirName  string
	packsWritten map[string]map[string]string
}

func (e *TemporaryProject) Filename(tmpPackName, tmpPackFileName string) string {
	return filepath.Join(e.goPathDir(tmpPackName), "src", tmpPackName, tmpPackFileName)
}

func (e *TemporaryProject) Finalize() error {
	e.Config.Env = append(e.Config.Env, "GO111MODULE=off")
	goPath := ""
	for tempPack := range e.packsWritten {
		if goPath != "" {
			goPath += string(filepath.ListSeparator)
		}
		dir := e.goPathDir(tempPack)
		goPath += dir
	}
	e.Config.Env = append(e.Config.Env, "GOPATH="+goPath)
	return nil
}

// must be called by the caller, as in `defer tmpProj.Cleanup()`
func (e *TemporaryProject) Cleanup() {
	if e.TempDirName == "" {
		return
	}
	if err := filepath.Walk(e.TempDirName, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			_ = os.Chmod(path, 0777)
		}
		return nil
	}); err != nil {
		log.Printf("error walking : %v\n", err)
	}
	if err := os.RemoveAll(e.TempDirName); err != nil {
		log.Printf("error removing : %v\n", err)
	}
	e.TempDirName = ""
}

func (e *TemporaryProject) File(tmpPackName, tmpPackFileName string) string {
	if m := e.packsWritten[tmpPackName]; m != nil {
		return m[tmpPackFileName]
	}
	return ""
}

var versionSuffixRE = regexp.MustCompile(`v\d+`)

func (e *TemporaryProject) goPathDir(tmpPackName string) string {
	dir := path.Base(tmpPackName)
	if versionSuffixRE.MatchString(dir) {
		dir = path.Base(path.Dir(tmpPackName)) + "_" + dir
	}
	return filepath.Join(e.TempDirName, dir)
}

type TemporaryPackage struct {
	Name  string
	Files map[string]interface{}
}

func CreateTempProj(tmpPacks []TemporaryPackage) (*TemporaryProject, error) {
	if len(tmpPacks) == 0 {
		return nil, errors.New("at least one temporary package is required")
	}
	dirName := strings.Replace(tmpPacks[0].Name, "/", "_", -1)
	tempFolder, err := ioutil.TempDir("", dirName)
	if err != nil {
		return nil, err
	}

	result := &TemporaryProject{
		Config: &packages.Config{
			Dir:   tempFolder,
			Env:   append(os.Environ(), "GOPACKAGESDRIVER=off", "GOROOT="),
			Tests: true,
			Mode:  packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles | packages.NeedImports,
		},
		Modules:      tmpPacks,
		TempDirName:  tempFolder,
		packsWritten: map[string]map[string]string{},
	}
	for _, tPack := range tmpPacks {
		for filePiece, value := range tPack.Files {
			fullPath := result.Filename(tPack.Name, filepath.FromSlash(filePiece))
			existingFilePiece, has := result.packsWritten[tPack.Name]
			if !has {
				existingFilePiece = map[string]string{}
				result.packsWritten[tPack.Name] = existingFilePiece
			}
			existingFilePiece[filePiece] = fullPath
			if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
				return nil, err
			}
			switch value := value.(type) {
			case string:
				if err := ioutil.WriteFile(fullPath, []byte(value), 0644); err != nil {
					return nil, err
				}
			default:
				return nil, fmt.Errorf("invalid type %T in files, must be string", value)
			}
		}
	}
	if err := result.Finalize(); err != nil {
		return nil, err
	}
	return result, nil
}
