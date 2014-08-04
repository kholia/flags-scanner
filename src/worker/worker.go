package main

// #cgo CFLAGS: -O2
// #cgo LDFLAGS: -larchive -lm -lrpm -lrpmio -lpopt
// #include <archive.h>
// #include <archive_entry.h>
// #include <rpm/rpmlib.h>
// #include <rpm/rpmio.h>
// #include <rpm/rpmts.h>
// #include <rpm/rpmte.h>
// #include <rpm/rpmcli.h>
// #include <rpm/rpmdb.h>
// #include <rpm/header.h>
// #include <math.h>
// #include <stdlib.h>
// #include <mcheck.h>
import "C"

import (
	"bufio"
	"bytes"
	"fmt"
	"reflect"

	// "crypto/md5"
	"debug/dwarf"
	"debug/elf"
	// "encoding/hex"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"
	// "time"
	"unsafe"
)

func dwzparser(dwzfile *elf.File, offset int64, debug_package string, debug_file string) string {
	if section := dwzfile.Section(".debug_str"); section != nil {
		reader := io.NewSectionReader(section, 0, int64(section.Size))
		reader.Seek(offset, 1)
		bufreader := bufio.NewReader(reader)
		// http://golang.org/pkg/bufio/#Reader.ReadString
		str, err := bufreader.ReadString('\x00')
		if err != nil {
			log.Println(err, debug_package, debug_file)
		}

		return strings.TrimSuffix(str, "\x00")
	}
	panic("dwzparser ran into problems ;(")
}

func producer(debugfile *elf.File, dwzfile *elf.File, debug_package string, debug_file string) []string {
	var output []string

	// iterate over all CUs present in debugfile
	d, e := debugfile.DWARF() // increases memory usage by 2x to 3x!

	if e != nil {
		log.Println(e, debug_package, debug_file)
		return output
	}

	reader := d.Reader()
	for {
		entry, err := reader.Next()
		if err != nil {
			log.Println(err, debug_file, debug_package)
			break
		}

		if entry == nil {
			break
		}

		if entry.Tag == dwarf.TagCompileUnit {
			for _, f := range entry.Field {
				if f.Attr == dwarf.AttrName {
					switch f.Val.(type) {
					case string:
						output = append(output, f.Val.(string))
					default:
						// distinguish between dwarf.OffsetStrpAlt, and dwarf.OffsetRefAlt
						switch f.Class {
						case dwarf.ClassStringAlt:
							output = append(output, dwzparser(dwzfile, (int64)(f.Val.(int64)), debug_package, debug_file)) // this are filenames!
							dummy := 0
							_ = dummy

							// panic("dwarf.ClassStringAlt is not handled currently in dwarf.AttrName, expect the unexpected ;(")
						case dwarf.ClassReferenceAlt:
							panic("dwarf.ClassReferenceAlt is not handled currently in dwarf.AttrName, expect the unexpected ;(")
						default:
							panic("form unhandled for dwarf.AttrName, expect the unexpected ;(")
						}
					}
				} else if f.Attr == dwarf.AttrProducer {
					switch f.Val.(type) {
					case string:
						output = append(output, f.Val.(string))
					default:
						// distinguish between dwarf.OffsetStrpAlt, and dwarf.OffsetRefAlt
						switch f.Class {
						case dwarf.ClassStringAlt:
							output = append(output, dwzparser(dwzfile, (int64)(f.Val.(int64)), debug_package, debug_file))
						case dwarf.ClassReferenceAlt:
							panic("dwarf.ClassReferenceAlt is not handled currently in dwarf.AttrProducer, expect the unexpected ;(")
						default:
							panic("form unhandled for dwarf.AttrProducer, expect the unexpected ;(")
						}
					}
				}
			}
		}
	}

	// log.Println(output)
	return output
}

