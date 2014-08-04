
// List all DWARF producers.
// gcc -Wall -g -O2 -o dwarf_producer_memory dwarf_producer_memory.c -ldw

#include <sys/types.h>
#include <sys/stat.h>
#include <fcntl.h>
#include <unistd.h>
#include <sys/types.h>
#include <errno.h>
#include <error.h>
#include <stdio.h>
#include <inttypes.h>
#include <dwarf.h>
#include <elfutils/libdw.h>
#include <sys/mman.h>
#include <string.h>

static int dwarf_producer(const char *file)
{

	FILE *fp = fopen(file, "rb");
	struct stat statbuf;

	if (lstat(file, &statbuf)) {
		perror(file);
		return -1;
	}

	size_t size = statbuf.st_size;
	unsigned char *buffer = (unsigned char *) malloc(size);

	if (fread(buffer, size, 1, fp) < sizeof buffer) {

	}

	int sfd = shm_open("something", O_CREAT | O_RDWR, 0);
	if (ftruncate(sfd, size) < 0) {
		perror(file);
		return -2;
	}
	// we could do without mmap!
	void *addr =
	    mmap(NULL, size, PROT_READ | PROT_WRITE, MAP_SHARED, sfd, 0);
	memcpy(addr, buffer, size);

	/* int fd = open(file, O_RDONLY);
	   if (fd < 0) {
	   error(0, errno, "open '%s' failed", file);
	   return -1;
	   }

	   Dwarf *dw = dwarf_begin(fd, DWARF_C_READ); */

	Dwarf *dw = dwarf_begin(sfd, DWARF_C_READ);
	if (dw == NULL) {
		error(0, 0, "dwarf_begin '%s': %s", file, dwarf_errmsg(-1));
		return -1;
	}

	bool producers_found = false;

	Dwarf_Off cuoffset;
	Dwarf_Off ncuoffset = 0;
	size_t hsize;
	while (dwarf_nextcu(dw, cuoffset =
		ncuoffset, &ncuoffset, &hsize, NULL, NULL, NULL) == 0) {
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

		printf("%s CU [%" PRIx64 "] %s: %s\n", file, cudieoff,
		    (name ? : "<unknown>"), producer);
		producers_found = true;
	}

	if (!producers_found) {
		error(0, 0, "no DW_AT_producers found in '%s'", file);
		return -1;
	}
	// cleanup
	munmap(addr, size);
	close(sfd);
	shm_unlink("something");

	fclose(fp);

	return 0;
}

int main(int argc, char **argv)
{
	if (argc < 2)
		error(-1, 0, "give one or more DWARF files as argument");

	int res = 0;
	int i;
	for (i = 1; i < argc; i++) {
		int r = dwarf_producer(argv[i]);
		if (r < 0)
			res = r;
	}

	return res;
}
