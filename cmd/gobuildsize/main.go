package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

func main() {
	args := append([]string{"build", "-work", "-a"}, os.Args[1:]...)
	cmd := exec.Command("go", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatalf("`go %s` command failed: %v", err, strings.Join(args, " "))
	}
	pat := regexp.MustCompile("WORK=(.*)")
	workdir := pat.FindStringSubmatch(string(out))[1]
	db := &db{}
	readFilesInFolder(workdir, db)
	for _, pkg := range db.packagesDescendingBySize() {
		fmt.Println(pkg.packageName, pkg.archiveSize)
	}
	if err := os.RemoveAll(workdir); err != nil {
		log.Fatalf("failed to remove WORK directory: %v", err)
	}
}

type pkgName string

type compiledPackage struct {
	packageName     pkgName
	archiveFilePath string
	archiveSize     int64
}

type db struct {
	pkgs map[pkgName]compiledPackage
}

func (db *db) packagesDescendingBySize() []compiledPackage {
	pkgs := make([]compiledPackage, 0, len(db.pkgs))
	for _, pkg := range db.pkgs {
		pkgs = append(pkgs, pkg)
	}
	sort.Slice(pkgs, func(i, j int) bool {
		if pkgs[i].archiveSize == pkgs[j].archiveSize {
			return pkgs[i].packageName < pkgs[j].packageName
		}
		return pkgs[i].archiveSize > pkgs[j].archiveSize
	})
	return pkgs
}

func (db *db) add(p compiledPackage) {
	if db.pkgs == nil {
		db.pkgs = map[pkgName]compiledPackage{}
	}
	_, ok := db.pkgs[p.packageName]
	if !ok {
		p.archiveSize = fileSizeInBytes(p.archiveFilePath)
		db.pkgs[p.packageName] = p
	}
}

func readFilesInFolder(dir string, db *db) {
	dirs, err := os.ReadDir(dir)
	if err != nil {
		log.Fatal(err)
	}
	for _, bdir := range dirs {
		importCfgPath, err := filepath.Abs(filepath.Join(dir, bdir.Name(), "importcfg"))
		if err != nil {
			log.Fatal(err)
		}
		importcfgBytes, err := os.ReadFile(importCfgPath)
		if err != nil {
			continue
		}
		parseImportCfg(string(importcfgBytes), db)
	}
}

var packageFilePattern = regexp.MustCompile(`packagefile ([^=]+)[=](.+)`)

func parseImportCfg(s string, db *db) {
	for _, pkg := range packageFilePattern.FindAllStringSubmatch(s, -1) {
		p := compiledPackage{
			packageName:     pkgName(pkg[1]),
			archiveFilePath: pkg[2],
		}
		db.add(p)
	}
}

func fileSizeInBytes(path string) int64 {
	file, err := os.Stat(path)
	if err != nil {
		log.Fatal(err)
	}
	return file.Size()
}
