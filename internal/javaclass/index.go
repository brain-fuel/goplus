// Package javaclass reads Java class metadata directly from JAR/JMOD files.
package javaclass

import (
	"archive/zip"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const Java25ClassMajor = 69

type location struct {
	archive string
	entry   string
	version int
}

// Index is an immutable classpath/module-path view.
type Index struct {
	classes map[string]location
	release int
}

// Class is the metadata needed for safe source binding.
type Class struct {
	Name         string
	Major        int
	Methods      map[string]bool
	MethodAccess map[string]uint16
	Super        string
	Interfaces   []string
	index        *Index
}

// Open indexes JDK 25 jmods plus explicit JARs or class directories. Earlier
// entries win, matching classpath lookup. Multi-release JAR entries select the
// highest version no newer than release.
func Open(jdkHome string, paths []string, release int) (*Index, error) {
	if release < 25 {
		release = 25
	}
	if jdkHome == "" {
		jdkHome = discoverJDKHome()
	}
	idx := &Index{classes: map[string]location{}, release: release}
	if jdkHome != "" {
		// ct.sym is the source of truth used by javac --release. Some complete
		// JDK distributions (notably macOS bundles) omit jmods, while ct.sym is
		// still present and exposes precisely the supported public API.
		symbols := filepath.Join(jdkHome, "lib", "ct.sym")
		if _, err := os.Stat(symbols); err == nil {
			if err := indexCTSym(idx, symbols, release); err != nil {
				return nil, err
			}
		} else if !os.IsNotExist(err) {
			return nil, err
		}
	}
	var archives []string
	if jdkHome != "" {
		matches, _ := filepath.Glob(filepath.Join(jdkHome, "jmods", "*.jmod"))
		sort.Strings(matches)
		archives = append(archives, matches...)
	}
	archives = append(archives, paths...)
	for _, path := range archives {
		info, err := os.Stat(path)
		if err != nil {
			return nil, err
		}
		if info.IsDir() {
			err := filepath.WalkDir(path, func(file string, entry os.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".class") {
					return nil
				}
				rel, _ := filepath.Rel(path, file)
				name := strings.TrimSuffix(filepath.ToSlash(rel), ".class")
				if _, exists := idx.classes[name]; !exists {
					idx.classes[name] = location{archive: path, entry: filepath.ToSlash(rel)}
				}
				return nil
			})
			if err != nil {
				return nil, err
			}
			continue
		}
		reader, err := zip.OpenReader(path)
		if err != nil {
			return nil, fmt.Errorf("opening Java archive %s: %w", path, err)
		}
		for _, file := range reader.File {
			name, version, ok := classEntry(file.Name, release)
			if !ok {
				continue
			}
			current, exists := idx.classes[name]
			if !exists || (current.archive == path && version > current.version) {
				idx.classes[name] = location{archive: path, entry: file.Name, version: version}
			}
		}
		_ = reader.Close()
	}
	return idx, nil
}

func indexCTSym(index *Index, path string, release int) error {
	reader, err := zip.OpenReader(path)
	if err != nil {
		return fmt.Errorf("opening Java signature archive %s: %w", path, err)
	}
	defer reader.Close()
	symbol := releaseSymbol(release)
	for _, file := range reader.File {
		parts := strings.Split(file.Name, "/")
		if len(parts) < 3 || !strings.Contains(parts[0], symbol) || !strings.HasSuffix(file.Name, ".sig") {
			continue
		}
		name := strings.TrimSuffix(strings.Join(parts[2:], "/"), ".sig")
		if name == "module-info" || strings.HasSuffix(name, "/module-info") {
			continue
		}
		if _, exists := index.classes[name]; !exists {
			index.classes[name] = location{archive: path, entry: file.Name}
		}
	}
	return nil
}

func releaseSymbol(release int) string {
	if release < 10 {
		return strconv.Itoa(release)
	}
	return string(rune('A' + release - 10))
}

// Lookup loads and parses one binary name such as java.util.ArrayList.
func (i *Index) Lookup(binaryName string) (*Class, error) {
	key := strings.ReplaceAll(binaryName, ".", "/")
	where, ok := i.classes[key]
	if !ok {
		return nil, fmt.Errorf("Java class %s was not found on the JDK, classpath, or module path", binaryName)
	}
	var data []byte
	info, err := os.Stat(where.archive)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		data, err = os.ReadFile(filepath.Join(where.archive, filepath.FromSlash(where.entry)))
	} else {
		reader, openErr := zip.OpenReader(where.archive)
		if openErr != nil {
			return nil, openErr
		}
		defer reader.Close()
		for _, file := range reader.File {
			if file.Name != where.entry {
				continue
			}
			stream, readErr := file.Open()
			if readErr != nil {
				return nil, readErr
			}
			data, err = io.ReadAll(stream)
			_ = stream.Close()
			break
		}
	}
	if err != nil {
		return nil, err
	}
	class, err := parse(data)
	if err != nil {
		return nil, fmt.Errorf("parsing %s from %s: %w", binaryName, where.archive, err)
	}
	class.Name = binaryName
	class.index = i
	maximumMajor := i.release + 44
	if class.Major > maximumMajor {
		return nil, fmt.Errorf("Java class %s uses classfile major %d, newer than Java %d (%d)", binaryName, class.Major, i.release, maximumMajor)
	}
	return class, nil
}

