package main

// #cgo CFLAGS: -O2
// #cgo LDFLAGS: -larchive -lm -lpopt
// #include <archive.h>
// #include <archive_entry.h>
// #include <math.h>
// #include <stdlib.h>
import "C"

import (
	"bytes"
	"fmt"
	"reflect"
	// "crypto/md5"
	"debug/dwarf"
	"debug/elf"
	// "encoding/hex"
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"strings"
	// "time"
	"unsafe"
)

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
					case dwarf.OffsetStrpAlt:
						output = append(output, (string)(f.Val.(dwarf.OffsetStrpAlt)))
					case dwarf.OffsetRefAlt:
						log.Panic("DW_FORM_GNU_ref_alt handling missing for dwarf.AttrName ;(")
					default:
						panic("form unhandled for dwarf.AttrName, expect the unexpected ;(")
					}
				} else if f.Attr == dwarf.AttrProducer {
					switch f.Val.(type) {
					case string:
						output = append(output, f.Val.(string))
					case dwarf.OffsetStrpAlt:
						log.Panic("XXX DW_FORM_GNU_ref_alt handling missing for dwarf.AttrProducer ;(")
					case dwarf.OffsetRefAlt:
						log.Panic("DW_FORM_GNU_ref_alt handling missing for dwarf.AttrProducer ;(")
					default:
						panic("form unhandled for dwarf.AttrProducer, expect the unexpected ;(")
					}

				}
			}
		}
	}

	// log.Println(output)
	return output
}

func process_package(packages []string, debug_package string) {
	// create a lookup table for the files in debuginfo RPM
	// libarchive is too slow for doing this due to lzma decoding!
	cmode := C.CString("r")
	defer C.free(unsafe.Pointer(cmode))
	cfilename := C.CString(debug_package)
	defer C.free(unsafe.Pointer(cfilename))

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
		output := producer(debugfile, dwzfile, entryname, debug_package)
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
	// "input" (stdin) format
	type input struct {
		Packages     []string `json:"packages"`
		DebugPackage string   `json:"debug_package"`
	}

	// process data from stdin
	bytes, _ := ioutil.ReadAll(os.Stdin)
	var m input
	err := json.Unmarshal(bytes, &m)
	if err != nil {
		panic(err)
	}
	process_package(m.Packages, m.DebugPackage)
}
