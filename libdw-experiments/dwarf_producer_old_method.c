
// List all DWARF producers.
// gcc -Wall -g -O2 -o dwarf_producer dwarf_producer.c -ldw

#include <sys/types.h>
#include <sys/stat.h>
#include <fcntl.h>

#include <errno.h>
#include <error.h>

#include <stdio.h>
#include <inttypes.h>

#include <dwarf.h>
#include <elfutils/libdw.h>

static int dwarf_producer(const char *file, const char *filealt)
{
	int fd = open(file, O_RDONLY);
	if (fd < 0) {
		error(0, errno, "open '%s' failed", file);
		return -1;
	}

	Dwarf *dw = dwarf_begin(fd, DWARF_C_READ);
	if (dw == NULL) {
		error(0, 0, "dwarf_begin '%s': %s", file, dwarf_errmsg(-1));
		return -1;
	}

	if (filealt) {
		int fdalt = open(filealt, O_RDONLY);
		if (fdalt < 0) {
			error(0, errno, "open '%s' failed", filealt);
			return -1;
		}

		Dwarf *dwalt = dwarf_begin(fdalt, DWARF_C_READ);
		if (dwalt == NULL) {
			error(0, 0, "dwarf_begin '%s': %s", filealt, dwarf_errmsg(-1));
			return -1;
		}
		dwarf_setalt(dw, dwalt);
	}

	bool producers_found = false;

	Dwarf_Off cuoffset;
	Dwarf_Off ncuoffset = 0;
	size_t hsize;
	while (dwarf_nextcu(dw, cuoffset = ncuoffset, &ncuoffset, &hsize,
		NULL, NULL, NULL) == 0) {
		Dwarf_Off cudieoff = cuoffset + hsize;
		Dwarf_Die cudie;
		if (dwarf_offdie(dw, cudieoff, &cudie) == NULL) {
			error(0, 0,
			    "Empty CU in '%s' at offset %" PRIx64 ": %s", file,
			    cudieoff, dwarf_errmsg(-1));
			continue;
		}

		const char *name = dwarf_diename(&cudie);
		Dwarf_Attribute attr;
		const char *producer;

		if (dwarf_attr(&cudie, DW_AT_producer, &attr) == NULL)
			continue;

		producer = dwarf_formstring(&attr);
		if (producer == NULL) {
			error(0, 0,
			    "Couldn't get producer for CU in '%s' at offset %"
			    PRIx64 ": %s", file, cudieoff, dwarf_errmsg(-1));
			continue;
		}

		printf("%s CU [%" PRIx64 "] %s: %s\n",
		    file, cudieoff, (name ? : "<unknown>"), producer);
		producers_found = true;
	}

	if (!producers_found) {
		error(0, 0, "no DW_AT_producers found in '%s'", file);
		return -1;
	}

	return 0;
}


int main(int argc, char **argv)
{
	if (argc < 2)
		error(-1, 0, "Usage: %s <.debug file> <optional .dwz file>\n", argv[0]);

	if (argc >= 3) {
		return dwarf_producer(argv[1], argv[2]);
	} else {
		return dwarf_producer(argv[1], NULL);
	}
}