// HasMethod checks an exact JVM name and descriptor.
func (c *Class) HasMethod(name, descriptor string) bool {
	return c.hasMethod(name, descriptor, map[string]bool{}, func(uint16) bool { return true })
}

// HasStaticMethod checks an exact public static method.
func (c *Class) HasStaticMethod(name, descriptor string) bool {
	return c.hasMethod(name, descriptor, map[string]bool{}, func(access uint16) bool { return access&0x0008 != 0 })
}

// HasVirtualMethod checks an exact public instance method.
func (c *Class) HasVirtualMethod(name, descriptor string) bool {
	return c.hasMethod(name, descriptor, map[string]bool{}, func(access uint16) bool { return access&0x0008 == 0 })
}

// HasConstructor checks a public constructor declared by this class.
func (c *Class) HasConstructor(descriptor string) bool {
	access, ok := c.MethodAccess["<init>"+descriptor]
	return ok && access&0x0008 == 0
}

func (c *Class) hasMethod(name, descriptor string, seen map[string]bool, accept func(uint16) bool) bool {
	if access, ok := c.MethodAccess[name+descriptor]; ok && accept(access) {
		return true
	}
	if name == "<init>" || c.index == nil || seen[c.Name] {
		return false
	}
	seen[c.Name] = true
	parents := append([]string{c.Super}, c.Interfaces...)
	for _, parent := range parents {
		if parent == "" {
			continue
		}
		class, err := c.index.Lookup(parent)
		if err == nil && class.hasMethod(name, descriptor, seen, accept) {
			return true
		}
	}
	return false
}

func classEntry(path string, release int) (string, int, bool) {
	name := path
	version := 0
	if strings.HasPrefix(name, "classes/") {
		name = strings.TrimPrefix(name, "classes/")
	}
	if strings.HasPrefix(name, "META-INF/versions/") {
		rest := strings.TrimPrefix(name, "META-INF/versions/")
		part, tail, ok := strings.Cut(rest, "/")
		if !ok {
			return "", 0, false
		}
		parsed, err := strconv.Atoi(part)
		if err != nil || parsed > release {
			return "", 0, false
		}
		version, name = parsed, tail
	}
	if !strings.HasSuffix(name, ".class") || name == "module-info.class" {
		return "", 0, false
	}
	return strings.TrimSuffix(name, ".class"), version, true
}

func discoverJDKHome() string {
	for _, env := range []string{"GOPLUS_JAVA_HOME", "JAVA_HOME"} {
		if home := strings.TrimSpace(os.Getenv(env)); home != "" {
			return home
		}
	}
	if home, err := os.UserHomeDir(); err == nil {
		matches, _ := filepath.Glob(filepath.Join(home, ".asdf", "installs", "java", "temurin-*"))
		sort.Sort(sort.Reverse(sort.StringSlice(matches)))
		for _, match := range matches {
			if _, err := os.Stat(filepath.Join(match, "bin", executable("javac"))); err == nil {
				return match
			}
		}
	}
	javac, err := exec.LookPath("javac")
	if err != nil {
		return ""
	}
	if resolved, err := filepath.EvalSymlinks(javac); err == nil {
		javac = resolved
	}
	return filepath.Dir(filepath.Dir(javac))
}

func executable(name string) string {
	if os.PathSeparator == '\\' {
		return name + ".exe"
	}
	return name
}

type classReader struct {
	data []byte
	at   int
}

func (r *classReader) u1() (byte, error) {
	if r.at+1 > len(r.data) {
		return 0, io.ErrUnexpectedEOF
	}
	v := r.data[r.at]
	r.at++
	return v, nil
}
func (r *classReader) u2() (uint16, error) {
	if r.at+2 > len(r.data) {
		return 0, io.ErrUnexpectedEOF
	}
	v := binary.BigEndian.Uint16(r.data[r.at:])
	r.at += 2
	return v, nil
}
func (r *classReader) u4() (uint32, error) {
	if r.at+4 > len(r.data) {
		return 0, io.ErrUnexpectedEOF
	}
	v := binary.BigEndian.Uint32(r.data[r.at:])
	r.at += 4
	return v, nil
}
func (r *classReader) skip(n int) error {
	if n < 0 || r.at+n > len(r.data) {
		return io.ErrUnexpectedEOF
	}
	r.at += n
	return nil
}

