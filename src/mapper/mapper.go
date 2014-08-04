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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"regexp"
	"time"
	"unsafe"
)

func process_packages(packages []string, debug_packages []string) {
	m := make(map[string][]string)

	ts := C.rpmtsCreate()
	defer C.rpmtsFree(ts)
	C.rpmtsSetRootDir(ts, nil)
	C.rpmtsSetVSFlags(ts, C._RPMVSF_NOSIGNATURES)

	for _, filename := range packages {
		cfilename := C.CString(filename)
		defer C.free(unsafe.Pointer(cfilename))
		cmode := C.CString("r")
		defer C.free(unsafe.Pointer(cmode))

		// extract metadata (RPM headers) from filename
		fd := C.Fopen(cfilename, cmode)
		if fd == nil {
			log.Println("fd == NULL while processing", filename)
			time.Sleep(100000 * time.Millisecond)
			panic("^^^")
		}
		var hdr C.Header
		rc := C.rpmReadPackageFile(ts, fd, cfilename, &hdr)

		// read header stuff
		if rc != C.RPMRC_OK {
			log.Println("[-] broken RPM?", filename)
			continue
		}

		// extract various "tags" from the RPM file
		csrpm := C.headerGetAsString(hdr, C.RPMTAG_SOURCERPM)
		defer C.free((unsafe.Pointer)(csrpm))
		srpm := C.GoString(csrpm)
		C.headerFree(hdr)
		/* cnvr := C.headerGetAsString(hdr, C.RPMTAG_NVRA)
		defer C.free((unsafe.Pointer)(cnvr))
		nvr := C.GoString(cnvr)
		log.Println(nvr) */

		// determine the name of debuginfo package from srpm name
		re := regexp.MustCompile("(?P<name>.*)-.*-.*")
		res := re.FindAllStringSubmatch(srpm, -1)[0]
		if len(res) == 0 {
			log.Println("[-] SRPM name missing for", filename)
		}
		names := re.SubexpNames()
		md := map[string]string{}
		for i, n := range res {
			md[names[i]] = n
		}
		target_debug_package := md["name"] + "-debuginfo-"

		// do we have this "target_debug_package" in "debug_packages"?
		debug_package := ""
		re = regexp.MustCompile(regexp.QuoteMeta(target_debug_package))
		for _, name := range debug_packages {
			if re.MatchString(path.Base(name)+"\\d") == true {
				debug_package = name
			}
		}
		if debug_package == "" {
			log.Println("[-] debuginfo missing (during preliminary scan) for", filename)
		}
		m[debug_package] = append(m[debug_package], filename)
		C.Fclose(fd) // critical step!
	}
	b, err := json.Marshal(m)
	if err != nil {
		log.Panic("JSON encoding problem ;(")
	}

	fmt.Println(string(b)) // output
}

func main() {
	// initialization stuff
	C.rpmInitCrypto()
	defer C.rpmFreeCrypto()

	// "input" (stdin) format
	type input struct {
		Packages      []string `json:"packages"`
		DebugPackages []string `json:"debug_packages"`
	}

	// process data from stdin
	bytes, _ := ioutil.ReadAll(os.Stdin)
	// log.Println(string(bytes))
	var m input
	err := json.Unmarshal(bytes, &m)
	if err != nil {
		panic(err)
	}
	log.Println("input is parsed now ;)")

	process_packages(m.Packages, m.DebugPackages)
}