func process_package(packages []string, filename string) {
	// RPM transcation stuff
	ts := C.rpmtsCreate()
	defer C.rpmtsFree(ts)
	C.rpmtsSetRootDir(ts, nil)
	C.rpmtsSetVSFlags(ts, C._RPMVSF_NOSIGNATURES)

	// create a lookup table for the files in debuginfo RPM
	// libarchive is too slow for doing this due to lzma decoding!
	cmode := C.CString("r")
	defer C.free(unsafe.Pointer(cmode))
	debug_package := filename
	cfilename := C.CString(debug_package)
	defer C.free(unsafe.Pointer(cfilename))
	/* debug_files := make(map[string]bool)
	dfd := C.Fopen(cfilename, cmode)
	defer C.Fclose(dfd)
	var dhdr C.Header
	drc := C.rpmReadPackageFile(ts, dfd, cfilename, &dhdr)
	if drc != C.RPMRC_OK {
		log.Println("[-] %s, broken RPM?", filename)
		return
	}
	defer C.headerFree(dhdr)
	fi := C.rpmfiNew(ts, dhdr, C.RPMTAG_BASENAMES, C.RPMFI_FLAGS_QUERY)
	defer C.rpmfiFree(fi)
	fi = C.rpmfiInit(fi, 0)
	for C.rpmfiNext(fi) >= 0 {
		fn := C.rpmfiFN(fi)
		// defer C.free(unsafe.Pointer(fn))
		gfn := C.GoString(fn)
		if strings.HasSuffix(gfn, ".debug") {
			debug_files[gfn] = true
		}
	} */

	// get the "dwz" content for this debug_package
	a := C.archive_read_new()
	defer C.archive_read_free(a)
	C.archive_read_support_filter_all(a)
	C.archive_read_support_format_all(a)
	r := C.archive_read_open_filename(a, cfilename, 30240)
	if r != C.ARCHIVE_OK {
		log.Println("[-] %s, broken archive?", debug_package)
	}
	pentry := C.archive_entry_new()
	var dwzfile *elf.File
	var err error
	for C.archive_read_next_header(a, &pentry) == C.ARCHIVE_OK {
		f := C.archive_entry_pathname(pentry)
		entryname := C.GoString(f)
		size := C.archive_entry_size(pentry)

		// skip folders
		mode := C.archive_entry_mode(pentry)
		gmode := int(mode)
		if gmode&0040000 != 0 {
			continue
		}

		// get the "dwz" content for this debug_package
		if strings.Contains(entryname, "lib/debug/.dwz/") {
			dwzbuf := unsafe.Pointer((*C.char)(C.malloc((C.size_t)(size))))
			defer C.free(unsafe.Pointer(dwzbuf))
			read_length := C.archive_read_data(a, dwzbuf, C.size_t(size))

			log.Println("[DWZ] read", read_length, "bytes from", entryname)

			// C array to Go array without explicit copying
			// https://code.google.com/p/go-wiki/wiki/cgo
			// dwzdata := (*[1<<31 - 1]byte)(unsafe.Pointer(dwzbuf))[:size]
			// https://code.google.com/p/go-wiki/wiki/cgo#Turning_C_arrays_into_Go_slices
			length := int(size)
			hdr := reflect.SliceHeader{
				Data: uintptr(unsafe.Pointer(dwzbuf)),
				Len:  length,
				Cap:  length,
			}
			dwzdata := *(*[]byte)(unsafe.Pointer(&hdr))

			if len(dwzdata) > 0 {
				dwzfile, err = elf.NewFile(bytes.NewReader(dwzdata))
				if err != nil {
					log.Println(err)
				}
			}
			break
		}
	}

	// process ".debug" file(s) in debug_package
	m := make(map[string][]string)
	var debugfile *elf.File
	var e error
	b := C.archive_read_new()
	defer C.archive_read_free(b)
	C.archive_read_support_filter_all(b)
	C.archive_read_support_format_all(b)
	pentry = C.archive_entry_new()
	C.archive_read_open_filename(b, cfilename, 30240)
	for C.archive_read_next_header(b, &pentry) == C.ARCHIVE_OK {
		f := C.archive_entry_pathname(pentry)
		entryname := C.GoString(f)
		size := C.archive_entry_size(pentry)

		// skip folders and symlinks
		mode := C.archive_entry_mode(pentry)
		gmode := int(mode)
		// S_IFLNK    0120000   symbolic link
		// S_IFMT     0170000   bit mask for the file type bit fields
		// S_IFDIR    0040000   directory
		if gmode&0040000 != 0 || gmode&0170000 == 0120000 {
			continue
		}

		// skip zero sized entries
		if size == 0 {
			continue
		}
		// skip non-debug files
		if !strings.HasSuffix(entryname, ".debug") || strings.Contains(entryname, "debug/.build-id/") {
			continue
		}

		// log.Println("<+> loading", entryname)

		// allocate buffer for reading data
		cbuf := unsafe.Pointer((*C.char)(C.malloc((C.size_t)(size))))
		read_length := C.archive_read_data(b, cbuf, C.size_t(size))
		log.Println("[DEBUG] read", read_length, "bytes from", entryname)

		// data := (*[1<<31 - 1]byte)(unsafe.Pointer(cbuf))[:size]  // will break at 2.14 GB!
		// https://code.google.com/p/go-wiki/wiki/cgo#Turning_C_arrays_into_Go_slices
		length := int(size)
		hdr := reflect.SliceHeader{
			Data: uintptr(unsafe.Pointer(cbuf)),
			Len:  length,
			Cap:  length,
		}
		data := *(*[]byte)(unsafe.Pointer(&hdr))

		if len(data) > 0 {
			debugfile, e = elf.NewFile(bytes.NewReader(data))
			if e != nil {
				log.Println(e, entryname, len(data))
				continue
			}
		} else {
			log.Println(len(data), entryname, debug_package)
		}

		// log.Println("<+> loaded", entryname)
		// time.Sleep(1 * time.Millisecond)

		// real action time ;(
		output := producer(debugfile, dwzfile, entryname, filename)
		m[entryname] = output
		C.free(unsafe.Pointer(cbuf))
	}
	stuff, bad := json.Marshal(m)
	if bad != nil {
		log.Panic(bad)
	}
	fmt.Println(string(stuff))
}

func main() {
	// initialization stuff
	C.rpmInitCrypto()
	defer C.rpmFreeCrypto()

	// log.Println("worker starting now ;)")

	// "input" (stdin) format
	type input struct {
		Packages     []string `json:"packages"`
		DebugPackage string   `json:"debug_package"`
	}

	// process data from stdin
	bytes, _ := ioutil.ReadAll(os.Stdin)
	// log.Println(string(bytes))
	var m input
	err := json.Unmarshal(bytes, &m)
	if err != nil {
		panic(err)
	}
	process_package(m.Packages, m.DebugPackage)
}