func parse(data []byte) (*Class, error) {
	r := &classReader{data: data}
	magic, err := r.u4()
	if err != nil || magic != 0xcafebabe {
		return nil, fmt.Errorf("invalid classfile magic")
	}
	if _, err := r.u2(); err != nil {
		return nil, err
	}
	major, err := r.u2()
	if err != nil {
		return nil, err
	}
	count, err := r.u2()
	if err != nil {
		return nil, err
	}
	utf8 := make([]string, count)
	classNames := make([]uint16, count)
	for index := 1; index < int(count); index++ {
		tag, err := r.u1()
		if err != nil {
			return nil, err
		}
		switch tag {
		case 1:
			length, err := r.u2()
			if err != nil {
				return nil, err
			}
			if r.at+int(length) > len(r.data) {
				return nil, io.ErrUnexpectedEOF
			}
			utf8[index] = string(r.data[r.at : r.at+int(length)])
			r.at += int(length)
		case 3, 4:
			if err := r.skip(4); err != nil {
				return nil, err
			}
		case 5, 6:
			if err := r.skip(8); err != nil {
				return nil, err
			}
			index++
		case 7:
			name, err := r.u2()
			if err != nil {
				return nil, err
			}
			classNames[index] = name
		case 8, 16, 19, 20:
			if err := r.skip(2); err != nil {
				return nil, err
			}
		case 9, 10, 11, 12, 17, 18:
			if err := r.skip(4); err != nil {
				return nil, err
			}
		case 15:
			if err := r.skip(3); err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("unknown constant-pool tag %d", tag)
		}
	}
	if _, err := r.u2(); err != nil { // class access flags
		return nil, err
	}
	if _, err := r.u2(); err != nil { // this_class
		return nil, err
	}
	superIndex, err := r.u2()
	if err != nil {
		return nil, err
	}
	interfaces, err := r.u2()
	if err != nil {
		return nil, err
	}
	interfaceNames := make([]string, 0, interfaces)
	for i := 0; i < int(interfaces); i++ {
		index, err := r.u2()
		if err != nil {
			return nil, err
		}
		name, err := resolveClassName(index, classNames, utf8)
		if err != nil {
			return nil, err
		}
		interfaceNames = append(interfaceNames, name)
	}
	fields, err := r.u2()
	if err != nil {
		return nil, err
	}
	for i := 0; i < int(fields); i++ {
		if err := skipMember(r); err != nil {
			return nil, err
		}
	}
	methods, err := r.u2()
	if err != nil {
		return nil, err
	}
	super, err := resolveClassName(superIndex, classNames, utf8)
	if err != nil {
		return nil, err
	}
	out := &Class{Major: int(major), Methods: map[string]bool{}, MethodAccess: map[string]uint16{}, Super: super, Interfaces: interfaceNames}
	for i := 0; i < int(methods); i++ {
		access, err := r.u2()
		if err != nil {
			return nil, err
		}
		name, err := r.u2()
		if err != nil {
			return nil, err
		}
		desc, err := r.u2()
		if err != nil {
			return nil, err
		}
		attrs, err := r.u2()
		if err != nil {
			return nil, err
		}
		if int(name) >= len(utf8) || int(desc) >= len(utf8) {
			return nil, fmt.Errorf("invalid method constant index")
		}
		if access&0x0001 != 0 {
			key := utf8[name] + utf8[desc]
			out.Methods[key] = true
			out.MethodAccess[key] = access
		}
		if err := skipAttributes(r, int(attrs)); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func resolveClassName(index uint16, classes []uint16, utf8 []string) (string, error) {
	if index == 0 {
		return "", nil
	}
	if int(index) >= len(classes) || classes[index] == 0 || int(classes[index]) >= len(utf8) {
		return "", fmt.Errorf("invalid class constant index %d", index)
	}
	return strings.ReplaceAll(utf8[classes[index]], "/", "."), nil
}

func skipMember(r *classReader) error {
	if err := r.skip(6); err != nil {
		return err
	}
	attrs, err := r.u2()
	if err != nil {
		return err
	}
	return skipAttributes(r, int(attrs))
}
func skipAttributes(r *classReader, count int) error {
	for i := 0; i < count; i++ {
		if err := r.skip(2); err != nil {
			return err
		}
		size, err := r.u4()
		if err != nil {
			return err
		}
		if err := r.skip(int(size)); err != nil {
			return err
		}
	}
	return nil
}
